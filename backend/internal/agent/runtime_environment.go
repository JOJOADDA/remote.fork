package agent

import (
	"sort"
	"strings"
)

// RuntimeEnvironment returns deterministic KEY=value entries for
// backend-issued, per-run capabilities. Invalid environment names are ignored.
func RuntimeEnvironment(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		if validEnvironmentName(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return out
}

// WithRuntimeEnvironment overlays valid runtime values on a base environment
// without leaving duplicate keys whose lookup order varies by executable.
func WithRuntimeEnvironment(base []string, values map[string]string) []string {
	entries := RuntimeEnvironment(values)
	if len(entries) == 0 {
		return append([]string(nil), base...)
	}
	keys := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		key, _, _ := strings.Cut(entry, "=")
		keys[key] = struct{}{}
	}
	out := make([]string, 0, len(base)+len(entries))
	for _, entry := range base {
		key, _, _ := strings.Cut(entry, "=")
		if _, replaced := keys[key]; !replaced {
			out = append(out, entry)
		}
	}
	return append(out, entries...)
}

// WithOpenRouterClaudeEnvironment adds the environment expected by Claude Code
// when a project supplies an OpenRouter API key. Explicit Anthropic endpoint and
// auth-token values remain authoritative so users can configure another gateway.
func WithOpenRouterClaudeEnvironment(values map[string]string) map[string]string {
	out := make(map[string]string, len(values)+2)
	for key, value := range values {
		out[key] = value
	}
	if key := strings.TrimSpace(out["OPENROUTER_API_KEY"]); key != "" {
		if strings.TrimSpace(out["ANTHROPIC_BASE_URL"]) == "" {
			out["ANTHROPIC_BASE_URL"] = "https://openrouter.ai/api"
		}
		if strings.TrimSpace(out["ANTHROPIC_AUTH_TOKEN"]) == "" {
			out["ANTHROPIC_AUTH_TOKEN"] = key
		}
	}
	return out
}

func validEnvironmentName(value string) bool {
	if value == "" || strings.ContainsRune(value, '=') {
		return false
	}
	for index, char := range value {
		if (char >= 'A' && char <= 'Z') || char == '_' || (index > 0 && char >= '0' && char <= '9') {
			continue
		}
		return false
	}
	return true
}
