package report_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hz/skillctl/internal/create"
	"github.com/hz/skillctl/internal/eval"
	"github.com/hz/skillctl/internal/report"
)

func TestWriteHTMLContainsOutputsAndBenchmark(t *testing.T) {
	parent := t.TempDir()
	skill, err := create.Skill(parent, "demo-skill")
	if err != nil {
		t.Fatal(err)
	}
	evals := `{"skill_name":"demo-skill","evals":[{"id":1,"name":"happy-path","prompt":"Do it","expected_output":"","files":[],"assertions":[]}]}`
	if err := os.WriteFile(filepath.Join(skill, "evals", "evals.json"), []byte(evals), 0o644); err != nil {
		t.Fatal(err)
	}
	iter, err := eval.Prepare(skill, 1, parent)
	if err != nil {
		t.Fatal(err)
	}
	evalDir := filepath.Join(iter, "eval-1-happy-path")
	if err := os.WriteFile(filepath.Join(evalDir, "with_skill", "transcript.md"), []byte("did the thing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evalDir, "with_skill", "outputs", "out.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := eval.Grading{Expectations: []eval.Expectation{{Text: "has output", Passed: true, Evidence: "out.txt"}}}
	raw, _ := json.Marshal(g)
	_ = os.WriteFile(filepath.Join(evalDir, "with_skill", "grading.json"), raw, 0o644)
	_ = os.WriteFile(filepath.Join(evalDir, "without_skill", "grading.json"), raw, 0o644)
	if _, err := eval.Aggregate(skill, 1, parent, false); err != nil {
		t.Fatal(err)
	}
	htmlPath, err := report.Write(skill, 1, parent)
	if err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)
	for _, needle := range []string{"id=\"outputs\"", "id=\"benchmark\"", "demo-skill", "did the thing", "out.txt", "Pass rate"} {
		if !strings.Contains(s, needle) {
			t.Errorf("missing %q in report", needle)
		}
	}
}
