package assets

import "embed"

//go:embed meta-skills sop.md report
var FS embed.FS

var MetaSkillNames = []string{"skill-author", "eval-runner", "skill-grader"}
