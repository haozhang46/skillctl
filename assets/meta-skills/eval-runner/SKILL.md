---
name: eval-runner
description: Execute skillctl eval prepare packs in the IDE. Use only when the user explicitly asks to run an eval iteration. Write outputs and transcript.md. Do not grade or aggregate.
---

# eval-runner

Run **one** eval iteration the user named. Each step requires an explicit user trigger; do not chain into grading.

## Preconditions

- `skillctl eval prepare <skill> [--iteration N]` has already been run, **or** run it now if the user asked you to prepare and run together **in this message**.
- Bench layout: `.skillbench/<skill>/iteration-N/eval-<id>-<slug>/{with_skill,without_skill}/`

## Steps

For each eval directory in that iteration:

1. Read `eval_metadata.json` and `with_skill/PROMPT.md`. Execute the task **with** the skill. Write artifacts to `with_skill/outputs/` and a `with_skill/transcript.md`.
2. Read `without_skill/PROMPT.md`. Execute the **same** task without loading the skill. Write `without_skill/outputs/` and `without_skill/transcript.md`.
3. Prefer launching with-skill and without-skill work in the same turn so they finish together.

## Must not

- Write `grading.json`.
- Run `skillctl eval aggregate` or `eval report`.
- Edit the business skill or `evals/evals.json`.
