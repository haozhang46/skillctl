package agents

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hz/skillctl/assets"
	"github.com/hz/skillctl/internal/app"
)

const (
	StartMarker = "<!-- skillctl:sop:start -->"
	EndMarker   = "<!-- skillctl:sop:end -->"
)

var discovery = []string{
	"agent.md",
	"AGENTS.md",
	".github/copilot-instructions.md",
}

func Inject(cwd, file string, force bool) (string, error) {
	sop, err := assets.FS.ReadFile("sop.md")
	if err != nil {
		return "", err
	}
	block := StartMarker + "\n" + strings.TrimSpace(string(sop)) + "\n" + EndMarker + "\n"

	target, err := resolveFile(cwd, file)
	if err != nil {
		return "", err
	}
	var existing []byte
	if raw, err := os.ReadFile(target); err == nil {
		existing = raw
	} else if !os.IsNotExist(err) {
		return "", err
	}

	next, mergeErr := merge(string(existing), block, force)
	if mergeErr != nil {
		return "", app.WithPath(mergeErr, target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, []byte(next), 0o644); err != nil {
		return "", err
	}
	return target, nil
}

func resolveFile(cwd, file string) (string, error) {
	if file != "" {
		if filepath.IsAbs(file) {
			return file, nil
		}
		return filepath.Join(cwd, file), nil
	}
	for _, rel := range discovery {
		p := filepath.Join(cwd, rel)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return filepath.Join(cwd, discovery[0]), nil
}

func merge(existing, block string, force bool) (string, *app.Error) {
	startCount := strings.Count(existing, StartMarker)
	endCount := strings.Count(existing, EndMarker)
	if startCount == 0 && endCount == 0 {
		if existing == "" {
			return block, nil
		}
		if !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		return existing + "\n" + block, nil
	}
	if startCount != 1 || endCount != 1 {
		if !force {
			return "", app.WithHint(
				app.Fail("corruption", "SOP markers are missing or duplicated"),
				"fix the file or pass --force to replace the marked region",
			)
		}
		stripped := stripAll(existing)
		if stripped != "" && !strings.HasSuffix(stripped, "\n") {
			stripped += "\n"
		}
		return stripped + block, nil
	}
	start := strings.Index(existing, StartMarker)
	end := strings.Index(existing, EndMarker)
	if start > end {
		if !force {
			return "", app.WithHint(
				app.Fail("corruption", "SOP end marker appears before start marker"),
				"fix the file or pass --force",
			)
		}
		return strings.TrimSpace(stripAll(existing)) + "\n" + block, nil
	}
	end += len(EndMarker)
	return existing[:start] + strings.TrimSuffix(block, "\n") + existing[end:], nil
}

func stripAll(s string) string {
	for {
		start := strings.Index(s, StartMarker)
		end := strings.Index(s, EndMarker)
		if start < 0 && end < 0 {
			return strings.TrimSpace(s) + "\n"
		}
		if start >= 0 && end >= 0 && start < end {
			s = s[:start] + s[end+len(EndMarker):]
			continue
		}
		if start >= 0 {
			s = s[:start]
			continue
		}
		if end >= 0 {
			s = s[:end] + s[end+len(EndMarker):]
		}
	}
}
