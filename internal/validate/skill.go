package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hz/skillctl/internal/app"
	"gopkg.in/yaml.v3"
)

const maxBodyLines = 500

var allowedKeys = map[string]struct{}{
	"name":          {},
	"description":   {},
	"license":       {},
	"allowed-tools": {},
	"metadata":      {},
	"compatibility": {},
}

var kebabName = regexp.MustCompile(`^[a-z0-9-]+$`)

type Result struct {
	OK       bool
	Warnings []string
	Err      *app.Error
	Name     string
}

func Skill(path string, strict bool) Result {
	skillMD := filepath.Join(path, "SKILL.md")
	raw, err := os.ReadFile(skillMD)
	if err != nil {
		if os.IsNotExist(err) {
			return fail(app.WithPath(app.Fail("validation", "SKILL.md not found"), path))
		}
		return fail(app.WithPath(app.Fail("validation", err.Error()), path))
	}

	fm, body, ferr := parseFrontmatter(string(raw))
	if ferr != nil {
		return fail(app.WithPath(ferr, skillMD))
	}

	unexpected := make([]string, 0)
	for k := range fm {
		if _, ok := allowedKeys[k]; !ok {
			unexpected = append(unexpected, k)
		}
	}
	if len(unexpected) > 0 {
		return fail(app.WithPath(app.Fail("validation",
			fmt.Sprintf("Unexpected key(s) in SKILL.md frontmatter: %s. Allowed properties are: allowed-tools, compatibility, description, license, metadata, name",
				strings.Join(sorted(unexpected), ", "))), skillMD))
	}

	name, errMsg := requireString(fm, "name")
	if errMsg != "" {
		return fail(app.WithPath(app.Fail("validation", errMsg), skillMD))
	}
	if err := CheckName(name); err != nil {
		return fail(app.WithPath(err, skillMD))
	}
	name = strings.TrimSpace(name)

	desc, errMsg := requireString(fm, "description")
	if errMsg != "" {
		return fail(app.WithPath(app.Fail("validation", errMsg), skillMD))
	}
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return fail(app.WithPath(app.Fail("validation", "Missing 'description' in frontmatter"), skillMD))
	}
	if strings.ContainsAny(desc, "<>") {
		return fail(app.WithPath(app.Fail("validation", "Description cannot contain angle brackets (< or >)"), skillMD))
	}
	if len(desc) > 1024 {
		return fail(app.WithPath(app.Fail("validation",
			fmt.Sprintf("Description is too long (%d characters). Maximum is 1024 characters.", len(desc))), skillMD))
	}

	if rawCompat, ok := fm["compatibility"]; ok && rawCompat != nil && rawCompat != "" {
		compat, ok := rawCompat.(string)
		if !ok {
			return fail(app.WithPath(app.Fail("validation",
				fmt.Sprintf("Compatibility must be a string, got %T", rawCompat)), skillMD))
		}
		if len(compat) > 500 {
			return fail(app.WithPath(app.Fail("validation",
				fmt.Sprintf("Compatibility is too long (%d characters). Maximum is 500 characters.", len(compat))), skillMD))
		}
	}

	var warnings []string
	bodyLines := countBodyLines(body)
	if bodyLines > maxBodyLines {
		msg := fmt.Sprintf("SKILL.md body is %d lines (recommended maximum is %d)", bodyLines, maxBodyLines)
		if strict {
			return fail(app.WithPath(app.Fail("validation", msg), skillMD))
		}
		warnings = append(warnings, msg)
	}

	return Result{OK: true, Warnings: warnings, Name: name}
}

func fail(err *app.Error) Result {
	return Result{OK: false, Err: err}
}

func CheckName(name string) *app.Error {
	name = strings.TrimSpace(name)
	if name == "" {
		return app.Fail("validation", "Missing 'name' in frontmatter")
	}
	if !kebabName.MatchString(name) {
		return app.Fail("validation",
			fmt.Sprintf("Name '%s' should be kebab-case (lowercase letters, digits, and hyphens only)", name))
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || strings.Contains(name, "--") {
		return app.Fail("validation",
			fmt.Sprintf("Name '%s' cannot start/end with hyphen or contain consecutive hyphens", name))
	}
	if len(name) > 64 {
		return app.Fail("validation",
			fmt.Sprintf("Name is too long (%d characters). Maximum is 64 characters.", len(name)))
	}
	return nil
}

func requireString(fm map[string]any, key string) (string, string) {
	v, ok := fm[key]
	if !ok {
		return "", fmt.Sprintf("Missing '%s' in frontmatter", key)
	}
	s, ok := v.(string)
	if !ok {
		label := key
		if key != "" {
			label = strings.ToUpper(key[:1]) + key[1:]
		}
		return "", fmt.Sprintf("%s must be a string, got %T", label, v)
	}
	return s, ""
}

func parseFrontmatter(content string) (map[string]any, string, *app.Error) {
	if !strings.HasPrefix(content, "---") {
		return nil, "", app.Fail("validation", "No YAML frontmatter found")
	}
	rest := strings.TrimPrefix(content, "---")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, "", app.Fail("validation", "Invalid frontmatter format")
	}
	yamlText := rest[:end]
	body := rest[end+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(yamlText), &fm); err != nil {
		return nil, "", app.Fail("validation", "Invalid YAML in frontmatter: "+err.Error())
	}
	if fm == nil {
		fm = map[string]any{}
	}
	return fm, body, nil
}

func countBodyLines(body string) int {
	if body == "" {
		return 0
	}
	n := strings.Count(body, "\n")
	if !strings.HasSuffix(body, "\n") {
		n++
	}
	return n
}

func sorted(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
