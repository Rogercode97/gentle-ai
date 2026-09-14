# Proposal: Fix Antigravity Orchestrator Skip-to-Apply Issue

## Intent

Exempt `sdd-apply` from autonomous delegation in the Antigravity orchestrator. This ensures that the orchestrator pauses and requests explicit user permission before starting the apply (implementation) phase, regardless of whether the session is running in interactive or automatic mode.

## Scope

### In scope

- Modify the Antigravity orchestrator prompt instructions (`internal/assets/antigravity/sdd-orchestrator.md`) to explicitly exempt `sdd-apply` from the autonomous delegation rule.
- Ensure that the Antigravity orchestrator always pauses and asks the user for permission before defining or invoking `sdd-apply`.
- Update the corresponding golden fixture (`testdata/golden/sdd-antigravity-rulesmd.golden`) to match the new orchestrator rules.

### Out of scope

- Modifying the orchestrator behavior for other SDD phases (such as `sdd-explore`, `sdd-spec`, `sdd-design`, or `sdd-verify`) which are meant to execute autonomously in automatic mode.
- Changing the execution mode logic in the Go source code.
- Editing production Go files.

## Proposed changes

1. **Update `internal/assets/antigravity/sdd-orchestrator.md`:**
   - In the introductory block (line 7), update the delegation instruction to exempt `sdd-apply`.
     - *From:* `Do not ask the user for permission to start or run subagents; execute delegation autonomously.`
     - *To:* `Do not ask the user for permission to start or run subagents; execute delegation autonomously (except for sdd-apply, which is exempt from autonomous delegation and always requires explicit user permission before definition/invocation).`
   
2. **Update `testdata/golden/sdd-antigravity-rulesmd.golden`:**
   - Synchronize the golden file to reflect the exact same phrasing update for consistency and to pass tests.

## Affected internal packages and areas

| Area | Impact |
|------|--------|
| `internal/assets/antigravity` | Update `sdd-orchestrator.md` prompt instructions. |
| `testdata/golden` | Update `sdd-antigravity-rulesmd.golden` regression fixture. |

## Risks and mitigations

| Risk | Mitigation |
|------|------------|
| The orchestrator fails to pause in automatic mode despite the prompt update. | The prompt instructions already contain highly explicit rules about the `sdd-apply` hard pause gate (lines 337 and 348). Exempting `sdd-apply` in the introductory block removes the conflicting "execute delegation autonomously" instruction, reinforcing compliance. |
| Golden tests fail due to mismatched fixtures. | Ensure the golden files are updated synchronously with the asset changes. |

## Rollback plan

If this change introduces unexpected behavior or breaks tests:
1. Revert the modifications to `internal/assets/antigravity/sdd-orchestrator.md`.
2. Revert the changes to `testdata/golden/sdd-antigravity-rulesmd.golden`.
