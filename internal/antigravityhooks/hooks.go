// Package antigravityhooks validates the hooks.json surfaces the Antigravity CLI
// (agy) runtime consumes.
//
// agy rejects a hooks.json file WHOLE when its JSON does not match the schema it
// expects, and it does so quietly: the rejection is only visible in agy's own
// log, so an invalid file leaves every hook in it silently inert.
//
// Two production incidents motivated this package:
//
//   - A hooks.json whose top level was a map of EVENT names to arrays, instead
//     of the required map of HOOK names to hook specs, was dropped on every
//     load. It accumulated 1067 rejections before anyone noticed.
//   - Hook commands pointing at scripts whose shebang resolved through
//     /usr/bin/env could not be executed at all. The hook executor does not run
//     with Termux's termux-exec LD_PRELOAD shim, which is the only thing that
//     rewrites /usr/bin/env on Android, so the script died before starting and
//     surfaced as "sh: 1: <script>: not found".
//
// Both classes are structurally detectable, which is the point of this package.
package antigravityhooks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Finding kinds. Callers map these to severity: schema and shebang findings mean
// the hook cannot work at all, while a disabled finding may be intentional.
const (
	KindSchema   = "schema"
	KindShebang  = "shebang"
	KindDisabled = "disabled"
)

// KnownEvents are the lifecycle events agy documents for a named hook. Unknown
// keys are ignored rather than rejected so a future agy event does not turn into
// a hard failure here.
var KnownEvents = []string{
	"PreToolUse", "PostToolUse", "PreInvocation", "PostInvocation", "Stop",
}

// groupedEvents are the events that must wrap their handlers in a
// {"matcher": ..., "hooks": [...]} group. The remaining events take a flat list.
var groupedEvents = map[string]bool{
	"PreToolUse":  true,
	"PostToolUse": true,
}

// Finding is one validation problem.
type Finding struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// Blocking reports whether the finding means the affected hook cannot run.
func (f Finding) Blocking() bool {
	return f.Kind == KindSchema || f.Kind == KindShebang
}

// String renders the finding for human-facing detail lines.
func (f Finding) String() string {
	return fmt.Sprintf("%s: %s", filepath.Base(f.Path), f.Detail)
}

// Validate checks one hooks.json payload against the schema agy requires.
//
// expected names hook names that, when present in the payload, must not be
// explicitly disabled. A hook name that is absent is not reported here: whether
// it should exist at all is a different question, answered by the installer's
// own contract checks.
func Validate(path string, raw []byte, expected []string) []Finding {
	var findings []Finding
	add := func(kind, format string, args ...any) {
		findings = append(findings, Finding{
			Path:   path,
			Kind:   kind,
			Detail: fmt.Sprintf(format, args...),
		})
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		// The historical bug landed here: an array (or any non-object) at the
		// top level. agy's message for it is
		// "cannot unmarshal array into Go struct field .<event> of type jsonhook.JSONHookSpec".
		add(KindSchema, "top level must be an object of named hooks, got %s: %v", jsonKind(raw), err)
		return findings
	}
	if len(root) == 0 {
		add(KindSchema, "no named hooks present; agy loads nothing from this file")
		return findings
	}

	names := make([]string, 0, len(root))
	for name := range root {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		specRaw := root[name]

		var spec map[string]json.RawMessage
		if err := json.Unmarshal(specRaw, &spec); err != nil {
			add(KindSchema, "hook %q must be an object, got %s", name, jsonKind(specRaw))
			continue
		}

		if enabledRaw, ok := spec["enabled"]; ok {
			var enabled bool
			if err := json.Unmarshal(enabledRaw, &enabled); err != nil {
				add(KindSchema, "hook %q: %q must be a boolean", name, "enabled")
			}
		}

		for _, event := range KnownEvents {
			eventRaw, ok := spec[event]
			if !ok {
				continue
			}
			var handlers []json.RawMessage
			if err := json.Unmarshal(eventRaw, &handlers); err != nil {
				add(KindSchema, "hook %q: %q must be an array of handlers, got %s", name, event, jsonKind(eventRaw))
				continue
			}
			if !groupedEvents[event] {
				continue
			}
			for i, handler := range handlers {
				var group struct {
					Hooks []json.RawMessage `json:"hooks"`
				}
				if err := json.Unmarshal(handler, &group); err != nil || group.Hooks == nil {
					add(KindSchema,
						"hook %q: %s[%d] must be grouped as {\"matcher\": ..., \"hooks\": [...]}", name, event, i)
				}
			}
		}
	}

	for _, name := range expected {
		specRaw, ok := root[name]
		if !ok {
			continue
		}
		var spec struct {
			Enabled *bool `json:"enabled"`
		}
		if err := json.Unmarshal(specRaw, &spec); err != nil {
			continue
		}
		if spec.Enabled != nil && !*spec.Enabled {
			add(KindDisabled, "hook %q is explicitly disabled (\"enabled\": false): none of its handlers run", name)
		}
	}

	return findings
}

// ValidateFile validates a hooks.json file. A missing file is a normal state,
// not a finding.
func ValidateFile(path string, expected []string) []Finding {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return Validate(path, raw, expected)
}

// ValidateScript verifies that a hook script can actually be executed by the agy
// hook executor. It checks two things the executor cannot work around:
//
//   - the shebang interpreter must exist on this machine, so /usr/bin/env and
//     /bin/bash are rejected on Android (the executor has no termux-exec shim);
//   - the file must carry an executable bit.
func ValidateScript(path string) []Finding {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil
	}
	if info.Mode().Perm()&0o111 == 0 {
		return []Finding{{
			Path:   path,
			Kind:   KindShebang,
			Detail: "script is not executable; the hook executor cannot run it",
		}}
	}
	return shebangFindings(path)
}

func shebangFindings(path string) []Finding {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	line, _ := bufio.NewReader(f).ReadString('\n')
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#!") {
		// No shebang: the executor falls back to the shell, which is fine.
		return nil
	}
	fields := strings.Fields(strings.TrimPrefix(line, "#!"))
	if len(fields) == 0 {
		return nil
	}
	interpreter := fields[0]
	if _, err := os.Stat(interpreter); err == nil {
		return nil
	}

	detail := fmt.Sprintf("shebang interpreter %q does not exist on this machine; use its absolute path", interpreter)
	if filepath.Base(interpreter) == "env" {
		detail += " (agy runs hooks without Termux's termux-exec shim, which is the only thing that rewrites /usr/bin/env)"
	}
	return []Finding{{Path: path, Kind: KindShebang, Detail: detail}}
}

// ScanConfigDir validates every hooks surface agy reads under configDir: the
// root hooks.json, each plugin's hooks.json, and the hook scripts directory.
// expected names hooks that must not be explicitly disabled.
func ScanConfigDir(configDir string, expected []string) []Finding {
	findings := ValidateFile(filepath.Join(configDir, "hooks.json"), expected)

	plugins, _ := filepath.Glob(filepath.Join(configDir, "plugins", "*", "hooks.json"))
	sort.Strings(plugins)
	for _, plugin := range plugins {
		findings = append(findings, ValidateFile(plugin, expected)...)
	}

	scripts, _ := filepath.Glob(filepath.Join(configDir, "scripts", "hooks", "*"))
	sort.Strings(scripts)
	for _, script := range scripts {
		findings = append(findings, ValidateScript(script)...)
	}

	return findings
}

// jsonKind names the JSON shape of raw for error messages, so an operator reads
// "got an array" instead of a wall of bytes.
func jsonKind(raw []byte) string {
	trimmed := strings.TrimSpace(string(raw))
	switch {
	case trimmed == "":
		return "an empty value"
	case strings.HasPrefix(trimmed, "["):
		return "an array"
	case strings.HasPrefix(trimmed, "{"):
		return "an object"
	case strings.HasPrefix(trimmed, "\""):
		return "a string"
	case trimmed == "null":
		return "null"
	case trimmed == "true" || trimmed == "false":
		return "a boolean"
	default:
		return "a number"
	}
}
