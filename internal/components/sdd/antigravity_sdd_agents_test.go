package sdd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v2/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v2/internal/assets"
	"github.com/gentleman-programming/gentle-ai/v2/internal/model"
)

func TestAntigravitySddAgentsPluginWritesManifestAndHooks(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".gemini", "antigravity-cli"), 0o755); err != nil {
		t.Fatal(err)
	}

	changed, files, err := installAntigravitySddAgentsPlugin(home)
	if err != nil {
		t.Fatalf("installAntigravitySddAgentsPlugin() error = %v", err)
	}
	if !changed {
		t.Fatalf("installAntigravitySddAgentsPlugin() changed = false, want true")
	}
	if len(files) != 2 {
		t.Fatalf("installAntigravitySddAgentsPlugin() files = %d, want 2", len(files))
	}

	pluginPath := filepath.Join(home, ".gemini", "antigravity-cli", "plugins", "gentle-ai-sdd-agents", "plugin.json")
	hooksPath := filepath.Join(home, ".gemini", "antigravity-cli", "plugins", "gentle-ai-sdd-agents", "hooks.json")

	for _, want := range []string{pluginPath, hooksPath} {
		if !containsString(files, want) {
			t.Fatalf("installAntigravitySddAgentsPlugin() files missing %q; got %v", want, files)
		}
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("expected %q to exist on disk: %v", want, err)
		}
	}

	manifest, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("ReadFile(plugin.json) error = %v", err)
	}
	for _, want := range []string{`"name": "gentle-ai-sdd-agents"`, `"version": "0.1.0"`} {
		if !strings.Contains(string(manifest), want) {
			t.Fatalf("plugin.json missing %q:\n%s", want, manifest)
		}
	}

	hooks, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("ReadFile(hooks.json) error = %v", err)
	}
	for _, want := range []string{
		`"gentle-ai-sdd-agents-hardening"`,
		`"PreInvocation"`,
		`"PreToolUse"`,
		`"matcher": "run_command"`,
		"Destructive command blocked by Gentle AI SDD guard",
		"printf",
		"sdd-explore",
		"sdd-apply",
		"review-*",
		"jd-judge-*",
		"jd-fix-agent",
		"fail closed",
	} {
		if !strings.Contains(string(hooks), want) {
			t.Fatalf("hooks.json missing %q:\n%s", want, hooks)
		}
	}
}

func TestAntigravitySddAgentsPluginMergesIntoExistingHooks(t *testing.T) {
	home := t.TempDir()
	hooksPath := filepath.Join(home, ".gemini", "antigravity-cli", "plugins", "gentle-ai-sdd-agents", "hooks.json")
	if err := os.MkdirAll(filepath.Dir(hooksPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing custom hook must be preserved after install.
	preExisting := `{"custom-hook":{"PreInvocation":[{"type":"command","command":"echo custom"}],"PreToolUse":[{"matcher":"custom_tool","hooks":[{"type":"command","command":"echo tool"}]}]}}`
	if err := os.WriteFile(hooksPath, []byte(preExisting), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := installAntigravitySddAgentsPlugin(home)
	if err != nil {
		t.Fatalf("installAntigravitySddAgentsPlugin() error = %v", err)
	}
	if !changed {
		t.Fatal("installAntigravitySddAgentsPlugin() changed = false on first install over existing hooks")
	}

	merged, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("ReadFile(hooks.json) error = %v", err)
	}
	for _, want := range []string{
		`"custom-hook"`,
		"echo custom",
		"custom_tool",
		"echo tool",
		`"gentle-ai-sdd-agents-hardening"`,
		`"PreInvocation"`,
		`"PreToolUse"`,
		"sdd-explore",
		"run_command",
	} {
		if !strings.Contains(string(merged), want) {
			t.Fatalf("merged hooks.json missing %q:\n%s", want, merged)
		}
	}
}

func TestAntigravitySddAgentsPluginIdempotent(t *testing.T) {
	home := t.TempDir()

	if _, _, err := installAntigravitySddAgentsPlugin(home); err != nil {
		t.Fatalf("first install error = %v", err)
	}
	second, files, err := installAntigravitySddAgentsPlugin(home)
	if err != nil {
		t.Fatalf("second install error = %v", err)
	}
	if second {
		t.Fatalf("second install changed = true, want false (idempotent)")
	}
	if len(files) != 2 {
		t.Fatalf("second install files = %d, want 2", len(files))
	}
}

func TestAntigravitySddAgentsHardeningContractPhrases(t *testing.T) {
	msg := AntigravitySddAgentsHardeningMessage()
	if strings.Contains(msg, "'") {
		t.Fatalf("hardening message must not contain single quotes because hooks.json embeds it in a single-quoted shell string: %q", msg)
	}
	for _, want := range antigravitySddAgentsHardeningContractPhrases {
		if !strings.Contains(msg, want) {
			t.Errorf("hardening message missing required phrase %q", want)
		}
	}
	// Assert TDD contract phrases specifically
	for _, want := range []string{"Strict TDD", "Red phase", "Red-Green-Refactor"} {
		if !strings.Contains(msg, want) {
			t.Errorf("hardening message missing required TDD phrase %q", want)
		}
	}
	// Assert Engram dual-path persistence contract phrases specifically
	for _, want := range []string{
		"enable_mcp_tools: true",
		"fallback persistence",
		"call_mcp_tool",
		`ServerName="engram"`,
		`ToolName="mem_save"`,
		"direct MCP access",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("hardening message missing required Engram persistence phrase %q", want)
		}
	}
	for _, forbidden := range antigravitySddAgentsHardeningContractForbids {
		if strings.Contains(msg, forbidden) {
			t.Errorf("hardening message must not contain %q (it would fake a real Antigravity API)", forbidden)
		}
	}
}

func TestAntigravityOrchestratorAssetContainsEngramPersistenceContract(t *testing.T) {
	content, err := assets.FS.ReadFile("antigravity/sdd-orchestrator.md")
	if err != nil {
		t.Fatalf("ReadFile(antigravity/sdd-orchestrator.md) error = %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"enable_mcp_tools: true",
		"mem_save",
		"sdd/{change-name}/{artifact-type}",
		"fallback persistence",
		"call_mcp_tool",
		`ServerName="engram"`,
		`ToolName="mem_save"`,
		"direct MCP access",
		"enable_write_tools",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("sdd-orchestrator.md missing required engram persistence directive %q", want)
		}
	}
}

func TestAntigravitySddAgentsRoleAllowed(t *testing.T) {
	allowed := []string{
		"sdd-explore", "sdd-spec", "sdd-design", "sdd-tasks", "sdd-apply",
		"sdd-verify", "sdd-archive", "sdd-onboard", "sdd-init", "sdd-propose",
		"review-risk", "review-resilience", "review-readability", "review-reliability",
		"review-refuter",
		"jd-judge-a", "jd-judge-b", "jd-fix-agent",
	}
	for _, role := range allowed {
		if !antigravitySddAgentsRoleAllowed(role) {
			t.Errorf("role %q must be allowed; got false", role)
		}
	}

	// Roles not in the canonical table must NOT be allowed. Locking this
	// prevents accidental widening of the hardening contract.
	for _, role := range []string{"sdd-evil", "review-anything", "jd-super-user", ""} {
		if antigravitySddAgentsRoleAllowed(role) {
			t.Errorf("role %q must NOT be allowed; got true", role)
		}
	}
}

func TestAntigravitySddAgentsRoleScopesSortedAndUnique(t *testing.T) {
	scopes := sortedAntigravitySddAgentsRoleScopes()
	if len(scopes) != len(antigravitySddAgentsRoleScopes) {
		t.Fatalf("sorted length %d != canonical length %d", len(scopes), len(antigravitySddAgentsRoleScopes))
	}
	seen := map[string]bool{}
	for i, entry := range scopes {
		if seen[entry.Role] {
			t.Errorf("duplicate role %q in scope table", entry.Role)
		}
		seen[entry.Role] = true
		if i > 0 && scopes[i-1].Role > entry.Role {
			t.Errorf("scope table not sorted: %q > %q", scopes[i-1].Role, entry.Role)
		}
		if entry.Scope == "" {
			t.Errorf("role %q has empty scope", entry.Role)
		}
	}
}

func TestHasAntigravitySddAgentsHardeningContractRejectsPhrasesOutsideManagedBlock(t *testing.T) {
	home := t.TempDir()
	hooksPath := filepath.Join(antigravitySddAgentsPluginDir(home), "hooks.json")
	if err := os.MkdirAll(filepath.Dir(hooksPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
  "gentle-ai-sdd-agents-hardening": {
    "PreInvocation": [
      {"type":"command","command":"printf empty"}
    ]
  },
  "unrelated": {
    "PreInvocation": [
      {"type":"command","command":"sdd-explore sdd-propose sdd-apply sdd-verify review-* jd-judge-* jd-fix-agent fail closed CodeGraph"}
    ]
  }
}`
	if err := os.WriteFile(hooksPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if HasAntigravitySddAgentsHardeningContract(home) {
		t.Fatal("HasAntigravitySddAgentsHardeningContract() = true when required phrases only appear outside managed block")
	}
}

func TestHasAntigravitySddAgentsHardeningContract(t *testing.T) {
	home := t.TempDir()
	if HasAntigravitySddAgentsHardeningContract(home) {
		t.Fatal("HasAntigravitySddAgentsHardeningContract() = true before install, want false")
	}

	if _, _, err := installAntigravitySddAgentsPlugin(home); err != nil {
		t.Fatalf("installAntigravitySddAgentsPlugin() error = %v", err)
	}
	if !HasAntigravitySddAgentsHardeningContract(home) {
		t.Fatal("HasAntigravitySddAgentsHardeningContract() = false after install, want true")
	}
}

func TestAntigravitySddAgentsPluginDoesNotAffectOtherAgents(t *testing.T) {
	// Sanity: installAntigravitySddAgentsPlugin must NEVER touch paths
	// outside ~/.gemini/antigravity-cli/plugins/gentle-ai-sdd-agents.
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".gemini", "antigravity-cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, files, err := installAntigravitySddAgentsPlugin(home)
	if err != nil {
		t.Fatalf("installAntigravitySddAgentsPlugin() error = %v", err)
	}
	for _, f := range files {
		clean := filepath.Clean(f)
		prefix := filepath.Clean(filepath.Join(home, ".gemini", "antigravity-cli", "plugins", "gentle-ai-sdd-agents")) + string(os.PathSeparator)
		if !strings.HasPrefix(clean, prefix) {
			t.Fatalf("installAntigravitySddAgentsPlugin() wrote %q outside the gentle-ai-sdd-agents plugin dir", clean)
		}
		if strings.Contains(clean, "settings.json") {
			t.Fatalf("installAntigravitySddAgentsPlugin() must not touch user settings.json; got %q", clean)
		}
		if strings.Contains(clean, "mcp_config.json") {
			t.Fatalf("installAntigravitySddAgentsPlugin() must not touch user mcp_config.json; got %q", clean)
		}
	}
}

func TestAntigravitySddAgentsPluginScopingDynamic(t *testing.T) {
	// If only desktop directory exists, it must write to desktop.
	home := t.TempDir()
	desktopDir := filepath.Join(home, ".gemini", "antigravity-desktop")
	if err := os.MkdirAll(desktopDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, err := installAntigravitySddAgentsPlugin(home)
	if err != nil {
		t.Fatalf("installAntigravitySddAgentsPlugin() error = %v", err)
	}

	desktopPlugin := filepath.Join(desktopDir, "plugins", "gentle-ai-sdd-agents", "plugin.json")
	if _, err := os.Stat(desktopPlugin); err != nil {
		t.Fatalf("expected plugin.json to be written to antigravity-desktop: %v", err)
	}

	cliPlugin := filepath.Join(home, ".gemini", "antigravity-cli", "plugins", "gentle-ai-sdd-agents", "plugin.json")
	if _, err := os.Stat(cliPlugin); !os.IsNotExist(err) {
		t.Fatalf("must NOT write to antigravity-cli when desktop is active; stat err = %v", err)
	}
}

func TestInjectAntigravityInstallsSddAgentsHardeningPlugin(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".gemini", "antigravity-cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter, err := agents.NewAdapter(model.AgentAntigravity)
	if err != nil {
		t.Fatalf("NewAdapter(antigravity) error = %v", err)
	}

	result, err := Inject(home, adapter, "")
	if err != nil {
		t.Fatalf("Inject(antigravity) error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject(antigravity) changed = false, want true")
	}

	pluginPath := filepath.Join(home, ".gemini", "antigravity-cli", "plugins", "gentle-ai-sdd-agents", "plugin.json")
	hooksPath := filepath.Join(home, ".gemini", "antigravity-cli", "plugins", "gentle-ai-sdd-agents", "hooks.json")
	for _, want := range []string{pluginPath, hooksPath} {
		if !containsString(result.Files, want) {
			t.Fatalf("Inject(antigravity).Files missing %q; got %v", want, result.Files)
		}
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("expected %q to exist after Inject: %v", want, err)
		}
	}

	if !HasAntigravitySddAgentsHardeningContract(home) {
		t.Fatal("HasAntigravitySddAgentsHardeningContract() = false after Inject, want true")
	}

	// Sanity: the hooks.json must be valid JSON with the expected schema.
	hooksBody, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("ReadFile(hooks.json) error = %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(hooksBody, &parsed); err != nil {
		t.Fatalf("hooks.json is not valid JSON: %v\n%s", err, hooksBody)
	}
	block, ok := parsed["gentle-ai-sdd-agents-hardening"].(map[string]any)
	if !ok {
		t.Fatalf("hooks.json missing gentle-ai-sdd-agents-hardening block: %s", hooksBody)
	}
	pre, ok := block["PreInvocation"].([]any)
	if !ok || len(pre) == 0 {
		t.Fatalf("hooks.json missing PreInvocation list: %s", hooksBody)
	}
	preTool, ok := block["PreToolUse"].([]any)
	if !ok || len(preTool) == 0 {
		t.Fatalf("hooks.json missing PreToolUse list: %s", hooksBody)
	}
	entry, ok := preTool[0].(map[string]any)
	if !ok || entry["matcher"] != "run_command" {
		t.Fatalf("hooks.json PreToolUse[0] missing matcher 'run_command': %s", hooksBody)
	}
}

func TestInjectDoesNotInstallSddAgentsPluginForNonAntigravity(t *testing.T) {
	// Lock the scope: only Antigravity gets the gentle-ai-sdd-agents plugin.
	// Other agents must not see plugin/hooks.json appear in their
	// injection result.
	cases := []struct {
		name    string
		agentID string
		invoke  func(home string) error
	}{
		{
			name:    "opencode",
			agentID: "opencode",
			invoke: func(home string) error {
				adapter, err := agents.NewAdapter("opencode")
				if err != nil {
					return err
				}
				_, err = Inject(home, adapter, "single")
				return err
			},
		},
		{
			name:    "claude",
			agentID: "claude-code",
			invoke: func(home string) error {
				adapter, err := agents.NewAdapter("claude-code")
				if err != nil {
					return err
				}
				_, err = Inject(home, adapter, "")
				return err
			},
		},
		{
			name:    "gemini-cli",
			agentID: "gemini-cli",
			invoke: func(home string) error {
				adapter, err := agents.NewAdapter("gemini-cli")
				if err != nil {
					return err
				}
				_, err = Inject(home, adapter, "")
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if err := tc.invoke(home); err != nil {
				t.Fatalf("Inject(%s) error = %v", tc.agentID, err)
			}
			// The plugin path must NOT be created for other
			// agents — it is Antigravity-only.
			for _, variant := range []string{"antigravity-cli", "antigravity-desktop", "antigravity"} {
				pluginPath := filepath.Join(home, ".gemini", variant, "plugins", "gentle-ai-sdd-agents", "plugin.json")
				if _, err := os.Stat(pluginPath); !os.IsNotExist(err) {
					t.Fatalf("Inject(%s) wrote Antigravity-only plugin path %q; stat err = %v", tc.agentID, pluginPath, err)
				}
			}
		})
	}
}

func TestTrustWorkspaceInAntigravitySettings(t *testing.T) {
	home := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentAntigravity)
	if err != nil {
		t.Fatalf("NewAdapter(antigravity) error = %v", err)
	}

	settingsDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Test when workspaceDir is empty
	changed, files, err := trustWorkspaceInAntigravitySettings(home, "", adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed || len(files) > 0 {
		t.Fatalf("expected no change when workspaceDir is empty; changed=%v, files=%v", changed, files)
	}

	// 2. Test when settings.json is missing (should be created with trustedWorkspaces)
	workspace := filepath.Join(home, "my-new-project")
	changed, files, err = trustWorkspaceInAntigravitySettings(home, workspace, adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed to be true")
	}
	wantFile := filepath.Join(settingsDir, "settings.json")
	if len(files) != 1 || files[0] != wantFile {
		t.Fatalf("expected changed files list to contain %q; got %v", wantFile, files)
	}

	// Verify content
	data, err := os.ReadFile(wantFile)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	trusted, ok := parsed["trustedWorkspaces"].([]any)
	if !ok || len(trusted) != 1 || trusted[0].(string) != filepath.Clean(workspace) {
		t.Fatalf("expected trustedWorkspaces to contain clean workspace path; got: %v", parsed)
	}

	// 3. Test idempotency (should not change if already present)
	changed, files, err = trustWorkspaceInAntigravitySettings(home, workspace, adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed || len(files) > 0 {
		t.Fatalf("expected no change on subsequent calls; changed=%v, files=%v", changed, files)
	}
}

// TestTrustWorkspaceInAntigravitySettings_MalformedJSONNoWrite guards against
// silent data loss: when settings.json already exists but is not valid JSON,
// the function MUST return a clear error and MUST NOT overwrite the file with
// an empty object. The user's existing (malformed) bytes must remain untouched.
func TestTrustWorkspaceInAntigravitySettings_MalformedJSONNoWrite(t *testing.T) {
	home := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentAntigravity)
	if err != nil {
		t.Fatalf("NewAdapter(antigravity) error = %v", err)
	}

	settingsDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(settingsDir, "settings.json")

	original := []byte("{this is not valid json")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	workspace := filepath.Join(home, "my-project")
	changed, files, err := trustWorkspaceInAntigravitySettings(home, workspace, adapter)
	if err == nil {
		t.Fatalf("expected error for malformed JSON, got nil (changed=%v, files=%v)", changed, files)
	}
	if changed {
		t.Fatalf("expected changed=false on error; got true")
	}
	if len(files) != 0 {
		t.Fatalf("expected no files reported on error; got %v", files)
	}

	got, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		t.Fatalf("ReadFile(settings.json) error = %v", readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("settings.json must remain untouched on malformed-JSON error\n before: %q\n after:  %q", original, got)
	}
}

// TestTrustWorkspaceInAntigravitySettings_NonObjectJSONNoWrite locks the
// guardrail for non-object top-level values (arrays, strings, numbers).
// Those MUST NOT be silently replaced with an empty object + trustedWorkspaces
// because that destroys the user's existing configuration.
func TestTrustWorkspaceInAntigravitySettings_EmptyExistingSettingsNoWrite(t *testing.T) {
	home := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentAntigravity)
	if err != nil {
		t.Fatalf("NewAdapter(antigravity) error = %v", err)
	}

	settingsDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(settingsDir, "settings.json")

	if err := os.WriteFile(settingsPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	workspace := filepath.Join(home, "my-project")
	changed, files, err := trustWorkspaceInAntigravitySettings(home, workspace, adapter)
	if err == nil {
		t.Fatalf("expected error for empty settings.json, got nil (changed=%v, files=%v)", changed, files)
	}
	if changed {
		t.Fatalf("expected changed=false on error; got true")
	}
	if len(files) != 0 {
		t.Fatalf("expected no files reported on error; got %v", files)
	}

	got, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		t.Fatalf("ReadFile(settings.json) error = %v", readErr)
	}
	if len(got) != 0 {
		t.Fatalf("settings.json must remain empty on empty-file error; got %q", got)
	}
}

func TestTrustWorkspaceInAntigravitySettings_NonObjectJSONNoWrite(t *testing.T) {
	home := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentAntigravity)
	if err != nil {
		t.Fatalf("NewAdapter(antigravity) error = %v", err)
	}

	cases := []struct {
		name  string
		bytes []byte
	}{
		{"array top-level", []byte("[1, 2, 3]")},
		{"string top-level", []byte(`"just a string"`)},
		{"number top-level", []byte(`42`)},
		{"bool top-level", []byte(`true`)},
		{"null top-level", []byte(`null`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settingsDir := filepath.Join(home, ".gemini", "antigravity-cli")
			if err := os.MkdirAll(settingsDir, 0o755); err != nil {
				t.Fatal(err)
			}
			settingsPath := filepath.Join(settingsDir, "settings.json")

			if err := os.WriteFile(settingsPath, tc.bytes, 0o644); err != nil {
				t.Fatal(err)
			}

			workspace := filepath.Join(home, "ws-"+tc.name)
			changed, files, err := trustWorkspaceInAntigravitySettings(home, workspace, adapter)
			if err == nil {
				t.Fatalf("expected error for non-object top-level JSON, got nil (changed=%v, files=%v)", changed, files)
			}
			if changed {
				t.Fatalf("expected changed=false on error; got true")
			}
			if len(files) != 0 {
				t.Fatalf("expected no files reported on error; got %v", files)
			}

			got, readErr := os.ReadFile(settingsPath)
			if readErr != nil {
				t.Fatalf("ReadFile(settings.json) error = %v", readErr)
			}
			if string(got) != string(tc.bytes) {
				t.Fatalf("settings.json must remain untouched on non-object error\n before: %q\n after:  %q", tc.bytes, got)
			}
		})
	}
}

// TestTrustWorkspaceInAntigravitySettings_NonArrayTrustedWorkspacesNoWrite
// guards against silent data loss when trustedWorkspaces exists with an
// unexpected non-array shape (string, object, number, null). The function
// MUST return a clear error and MUST NOT replace the existing key with a
// fresh array — that would destroy the user's existing trust config.
func TestTrustWorkspaceInAntigravitySettings_NonArrayTrustedWorkspacesNoWrite(t *testing.T) {
	home := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentAntigravity)
	if err != nil {
		t.Fatalf("NewAdapter(antigravity) error = %v", err)
	}

	cases := []struct {
		name string
		body string
	}{
		{"trustedWorkspaces is string", `{"trustedWorkspaces": "oops"}`},
		{"trustedWorkspaces is object", `{"trustedWorkspaces": {"a": "/some/path"}}`},
		{"trustedWorkspaces is number", `{"trustedWorkspaces": 7}`},
		{"trustedWorkspaces is null", `{"trustedWorkspaces": null}`},
		{"trustedWorkspaces is bool", `{"trustedWorkspaces": true}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settingsDir := filepath.Join(home, ".gemini", "antigravity-cli")
			if err := os.MkdirAll(settingsDir, 0o755); err != nil {
				t.Fatal(err)
			}
			settingsPath := filepath.Join(settingsDir, "settings.json")

			if err := os.WriteFile(settingsPath, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}

			workspace := filepath.Join(home, "ws-"+tc.name)
			changed, files, err := trustWorkspaceInAntigravitySettings(home, workspace, adapter)
			if err == nil {
				t.Fatalf("expected error for non-array trustedWorkspaces, got nil (changed=%v, files=%v)", changed, files)
			}
			if changed {
				t.Fatalf("expected changed=false on error; got true")
			}
			if len(files) != 0 {
				t.Fatalf("expected no files reported on error; got %v", files)
			}

			got, readErr := os.ReadFile(settingsPath)
			if readErr != nil {
				t.Fatalf("ReadFile(settings.json) error = %v", readErr)
			}
			if string(got) != tc.body {
				t.Fatalf("settings.json must remain untouched on non-array trustedWorkspaces error\n before: %s\n after:  %s", tc.body, got)
			}
		})
	}
}

// TestTrustWorkspaceInAntigravitySettings_PreservesUnrelatedKeys verifies the
// happy path does NOT clobber unrelated top-level keys the user already has
// in their settings.json. This locks the preservation invariant so future
// refactors cannot silently regress to a replacement that drops keys.
func TestTrustWorkspaceInAntigravitySettings_PreservesUnrelatedKeys(t *testing.T) {
	home := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentAntigravity)
	if err != nil {
		t.Fatalf("NewAdapter(antigravity) error = %v", err)
	}

	settingsDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(settingsDir, "settings.json")

	existing := map[string]any{
		"editor.fontSize":              14,
		"editor.tabSize":               4,
		"window.theme":                 "dark",
		"extensions.autoUpdate":        true,
		"security.workspace.trust":     false,
		"telemetry.telemetryLevel":     "off",
		"nested.deeply.structured.key": []any{"alpha", "beta", float64(7)},
	}
	body, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, body, 0o644); err != nil {
		t.Fatal(err)
	}

	workspace := filepath.Join(home, "preserved-keys-project")
	changed, files, err := trustWorkspaceInAntigravitySettings(home, workspace, adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when adding a new workspace to existing settings")
	}
	wantFile := settingsPath
	if len(files) != 1 || files[0] != wantFile {
		t.Fatalf("expected changed files list to contain %q; got %v", wantFile, files)
	}

	got, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	parsed := make(map[string]any)
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("settings.json is not valid JSON after merge: %v\n%s", err, got)
	}

	for key, want := range existing {
		gotVal, present := parsed[key]
		if !present {
			t.Errorf("unrelated key %q was dropped during merge; got: %v", key, parsed)
			continue
		}
		gotJSON, _ := json.Marshal(gotVal)
		wantJSON, _ := json.Marshal(want)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("unrelated key %q mutated during merge\n want: %s\n got:  %s", key, wantJSON, gotJSON)
		}
	}

	trusted, ok := parsed["trustedWorkspaces"].([]any)
	if !ok || len(trusted) != 1 {
		t.Fatalf("expected trustedWorkspaces=[<workspace>]; got: %v", parsed["trustedWorkspaces"])
	}
	if trusted[0].(string) != filepath.Clean(workspace) {
		t.Fatalf("expected trustedWorkspaces[0]=%q; got %q", filepath.Clean(workspace), trusted[0])
	}
}

// TestTrustWorkspaceInAntigravitySettings_PreservesUnrelatedKeysWithExistingArray
// verifies that when the user already has a valid trustedWorkspaces array and
// unrelated keys, adding a new workspace preserves every other key AND every
// existing trusted workspace.
func TestTrustWorkspaceInAntigravitySettings_PreservesUnrelatedKeysWithExistingArray(t *testing.T) {
	home := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentAntigravity)
	if err != nil {
		t.Fatalf("NewAdapter(antigravity) error = %v", err)
	}

	settingsDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(settingsDir, "settings.json")

	existingA := filepath.Join(home, "already-trusted-A")
	existingB := filepath.Join(home, "already-trusted-B")
	body := map[string]any{
		"editor.fontSize":   16,
		"window.theme":      "light",
		"trustedWorkspaces": []any{existingA, existingB},
	}
	raw, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	newWorkspace := filepath.Join(home, "newly-trusted")
	changed, files, err := trustWorkspaceInAntigravitySettings(home, newWorkspace, adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when adding new workspace")
	}
	if len(files) != 1 || files[0] != settingsPath {
		t.Fatalf("expected files to contain %q; got %v", settingsPath, files)
	}

	got, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	parsed := make(map[string]any)
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("settings.json is not valid JSON after merge: %v\n%s", err, got)
	}

	if got := parsed["editor.fontSize"]; got != float64(16) {
		t.Errorf("unrelated key editor.fontSize mutated; want 16, got %v", got)
	}
	if got := parsed["window.theme"]; got != "light" {
		t.Errorf("unrelated key window.theme mutated; want %q, got %v", "light", got)
	}

	trusted, ok := parsed["trustedWorkspaces"].([]any)
	if !ok {
		t.Fatalf("trustedWorkspaces missing or wrong type: %v", parsed["trustedWorkspaces"])
	}
	want := map[string]bool{
		filepath.Clean(existingA):    false,
		filepath.Clean(existingB):    false,
		filepath.Clean(newWorkspace): false,
	}
	for _, v := range trusted {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("trustedWorkspaces contains non-string entry: %v", v)
		}
		if _, expected := want[s]; !expected {
			t.Errorf("unexpected entry in trustedWorkspaces: %q", s)
		}
		want[s] = true
	}
	for ws, present := range want {
		if !present {
			t.Errorf("trustedWorkspaces missing %q", ws)
		}
	}
	if len(trusted) != 3 {
		t.Fatalf("expected 3 trusted workspaces (A+B+new); got %d: %v", len(trusted), trusted)
	}
}

func TestAntigravitySddAgentsDynamicRolesAndScopes(t *testing.T) {
	expectedScopes := map[string]string{
		"review-risk":        "read-only; emit ledger rows only",
		"review-resilience":  "read-only; emit ledger rows only",
		"review-readability": "read-only; emit ledger rows only",
		"review-reliability": "read-only; emit ledger rows only",
		"review-refuter":     "read-only; emit verdicts only",
		"jd-judge-a":         "read-only; emit ledger rows only",
		"jd-judge-b":         "read-only; emit ledger rows only",
		"jd-fix-agent":       "edit only confirmed ledger findings; do not discover new findings",
	}

	for role, wantScope := range expectedScopes {
		found := false
		for _, entry := range antigravitySddAgentsRoleScopes {
			if entry.Role == role {
				found = true
				if entry.Scope != wantScope {
					t.Errorf("role %q has unexpected scope: want %q, got %q", role, wantScope, entry.Scope)
				}
				break
			}
		}
		if !found {
			t.Errorf("role %q is missing from the canonical role-scope validation table", role)
		}
	}
}

func TestAntigravitySddAgentsPreToolUseHookStructure(t *testing.T) {
	data := antigravitySddAgentsHooksJSON()
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("antigravitySddAgentsHooksJSON() is not valid JSON: %v", err)
	}

	block, ok := parsed["gentle-ai-sdd-agents-hardening"].(map[string]any)
	if !ok {
		t.Fatalf("missing gentle-ai-sdd-agents-hardening block: %v", parsed)
	}

	preToolUse, ok := block["PreToolUse"].([]any)
	if !ok || len(preToolUse) == 0 {
		t.Fatalf("missing or empty PreToolUse: %v", block)
	}

	entry, ok := preToolUse[0].(map[string]any)
	if !ok {
		t.Fatalf("PreToolUse[0] is not an object: %T", preToolUse[0])
	}

	matcher, ok := entry["matcher"].(string)
	if !ok || matcher != "run_command" {
		t.Fatalf("PreToolUse[0].matcher = %q, want %q", matcher, "run_command")
	}

	hooksList, ok := entry["hooks"].([]any)
	if !ok || len(hooksList) == 0 {
		t.Fatalf("PreToolUse[0].hooks is missing or empty: %v", entry)
	}

	hookObj, ok := hooksList[0].(map[string]any)
	if !ok {
		t.Fatalf("hooks[0] is not an object: %T", hooksList[0])
	}

	hookType, ok := hookObj["type"].(string)
	if !ok || hookType != "command" {
		t.Errorf("hook type = %q, want %q", hookType, "command")
	}

	cmd, ok := hookObj["command"].(string)
	if !ok || cmd == "" {
		t.Fatalf("hook command is missing or empty")
	}

	for _, phrase := range []string{
		"node -e",
		"Destructive command blocked by Gentle AI SDD guard",
		`"decision":"deny"`,
		`"decision":"allow"`,
		"--force",
		"-f",
		"--hard",
		"rm",
		"printf",
	} {
		if !strings.Contains(cmd, phrase) {
			t.Errorf("PreToolUse command missing expected phrase %q:\n%s", phrase, cmd)
		}
	}
}

func TestAntigravityActiveConfigDirs_Discovery(t *testing.T) {
	home := t.TempDir()

	// 1. None exist -> falls back to default
	dirs := antigravityActiveConfigDirs(home)
	if len(dirs) != 1 {
		t.Fatalf("expected 1 fallback dir when none exist, got %d: %v", len(dirs), dirs)
	}

	// 2. Create all 4 directories
	candidates := []string{
		filepath.Join(home, ".gemini", "config"),
		filepath.Join(home, ".gemini", "antigravity-cli"),
		filepath.Join(home, ".gemini", "antigravity-desktop"),
		filepath.Join(home, ".gemini", "antigravity"),
	}
	for _, c := range candidates {
		if err := os.MkdirAll(c, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	discovered := antigravityActiveConfigDirs(home)
	if len(discovered) != 4 {
		t.Fatalf("expected all 4 config dirs discovered, got %d: %v", len(discovered), discovered)
	}
	for _, c := range candidates {
		if !slices.Contains(discovered, c) {
			t.Errorf("discovered missing %q; got %v", c, discovered)
		}
	}
}

func TestAntigravitySddAgentsPreToolUseExecution(t *testing.T) {
	hooksData := antigravitySddAgentsHooksJSON()
	var root map[string]any
	if err := json.Unmarshal(hooksData, &root); err != nil {
		t.Fatalf("unmarshal hooks json: %v", err)
	}
	block, ok := root["gentle-ai-sdd-agents-hardening"].(map[string]any)
	if !ok {
		t.Fatalf("missing gentle-ai-sdd-agents-hardening: %v", root)
	}
	preToolUse, ok := block["PreToolUse"].([]any)
	if !ok || len(preToolUse) == 0 {
		t.Fatalf("missing PreToolUse: %v", block)
	}
	entry, ok := preToolUse[0].(map[string]any)
	if !ok {
		t.Fatalf("PreToolUse[0] is not a map: %v", preToolUse[0])
	}
	hooksList, ok := entry["hooks"].([]any)
	if !ok || len(hooksList) == 0 {
		t.Fatalf("missing hooks in PreToolUse[0]: %v", entry)
	}
	hookObj, ok := hooksList[0].(map[string]any)
	if !ok {
		t.Fatalf("hooks[0] is not a map: %v", hooksList[0])
	}
	fullCmd, ok := hookObj["command"].(string)
	if !ok || fullCmd == "" {
		t.Fatalf("hook command is empty: %v", hookObj)
	}

	parts := strings.SplitN(fullCmd, " || ", 2)
	if len(parts) != 2 {
		t.Fatalf("expected command to contain ' || ' separating node and shell fallback; got: %q", fullCmd)
	}
	nodeScript := parts[0]
	shellFallback := parts[1]

	cases := []struct {
		name     string
		command  string
		wantDeny bool
	}{
		// Positive destructive commands
		{"rm -rf", "rm -rf dir", true},
		{"rm -rfv", "rm -rfv dir", true},
		{"rm -fr", "rm -fr dir", true},
		{"rm -r -f", "rm -r -f dir", true},
		{"git reset --hard", "git reset --hard HEAD~1", true},
		{"git push -f", "git push -f origin main", true},
		{"git push --force", "git push --force origin main", true},

		// Negative non-destructive commands
		{"git status", "git status", false},
		{"rm file.txt", "rm file.txt", false},
		{"npm test", "npm test", false},
		{"git push origin main", "git push origin main", false},
	}

	makePayloads := func(cmd string) []struct {
		keyName string
		jsonStr string
	}{
		return []struct {
			keyName string
			jsonStr string
		}{
			{"CommandLine", fmt.Sprintf(`{"toolCall":{"args":{"CommandLine":%q}}}`, cmd)},
			{"commandLine", fmt.Sprintf(`{"toolCall":{"args":{"commandLine":%q}}}`, cmd)},
			{"command", fmt.Sprintf(`{"args":{"command":%q}}`, cmd)},
			{"cmd", fmt.Sprintf(`{"args":{"cmd":%q}}`, cmd)},
			{"direct_cmd", fmt.Sprintf(`{"cmd":%q}`, cmd)},
			{"direct_CommandLine", fmt.Sprintf(`{"CommandLine":%q}`, cmd)},
		}
	}

	hasNode := true
	if _, err := exec.LookPath("node"); err != nil {
		hasNode = false
		t.Log("node binary not found in PATH; skipping live node execution")
	}

	if hasNode {
		t.Run("node_script", func(t *testing.T) {
			for _, tc := range cases {
				for _, p := range makePayloads(tc.command) {
					testName := fmt.Sprintf("%s/%s", tc.name, p.keyName)
					t.Run(testName, func(t *testing.T) {
						cmd := exec.Command("sh", "-c", nodeScript)
						cmd.Stdin = strings.NewReader(p.jsonStr)
						out, err := cmd.CombinedOutput()
						if err != nil {
							t.Fatalf("node script failed with err %v, output: %s", err, string(out))
						}
						outStr := strings.TrimSpace(string(out))
						var resp map[string]any
						if err := json.Unmarshal([]byte(outStr), &resp); err != nil {
							t.Fatalf("failed to parse output %q as JSON: %v", outStr, err)
						}
						decision, _ := resp["decision"].(string)
						if tc.wantDeny {
							if decision != "deny" {
								t.Errorf("expected decision 'deny' for %q, got %q (output: %s)", tc.command, decision, outStr)
							}
						} else {
							if decision != "allow" {
								t.Errorf("expected decision 'allow' for %q, got %q (output: %s)", tc.command, decision, outStr)
							}
						}
					})
				}
			}
		})
	}

	t.Run("shell_fallback", func(t *testing.T) {
		for _, tc := range cases {
			for _, p := range makePayloads(tc.command) {
				testName := fmt.Sprintf("%s/%s", tc.name, p.keyName)
				t.Run(testName, func(t *testing.T) {
					cmd := exec.Command("sh", "-c", shellFallback)
					cmd.Stdin = strings.NewReader(p.jsonStr)
					out, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatalf("shell fallback failed with err %v, output: %s", err, string(out))
					}
					outStr := strings.TrimSpace(string(out))
					var resp map[string]any
					if err := json.Unmarshal([]byte(outStr), &resp); err != nil {
						t.Fatalf("failed to parse output %q as JSON: %v", outStr, err)
					}
					decision, _ := resp["decision"].(string)
					if tc.wantDeny {
						if decision != "deny" {
							t.Errorf("expected decision 'deny' for %q, got %q (output: %s)", tc.command, decision, outStr)
						}
					} else {
						if decision != "allow" {
							t.Errorf("expected decision 'allow' for %q, got %q (output: %s)", tc.command, decision, outStr)
						}
					}
				})
			}
		}
	})
}
