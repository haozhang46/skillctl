# skillctl

Go CLI for creating, validating, packaging, and (semi-automatically) evaluating Agent Skills for **Cursor** and **VS Code Copilot**.

This repository is intentionally separate from product-specific skill content (for example `formily-ai-form`). Product repos export their own skill/context; `skillctl` owns the skill lifecycle tooling.

Design: [`docs/superpowers/specs/2026-09-20-skillctl-design.md`](docs/superpowers/specs/2026-09-20-skillctl-design.md)

## Install

```bash
npx @hz/skillctl --help
# or
npm i -g @hz/skillctl
```

Unscoped `skillctl` is taken on npm. The command name is still `skillctl` after a global install.

## Build

Requires Go 1.22+.

```bash
go build -o skillctl ./cmd/skillctl
```

The npm wrapper lives in [`npm/skillctl`](npm/skillctl) (`@hz/skillctl`). Platform binaries are sibling packages under `npm/`.

### npm packages

Cross-compile platform binaries into `npm/skillctl-*/bin`, then pack tarballs:

```bash
./scripts/build-npm.sh pack
```

Publish **platform packages first**, then the wrapper (`@hz/skillctl`). Requires the public npm org `hz`.

```bash
# after build-npm.sh
for dir in npm/skillctl-darwin-arm64 npm/skillctl-darwin-amd64 npm/skillctl-linux-amd64 npm/skillctl-linux-arm64 npm/skillctl-windows-amd64; do
  (cd "$dir" && npm publish --access public)
done
(cd npm/skillctl && npm publish --access public)
```

Local smoke test (skip registry optional deps until the platform packages exist on npm):

```bash
./scripts/build-npm.sh pack
tmp=$(mktemp -d)
npm i --prefix "$tmp" ./npm/skillctl-darwin-arm64/hz-skillctl-darwin-arm64-0.1.0.tgz
npm i --prefix "$tmp" ./npm/skillctl/hz-skillctl-0.1.0.tgz --omit=optional
"$tmp/node_modules/.bin/skillctl" --help
```

## Commands

| Command | Behavior |
|---------|----------|
| `skillctl create <name>` | Scaffold skill dir |
| `skillctl validate <path> [--strict]` | Static checks (skill-creator `quick_validate`) |
| `skillctl package <path> [-o dir]` | Validate then write `<name>.skill` zip |
| `skillctl eval init <skill>` | Ensure `evals/evals.json` skeleton |
| `skillctl eval prepare <skill> [--iteration N]` | Prompt packs under `.skillbench/` |
| `skillctl eval aggregate <skill> [--iteration N] [--allow-partial]` | `benchmark.json` + `benchmark.md` |
| `skillctl eval report <skill> [--iteration N]` | Static `report.html` |
| `skillctl skills install-meta [--target cursor\|copilot\|both]` | Install bundled meta-skills |
| `skillctl agents inject [--file path] [--force]` | SOP block in agent instruction file |

Global: `--json` for machine-readable stdout. Exit `0` ok, `1` business/validation, `2` usage.

`skillctl eval trigger` is reserved and not implemented in v1.

## SOP

Each eval step needs an explicit user trigger. Meta-skills `skill-author`, `eval-runner`, and `skill-grader` teach agents when to call this CLI; they must not auto-chain the full pipeline.
