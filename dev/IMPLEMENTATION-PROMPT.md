# The single prompt for the implementation LLM

Give the implementation LLM access to **one** phase folder only, then send exactly this, replacing `(number)` with the phase number:

```
Implement the PHASE-PLAN(number).md following the rules of your AGENTS.md until the end and send me a message: (I finished the implementation) at the end.
```

Nothing else. The folder's `AGENTS.md` tells it to read `PHASE-PLAN<n>.md`, `QUALITY-GATES.md`, and `GO-QUALITY-GATE.md`, to work TDD, to fill `PHASE-RESULT.md` with evidence, and to end with exactly: `I finished the implementation`.
