package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bashfulrobot/walkr/internal/library"
)

// maxDiffLines caps how many out-of-date files --check prints.
const maxDiffLines = 40

func buildAllCmd() *cobra.Command {
	var perSite, check bool
	cmd := &cobra.Command{
		Use:   "build-all [root]",
		Short: "Build every walkthrough under a directory tree, sharing one copy of the assets",
		Long: "Finds every .walkr directory (containing steps/) under root, default the current\n" +
			"directory, and builds each site into a site/ directory beside it. All sites share\n" +
			"one copy of the vendored libraries and stylesheet in <root>/_walkr, which keeps a\n" +
			"large collection small. Each site/ directory is regenerated in full.\n\n" +
			"With --check nothing is written. It exits non-zero if any committed site or the\n" +
			"shared assets differ from a fresh build, so CI or a hook can catch a stale site.\n" +
			"With --per-site every site carries its own assets, like walkr build.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Failures here are results, not usage mistakes, and main prints the error.
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			opt := library.Options{PerSite: perSite}
			if check {
				return runCheck(cmd, root, opt)
			}
			return runBuildAll(cmd, root, opt)
		},
	}
	cmd.Flags().BoolVar(&perSite, "per-site", false, "give every site its own copy of the assets")
	cmd.Flags().BoolVar(&check, "check", false, "write nothing, fail if any committed site is out of date")
	return cmd
}

func runBuildAll(cmd *cobra.Command, root string, opt library.Options) error {
	rep, err := library.BuildAll(root, root, opt)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	for _, f := range rep.Failures {
		fmt.Fprintf(cmd.ErrOrStderr(), "failed  %s: %v\n", f.Rel, f.Err)
	}
	if opt.PerSite {
		fmt.Fprintf(out, "built %d site(s) under %s\n", rep.Built, root)
	} else {
		fmt.Fprintf(out, "built %d site(s) under %s, shared assets in %s\n", rep.Built, root, filepath.Join(root, library.SharedDir))
	}
	if len(rep.Failures) > 0 {
		return fmt.Errorf("%d tutorial(s) failed to build", len(rep.Failures))
	}
	return nil
}

func runCheck(cmd *cobra.Command, root string, opt library.Options) error {
	res, err := library.Check(root, opt)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	for _, f := range res.Failures {
		fmt.Fprintf(cmd.ErrOrStderr(), "failed  %s: %v\n", f.Rel, f.Err)
	}
	for i, d := range res.Diffs {
		if i == maxDiffLines {
			fmt.Fprintf(out, "... and %d more\n", len(res.Diffs)-maxDiffLines)
			break
		}
		fmt.Fprintf(out, "%-8s %s\n", d.Kind, d.Path)
	}
	if res.OK() {
		fmt.Fprintln(out, "all sites are up to date")
		return nil
	}
	return fmt.Errorf("%d file(s) out of date and %d tutorial(s) failed, run: walkr build-all %s", len(res.Diffs), len(res.Failures), root)
}
