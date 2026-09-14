# Design: Dynamic Subagent Hardening & Refactoring for Antigravity

## 1. Architectural Decisions & Rationale

### A. Encapsulation & Refactoring (Clean Footprint)
To avoid merge conflicts and isolate changes to Antigravity, we encapsulate all Antigravity-specific logic from shared files into dedicated packages:
- **`internal/components/sdd/inject.go`**: Move `ensureAntigravitySkillRegistryHook` out of the shared `inject.go` and place it in the dedicated `antigravity_sdd_agents.go` file. Keep only a minimal bridge invocation in the core `Inject` function.
- **`internal/components/communitytool/codegraph_contract.go`**: Move paths definition and JSON verification logic for Antigravity CodeGraph wiring into `antigravity_sdd_agents.go`. The shared `codegraph_contract.go` file will only contain minimal bridge calls to `sdd.AntigravityCodeGraphToolWiringPaths` and `sdd.HasAntigravityCodeGraphToolWiring`.
- **`internal/cli/doctor.go`**: Retain only the bridge call to `checkAntigravityDynamicSubagentRuntime` located in the dedicated `antigravity_doctor.go` file.

### B. Dynamic Subagent Scoping & Prompt Contracts (Full Pi Parity)
- Explicitly register all 4R lenses (`review-risk`, `review-readability`, `review-reliability`, `review-resilience`) and Judgment Day roles (`jd-judge-a`, `jd-judge-b`, `jd-fix-agent`) in the canonical role-to-scope lookup mapping in `antigravity_sdd_agents.go`.
- Expand the PreInvocation hardening prompt template (`antigravitySddAgentsHardeningMessage`) with explicit rules and capabilities for the new lenses and Judgment Day roles to prevent scope escalation and enforce TDD boundaries.
- Allow read-only access for all 4R lenses and arbitration judges, but grant write tool permissions specifically to `jd-fix-agent` restricted to confirmed ledger entries.

## 2. Component Details

### `internal/components/sdd/antigravity_sdd_agents.go`
- `antigravitySddAgentsRoleScopes`: Canonical table containing all standard, 4R, and Judgment Day roles.
- `ensureAntigravitySkillRegistryHook`: Encapsulated hook logic migrated from `inject.go`.
- `AntigravityCodeGraphToolWiringPaths(homeDir)`: Encapsulated paths helper migrated from `codegraph_contract.go`.
- `HasAntigravityCodeGraphToolWiring(homeDir, adapter)`: Encapsulated detection helper migrated from `codegraph_contract.go`.

### `internal/components/communitytool/codegraph_contract.go`
- Delegate the Antigravity `model.AgentAntigravity` case inside `codeGraphToolWiringPaths` and `hasCodeGraphToolWiring` to the `sdd` package helper functions.
