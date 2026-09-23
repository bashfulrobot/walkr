package confluence

import (
	"context"
	"fmt"
	"io"

	"github.com/bashfulrobot/walkr/internal/config"
	"github.com/bashfulrobot/walkr/internal/secrets"
)

// Check resolves the API token, proves it authenticates, and, when a target
// is named, that its parent page is readable. Output contains no secrets.
func Check(ctx context.Context, w io.Writer, cfg *config.Config, target string, res secrets.Resolver, opts ...Option) error {
	if err := cfg.RequireConfluence(); err != nil {
		return err
	}
	var tgt config.Target
	if target != "" {
		var err error
		if tgt, err = cfg.Target(target); err != nil {
			return err
		}
	}

	token, err := res.Resolve(ctx, cfg.Confluence.Auth.TokenRef)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "token:   resolved %s\n", cfg.Confluence.Auth.TokenRef)

	c := New(cfg.Confluence.CloudID, cfg.Confluence.Email, token, opts...)
	spaces, err := c.Spaces(ctx, 1)
	if err != nil {
		return fmt.Errorf("read spaces: %w", err)
	}
	fmt.Fprintf(w, "spaces:  ok (%d returned)\n", len(spaces))

	if tgt.ParentID != "" {
		p, err := c.Content(ctx, tgt.ParentID)
		if err != nil {
			return fmt.Errorf("read parent page %s: %w", tgt.ParentID, err)
		}
		fmt.Fprintf(w, "parent:  ok, %q (%s, v%d)\n", p.Title, p.Status, p.Version.Number)
	}
	return nil
}
