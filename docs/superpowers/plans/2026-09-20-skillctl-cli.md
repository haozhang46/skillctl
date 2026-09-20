# skillctl CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a single Go binary `skillctl` with create, validate, package, eval prepare/aggregate/report, install-meta, and agents inject.

**Architecture:** Cobra in `internal/cli` wires commands; each domain lives in `internal/{create,validate,pack,eval,agents,skills,report}`. Assets (meta-skills, SOP, HTML template) are `embed.FS`. Cobra parses flags and prints; business packages never import cobra.

**Tech Stack:** Go 1.22, `spf13/cobra`, `gopkg.in/yaml.v3`, `embed`, standard `archive/zip`, `html/template`, `testing`.

## Global Constraints

- Pure Go CLI; no model API calls; no Formily imports.
- Exit codes: `0` ok, `1` validation/business failure, `2` usage error.
- `--json`: machine-readable stdout; errors `{"ok":false,"error":{...}}`.
- `--strict`: treat warnings (e.g. SKILL.md body over 500 lines) as errors.
- Cobra only: no Viper, no Fang/TUI, no OpenTelemetry.
- Target runtimes: Cursor and VS Code Copilot (not Claude Code).
- `eval trigger` reserved in help; do not implement.
- Meta install whitelist only: `skill-author`, `eval-runner`, `skill-grader`.
- SOP markers: `<!-- skillctl:sop:start -->` … `<!-- skillctl:sop:end -->`.
- Cursor install: `.cursor/skills/<name>/SKILL.md`; Copilot: `.github/skills/<name>/SKILL.md`.
- Bench workspace: `.skillbench/<skill-name>/iteration-N/...`.
- Package output: `<dir-basename>.skill` zip; exclude root `evals/`, `__pycache__`, `node_modules`, `.DS_Store`, `*.pyc`.
- Do not commit unless the user asks.

## File map

- `cmd/skillctl/main.go` — `os.Exit(cli.Execute())`
- `internal/app/error.go` — `app.Error` with `Code` (1|2), `Kind`, `Message`, `Path`, `Hint`
- `internal/app/json.go` — encode `{ok, error?, ...}`
- `internal/validate` — frontmatter + quick_validate rules
- `internal/create` — scaffold
- `internal/pack` — zip after validate
- `internal/eval` — init / prepare / aggregate
- `internal/report` — HTML
- `internal/agents` — inject SOP
- `internal/skills` — install-meta
- `internal/cli` — cobra tree
- `assets/` — embed.FS (package `assets`)

## Interfaces

```go
package app
type Error struct { ExitCode int; Kind, Message, Path, Hint string }

package validate
func Skill(path string, strict bool) Result // Result has OK, Warnings, Error

package create
func Skill(parentDir, name string) (string, error)

package pack
func Skill(skillPath, outputDir string) (string, error)

package eval
func Init(skillPath string) error
func Prepare(skillPath string, iteration int) error
func Aggregate(skillPath string, iteration int, allowPartial bool) error

package report
func Write(skillPath string, iteration int) (string, error)

package agents
func Inject(cwd, file string, force bool) (string, error)

package skills
func InstallMeta(cwd, target string) error // target: cursor|copilot|both
```

## Tasks

### Task 1: app errors + validate
TDD: missing SKILL.md, invalid YAML, unexpected keys, kebab-case name, description `<>`, lengths, empty required fields, `--strict` long body.

### Task 2: create scaffold
TDD: writes SKILL.md + evals.json + optional dirs; rejects bad names; refuses overwrite.

### Task 3: package
TDD: validates first; zip contains `name/SKILL.md`; excludes `evals/`.

### Task 4: eval init/prepare/aggregate
TDD: skeleton evals.json; iteration dirs + prompt packs; grading schema; `--allow-partial`.

### Task 5: HTML report
TDD: golden fragment contains Outputs + Benchmark.

### Task 6: agents inject + install-meta
TDD: idempotent markers; discovery order; `--force`; whitelist only.

### Task 7: cobra CLI
TDD: usage → exit 2; validate fail → exit 1; `--json` error envelope; `eval trigger` documented not implemented.

### Task 8: README + `go test ./...`
