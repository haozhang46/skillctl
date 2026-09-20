package validate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hz/skillctl/internal/validate"
)

func writeSkill(t *testing.T, dir, name, content string) string {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

func TestSkillMissingSKILLMD(t *testing.T) {
	dir := t.TempDir()
	res := validate.Skill(dir, false)
	if res.OK {
		t.Fatal("expected failure")
	}
	if res.Err == nil || !strings.Contains(res.Err.Error(), "SKILL.md not found") {
		t.Fatalf("got %+v", res.Err)
	}
}

func TestSkillNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := writeSkill(t, dir, "demo", "# just markdown\n")
	res := validate.Skill(path, false)
	if res.OK {
		t.Fatal("expected failure")
	}
	if !strings.Contains(res.Err.Error(), "YAML frontmatter") {
		t.Fatalf("got %v", res.Err)
	}
}

func TestSkillValid(t *testing.T) {
	dir := t.TempDir()
	path := writeSkill(t, dir, "demo-skill", `---
name: demo-skill
description: Does a thing when the user asks for demos.
---

# Demo
`)
	res := validate.Skill(path, false)
	if !res.OK {
		t.Fatalf("expected ok, got %v", res.Err)
	}
}

func TestSkillUnexpectedKey(t *testing.T) {
	dir := t.TempDir()
	path := writeSkill(t, dir, "demo", `---
name: demo
description: hello
version: 1
---
`)
	res := validate.Skill(path, false)
	if res.OK {
		t.Fatal("expected failure")
	}
	if !strings.Contains(res.Err.Error(), "Unexpected key") {
		t.Fatalf("got %v", res.Err)
	}
}

func TestSkillMissingName(t *testing.T) {
	dir := t.TempDir()
	path := writeSkill(t, dir, "demo", `---
description: hello
---
`)
	res := validate.Skill(path, false)
	if res.OK || !strings.Contains(res.Err.Error(), "name") {
		t.Fatalf("got %v", res.Err)
	}
}

func TestSkillNameNotKebab(t *testing.T) {
	dir := t.TempDir()
	path := writeSkill(t, dir, "demo", `---
name: Demo_Skill
description: hello
---
`)
	res := validate.Skill(path, false)
	if res.OK || !strings.Contains(res.Err.Error(), "kebab-case") {
		t.Fatalf("got %v", res.Err)
	}
}

func TestSkillNameDoubleHyphen(t *testing.T) {
	dir := t.TempDir()
	path := writeSkill(t, dir, "demo", `---
name: demo--skill
description: hello
---
`)
	res := validate.Skill(path, false)
	if res.OK || !strings.Contains(res.Err.Error(), "hyphen") {
		t.Fatalf("got %v", res.Err)
	}
}

func TestSkillNameTooLong(t *testing.T) {
	dir := t.TempDir()
	name := strings.Repeat("a", 65)
	path := writeSkill(t, dir, "demo", "---\nname: "+name+"\ndescription: hello\n---\n")
	res := validate.Skill(path, false)
	if res.OK || !strings.Contains(res.Err.Error(), "64") {
		t.Fatalf("got %v", res.Err)
	}
}

func TestSkillDescriptionAngleBrackets(t *testing.T) {
	dir := t.TempDir()
	path := writeSkill(t, dir, "demo", `---
name: demo
description: uses <tool>
---
`)
	res := validate.Skill(path, false)
	if res.OK || !strings.Contains(res.Err.Error(), "angle brackets") {
		t.Fatalf("got %v", res.Err)
	}
}

func TestSkillDescriptionTooLong(t *testing.T) {
	dir := t.TempDir()
	desc := strings.Repeat("a", 1025)
	path := writeSkill(t, dir, "demo", "---\nname: demo\ndescription: "+desc+"\n---\n")
	res := validate.Skill(path, false)
	if res.OK || !strings.Contains(res.Err.Error(), "1024") {
		t.Fatalf("got %v", res.Err)
	}
}

func TestSkillCompatibilityTooLong(t *testing.T) {
	dir := t.TempDir()
	compat := strings.Repeat("a", 501)
	path := writeSkill(t, dir, "demo", "---\nname: demo\ndescription: hello\ncompatibility: "+compat+"\n---\n")
	res := validate.Skill(path, false)
	if res.OK || !strings.Contains(res.Err.Error(), "500") {
		t.Fatalf("got %v", res.Err)
	}
}

func TestSkillLongBodyWarning(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("---\nname: demo\ndescription: hello\n---\n")
	for i := 0; i < 501; i++ {
		b.WriteString("line\n")
	}
	path := writeSkill(t, dir, "demo", b.String())
	res := validate.Skill(path, false)
	if !res.OK {
		t.Fatalf("warnings should not fail: %v", res.Err)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("expected long-body warning")
	}
	strict := validate.Skill(path, true)
	if strict.OK {
		t.Fatal("strict should fail on long body")
	}
}

func TestSkillAllowedOptionalKeys(t *testing.T) {
	dir := t.TempDir()
	path := writeSkill(t, dir, "demo", `---
name: demo
description: hello
license: MIT
allowed-tools: Bash
metadata:
  author: testers
compatibility: go
---
`)
	res := validate.Skill(path, false)
	if !res.OK {
		t.Fatalf("expected ok, got %v", res.Err)
	}
}
