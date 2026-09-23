// Package config loads walkr's single global configuration file. The file
// holds pointers to secrets (op:// references, env var names), never secret
// values: the parser rejects unknown keys, so a literal `token:` is an error.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultServiceAccountTokenEnv names the env var holding the 1Password
// service account token when the config does not override it.
const DefaultServiceAccountTokenEnv = "OP_SERVICE_ACCOUNT_TOKEN"

// Config is the whole global file.
type Config struct {
	Confluence Confluence `yaml:"confluence"`
}

// Confluence is the publishing configuration.
type Confluence struct {
	Site    string            `yaml:"site"` // host of the Confluence site, e.g. example.atlassian.net
	CloudID string            `yaml:"cloud_id"`
	Email   string            `yaml:"email"`
	Auth    Auth              `yaml:"auth"`
	Targets map[string]Target `yaml:"targets"`
}

// PageURL is the browser URL of a page, which stays valid if the page is
// renamed or moved.
func (c Confluence) PageURL(id string) string {
	return "https://" + c.Site + "/wiki/pages/viewpage.action?pageId=" + id
}

// Auth says where the API token comes from. Neither field is a secret.
type Auth struct {
	// ServiceAccountTokenEnv is the env var holding the 1Password service
	// account token. Defaults to DefaultServiceAccountTokenEnv.
	ServiceAccountTokenEnv string `yaml:"service_account_token_env"`
	// TokenRef is the op:// reference of the Atlassian API token.
	TokenRef string `yaml:"token_ref"`
}

// Target is one publishable walkthrough.
type Target struct {
	Dir      string `yaml:"dir"`
	SpaceKey string `yaml:"space_key"`
	ParentID string `yaml:"parent_id"` // where the section (or, without one, the tutorial page) goes
	Section  string `yaml:"section"`   // optional section page, created under parent_id if absent
	Diagrams string `yaml:"diagrams"`  // "png" or "source"
}

// DefaultPath returns $XDG_CONFIG_HOME/walkr/config.yaml, falling back to
// ~/.config/walkr/config.yaml. It is the same on every OS so the file lives
// where dotfile managers expect it.
func DefaultPath() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "walkr", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate config: %w", err)
	}
	return filepath.Join(home, ".config", "walkr", "config.yaml"), nil
}

// Load reads and validates the file at path.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file %s not found", path)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	return Parse(raw, path)
}

// Parse decodes and validates raw YAML. name is used in error messages.
func Parse(raw []byte, name string) (*Config, error) {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	var c Config
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if c.Confluence.Auth.ServiceAccountTokenEnv == "" {
		c.Confluence.Auth.ServiceAccountTokenEnv = DefaultServiceAccountTokenEnv
	}
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return &c, nil
}

func (c *Config) validate() error {
	a := c.Confluence.Auth
	if a.TokenRef != "" && !strings.HasPrefix(a.TokenRef, "op://") {
		return errors.New("confluence.auth.token_ref must be an op:// reference, not a token value")
	}
	for name, t := range c.Confluence.Targets {
		switch t.Diagrams {
		case "", "png", "source":
		default:
			return fmt.Errorf("confluence.targets.%s.diagrams must be \"png\" or \"source\", got %q", name, t.Diagrams)
		}
	}
	return nil
}

// RequireConfluence checks the fields every Confluence command needs.
func (c *Config) RequireConfluence() error {
	switch {
	case c.Confluence.Site == "":
		return errors.New("confluence.site is not set")
	case c.Confluence.CloudID == "":
		return errors.New("confluence.cloud_id is not set")
	case c.Confluence.Email == "":
		return errors.New("confluence.email is not set")
	case c.Confluence.Auth.TokenRef == "":
		return errors.New("confluence.auth.token_ref is not set")
	}
	return nil
}

// Target returns the named target, or an error listing the valid names.
func (c *Config) Target(name string) (Target, error) {
	t, ok := c.Confluence.Targets[name]
	if !ok {
		names := make([]string, 0, len(c.Confluence.Targets))
		for n := range c.Confluence.Targets {
			names = append(names, n)
		}
		return Target{}, fmt.Errorf("no confluence target %q (have: %s)", name, strings.Join(names, ", "))
	}
	return t, nil
}
