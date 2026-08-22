package antigravity

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gentleman-programming/gentle-ai/v2/internal/agents/capabilitymanifest"
	"github.com/gentleman-programming/gentle-ai/v2/internal/model"
	"github.com/gentleman-programming/gentle-ai/v2/internal/system"
)

type statResult struct {
	isDir bool
	err   error
}

type Adapter struct {
	statPath func(string) statResult
}

func NewAdapter() *Adapter {
	return &Adapter{
		statPath: defaultStat,
	}
}

// antigravityVariantDir retains Gentle AI's legacy settings/skills selection.
func (a *Adapter) antigravityVariantDir(homeDir string) string {
	desktop := filepath.Join(homeDir, ".gemini", "antigravity-desktop")
	if stat := a.statPath(desktop); stat.err == nil {
		return desktop
	}
	return filepath.Join(homeDir, ".gemini", "antigravity-cli")
}

func (a *Adapter) antigravityRoot(homeDir string) string {
	return filepath.Join(homeDir, ".gemini", "antigravity")
}

func (a *Adapter) migratedMarker(homeDir string) string {
	return filepath.Join(homeDir, ".gemini", "config", ".migrated")
}

// --- Identity ---

func (a *Adapter) Agent() model.AgentID {
	return model.AgentAntigravity
}

func (a *Adapter) Tier() model.SupportTier {
	return model.TierFull
}

// --- Detection ---

func (a *Adapter) Detect(_ context.Context, homeDir string) (bool, string, string, bool, error) {
	configPath := a.GlobalConfigDir(homeDir)
	for _, evidence := range []string{a.antigravityRoot(homeDir), a.migratedMarker(homeDir), a.antigravityVariantDir(homeDir)} {
		stat := a.statPath(evidence)
		if stat.err == nil {
			return true, "", configPath, true, nil
		}
		if !os.IsNotExist(stat.err) {
			return false, "", "", false, stat.err
		}
	}
	return false, "", configPath, false, nil
}

// --- Installation ---

func (a *Adapter) CapabilityManifest() capabilitymanifest.AgentCapabilityManifest {
	return capabilitymanifest.MustForAgent(model.AgentAntigravity)
}

func (a *Adapter) InstallCommand(_ system.PlatformProfile) ([][]string, error) {
	return nil, AgentNotInstallableError{Agent: model.AgentAntigravity}
}

// --- Config paths ---

func (a *Adapter) GlobalConfigDir(homeDir string) string {
	if stat := a.statPath(a.migratedMarker(homeDir)); stat.err == nil {
		return filepath.Join(homeDir, ".gemini", "config")
	}
	if stat := a.statPath(a.antigravityRoot(homeDir)); stat.err == nil {
		return a.antigravityRoot(homeDir)
	}
	if stat := a.statPath(a.antigravityVariantDir(homeDir)); stat.err == nil {
		return a.antigravityVariantDir(homeDir)
	}
	return a.antigravityRoot(homeDir)
}

func (a *Adapter) SystemPromptDir(homeDir string) string {
	return filepath.Join(homeDir, ".gemini")
}

func (a *Adapter) SystemPromptFile(homeDir string) string {
	return filepath.Join(homeDir, ".gemini", "GEMINI.md")
}

func (a *Adapter) SkillsDir(homeDir string) string {
	return filepath.Join(a.antigravityVariantDir(homeDir), "skills")
}

func (a *Adapter) SettingsPath(homeDir string) string {
	return filepath.Join(a.antigravityVariantDir(homeDir), "settings.json")
}

// --- Config strategies ---

func (a *Adapter) SystemPromptStrategy() model.SystemPromptStrategy {
	return model.StrategyAppendToFile
}

func (a *Adapter) MCPStrategy() model.MCPStrategy {
	return model.StrategyMCPConfigFile
}

// --- MCP ---

func (a *Adapter) MCPConfigPath(homeDir string, _ string) string {
	return filepath.Join(a.antigravityVariantDir(homeDir), "mcp_config.json")
}

// --- Optional capabilities ---

func (a *Adapter) SupportsOutputStyles() bool {
	return a.CapabilityManifest().Features.OutputStyles
}

func (a *Adapter) OutputStyleDir(_ string) string {
	return ""
}

func (a *Adapter) SupportsSlashCommands() bool {
	return a.CapabilityManifest().Features.SlashCommands
}

func (a *Adapter) CommandsDir(_ string) string {
	return ""
}

func (a *Adapter) SupportsSubAgents() bool {
	return a.CapabilityManifest().Features.FileSubAgents
}

func (a *Adapter) SubAgentsDir(homeDir string) string {
	return filepath.Join(a.GlobalConfigDir(homeDir), "agents")
}

func (a *Adapter) EmbeddedSubAgentsDir() string {
	return "antigravity/agents"
}

func (a *Adapter) SupportsSkills() bool {
	return a.CapabilityManifest().Features.Skills
}

func (a *Adapter) SupportsSystemPrompt() bool {
	return a.CapabilityManifest().Features.SystemPrompt
}

func (a *Adapter) SupportsMCP() bool {
	return a.CapabilityManifest().Features.MCP
}

// DetectLowModel returns true if the active model is classified as low-tier.
// It checks GEMINI_MODEL and ANTIGRAVITY_MODEL environment variables, falling back to settings.json.
func (a *Adapter) DetectLowModel(homeDir string) bool {
	// 1. Check env vars
	for _, env := range []string{"GEMINI_MODEL", "ANTIGRAVITY_MODEL"} {
		if val := os.Getenv(env); val != "" {
			return model.ModelCapability(val) == "small"
		}
	}
	// 2. Fall back to settings.json
	settingsPath := a.SettingsPath(homeDir)
	if data, err := os.ReadFile(settingsPath); err == nil {
		var cfg struct {
			Model   string `json:"model"`
			ModelID string `json:"modelId"`
		}
		if err := json.Unmarshal(data, &cfg); err == nil {
			m := cfg.Model
			if m == "" {
				m = cfg.ModelID
			}
			if m != "" {
				return model.ModelCapability(m) == "small"
			}
		}
	}
	return false
}

// GetWorkspaceRules loads the workspace-specific workflow rules from .agents/rules/sdd-workflow.md.
// If the file is missing, it returns an empty string without an error.
func (a *Adapter) GetWorkspaceRules(cwd string) (string, error) {
	rulesPath := filepath.Join(cwd, ".agents", "rules", "sdd-workflow.md")
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

type AgentNotInstallableError struct {
	Agent model.AgentID
}

func (e AgentNotInstallableError) Error() string {
	return "agent " + string(e.Agent) + " is managed by Antigravity and cannot be auto-installed"
}

func defaultStat(path string) statResult {
	info, err := os.Stat(path)
	if err != nil {
		return statResult{err: err}
	}

	return statResult{isDir: info.IsDir()}
}
