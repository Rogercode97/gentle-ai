package system

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// withShebangLookPath installs a fake PATH resolver for one test.
func withShebangLookPath(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	prev := shebangLookPath
	shebangLookPath = fn
	t.Cleanup(func() { shebangLookPath = prev })
}

// writeScript creates an executable script with the given shebang line.
func writeScript(t *testing.T, dir, name, shebang string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(shebang+"\nconsole.log(1);\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func TestResolveShebangEnvNodeUsesPathInterpreter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shebang contract")
	}
	dir := t.TempDir()
	script := writeScript(t, dir, "pi", "#!/usr/bin/env node")
	withShebangLookPath(t, func(name string) (string, error) {
		if name == "pi" {
			return script, nil
		}
		if name == "node" {
			return "/data/data/com.termux/files/usr/bin/node", nil
		}
		return "", os.ErrNotExist
	})

	gotScript, interpreter, interpreterArgs, ok := resolveShebang("pi")
	if !ok {
		t.Fatal("expected upstream /usr/bin/env shebang to resolve on Termux")
	}
	if gotScript != script {
		t.Errorf("script = %q, want %q", gotScript, script)
	}
	if interpreter != "/data/data/com.termux/files/usr/bin/node" {
		t.Errorf("interpreter = %q, want the PATH-resolved node", interpreter)
	}
	if len(interpreterArgs) != 0 {
		t.Errorf("interpreterArgs = %v, want none", interpreterArgs)
	}
}

func TestResolveShebangKeepsEnvOperandAfterOption(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shebang contract")
	}
	dir := t.TempDir()
	script := writeScript(t, dir, "cli", "#!/usr/bin/env -S NODE_OPTIONS=--no-warnings node --experimental-x")
	withShebangLookPath(t, func(name string) (string, error) {
		switch name {
		case "cli":
			return script, nil
		case "node":
			return "/usr/bin/node", nil
		}
		return "", os.ErrNotExist
	})

	_, interpreter, interpreterArgs, ok := resolveShebang("cli")
	if !ok {
		t.Fatal("expected env -S shebang to resolve")
	}
	if interpreter != "/usr/bin/node" {
		t.Errorf("interpreter = %q, want node", interpreter)
	}
	want := []string{"--experimental-x"}
	if len(interpreterArgs) != len(want) || interpreterArgs[0] != want[0] {
		t.Errorf("interpreterArgs = %v, want %v", interpreterArgs, want)
	}
}

func TestResolveShebangAbsoluteInterpreterThatExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shebang contract")
	}
	interpreter := filepath.Join(t.TempDir(), "node")
	if err := os.WriteFile(interpreter, []byte("#!/bin/true\n"), 0o755); err != nil {
		t.Fatalf("write interpreter: %v", err)
	}
	dir := t.TempDir()
	script := writeScript(t, dir, "tool", "#!"+interpreter+" --jitless")
	withShebangLookPath(t, func(name string) (string, error) {
		if name == "tool" {
			return script, nil
		}
		return "", os.ErrNotExist
	})

	gotInterpreter, interpreterArgs, ok := func() (string, []string, bool) {
		_, i, a, ok := resolveShebang("tool")
		return i, a, ok
	}()
	if !ok {
		t.Fatal("expected existing absolute interpreter to resolve")
	}
	if gotInterpreter != interpreter {
		t.Errorf("interpreter = %q, want %q", gotInterpreter, interpreter)
	}
	if len(interpreterArgs) != 1 || interpreterArgs[0] != "--jitless" {
		t.Errorf("interpreterArgs = %v, want [--jitless]", interpreterArgs)
	}
}

func TestResolveShebangMissingAbsoluteInterpreterFallsBackToPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shebang contract")
	}
	dir := t.TempDir()
	// A shebang path Termux does not provide must resolve by base name on PATH.
	script := writeScript(t, dir, "hook", "#!/opt/termux-absent/node")
	withShebangLookPath(t, func(name string) (string, error) {
		switch name {
		case "hook":
			return script, nil
		case "node":
			return "/data/data/com.termux/files/usr/bin/node", nil
		}
		return "", os.ErrNotExist
	})

	_, interpreter, _, ok := resolveShebang("hook")
	if !ok {
		t.Fatal("expected an absent absolute interpreter to resolve through PATH on Termux")
	}
	if interpreter != "/data/data/com.termux/files/usr/bin/node" {
		t.Errorf("interpreter = %q, want the PATH-resolved node", interpreter)
	}
}

func TestResolveShebangRejectsUnresolvableTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shebang contract")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "node-binary")
	if err := os.WriteFile(binary, []byte("\x7fELF\x02\x01\x01\x00native"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	script := writeScript(t, dir, "broken", "#!/usr/bin/env ghost-interpreter")
	empty := writeScript(t, dir, "no-shebang", "")
	truncated := filepath.Join(dir, "truncated")
	if err := os.WriteFile(truncated, []byte("#!"), 0o755); err != nil {
		t.Fatalf("write truncated: %v", err)
	}

	withShebangLookPath(t, func(name string) (string, error) {
		switch name {
		case "node-binary":
			return binary, nil
		case "broken":
			return script, nil
		case "empty":
			return empty, nil
		case "truncated":
			return truncated, nil
		}
		return "", os.ErrNotExist
	})

	for _, name := range []string{"node-binary", "broken", "empty", "truncated", "absent"} {
		if _, _, _, ok := resolveShebang(name); ok {
			t.Errorf("%s: expected unresolved invocation to keep the original command", name)
		}
	}
}

func TestCommandBuildsInterpreterArgvOnTermux(t *testing.T) {
	if !IsTermux() {
		t.Skip("Termux-only rewrite")
	}
	dir := t.TempDir()
	script := writeScript(t, dir, "pi", "#!/usr/bin/env node")
	withShebangLookPath(t, func(name string) (string, error) {
		switch name {
		case "pi":
			return script, nil
		case "node":
			return "/data/data/com.termux/files/usr/bin/node", nil
		}
		return "", os.ErrNotExist
	})

	cmd := Command("pi", "install", "npm:gentle-pi")
	want := []string{script, "install", "npm:gentle-pi"}
	if len(cmd.Args) != len(want)+1 {
		t.Fatalf("cmd.Args = %v, want interpreter plus %v", cmd.Args, want)
	}
	if cmd.Args[0] != "/data/data/com.termux/files/usr/bin/node" {
		t.Errorf("cmd.Args[0] = %q, want the resolved interpreter", cmd.Args[0])
	}
	if strings.Join(cmd.Args[1:], "\x00") != strings.Join(want, "\x00") {
		t.Errorf("cmd.Args[1:] = %v, want %v", cmd.Args[1:], want)
	}
}

func TestCommandContextKeepsContextOnTermux(t *testing.T) {
	if !IsTermux() {
		t.Skip("Termux-only rewrite")
	}
	dir := t.TempDir()
	script := writeScript(t, dir, "pi", "#!/usr/bin/env node")
	withShebangLookPath(t, func(name string) (string, error) {
		switch name {
		case "pi":
			return script, nil
		case "node":
			return "/data/data/com.termux/files/usr/bin/node", nil
		}
		return "", os.ErrNotExist
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := CommandContext(ctx, "pi", "--version")
	if cmd.Args[0] != "/data/data/com.termux/files/usr/bin/node" || cmd.Args[1] != script {
		t.Fatalf("cmd.Args = %v, want interpreter and script", cmd.Args)
	}
	if cmd.Cancel == nil {
		t.Error("expected CommandContext to keep cancellation wired to the interpreter")
	}
}
