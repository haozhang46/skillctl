package pack

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hz/skillctl/internal/app"
	"github.com/hz/skillctl/internal/validate"
)

func Skill(skillPath, outputDir string) (string, error) {
	skillPath, err := filepath.Abs(skillPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(skillPath)
	if err != nil {
		return "", app.WithHint(
			app.WithPath(app.Fail("not_found", "skill folder not found"), skillPath),
			"pass a skill directory that contains SKILL.md",
		)
	}
	if !info.IsDir() {
		return "", app.WithPath(app.Fail("validation", "path is not a directory"), skillPath)
	}

	res := validate.Skill(skillPath, false)
	if !res.OK {
		return "", res.Err
	}

	if outputDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		outputDir = cwd
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}

	base := filepath.Base(skillPath)
	out := filepath.Join(outputDir, base+".skill")
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	parent := filepath.Dir(skillPath)
	err = filepath.Walk(skillPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(parent, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if shouldExclude(rel) {
			return nil
		}
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(w, in)
		return err
	})
	if err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}
	return out, f.Close()
}

func shouldExclude(rel string) bool {
	parts := strings.Split(rel, "/")
	for _, p := range parts {
		if p == "__pycache__" || p == "node_modules" {
			return true
		}
	}
	if len(parts) > 1 && parts[1] == "evals" {
		return true
	}
	name := parts[len(parts)-1]
	if name == ".DS_Store" || strings.HasSuffix(name, ".pyc") {
		return true
	}
	return false
}
