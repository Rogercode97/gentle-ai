package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v2/internal/components/sdd"
)

func writeDoctorHardeningHook(t *testing.T, home string) {
	t.Helper()
	hooksPath := filepath.Join(sdd.AntigravitySddAgentsPluginDir(home), "hooks.json")
	if err := os.MkdirAll(filepath.Dir(hooksPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"gentle-ai-sdd-agents-hardening": map[string]any{
			"PreInvocation": []any{
				map[string]any{"type": "command", "command": sdd.AntigravitySddAgentsHardeningMessage()},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hooksPath, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- isAntigravityInstalled ---

func TestIsAntigravityInstalled_NoSurface(t *testing.T) {
	home := t.TempDir()
	if isAntigravityInstalled(home) {
		t.Fatal("isAntigravityInstalled() = true on empty home, want false")
	}
}

func TestIsAntigravityInstalled_AnyVariantPresent(t *testing.T) {
	cases := []string{"antigravity-cli", "antigravity-desktop", "antigravity"}
	for _, variant := range cases {
		t.Run(variant, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(home, ".gemini", variant)
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			if !isAntigravityInstalled(home) {
				t.Fatalf("isAntigravityInstalled() = false with %q present, want true", variant)
			}
		})
	}
}

func TestIsAntigravityInstalled_RejectsFile(t *testing.T) {
	home := t.TempDir()
	// A regular file at the antigravity-cli path must not count as
	// "installed" — discovery only counts directories.
	path := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if isAntigravityInstalled(home) {
		t.Fatal("isAntigravityInstalled() = true when path is a file, want false")
	}
}

// --- checkAntigravityDynamicSubagentRuntime ---

func TestCheckAntigravityDynamicSubagentRuntime_NotInstalled(t *testing.T) {
	home := t.TempDir()
	results := checkAntigravityDynamicSubagentRuntime(home)
	if len(results) != 0 {
		t.Fatalf("checkAntigravityDynamicSubagentRuntime() = %d results on empty home, want 0", len(results))
	}
}

func TestCheckAntigravityDynamicSubagentRuntime_AntigravityNoPlugin(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".gemini", "antigravity-cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	results := checkAntigravityDynamicSubagentRuntime(home)
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2 (installable surface + hardening contract)", len(results))
	}
	// Installable surface should PASS (the CLI variant dir exists).
	if results[0].Name != "antigravity:installed" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "antigravity:installed")
	}
	if results[0].Status != CheckStatusPass {
		t.Errorf("results[0].Status = %q, want %q", results[0].Status, CheckStatusPass)
	}
	// Hardening contract should WARN (plugin missing).
	if results[1].Name != "antigravity:dynamic-subagent-hardening" {
		t.Errorf("results[1].Name = %q, want %q", results[1].Name, "antigravity:dynamic-subagent-hardening")
	}
	if results[1].Status != CheckStatusWarn {
		t.Errorf("results[1].Status = %q, want %q", results[1].Status, CheckStatusWarn)
	}
	if results[1].Remedy == nil {
		t.Error("results[1].Remedy must be non-nil when the hardening contract is missing")
	}
	// The runtime-probe limitation MUST be documented in the detail so
	// users understand why this check does not shell-probe
	// define_subagent/invoke_subagent.
	for _, want := range []string{
		"define_subagent",
		"invoke_subagent",
		"runtime tools",
		"not external commands",
		"shell-probe",
		"fail closed",
	} {
		if !strings.Contains(results[1].Detail, want) {
			t.Errorf("results[1].Detail missing %q:\n%s", want, results[1].Detail)
		}
	}
}

func TestCheckAntigravityDynamicSubagentRuntime_AntigravityWithPlugin(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".gemini", "antigravity-cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeDoctorHardeningHook(t, home)
	results := checkAntigravityDynamicSubagentRuntime(home)
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if results[1].Status != CheckStatusPass {
		t.Errorf("results[1].Status = %q, want %q (hardening contract in place)", results[1].Status, CheckStatusPass)
	}
	// The runtime-probe limitation note must be present even on the PASS
	// path so the user is never surprised that the doctor cannot
	// guarantee runtime availability.
	if !strings.Contains(results[1].Detail, "define_subagent") || !strings.Contains(results[1].Detail, "fail closed") {
		t.Errorf("results[1].Detail missing runtime-probe limitation note:\n%s", results[1].Detail)
	}
}

// --- ProbeAntigravityDynamicSubagentRuntime ---

func TestProbeAntigravityDynamicSubagentRuntime_NotInstalled(t *testing.T) {
	home := t.TempDir()
	probe := ProbeAntigravityDynamicSubagentRuntime(home)
	if probe.Installed {
		t.Fatal("Installed = true on empty home, want false")
	}
	if probe.PluginDir == "" {
		t.Error("PluginDir must be set even when the plugin is missing")
	}
	// runtimeProbable must be hard-coded to false regardless of
	// installation state — `define_subagent` / `invoke_subagent` are
	// runtime tools and cannot be shell-probed. The orchestrator's
	// fail-closed contract is the source of truth.
	if probe.RuntimeProbable {
		t.Fatal("RuntimeProbable = true; this MUST be hard-coded false. Shell probes cannot inspect Antigravity runtime tools.")
	}
	if !strings.Contains(probe.RuntimeNote, "runtime tools") {
		t.Errorf("RuntimeNote must document runtime-tool limitation; got %q", probe.RuntimeNote)
	}
	if !strings.Contains(probe.RuntimeNote, "fail closed") {
		t.Errorf("RuntimeNote must reference the in-runtime fail-closed contract; got %q", probe.RuntimeNote)
	}
}

func TestProbeAntigravityDynamicSubagentRuntime_Installed(t *testing.T) {
	home := t.TempDir()
	writeDoctorHardeningHook(t, home)
	probe := ProbeAntigravityDynamicSubagentRuntime(home)
	if !probe.Installed {
		t.Fatal("Installed = false after install, want true")
	}
	if probe.RuntimeProbable {
		t.Fatal("RuntimeProbable must remain false; runtime tools are not shell-probable")
	}
}

func TestProbeAntigravityDynamicSubagentRuntime_JSONShape(t *testing.T) {
	// The probe shape is consumed by CLI subcommands and tests; the JSON
	// tags must remain stable. Lock the field names so a future refactor
	// does not silently rename them.
	home := t.TempDir()
	probe := ProbeAntigravityDynamicSubagentRuntime(home)
	data, err := json.Marshal(probe)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	for _, want := range []string{"installed", "pluginDir", "runtimeProbable", "runtimeNote"} {
		if _, ok := parsed[want]; !ok {
			t.Errorf("probe JSON missing field %q: %s", want, data)
		}
	}
	// runtimeProbable must be a JSON boolean false (not omitted, not a
	// string). The downstream consumers rely on a strict bool type.
	if v, ok := parsed["runtimeProbable"].(bool); !ok {
		t.Errorf("runtimeProbable is not a JSON bool: %T", parsed["runtimeProbable"])
	} else if v {
		t.Errorf("runtimeProbable = true in JSON; must be false")
	}
}
