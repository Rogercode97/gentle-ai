package reviewerprovider

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// AntigravityAdapter invokes Antigravity CLI (agy or gemini) with an opaque provider
// invocation and returns its stdout bytes without interpreting them.
type AntigravityAdapter struct {
	LookPath       func(string) (string, error)
	commandContext func(context.Context, string, ...string) *exec.Cmd
}

// NewAntigravityAdapter returns an adapter using the Antigravity binary resolved from PATH.
func NewAntigravityAdapter() *AntigravityAdapter {
	return &AntigravityAdapter{LookPath: exec.LookPath, commandContext: exec.CommandContext}
}

// Review runs Antigravity in an empty temporary directory. The provider
// material is delivered through stdin so command arguments stay opaque.
func (adapter *AntigravityAdapter) Review(ctx context.Context, invocation Invocation) ([]byte, error) {
	lookPath := adapter.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	binary, err := lookPath("agy")
	if err != nil {
		binary, err = lookPath("gemini")
	}
	if err != nil {
		return nil, fmt.Errorf("antigravity reviewer transport unavailable: %w", err)
	}

	scratch, err := os.MkdirTemp("", "gentle-ai-antigravity-reviewer-*")
	if err != nil {
		return nil, fmt.Errorf("antigravity reviewer transport unavailable: create scratch directory: %w", err)
	}
	defer os.RemoveAll(scratch)

	commandContext := adapter.commandContext
	if commandContext == nil {
		commandContext = exec.CommandContext
	}
	command := commandContext(ctx, binary)
	command.Dir = scratch
	command.Stdin = bytes.NewReader(invocation.Prompt())
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("antigravity reviewer transport failed: %w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}
