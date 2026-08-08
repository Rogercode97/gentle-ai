package sdd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v2/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v2/internal/agents/antigravity"
	"github.com/gentleman-programming/gentle-ai/v2/internal/components/filemerge"
)

// Antigravity has no static sub-agent registry with `permission.task` blocks
// like OpenCode. Sub-agents are defined dynamically at runtime via
// `define_subagent` / `invoke_subagent`. That means we cannot install a static
// "deny all sub-agents except sdd-*/review-*/jd-*" policy the way OpenCode's
// `sdd-overlay-multi.json` does.
//
// The safest supported Antigravity equivalent is a thin plugin whose only
// job is to inject a deterministic Tool Hardening contract into the
// orchestrator's context. The contract is identical in spirit to the
// OpenCode `permission.task.__replace__` overlay: define_subagent calls that
// try to step outside the role's allowed tool scope must fail closed.
//
// Files written by this installer:
//
//	~/.gemini/antigravity-cli/plugins/gentle-ai-sdd-agents/plugin.json  // plugin manifest
//	~/.gemini/antigravity-cli/plugins/gentle-ai-sdd-agents/hooks.json   // PreInvocation inject
//
// The plugin lives under the antigravity-cli surface only, matching the
// existing gentle-ai-engram / gentle-ai-codegraph plugin layout. The
// antigravity-desktop variant does not consume plugin/hooks.json today;
// gentle-ai-install for the desktop surface will not write this plugin to
// keep behavior aligned with the supported runtime surface.

const antigravitySddAgentsPluginName = "gentle-ai-sdd-agents"

const antigravitySddAgentsPluginJSON = `{
  "name": "gentle-ai-sdd-agents",
  "description": "Injects the SDD/Review/Judgment-Day tool-hardening contract for Antigravity dynamic sub-agents. Mirrors the OpenCode permission.task overlay for the Antigravity runtime surface.",
  "version": "0.1.0"
}
`

// antigravitySddAgentsHardeningMessage is the ephemeral message injected via
// PreInvocation. It mirrors the role-by-role tool scope from
// internal/assets/antigravity/sdd-orchestrator.md (Antigravity Tool Hardening)
// and the OpenCode sdd-overlay-*.json permission.task.__replace__ policy.
//
// The text is intentionally human-readable so the Antigravity runtime can
// surface it as a system-level reminder. We do NOT invent Antigravity API
// fields that the runtime does not consume; this is the safest supported
// installable permission surface.
const antigravitySddAgentsHardeningMessage = "Gentle AI SDD/Review/JD hardening contract for Antigravity dynamic sub-agents. " +
	"This contract mirrors the OpenCode permission.task overlay; Antigravity has no static agent registry, " +
	"so the policy is enforced as a runtime instruction bound to define_subagent calls. " +
	"Allowed roles and their tool scopes: " +
	"sdd-explore = read/search/CodeGraph/Engram only, no source writes; " +
	"sdd-propose, sdd-spec, sdd-design, sdd-tasks = artifact reads/writes only, no source edits; " +
	"sdd-apply = source edits and targeted verification commands only, no commit/push/PR/publish/destructive git; " +
	"sdd-verify = read plus test/build commands, no source edits unless explicitly approved; " +
	"sdd-archive, sdd-onboard, sdd-init = read plus scoped writes; " +
	"review-* (including review-risk, review-readability, review-reliability, review-resilience, and review-refuter) and jd-judge-* (including jd-judge-a, jd-judge-b) = read-only, emit ledger rows or verdicts only; " +
	"jd-fix-agent = edit only confirmed ledger findings, do not discover new findings. " +
	"Strict TDD (Test-Driven Development) enforcement rules: When strict_tdd: true is active, " +
	"sdd-apply is prohibited from editing production files without first writing or modifying test files and running the test runner to observe test failure (Red phase). " +
	"sdd-verify must run tests to verify behavior and is prohibited from editing source code. " +
	"Any attempt to bypass the TDD Red-Green-Refactor sequence must fail closed. " +
	"Any define_subagent call that tries to widen its tool scope above the allowed scope for that role MUST fail closed and surface status: blocked with the missing capability. " +
	"Dynamic sub-agents MUST NOT use broad repository search (grep -R, find sweeps, full-tree reads) until CodeGraph has failed or returned insufficient results. " +
	"Web/internet search is denied by default for code implementation, review, and verification phases unless the task explicitly requires external research."

func antigravityActiveConfigDir(homeDir string) string {
	return antigravity.NewAdapter().GlobalConfigDir(homeDir)
}

func antigravitySddAgentsPluginDir(homeDir string) string {
	return filepath.Join(antigravityActiveConfigDir(homeDir), "plugins", antigravitySddAgentsPluginName)
}

func antigravitySddAgentsHooksJSON() []byte {
	cfg := map[string]any{
		"gentle-ai-sdd-agents-hardening": map[string]any{
			"PreInvocation": []any{
				map[string]any{
					"type": "command",
					"command": "printf '%s\\n' '" + mustJSONStringSDDAgents(map[string]any{
						"injectSteps": []any{
							map[string]any{"ephemeralMessage": antigravitySddAgentsHardeningMessage},
						},
					}) + "'",
				},
			},
		},
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return append(b, '\n')
}

func mustJSONStringSDDAgents(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// installAntigravitySddAgentsPlugin writes the gentle-ai-sdd-agents plugin
// (plugin.json + hooks.json) under ~/.gemini/antigravity-cli/plugins/. It
// returns (changed, files, err) so the SDD injector can fold the result into
// its InjectionResult.
//
// This is the Antigravity equivalent of the OpenCode sdd-overlay-*.json
// `permission.task.__replace__` block. We do NOT touch user-owned
// settings.json or mcp_config.json here — the policy lives entirely inside
// the plugin so a future uninstall deletes the plugin directory and
// removes the hardening contract atomically.
func installAntigravitySddAgentsPlugin(homeDir string) (bool, []string, error) {
	pluginDir := antigravitySddAgentsPluginDir(homeDir)
	files := make([]string, 0, 2)
	changed := false

	pluginPath := filepath.Join(pluginDir, "plugin.json")
	pluginWrite, err := filemerge.WriteFileAtomic(pluginPath, []byte(antigravitySddAgentsPluginJSON), 0o644)
	if err != nil {
		return false, nil, fmt.Errorf("write Antigravity SDD agents plugin manifest: %w", err)
	}
	changed = changed || pluginWrite.Changed
	files = append(files, pluginPath)

	hooksPath := filepath.Join(pluginDir, "hooks.json")
	hooksWrite, err := mergeJSONFile(hooksPath, antigravitySddAgentsHooksJSON())
	if err != nil {
		return false, nil, fmt.Errorf("write Antigravity SDD agents plugin hooks: %w", err)
	}
	changed = changed || hooksWrite.writeResult.Changed
	files = append(files, hooksPath)

	return changed, files, nil
}

// antigravitySddAgentsRoleScopes is the canonical role→tool-scope table used
// by the hardening contract. The OpenCode permission.task overlay is
// equivalent for the sdd-*, review-*, and jd-* sub-agent keys; Antigravity
// cannot enforce it statically, so the table is exposed here for tests and
// for any future static validation that needs to assert contract drift.
var antigravitySddAgentsRoleScopes = []struct {
	Role  string
	Scope string
}{
	{"sdd-explore", "read/search/CodeGraph/Engram only; no source writes"},
	{"sdd-spec", "artifact reads/writes only; no source edits"},
	{"sdd-design", "artifact reads/writes only; no source edits"},
	{"sdd-tasks", "artifact reads/writes only; no source edits"},
	{"sdd-apply", "source edits and targeted verification commands only; no commit/push/PR/publish/destructive git"},
	{"sdd-verify", "read plus test/build commands; no source edits unless explicitly approved"},
	{"sdd-archive", "read plus scoped writes"},
	{"sdd-onboard", "read plus scoped writes"},
	{"sdd-init", "read plus scoped writes"},
	{"sdd-propose", "artifact reads/writes only; no source edits"},
	{"review-risk", "read-only; emit ledger rows only"},
	{"review-resilience", "read-only; emit ledger rows only"},
	{"review-readability", "read-only; emit ledger rows only"},
	{"review-reliability", "read-only; emit ledger rows only"},
	{"review-refuter", "read-only; emit verdicts only"},
	{"jd-judge-a", "read-only; emit ledger rows only"},
	{"jd-judge-b", "read-only; emit ledger rows only"},
	{"jd-fix-agent", "edit only confirmed ledger findings; do not discover new findings"},
}

// antigravitySddAgentsRoleAllowed reports whether the given role appears in
// the canonical role-scope table. Used by tests to lock the fail-closed
// contract; any role not in the table must not be allowed.
func antigravitySddAgentsRoleAllowed(role string) bool {
	for _, entry := range antigravitySddAgentsRoleScopes {
		if entry.Role == role {
			return true
		}
	}
	return false
}

// sortedAntigravitySddAgentsRoleScopes returns the role table sorted by role
// name. Used by tests for stable comparison.
func sortedAntigravitySddAgentsRoleScopes() []struct {
	Role  string
	Scope string
} {
	out := make([]struct {
		Role  string
		Scope string
	}, len(antigravitySddAgentsRoleScopes))
	copy(out, antigravitySddAgentsRoleScopes)
	sort.Slice(out, func(i, j int) bool { return out[i].Role < out[j].Role })
	return out
}

// antigravitySddAgentsHardeningContractPhrases is the set of substrings the
// hardening message MUST contain. Used by tests to lock the message text
// against accidental edits that would weaken the contract.
var antigravitySddAgentsHardeningContractPhrases = []string{
	"sdd-explore",
	"sdd-propose",
	"sdd-apply",
	"sdd-verify",
	"review-*",
	"jd-judge-*",
	"jd-fix-agent",
	"fail closed",
	"CodeGraph",
	"Strict TDD",
	"Red phase",
	"Red-Green-Refactor",
	"review-risk",
	"review-readability",
	"review-reliability",
	"review-resilience",
	"jd-judge-a",
	"jd-judge-b",
}

// antigravitySddAgentsHardeningContractForbids is the set of substrings the
// hardening message MUST NOT contain. We forbid wording that would fake a
// real Antigravity API we cannot actually invoke (e.g. permission.task,
// __replace__) — the contract is an ephemeral instruction, not a static
// permission schema.
var antigravitySddAgentsHardeningContractForbids = []string{
	"__replace__",
}

// AntigravitySddAgentsHardeningMessage is the exported, read-only view of
// the hardening contract. Exposed for callers (CLI doctor/validate) that
// want to print or compare the message without depending on the unexported
// constant directly.
func AntigravitySddAgentsHardeningMessage() string {
	return antigravitySddAgentsHardeningMessage
}

// AntigravitySddAgentsPluginDir is the exported, read-only view of the
// plugin directory. Mirrors AntigravitySddAgentsHardeningMessage for tests
// and diagnostics.
func AntigravitySddAgentsPluginDir(homeDir string) string {
	return antigravitySddAgentsPluginDir(homeDir)
}

// HasAntigravitySddAgentsHardeningContract reports whether the gentle-ai-sdd-agents
// plugin is installed AND the hardening contract is present in its hooks.json.
// This is the read-only, conservative check used by diagnostic surfaces.
func HasAntigravitySddAgentsHardeningContract(homeDir string) bool {
	hooksPath := filepath.Join(antigravitySddAgentsPluginDir(homeDir), "hooks.json")
	data, err := readFileOrEmpty(hooksPath)
	if err != nil {
		return false
	}
	raw := strings.TrimSpace(data)
	if raw == "" {
		return false
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return false
	}
	block, ok := root["gentle-ai-sdd-agents-hardening"].(map[string]any)
	if !ok {
		return false
	}
	pre, ok := block["PreInvocation"].([]any)
	if !ok || len(pre) == 0 {
		return false
	}
	commands := strings.Builder{}
	for _, item := range pre {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		command, ok := entry["command"].(string)
		if !ok {
			continue
		}
		commands.WriteString(command)
		commands.WriteByte('\n')
	}
	contract := commands.String()
	for _, phrase := range antigravitySddAgentsHardeningContractPhrases {
		if !strings.Contains(contract, phrase) {
			return false
		}
	}
	return true
}

// trustWorkspaceInAntigravitySettings adds workspaceDir to the trustedWorkspaces list
// in the Antigravity settings.json if it is not already present and if workspaceDir is not empty.
//
// The function is fail-closed against silent user-config data loss: if the
// existing settings.json cannot be parsed as a JSON object, or if the existing
// trustedWorkspaces key is present but not an array of strings, the function
// returns an error and does NOT write. Unrelated top-level keys are preserved
// on the merge path because we unmarshal into a map and only mutate the
// trustedWorkspaces key.
func trustWorkspaceInAntigravitySettings(homeDir, workspaceDir string, adapter agents.Adapter) (bool, []string, error) {
	if strings.TrimSpace(workspaceDir) == "" {
		return false, nil, nil
	}

	settingsPath := adapter.SettingsPath(homeDir)
	if settingsPath == "" {
		return false, nil, nil
	}

	// Clean the workspace directory path to make it standard
	workspaceDir = filepath.Clean(workspaceDir)

	baseJSON, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			baseJSON = []byte("{}")
		} else {
			return false, nil, fmt.Errorf("read settings file %q: %w", settingsPath, err)
		}
	}

	if strings.TrimSpace(string(baseJSON)) == "" {
		return false, nil, fmt.Errorf("settings file %q is empty (want JSON object); refusing to write to avoid silent data loss", settingsPath)
	}

	var data map[string]any
	if len(baseJSON) > 0 {
		if err := json.Unmarshal(baseJSON, &data); err != nil {
			return false, nil, fmt.Errorf("parse settings file %q: %w", settingsPath, err)
		}
		// A successful unmarshal with non-empty input that produces a nil map
		// means the top-level value was JSON `null` (the only JSON value that
		// decodes into a nil Go map[string]interface{} without an error).
		// That is not a JSON object — refusing to write protects the user's
		// existing null-sentinel value from being silently overwritten.
		if data == nil {
			return false, nil, fmt.Errorf("settings file %q has top-level value %q (want JSON object); refusing to write to avoid silent data loss", settingsPath, string(baseJSON))
		}
	}
	if data == nil {
		data = make(map[string]interface{})
	}

	// Fail-closed: if trustedWorkspaces exists but is not an array of strings,
	// refuse to write. Silently overwriting it with a fresh array would destroy
	// the user's existing trust configuration.
	if tVal, ok := data["trustedWorkspaces"]; ok {
		trusted, ok := tVal.([]any)
		if !ok {
			return false, nil, fmt.Errorf("settings file %q has %q with unexpected type %T (want array of strings); refusing to write to avoid silent data loss", settingsPath, "trustedWorkspaces", tVal)
		}
		for i, v := range trusted {
			if _, ok := v.(string); !ok {
				return false, nil, fmt.Errorf("settings file %q has %q[%d] with unexpected type %T (want string); refusing to write to avoid silent data loss", settingsPath, "trustedWorkspaces", i, v)
			}
		}

		existing := make(map[string]bool)
		for _, val := range trusted {
			existing[filepath.Clean(val.(string))] = true
		}

		if existing[workspaceDir] {
			return false, nil, nil
		}

		trusted = append(trusted, workspaceDir)
		data["trustedWorkspaces"] = trusted

		newJSON, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return false, nil, fmt.Errorf("marshal settings file %q: %w", settingsPath, err)
		}

		writeResult, err := filemerge.WriteFileAtomic(settingsPath, newJSON, 0o644)
		if err != nil {
			return false, nil, fmt.Errorf("write settings file %q: %w", settingsPath, err)
		}

		return writeResult.Changed, []string{settingsPath}, nil
	}

	// No existing trustedWorkspaces — create it. Unrelated keys are preserved
	// because data is a map of the unmarshaled JSON.
	trusted := []any{workspaceDir}
	data["trustedWorkspaces"] = trusted

	newJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return false, nil, fmt.Errorf("marshal settings file %q: %w", settingsPath, err)
	}

	writeResult, err := filemerge.WriteFileAtomic(settingsPath, newJSON, 0o644)
	if err != nil {
		return false, nil, fmt.Errorf("write settings file %q: %w", settingsPath, err)
	}

	return writeResult.Changed, []string{settingsPath}, nil
}

var AntigravityCodeGraphToolWiringPathsFn = AntigravityCodeGraphToolWiringPaths
var HasAntigravityCodeGraphToolWiringFn = HasAntigravityCodeGraphToolWiring

func AntigravityCodeGraphToolWiringPaths(homeDir string, adapter agents.Adapter) []string {
	return []string{
		filepath.Join(homeDir, ".gemini", "config", "mcp_config.json"),
		filepath.Join(homeDir, ".gemini", "antigravity", "mcp_config.json"),
	}
}

func HasAntigravityCodeGraphToolWiring(homeDir string, adapter agents.Adapter) (string, bool) {
	pluginPath := filepath.Join(adapter.GlobalConfigDir(homeDir), "plugins", "gentle-ai-codegraph", "mcp_config.json")
	for _, path := range []string{
		filepath.Join(homeDir, ".gemini", "config", "mcp_config.json"),
		filepath.Join(homeDir, ".gemini", "antigravity", "mcp_config.json"),
		pluginPath,
	} {
		data, err := os.ReadFile(path)
		if err == nil && hasCanonicalCodeGraphServer(data) {
			return path, true
		}
	}
	path := filepath.Join(homeDir, ".gemini", "antigravity", "mcp_config.json")
	if _, err := os.Stat(filepath.Join(homeDir, ".gemini", "config", ".migrated")); err == nil {
		path = filepath.Join(homeDir, ".gemini", "config", "mcp_config.json")
	}
	data, err := os.ReadFile(path)
	return path, err == nil && hasCanonicalCodeGraphServer(data)
}

func hasCanonicalCodeGraphServer(data []byte) bool {
	var config struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if json.Unmarshal(data, &config) != nil {
		return false
	}
	var server struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	raw, ok := config.MCPServers["codegraph"]
	if !ok || json.Unmarshal(raw, &server) != nil {
		return false
	}
	if server.Command != "codegraph" {
		if !filepath.IsAbs(server.Command) {
			return false
		}
		command := filepath.Base(server.Command)
		if command != "codegraph" && !strings.EqualFold(command, "codegraph.exe") {
			return false
		}
	}
	return slices.Equal(server.Args, []string{"serve", "--mcp"})
}

func ensureAntigravitySkillRegistryHook(hooksPath string) (bool, error) {
	root := map[string]any{}
	if data, err := os.ReadFile(hooksPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &root); err != nil {
			return false, fmt.Errorf("parse Antigravity hooks %q: %w", hooksPath, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	const command = `gentle-ai skill-registry refresh --quiet --no-gitignore --cwd "$PWD" || true`

	pluginRaw, ok := root["gentle-ai-engram-tools"]
	if !ok {
		pluginRaw = map[string]any{}
	}
	pluginMap, _ := pluginRaw.(map[string]any)
	if pluginMap == nil {
		pluginMap = map[string]any{}
	}

	preInvRaw, ok := pluginMap["PreInvocation"]
	if !ok {
		preInvRaw = []any{}
	}
	preInvList, _ := preInvRaw.([]any)
	if preInvList == nil {
		preInvList = []any{}
	}

	exists := false
	for _, item := range preInvList {
		itemMap, ok := item.(map[string]any)
		if ok && itemMap["command"] == command {
			exists = true
			break
		}
	}

	if exists {
		return false, nil
	}

	newPreInv := append([]any{map[string]any{
		"type":    "command",
		"command": command,
	}}, preInvList...)

	pluginMap["PreInvocation"] = newPreInv
	root["gentle-ai-engram-tools"] = pluginMap

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(hooksPath), 0o755); err != nil {
		return false, err
	}
	wr, err := filemerge.WriteFileAtomic(hooksPath, out, 0o644)
	if err != nil {
		return false, err
	}
	return wr.Changed, nil
}
