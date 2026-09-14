# Technical Specification: Antigravity Engram Orchestrator Persistence

## Purpose

Defines formal requirement contracts, schema invariants, and test assertions for dual-path Engram persistence across Google Antigravity SDD orchestration and dynamic subagents.

---

## Requirements

### Requirement: REQ-AGY-ENGRAM-01 Dynamic Subagent MCP Tool Enablement

Dynamic subagents defined by the Antigravity SDD orchestrator MUST have MCP tools enabled (`enable_mcp_tools: true`) in their definition parameters to allow direct execution of Engram MCP tools (`mem_save`, `mem_search`, `mem_get_observation`).

#### Scenario: Dynamic subagent defined with MCP tool capability
- GIVEN the Antigravity SDD orchestrator invokes `define_subagent`
- WHEN setting tool permissions for any SDD phase (`sdd-explore`, `sdd-propose`, `sdd-spec`, `sdd-design`, `sdd-tasks`, `sdd-apply`, `sdd-verify`)
- THEN `enable_mcp_tools` is set to `true`
- AND `enable_subagent_tools` is set to `false`.

#### Scenario: Read-only phase retains MCP capability without write access
- GIVEN a read-only dynamic subagent role (`sdd-explore`, `review-*`, `jd-judge-*`)
- WHEN `define_subagent` is executed
- THEN `enable_write_tools` is set to `false`
- AND `enable_mcp_tools` is set to `true`.

---

### Requirement: REQ-AGY-ENGRAM-02 Orchestrator Engram Fallback Persistence

The Antigravity SDD orchestrator prompt (`sdd-orchestrator.md`) and pre-invocation hardening (`antigravity_sdd_agents.go`) MUST mandate that upon child subagent completion, the orchestrator inspects the return envelope and executes `mem_save` under `sdd/{change-name}/{artifact-type}` if the subagent did not persist directly to Engram.

#### Scenario: Orchestrator executes fallback save on missing artifact
- GIVEN a subagent completes and returns an envelope containing artifact content
- WHEN the orchestrator detects the artifact was not persisted to Engram
- THEN the orchestrator calls `mem_save` with title and `topic_key` set to `sdd/{change-name}/{artifact-type}`
- AND the artifact type is set to `architecture`.

#### Scenario: Orchestrator avoids duplicate save when artifact is present
- GIVEN a subagent has directly persisted its phase artifact to Engram
- WHEN the orchestrator processes the return envelope
- THEN the orchestrator verifies the artifact in Engram and advances without duplicate writes.

---

### Requirement: REQ-AGY-ENGRAM-03 Shared Convention Fallback Guarantee

Shared SDD conventions in `internal/assets/skills/_shared/sdd-phase-common.md` Section C MUST document the dual-path persistence guarantee between subagents and orchestrators.

#### Scenario: Section C documents dual-path contract
- GIVEN the shared phase documentation in `sdd-phase-common.md`
- WHEN Section C (Artifact Persistence) is rendered
- THEN it specifies that subagents MUST attempt direct persistence via `mem_save`
- AND it specifies that the orchestrator provides an active fallback persistence guarantee.

---

### Requirement: REQ-AGY-ENGRAM-04 Test & Verification Assertions

Unit tests in `internal/components/sdd/` MUST verify hardening message phrases and orchestrator prompt asset integrity.

#### Scenario: Hardening contract phrase test coverage
- GIVEN `antigravity_sdd_agents_test.go`
- WHEN `TestAntigravitySddAgentsHardeningContractPhrases` executes
- THEN all required Engram fallback persistence phrases are asserted.

#### Scenario: Orchestrator prompt asset integrity test coverage
- GIVEN prompt asset test suite in `internal/components/sdd/`
- WHEN template verification runs
- THEN `internal/assets/antigravity/sdd-orchestrator.md` is asserted to contain `enable_mcp_tools: true` and fallback persistence instructions.
