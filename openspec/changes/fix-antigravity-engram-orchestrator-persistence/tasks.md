# Tasks: Fix Antigravity Engram Orchestrator Persistence

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~55 lines (additions + deletions) |
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
| 1 | Hardening phrases, orchestrator prompts, and shared conventions for Engram fallback persistence | PR 1 | `go test -v -run TestAntigravitySddAgents ./internal/components/sdd/...` | N/A — prompt assets and contract string constants only | Revert edits in `antigravity_sdd_agents.go`, `sdd-orchestrator.md`, and `sdd-phase-common.md` |

## Phase 1: Test Scaffolding (TDD RED)

- [x] 1.1 Add failing unit test assertions for `enable_mcp_tools: true` and `fallback persistence` phrases in `internal/components/sdd/antigravity_sdd_agents_test.go`
- [x] 1.2 Add test asserting `internal/assets/antigravity/sdd-orchestrator.md` contains dynamic subagent MCP tool enablement and fallback persistence instructions

## Phase 2: Core Implementation (TDD GREEN)

- [x] 2.1 Update `antigravitySddAgentsHardeningMessage` and `antigravitySddAgentsHardeningContractPhrases` in `internal/components/sdd/antigravity_sdd_agents.go`
- [x] 2.2 Update `internal/assets/antigravity/sdd-orchestrator.md` to specify `enable_mcp_tools: true` and orchestrator `mem_save` fallback persistence instructions
- [x] 2.3 Update `internal/assets/skills/_shared/sdd-phase-common.md` Section C to document the dual-path persistence model and orchestrator fallback guarantee

## Phase 3: Verification & Polish (REFACTOR)

- [x] 3.1 Run Go test suite `go test ./internal/components/sdd/...` to ensure all tests pass cleanly
- [x] 3.2 Verify no unexpected diffs or regressions across other agent adapters
