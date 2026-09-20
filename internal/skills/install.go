package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hz/skillctl/assets"
	"github.com/hz/skillctl/internal/app"
)

func InstallMeta(cwd, target string) ([]string, error) {
	dests, err := destinations(cwd, target)
	if err != nil {
		return nil, err
	}
	var written []string
	for _, destRoot := range dests {
		for _, name := range assets.MetaSkillNames {
			src := filepath.Join("meta-skills", name)
			err := fs.WalkDir(assets.FS, src, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(src, path)
				if err != nil {
					return err
				}
				out := filepath.Join(destRoot, name, rel)
				if d.IsDir() {
					return os.MkdirAll(out, 0o755)
				}
				data, err := assets.FS.ReadFile(path)
				if err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(out, data, 0o644); err != nil {
					return err
				}
				written = append(written, out)
				return nil
			})
			if err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

func destinations(cwd, target string) ([]string, error) {
	switch target {
	case "", "both":
		return []string{
			filepath.Join(cwd, ".cursor", "skills"),
			filepath.Join(cwd, ".github", "skills"),
		}, nil
	case "cursor":
		return []string{filepath.Join(cwd, ".cursor", "skills")}, nil
	case "copilot":
		return []string{filepath.Join(cwd, ".github", "skills")}, nil
	default:
		return nil, app.Usage(fmt.Sprintf("invalid --target %q (cursor|copilot|both)", target))
	}
}
