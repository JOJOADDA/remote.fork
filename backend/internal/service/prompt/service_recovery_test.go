package prompt

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	servicechat "github.com/futrx-com/remote.futrx.com/internal/service/chat"
	"github.com/futrx-com/remote.futrx.com/internal/service/runhub"
	"github.com/futrx-com/remote.futrx.com/internal/stores/filechat"
)

type recoveryProvider struct {
	requests []agent.RunRequest
}

func (p *recoveryProvider) ID() agent.ProviderID                     { return agent.ProviderCodex }
func (p *recoveryProvider) Parser(agent.RunRequest) agent.LineParser { return nil }
func (p *recoveryProvider) Run(_ context.Context, req agent.RunRequest, emit func(agent.Event)) error {
	p.requests = append(p.requests, req)
	if req.ResumeID != "" {
		return agent.ErrSessionNotFound
	}
	emit(agent.Event{
		T: time.Now().UnixMilli(), Type: agent.EventSessionUpdated,
		Provider: agent.ProviderCodex, SessionID: "new-thread",
	})
	emit(agent.Event{T: time.Now().UnixMilli(), Type: agent.EventAssistantTextDelta, Text: "recovered"})
	emit(agent.Event{T: time.Now().UnixMilli(), Type: agent.EventRunCompleted})
	return nil
}

func TestRunPromptRecoversMissingCodexSessionFromVisibleTranscript(t *testing.T) {
	ctx := context.Background()
	store, err := filechat.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	meta, err := store.Create(ctx, servicechat.Meta{
		ID: "abcdef123456", Title: "existing", Provider: servicechat.ProviderCodex,
		CodexSessionID: "missing-thread", Cwd: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []servicechat.Event{
		{T: 1, Type: "user", Text: "earlier question"},
		{T: 2, Type: "assistant_text", Text: "earlier answer"},
		{T: 3, Type: "complete"},
	} {
		if _, err := store.AppendEvent(ctx, meta.ID, event); err != nil {
			t.Fatal(err)
		}
	}

	provider := &recoveryProvider{}
	registry := agent.NewRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatal(err)
	}
	service := New(store, nil, nil, runhub.New(store), registry)
	emit := func(event ChatEvent) {
		if _, err := store.AppendEvent(ctx, meta.ID, event); err != nil {
			t.Fatal(err)
		}
	}
	service.runPrompt(ctx, meta.ID, "current question", emit, emit)

	if len(provider.requests) != 1 {
		t.Fatalf("requests = %d, want one fresh API turn", len(provider.requests))
	}
	if provider.requests[0].ResumeID != "" {
		t.Fatalf("resume id = %q, want empty for API turn", provider.requests[0].ResumeID)
	}
	for _, want := range []string{"earlier question", "earlier answer", "Current user request:\ncurrent question"} {
		if !strings.Contains(provider.requests[0].Prompt, want) {
			t.Fatalf("visible transcript missing %q:\n%s", want, provider.requests[0].Prompt)
		}
	}

	gotMeta, err := store.Get(ctx, meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotMeta.CodexSessionID != "new-thread" {
		t.Fatalf("session id = %q, want new-thread", gotMeta.CodexSessionID)
	}
	events, err := store.ReadEvents(ctx, meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	foundRecovery, foundAnswer := false, false
	for _, event := range events {
		foundRecovery = foundRecovery || event.Type == "system" && event.Subtype == "session_recovered"
		foundAnswer = foundAnswer || event.Type == "assistant_text" && event.Text == "recovered"
		if event.Type == "error" {
			t.Fatalf("unexpected recovery error: %s", event.Message)
		}
	}
	if foundRecovery || !foundAnswer {
		t.Fatalf("unexpected recovery markers or missing answer: %#v", events)
	}
}

func TestClearSessionIDForProvider(t *testing.T) {
	meta := &ChatMeta{ClaudeSessionID: "c", CodexSessionID: "o", KimiSessionID: "k"}
	clearSessionIDForProvider(meta, agent.ProviderCodex)
	if meta.CodexSessionID != "" || meta.ClaudeSessionID != "c" || meta.KimiSessionID != "k" {
		t.Fatalf("wrong provider session cleared: %#v", meta)
	}
}
