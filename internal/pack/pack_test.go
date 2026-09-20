package pack_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hz/skillctl/internal/create"
	"github.com/hz/skillctl/internal/pack"
)

func TestPackageWritesZipExcludingEvals(t *testing.T) {
	parent := t.TempDir()
	skill, err := create.Skill(parent, "demo-skill")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "scripts", "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "evals", "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(parent, "dist")
	out, err := pack.Skill(skill, outDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(out) != "demo-skill.skill" {
		t.Fatalf("output name %s", out)
	}
	r, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
		if strings.Contains(f.Name, "evals") {
			t.Fatalf("evals should be excluded: %s", f.Name)
		}
	}
	joined := strings.Join(names, "\n")
	if !strings.Contains(joined, "demo-skill/SKILL.md") {
		t.Fatalf("missing SKILL.md in zip:\n%s", joined)
	}
	if !strings.Contains(joined, "demo-skill/scripts/run.sh") {
		t.Fatalf("missing script:\n%s", joined)
	}
}

func TestPackageFailsInvalidSkill(t *testing.T) {
	dir := t.TempDir()
	skill := filepath.Join(dir, "bad")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := pack.Skill(skill, dir)
	if err == nil {
		t.Fatal("expected validation failure")
	}
}
