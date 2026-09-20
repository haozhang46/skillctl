package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/hz/skillctl/assets"
	"github.com/hz/skillctl/internal/app"
	"github.com/hz/skillctl/internal/eval"
)

type page struct {
	SkillName    string
	Iteration    int
	Evals        []evalView
	HasBenchmark bool
	WithPct      float64
	WithoutPct   float64
	Gaps         []string
}

type evalView struct {
	ID           int
	Name         string
	Configs      []configView
	WithLabel    string
	WithoutLabel string
}

type configView struct {
	Name         string
	Files        []string
	Transcript   string
	Expectations []eval.Expectation
	Partial      bool
}

func Write(skillPath string, iteration int, workspace string) (string, error) {
	if iteration < 1 {
		return "", app.Usage("--iteration must be >= 1")
	}
	_, name, err := eval.ResolveSkill(skillPath)
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
	iterDir := eval.IterationDir(workspace, name, iteration)
	entries, err := os.ReadDir(iterDir)
	if err != nil {
		return "", app.WithHint(
			app.WithPath(app.Fail("not_found", "iteration directory not found"), iterDir),
			"run skillctl eval prepare first",
		)
	}

	p := page{SkillName: name, Iteration: iteration}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "eval-") {
			continue
		}
		evalDir := filepath.Join(iterDir, e.Name())
		metaRaw, err := os.ReadFile(filepath.Join(evalDir, "eval_metadata.json"))
		if err != nil {
			return "", err
		}
		var meta struct {
			ID   int    `json:"eval_id"`
			Name string `json:"eval_name"`
		}
		if err := json.Unmarshal(metaRaw, &meta); err != nil {
			return "", err
		}
		ev := evalView{ID: meta.ID, Name: meta.Name}
		with := readConfig(evalDir, "with_skill")
		without := readConfig(evalDir, "without_skill")
		ev.Configs = []configView{with, without}
		ev.WithLabel = label(with)
		ev.WithoutLabel = label(without)
		p.Evals = append(p.Evals, ev)
	}

	benchPath := filepath.Join(iterDir, "benchmark.json")
	if raw, err := os.ReadFile(benchPath); err == nil {
		var b eval.Benchmark
		if err := json.Unmarshal(raw, &b); err != nil {
			return "", app.WithPath(app.Fail("validation", "invalid benchmark.json"), benchPath)
		}
		p.HasBenchmark = true
		p.WithPct = b.Summary.WithSkillPassRate * 100
		p.WithoutPct = b.Summary.WithoutSkillPassRate * 100
		p.Gaps = b.Gaps
	}

	tmplSrc, err := assets.FS.ReadFile("report/report.html.tmpl")
	if err != nil {
		return "", err
	}
	tmpl, err := template.New("report").Parse(string(tmplSrc))
	if err != nil {
		return "", err
	}
	outPath := filepath.Join(iterDir, "report.html")
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := tmpl.Execute(f, p); err != nil {
		return "", err
	}
	return outPath, nil
}

func readConfig(evalDir, name string) configView {
	cfg := configView{Name: name}
	outDir := filepath.Join(evalDir, name, "outputs")
	_ = filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(outDir, path)
		if rel != ".gitkeep" {
			cfg.Files = append(cfg.Files, rel)
		}
		return nil
	})
	if t, err := os.ReadFile(filepath.Join(evalDir, name, "transcript.md")); err == nil {
		cfg.Transcript = string(t)
	}
	if g, err := os.ReadFile(filepath.Join(evalDir, name, "grading.json")); err == nil {
		var grading eval.Grading
		if json.Unmarshal(g, &grading) == nil {
			cfg.Expectations = grading.Expectations
		}
	} else {
		cfg.Partial = true
	}
	return cfg
}

func label(c configView) string {
	if c.Partial || len(c.Expectations) == 0 && c.Transcript == "" && len(c.Files) == 0 {
		if c.Partial && len(c.Expectations) == 0 {
			return "—"
		}
	}
	if len(c.Expectations) == 0 {
		return "no assertions"
	}
	pass := 0
	for _, e := range c.Expectations {
		if e.Passed {
			pass++
		}
	}
	return fmt.Sprintf("%d/%d", pass, len(c.Expectations))
}
