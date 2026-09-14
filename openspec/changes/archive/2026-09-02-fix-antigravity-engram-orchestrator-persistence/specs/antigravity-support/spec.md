# Delta for antigravity-support

## MODIFIED Requirements

### Requirement: Antigravity uses dynamic subagents

The Antigravity orchestrator MUST use runtime dynamic subagent tools rather than static subagent files. When running on a low-tier model, the system MUST enforce dynamic subagent delegation and MUST NOT execute SDD phases (such as explore, propose, spec, design, tasks, apply, verify) inline. Dynamic subagent definitions MUST explicitly configure `enable_mcp_tools: true` so subagents can access MCP servers.

(Previously: Dynamic subagent definitions did not specify explicit MCP tool enablement.)

#### Scenario: SDD orchestration runs in Antigravity

- GIVEN the Antigravity SDD orchestrator is installed
- WHEN an SDD phase requires a subagent
- THEN the prompt instructs Antigravity to call `define_subagent` with `enable_mcp_tools: true`
- AND then call `invoke_subagent`.

#### Scenario: Low-model dynamic subagent enforcement

- GIVEN a low-tier model is active in the `antigravity` agent CLI
- WHEN the orchestrator compiles system instructions
- THEN the prompt MUST include explicit instructions warning the orchestrator to call `define_subagent` and `invoke_subagent` for each phase (`sdd-explore`, `sdd-propose`, `sdd-spec`, `sdd-design`, `sdd-tasks`, `sdd-apply`, `sdd-verify`)
- AND the prompt MUST explicitly forbid inline phase execution.

## ADDED Requirements

### Requirement: Orchestrator Engram Fallback Persistence

Upon receiving a subagent return envelope, the orchestrator MUST verify whether the phase artifact is persisted in Engram under `sdd/{change-name}/{artifact-type}`. If missing, the orchestrator MUST execute `mem_save` to persist the artifact before proceeding.

#### Scenario: Subagent artifact persisted by orchestrator fallback

- GIVEN a dynamic subagent returns an artifact without saving to Engram
- WHEN the orchestrator receives the child envelope
- THEN the orchestrator executes `mem_save` under `sdd/{change-name}/{artifact-type}`
- AND the artifact is saved before the next phase begins.

#### Scenario: Subagent direct persistence verified

- GIVEN a dynamic subagent already saved its artifact to Engram
- WHEN the orchestrator receives the child envelope
- THEN the orchestrator verifies the artifact in Engram and avoids duplicate writes.

### Requirement: Hardening Contract Persistence and Test Assertions

The hardening message in `antigravity_sdd_agents.go` and its unit tests MUST assert dual-path Engram persistence and `enable_mcp_tools: true` contract phrases.

#### Scenario: Hardening contract phrase validation

- GIVEN `antigravity_sdd_agents_test.go` executes
- WHEN hardening contract phrase tests run
- THEN all dual-path Engram persistence phrases are verified.
