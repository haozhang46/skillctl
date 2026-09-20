package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hz/skillctl/internal/skills"
	"github.com/hz/skillctl/internal/validate"
)

func TestInstallMetaBoth(t *testing.T) {
	dir := t.TempDir()
	written, err := skills.InstallMeta(dir, "both")
	if err != nil {
		t.Fatal(err)
	}
	if len(written) == 0 {
		t.Fatal("expected files")
	}
	for _, name := range []string{"skill-author", "eval-runner", "skill-grader"} {
		cursor := filepath.Join(dir, ".cursor", "skills", name, "SKILL.md")
		copilot := filepath.Join(dir, ".github", "skills", name, "SKILL.md")
		if _, err := os.Stat(cursor); err != nil {
			t.Errorf("missing %s", cursor)
		}
		if _, err := os.Stat(copilot); err != nil {
			t.Errorf("missing %s", copilot)
		}
		raw, _ := os.ReadFile(cursor)
		if !strings.Contains(string(raw), "name: "+name) {
			t.Errorf("%s content:\n%s", name, raw)
		}
		res := validate.Skill(filepath.Dir(cursor), false)
		if !res.OK {
			t.Errorf("%s does not validate: %v", name, res.Err)
		}
	}
	// does not write arbitrary skills
	if _, err := os.Stat(filepath.Join(dir, ".cursor", "skills", "other")); !os.IsNotExist(err) {
		t.Fatal("must not write non-whitelist skills")
	}
}

func TestInstallMetaCursorOnly(t *testing.T) {
	dir := t.TempDir()
	if _, err := skills.InstallMeta(dir, "cursor"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".github", "skills")); !os.IsNotExist(err) {
		t.Fatal("copilot path should be absent")
	}
}

func TestInstallMetaInvalidTarget(t *testing.T) {
	_, err := skills.InstallMeta(t.TempDir(), "claude")
	if err == nil {
		t.Fatal("expected usage error")
	}
}

func TestInstallMetaDoesNotTouchUserSkill(t *testing.T) {
	dir := t.TempDir()
	user := filepath.Join(dir, ".cursor", "skills", "my-biz", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(user), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(user, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := skills.InstallMeta(dir, "cursor"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(user)
	if string(raw) != "keep me" {
		t.Fatalf("user skill overwritten: %s", raw)
	}
}
