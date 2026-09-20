package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hz/skillctl/internal/app"
	"github.com/hz/skillctl/internal/validate"
)

type EvalsFile struct {
	SkillName string `json:"skill_name"`
	Evals     []Case `json:"evals"`
}

type Case struct {
	ID             int         `json:"id"`
	Name           string      `json:"name"`
	Prompt         string      `json:"prompt"`
	ExpectedOutput string      `json:"expected_output"`
	Files          []string    `json:"files"`
	Assertions     []Assertion `json:"assertions"`
}

type Assertion struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Grading struct {
	Expectations []Expectation `json:"expectations"`
}

type Expectation struct {
	Text     string `json:"text"`
	Passed   bool   `json:"passed"`
	Evidence string `json:"evidence"`
}

type Benchmark struct {
	SkillName string           `json:"skill_name"`
	Iteration int              `json:"iteration"`
	Evals     []EvalResult     `json:"evals"`
	Summary   BenchmarkSummary `json:"summary"`
	Gaps      []string         `json:"gaps,omitempty"`
}

type EvalResult struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	WithSkill    *Run   `json:"with_skill"`
	WithoutSkill *Run   `json:"without_skill"`
}

type Run struct {
	PassRate     float64       `json:"pass_rate"`
	Passed       int           `json:"passed"`
	Failed       int           `json:"failed"`
	Total        int           `json:"total"`
	Expectations []Expectation `json:"expectations"`
	Partial      bool          `json:"partial,omitempty"`
}

type BenchmarkSummary struct {
	WithSkillPassRate    float64 `json:"with_skill_pass_rate"`
	WithoutSkillPassRate float64 `json:"without_skill_pass_rate"`
}

func Init(skillPath string) (string, error) {
	skillPath, name, err := resolveSkill(skillPath)
	if err != nil {
		return "", err
	}
	evalsPath := filepath.Join(skillPath, "evals", "evals.json")
	if _, err := os.Stat(evalsPath); err == nil {
		return evalsPath, nil
	}
	if err := os.MkdirAll(filepath.Dir(evalsPath), 0o755); err != nil {
		return "", err
	}
	skel := EvalsFile{SkillName: name, Evals: []Case{}}
	raw, err := json.MarshalIndent(skel, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(evalsPath, raw, 0o644); err != nil {
		return "", err
	}
	return evalsPath, nil
}

func Prepare(skillPath string, iteration int, workspace string) (string, error) {
	if iteration < 1 {
		return "", app.Usage("--iteration must be >= 1")
	}
	skillPath, name, err := resolveSkill(skillPath)
	if err != nil {
		return "", err
	}
	evals, err := loadEvals(skillPath)
	if err != nil {
		return "", err
	}
	if workspace == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		workspace = cwd
	}
	if len(evals.Evals) == 0 {
		return "", app.WithHint(
			app.WithPath(app.Fail("validation", "evals.json has no eval cases"), filepath.Join(skillPath, "evals", "evals.json")),
			"add at least one eval, then re-run skillctl eval prepare",
		)
	}
	iterDir := filepath.Join(workspace, ".skillbench", name, fmt.Sprintf("iteration-%d", iteration))
	if err := os.MkdirAll(iterDir, 0o755); err != nil {
		return "", err
	}
	for _, c := range evals.Evals {
		slug := slugify(c.Name)
		if slug == "" {
			slug = "eval"
		}
		evalDir := filepath.Join(iterDir, fmt.Sprintf("eval-%d-%s", c.ID, slug))
		meta := map[string]any{
			"eval_id":         c.ID,
			"eval_name":       c.Name,
			"prompt":          c.Prompt,
			"expected_output": c.ExpectedOutput,
			"files":           c.Files,
			"assertions":      c.Assertions,
			"skill_path":      skillPath,
		}
		if err := writeJSON(filepath.Join(evalDir, "eval_metadata.json"), meta); err != nil {
			return "", err
		}
		for _, cfg := range []string{"with_skill", "without_skill"} {
			outDir := filepath.Join(evalDir, cfg, "outputs")
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return "", err
			}
			prompt := withSkillPrompt(c, skillPath)
			if cfg == "without_skill" {
				prompt = withoutSkillPrompt(c)
			}
			if err := os.WriteFile(filepath.Join(evalDir, cfg, "PROMPT.md"), []byte(prompt), 0o644); err != nil {
				return "", err
			}
		}
	}
	return iterDir, nil
}

func Aggregate(skillPath string, iteration int, workspace string, allowPartial bool) (string, error) {
	if iteration < 1 {
		return "", app.Usage("--iteration must be >= 1")
	}
	skillPath, name, err := resolveSkill(skillPath)
	if err != nil {
		return "", err
	}
	if workspace == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		workspace = cwd
	}
	iterDir := filepath.Join(workspace, ".skillbench", name, fmt.Sprintf("iteration-%d", iteration))
	entries, err := os.ReadDir(iterDir)
	if err != nil {
		return "", app.WithHint(
			app.WithPath(app.Fail("not_found", "iteration directory not found"), iterDir),
			"run skillctl eval prepare first",
		)
	}
	bench := Benchmark{SkillName: name, Iteration: iteration}
	var gaps []string
	var withRates, withoutRates []float64
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "eval-") {
			continue
		}
		evalDir := filepath.Join(iterDir, e.Name())
		meta, err := readMeta(filepath.Join(evalDir, "eval_metadata.json"))
		if err != nil {
			return "", err
		}
		result := EvalResult{ID: meta.ID, Name: meta.Name}
		ws, g, err := loadRun(filepath.Join(evalDir, "with_skill", "grading.json"), allowPartial)
		if err != nil {
			return "", err
		}
		if g != "" {
			gaps = append(gaps, g)
		}
		result.WithSkill = ws
		if ws != nil && !ws.Partial {
			withRates = append(withRates, ws.PassRate)
		}
		wos, g, err := loadRun(filepath.Join(evalDir, "without_skill", "grading.json"), allowPartial)
		if err != nil {
			return "", err
		}
		if g != "" {
			gaps = append(gaps, g)
		}
		result.WithoutSkill = wos
		if wos != nil && !wos.Partial {
			withoutRates = append(withoutRates, wos.PassRate)
		}
		bench.Evals = append(bench.Evals, result)
	}
	if len(bench.Evals) == 0 {
		return "", app.WithPath(app.Fail("validation", "no eval directories found"), iterDir)
	}
	bench.Summary.WithSkillPassRate = mean(withRates)
	bench.Summary.WithoutSkillPassRate = mean(withoutRates)
	bench.Gaps = gaps
	if err := writeJSON(filepath.Join(iterDir, "benchmark.json"), bench); err != nil {
		return "", err
	}
	md := renderMarkdown(bench)
	if err := os.WriteFile(filepath.Join(iterDir, "benchmark.md"), []byte(md), 0o644); err != nil {
		return "", err
	}
	return iterDir, nil
}

func IterationDir(workspace, skillName string, iteration int) string {
	return filepath.Join(workspace, ".skillbench", skillName, fmt.Sprintf("iteration-%d", iteration))
}

func ResolveSkill(skillPath string) (abs string, name string, err error) {
	return resolveSkill(skillPath)
}

type metaFile struct {
	ID   int    `json:"eval_id"`
	Name string `json:"eval_name"`
}

func resolveSkill(skillPath string) (string, string, error) {
	abs, err := filepath.Abs(skillPath)
	if err != nil {
		return "", "", err
	}
	res := validate.Skill(abs, false)
	if !res.OK {
		return "", "", res.Err
	}
	return abs, res.Name, nil
}

func loadEvals(skillPath string) (*EvalsFile, error) {
	p := filepath.Join(skillPath, "evals", "evals.json")
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, app.WithHint(
			app.WithPath(app.Fail("not_found", "evals/evals.json not found"), p),
			"run skillctl eval init or add evals.json",
		)
	}
	var ef EvalsFile
	if err := json.Unmarshal(raw, &ef); err != nil {
		return nil, app.WithPath(app.Fail("validation", "invalid evals.json: "+err.Error()), p)
	}
	return &ef, nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func readMeta(path string) (metaFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return metaFile{}, app.WithPath(app.Fail("not_found", "eval_metadata.json not found"), path)
	}
	var m metaFile
	if err := json.Unmarshal(raw, &m); err != nil {
		return metaFile{}, app.WithPath(app.Fail("validation", "invalid eval_metadata.json"), path)
	}
	return m, nil
}

func loadRun(path string, allowPartial bool) (*Run, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if allowPartial {
				return &Run{Partial: true}, path, nil
			}
			return nil, "", app.WithHint(
				app.WithPath(app.Fail("not_found", "grading.json not found"), path),
				"run skill-grader, or pass --allow-partial",
			)
		}
		return nil, "", err
	}
	var envelope struct {
		Expectations []map[string]json.RawMessage `json:"expectations"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, "", app.WithPath(app.Fail("validation", "invalid grading.json: "+err.Error()), path)
	}
	if envelope.Expectations == nil {
		return nil, "", app.WithPath(app.Fail("validation", "grading.json schema mismatch: missing expectations"), path)
	}
	g := Grading{Expectations: make([]Expectation, 0, len(envelope.Expectations))}
	for i, m := range envelope.Expectations {
		for _, key := range []string{"text", "passed", "evidence"} {
			if _, ok := m[key]; !ok {
				return nil, "", app.WithPath(app.Fail("validation",
					fmt.Sprintf("grading.json schema mismatch: expectations[%d] missing %s", i, key)), path)
			}
		}
		var e Expectation
		buf, _ := json.Marshal(m)
		if err := json.Unmarshal(buf, &e); err != nil {
			return nil, "", app.WithPath(app.Fail("validation",
				fmt.Sprintf("grading.json schema mismatch: expectations[%d]: %s", i, err.Error())), path)
		}
		if strings.TrimSpace(e.Text) == "" {
			return nil, "", app.WithPath(app.Fail("validation",
				fmt.Sprintf("grading.json schema mismatch: expectations[%d] missing text", i)), path)
		}
		g.Expectations = append(g.Expectations, e)
	}
	passed := 0
	for _, e := range g.Expectations {
		if e.Passed {
			passed++
		}
	}
	total := len(g.Expectations)
	rate := 0.0
	if total > 0 {
		rate = float64(passed) / float64(total)
	}
	return &Run{
		PassRate:     rate,
		Passed:       passed,
		Failed:       total - passed,
		Total:        total,
		Expectations: g.Expectations,
	}, "", nil
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func withSkillPrompt(c Case, skillPath string) string {
	var b strings.Builder
	b.WriteString("# Eval task (with skill)\n\n")
	b.WriteString("Use the skill at `")
	b.WriteString(skillPath)
	b.WriteString("`.\n\n")
	b.WriteString("## Task\n\n")
	b.WriteString(c.Prompt)
	b.WriteString("\n\n")
	b.WriteString("Save outputs into `outputs/` in this directory. Write a `transcript.md` summarizing what you did.\n")
	b.WriteString("Do not write `grading.json`. Do not run aggregate or report.\n")
	return b.String()
}

func withoutSkillPrompt(c Case) string {
	var b strings.Builder
	b.WriteString("# Eval task (without skill)\n\n")
	b.WriteString("Do **not** load or follow a skill. Complete the task from general knowledge only.\n\n")
	b.WriteString("## Task\n\n")
	b.WriteString(c.Prompt)
	b.WriteString("\n\n")
	b.WriteString("Save outputs into `outputs/` in this directory. Write a `transcript.md` summarizing what you did.\n")
	b.WriteString("Do not write `grading.json`. Do not run aggregate or report.\n")
	return b.String()
}

func renderMarkdown(b Benchmark) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# Skill Benchmark: %s\n\n", b.SkillName)
	fmt.Fprintf(&s, "**Iteration**: %d\n\n", b.Iteration)
	s.WriteString("## Summary\n\n")
	s.WriteString("| Metric | With skill | Without skill |\n")
	s.WriteString("|--------|------------|---------------|\n")
	fmt.Fprintf(&s, "| Pass rate | %.0f%% | %.0f%% |\n\n", b.Summary.WithSkillPassRate*100, b.Summary.WithoutSkillPassRate*100)
	s.WriteString("## Evals\n\n")
	for _, e := range b.Evals {
		fmt.Fprintf(&s, "### eval-%d %s\n\n", e.ID, e.Name)
		fmt.Fprintf(&s, "- with_skill: %s\n", formatRun(e.WithSkill))
		fmt.Fprintf(&s, "- without_skill: %s\n\n", formatRun(e.WithoutSkill))
	}
	if len(b.Gaps) > 0 {
		s.WriteString("## Gaps\n\n")
		for _, g := range b.Gaps {
			fmt.Fprintf(&s, "- missing %s\n", g)
		}
	}
	return s.String()
}

func formatRun(r *Run) string {
	if r == nil {
		return "missing"
	}
	if r.Partial {
		return "partial (no grading.json)"
	}
	return fmt.Sprintf("%d/%d (%.0f%%)", r.Passed, r.Total, r.PassRate*100)
}
