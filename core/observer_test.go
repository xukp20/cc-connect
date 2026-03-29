package core

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

type stubObserverPlatform struct {
	stubPlatformEngine
	obsMu    sync.Mutex
	observed []string
}

func (p *stubObserverPlatform) SendObservation(_ context.Context, _ string, text string) error {
	p.obsMu.Lock()
	p.observed = append(p.observed, text)
	p.obsMu.Unlock()
	return nil
}

func (p *stubObserverPlatform) getObserved() []string {
	p.obsMu.Lock()
	defer p.obsMu.Unlock()
	return slices.Clone(p.observed)
}

type stubObservationAgent struct {
	stubAgent
	dir   string
	label string
}

func (a *stubObservationAgent) SessionObservationDir() (string, error) {
	return a.dir, nil
}

func (a *stubObservationAgent) SessionObservationAssistantName() string {
	return a.label
}

func TestSessionObserver_PrimeSkipsExistingAndForwardsNewLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "native.jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"old\"}]}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	target := &stubObserverPlatform{}
	obs := newSessionObserver(target, "C123", dir, "Claude", time.Millisecond)
	if err := obs.prime(); err != nil {
		t.Fatalf("prime: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"old\"}]}}\n{\"type\":\"user\",\"message\":{\"content\":\"what's up?\"}}\n{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"all good\"}]}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := obs.poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}

	got := target.getObserved()
	want := []string{"user: what's up?", "Claude: all good"}
	if !slices.Equal(got, want) {
		t.Fatalf("observed = %#v, want %#v", got, want)
	}
}

func TestSessionObserver_SkipsMetaAndSDKCLISessions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "native.jsonl")
	content := "" +
		"{\"type\":\"user\",\"isMeta\":true,\"message\":{\"content\":\"<local-command-caveat>ignore me</local-command-caveat>\"}}\n" +
		"{\"type\":\"assistant\",\"entrypoint\":\"sdk-cli\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"ignore sdk\"}]}}\n" +
		"{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"also ignored after sdk-cli\"}]}}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	target := &stubObserverPlatform{}
	obs := newSessionObserver(target, "C123", dir, "Claude", time.Millisecond)
	if err := obs.poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if got := target.getObserved(); len(got) != 0 {
		t.Fatalf("observed = %#v, want empty", got)
	}
}

func TestEngineStart_StartsObserverWhenConfigured(t *testing.T) {
	prev := defaultObservePollInterval
	defaultObservePollInterval = 5 * time.Millisecond
	defer func() { defaultObservePollInterval = prev }()

	dir := t.TempDir()
	agent := &stubObservationAgent{dir: dir, label: "Claude"}
	platform := &stubObserverPlatform{stubPlatformEngine: stubPlatformEngine{n: "slack"}}
	e := NewEngine("test", agent, []Platform{platform}, "", LangEnglish)
	e.SetObserveConfig(ObserveCfg{Enabled: true, Channel: "C123"})

	if err := e.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer e.Stop()

	path := filepath.Join(dir, "native.jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"hello team\"}]}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if got := platform.getObserved(); len(got) == 1 && got[0] == "Claude: hello team" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("observed = %#v, want Claude observation", platform.getObserved())
}
