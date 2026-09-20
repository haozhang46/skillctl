package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hz/skillctl/internal/app"
	"github.com/hz/skillctl/internal/cli"
)

func TestUsageMissingArgsExit2(t *testing.T) {
	code, _, errOut := run(t, nil, "validate")
	if code != app.ExitUsage {
		t.Fatalf("code=%d err=%s", code, errOut)
	}
}

func TestJSONValidationError(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := run(t, &dir, "--json", "validate", dir)
	if code != app.ExitFail {
		t.Fatalf("code=%d out=%s", code, out)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	if env["ok"] != false {
		t.Fatalf("%s", out)
	}
	errObj := env["error"].(map[string]any)
	if !strings.Contains(errObj["message"].(string), "SKILL.md not found") {
		t.Fatalf("%s", out)
	}
}

func TestCreateValidatePackageRoundTrip(t *testing.T) {
	dir := t.TempDir()
	code, out, errOut := run(t, &dir, "create", "demo-skill")
	if code != 0 {
		t.Fatalf("create %d %s %s", code, out, errOut)
	}
	skill := filepath.Join(dir, "demo-skill")
	code, out, errOut = run(t, &dir, "validate", skill)
	if code != 0 {
		t.Fatalf("validate %d %s %s", code, out, errOut)
	}
	if !strings.Contains(out, "Skill is valid") {
		t.Fatalf("%s", out)
	}
	code, out, errOut = run(t, &dir, "package", skill, "-o", "dist")
	if code != 0 {
		t.Fatalf("package %d %s %s", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "dist", "demo-skill.skill")); err != nil {
		t.Fatal(err)
	}
}

func TestEvalTriggerReserved(t *testing.T) {
	code, _, errOut := run(t, nil, "eval", "trigger")
	if code != app.ExitFail {
		t.Fatalf("code=%d %s", code, errOut)
	}
	if !strings.Contains(errOut, "not implemented") {
		t.Fatalf("%s", errOut)
	}
}

func TestAgentsInjectAndInstallMeta(t *testing.T) {
	dir := t.TempDir()
	code, out, errOut := run(t, &dir, "agents", "inject")
	if code != 0 {
		t.Fatalf("inject %d %s %s", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "agent.md")); err != nil {
		t.Fatal(err)
	}
	code, out, errOut = run(t, &dir, "skills", "install-meta", "--target", "cursor")
	if code != 0 {
		t.Fatalf("install %d %s %s", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, ".cursor", "skills", "skill-author", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestEvalPrepareAggregateReport(t *testing.T) {
	dir := t.TempDir()
	if code, _, errOut := run(t, &dir, "create", "demo-skill"); code != 0 {
		t.Fatal(errOut)
	}
	skill := filepath.Join(dir, "demo-skill")
	evals := `{"skill_name":"demo-skill","evals":[{"id":1,"name":"happy-path","prompt":"Do it","expected_output":"","files":[],"assertions":[]}]}`
	if err := os.WriteFile(filepath.Join(skill, "evals", "evals.json"), []byte(evals), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run(t, &dir, "eval", "prepare", skill); code != 0 {
		t.Fatal(errOut)
	}
	g := `{"expectations":[{"text":"ok","passed":true,"evidence":"yes"}]}`
	base := filepath.Join(dir, ".skillbench", "demo-skill", "iteration-1", "eval-1-happy-path")
	_ = os.WriteFile(filepath.Join(base, "with_skill", "grading.json"), []byte(g), 0o644)
	_ = os.WriteFile(filepath.Join(base, "without_skill", "grading.json"), []byte(g), 0o644)
	if code, _, errOut := run(t, &dir, "eval", "aggregate", skill); code != 0 {
		t.Fatal(errOut)
	}
	if code, out, errOut := run(t, &dir, "eval", "report", skill); code != 0 {
		t.Fatalf("%s %s", out, errOut)
	}
	html, err := os.ReadFile(filepath.Join(dir, ".skillbench", "demo-skill", "iteration-1", "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "Benchmark") {
		t.Fatalf("%s", html)
	}
}

func run(t *testing.T, cwd *string, args ...string) (int, string, string) {
	t.Helper()
	dir := t.TempDir()
	if cwd != nil {
		dir = *cwd
	}
	var out, err bytes.Buffer
	code := cli.Run(args, &out, &err, dir)
	return code, out.String(), err.String()
}
