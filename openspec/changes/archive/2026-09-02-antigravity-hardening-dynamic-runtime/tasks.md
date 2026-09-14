# Tasks: Antigravity Dynamic Runtime Hardening

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 100–150 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Refactor footprint & subagent parity | PR 1 | `go test ./internal/...` | N/A | [antigravity_sdd_agents.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/antigravity_sdd_agents.go) |

## Phase 1: Bridge & Setup
- [x] 1.1 RED: Add tests asserting that [codegraph_contract.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/communitytool/codegraph_contract.go) calls helper functions in `sdd` rather than hardcoding paths.
- [x] 1.2 RED: Add tests asserting all 4R lenses (`review-risk`, `review-readability`, `review-reliability`, `review-resilience`) and Judgment Day roles are registered in the role-to-scope table.
- [x] 1.3 RED: Update [antigravity_sdd_agents_test.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/antigravity_sdd_agents_test.go) to assert prompt contains specific hardening keywords for new roles.

## Phase 2: Refactoring Shared Files
- [x] 2.1 REFACTOR: Move `ensureAntigravitySkillRegistryHook` from [inject.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/inject.go) to [antigravity_sdd_agents.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/antigravity_sdd_agents.go) and replace with a minimal delegate call.
- [x] 2.2 REFACTOR: Extract path resolution and JSON validation logic from [codegraph_contract.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/communitytool/codegraph_contract.go) to [antigravity_sdd_agents.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/antigravity_sdd_agents.go) and replace with bridge calls.
- [x] 2.3 REFACTOR: Clean [adapter.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/agents/antigravity/adapter.go) and [doctor.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/cli/doctor.go) to keep only minimal hooks for Antigravity-specific subagent capabilities.

## Phase 3: Dynamic Parity Implementation
- [x] 3.1 GREEN: Add 4R Lenses and Judgment Day roles registration to `antigravitySddAgentsRoleScopes` in [antigravity_sdd_agents.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/antigravity_sdd_agents.go).
- [x] 3.2 GREEN: Update the PreInvocation message `antigravitySddAgentsHardeningMessage` with instructions for 4R lenses, Judgement Day roles, and Strict TDD boundaries.
- [x] 3.3 GREEN: Implement the runtime checks/delegations inside [antigravity_sdd_agents.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/antigravity_sdd_agents.go) matching the expected capabilities of Pi.

## Phase 4: Verification
- [x] 4.1 Verify formatting: Run `gofmt -s -w` on all edited Go files.
- [x] 4.2 Run component tests: Execute `go test -v ./internal/components/sdd/...` and verify green.
- [x] 4.3 Workspace verification: Execute `go test ./...` and `go vet ./...` to guarantee no regressions in other agents.
