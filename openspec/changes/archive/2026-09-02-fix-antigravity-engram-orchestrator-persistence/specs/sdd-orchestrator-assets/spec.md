# Delta for sdd-orchestrator-assets

## ADDED Requirements

### Requirement: Antigravity orchestrator dual-path Engram persistence

The Antigravity orchestrator asset (`internal/assets/antigravity/sdd-orchestrator.md`) MUST instruct setting `enable_mcp_tools: true` when defining dynamic subagents, and MUST mandate that the orchestrator inspects the return envelope and executes `mem_save` for any missing phase artifact under `sdd/{change-name}/{artifact-type}`.

#### Scenario: Orchestrator asset mandates MCP tools enablement

- GIVEN the Antigravity orchestrator template is rendered
- WHEN dynamic delegation instructions are inspected
- THEN `enable_mcp_tools: true` is explicitly specified for dynamic subagents.

#### Scenario: Orchestrator asset mandates fallback save

- GIVEN a subagent finishes and returns an envelope
- WHEN the orchestrator processes the result
- THEN guidance mandates calling `mem_save` if the artifact was not persisted by the child.

### Requirement: Antigravity prompt asset integrity assertions

Static unit tests in `internal/components/sdd/` MUST assert that `internal/assets/antigravity/sdd-orchestrator.md` contains dual-path persistence and `enable_mcp_tools: true` instructions.

#### Scenario: Orchestrator template asset assertions pass

- GIVEN unit tests verifying SDD prompt assets execute
- WHEN assertions evaluate `sdd-orchestrator.md` content
- THEN the presence of `enable_mcp_tools: true` and fallback persistence instructions is verified.
