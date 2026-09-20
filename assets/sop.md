## skillctl workflow

Use `skillctl` for Agent Skill lifecycle in this repo. **Each step requires an explicit user trigger.** Do not auto-chain create → eval → grade.

1. Author: `skillctl create <name>` then edit `SKILL.md` (or follow **skill-author**).
2. Validate anytime: `skillctl validate <path>`.
3. User edits `evals/evals.json` (AI may draft; the user owns the file).
4. `skillctl eval prepare <skill> [--iteration N]`.
5. User triggers **eval-runner** to write `outputs/` and `transcript.md` (with_skill and without_skill).
6. User triggers **skill-grader** to write `grading.json`.
7. `skillctl eval aggregate <skill>` then `skillctl eval report <skill>` (static HTML, not an agent).

Paths: skill directories contain `SKILL.md`; bench data is `.skillbench/<skill>/iteration-N/`. Deep instructions live in the meta-skills, not here.

Reserved: `skillctl eval trigger` is not implemented in v1.
