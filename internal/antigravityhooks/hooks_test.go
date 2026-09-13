package antigravityhooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validHooks is the named-hook shape agy documents, used as the control for the
// accept tests.
const validHooks = `{
  "sovereign-guardrails": {
    "PreInvocation": [
      {"type": "command", "command": "./scripts/hooks/report.sh", "timeout": 5}
    ],
    "PreToolUse": [
      {"matcher": "run_command", "hooks": [{"type": "command", "command": "./scripts/hooks/guard.sh"}]}
    ],
    "Stop": [
      {"type": "command", "command": "./scripts/hooks/report.sh"}
    ]
  }
}`

func findingsOfKind(findings []Finding, kind string) []Finding {
	var out []Finding
	for _, f := range findings {
		if f.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}

func TestValidate_AcceptsNamedHookShape(t *testing.T) {
	if got := Validate("hooks.json", []byte(validHooks), nil); len(got) != 0 {
		t.Fatalf("expected no findings for the documented shape, got %v", got)
	}
}

// The historical incident: the top level was a map of EVENT names to arrays
// instead of a map of HOOK names to hook specs. agy dropped the whole file and
// accumulated 1067 rejections before anyone noticed.
func TestValidate_RejectsTopLevelEventMap(t *testing.T) {
	raw := `{
  "PreInvocation": [{"type": "command", "command": "x"}],
  "PreToolUse": [{"matcher": "run_command", "hooks": [{"command": "y"}]}]
}`
	got := Validate("hooks.json", []byte(raw), nil)

	schema := findingsOfKind(got, KindSchema)
	if len(schema) == 0 {
		t.Fatalf("expected schema findings, got %v", got)
	}
	if !strings.Contains(schema[0].Detail, "must be an object, got an array") {
		t.Errorf("finding should name the array-vs-object mismatch, got %q", schema[0].Detail)
	}
}

func TestValidate_RejectsArrayTopLevel(t *testing.T) {
	got := Validate("hooks.json", []byte(`[{"event": "PreToolUse"}]`), nil)

	if len(got) == 0 {
		t.Fatal("expected a finding for an array top level")
	}
	if got[0].Kind != KindSchema || !strings.Contains(got[0].Detail, "got an array") {
		t.Errorf("unexpected finding: %+v", got[0])
	}
}

func TestValidate_RejectsEventThatIsNotAnArray(t *testing.T) {
	raw := `{"probe": {"PreInvocation": {"type": "command"}}}`
	got := Validate("hooks.json", []byte(raw), nil)

	if len(got) != 1 || !strings.Contains(got[0].Detail, `"PreInvocation" must be an array`) {
		t.Fatalf("expected an array-shape finding, got %v", got)
	}
}

func TestValidate_RejectsUngroupedPreToolUse(t *testing.T) {
	// PreToolUse requires the {"matcher": ..., "hooks": [...]} wrapper; a flat
	// handler list is valid for PreInvocation but not here.
	raw := `{"probe": {"PreToolUse": [{"type": "command", "command": "x"}]}}`
	got := Validate("hooks.json", []byte(raw), nil)

	if len(got) != 1 || !strings.Contains(got[0].Detail, "must be grouped") {
		t.Fatalf("expected a grouping finding, got %v", got)
	}
}

func TestValidate_ReportsDisabledExpectedHook(t *testing.T) {
	raw := `{"gentle-ai-sdd-agents-hardening": {"enabled": false, "PreInvocation": [{"command": "x"}]}}`
	expected := []string{"gentle-ai-sdd-agents-hardening"}

	got := Validate("hooks.json", []byte(raw), expected)

	disabled := findingsOfKind(got, KindDisabled)
	if len(disabled) != 1 {
		t.Fatalf("expected exactly one disabled finding, got %v", got)
	}
	if disabled[0].Blocking() {
		t.Error("a disabled hook must not be reported as blocking: it may be intentional")
	}
}

func TestValidate_DoesNotReportEnabledOrAbsentExpectedHooks(t *testing.T) {
	raw := `{"gentle-ai-sdd-agents-hardening": {"enabled": true, "PreInvocation": [{"command": "x"}]}}`

	got := Validate("hooks.json", []byte(raw), []string{"gentle-ai-sdd-agents-hardening", "not-in-this-file"})
	if len(got) != 0 {
		t.Fatalf("expected no findings, got %v", got)
	}
}

func TestValidate_IgnoresUnknownEventKeys(t *testing.T) {
	// A future agy event must not turn into a hard failure.
	raw := `{"probe": {"FutureEvent": [{"command": "x"}]}}`
	if got := Validate("hooks.json", []byte(raw), nil); len(got) != 0 {
		t.Fatalf("unknown event keys must be tolerated, got %v", got)
	}
}

func TestValidateFile_MissingFileIsNotAFinding(t *testing.T) {
	got := ValidateFile(filepath.Join(t.TempDir(), "absent.json"), nil)
	if len(got) != 0 {
		t.Fatalf("a missing file is a normal state, got %v", got)
	}
}

// writeScript creates an executable script with the given first line.
func writeScript(t *testing.T, dir, name, firstLine string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(firstLine+"\necho ok\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestValidateScript_AcceptsResolvableInterpreter(t *testing.T) {
	dir := t.TempDir()
	interpreter := writeScript(t, dir, "interp", "#!/bin/sh")

	script := writeScript(t, dir, "ok.sh", "#!"+interpreter)
	if got := ValidateScript(script); len(got) != 0 {
		t.Fatalf("expected no findings for a resolvable interpreter, got %v", got)
	}
}

func TestValidateScript_RejectsMissingInterpreter(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "broken.sh", "#!/nowhere/at/all/bash")

	got := ValidateScript(script)
	if len(got) != 1 || got[0].Kind != KindShebang {
		t.Fatalf("expected a shebang finding, got %v", got)
	}
	if !got[0].Blocking() {
		t.Error("a missing interpreter means the hook cannot run: it must be blocking")
	}
}

// The termux-exec hint is the difference between an operator fixing this in a
// minute and losing an afternoon.
func TestValidateScript_HintsTermuxExecForEnvInterpreter(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "env.sh", "#!/nowhere/env bash")

	got := ValidateScript(script)
	if len(got) != 1 {
		t.Fatalf("expected one finding, got %v", got)
	}
	if !strings.Contains(got[0].Detail, "termux-exec") {
		t.Errorf("an env-based shebang should mention termux-exec, got %q", got[0].Detail)
	}
}

func TestValidateScript_RejectsNonExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noexec.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho ok\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := ValidateScript(path)
	if len(got) != 1 || !strings.Contains(got[0].Detail, "not executable") {
		t.Fatalf("expected a not-executable finding, got %v", got)
	}
}

func TestValidateScript_NoShebangIsAccepted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.sh")
	if err := os.WriteFile(path, []byte("echo ok\n"), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := ValidateScript(path); len(got) != 0 {
		t.Fatalf("a script without a shebang falls back to the shell, got %v", got)
	}
}

func TestScanConfigDir_ValidatesEverySurface(t *testing.T) {
	dir := t.TempDir()

	// Root hooks.json: malformed (events at the top level).
	if err := os.WriteFile(filepath.Join(dir, "hooks.json"), []byte(`{"PreInvocation": [{"command": "x"}]}`), 0o644); err != nil {
		t.Fatalf("write root hooks: %v", err)
	}

	// One plugin: valid shape, but the expected hook is disabled.
	pluginDir := filepath.Join(dir, "plugins", "gentle-ai-sdd-agents")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin: %v", err)
	}
	disabled := `{"gentle-ai-sdd-agents-hardening": {"enabled": false, "PreInvocation": [{"command": "x"}]}}`
	if err := os.WriteFile(filepath.Join(pluginDir, "hooks.json"), []byte(disabled), 0o644); err != nil {
		t.Fatalf("write plugin hooks: %v", err)
	}

	// One broken hook script.
	scriptsDir := filepath.Join(dir, "scripts", "hooks")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	writeScript(t, scriptsDir, "broken.sh", "#!/nowhere/env bash")

	got := ScanConfigDir(dir, []string{"gentle-ai-sdd-agents-hardening"})

	if len(findingsOfKind(got, KindSchema)) == 0 {
		t.Errorf("expected a schema finding for the root file, got %v", got)
	}
	if len(findingsOfKind(got, KindDisabled)) == 0 {
		t.Errorf("expected a disabled finding for the plugin hook, got %v", got)
	}
	if len(findingsOfKind(got, KindShebang)) == 0 {
		t.Errorf("expected a shebang finding for the broken script, got %v", got)
	}
}
