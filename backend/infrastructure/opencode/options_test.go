package opencode

import (
	"os"
	"strings"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

func TestBuildCLIArgs(t *testing.T) {
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("p1"),
		"user1",
		"anthropic",
		"Anthropic",
		"sk-test",
		"https://api.anthropic.com",
	)
	provider.SetDefaultModel("claude-sonnet-4")

	tests := []struct {
		name      string
		userInput string
		workDir   string
		provider  *domain.LLMProvider
		config    *domain.OpenCodeConfig
		sessionID string
		want      []string
	}{
		{
			name:      "basic with minimal config",
			userInput: "hello",
			workDir:   "",
			provider:  nil,
			config:    &domain.OpenCodeConfig{},
			sessionID: "",
			want:      []string{"run", "--format", "json", "--", "hello"},
		},
		{
			name:      "with model from config",
			userInput: "test",
			workDir:   "",
			provider:  nil,
			config:    &domain.OpenCodeConfig{Model: "claude-opus-4"},
			sessionID: "",
			want:      []string{"run", "--model", "claude-opus-4", "--format", "json", "--", "test"},
		},
		{
			name:      "model fallback to provider default",
			userInput: "test",
			workDir:   "",
			provider:  provider,
			config:    &domain.OpenCodeConfig{},
			sessionID: "",
			want:      []string{"run", "--model", "anthropic/claude-sonnet-4", "--format", "json", "--", "test"},
		},
		{
			name:      "workDir fallback",
			userInput: "test",
			workDir:   "/tmp/fallback",
			provider:  nil,
			config:    &domain.OpenCodeConfig{},
			sessionID: "",
			want:      []string{"run", "--dir", "/tmp/fallback", "--format", "json", "--", "test"},
		},
		{
			name:      "sessionID takes priority over continue",
			userInput: "test",
			workDir:   "",
			provider:  nil,
			config:    &domain.OpenCodeConfig{Continue: true},
			sessionID: "sess-123",
			want:      []string{"run", "--session", "sess-123", "--format", "json", "--", "test"},
		},
		{
			name:      "all boolean flags enabled",
			userInput: "test",
			workDir:   "",
			provider:  nil,
			config: &domain.OpenCodeConfig{
				Fork:            true,
				SkipPermissions: true,
				ShowThinking:    true,
				ShareSession:    true,
			},
			sessionID: "",
			want: []string{
				"run", "--fork", "--dangerously-skip-permissions",
				"--thinking", "--share", "--format", "json", "--", "test",
			},
		},
		{
			name:      "agent type and variant",
			userInput: "test",
			workDir:   "",
			provider:  nil,
			config: &domain.OpenCodeConfig{
				AgentType: domain.OpenCodeAgentTypeBuild,
				Variant:   "claude-sonnet-4-20250514",
			},
			sessionID: "",
			want:      []string{"run", "--agent", "build", "--variant", "claude-sonnet-4-20250514", "--format", "json", "--", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCLIArgs(tt.userInput, tt.workDir, tt.provider, tt.config, tt.sessionID)
			if len(got) != len(tt.want) {
				t.Fatalf("buildCLIArgs() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("buildCLIArgs()[%d] = %q, want %q, got=%v want=%v", i, got[i], tt.want[i], got, tt.want)
				}
			}
		})
	}
}

func diffEnv(base, result []string) []string {
	baseSet := make(map[string]bool, len(base))
	for _, e := range base {
		baseSet[e] = true
	}
	var added []string
	for _, e := range result {
		if !baseSet[e] {
			added = append(added, e)
		}
	}
	return added
}

func TestBuildEnv(t *testing.T) {
	provider, _ := domain.NewLLMProvider(
		domain.NewLLMProviderID("p1"),
		"user1",
		"anthropic",
		"Anthropic",
		"sk-secret",
		"https://api.example.com",
	)

	tests := []struct {
		name     string
		provider *domain.LLMProvider
		config   *domain.OpenCodeConfig
		contains []string
		missing  []string
	}{
		{
			name:     "provider only",
			provider: provider,
			config:   nil,
			contains: []string{"ANTHROPIC_AUTH_TOKEN=sk-secret", "ANTHROPIC_BASE_URL=https://api.example.com"},
			missing:  []string{"OPENCODE_SYSTEM_PROMPT"},
		},
		{
			name:     "config env and system prompt",
			provider: nil,
			config: &domain.OpenCodeConfig{
				Env:          map[string]string{"CUSTOM_VAR": "custom_value"},
				SystemPrompt: "You are helpful",
			},
			contains: []string{"CUSTOM_VAR=custom_value", "OPENCODE_SYSTEM_PROMPT=You are helpful"},
			missing:  []string{"ANTHROPIC_AUTH_TOKEN"},
		},
		{
			name:     "provider and config combined",
			provider: provider,
			config: &domain.OpenCodeConfig{
				Env:          map[string]string{"FOO": "bar"},
				SystemPrompt: "System prompt",
			},
			contains: []string{
				"ANTHROPIC_AUTH_TOKEN=sk-secret",
				"ANTHROPIC_BASE_URL=https://api.example.com",
				"FOO=bar",
				"OPENCODE_SYSTEM_PROMPT=System prompt",
			},
			missing: []string{},
		},
		{
			name:     "nil provider and nil config",
			provider: nil,
			config:   nil,
			contains: []string{},
			missing:  []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_BASE_URL", "OPENCODE_SYSTEM_PROMPT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildEnv(tt.provider, tt.config)
			added := diffEnv(os.Environ(), got)
			addedStr := strings.Join(added, "\n")
			for _, want := range tt.contains {
				if !strings.Contains(addedStr, want) {
					t.Errorf("buildEnv() missing %q in added env", want)
				}
			}
			for _, notWant := range tt.missing {
				if strings.Contains(addedStr, notWant) {
					t.Errorf("buildEnv() should not contain %q in added env", notWant)
				}
			}
		})
	}
}

func TestOpenCodeNotFoundError(t *testing.T) {
	err1 := &OpenCodeNotFoundError{}
	if err1.Error() != "opencode not found in PATH" {
		t.Errorf("unexpected error message: %q", err1.Error())
	}

	inner := &OpenCodeNotFoundError{err: &OpenCodeNotFoundError{}}
	want := "opencode not found: opencode not found in PATH"
	if inner.Error() != want {
		t.Errorf("unexpected error message: %q, want %q", inner.Error(), want)
	}
}
