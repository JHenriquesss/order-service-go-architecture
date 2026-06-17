# AGENTS.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

---

# Operating Protocol (read this — it governs how you finish)

You are an implementation LLM working inside **one isolated phase folder**. You can see only this folder. Treat it as a **standalone Go module**: it must build and test on its own, with its own `go.mod` and any minimal scaffolding the phase needs. Do **not** assume code from other phases exists.

## Your single job

Implement `PHASE-PLAN<n>.md` (the plan file in this folder) until the end, satisfying every gate, then send exactly one final message.

## Mandatory reading order

1. `PHASE-PLAN<n>.md` — what to build, entry/exit conditions, must-exist / must-not-exist checklists, positive/negative tests.
2. `QUALITY-GATES.md` — measurable acceptance criteria for this phase. These are **mandatory**, not advice.
3. `GO-QUALITY-GATE.md` — Go coding-quality rules scoped to this phase's code.

If two files conflict, follow the **stricter** rule and document the deviation in `PHASE-RESULT.md`.

## TDD is required

Red → green → refactor. Write the test first (it must encode the phase goal), make it pass, then refactor while keeping it green. Tests must exercise real behavior, not mirror the implementation.

## Definition of done (all required)

1. Every item in the plan's **must-exist** checklist is true.
2. Every item in the plan's **must-not-exist** checklist is false (absent).
3. Code builds: `go build ./...`.
4. Formatted: `gofmt -l .` prints nothing.
5. `go vet ./...` clean (or documented justified exception).
6. All positive and negative tests from the plan exist and pass: `go test ./...`.
7. Coverage and any other thresholds in `QUALITY-GATES.md` are met with evidence.
8. `PHASE-RESULT.md` is filled in with evidence (see its template).

"Files were created" is **not** done. You may only finish after the gates were actually run, or it is documented why a check could not run in this isolated environment (with the concrete blocker, never "tool unavailable").

## PHASE-RESULT.md

Before your final message, fill `PHASE-RESULT.md` with: what was implemented, tests added, commands run, commands passed, commands failed (+reason/impact/fix), known limitations, evidence-based quality score 0-100, and remaining work to reach 100/100. The score must be justified by evidence, not optimism.

## Final message

After `PHASE-RESULT.md` exists and the available gates have been run, send exactly — no extra words, no markdown:

I finished the implementation
