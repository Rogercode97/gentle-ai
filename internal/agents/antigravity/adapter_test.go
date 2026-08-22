package antigravity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v2/internal/model"
	"github.com/gentleman-programming/gentle-ai/v2/internal/system"
)

// makeStatFn returns a statPath function that reports the given paths as existing
// directories. Any path not in the set returns os.ErrNotExist.
func makeStatFn(existingDirs ...string) func(string) statResult {
	set := make(map[string]struct{}, len(existingDirs))
	for _, d := range existingDirs {
		set[d] = struct{}{}
	}
	return func(path string) statResult {
		if _, ok := set[path]; ok {
			return statResult{isDir: true}
		}
		return statResult{err: os.ErrNotExist}
	}
}

// --- antigravityVariantDir ---

func TestAntigravityVariantDir(t *testing.T) {
	home := t.TempDir()
	cliDir := filepath.Join(home, ".gemini", "antigravity-cli")
	desktopDir := filepath.Join(home, ".gemini", "antigravity-desktop")

	tests := []struct {
		name         string
		existingDirs []string
		wantSuffix   string // last two path segments expected
	}{
		{
			name:         "only CLI dir exists resolves to CLI",
			existingDirs: []string{cliDir},
			wantSuffix:   filepath.Join(".gemini", "antigravity-cli"),
		},
		{
			name:         "only Desktop dir exists resolves to Desktop",
			existingDirs: []string{desktopDir},
			wantSuffix:   filepath.Join(".gemini", "antigravity-desktop"),
		},
		{
			name:         "both exist prefers Desktop",
			existingDirs: []string{cliDir, desktopDir},
			wantSuffix:   filepath.Join(".gemini", "antigravity-desktop"),
		},
		{
			name:         "neither exists falls back to CLI",
			existingDirs: []string{},
			wantSuffix:   filepath.Join(".gemini", "antigravity-cli"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Adapter{statPath: makeStatFn(tt.existingDirs...)}
			got := a.antigravityVariantDir(home)
			want := filepath.Join(home, tt.wantSuffix)
			if got != want {
				t.Fatalf("antigravityVariantDir() = %q, want %q", got, want)
			}
		})
	}
}

// --- Detect ---

func TestDetect(t *testing.T) {
	home := t.TempDir()
	cliDir := filepath.Join(home, ".gemini", "antigravity-cli")
	desktopDir := filepath.Join(home, ".gemini", "antigravity-desktop")

	tests := []struct {
		name            string
		existingDirs    []string
		overrideStat    func(string) statResult // optional; overrides makeStatFn when set
		wantInstalled   bool
		wantBinaryPath  string
		wantConfigPath  string
		wantConfigFound bool
		wantErr         bool
	}{
		{
			name:            "CLI dir found, no Desktop",
			existingDirs:    []string{cliDir},
			wantInstalled:   true,
			wantConfigPath:  cliDir,
			wantConfigFound: true,
		},
		{
			name:            "Desktop dir found, no CLI",
			existingDirs:    []string{desktopDir},
			wantInstalled:   true,
			wantConfigPath:  desktopDir,
			wantConfigFound: true,
		},
		{
			name:            "both exist, prefers Desktop",
			existingDirs:    []string{cliDir, desktopDir},
			wantInstalled:   true,
			wantConfigPath:  desktopDir,
			wantConfigFound: true,
		},
		{
			name:            "neither exists, falls back to CLI path",
			existingDirs:    []string{},
			wantInstalled:   false,
			wantConfigPath:  filepath.Join(home, ".gemini", "antigravity"),
			wantConfigFound: false,
		},
		{
			name: "stat error bubbles up",
			overrideStat: func(string) statResult {
				return statResult{err: errors.New("permission denied")}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statFn := makeStatFn(tt.existingDirs...)
			if tt.overrideStat != nil {
				statFn = tt.overrideStat
			}
			a := &Adapter{statPath: statFn}

			installed, binaryPath, configPath, configFound, err := a.Detect(context.Background(), home)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Detect() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if installed != tt.wantInstalled {
				t.Fatalf("Detect() installed = %v, want %v", installed, tt.wantInstalled)
			}
			if binaryPath != tt.wantBinaryPath {
				t.Fatalf("Detect() binaryPath = %q, want %q", binaryPath, tt.wantBinaryPath)
			}
			if configPath != tt.wantConfigPath {
				t.Fatalf("Detect() configPath = %q, want %q", configPath, tt.wantConfigPath)
			}
			if configFound != tt.wantConfigFound {
				t.Fatalf("Detect() configFound = %v, want %v", configFound, tt.wantConfigFound)
			}
		})
	}
}

// --- Config paths ---

func TestConfigPathsCLIOnly(t *testing.T) {
	home := t.TempDir()
	cliDir := filepath.Join(home, ".gemini", "antigravity-cli")
	a := &Adapter{statPath: makeStatFn(cliDir)}

	if got := a.GlobalConfigDir(home); got != cliDir {
		t.Fatalf("GlobalConfigDir() = %q, want %q", got, cliDir)
	}
	if got := a.SkillsDir(home); got != filepath.Join(cliDir, "skills") {
		t.Fatalf("SkillsDir() = %q, want %q", got, filepath.Join(cliDir, "skills"))
	}
	if got := a.SettingsPath(home); got != filepath.Join(cliDir, "settings.json") {
		t.Fatalf("SettingsPath() = %q, want %q", got, filepath.Join(cliDir, "settings.json"))
	}
	if got := a.MCPConfigPath(home, "ctx7"); got != filepath.Join(cliDir, "mcp_config.json") {
		t.Fatalf("MCPConfigPath() = %q, want Gentle AI CLI path", got)
	}
}

func TestConfigPathsDesktopOnly(t *testing.T) {
	home := t.TempDir()
	desktopDir := filepath.Join(home, ".gemini", "antigravity-desktop")
	a := &Adapter{statPath: makeStatFn(desktopDir)}

	if got := a.GlobalConfigDir(home); got != desktopDir {
		t.Fatalf("GlobalConfigDir() = %q, want %q", got, desktopDir)
	}
	if got := a.SkillsDir(home); got != filepath.Join(desktopDir, "skills") {
		t.Fatalf("SkillsDir() = %q, want %q", got, filepath.Join(desktopDir, "skills"))
	}
	if got := a.SettingsPath(home); got != filepath.Join(desktopDir, "settings.json") {
		t.Fatalf("SettingsPath() = %q, want %q", got, filepath.Join(desktopDir, "settings.json"))
	}
	if got := a.MCPConfigPath(home, "ctx7"); got != filepath.Join(desktopDir, "mcp_config.json") {
		t.Fatalf("MCPConfigPath() = %q, want Gentle AI desktop path", got)
	}
}

func TestConfigPathsBothExistPrefersDesktop(t *testing.T) {
	home := t.TempDir()
	cliDir := filepath.Join(home, ".gemini", "antigravity-cli")
	desktopDir := filepath.Join(home, ".gemini", "antigravity-desktop")
	a := &Adapter{statPath: makeStatFn(cliDir, desktopDir)}

	if got := a.GlobalConfigDir(home); got != desktopDir {
		t.Fatalf("GlobalConfigDir() = %q, want %q (should prefer Desktop)", got, desktopDir)
	}
	if got := a.SkillsDir(home); got != filepath.Join(desktopDir, "skills") {
		t.Fatalf("SkillsDir() = %q, want %q", got, filepath.Join(desktopDir, "skills"))
	}
}

func TestConfigPathsNeitherExistsFallsBackToCLI(t *testing.T) {
	home := t.TempDir()
	a := &Adapter{statPath: makeStatFn()}

	if got := a.GlobalConfigDir(home); got != filepath.Join(home, ".gemini", "antigravity") {
		t.Fatalf("GlobalConfigDir() = %q, want upstream legacy root", got)
	}
}

func TestUpstreamDetectionScenarios(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(home, ".gemini", "antigravity")
	config := filepath.Join(home, ".gemini", "config")
	marker := filepath.Join(config, ".migrated")
	geminiOnly := filepath.Join(home, ".gemini")
	tests := []struct {
		name     string
		existing []string
		want     bool
		root     string
	}{
		{"legacy", []string{legacy}, true, legacy},
		{"unified", []string{config, marker}, true, config},
		{"half migrated marker", []string{marker}, true, config},
		{"Gemini only", []string{geminiOnly}, false, legacy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Adapter{statPath: makeStatFn(tt.existing...)}
			installed, _, _, _, err := a.Detect(context.Background(), home)
			if err != nil || installed != tt.want {
				t.Fatalf("Detect() = %v, %v, want %v", installed, err, tt.want)
			}
			if got := a.GlobalConfigDir(home); got != tt.root {
				t.Errorf("GlobalConfigDir() = %q, want %q", got, tt.root)
			}
		})
	}
}

func TestConfigPathsStaticPaths(t *testing.T) {
	// SystemPromptDir and SystemPromptFile are not variant-dependent.
	a := NewAdapter()
	home := "/tmp/home"

	if got := a.SystemPromptDir(home); got != filepath.Join(home, ".gemini") {
		t.Fatalf("SystemPromptDir() = %q, want %q", got, filepath.Join(home, ".gemini"))
	}
	if got := a.SystemPromptFile(home); got != filepath.Join(home, ".gemini", "GEMINI.md") {
		t.Fatalf("SystemPromptFile() = %q, want %q", got, filepath.Join(home, ".gemini", "GEMINI.md"))
	}
}

// --- Installation ---

func TestInstallCommand(t *testing.T) {
	a := NewAdapter()

	_, err := a.InstallCommand(system.PlatformProfile{OS: "darwin"})
	if err == nil {
		t.Fatal("InstallCommand() expected error for CLI agent, got nil")
	}

	var notInstallable AgentNotInstallableError
	if !errors.As(err, &notInstallable) {
		t.Fatalf("InstallCommand() error type = %T, want AgentNotInstallableError", err)
	}

	if notInstallable.Agent != model.AgentAntigravity {
		t.Fatalf("AgentNotInstallableError.Agent = %q, want %q", notInstallable.Agent, model.AgentAntigravity)
	}
}

// --- Capabilities ---

func TestCapabilities(t *testing.T) {
	a := NewAdapter()

	if !a.SupportsSkills() {
		t.Fatal("SupportsSkills() = false, want true")
	}
	if !a.SupportsSystemPrompt() {
		t.Fatal("SupportsSystemPrompt() = false, want true")
	}
	if !a.SupportsMCP() {
		t.Fatal("SupportsMCP() = false, want true")
	}
	if a.SupportsOutputStyles() {
		t.Fatal("SupportsOutputStyles() = true, want false")
	}
	if a.SupportsSlashCommands() {
		t.Fatal("SupportsSlashCommands() = true, want false")
	}
	if !a.SupportsSubAgents() {
		t.Fatal("SupportsSubAgents() = false, want true")
	}
	if got := a.OutputStyleDir("/tmp/home"); got != "" {
		t.Fatalf("OutputStyleDir() = %q, want empty string", got)
	}
	if got := a.CommandsDir("/tmp/home"); got != "" {
		t.Fatalf("CommandsDir() = %q, want empty string", got)
	}
	wantSubAgentsDir := filepath.Join(a.GlobalConfigDir("/tmp/home"), "agents")
	if got := a.SubAgentsDir("/tmp/home"); got != wantSubAgentsDir {
		t.Fatalf("SubAgentsDir() = %q, want %q", got, wantSubAgentsDir)
	}
	if got := a.EmbeddedSubAgentsDir(); got != "antigravity/agents" {
		t.Fatalf("EmbeddedSubAgentsDir() = %q, want %q", got, "antigravity/agents")
	}
}

// --- Strategies ---

func TestStrategies(t *testing.T) {
	a := NewAdapter()

	if got := a.SystemPromptStrategy(); got != model.StrategyAppendToFile {
		t.Fatalf("SystemPromptStrategy() = %v, want StrategyAppendToFile", got)
	}
	if got := a.MCPStrategy(); got != model.StrategyMCPConfigFile {
		t.Fatalf("MCPStrategy() = %v, want StrategyMCPConfigFile", got)
	}
}

// --- Identity ---

func TestIdentity(t *testing.T) {
	a := NewAdapter()

	if got := a.Agent(); got != model.AgentAntigravity {
		t.Fatalf("Agent() = %q, want %q", got, model.AgentAntigravity)
	}
	if got := a.Tier(); got != model.TierFull {
		t.Fatalf("Tier() = %q, want %q", got, model.TierFull)
	}
}

func TestAdapter_DetectLowModel(t *testing.T) {
	defer func() {
		os.Unsetenv("GEMINI_MODEL")
		os.Unsetenv("ANTIGRAVITY_MODEL")
	}()

	t.Run("EnvVar GEMINI_MODEL Small", func(t *testing.T) {
		os.Setenv("GEMINI_MODEL", "gemini-1.5-flash")
		os.Unsetenv("ANTIGRAVITY_MODEL")
		a := NewAdapter()
		if !a.DetectLowModel(t.TempDir()) {
			t.Fatal("expected DetectLowModel to return true for small GEMINI_MODEL")
		}
	})

	t.Run("EnvVar GEMINI_MODEL Capable", func(t *testing.T) {
		os.Setenv("GEMINI_MODEL", "gpt-4o")
		os.Unsetenv("ANTIGRAVITY_MODEL")
		a := NewAdapter()
		if a.DetectLowModel(t.TempDir()) {
			t.Fatal("expected DetectLowModel to return false for capable GEMINI_MODEL")
		}
	})

	t.Run("EnvVar ANTIGRAVITY_MODEL Small", func(t *testing.T) {
		os.Unsetenv("GEMINI_MODEL")
		os.Setenv("ANTIGRAVITY_MODEL", "gpt-4o-mini")
		a := NewAdapter()
		if !a.DetectLowModel(t.TempDir()) {
			t.Fatal("expected DetectLowModel to return true for small ANTIGRAVITY_MODEL")
		}
	})

	t.Run("Settings JSON model", func(t *testing.T) {
		os.Unsetenv("GEMINI_MODEL")
		os.Unsetenv("ANTIGRAVITY_MODEL")
		home := t.TempDir()
		cliDir := filepath.Join(home, ".gemini", "antigravity-cli")
		if err := os.MkdirAll(cliDir, 0755); err != nil {
			t.Fatal(err)
		}
		a := &Adapter{statPath: makeStatFn(cliDir)}
		settingsPath := a.SettingsPath(home)
		content := `{"model": "claude-3-5-haiku"}`
		if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		if !a.DetectLowModel(home) {
			t.Fatal("expected DetectLowModel to return true for small model in settings.json model field")
		}
	})

	t.Run("Settings JSON modelId", func(t *testing.T) {
		os.Unsetenv("GEMINI_MODEL")
		os.Unsetenv("ANTIGRAVITY_MODEL")
		home := t.TempDir()
		cliDir := filepath.Join(home, ".gemini", "antigravity-cli")
		if err := os.MkdirAll(cliDir, 0755); err != nil {
			t.Fatal(err)
		}
		a := &Adapter{statPath: makeStatFn(cliDir)}
		settingsPath := a.SettingsPath(home)
		content := `{"modelId": "gpt-4o-mini"}`
		if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		if !a.DetectLowModel(home) {
			t.Fatal("expected DetectLowModel to return true for small modelId in settings.json")
		}
	})
}

func TestAdapter_GetWorkspaceRules(t *testing.T) {
	t.Run("rules file exists", func(t *testing.T) {
		cwd := t.TempDir()
		rulesDir := filepath.Join(cwd, ".agents", "rules")
		if err := os.MkdirAll(rulesDir, 0755); err != nil {
			t.Fatal(err)
		}
		rulesFile := filepath.Join(rulesDir, "sdd-workflow.md")
		expectedContent := "# Workspace Workflow Rules\nStrict mode enabled."
		if err := os.WriteFile(rulesFile, []byte(expectedContent), 0644); err != nil {
			t.Fatal(err)
		}

		a := NewAdapter()
		rules, err := a.GetWorkspaceRules(cwd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rules != expectedContent {
			t.Fatalf("got rules = %q, want %q", rules, expectedContent)
		}
	})

	t.Run("rules file missing", func(t *testing.T) {
		cwd := t.TempDir()
		a := NewAdapter()
		rules, err := a.GetWorkspaceRules(cwd)
		if err != nil {
			t.Fatalf("unexpected error for missing rules: %v", err)
		}
		if rules != "" {
			t.Fatalf("expected empty rules, got: %q", rules)
		}
	})
}
