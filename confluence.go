package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bashfulrobot/walkr/internal/config"
	"github.com/bashfulrobot/walkr/internal/confluence"
	"github.com/bashfulrobot/walkr/internal/secrets"
)

// version is reported to 1Password as the integration version. Set with
// -ldflags "-X main.version=...".
var version = "dev"

func confluenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "confluence",
		Short: "Work with Confluence publishing",
	}
	cmd.AddCommand(confluenceCheckCmd())
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
