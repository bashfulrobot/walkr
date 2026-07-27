// Command repo-walker renders a hand-authored markdown walkthrough into an
// interactive, wizard-style static site. See docs/content-format.md for the
// authoring format and extras/prompt/ai-prompt.md for the project brief.
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/bashfulrobot/repo-walker/internal/site"
	"github.com/bashfulrobot/repo-walker/internal/walkthrough"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "repo-walker",
		Short: "Render a hand-authored markdown walkthrough into an interactive teaching site",
	}
	root.AddCommand(buildCmd(), serveCmd(), initCmd())
	return root
}

func buildCmd() *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:   "build [dir]",
		Short: "Render a walkthrough directory into a self-contained static site",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := ".repo-walker"
			if len(args) == 1 {
				dir = args[0]
			}
			wt, err := walkthrough.Load(dir)
			if err != nil {
				return err
			}
			if err := site.Build(wt, outDir); err != nil {
				return err
			}
			fmt.Printf("built %d step(s) -> %s\n", len(wt.Steps), outDir)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outDir, "out", "o", "site", "output directory")
	return cmd
}

func serveCmd() *cobra.Command {
	var port int
	var open bool
	cmd := &cobra.Command{
		Use:   "serve [dir]",
		Short: "Build to a temp dir and serve it locally for preview",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := ".repo-walker"
			if len(args) == 1 {
				dir = args[0]
			}
			wt, err := walkthrough.Load(dir)
			if err != nil {
				return err
			}
			tmp, err := os.MkdirTemp("", "repo-walker-serve-*")
			if err != nil {
				return err
			}
			if err := site.Build(wt, tmp); err != nil {
				return err
			}
			addr := fmt.Sprintf("127.0.0.1:%d", port)
			url := "http://" + addr + "/"
			fmt.Printf("serving %s at %s (ctrl-c to stop)\n", tmp, url)
			if open {
				go openBrowser(url)
			}
			return http.ListenAndServe(addr, http.FileServer(http.Dir(tmp)))
		},
	}
	cmd.Flags().IntVar(&port, "port", 4400, "port to serve on")
	cmd.Flags().BoolVar(&open, "open", false, "open the default browser")
	return cmd
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Scaffold an empty walkthrough",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := ".repo-walker"
			if len(args) == 1 {
				dir = args[0]
			}
			return scaffold(dir)
		},
	}
	return cmd
}

func scaffold(dir string) error {
	for _, sub := range []string{"steps", "glossary", "assets"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return err
		}
	}
	manifest := "title: Repo Walker\ntagline: Field Guide\nrepo: org/repo\n"
	if err := writeIfAbsent(filepath.Join(dir, "walkthrough.yaml"), manifest); err != nil {
		return err
	}
	glossary := "example:\n  term: example\n  definition: Replace this with a real glossary entry.\n"
	if err := writeIfAbsent(filepath.Join(dir, "glossary.yaml"), glossary); err != nil {
		return err
	}
	step := "---\n" +
		"title: Overview\n" +
		"label: Overview\n" +
		"kind: Structure\n" +
		"order: 1\n" +
		"layout: overview\n" +
		"summary: One sentence describing this step.\n" +
		"---\n" +
		"Replace this with your first step's markdown body.\n"
	if err := writeIfAbsent(filepath.Join(dir, "steps", "01-overview.md"), step); err != nil {
		return err
	}
	fmt.Printf("scaffolded a walkthrough at %s\n", dir)
	return nil
}

func writeIfAbsent(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
