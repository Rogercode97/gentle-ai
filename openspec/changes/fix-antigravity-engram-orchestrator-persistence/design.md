# Design: Fix Antigravity Engram Orchestrator Persistence

## Technical Approach

Establish a dual-path persistence architecture for Google Antigravity SDD orchestration. When running SDD phases, dynamic subagents are explicitly configured with `enable_mcp_tools: true` to perform direct Engram persistence (`mem_save`). As a guaranteed safety net, the Antigravity orchestrator prompt and hardening contract mandate fallback persistence: if a subagent (e.g., static or degraded) fails to persist its artifact directly to Engram, the orchestrator inspects the child return envelope and actively calls `mem_save` under `sdd/{change-name}/{artifact-type}` before advancing the pipeline.

## Architecture Decisions

| Decision | Choice | Tradeoffs / Alternatives Considered | Rationale |
|---|---|---|---|
| **Persistence Topology** | Dual-Path (Subagent Direct + Orchestrator Fallback) | Subagent-only (fails on static agents); Orchestrator-only (inflates prompt context and loses execution context). | Maximizes subagent autonomy while guaranteeing 100% artifact durability across sessions. |
| **Deduplication & Idempotency** | Topic-Key Upsert (`sdd/{change-name}/{artifact}`) | Custom versioning or orchestrator pre-check heuristics. | Engram natively updates existing topic keys without duplicate entries or state pollution. |
| **Hardening Injection** | PreInvocation Hook String Update | Static permission manifest (unsupported in Antigravity) vs binary patching. | Aligns with existing `gentle-ai-sdd-agents` plugin design and OpenCode `permission.task` parity. |

## Data Flow

```text
 User / Orchestrator               Dynamic Subagent                  Engram MCP / Disk
        │                                  │                                 │
        ├────── 1. define_subagent ───────>│ (enable_mcp_tools: true)        │
        ├────── 2. invoke_subagent ───────>│                                 │
        │                                  ├──── 3. Direct mem_save ────────>│ [Engram Store]
        │                                  ├──── 4. Write OpenSpec file ────>│ [Filesystem]
        │<───── 5. Return Envelope ────────┤                                 │
        │       (status, artifacts, body)  │                                 │
        │                                                                    │
        ├── 6. Verify Engram Artifact ──────────────────────────────────────>│
        │      (If missing, execute Fallback mem_save)                       │
        │                                                                    │
        ├────── 7. Gatekeeper Validation & Transition ───────────────────────┤
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/assets/antigravity/sdd-orchestrator.md` | Modify | Mandate `enable_mcp_tools: true` on `define_subagent` and document orchestrator fallback `mem_save` verification. |
| `internal/components/sdd/antigravity_sdd_agents.go` | Modify | Update `antigravitySddAgentsHardeningMessage` and phrase validation table to enforce fallback persistence and MCP enablement. |
| `internal/assets/skills/_shared/sdd-phase-common.md` | Modify | Update Section C (Artifact Persistence) with orchestrator fallback persistence guarantee. |
| `internal/components/sdd/antigravity_sdd_agents_test.go` | Modify | Add unit test assertions for new hardening phrases and dual-path invariants. |

## Interfaces / Contracts

### 1. Hardening Contract Injection (`antigravity_sdd_agents.go`)

```go
// antigravitySddAgentsHardeningContractPhrases locks required substrings
var antigravitySddAgentsHardeningContractPhrases = []string{
    "sdd-explore",
    "sdd-propose",
    "sdd-apply",
    "sdd-verify",
    "review-*",
    "jd-judge-*",
    "jd-fix-agent",
    "fail closed",
    "CodeGraph",
    "Strict TDD",
    "Red phase",
    "Red-Green-Refactor",
    "review-risk",
    "review-readability",
    "review-reliability",
    "review-resilience",
    "jd-judge-a",
    "jd-judge-b",
    "define_subagent",
    "invoke_subagent",
    "enable_mcp_tools: true",
    "fallback persistence",
    "Strict phase boundaries contract",
    "Engram memory contract",
}
```

### 2. Orchestrator Dynamic Delegation & Fallback Contract (`sdd-orchestrator.md`)

```markdown
Tool scopes: Set `enable_mcp_tools: true` so phase agents can use configured MCP tools such as Engram.
Set `enable_subagent_tools: false` for all subagents, and `enable_write_tools: false` for read-only roles.

Fallback Persistence: If a subagent completes without persisting directly to Engram, the orchestrator
MUST inspect the return envelope and execute `mem_save` under `sdd/{change-name}/{artifact-type}`.
```

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | Hardening message phrases & forbids | Run `TestAntigravitySddAgentsHardeningContractPhrases` in `antigravity_sdd_agents_test.go`. |
| Unit | Plugin generation & hooks JSON | Run `TestAntigravitySddAgentsPluginWritesManifestAndHooks` and idempotency tests. |
| Integration | Orchestrator asset template validation | Run `internal/assets` test suite to assert required prompts and tool scopes in `sdd-orchestrator.md`. |

## Threat Matrix

`N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.`

## Migration / Rollout

No migration required. This change updates prompt assets, Go constants, and shared skill markdown files. Non-Antigravity adapters (Claude Code, OpenCode, Codex) are unaffected.

## Open Questions

None.
