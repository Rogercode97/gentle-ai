# Tasks: Antigravity RDD Advisory Transport

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~180-250 lines |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Headless scratch adapter and runtime identity | PR 1 | `go test ./internal/reviewerprovider/...` | N/A (unit test mocks exec) | `internal/reviewerprovider/antigravity_adapter*` |
| 2 | CLI routing, capability manifest, and test matrix | PR 1 | `go test ./internal/cli/... ./internal/agents/... ./internal/components/sdd/...` | `gentle-ai review status --agent antigravity` | Manifest exposures and CLI dispatch |

## Phase 1: Reviewer Provider Adapter & Identity

- [x] 1.1 Write failing unit tests in `internal/reviewerprovider/antigravity_adapter_test.go` covering `agy` resolution, `gemini` fallback, missing binary error, stdin piping, and scratch cleanup.
- [x] 1.2 Implement `reviewerprovider.AntigravityAdapter` in `internal/reviewerprovider/antigravity_adapter.go` with dual binary lookup, temp directory isolation, and raw stdout capture.
- [x] 1.3 Add `"antigravity"` to `registeredRuntimeIdentities` in `internal/reviewerprovider/runtimes.go`.
- [x] 1.4 Register `antigravity_adapter.go` in `reviewerAdapterImplementations` in `internal/reviewerprovider/adapter_minimality_guard_test.go`.

## Phase 2: CLI Routing & Direct Capture Dispatch

- [x] 2.1 Define `reviewImmutableTransportAntigravityPromptCarried` and wire `model.AgentAntigravity` in `internal/cli/review_transport_capability.go`.
- [x] 2.2 Wire `model.AgentAntigravity` in `reviewProviderAdapterFor` and `reviewProviderCaptureRuntime` in `internal/cli/review_provider_runtime.go`.

## Phase 3: Capability Manifest & Guardrails

- [x] 3.1 Advertise `ContractReviewTransportV1` and `ContractImmutableReviewExecutorV1` for `AgentAntigravity` in `internal/agents/capabilitymanifest/manifest.go`.
- [x] 3.2 Add `filepath.Join("antigravity")` to `adapterForbiddenConstructionPackageDirs` in `internal/agents/adapter_forbidden_construction_guard_test.go`.

## Phase 4: Test Matrix Updates & Full Verification

- [x] 4.1 Update manifest digests, advertised count (5), and assertions in `internal/agents/capabilitymanifest/manifest_test.go`.
- [x] 4.2 Update runtime capability matrices, advertised count (5), and refusal expectations in `internal/cli/review_transport_capability_test.go`.
- [x] 4.3 Update `expectedReviewLifecycleRuntime`, advertised count (5), and runtime assertions in `internal/components/sdd/bounded_review_contract_test.go` and `internal/components/sdd/review_runtime_identity_test.go`.
- [x] 4.4 Run full test verification suite (`go test ./...`) across all packages to ensure zero regressions and 100% clean test execution.
