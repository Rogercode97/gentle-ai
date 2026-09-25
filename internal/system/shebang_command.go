package system

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// shebangLookPath is a test seam over exec.LookPath.
var shebangLookPath = exec.LookPath

// shebangMaxBytes bounds the first line read from a script entrypoint. Every
// shebang this resolver supports is far shorter, and an unbounded read of a
// large binary would be pure waste.
const shebangMaxBytes = 256

// Command builds the command used to run a managed external program.
//
// Off Termux it is exactly exec.Command. On Termux it additionally resolves a
// POSIX shebang: Go issues the execve(2) syscall directly rather than calling
// libc execve, so Termux's libtermux-exec preload cannot rewrite the shebang
// of npm-installed programs, whose upstream `#!/usr/bin/env <interpreter>`
// line names an interpreter path that does not exist on Android. Resolving the
// interpreter here keeps `pi`, `npx`, and any other npm-installed CLI
// launchable from a Go-managed process.
func Command(name string, args ...string) *exec.Cmd {
	return commandFor(context.Background(), name, args)
}

// CommandContext is Command with cancellation, for probes that must not outlive
// their caller.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return commandFor(ctx, name, args)
}

func commandFor(ctx context.Context, name string, args []string) *exec.Cmd {
	if !IsTermux() {
		return exec.CommandContext(ctx, name, args...)
	}
	script, interpreter, interpreterArgs, ok := resolveShebang(name)
	if !ok {
		return exec.CommandContext(ctx, name, args...)
	}
	// argv is assembled as a slice: the script path and the original arguments
	// are never concatenated into a command line.
	argv := make([]string, 0, len(interpreterArgs)+1+len(args))
	argv = append(argv, interpreterArgs...)
	argv = append(argv, script)
	argv = append(argv, args...)
	return exec.CommandContext(ctx, interpreter, argv...)
}

// resolveShebang reports the script path, interpreter, and interpreter
// arguments for name. It reports false whenever the target is not a script with
// a resolvable shebang, leaving the original invocation untouched so the kernel
// keeps ownership of any error.
func resolveShebang(name string) (script, interpreter string, interpreterArgs []string, ok bool) {
	script, err := shebangLookPath(name)
	if err != nil {
		return "", "", nil, false
	}
	line, ok := readShebang(script)
	if !ok {
		return "", "", nil, false
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", "", nil, false
	}
	interpreterPath, rest := fields[0], fields[1:]
	// `env` is a launcher, not the interpreter: the real one is the first
	// non-option, non-assignment operand.
	if filepath.Base(interpreterPath) == "env" {
		interpreterPath, rest = envInterpreter(rest)
	}
	resolved, err := resolveInterpreter(interpreterPath)
	if err != nil {
		return "", "", nil, false
	}
	return script, resolved, rest, true
}

// envInterpreter extracts the interpreter operand from `env` operands, skipping
// options and variable assignments. It reports an empty interpreter when none
// remains, which leaves the invocation unresolved.
func envInterpreter(operands []string) (string, []string) {
	for i, operand := range operands {
		if operand == "" || strings.HasPrefix(operand, "-") || strings.Contains(operand, "=") {
			continue
		}
		return operand, operands[i+1:]
	}
	return "", nil
}

// resolveInterpreter maps a shebang interpreter to an executable path. An
// absolute path that exists is used as is; anything else, including the
// /usr/bin/env and /bin/sh paths that Termux does not provide, is looked up by
// base name on PATH.
func resolveInterpreter(interpreter string) (string, error) {
	if interpreter == "" {
		return "", exec.ErrNotFound
	}
	if filepath.IsAbs(interpreter) {
		if _, err := os.Stat(interpreter); err == nil {
			return interpreter, nil
		}
		interpreter = filepath.Base(interpreter)
	}
	resolved, err := shebangLookPath(interpreter)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

// readShebang returns the first line of file when it is a shebang script.
func readShebang(file string) (string, bool) {
	f, err := os.Open(file)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()

	reader := bufio.NewReader(f)
	head := make([]byte, shebangMaxBytes)
	n, err := reader.Read(head)
	if n < 2 || (err != nil && !errors.Is(err, io.EOF)) {
		return "", false
	}
	head = head[:n]
	if string(head[:2]) != "#!" {
		return "", false
	}
	line := string(head[2:])
	if idx := strings.IndexAny(line, "\r\n"); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(line), true
}
