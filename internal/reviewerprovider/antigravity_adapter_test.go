package reviewerprovider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const antigravityAdapterHelperEnvironment = "GENTLE_AI_REVIEWER_PROVIDER_ANTIGRAVITY_HELPER"
const antigravityAdapterPromptPathEnvironment = "GENTLE_AI_REVIEWER_PROVIDER_ANTIGRAVITY_PROMPT_PATH"
const antigravityAdapterScratchPathEnvironment = "GENTLE_AI_REVIEWER_PROVIDER_ANTIGRAVITY_SCRATCH_PATH"

func TestAntigravityAdapterReturnsNoBytesWhenUnavailable(t *testing.T) {
	adapter := &AntigravityAdapter{
		LookPath: func(binary string) (string, error) {
			return "", errors.New("not found")
		},
	}
	raw, err := adapter.Review(context.Background(), NewInvocation([]byte("provider prompt")))
	if err == nil || !strings.Contains(err.Error(), "antigravity reviewer transport unavailable") {
		t.Fatalf("Review() error = %v, want unavailable transport error", err)
	}
	if raw != nil {
		t.Fatalf("Review() raw = %q with transport error, want no result bytes", raw)
	}
}

func TestAntigravityAdapterPrefersAgyOverGemini(t *testing.T) {
	var lookedUp []string
	var executedBinary string

	adapter := &AntigravityAdapter{
		LookPath: func(binary string) (string, error) {
			lookedUp = append(lookedUp, binary)
			if binary == "agy" {
				return "/path/to/agy", nil
			}
			if binary == "gemini" {
				return "/path/to/gemini", nil
			}
			return "", errors.New("not found")
		},
		commandContext: func(ctx context.Context, binary string, args ...string) *exec.Cmd {
			executedBinary = binary
			return exec.CommandContext(ctx, "true")
		},
	}

	_, _ = adapter.Review(context.Background(), NewInvocation([]byte("prompt")))
	if len(lookedUp) == 0 || lookedUp[0] != "agy" {
		t.Fatalf("LookPath called with %v, want agy first", lookedUp)
	}
	if executedBinary != "/path/to/agy" {
		t.Fatalf("executed binary = %q, want /path/to/agy", executedBinary)
	}
}

func TestAntigravityAdapterFallsBackToGeminiWhenAgyMissing(t *testing.T) {
	var lookedUp []string
	var executedBinary string

	adapter := &AntigravityAdapter{
		LookPath: func(binary string) (string, error) {
			lookedUp = append(lookedUp, binary)
			if binary == "agy" {
				return "", errors.New("agy not found")
			}
			if binary == "gemini" {
				return "/path/to/gemini", nil
			}
			return "", errors.New("not found")
		},
		commandContext: func(ctx context.Context, binary string, args ...string) *exec.Cmd {
			executedBinary = binary
			return exec.CommandContext(ctx, "true")
		},
	}

	_, _ = adapter.Review(context.Background(), NewInvocation([]byte("prompt")))
	if len(lookedUp) < 2 || lookedUp[0] != "agy" || lookedUp[1] != "gemini" {
		t.Fatalf("LookPath called with %v, want agy then gemini", lookedUp)
	}
	if executedBinary != "/path/to/gemini" {
		t.Fatalf("executed binary = %q, want /path/to/gemini", executedBinary)
	}
}

func TestAntigravityAdapterPipesStdinAndReturnsUntouchedRawOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the helper process uses POSIX argument handling")
	}
	promptPath := filepath.Join(t.TempDir(), "prompt")
	scratchPath := filepath.Join(t.TempDir(), "scratch_used")
	t.Setenv(antigravityAdapterHelperEnvironment, "1")
	t.Setenv(antigravityAdapterPromptPathEnvironment, promptPath)
	t.Setenv(antigravityAdapterScratchPathEnvironment, scratchPath)

	var recordedScratchDir string
	adapter := &AntigravityAdapter{
		LookPath: func(string) (string, error) { return "agy", nil },
		commandContext: func(ctx context.Context, _ string, arguments ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestAntigravityAdapterHelperProcess$", "--"}, arguments...)...)
			return cmd
		},
	}
	prompt := []byte("provider prompt\nwith bytes \x00\xff")
	raw, err := adapter.Review(context.Background(), NewInvocation(prompt))
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("raw\x00antigravity\xffoutput"); !bytes.Equal(raw, want) {
		t.Fatalf("Review() = %q, want untouched raw bytes %q", raw, want)
	}
	if got, err := os.ReadFile(promptPath); err != nil || !bytes.Equal(got, prompt) {
		t.Fatalf("reviewer stdin = %q, %v; want %q", got, err, prompt)
	}
	usedScratch, err := os.ReadFile(scratchPath)
	if err != nil {
		t.Fatal(err)
	}
	recordedScratchDir = string(usedScratch)
	if !strings.Contains(recordedScratchDir, "gentle-ai-antigravity-reviewer-") {
		t.Fatalf("scratch directory %q does not match pattern", recordedScratchDir)
	}
	// Verify scratch dir was cleaned up
	if _, err := os.Stat(recordedScratchDir); !os.IsNotExist(err) {
		t.Fatalf("scratch directory %q still exists after Review() completion", recordedScratchDir)
	}
}

func TestAntigravityAdapterHelperProcess(t *testing.T) {
	if os.Getenv(antigravityAdapterHelperEnvironment) != "1" {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	if err := os.WriteFile(os.Getenv(antigravityAdapterScratchPathEnvironment), []byte(cwd), 0o600); err != nil {
		os.Exit(1)
	}
	prompt, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(1)
	}
	if err := os.WriteFile(os.Getenv(antigravityAdapterPromptPathEnvironment), prompt, 0o600); err != nil {
		os.Exit(1)
	}
	_, _ = os.Stdout.Write([]byte("raw\x00antigravity\xffoutput"))
	os.Exit(0)
}
