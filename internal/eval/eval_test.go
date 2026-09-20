package eval_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hz/skillctl/internal/create"
	"github.com/hz/skillctl/internal/eval"
)

func TestInitCreatesSkeleton(t *testing.T) {
	parent := t.TempDir()
	skill, err := create.Skill(parent, "demo-skill")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(skill, "evals", "evals.json")); err != nil {
		t.Fatal(err)
	}
	path, err := eval.Init(skill)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"evals"`) {
		t.Fatalf("%s", raw)
	}
}

func TestPrepareRequiresEvalCases(t *testing.T) {
	parent := t.TempDir()
	skill, err := create.Skill(parent, "demo-skill")
	if err != nil {
		t.Fatal(err)
	}
	_, err = eval.Prepare(skill, 1, parent)
	if err == nil || !strings.Contains(err.Error(), "no eval cases") {
		t.Fatalf("got %v", err)
	}
}

func TestPrepareWritesPromptPacks(t *testing.T) {
	parent := t.TempDir()
	skill, err := create.Skill(parent, "demo-skill")
	if err != nil {
		t.Fatal(err)
	}
	evals := `{
  "skill_name": "demo-skill",
  "evals": [
    {
      "id": 1,
      "name": "happy-path",
      "prompt": "Do the thing",
      "expected_output": "A thing",
      "files": [],
      "assertions": [{"id": "a1", "text": "Output exists"}]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(skill, "evals", "evals.json"), []byte(evals), 0o644); err != nil {
		t.Fatal(err)
	}
	iter, err := eval.Prepare(skill, 1, parent)
	if err != nil {
		t.Fatal(err)
	}
	evalDir := filepath.Join(iter, "eval-1-happy-path")
	for _, rel := range []string{
		"eval_metadata.json",
		"with_skill/PROMPT.md",
		"with_skill/outputs",
		"without_skill/PROMPT.md",
		"without_skill/outputs",
	} {
		if _, err := os.Stat(filepath.Join(evalDir, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	prompt, _ := os.ReadFile(filepath.Join(evalDir, "with_skill", "PROMPT.md"))
	if !strings.Contains(string(prompt), "Do the thing") || !strings.Contains(string(prompt), skill) {
		t.Fatalf("with_skill prompt:\n%s", prompt)
	}
	base, _ := os.ReadFile(filepath.Join(evalDir, "without_skill", "PROMPT.md"))
	if !strings.Contains(string(base), "Do **not** load") {
		t.Fatalf("without_skill prompt:\n%s", base)
	}
}

func TestAggregateReadsGrading(t *testing.T) {
	parent, skill, iter := prepareEval(t)
	writeGrading(t, filepath.Join(iter, "eval-1-happy-path", "with_skill", "grading.json"), true)
	writeGrading(t, filepath.Join(iter, "eval-1-happy-path", "without_skill", "grading.json"), false)
	out, err := eval.Aggregate(skill, 1, parent, false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "benchmark.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bench map[string]any
	if err := json.Unmarshal(raw, &bench); err != nil {
		t.Fatal(err)
	}
	summary := bench["summary"].(map[string]any)
	if summary["with_skill_pass_rate"].(float64) != 1 {
		t.Fatalf("summary=%v", summary)
	}
	if summary["without_skill_pass_rate"].(float64) != 0 {
		t.Fatalf("summary=%v", summary)
	}
	md, _ := os.ReadFile(filepath.Join(out, "benchmark.md"))
	if !strings.Contains(string(md), "Skill Benchmark") {
		t.Fatalf("md:\n%s", md)
	}
}

func TestAggregateFailsMissingGrading(t *testing.T) {
	parent, skill, _ := prepareEval(t)
	_, err := eval.Aggregate(skill, 1, parent, false)
	if err == nil || !strings.Contains(err.Error(), "grading.json not found") {
		t.Fatalf("got %v", err)
	}
}

func TestAggregateAllowPartial(t *testing.T) {
	parent, skill, _ := prepareEval(t)
	out, err := eval.Aggregate(skill, 1, parent, true)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(out, "benchmark.json"))
	if !strings.Contains(string(raw), `"gaps"`) {
		t.Fatalf("%s", raw)
	}
}

func TestAggregateSchemaMismatch(t *testing.T) {
	parent, skill, iter := prepareEval(t)
	if err := os.WriteFile(filepath.Join(iter, "eval-1-happy-path", "with_skill", "grading.json"), []byte(`{"expectations":[{"text":"x"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	writeGrading(t, filepath.Join(iter, "eval-1-happy-path", "without_skill", "grading.json"), true)
	_, err := eval.Aggregate(skill, 1, parent, false)
	if err == nil || !strings.Contains(err.Error(), "schema mismatch") {
		t.Fatalf("got %v", err)
	}
}

func prepareEval(t *testing.T) (parent, skill, iter string) {
	t.Helper()
	parent = t.TempDir()
	var err error
	skill, err = create.Skill(parent, "demo-skill")
	if err != nil {
		t.Fatal(err)
	}
	evals := `{
  "skill_name": "demo-skill",
  "evals": [{"id": 1, "name": "happy-path", "prompt": "Do it", "expected_output": "", "files": [], "assertions": []}]
}`
	if err := os.WriteFile(filepath.Join(skill, "evals", "evals.json"), []byte(evals), 0o644); err != nil {
		t.Fatal(err)
	}
	iter, err = eval.Prepare(skill, 1, parent)
	if err != nil {
		t.Fatal(err)
	}
	return parent, skill, iter
}

func writeGrading(t *testing.T, path string, passed bool) {
	t.Helper()
	g := eval.Grading{Expectations: []eval.Expectation{{
		Text: "Output contains valid JSON", Passed: passed, Evidence: "saw json",
	}}}
	raw, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
