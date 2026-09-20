package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hz/skillctl/internal/agents"
	"github.com/hz/skillctl/internal/app"
	"github.com/hz/skillctl/internal/create"
	"github.com/hz/skillctl/internal/eval"
	"github.com/hz/skillctl/internal/pack"
	"github.com/hz/skillctl/internal/report"
	"github.com/hz/skillctl/internal/skills"
	"github.com/hz/skillctl/internal/validate"
	"github.com/spf13/cobra"
)

type runtime struct {
	stdout io.Writer
	stderr io.Writer
	cwd    string
	json   bool
}

func Execute() int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return app.ExitFail
	}
	return Run(os.Args[1:], os.Stdout, os.Stderr, cwd)
}

func Run(args []string, stdout, stderr io.Writer, cwd string) int {
	rt := &runtime{stdout: stdout, stderr: stderr, cwd: cwd}
	cmd := newRoot(rt)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if err := cmd.Execute(); err != nil {
		return finish(rt, err)
	}
	return app.ExitOK
}

func newRoot(rt *runtime) *cobra.Command {
	root := &cobra.Command{
		Use:           "skillctl",
		Short:         "Create, validate, package, and evaluate Agent Skills for Cursor and VS Code Copilot",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().BoolVar(&rt.json, "json", false, "machine-readable JSON on stdout")

	root.AddCommand(
		newCreate(rt),
		newValidate(rt),
		newPackage(rt),
		newEval(rt),
		newSkills(rt),
		newAgents(rt),
	)
	return root
}

func newCreate(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Scaffold a skill directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := create.Skill(rt.cwd, args[0])
			if err != nil {
				return err
			}
			return rt.ok(map[string]any{"path": path}, "Created skill at %s\n", path)
		},
	}
}

func newValidate(rt *runtime) *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "validate <path>",
		Short: "Static checks aligned with skill-creator quick_validate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := abs(rt.cwd, args[0])
			res := validate.Skill(path, strict)
			if !res.OK {
				return res.Err
			}
			if rt.json {
				extra := map[string]any{"path": path, "name": res.Name}
				if len(res.Warnings) > 0 {
					extra["warnings"] = res.Warnings
				}
				return app.WriteJSON(rt.stdout, app.Success(extra))
			}
			fmt.Fprintf(rt.stdout, "Skill is valid!\n")
			for _, w := range res.Warnings {
				fmt.Fprintf(rt.stderr, "warning: %s\n", w)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "treat warnings as errors")
	return cmd
}

func newPackage(rt *runtime) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "package <path>",
		Short: "Validate then write a .skill zip",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outDir := output
			if outDir == "" {
				outDir = rt.cwd
			} else if !filepath.IsAbs(outDir) {
				outDir = filepath.Join(rt.cwd, outDir)
			}
			path, err := pack.Skill(abs(rt.cwd, args[0]), outDir)
			if err != nil {
				return err
			}
			return rt.ok(map[string]any{"output": path}, "Packaged %s\n", path)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "output directory (default: current directory)")
	return cmd
}

func newEval(rt *runtime) *cobra.Command {
	var iteration int
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Eval workspace protocol (prepare, aggregate, report)",
	}
	cmd.PersistentFlags().IntVar(&iteration, "iteration", 1, "iteration number (N in iteration-N)")

	cmd.AddCommand(&cobra.Command{
		Use:   "init <skill>",
		Short: "Ensure evals/evals.json skeleton exists",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := eval.Init(abs(rt.cwd, args[0]))
			if err != nil {
				return err
			}
			return rt.ok(map[string]any{"path": path}, "Wrote %s\n", path)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "prepare <skill>",
		Short: "Create .skillbench prompt packs and empty output dirs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := eval.Prepare(abs(rt.cwd, args[0]), iteration, rt.cwd)
			if err != nil {
				return err
			}
			return rt.ok(map[string]any{"path": path, "iteration": iteration}, "Prepared %s\n", path)
		},
	})
	var allowPartial bool
	agg := &cobra.Command{
		Use:   "aggregate <skill>",
		Short: "Read grading.json files into benchmark.json and benchmark.md",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := eval.Aggregate(abs(rt.cwd, args[0]), iteration, rt.cwd, allowPartial)
			if err != nil {
				return err
			}
			return rt.ok(map[string]any{"path": path, "iteration": iteration}, "Aggregated %s\n", path)
		},
	}
	agg.Flags().BoolVar(&allowPartial, "allow-partial", false, "skip missing grading.json and record gaps")
	cmd.AddCommand(agg)
	cmd.AddCommand(&cobra.Command{
		Use:   "report <skill>",
		Short: "Write static HTML report (Outputs + Benchmark)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := report.Write(abs(rt.cwd, args[0]), iteration, rt.cwd)
			if err != nil {
				return err
			}
			return rt.ok(map[string]any{"path": path, "iteration": iteration}, "Wrote %s\n", path)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "trigger",
		Short: "Reserved: trigger evaluation (not implemented in v1)",
		Long:  "Reserved for future work. v1 does not implement skillctl eval trigger. Use eval-runner in Cursor/Copilot instead.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Fail("unimplemented", "eval trigger is not implemented in v1; use eval-runner in the IDE")
		},
	})
	return cmd
}

func newSkills(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage bundled meta-skills",
	}
	var target string
	inst := &cobra.Command{
		Use:   "install-meta",
		Short: "Install skill-author, eval-runner, and skill-grader",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			written, err := skills.InstallMeta(rt.cwd, target)
			if err != nil {
				return err
			}
			return rt.ok(map[string]any{"written": written, "target": target}, "Installed %d meta-skill files\n", len(written))
		},
	}
	inst.Flags().StringVar(&target, "target", "both", "cursor, copilot, or both")
	cmd.AddCommand(inst)
	return cmd
}

func newAgents(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Agent instruction file helpers",
	}
	var file string
	var force bool
	inj := &cobra.Command{
		Use:   "inject",
		Short: "Idempotently insert the skillctl SOP into an agent instruction file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := agents.Inject(rt.cwd, file, force)
			if err != nil {
				return err
			}
			return rt.ok(map[string]any{"path": path}, "Updated %s\n", path)
		},
	}
	inj.Flags().StringVar(&file, "file", "", "instruction file (default: agent.md, then AGENTS.md, then .github/copilot-instructions.md)")
	inj.Flags().BoolVar(&force, "force", false, "replace corrupted SOP markers")
	cmd.AddCommand(inj)
	return cmd
}

func (rt *runtime) ok(extra map[string]any, format string, args ...any) error {
	if rt.json {
		return app.WriteJSON(rt.stdout, app.Success(extra))
	}
	fmt.Fprintf(rt.stdout, format, args...)
	return nil
}

func finish(rt *runtime, err error) int {
	ae := app.AsError(err)
	if ae.Kind == "usage" || isUsage(err) {
		if ae.Kind != "usage" {
			ae = app.Usage(err.Error())
		}
	}
	if rt.json {
		_ = app.WriteJSON(rt.stdout, app.Failure(ae))
	} else {
		fmt.Fprintln(rt.stderr, ae.Error())
		if ae.Hint != "" {
			fmt.Fprintf(rt.stderr, "hint: %s\n", ae.Hint)
		}
	}
	if ae.ExitCode == 0 {
		return app.ExitFail
	}
	return ae.ExitCode
}

func isUsage(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return containsAny(s, "arg(s)", "unknown command", "unknown flag", "required flag", "accepts")
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func abs(cwd, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(cwd, p)
}
