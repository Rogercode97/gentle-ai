# Termux Shebang-Aware Command Execution

## Objective

Make every managed command invocation able to launch npm-installed CLI programs on Termux/Android, where the kernel cannot resolve the upstream `#!/usr/bin/env <interpreter>` shebang.

## Problem

`go install`-managed binaries exec their children with `os/exec`, which issues the `execve(2)` syscall directly instead of calling the libc `execve` that Termux's `libtermux-exec-ld-preload.so` interposes. Programs installed by npm keep their upstream shebang `#!/usr/bin/env node`, and Termux has no `/usr/bin/env`, so the kernel returns `ENOENT` and the command fails with a misleading `fork/exec <path>: no such file or directory` even though the file exists and is executable from an interactive shell.

Observed on device: `gentle-ai install --agents pi` failed at the `agent:pi` step with `run command "pi install npm:gentle-pi": fork/exec /data/data/com.termux/files/usr/bin/pi: no such file or directory`, while the same `pi install npm:gentle-pi` executed from a Termux shell returned exit 0. Termux-built programs (`npm`, `node`) carry an absolute shebang and were unaffected, which is why the dependency probe still reported versions.

## Why

The current workaround is to rewrite the first line of an installed package entrypoint, which npm overwrites on every reinstall or `pi update`, so the failure returns with the next package refresh instead of being fixed. A machine that cannot run the agent it just installed is a hard stop, not a degraded path.

## Scope

- Add one shared Termux-aware command constructor in `internal/system` that resolves a POSIX shebang and rewrites the invocation to `interpreter [interpreter args] <script> <args...>`.
- Resolve the interpreter against `PATH` when the shebang path is absolute but absent, which covers `/usr/bin/env`, `/bin/sh`, and `/usr/bin/env -S` forms.
- Route the pipeline command runner, the dependency version probe, and the npm-prefix probe through it.
- Leave every non-Termux platform on plain `exec.Command` semantics.

## Constraints

- Allowed edit surfaces: `internal/system/shebang_command.go`, `internal/system/shebang_command_test.go`, `internal/system/deps.go`, `internal/system/detect.go`, `internal/cli/run.go`, and this task document.
- No behavior change off Termux; the resolver must be a no-op there.
- An unresolvable shebang must fall back to the original invocation so the kernel keeps ownership of the error.
- No shell string interpolation: the interpreter argv is built as a slice, never concatenated into a command line.
- Technical artifacts remain in English.
- No commit, push, or pull request without separate authorization.

## Tasks

- [x] T1 Add the Termux shebang-aware command constructor with focused unit tests.
- [x] T2 Route the pipeline runner, dependency probe, and npm-prefix probe through it.
- [x] T3 Verify with `go build ./...`, the `internal/system` and `internal/cli` suites, and an on-device install run.

## Evidence

- Commit `f39c6844` on `termux-min`, on top of the rebased layer `f797c139` (`upstream/main` `6c7f162f` + 7 Termux patches, 0 behind / 8 ahead).
- `go build ./...` green; `gofmt -l` and `go vet ./internal/system/ ./internal/cli/` clean.
- `go test -count=1 ./internal/system/` green, including the new shebang table tests.
- `internal/cli` cannot complete on this device: the package hits the 10m Go test timeout inside review-authority lifecycle tests (`runLegacyFacadeStartForTest` → `canonicalLocatorDirectory` → `EvalSymlinks`). The filtered subset covering the changed path reports only two failures, both reproduced identically on the pre-change baseline `f797c139`: `TestRunInstallEngramForPiTargetsConfiguredAgentDirectory` and `TestRunInstallGGALinuxIncludesTempCleanupBeforeClone`. The first asserts the real `~/.pi` does not exist, which is false on this machine; the second is a Linux-specific gga temp-cleanup case. Neither is a regression.
- On-device proof, with `pi`'s shebang restored to the broken upstream `#!/usr/bin/env node` and no local patch: `gentle-ai install --agents pi` → `exit=0`, `You're ready. Start building.`, all five packages installed and registered (`gentle-pi`, `gentle-engram`, `pi-mcp-adapter`, `pi-web-access`, `pi-btw`). Before the fix the same command failed with `fork/exec /data/data/com.termux/files/usr/bin/pi: no such file or directory`.
