package agent

import (
	"slices"
	"testing"
)

func TestRuntimeEnvironmentIsSortedAndRejectsInvalidNames(t *testing.T) {
	got := RuntimeEnvironment(map[string]string{
		"Z_TOKEN": "z",
		"A_URL":   "a",
		"bad-key": "ignored",
		"9BAD":    "ignored",
	})
	want := []string{"A_URL=a", "Z_TOKEN=z"}
	if !slices.Equal(got, want) {
		t.Fatalf("RuntimeEnvironment() = %#v, want %#v", got, want)
	}
}

func TestWithOpenRouterClaudeEnvironment(t *testing.T) {
	got := WithOpenRouterClaudeEnvironment(map[string]string{
		"OPENROUTER_API_KEY": "sk-or-test",
	})
	if got["ANTHROPIC_BASE_URL"] != "https://openrouter.ai/api" {
		t.Fatalf("ANTHROPIC_BASE_URL = %q", got["ANTHROPIC_BASE_URL"])
	}
	if got["ANTHROPIC_AUTH_TOKEN"] != "sk-or-test" {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN = %q", got["ANTHROPIC_AUTH_TOKEN"])
	}

	overridden := WithOpenRouterClaudeEnvironment(map[string]string{
		"OPENROUTER_API_KEY":   "sk-or-test",
		"ANTHROPIC_BASE_URL":   "https://gateway.test",
		"ANTHROPIC_AUTH_TOKEN": "explicit-token",
	})
	if overridden["ANTHROPIC_BASE_URL"] != "https://gateway.test" || overridden["ANTHROPIC_AUTH_TOKEN"] != "explicit-token" {
		t.Fatalf("explicit Claude environment was overwritten: %#v", overridden)
	}

	withoutKey := WithOpenRouterClaudeEnvironment(map[string]string{"OPENAI_API_KEY": "sk-test"})
	if _, ok := withoutKey["ANTHROPIC_BASE_URL"]; ok {
		t.Fatalf("unexpected Anthropic endpoint without OpenRouter key: %#v", withoutKey)
	}
}

func TestWithOpenRouterOpenAIEnvironment(t *testing.T) {
	got := WithOpenRouterOpenAIEnvironment(map[string]string{"OPENROUTER_API_KEY": "sk-or-test"})
	if got["OPENAI_API_KEY"] != "sk-or-test" || got["OPENAI_BASE_URL"] != "https://openrouter.ai/api/v1" {
		t.Fatalf("OpenRouter Codex mapping = %#v", got)
	}
	overridden := WithOpenRouterOpenAIEnvironment(map[string]string{
		"OPENROUTER_API_KEY": "sk-or-test",
		"OPENAI_API_KEY":     "explicit-key",
		"OPENAI_BASE_URL":    "https://gateway.test/v1",
	})
	if overridden["OPENAI_API_KEY"] != "explicit-key" || overridden["OPENAI_BASE_URL"] != "https://gateway.test/v1" {
		t.Fatalf("explicit OpenAI environment was overwritten: %#v", overridden)
	}
}

func TestWithRuntimeEnvironmentReplacesExistingValues(t *testing.T) {
	got := WithRuntimeEnvironment(
		[]string{"PATH=/bin", "REMOTE_SCHEDULE_GRANT=stale", "KEEP=yes"},
		map[string]string{"REMOTE_SCHEDULE_GRANT": "fresh"},
	)
	want := []string{"PATH=/bin", "KEEP=yes", "REMOTE_SCHEDULE_GRANT=fresh"}
	if !slices.Equal(got, want) {
		t.Fatalf("WithRuntimeEnvironment() = %#v, want %#v", got, want)
	}
}
