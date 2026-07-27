// Package walkthrough loads a .repo-walker/ directory into memory: the
// optional global manifest, the glossary, and every step's frontmatter+body.
package walkthrough

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest is walkthrough.yaml. Every field here is consumed by the rail
// header — see docs/content-format.md. Don't add a field the UI doesn't render.
type Manifest struct {
	Title   string `yaml:"title"`
	Tagline string `yaml:"tagline"`
	Repo    string `yaml:"repo"`
}

// GlossaryEntry is one glossary.yaml value.
type GlossaryEntry struct {
	Term       string `yaml:"term"`
	Definition string `yaml:"definition"`
	LearnMore  string `yaml:"learn_more"`
}

// Glossary maps a def id to its entry.
type Glossary map[string]GlossaryEntry

// Layout is a step's render_mode. See docs/content-format.md.
type Layout string

const (
	LayoutOverview Layout = "overview"
	LayoutCodeWalk Layout = "code-walk"
	LayoutConfig   Layout = "config"
)

// Step is one steps/*.md file: frontmatter plus raw markdown body.
type Step struct {
	ID      string // derived from filename, without extension
	Title   string `yaml:"title"`
	Label   string `yaml:"label"`
	Kind    string `yaml:"kind"`
	Order   int    `yaml:"order"`
	Layout  Layout `yaml:"layout"`
	Summary string `yaml:"summary"`
	Body    string // raw markdown, frontmatter stripped
}

// Walkthrough is a fully loaded .repo-walker/ directory.
type Walkthrough struct {
	Manifest Manifest
	Glossary Glossary
	Steps    []Step
}

// Load reads dir (a .repo-walker/-shaped directory) into a Walkthrough.
func Load(dir string) (*Walkthrough, error) {
	wt := &Walkthrough{Glossary: Glossary{}}

	if b, err := os.ReadFile(filepath.Join(dir, "walkthrough.yaml")); err == nil {
		if err := yaml.Unmarshal(b, &wt.Manifest); err != nil {
			return nil, fmt.Errorf("walkthrough.yaml: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if wt.Manifest.Title == "" {
		wt.Manifest.Title = "Repo Walker"
	}
	if wt.Manifest.Tagline == "" {
		wt.Manifest.Tagline = "Field Guide"
	}
	if wt.Manifest.Repo == "" {
		wt.Manifest.Repo = filepath.Base(filepath.Dir(dir))
	}

	if b, err := os.ReadFile(filepath.Join(dir, "glossary.yaml")); err == nil {
		if err := yaml.Unmarshal(b, &wt.Glossary); err != nil {
			return nil, fmt.Errorf("glossary.yaml: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	stepsDir := filepath.Join(dir, "steps")
	entries, err := os.ReadDir(stepsDir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", stepsDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(stepsDir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		step, err := parseStep(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		step.ID = strings.TrimSuffix(e.Name(), ".md")
		if step.Title == "" {
			return nil, fmt.Errorf("%s: missing required frontmatter key 'title'", e.Name())
		}
		if step.Label == "" {
			return nil, fmt.Errorf("%s: missing required frontmatter key 'label'", e.Name())
		}
		if step.Layout == "" {
			return nil, fmt.Errorf("%s: missing required frontmatter key 'layout'", e.Name())
		}
		switch step.Layout {
		case LayoutOverview, LayoutCodeWalk, LayoutConfig:
		default:
			return nil, fmt.Errorf("%s: unknown layout %q (want overview, code-walk, or config)", e.Name(), step.Layout)
		}
		wt.Steps = append(wt.Steps, *step)
	}

	sort.SliceStable(wt.Steps, func(i, j int) bool {
		if wt.Steps[i].Order != wt.Steps[j].Order {
			return wt.Steps[i].Order < wt.Steps[j].Order
		}
		return wt.Steps[i].ID < wt.Steps[j].ID
	})

	if len(wt.Steps) == 0 {
		return nil, fmt.Errorf("%s: no steps found", stepsDir)
	}

	return wt, nil
}

const frontmatterDelim = "---"

// parseStep splits YAML frontmatter (between --- lines) from the markdown body.
func parseStep(raw []byte) (*Step, error) {
	s := string(raw)
	s = strings.TrimPrefix(s, "\uFEFF") // BOM
	if !strings.HasPrefix(strings.TrimLeft(s, "\n"), frontmatterDelim) {
		return nil, fmt.Errorf("missing YAML frontmatter (expected leading %q)", frontmatterDelim)
	}
	s = strings.TrimLeft(s, "\n")
	rest := s[len(frontmatterDelim):]
	idx := strings.Index(rest, "\n"+frontmatterDelim)
	if idx < 0 {
		return nil, fmt.Errorf("unterminated YAML frontmatter (no closing %q)", frontmatterDelim)
	}
	fmBlock := rest[:idx]
	body := rest[idx+len("\n"+frontmatterDelim):]
	body = strings.TrimPrefix(body, "\n")

	var step Step
	if err := yaml.Unmarshal([]byte(fmBlock), &step); err != nil {
		return nil, fmt.Errorf("invalid frontmatter: %w", err)
	}
	step.Body = body
	return &step, nil
}
