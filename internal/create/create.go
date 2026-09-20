package create

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hz/skillctl/internal/app"
	"github.com/hz/skillctl/internal/validate"
)

func Skill(parentDir, name string) (string, error) {
	if err := validate.CheckName(name); err != nil {
		err.Message = strings.Replace(err.Message, " in frontmatter", "", 1)
		return "", err
	}
	name = strings.TrimSpace(name)
	dest := filepath.Join(parentDir, name)
	if _, err := os.Stat(dest); err == nil {
		return "", app.WithHint(
			app.WithPath(app.Fail("exists", "skill directory already exists"), dest),
			"choose a different name or remove the existing directory",
		)
	} else if !os.IsNotExist(err) {
		return "", app.WithPath(app.Fail("io", err.Error()), dest)
	}

	dirs := []string{
		dest,
		filepath.Join(dest, "scripts"),
		filepath.Join(dest, "references"),
		filepath.Join(dest, "assets"),
		filepath.Join(dest, "evals"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", app.WithPath(app.Fail("io", err.Error()), d)
		}
	}
	for _, rel := range []string{"scripts/.gitkeep", "references/.gitkeep", "assets/.gitkeep"} {
		if err := os.WriteFile(filepath.Join(dest, rel), []byte{}, 0o644); err != nil {
			return "", err
		}
	}

	skillMD := fmt.Sprintf(`---
name: %s
description: TODO. Describe what this skill does and when to use it. Include trigger contexts.
---

# %s

## Instructions

Write the workflow the agent should follow.

## Resources

- `+"`scripts/`"+` — deterministic helpers
- `+"`references/`"+` — docs loaded on demand
- `+"`assets/`"+` — templates and other files used in output
`, name, name)
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		return "", err
	}

	evals := map[string]any{
		"skill_name": name,
		"evals":      []any{},
	}
	raw, err := json.MarshalIndent(evals, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(dest, "evals", "evals.json"), raw, 0o644); err != nil {
		return "", err
	}
	return dest, nil
}
