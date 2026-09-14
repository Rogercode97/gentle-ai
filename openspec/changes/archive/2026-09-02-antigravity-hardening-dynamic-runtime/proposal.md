# Proposal: Dynamic Subagent Hardening & Refactoring for Antigravity

## 1. Objectives & Scope
This proposal expands the Antigravity subagent tool hardening mechanism to support:
- **Refactor/Clean Footprint**: Reduce the edit footprint in shared files (`adapter.go`, `inject.go`, `doctor.go`, `codegraph_contract.go`) to minimal bridge calls, encapsulating all Antigravity-specific logic in dedicated files (`antigravity_sdd_agents.go` and `antigravity_doctor.go`).
- **Full Pi Parity**: Implement full dynamic subagent integration for Antigravity matching Pi's capabilities, adding active support and prompt contracts for spawning and hardening:
  - 4R Lenses (`review-risk`, `review-readability`, `review-reliability`, `review-resilience`)
  - Judgment Day roles (`jd-judge-a`, `jd-judge-b`, `jd-fix-agent`).
- **Strict TDD Enforcement**: Keep Strict TDD constraints at the plugin level, restricting `sdd-apply` and `sdd-verify` behavior when `strict_tdd: true` is active.

## 2. Affected Packages
- `internal/agents/antigravity/`:
  - [adapter.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/agents/antigravity/adapter.go) - Bridge calls to check capabilities.
- `internal/components/sdd/`:
  - [inject.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/inject.go) - Minimized injection hooks calling `antigravity_sdd_agents.go`.
  - [antigravity_sdd_agents.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/antigravity_sdd_agents.go) - Main plugin definition, role tables, and hardening prompt generation.
  - [antigravity_sdd_agents_test.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/sdd/antigravity_sdd_agents_test.go) - Unit tests.
- `internal/components/communitytool/`:
  - [codegraph_contract.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/components/communitytool/codegraph_contract.go) - Minimized codegraph contract detection delegating to `antigravity_sdd_agents.go`.
- `internal/cli/`:
  - [doctor.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/cli/doctor.go) - Minimized doctor checks calling `antigravity_doctor.go`.
  - [antigravity_doctor.go](file:///mnt/c/Users/Cobies/Desktop/Proyectos/COLAB/gentle-ai/internal/cli/antigravity_doctor.go) - Dedicated file containing the doctor checks for Antigravity.

## 3. Rollback Plan
- Restore the original files from git and run `go test ./...` and `go vet ./...` to verify no regression.
