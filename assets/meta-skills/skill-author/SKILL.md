---
name: skill-author
description: Interview the user, then scaffold and write an Agent Skill with skillctl. Use when the user wants to create or substantially rewrite a skill for Cursor or VS Code Copilot. Do not auto-run evals.
---

# skill-author

You help a human author an Agent Skill. **Do not** run the eval chain unless the user explicitly asks for a later step.

## Steps

1. Interview: what the skill does, when it should trigger, expected outputs, edge cases.
2. Confirm the kebab-case `name`.
3. Run `skillctl create <name>` from the project root (or the directory the user named).
4. Write `SKILL.md`: pushy description (what + when), imperative body, progressive disclosure into `references/` if long.
5. Optionally draft prompts and assertions into `evals/evals.json`. The user owns that file; do not treat drafts as approved.
6. Run `skillctl validate <path>`. Fix failures.

## Must not

- Auto-run `eval prepare`, eval-runner, grader, aggregate, or report.
- Call model APIs or Claude Code (`claude -p`).
- Overwrite an existing skill directory; ask the user instead.
