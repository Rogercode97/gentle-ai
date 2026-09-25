---
session_summary:
  project: gentle-ai
  engram_sync: false
  reason: "Engram unavailable: mem_save and mem_session_summary return 'session has already ended'; mem_session_start returns 'Engram project detection is unavailable'."
  date: 2026-09-25
  branch: termux-min
  head: 9555395d
---

# Session Handoff — 2026-09-25

## Goal

Fix the `gentle-ai` install failure on Termux and move `termux-min` onto the current
`upstream/main`.

## Root causes found

1. **Install failed with a misleading `ENOENT`.** npm-installed CLIs keep the upstream
   shebang `#!/usr/bin/env node`, and Termux has no `/usr/bin/env`. `os/exec` in Go issues
   the `execve(2)` syscall directly instead of calling libc `execve`, so Termux's
   `libtermux-exec` preload never rewrites it. Interactive shells work because bash goes
   through libc.
2. **`sync-gentle-main` used the wrong git operation.** Upstream rewrote its history, so
   `git merge upstream/main` reconciles two histories with no common ancestor: 275 conflicts.
   The local Termux layer is only 8 commits, so `git rebase --onto upstream/main <base>`
   applies it with 0 conflicts.
3. **`sync-gentle-main` fast-path lied.** It printed `harnesses_synced: true` without ever
   running `gentle-ai sync`, so `installed_binary_version` in `~/.gentle-ai/state.json` kept
   the old stamp and `doctor` never cleared `installed:asset_version`.
4. **Version labels were meaningless.** Upstream publishes `v3.x` tags from its abandoned
   history while the live branch runs a `v1.x` line on a `/v3` module path. Go rejects a tag
   whose major does not match the module path, so a plain local build stamps `v3.0.0-<ts>-<sha>`.

## What changed

| Area | Change | Evidence |
|---|---|---|
| Repo | `internal/system/shebang_command.go` + tests; wired into `run.go:executeCommand`, `deps.go:detectSingleDep`, `detect.go:detectNpmWritable` | commits `f39c6844`, `9555395d` |
| Branch | `termux-min` = `upstream/main` `6c7f162f` + 9 local commits, 0 behind | `git rev-list --left-right --count upstream/main...HEAD` |
| Rollback | `backup/termux-min-pre-rebase-20260925` = `4a215dc8` | `git log -1 backup/…` |
| Binary | `v1.0.8-3740-g9555395d`, built staged + `mv -f`, `-ldflags -X main.version=$(git describe --tags --always)` | `gentle-ai --version` |
| Script | `~/.local/bin/sync-gentle-main`: rebase-only, rollback ref, dirty-tree guard, asset-staleness detection, honest ledger | `bash -n`, `--check` |
| Skill | `~/.agents/skills/sync-gentle-main/SKILL.md` rewritten to the real flow (10 invariants, the 8-commit layer, `doctor` as the gate) | 209 lines |
| Lean retired | injection step removed, `~/.gemini/GEMINI.md` restored to the upstream template, 3 files deleted, `g-sync` alias repointed, backup in `~/.gemini/lean-backup/` | `doctor` healthy |
| Pi state | duplicate `npm:gentle-engram@0.1.15` entry removed | 12 → 11 packages, JSON valid |

## Verification

- `gentle-ai install --agents pi` → `exit=0`, 19/19 steps in the interactive TUI, all five
  pi packages installed. Proven with `pi`'s shebang restored to the broken upstream value
  and no local patch.
- `gentle-ai doctor` → `8 passed, 0 failed, 0 warnings`, status `healthy`.
- `go build ./...` green, `gofmt`/`go vet` clean, `internal/system` suite green.

## Known non-regressions

`internal/cli` cannot complete on this device: it hits Go's 10m test timeout inside
review-authority lifecycle tests (`runLegacyFacadeStartForTest` → `canonicalLocatorDirectory`
→ `EvalSymlinks`). Use `-run` filters. Two failures are pre-existing and reproduce identically
on the pre-change baseline `f797c139`:

- `TestRunInstallEngramForPiTargetsConfiguredAgentDirectory` — asserts the real `~/.pi` does
  not exist, false on this machine.
- `TestRunInstallGGALinuxIncludesTempCleanupBeforeClone` — a Linux case executed on Android.

## The Termux layer (8 commits, do not drop)

`IsTermux()` + android OS, platform detection that does not assume GNU Linux, `os.TempDir`
instead of `/tmp`, `npm_config_force` for Android, native auto-updater disabled, its test,
missing Pi MCP adapter tolerance, and POSIX shebang resolution. `upstream/main` contains no
`IsTermux` reference: Android support is entirely the fork's.

## Routine

```bash
g-sync                                     # update (rebase layer + build + sync)
gentle-ai doctor                           # gate: healthy
git rev-list --count HEAD..upstream/main   # 0 = up to date
```

Never trust release numbers. Never run `go install …gentle-ai/cmd/gentle-ai@latest` against
`~/.local/bin/gentle-ai`: it overwrites the Termux build with an upstream one that has no
Android support (the `gentle-update` alias in `~/.bashrc` still does exactly that).

## Pending

1. ~~Remove or repoint the `gentle-update` alias in `~/.bashrc`.~~ Done: the alias and its
   `gentle-ai-update` mirror are gone, replaced by a comment explaining why they are an
   antipattern. `g-sync` / `gentle-sync` are the only update entry points. Backup at
   `~/.bashrc.bak-20260925-leanalias`.
2. Run `/reload` so the edited skill is active in the session.
3. Optional: note in `docs/platforms.md` that Android support comes from the fork, not upstream.
4. Optional: `git push -u rogercode termux-min` (still local only).
5. Known vestige: `review-prompt-parity/scripts/verify_prompt_parity.py` keeps a `--lean`
   default pointing at the deleted template; it only fails if invoked without `--lean`.
6. The 3 lean deletions were committed in `~/skills-sovereign` as `d5e40b0`. That repo still
   carries an unrelated uncommitted change in `.agents/handoff.md` from another work session.
