package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v2/internal/agents/antigravity"
	"github.com/gentleman-programming/gentle-ai/v2/internal/components/sdd"
	"github.com/gentleman-programming/gentle-ai/v2/internal/doctor"
)

// Antigravity dynamic subagents are runtime tools of the Mission Control
// agent runtime — they are NOT external commands on $PATH, so they cannot be
// shell-probed by gentle-ai doctor. The runtime check the orchestrator must
// perform is to call `define_subagent` and `invoke_subagent` and observe the
// tool availability response. gentle-ai doctor provides only the
// installable-surface check: it verifies that the gentle-ai-sdd-agents
// hardening plugin is in place so that, if the runtime exposes the tools,
// the contract is enforced; if the runtime does NOT expose the tools, the
// orchestrator must fail closed (per the SDD orchestrator's "fail closed"
// contract).
//
// We deliberately do NOT invent a shell command that pretends to probe the
// runtime — every shell probe we could write would either probe a file that
// does not exist (false negative) or probe a directory that exists but has
// no relationship to runtime tool availability (false positive). The
// installable-surface check is the strongest signal we can produce from
// outside the agent runtime.

// checkAntigravityDynamicSubagentRuntime returns a slice of doctor checks
// related to Antigravity dynamic subagents. It returns an empty slice when
// Antigravity is not installed, so non-Antigravity users see no noise.
//
// Two checks are produced:
//
//  1. "antigravity:installed" — PASS/WARN depending on whether the variant
//     directory is present. The check name mirrors the installable surface.
//  2. "antigravity:dynamic-subagent-hardening" — PASS/WARN/FAIL depending
//     on whether the gentle-ai-sdd-agents plugin is installed, with a
//     specific note that runtime availability of `define_subagent` /
//     `invoke_subagent` is NOT shell-probable. Users are pointed to the
//     in-runtime fail-closed contract as the source of truth.
func antigravityActiveConfigDir(homeDir string) string {
	return antigravity.NewAdapter().GlobalConfigDir(homeDir)
}

func checkAntigravityDynamicSubagentRuntime(homeDir string) []CheckResult {
	if !isAntigravityInstalled(homeDir) {
		return nil
	}

	results := make([]CheckResult, 0, 2)

	// 1. Installable-surface check: is the Antigravity CLI/Desktop surface
	// detected? The doctor is intentionally permissive here — even a bare
	// ~/.gemini/antigravity-cli without settings.json is enough to count as
	// installed, because the runtime may still be functional.
	results = append(results, CheckResult{
		Name:   "antigravity:installed",
		Status: CheckStatusPass,
		Detail: antigravityInstallableSurface(homeDir),
	})

	// 2. Hardening contract check: is the gentle-ai-sdd-agents plugin
	// present? We do not try to runtime-probe `define_subagent` or
	// `invoke_subagent` — they are not external commands and any shell probe
	// we wrote would be misleading. The strongest signal we can produce
	// outside the agent runtime is "the installable contract is in place".
	configDir := antigravityActiveConfigDir(homeDir)
	hardeningOK := sdd.HasAntigravitySddAgentsHardeningContract(homeDir)
	detail := "Antigravity dynamic subagent hardening plugin is installed at " +
		sdd.AntigravitySddAgentsPluginDir(homeDir)
	var checkRemedy *doctor.Remedy
	if !hardeningOK {
		detail = "Antigravity dynamic subagent hardening plugin is NOT installed; dynamic subagents will not be bound to the SDD/Review/JD tool-hardening contract."
		remedyMsg := fmt.Sprintf("Run `gentle-ai install --agent antigravity` (or `gentle-ai sync`) to install the gentle-ai-sdd-agents plugin under %s/plugins/.", configDir)
		checkRemedy = doctor.NewRemedy(doctor.RemedySync, remedyMsg)
	}
	status := CheckStatusPass
	if !hardeningOK {
		status = CheckStatusWarn
	}
	// Always append the runtime-probe limitation note. This documents the
	// fact that gentle-ai cannot shell-probe `define_subagent` /
	// `invoke_subagent` availability; the in-runtime fail-closed contract
	// is the source of truth.
	detail += " Note: define_subagent/invoke_subagent are Mission Control runtime tools, not external commands, so gentle-ai doctor cannot shell-probe their availability. The Antigravity SDD orchestrator must call them and fail closed if either tool is unavailable."
	results = append(results, CheckResult{
		Name:   "antigravity:dynamic-subagent-hardening",
		Status: status,
		Detail: detail,
		Remedy: checkRemedy,
	})

	return results
}

// isAntigravityInstalled reports whether any Antigravity variant directory
// is present. Used to gate the doctor check so non-Antigravity users see no
// noise in the report.
func isAntigravityInstalled(homeDir string) bool {
	for _, dir := range []string{
		filepath.Join(homeDir, ".gemini", "antigravity-cli"),
		filepath.Join(homeDir, ".gemini", "antigravity-desktop"),
		filepath.Join(homeDir, ".gemini", "antigravity"),
	} {
		info, err := os.Stat(dir)
		if err != nil {
			continue
		}
		if info.IsDir() {
			return true
		}
	}
	return false
}

// antigravityInstallableSurface returns a human-readable summary of which
// Antigravity variant directories are present. Used for the installable
// surface doctor check.
func antigravityInstallableSurface(homeDir string) string {
	present := make([]string, 0, 3)
	for _, variant := range []string{"antigravity-cli", "antigravity-desktop", "antigravity"} {
		path := filepath.Join(homeDir, ".gemini", variant)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		present = append(present, path)
	}
	if len(present) == 0 {
		return "no Antigravity variant directory present (this check should not have run)"
	}
	return "Antigravity installable surface detected at: " + strings.Join(present, ", ")
}

// AntigravityDynamicSubagentProbe is the read-only diagnostic surface for
// callers (CLI subcommands, tests) that want to surface the installable
// hardening contract status. It returns:
//
//   - installed: true when the gentle-ai-sdd-agents plugin is present and
//     has a valid hooks.json with the gentle-ai-sdd-agents-hardening block.
//   - pluginDir: the resolved plugin directory on disk.
//   - runtimeProbable: always false. This is intentional and reflects the
//     architectural limitation that `define_subagent` / `invoke_subagent`
//     are runtime tools and cannot be shell-probed. The orchestrator's
//     fail-closed contract is the authoritative source.
//
// The shape is stable and JSON-serializable; CLI subcommands and the
// in-runtime agent should treat the runtimeProbable=false signal as
// expected (do not "fix" it by inventing shell probes).
type AntigravityDynamicSubagentProbe struct {
	Installed       bool   `json:"installed"`
	PluginDir       string `json:"pluginDir"`
	RuntimeProbable bool   `json:"runtimeProbable"`
	RuntimeNote     string `json:"runtimeNote"`
}

// ProbeAntigravityDynamicSubagentRuntime returns the diagnostic surface for
// the Antigravity dynamic subagent runtime. The function never errors —
// missing files are a normal state, not an error condition. Use this in CLI
// subcommands, status output, or tests.
//
// runtimeProbable is hard-coded to false because the runtime tools
// (`define_subagent` / `invoke_subagent`) are not external commands and
// cannot be inspected from a shell. This is documented in the SDD
// orchestrator's "fail closed" contract as the explicit limitation that
// the orchestrator must check at runtime via the agent tool surface, not
// via a shell probe.
func ProbeAntigravityDynamicSubagentRuntime(homeDir string) AntigravityDynamicSubagentProbe {
	dir := sdd.AntigravitySddAgentsPluginDir(homeDir)
	return AntigravityDynamicSubagentProbe{
		Installed:       sdd.HasAntigravitySddAgentsHardeningContract(homeDir),
		PluginDir:       dir,
		RuntimeProbable: false,
		RuntimeNote: fmt.Sprintf(
			"%s and %s are Antigravity Mission Control runtime tools and are not external commands on $PATH. "+
				"The Antigravity SDD orchestrator must call them and fail closed if either tool is unavailable. "+
				"This probe only verifies the installable hardening contract; it does not and cannot verify runtime availability.",
			"define_subagent", "invoke_subagent",
		),
	}
}
