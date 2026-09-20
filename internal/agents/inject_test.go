package agents_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hz/skillctl/internal/agents"
)

func TestInjectCreatesAgentMD(t *testing.T) {
	dir := t.TempDir()
	path, err := agents.Inject(dir, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "agent.md" {
		t.Fatalf("created %s", path)
	}
	raw, _ := os.ReadFile(path)
	s := string(raw)
	if !strings.Contains(s, agents.StartMarker) || !strings.Contains(s, "skillctl create") {
		t.Fatalf("%s", s)
	}
}

func TestInjectIdempotent(t *testing.T) {
	dir := t.TempDir()
	p, err := agents.Inject(dir, "", false)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(p)
	if _, err := agents.Inject(dir, "", false); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(p)
	if string(first) != string(second) {
		t.Fatal("re-inject should be identical")
	}
}

func TestInjectPrefersExistingAGENTS(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(target, []byte("# Project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := agents.Inject(dir, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if path != target {
		t.Fatalf("got %s", path)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "# Project") || !strings.Contains(string(raw), agents.StartMarker) {
		t.Fatalf("%s", raw)
	}
}

func TestInjectRefusesCorruption(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "agent.md")
	if err := os.WriteFile(p, []byte(agents.StartMarker+"\nno end\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := agents.Inject(dir, "", false)
	if err == nil || !strings.Contains(err.Error(), "markers") {
		t.Fatalf("got %v", err)
	}
	if _, err := agents.Inject(dir, "", true); err != nil {
		t.Fatal(err)
	}
}

func TestInjectFileOverride(t *testing.T) {
	dir := t.TempDir()
	path, err := agents.Inject(dir, "docs/AGENT.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "docs/AGENT.md") {
		t.Fatalf("%s", path)
	}
}
