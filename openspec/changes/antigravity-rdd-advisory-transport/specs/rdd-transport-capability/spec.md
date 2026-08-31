# Delta for RDD Transport Capability

## ADDED Requirements

### Requirement: Antigravity Headless Scratch Review Transport

The system MUST provide an `AntigravityAdapter` implementing `reviewerprovider.Adapter`. The adapter MUST resolve the executable by prioritizing `agy` and falling back to `gemini` via `exec.LookPath`. It MUST execute within an ephemeral scratch directory (`gentle-ai-antigravity-reviewer-*`) deleted upon completion, pipe `invocation.Prompt()` over stdin, and capture raw stdout bytes without parsing prompt semantics, JSON tokens, or schemas.

#### Scenario: Headless Antigravity execution via agy binary
- GIVEN `agy` is available in PATH and an opaque review invocation is supplied
- WHEN `AntigravityAdapter.Review` executes
- THEN an isolated scratch directory is created
- AND the subprocess runs with prompt piped to stdin and returns raw stdout bytes

#### Scenario: Binary resolution fallback to gemini
- GIVEN `agy` is missing from PATH but `gemini` is present
- WHEN `AntigravityAdapter.Review` resolves binary
- THEN `gemini` is selected and executed successfully

#### Scenario: Missing binary fails closed
- GIVEN neither `agy` nor `gemini` is available in PATH
- WHEN `AntigravityAdapter.Review` executes
- THEN it returns a typed unavailable transport error and removes the scratch directory

### Requirement: Antigravity Capability Manifest Advertisement

`model.AgentAntigravity` MUST advertise `ContractReviewTransportV1` and `ContractImmutableReviewExecutorV1` with `ContractExposureAdvertised` in `internal/agents/capabilitymanifest/`.

#### Scenario: Capability manifest verification
- GIVEN `capabilitymanifest.ForAgent(model.AgentAntigravity)` is called
- WHEN advertised contracts are evaluated
- THEN `Advertises(ContractReviewTransportV1)` and `Advertises(ContractImmutableReviewExecutorV1)` both return true

### Requirement: CLI Review Provider Capture and Direct Dispatch

`internal/cli/review_transport_capability.go` MUST mark `model.AgentAntigravity` as eligible with transport `reviewImmutableTransportAntigravityPromptCarried`, and `internal/cli/review_provider_runtime.go` MUST register `model.AgentAntigravity` in `reviewProviderCaptureRuntime` and dispatch to `reviewerprovider.NewAntigravityAdapter()`.

#### Scenario: Direct capture dispatches to Antigravity adapter
- GIVEN `gentle-ai review capture-result --agent antigravity` is executed
- WHEN review provider adapter is resolved
- THEN `NewAntigravityAdapter()` is instantiated and executes review transport

### Requirement: Antigravity Static Minimality and Parity Guards

`AntigravityAdapter` MUST be classified in `reviewerAdapterImplementations` in `adapter_minimality_guard_test.go` and pass static AST checks without importing provider review semantics, measuring prompt length, or executing loops.

#### Scenario: Minimality guard passes
- GIVEN `antigravity_adapter.go` is inspected by `TestAdapterMinimalityGuard`
- WHEN AST inspection runs
- THEN zero violations are detected

## MODIFIED Requirements

### Requirement: Adapter Declares, Provider Fails Closed, No Probing

Each adapter MUST declare its own capability. The provider MUST NOT probe or infer adapter capability. The provider MUST fail closed when a declaration is absent or unrecognized.
(Previously: Enumerated only opencode, claude, and Pi in adapter examples)

#### Scenario: Adapter self-declares capability
- GIVEN a specific adapter (opencode, claude, codex, pi, or antigravity)
- WHEN it initiates a review-capable interaction
- THEN it declares its own transport capability explicitly
- AND the provider does not attempt to detect capability independently

#### Scenario: Absent or unrecognized declaration fails closed
- GIVEN a declaration is missing or does not match a known capability value
- WHEN the provider evaluates it
- THEN the provider treats the transport as unsupported
- AND no review state is created

### Requirement: Per-Adapter Unavailable Mode, Never Unsafe Fallback

On unsupported or absent capability, the adapter MUST enter a per-adapter unavailable mode. The adapter MUST NOT construct its own flags, revisions, targets, or bindings, and MUST NOT silently fall back to legacy/unsafe behavior. (Issue #1385)
(Previously: Listed only opencode and claude for in-repo capable adapters)

#### Scenario: Pi adapter without capability enters unavailable mode
- GIVEN the Pi adapter still uses the old, non-opaque shape and has not declared capability
- WHEN it attempts a review interaction
- THEN it enters unavailable mode
- AND it does not self-construct a transition, flag, or binding of any kind

#### Scenario: Capable in-repo adapter executes only opaque transitions
- GIVEN the opencode, claude, codex, or antigravity adapter declares supported capability
- WHEN it participates in a review
- THEN it executes only provider-issued opaque transitions
- AND it constructs no flag, revision, target, or binding itself
