---
name: skill-grader
description: Grade skillctl eval outputs against assertions and write grading.json. Use only when the user explicitly asks to grade. Do not edit the skill or assertions.
---

# skill-grader

Judge existing outputs. The user must have already run eval-runner (or equivalent) for this iteration.

## Steps

For each `.skillbench/<skill>/iteration-N/eval-*/{with_skill,without_skill}/`:

1. Read sibling `../eval_metadata.json` assertions (and `evals/evals.json` if needed).
2. Read `outputs/` and `transcript.md`.
3. Write `grading.json` in **that** config directory:

```json
{
  "expectations": [
    { "text": "Output contains valid JSON", "passed": true, "evidence": "outputs/result.json parsed" }
  ]
}
```

Every assertion becomes one expectation. `text`, `passed`, and `evidence` are required. Empty assertions → `"expectations": []`.

## Must not

- Invent assertions or edit `evals/evals.json` / the business skill.
- Run aggregate or report unless the user asked in the same message.
- Grade a run that has no outputs; say so and stop.
