package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bashfulrobot/walkr/internal/config"
	"github.com/bashfulrobot/walkr/internal/confluence"
	"github.com/bashfulrobot/walkr/internal/diagram"
	"github.com/bashfulrobot/walkr/internal/secrets"
	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

// version is reported to 1Password as the integration version. Set with
// -ldflags "-X main.version=...".
var version = "dev"

func confluenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "confluence",
		Short: "Work with Confluence publishing",
	}
	cmd.AddCommand(confluenceCheckCmd(), confluencePublishCmd())
	return cmd
}

func confluenceCheckCmd() *cobra.Command {
	var cfgPath, target string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Verify the configured credentials can read Confluence",
		Long: "Resolves the API token from 1Password through the service account named in the\n" +
			"config, then makes read-only calls. Prints no secret values.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cfgPath)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			res, err := newResolver(ctx, cfg)
			if err != nil {
				return err
			}
			return confluence.Check(ctx, cmd.OutOrStdout(), cfg, target, res)
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config", "", "config file (default $XDG_CONFIG_HOME/walkr/config.yaml)")
	cmd.Flags().StringVar(&target, "target", "", "also check this target's parent page")
	return cmd
}

func loadConfig(path string) (*config.Config, error) {
	if path == "" {
		var err error
		if path, err = config.DefaultPath(); err != nil {
			return nil, err
		}
	}
	return config.Load(path)
}

func newResolver(ctx context.Context, cfg *config.Config) (secrets.Resolver, error) {
	env := cfg.Confluence.Auth.ServiceAccountTokenEnv
	tok := os.Getenv(env)
	if tok == "" {
		return nil, fmt.Errorf("environment variable %s (1Password service account token) is not set", env)
	}
	return secrets.NewOnePassword(ctx, secrets.New(tok), version)
}

func confluencePublishCmd() *cobra.Command {
	var cfgPath, target string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish a walkthrough to Confluence as a tutorial page with one child page per step",
		Long: "Publishes the walkthrough in the target's dir under the target's section page.\n" +
			"Pages are found by label, then by title, and updated in place, so running it\n" +
			"again never duplicates pages. Edits made in Confluence are overwritten.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if target == "" {
				return fmt.Errorf("--target is required")
			}
			cfg, err := loadConfig(cfgPath)
			if err != nil {
				return err
			}
			if err := cfg.RequireConfluence(); err != nil {
				return err
			}
			tgt, err := cfg.Target(target)
			if err != nil {
				return err
			}
			switch {
			case tgt.Dir == "":
				return fmt.Errorf("confluence.targets.%s.dir is not set", target)
			case tgt.SpaceKey == "":
				return fmt.Errorf("confluence.targets.%s.space_key is not set", target)
			case tgt.ParentID == "":
				return fmt.Errorf("confluence.targets.%s.parent_id is not set", target)
			}
			dir, err := expandHome(tgt.Dir)
			if err != nil {
				return err
			}
			wt, err := walkthrough.Load(dir)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			res, err := newResolver(ctx, cfg)
			if err != nil {
				return err
			}
			token, err := res.Resolve(ctx, cfg.Confluence.Auth.TokenRef)
			if err != nil {
				return err
			}
			client := confluence.New(cfg.Confluence.CloudID, cfg.Confluence.Email, token)

			pub := &confluence.Publisher{
				Client: client, Site: cfg.Confluence, Name: target, Target: tgt, DryRun: dryRun,
			}
			if !dryRun && tgt.Diagrams != "source" {
				chrome, err := diagram.NewChrome()
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v, diagrams will show as source\n", err)
				} else {
					defer chrome.Close()
					pub.Diagrams = chrome
				}
			}
			result, err := pub.Publish(ctx, wt)
			if err != nil {
				return err
			}
			printPublishResult(cmd, result, dryRun)
			return nil
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config", "", "config file (default $XDG_CONFIG_HOME/walkr/config.yaml)")
	cmd.Flags().StringVar(&target, "target", "", "target name from the config (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "look pages up and report what would be created, writing nothing")
	return cmd
}

func printPublishResult(cmd *cobra.Command, r *confluence.Result, dryRun bool) {
	out := cmd.OutOrStdout()
	for _, p := range r.Pages {
		verb := "updated"
		switch {
		case p.Key == "section" && !p.Created:
			verb = "exists"
		case p.Created && dryRun:
			verb = "would create"
		case p.Created:
			verb = "created"
		case dryRun:
			verb = "would update"
		}
		fmt.Fprintf(out, "%-13s %s\n", verb, p.Title)
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
	}
	if !dryRun {
		fmt.Fprintf(out, "\nstart here: %s\n", r.Tutorial().URL)
	}
}

func expandHome(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(p, "~")), nil
	}
	return p, nil
}
