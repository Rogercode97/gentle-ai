# Tasks: Fix Antigravity Orchestrator Skip-to-Apply Issue

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | <20 lines |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | One focused PR |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Update Orchestrator Prompt Instructions | One focused PR | `go test ./internal/components/ -run ^TestGoldenSDD_Antigravity$` | Golden-fixture assertions | Revert orchestrator asset & golden changes |

## Phase 1: Modify Prompt Instructions

- [x] 1.1 **Update Prompt Asset** — In [sdd-orchestrator.md](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/assets/antigravity/sdd-orchestrator.md#L7), update the introductory delegation rule to exempt `sdd-apply` from autonomous delegation and specify that it always requires explicit user permission.
  - *Target text*: `Do not ask the user for permission to start or run subagents; execute delegation autonomously.`
  - *Replacement text*: `Do not ask the user for permission to start or run subagents; execute delegation autonomously (except for sdd-apply, which is exempt from autonomous delegation and always requires explicit user permission before definition/invocation).`

## Phase 2: Synchronize and Test Golden Fixtures

- [x] 2.1 **Observe test failure** — Run the test suite `go test ./internal/components/ -run ^TestGoldenSDD_Antigravity$` to observe the golden mismatch test failure.
- [x] 2.2 **Update Golden Fixture** — Run `go test ./internal/components/ -run ^TestGoldenSDD_Antigravity$ -update` to automatically regenerate [sdd-antigravity-rulesmd.golden](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/testdata/golden/sdd-antigravity-rulesmd.golden) with the updated prompt instructions, or update it manually.
- [x] 2.3 **Verify Golden Test Passes** — Run `go test ./internal/components/ -run ^TestGoldenSDD_Antigravity$` again to verify it runs and passes successfully.

## Phase 3: Final Verification

- [ ] 3.1 **Run Entire Test Suite** — Run `go test ./...` and `go vet ./...` to ensure no other components are broken by the change.
- [ ] 3.2 **Inspect Git Diff** — Review `git diff --stat` to verify that modifications are limited to `internal/assets/antigravity/sdd-orchestrator.md` and `testdata/golden/sdd-antigravity-rulesmd.golden`, keeping total changes under 20 lines.
