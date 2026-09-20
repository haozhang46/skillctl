# skillctl Design

Date: 2026-09-20  
Status: Draft for implementation planning  
Repo: standalone (`skillctl`), not inside `formily-ai-form`

## Problem

Anthropic’s [skill-creator](https://github.com/anthropics/skills/tree/main/skills/skill-creator) provides create / validate / package / eval workflows, but they are Python scripts plus Claude Code orchestration (`claude -p`, `.claude/commands`). Teams using **Cursor** and **VS Code Copilot** need the same lifecycle as a **pure Go CLI**, with humans and agents sharing one command surface.

## Goals

- Ship a single Go binary `skillctl` with deterministic commands: create, validate, package, eval prepare/aggregate/report, install meta-skills, inject SOP into agent instruction files.
- Support skill-creator-equivalent capability depth over time (full parity as product goal).
- Quality evaluation is **semi-automatic**: IDE agents execute tasks and grade; CLI owns directory protocol, aggregation, and HTML report.
- Assertions may be AI-drafted into files; **no approval state machine**. Users edit assertions and **manually trigger** each subsequent step.
- Target runtimes: Cursor and VS Code Copilot (not Claude Code).

## Non-goals (v1)

- Calling model APIs inside the CLI to run tasks or grade.
- Claude Code-specific trigger injection (`.claude/commands` + `claude -p`).
- Product-specific skills (Formily schema, etc.) — those live in product repos and their own CLIs.
- Automatic end-to-end eval pipelines without user triggers.

## Relationship to other tools

| Tool | Owns |
|------|------|
| `skillctl` (this repo) | Generic skill lifecycle |
| Product CLI (e.g. `formily-ai` in `formily-ai-form`) | Export/install that product’s skill + AI context pack |

`skillctl` must not import or depend on Formily packages.

## Architecture

Hybrid **CLI-first + skill-first**:

1. Humans can run every command directly.
2. Bundled **meta-skills** teach agents when and how to call `skillctl`.
3. `agents inject` writes a short SOP into project agent instruction files so agents discover the workflow.

```text
skillctl (Go, embed assets)
├── commands (cobra)
├── validate / package / bench FS protocol
├── HTML report templates
└── meta-skills: skill-author, eval-runner, skill-grader

Cursor / Copilot
├── business skills (any)
├── meta-skills (installed by skillctl)
└── agent.md / AGENTS.md (SOP from agents inject)
```

Suggested Go libraries: `spf13/cobra` (or equivalent), `embed` for assets, standard `testing`.

## Command surface (v1)

| Command | Actor | Behavior |
|---------|-------|----------|
| `skillctl create <name>` | Human / agent | Scaffold skill dir: `SKILL.md`, optional `scripts/`, `references/`, `assets/`, `evals/evals.json` |
| `skillctl validate <path>` | Human / agent / CI | Static checks aligned with skill-creator `quick_validate` |
| `skillctl package <path>` | Human / agent | Validate then write `<name>.skill` (zip) |
| `skillctl eval init <skill>` | Human / agent | Ensure `evals/evals.json` skeleton exists |
| `skillctl eval prepare <skill> [--iteration N]` | Human / agent | Create `.skillbench/<skill>/iteration-N/...` prompt packs and empty output dirs for with_skill / without_skill |
| `skillctl eval aggregate <skill> [--iteration N]` | Human / agent | Read `grading.json` files → `benchmark.json` + `benchmark.md` |
| `skillctl eval report <skill> [--iteration N]` | Human | Write static HTML (`report.html`) with Outputs + Benchmark views |
| `skillctl skills install-meta [--target cursor\|copilot\|both]` | Human | Install embedded meta-skills into IDE paths |
| `skillctl agents inject [--file path]` | Human | Idempotently insert/update SOP block in agent instruction file |

Trigger evaluation (`run_eval`-style) is deferred: reserve `skillctl eval trigger` in help/docs as future work; do not implement in v1.

Removed vs earlier draft: `eval assert-check` and any `draft|approved` assertion gates.

## Directory protocol

### Skill package

```text
my-skill/
├── SKILL.md
├── scripts/          # optional
├── references/       # optional
├── assets/           # optional
└── evals/
    └── evals.json
```

### `evals/evals.json`

```json
{
  "skill_name": "my-skill",
  "evals": [
    {
      "id": 1,
      "name": "happy-path",
      "prompt": "User task prompt",
      "expected_output": "Optional prose expectation",
      "files": [],
      "assertions": [
        { "id": "a1", "text": "Output contains valid JSON" }
      ]
    }
  ]
}
```

No `review.status` fields. Empty `assertions` is allowed; aggregate/report still work for human inspection of outputs.

### Bench workspace

```text
.skillbench/
└── <skill-name>/
    └── iteration-N/
        ├── eval-<id>-<slug>/
        │   ├── eval_metadata.json
        │   ├── with_skill/
        │   │   ├── outputs/
        │   │   ├── transcript.md
        │   │   └── grading.json
        │   └── without_skill/
        │       ├── outputs/
        │       ├── transcript.md
        │       └── grading.json
        ├── benchmark.json
        ├── benchmark.md
        └── report.html
```

`eval prepare` writes metadata + prompt pack files agents read. IDE agents write outputs, transcripts, and grading JSON. CLI never invents grading results.

## Validation rules

Aligned with skill-creator `quick_validate.py`:

- `SKILL.md` must exist with YAML frontmatter.
- Required: `name`, `description`.
- Allowed top-level keys: `name`, `description`, `license`, `allowed-tools`, `metadata`, `compatibility`.
- `name`: kebab-case `[a-z0-9-]+`, no leading/trailing/double hyphens, max 64 chars.
- `description`: no `<` or `>`, max 1024 chars.
- Optional `compatibility`: string, max 500 chars.
- `--strict`: treat warnings (e.g. very long body) as errors.
- Exit codes: `0` ok, `1` validation/business failure, `2` usage error.
- `--json`: machine-readable stdout; errors as `{"ok":false,"error":{...}}`.

## Grading contract

`grading.json` must use:

```json
{
  "expectations": [
    { "text": "...", "passed": true, "evidence": "..." }
  ]
}
```

`aggregate` fails on schema mismatch. Missing `grading.json` fails by default; `--allow-partial` skips and marks gaps in benchmark.

## Meta-skills

| Skill | Role | Must not |
|-------|------|----------|
| `skill-author` | Interview → `create` → write `SKILL.md` → optional draft assertions into `evals.json` | Auto-run full eval chain |
| `eval-runner` | On explicit user request: execute prepare packs, write `outputs/` + `transcript.md` for with/without | Grade or aggregate |
| `skill-grader` | On explicit user request: grade using assertions → `grading.json` | Edit business skill or assertions |

Install targets (v1 defaults; adjust if vendor docs change):

- Cursor project: `.cursor/skills/<name>/SKILL.md`
- Copilot project: `.github/skills/<name>/SKILL.md` (or documented successor path via `--target`)

`install-meta` only writes the fixed whitelist of meta-skill names; never overwrites arbitrary user skills outside those names.

## `agents inject`

- Default file discovery order: `agent.md` → `AGENTS.md` → `.github/copilot-instructions.md`; override with `--file`.
- Wrap SOP in stable markers (e.g. `<!-- skillctl:sop:start -->` … `<!-- skillctl:sop:end -->`).
- Re-run updates only the marked block (idempotent).
- Create file if missing.
- SOP content: ordered steps, path conventions, key commands, **each step requires explicit user trigger**, do not auto-chain.
- Keep SOP short; deep instructions live in meta-skills.

## Semi-automatic SOP (user-triggered)

1. Author skill (`create` + edit / `skill-author`).
2. `validate` (anytime).
3. User edits `evals/evals.json` (AI draft optional, then user owns the file).
4. `eval prepare`.
5. User triggers `eval-runner` in Cursor/Copilot.
6. User triggers `skill-grader` when ready.
7. `eval aggregate` → `eval report` (static HTML, not an agent).

HTML report is a static artifact (Outputs + Benchmark), equivalent in role to skill-creator’s eval-viewer, without a viewer agent.

## Error handling

- Clear path + remediation on missing files.
- `agents inject` refuses ambiguous marker corruption unless `--force`.
- Never delete user business skill trees as part of meta install.

## Testing

| Layer | Coverage |
|-------|----------|
| Unit | Frontmatter parse, validate rules, schema checks, inject idempotency |
| Golden | Scaffold layout, sample HTML fragments, inject before/after agent.md |
| Integration | Temp dir: prepare → fixture outputs/grading → aggregate → report |
| Out of scope | Live Cursor/Copilot sessions, live LLM calls |

## Implementation sketch (post-spec)

```text
skillctl/
├── cmd/skillctl/
├── internal/create|validate|package|eval|agents|skills|report/
├── assets/meta-skills/...
├── assets/report/...
├── docs/superpowers/specs/
└── go.mod
```

## Success criteria

- A developer can scaffold, validate, and package a skill without Python.
- A Cursor/Copilot user can follow injected SOP + meta-skills to run a with/without eval iteration and get `report.html`.
- Product repos remain independent; they only consume `skillctl` as an external tool.
