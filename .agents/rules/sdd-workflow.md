# SDD Workflow & Enforcement Rules

This workspace enforces strict adherence to the Software Design & Development (SDD) lifecycle. No agent or subagent is permitted to bypass these rules.

## 1. Absolute Stop: No Auto-Apply
- **NEVER** apply code changes or write to files without explicit user confirmation first.
- The orchestrator must present the proposed implementation plan, architecture, or task list and pause, waiting for the user to approve before delegating to `sdd-apply`. **This is a hard pause gate**: you **MUST NOT** call `define_subagent` or `invoke_subagent` for `sdd-apply` in the same turn that you ask for approval; you **MUST** end your turn and wait for the user's explicit response in the chat confirming they want to proceed.
- The orchestrator must detail proposals, specs, tasks, and reports directly in the chat/conversation, and not just reference the created `.md` artifacts.
- The orchestrator must yield execution and rely on reactive wakeup for the completion of ALL dynamic subagents (including planning, apply, verification, and review/judging subagents) without active polling before proceeding to subsequent steps or responding to the user.
- The orchestrator and any interactive tools must present all questions, prompts, and options to the user in the conversation's active language (matching the user's current language) rather than English.

## 2. Mandatory SDD Phase Boundaries
Every code change, refactoring, or new feature request must run through the following pipeline:
1. **Exploration & Planning (`sdd-explore` / `sdd-propose` / `sdd-tasks`)**: Map the affected files, plan the approach, and write out the tasks.
2. **Apply (`sdd-apply`)**: Edit the code using the approved plan.
3. **Verify (`sdd-verify` with TDD)**: Write tests first (or concurrently), run the test suite, and verify all functionality. 
4. **Review (4R & Judgment Day)**: Run a code review on the changes.
   - For standard changes, select and run at least one specific 4R lens (`review-readability`, `review-reliability`, `review-resilience`, `review-risk`).
   - For hot paths or diffs > 400 lines, run the full 4R review and Judgment Day (blind dual review).

## 3. Test-Driven Development (TDD) Requirement
- All new features and bug fixes must have corresponding tests.
- Execution of tests is mandatory inside `sdd-verify` before completing a task.
