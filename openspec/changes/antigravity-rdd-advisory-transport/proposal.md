# Proposal: Antigravity RDD Advisory Transport

## Intent

Enable Google Antigravity (`model.AgentAntigravity`) as an active, supported runtime for receipt-driven development (RDD) advisory review (`ContractReviewTransportV1`) and immutable review execution (`ContractImmutableReviewExecutorV1`), transitioning it from `ContractExposureDormant` to `ContractExposureAdvertised` via a zero-semantics headless adapter.

## Scope

### In Scope
- Implement `reviewerprovider.AntigravityAdapter` using isolated scratch execution (`agy` or `gemini` fallback, stdin prompt, raw stdout capture).
- Register `"antigravity"` in `internal/reviewerprovider/runtimes.go`.
- Advertise `ContractReviewTransportV1` and `ContractImmutableReviewExecutorV1` for `AgentAntigravity` in `capabilitymanifest`.
- Wire `AgentAntigravity` in `internal/cli/` runtime capability and direct capture resolution.
- Enforce static minimality guards and update parity/digest test suites.

### Out of Scope
- Prompt assembly, schema validation, or review decision logic inside the adapter.
- State persistence or interactive UI sessions during review transport.
- Modifications to core RDD review lifecycle state machines or receipt hashing.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `rdd-transport-capability`: Admit `model.AgentAntigravity` alongside Claude, Codex, OpenCode, and Pi for immutable receipt review transport and execution.

## Approach

Implement a minimal, fail-closed `AntigravityAdapter` in `internal/reviewerprovider/` following the existing `ClaudeAdapter`/`CodexAdapter` isolation model: execute in a temporary scratch directory, pipe the opaque prompt via stdin, capture raw bytes from stdout/file, and prevent prompt or schema leakage into adapter code. Update capability manifest exposures, CLI routing tables, and static guard registries.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/reviewerprovider/` | New / Modified | Add `antigravity_adapter.go`, update `runtimes.go` & minimality guard |
| `internal/agents/capabilitymanifest/` | Modified | Advertise review transport contracts and update canonical digests |
| `internal/cli/` | Modified | Enable `AgentAntigravity` in review transport capability & provider runtime |
| `internal/components/sdd/` | Modified | Update bounded review contract parity test clauses |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Binary resolution fails on host without `agy` | Low | Fallback resolution between `agy` and `gemini` executable names |
| Prompt or schema leakage into adapter | Low | AST static guards (`adapter_minimality_guard_test.go`) reject non-opaque tokens |
| State pollution across review invocations | Low | Execute in dedicated, ephemeral temp directory with deferred cleanup |

## Rollback Plan

Revert manifest exposures to `ContractExposureDormant` and remove `AntigravityAdapter` bindings in `internal/cli/` and `internal/reviewerprovider/`.

## Dependencies

- RDD v1 review protocol and `reviewerprovider` execution framework.
- Local `agy` or `gemini` CLI binary.

## Success Criteria

- [ ] `model.AgentAntigravity` advertises `ContractReviewTransportV1` and `ContractImmutableReviewExecutorV1`.
- [ ] `reviewerprovider.AntigravityAdapter` executes headlessly in scratch isolation and passes minimality guard tests.
- [ ] Direct capture succeeds for `gentle-ai review capture-result --agent antigravity`.
- [ ] All manifest digests and catalog parity tests pass cleanly.

## Proposal Question Round

### Proposed Questions & Assumptions
1. **Binary Name Resolution**: Should the adapter prefer `agy` with fallback to `gemini`, or strictly require `agy`? *(Assumption: Try `agy` first, fallback to `gemini` on `exec.LookPath`).*
2. **Headless Invocation Flags**: What minimal flags ensure zero interactive prompt / zero tool state? *(Assumption: Headless execution via stdin prompt matching Antigravity CLI non-interactive mode).*
3. **Failure Semantics**: Should missing binaries fail closed with typed unavailable error before review start? *(Assumption: Yes, fail closed cleanly without orphan state).*
