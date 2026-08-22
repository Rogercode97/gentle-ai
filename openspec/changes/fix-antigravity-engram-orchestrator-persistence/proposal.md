# Proposal: Fix Antigravity Engram Orchestrator Persistence

## Intent

Static Antigravity subagents lack direct access to Engram MCP tools (`call_mcp_tool`), causing silent persistence failures during subagent SDD phases and leaving Engram memory without phase artifacts. This change establishes a robust dual-path persistence contract ensuring dynamic subagents enable MCP tools and the orchestrator acts as an active fallback to persist artifacts to Engram.

## Scope

### In Scope
- Dual-path persistence contract in Antigravity SDD (`enable_mcp_tools: true` on `define_subagent`).
- Orchestrator prompt enforcement in `sdd-orchestrator.md` to verify and save subagent artifacts to Engram (`mem_save` under `sdd/{change-name}/{artifact-type}`) when not persisted directly.
- Hardening contract update in `antigravity_sdd_agents.go` reflecting fallback persistence requirements.
- Shared conventions update in `internal/assets/skills/_shared/sdd-phase-common.md` documenting orchestrator fallback persistence.
- Golden tests and package test assertions in `internal/components/sdd/` for new contract phrases and fallback behavior.

### Out of Scope
- Modifying non-Antigravity orchestrators (Claude, OpenCode, Cursor, etc.).
- Adding external network tools or modifying non-Engram MCP servers.
- Changes to Engram server backend code.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- `antigravity-support`: Enforce dynamic subagent MCP tool configuration (`enable_mcp_tools: true`) and orchestrator fallback artifact persistence.
- `sdd-orchestrator-assets`: Mandate Antigravity orchestrator artifact verification and Engram persistence fallback.

## Approach

1. Configure Antigravity dynamic subagent tool permissions with `enable_mcp_tools: true` in `sdd-orchestrator.md` and hardening contract strings in `antigravity_sdd_agents.go`.
2. Update Antigravity orchestrator prompt instructions so that after receiving a subagent result, the orchestrator checks if the artifact exists in Engram and actively calls `mem_save` if missing.
3. Update `sdd-phase-common.md` Section C to document the dual-path persistence model and orchestrator fallback guarantee.
4. Update unit tests in `antigravity_sdd_agents_test.go` and orchestrator asset tests to validate contract phrases and idempotency.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/assets/antigravity/sdd-orchestrator.md` | Modified | Add dynamic subagent MCP configuration and orchestrator fallback persistence instructions. |
| `internal/components/sdd/antigravity_sdd_agents.go` | Modified | Update `antigravitySddAgentsHardeningMessage` with dual-path persistence phrases. |
| `internal/assets/skills/_shared/sdd-phase-common.md` | Modified | Document orchestrator fallback persistence guarantee across SDD phases. |
| `internal/components/sdd/antigravity_sdd_agents_test.go` | Modified | Update contract phrase tests and assertions. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Duplicate `mem_save` calls overwriting artifacts | Low | `topic_key` ensures idempotent upserts in Engram. |
| Hardening contract phrase drift in tests | Low | Update test phrase tables to lock exact wording. |

## Rollback Plan

Revert prompt changes in `sdd-orchestrator.md`, `antigravity_sdd_agents.go`, `sdd-phase-common.md`, and test assertions via Git.

## Dependencies

- Engram MCP server tools (`mem_save`, `mem_search`, `mem_get_observation`).

## Success Criteria

- [ ] Dynamic subagents are defined with `enable_mcp_tools: true`.
- [ ] Orchestrator verifies and persists missing subagent artifacts to Engram on phase completion.
- [ ] `sdd-phase-common.md` documents the fallback persistence guarantee.
- [ ] All `internal/components/sdd` unit tests pass.
