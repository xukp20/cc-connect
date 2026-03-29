package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var defaultObservePollInterval = 2 * time.Second
var observationXMLTagRe = regexp.MustCompile(`<[^>]+>`)

type observerFileState struct {
	offset  int64
	pending []byte
	skip    bool
}

type sessionObserver struct {
	target         ObserverTarget
	channelID      string
	dir            string
	assistantLabel string
	pollInterval   time.Duration

	mu    sync.Mutex
	files map[string]*observerFileState
}

func newSessionObserver(target ObserverTarget, channelID, dir, assistantLabel string, pollInterval time.Duration) *sessionObserver {
	if pollInterval <= 0 {
		pollInterval = defaultObservePollInterval
	}
	if assistantLabel == "" {
		assistantLabel = "assistant"
	}
	return &sessionObserver{
		target:         target,
		channelID:      channelID,
		dir:            dir,
		assistantLabel: assistantLabel,
		pollInterval:   pollInterval,
		files:          make(map[string]*observerFileState),
	}
}

func (e *Engine) startObserver() {
	if !e.observe.Enabled || e.observe.Channel == "" {
		return
	}
	src, ok := e.agent.(SessionObservationSource)
	if !ok {
		slog.Debug("observer disabled: agent does not support session observation", "project", e.name, "agent", e.agent.Name())
		return
	}

	var target ObserverTarget
	for _, p := range e.platforms {
		if ot, ok := p.(ObserverTarget); ok {
			target = ot
			break
		}
	}
	if target == nil {
		slog.Debug("observer disabled: no observer-capable platform", "project", e.name)
		return
	}

	dir, err := src.SessionObservationDir()
	if err != nil {
		slog.Warn("observer disabled: resolve session log dir", "project", e.name, "error", err)
		return
	}

	obs := newSessionObserver(target, e.observe.Channel, dir, src.SessionObservationAssistantName(), defaultObservePollInterval)
	if err := obs.prime(); err != nil {
		slog.Warn("observer prime failed", "project", e.name, "dir", dir, "error", err)
	}
	go obs.run(e.ctx)
	slog.Info("observer started", "project", e.name, "dir", dir, "channel", e.observe.Channel)
}

func (o *sessionObserver) prime() error {
	entries, err := os.ReadDir(o.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		o.files[filepath.Join(o.dir, entry.Name())] = &observerFileState{offset: info.Size()}
	}
	return nil
}

func (o *sessionObserver) run(ctx context.Context) {
	ticker := time.NewTicker(o.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := o.poll(ctx); err != nil {
				slog.Debug("observer poll failed", "dir", o.dir, "error", err)
			}
		}
	}
}

func (o *sessionObserver) poll(ctx context.Context) error {
	entries, err := os.ReadDir(o.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		if err := o.pollFile(ctx, filepath.Join(o.dir, entry.Name())); err != nil {
			slog.Debug("observer file poll failed", "path", entry.Name(), "error", err)
		}
	}
	return nil
}

func (o *sessionObserver) pollFile(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	o.mu.Lock()
	state := o.files[path]
	if state == nil {
		state = &observerFileState{}
		o.files[path] = state
	}
	if info.Size() < state.offset {
		state.offset = 0
		state.pending = nil
		state.skip = false
	}
	offset := state.offset
	pending := append([]byte(nil), state.pending...)
	skip := state.skip
	o.mu.Unlock()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	buf := append(pending, data...)
	lines := bytes.Split(buf, []byte{'\n'})
	newPending := []byte(nil)
	if len(lines) > 0 && len(lines[len(lines)-1]) > 0 {
		newPending = append(newPending, lines[len(lines)-1]...)
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		msg, shouldSkipFile, ok := o.parseLine(line)
		if shouldSkipFile {
			skip = true
			msg = ""
		}
		if skip || !ok || msg == "" {
			continue
		}
		if err := o.target.SendObservation(ctx, o.channelID, msg); err != nil {
			slog.Warn("observer send failed", "channel", o.channelID, "error", err)
		}
	}

	o.mu.Lock()
	state.offset = offset + int64(len(data))
	state.pending = newPending
	if skip {
		state.skip = true
		state.pending = nil
	}
	o.mu.Unlock()
	return nil
}

func (o *sessionObserver) parseLine(line []byte) (msg string, skipFile bool, ok bool) {
	var raw struct {
		Type       string `json:"type"`
		Entrypoint string `json:"entrypoint"`
		IsMeta     bool   `json:"isMeta"`
		Message    struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return "", false, false
	}
	if raw.Entrypoint == "sdk-cli" {
		return "", true, false
	}
	if raw.IsMeta {
		return "", false, false
	}
	if raw.Type != "user" && raw.Type != "assistant" {
		return "", false, false
	}
	text := extractObservationText(raw.Message.Content)
	if text == "" {
		return "", false, false
	}
	label := "user"
	if raw.Type == "assistant" {
		label = o.assistantLabel
	}
	return fmt.Sprintf("%s: %s", label, text), false, true
}

func extractObservationText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var direct string
	if err := json.Unmarshal(raw, &direct); err == nil {
		return strings.TrimSpace(stripObservationTags(direct))
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	var textParts []string
	for _, part := range parts {
		if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
			textParts = append(textParts, stripObservationTags(strings.TrimSpace(part.Text)))
		}
	}
	return strings.Join(textParts, "\n\n")
}

func stripObservationTags(s string) string {
	return observationXMLTagRe.ReplaceAllString(s, "")
}
