package prompt

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	servicechat "github.com/futrx-com/remote.futrx.com/internal/service/chat"
	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
	"github.com/futrx-com/remote.futrx.com/internal/service/runhub"
)

type ChatEvent = servicechat.Event
type ChatMeta = servicechat.Meta

type TmuxClient interface {
	Cwd(session string) (string, error)
}

// ProjectResolver decouples runner from project service internals. Lets tests
// stub project lookup/start without pulling in HTTP or persistence.
type ProjectResolver interface {
	Get(ctx context.Context, id serviceproject.ID) (serviceproject.Meta, error)
	Start(ctx context.Context, id serviceproject.ID) (serviceproject.Meta, error)
	ListSecrets(ctx context.Context, id serviceproject.ID) ([]serviceproject.Secret, error)
}

type agentBrowserActivityRecorder interface {
	TouchAgentBrowserActivity(ctx context.Context, id serviceproject.ID)
}

var ErrPromptAlreadyRunning = errors.New("a previous prompt is still running")

type Actor struct {
	Email   string
	IsAdmin bool
}

type StartInput struct {
	ChatID          servicechat.ID
	Prompt          string
	Actor           Actor
	ScheduledTaskID string
	ScheduledRunID  string
	ParentContext   context.Context
	// Autonomous keeps the run independent from a browser/WebSocket lifetime.
	// A nil ParentContext already has this behavior for backward compatibility.
	Autonomous bool
}

type RunResult struct {
	Output string
	Err    error
}

type RunHandle struct {
	ID   uint64
	Done <-chan RunResult
}

type ScheduleToolRequest struct {
	Actor           Actor
	ChatID          servicechat.ID
	ProjectID       serviceproject.ID
	ScheduledTaskID string
	ScheduledRunID  string
}

type ScheduleToolAccess struct {
	APIURL string
	Token  string
	Revoke func()
}

type ScheduleToolIssuer interface {
	IssueScheduleTool(context.Context, ScheduleToolRequest) (ScheduleToolAccess, error)
}

type Option func(*Service)

func WithScheduleToolIssuer(issuer ScheduleToolIssuer) Option {
	return func(service *Service) {
		service.scheduleTools = issuer
	}
}

type Service struct {
	store         servicechat.Repository
	tmux          TmuxClient
	projects      ProjectResolver
	hub           *runhub.Hub
	agents        *agent.Registry
	scheduleTools ScheduleToolIssuer
}

func New(
	store servicechat.Repository,
	tmux TmuxClient,
	projects ProjectResolver,
	hub *runhub.Hub,
	agents *agent.Registry,
	options ...Option,
) *Service {
	if hub == nil {
		hub = runhub.New(store)
	}
	service := &Service{
		store:    store,
		tmux:     tmux,
		projects: projects,
		hub:      hub,
		agents:   agents,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

func (rnr *Service) StartPrompt(id servicechat.ID, prompt string, emitTransient func(ChatEvent)) {
	_, _ = rnr.Start(StartInput{ChatID: id, Prompt: prompt}, emitTransient)
}

func (rnr *Service) Start(input StartInput, emitTransient func(ChatEvent)) (RunHandle, error) {
	if emitTransient == nil {
		emitTransient = func(ChatEvent) {}
	}
	parentCtx := input.ParentContext
	if input.Autonomous || parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)
	runID, ok := rnr.hub.StartRun(input.ChatID, cancel)
	if !ok {
		cancel()
		emitTransient(ChatEvent{
			T: time.Now().UnixMilli(), Type: "error",
			Message: "a previous prompt is still running — cancel first",
		})
		return RunHandle{}, ErrPromptAlreadyRunning
	}

	done := make(chan RunResult, 1)
	go func() {
		defer close(done)
		defer rnr.hub.FinishRun(input.ChatID, runID)
		var output strings.Builder
		err := rnr.runPromptAs(
			ctx,
			input,
			func(ev ChatEvent) {
				rnr.hub.Emit(input.ChatID, ev)
				if ev.Type == "assistant_text" {
					output.WriteString(ev.Text)
				}
			},
			emitTransient,
		)
		done <- RunResult{Output: output.String(), Err: err}
	}()
	return RunHandle{ID: runID, Done: done}, nil
}

func (rnr *Service) CancelPrompt(id servicechat.ID) bool {
	return rnr.hub.CancelRun(id)
}

func (rnr *Service) runPrompt(
	ctx context.Context,
	id servicechat.ID,
	prompt string,
	emit func(ChatEvent),
	emitTransient func(ChatEvent),
) error {
	return rnr.runPromptAs(ctx, StartInput{ChatID: id, Prompt: prompt}, emit, emitTransient)
}

func (rnr *Service) runPromptAs(
	ctx context.Context,
	input StartInput,
	emit func(ChatEvent),
	emitTransient func(ChatEvent),
) error {
	id := input.ChatID
	prompt := input.Prompt
	meta, err := rnr.store.Get(ctx, id)
	if err != nil {
		emitTransient(ChatEvent{T: time.Now().UnixMilli(), Type: "error", Message: err.Error()})
		return err
	}

	// Auto-title from first user prompt if still default.
	if meta.Title == "" || meta.Title == "New chat" {
		_, _ = rnr.store.Update(ctx, id, func(m *ChatMeta) {
			m.Title = servicechat.TitleFromPrompt(prompt)
		})
	}

	// Resolve a fresh cwd: live tmux pane_current_path if linked, else stored.
	cwd := meta.Cwd
	if meta.TmuxSession != "" {
		if c, err := rnr.tmux.Cwd(meta.TmuxSession); err == nil && c != "" {
			cwd = c
		}
	}
	if cwd == "" {
		cwd = os.Getenv("HOME")
		if cwd == "" {
			cwd = "/root"
		}
	}

	priorEvents, _ := rnr.store.ReadEvents(ctx, id)

	// Persist the user message before spawning the selected agent.
	emit(ChatEvent{T: time.Now().UnixMilli(), Type: "user", Text: prompt})

	providerID := providerIDFromChatProvider(meta.Provider)
	promptSkills := meta.SelectedSkills
	if input.ScheduledTaskID != "" && !hasScheduledTasksSkill(promptSkills) {
		promptSkills = append(
			append([]servicechat.SkillRef(nil), promptSkills...),
			servicechat.SkillRef{
				Name:     "Scheduled Tasks",
				Command:  scheduledTasksSkillName,
				Provider: servicechat.Provider(providerID),
				Source:   "remote",
			},
		)
	}
	resumeID := sessionIDForProvider(meta, providerID)
	effectivePrompt := promptForMode(meta.Mode, prompt)
	enableBrowser := hasBrowserSkill(meta.SelectedSkills)
	if enableBrowser && meta.ProjectID != "" {
		stopBrowserKeepalive := rnr.keepAgentBrowserActivity(ctx, serviceproject.ID(meta.ProjectID))
		defer stopBrowserKeepalive()
	}
	// Codex Responses API providers can reject native rollout replay when a
	// prior assistant item contains provider-specific content metadata. Use the
	// persisted visible transcript for every Codex turn instead.
	if resumeID == "" || providerID == agent.ProviderCodex {
		effectivePrompt = promptWithVisibleHistory(priorEvents, effectivePrompt)
	}
	resumeIDForRun := resumeID
	forkForRun := meta.ForkPending
	if providerID == agent.ProviderCodex {
		resumeIDForRun = ""
		forkForRun = false
	}
	effectivePrompt = promptWithSelectedSkills(providerID, promptSkills, effectivePrompt)

	provider := rnr.agents.Lookup(providerID)
	if provider == nil {
		err := errors.New(string(providerID) + " provider not configured")
		emit(ChatEvent{T: time.Now().UnixMilli(), Type: "error", Message: err.Error()})
		return err
	}

	enableScheduleTools := hasScheduledTasksSkill(meta.SelectedSkills) || input.ScheduledTaskID != ""
	runtimeEnv := map[string]string(nil)
	if enableScheduleTools {
		if meta.ProjectID == "" {
			err := errors.New("scheduled tasks are only available in project chats")
			emit(ChatEvent{T: time.Now().UnixMilli(), Type: "error", Message: err.Error()})
			return err
		}
		if rnr.scheduleTools == nil {
			err := errors.New("scheduled task tools are unavailable")
			emit(ChatEvent{T: time.Now().UnixMilli(), Type: "error", Message: err.Error()})
			return err
		}
		access, accessErr := rnr.scheduleTools.IssueScheduleTool(ctx, ScheduleToolRequest{
			Actor:           input.Actor,
			ChatID:          id,
			ProjectID:       serviceproject.ID(meta.ProjectID),
			ScheduledTaskID: input.ScheduledTaskID,
			ScheduledRunID:  input.ScheduledRunID,
		})
		if accessErr != nil {
			emit(ChatEvent{T: time.Now().UnixMilli(), Type: "error", Message: accessErr.Error()})
			return accessErr
		}
		if access.Revoke != nil {
			defer access.Revoke()
		}
		runtimeEnv = map[string]string{
			"REMOTE_SCHEDULE_API":   access.APIURL,
			"REMOTE_SCHEDULE_GRANT": access.Token,
		}
	}

	toolActivitySeen := false
	lastAssistantText := strings.Builder{}
	var pendingComplete *agent.Event
	run := func(runPrompt, runResumeID string) error {
		pendingComplete = nil
		return provider.Run(ctx, agent.RunRequest{
			Provider:       providerID,
			ConversationID: string(id),
			Prompt:         runPrompt,
			Cwd:            cwd,
			Model:          meta.Model,
			Mode:           meta.Mode,
			ResumeID:       runResumeID,
			ProjectID:      string(meta.ProjectID),
			Fork:           forkForRun,

			Preferences: agent.RunPreferences{
				ReasoningEffort: agent.ReasoningEffort(meta.ReasoningEffort),
				ServiceTier:     agent.ServiceTier(meta.ServiceTier),
			},
			EnableBrowser:       enableBrowser,
			EnableScheduleTools: enableScheduleTools,
			RuntimeEnv:          runtimeEnv,
		}, func(ev agent.Event) {
			switch ev.Type {
			case agent.EventToolStarted, agent.EventToolCompleted:
				toolActivitySeen = true
			case agent.EventAssistantTextDelta:
				lastAssistantText.WriteString(ev.Text)
			case agent.EventRunCompleted:
				copy := ev
				pendingComplete = &copy
				return
			}
			rnr.emitAgentEvent(ctx, id, ev, emit)
		})
	}

	err = run(effectivePrompt, resumeIDForRun)
	if err == nil && meta.Mode == "full-auto" && shouldContinueAfterPromise(prompt, lastAssistantText.String(), toolActivitySeen) {
		emit(ChatEvent{T: time.Now().UnixMilli(), Type: "system", Subtype: "autonomous_continuation", Message: "The agent acknowledged the request without using tools; continuing execution."})
		toolActivitySeen = false
		lastAssistantText.Reset()
		continuation := "Continue the implementation now. Do not acknowledge or describe future work. Use the available tools and complete the original request. Return only after the requested changes and verification are actually finished. Original request:\n\n" + prompt
		err = run(continuation, "")
	}
	if err == nil && pendingComplete != nil {
		rnr.emitAgentEvent(ctx, id, *pendingComplete, emit)
	}
	if errors.Is(err, agent.ErrSessionNotFound) && resumeIDForRun != "" {
		_, _ = rnr.store.Update(ctx, id, func(m *ChatMeta) {
			clearSessionIDForProvider(m, providerID)
			m.ForkPending = false
		})
		emit(ChatEvent{T: time.Now().UnixMilli(), Type: "system", Subtype: "session_recovered"})
		freshPrompt := promptForMode(meta.Mode, prompt)
		freshPrompt = promptWithVisibleHistory(priorEvents, freshPrompt)
		freshPrompt = promptWithSelectedSkills(providerID, promptSkills, freshPrompt)
		err = run(freshPrompt, "")
	}
	if err != nil && !errors.Is(err, agent.ErrRunFailed) {
		emit(ChatEvent{T: time.Now().UnixMilli(), Type: "error", Message: string(providerID) + " exit: " + err.Error()})
	}
	return err
}

func clearSessionIDForProvider(meta *ChatMeta, provider agent.ProviderID) {
	switch provider {
	case agent.ProviderCodex:
		meta.CodexSessionID = ""
	case agent.ProviderKimi:
		meta.KimiSessionID = ""
	case agent.ProviderAntigravity:
		meta.AntigravitySessionID = ""
	default:
		meta.ClaudeSessionID = ""
	}
}

func (rnr *Service) keepAgentBrowserActivity(ctx context.Context, projectID serviceproject.ID) func() {
	recorder, ok := rnr.projects.(agentBrowserActivityRecorder)
	if !ok || recorder == nil {
		return func() {}
	}
	recorder.TouchAgentBrowserActivity(ctx, projectID)
	keepaliveCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-keepaliveCtx.Done():
				return
			case <-ticker.C:
				recorder.TouchAgentBrowserActivity(keepaliveCtx, projectID)
			}
		}
	}()
	return cancel
}

func providerIDFromChatProvider(provider servicechat.Provider) agent.ProviderID {
	switch servicechat.NormalizeProvider(provider) {
	case servicechat.ProviderCodex:
		return agent.ProviderCodex
	case servicechat.ProviderKimi:
		return agent.ProviderKimi
	case servicechat.ProviderAntigravity:
		return agent.ProviderAntigravity
	default:
		return agent.ProviderClaude
	}
}

func sessionIDForProvider(meta ChatMeta, provider agent.ProviderID) string {
	switch provider {
	case agent.ProviderCodex:
		return meta.CodexSessionID
	case agent.ProviderKimi:
		return meta.KimiSessionID
	case agent.ProviderAntigravity:
		return meta.AntigravitySessionID
	default:
		return meta.ClaudeSessionID
	}
}

func shouldContinueAfterPromise(request, response string, toolActivitySeen bool) bool {
	if toolActivitySeen || strings.TrimSpace(response) == "" {
		return false
	}
	requestLower := strings.ToLower(request)
	responseLower := strings.ToLower(response)
	implementationRequest := containsAny(requestLower, []string{
		"implement", "create", "build", "add", "fix", "modify", "update", "refactor", "write", "deploy", "run", "test",
		"نفذ", "أنشئ", "انشئ", "ابن", "أضف", "اضف", "أصلح", "اصلح", "عدل", "حدّث", "حدث", "اكتب", "شغّل", "شغل", "اختبر",
	})
	promise := containsAny(responseLower, []string{
		"i will", "i'll", "let me", "i am going to", "i'm going to", "i'll get back", "i will return",
		"سأنفذ", "ساقوم", "سأقوم", "سأبدأ", "سأعمل", "سأعود", "سأضيف", "سأجهز", "سأكمل", "سوف أنفذ", "سوف أقوم",
	})
	return implementationRequest && promise
}

func containsAny(value string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func promptForMode(mode, prompt string) string {
	switch mode {
	case "plan":
		return "Work in planning mode. Inspect enough context to be concrete, then propose the implementation plan before changing files. Avoid file edits until the user asks you to proceed.\n\n" + prompt
	case "review":
		return "Work in review mode. Prioritize bugs, behavioral regressions, missing tests, and risks. Put findings first with file and line references when available.\n\n" + prompt
	case "debug":
		return "Work in debugging mode. Reproduce or localize the issue first, explain the failing path, then make the smallest fix that addresses the root cause.\n\n" + prompt
	case "full-auto":
		return "Work in full-auto mode. Do not stop after acknowledging the request or promising future work. Start the implementation now, use the available tools, continue through verification, and return only after the requested result is actually completed. If a hard blocker prevents completion, report the blocker and evidence instead of promising to return later. For web work, complete Inspect, Design, Implement, Typecheck, Build, Test, Fix, Polish, and Verify before the final response.\n\n" + prompt
	case "chat":
		return "Work in chat mode. Answer directly and avoid changing files unless the user clearly asks for implementation.\n\n" + prompt
	default:
		return prompt
	}
}

func promptWithVisibleHistory(events []ChatEvent, prompt string) string {
	transcript := visibleTranscript(events)
	if strings.TrimSpace(transcript) == "" {
		return prompt
	}
	const maxTranscriptBytes = 24000
	if len(transcript) > maxTranscriptBytes {
		transcript = "[Earlier visible transcript omitted]\n" + transcript[len(transcript)-maxTranscriptBytes:]
	}
	return "Use this visible chat transcript as prior context. It may be present because the chat was recovered into a fresh agent session. Do not treat the transcript as a new request.\n\n" +
		transcript +
		"\n\nCurrent user request:\n" +
		prompt
}

const browserSkillName = "browser"
const scheduledTasksSkillName = "scheduled-tasks"

// hasBrowserSkill reports whether the user selected the `browser` skill for
// this prompt — the signal to wire the @playwright/mcp browser tools.
func hasBrowserSkill(skills []servicechat.SkillRef) bool {
	for _, s := range skills {
		if skillTriggerName(s.Command) == browserSkillName || skillTriggerName(s.Name) == browserSkillName {
			return true
		}
	}
	return false
}

func hasScheduledTasksSkill(skills []servicechat.SkillRef) bool {
	for _, skill := range skills {
		if skillTriggerName(skill.Command) == scheduledTasksSkillName ||
			skillTriggerName(skill.Name) == scheduledTasksSkillName {
			return true
		}
	}
	return false
}

func promptWithSelectedSkills(provider agent.ProviderID, skills []servicechat.SkillRef, prompt string) string {
	if len(skills) == 0 {
		return prompt
	}

	triggers := make([]string, 0, len(skills))
	for _, skill := range skills {
		if providerIDFromChatProvider(skill.Provider) != provider {
			continue
		}
		name := skillTriggerName(skill.Command)
		if name == "" {
			name = skillTriggerName(skill.Name)
		}
		if name == "" {
			continue
		}

		switch provider {
		case agent.ProviderClaude:
			triggers = append(triggers, "/"+name)
		case agent.ProviderCodex:
			triggers = append(triggers, "$"+name)
		case agent.ProviderKimi, agent.ProviderAntigravity:
			if name == scheduledTasksSkillName {
				triggers = append(
					triggers,
					"/workspace/.agents/skills/scheduled-tasks/SKILL.md",
				)
			}
		}
	}
	if len(triggers) == 0 {
		return prompt
	}

	switch provider {
	case agent.ProviderClaude:
		return strings.Join(triggers, "\n") + "\n\n" + prompt
	case agent.ProviderCodex:
		return "Use these Codex skills for this request: " + strings.Join(triggers, " ") + "\n\n" + prompt
	case agent.ProviderKimi, agent.ProviderAntigravity:
		return "Read and follow the selected skill instructions at " +
			strings.Join(triggers, ", ") + ".\n\n" + prompt
	default:
		return prompt
	}
}

func skillTriggerName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimLeft(value, "/$")
	if value == "" {
		return ""
	}
	parts := strings.Fields(value)
	if len(parts) <= 1 {
		return value
	}
	return strings.Join(parts, "-")
}

func visibleTranscript(events []ChatEvent) string {
	var out strings.Builder
	var assistant strings.Builder

	flushAssistant := func() {
		text := strings.TrimSpace(assistant.String())
		if text == "" {
			assistant.Reset()
			return
		}
		out.WriteString("Assistant:\n")
		out.WriteString(text)
		out.WriteString("\n\n")
		assistant.Reset()
	}

	for _, ev := range events {
		switch ev.Type {
		case "user":
			flushAssistant()
			out.WriteString("User:\n")
			out.WriteString(strings.TrimSpace(ev.Text))
			out.WriteString("\n\n")
		case "assistant_text":
			assistant.WriteString(ev.Text)
		case "complete", "error":
			flushAssistant()
		}
	}
	flushAssistant()
	return out.String()
}
