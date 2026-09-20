package create_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hz/skillctl/internal/create"
	"github.com/hz/skillctl/internal/validate"
)

func TestCreateWritesScaffold(t *testing.T) {
	parent := t.TempDir()
	path, err := create.Skill(parent, "demo-skill")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(parent, "demo-skill")
	if path != want {
		t.Fatalf("path=%s want %s", path, want)
	}
	for _, rel := range []string{
		"SKILL.md",
		"evals/evals.json",
		"scripts/.gitkeep",
		"references/.gitkeep",
		"assets/.gitkeep",
	} {
		if _, err := os.Stat(filepath.Join(path, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(path, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "name: demo-skill") {
		t.Fatalf("SKILL.md:\n%s", raw)
	}
	res := validate.Skill(path, false)
	if !res.OK {
		t.Fatalf("scaffold must validate: %v", res.Err)
	}
	var evals map[string]any
	b, err := os.ReadFile(filepath.Join(path, "evals/evals.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &evals); err != nil {
		t.Fatal(err)
	}
	if evals["skill_name"] != "demo-skill" {
		t.Fatalf("evals=%v", evals)
	}
}

func TestCreateRejectsBadName(t *testing.T) {
	_, err := create.Skill(t.TempDir(), "Not_Valid")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateRefusesOverwrite(t *testing.T) {
	parent := t.TempDir()
	if _, err := create.Skill(parent, "demo-skill"); err != nil {
		t.Fatal(err)
	}
	_, err := create.Skill(parent, "demo-skill")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("got %v", err)
	}
}
