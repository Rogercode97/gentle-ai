package cli

import (
	"fmt"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v2/internal/antigravityhooks"
	"github.com/gentleman-programming/gentle-ai/v2/internal/doctor"
)

// antigravityExpectedHookNames are the named hooks gentle-ai installs for the
// Antigravity runtime, listed so the doctor can report a hook that exists but is
// switched off.
//
// agy loads a named hook whose spec carries "enabled": false and then runs none
// of its handlers. From the outside that is indistinguishable from a missing
// hook — which is exactly how the SDD/Review/JD hardening contract sat inert for
// ten days after someone (or something) turned it off.
var antigravityExpectedHookNames = []string{
	"gentle-ai-engram-tools",
	"gentle-ai-codegraph-first",
	"gentle-ai-sdd-agents-hardening",
}

// checkAntigravityHooksIntegrity validates the hooks surfaces agy actually
// consumes: the root hooks.json, every plugin hooks.json, and the hook scripts.
//
// It exists because both failure modes are silent. agy rejects a hooks.json file
// whole when its JSON does not match its schema and only records that in its own
// log — 1067 such rejections accumulated before anyone noticed, with every hook
// in the file inert. A hook script whose shebang interpreter does not exist
// cannot run either, and agy's executor does not have Termux's termux-exec shim
// that would otherwise rewrite /usr/bin/env.
//
// Schema and shebang problems are reported as FAIL because the affected hooks
// cannot work at all. A disabled hook is reported as WARN, because that may be a
// deliberate operator choice.
func checkAntigravityHooksIntegrity(homeDir string) CheckResult {
	configDir := antigravityActiveConfigDir(homeDir)
	findings := antigravityhooks.ScanConfigDir(configDir, antigravityExpectedHookNames)

	result := CheckResult{
		Name:   "antigravity:hooks-integrity",
		Status: CheckStatusPass,
		Detail: fmt.Sprintf(
			"hooks.json surfaces under %s match the schema agy requires, every expected hook is enabled, and hook scripts are executable with a resolvable shebang.",
			configDir),
	}
	if len(findings) == 0 {
		return result
	}

	blocking := 0
	details := make([]string, 0, len(findings))
	for _, finding := range findings {
		if finding.Blocking() {
			blocking++
		}
		details = append(details, finding.String())
	}

	// Bound the report: a single broken file yields one finding per hook, and
	// the operator needs to be sent to the file, not read every row.
	const maxShown = 5
	if len(details) > maxShown {
		details = append(details[:maxShown], fmt.Sprintf("and %d more", len(details)-maxShown))
	}

	result.Status = CheckStatusWarn
	if blocking > 0 {
		result.Status = CheckStatusFail
	}
	result.Detail = fmt.Sprintf("%d problem(s) found: %s", len(findings), strings.Join(details, "; "))
	result.Remedy = doctor.NewRemedy(doctor.RemedySync, fmt.Sprintf(
		"Run `gentle-ai sync` to rewrite %s/plugins/*/hooks.json, then review the remaining findings by hand. "+
			"A hooks.json whose top level is not an object of NAMED hooks is rejected whole by the Antigravity runtime, so every hook in it is inert; "+
			"a hook script whose shebang interpreter does not exist on this machine cannot start at all, and the runtime's hook executor runs without the termux-exec shim that rewrites /usr/bin/env.",
		configDir))
	return result
}
