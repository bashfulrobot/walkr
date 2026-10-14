// Command walkr renders a hand-authored markdown walkthrough into an
// interactive, wizard-style static site. See docs/ai/content-format.md for the
// authoring format and extras/prompt/ai-prompt.md for the project brief.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/bashfulrobot/walkr/internal/site"
	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "walkr",
		Short: "Render a hand-authored markdown walkthrough into an interactive teaching site",
	}
	root.AddCommand(buildCmd(), buildAllCmd(), serveCmd(), initCmd(), confluenceCmd())
	return root
}

// themeFlagHelp is the --theme description shared by build, build-all and serve.
var themeFlagHelp = "page theme: " + strings.Join(site.ThemeNames(), " or ") + " (default: the walkthrough.yaml theme, then " + site.DefaultTheme + ")"

// checkTheme rejects an unknown --theme value before any work starts.
func checkTheme(name string) error {
	_, err := site.ResolveTheme(name, "")
	return err
}

func buildCmd() *cobra.Command {
	var outDir, theme string
	cmd := &cobra.Command{
		Use:   "build [dir]",
		Short: "Render a walkthrough directory into a self-contained static site",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := ".walkr"
			if len(args) == 1 {
				dir = args[0]
			}
			if err := checkTheme(theme); err != nil {
				return err
			}
			wt, err := walkthrough.Load(dir)
			if err != nil {
				return err
			}
			if err := site.BuildWith(wt, outDir, site.Options{Theme: theme}); err != nil {
				return err
			}
			fmt.Printf("built %d step(s) -> %s\n", len(wt.Steps), outDir)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outDir, "out", "o", "site", "output directory")
	cmd.Flags().StringVar(&theme, "theme", "", themeFlagHelp)
	return cmd
}

// serveDir serves dir on ln until ctx is cancelled, then shuts the server down
// so the caller can remove dir. A clean shutdown returns nil.
func serveDir(ctx context.Context, ln net.Listener, dir string) error {
	srv := &http.Server{Handler: http.FileServer(http.Dir(dir))}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func serveCmd() *cobra.Command {
	var port int
	var open bool
	var theme string
	cmd := &cobra.Command{
		Use:   "serve [dir]",
		Short: "Build to a temp dir and serve it locally for preview",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := ".walkr"
			if len(args) == 1 {
				dir = args[0]
			}
			if err := checkTheme(theme); err != nil {
				return err
			}
			wt, err := walkthrough.Load(dir)
			if err != nil {
				return err
			}
			tmp, err := os.MkdirTemp("", "walkr-serve-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)
			if err := site.BuildWith(wt, tmp, site.Options{Theme: theme}); err != nil {
				return err
			}
			addr := fmt.Sprintf("127.0.0.1:%d", port)
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}
			url := "http://" + addr + "/"
			fmt.Printf("serving %s at %s (ctrl-c to stop)\n", tmp, url)
			if open {
				go openBrowser(url)
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return serveDir(ctx, ln, tmp)
		},
	}
	cmd.Flags().IntVar(&port, "port", 4400, "port to serve on")
	cmd.Flags().BoolVar(&open, "open", false, "open the default browser")
	cmd.Flags().StringVar(&theme, "theme", "", themeFlagHelp)
	return cmd
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Scaffold an empty walkthrough",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := ".walkr"
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
	manifest := "title: Walkr\ntagline: Field Guide\nrepo: org/repo\n"
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
