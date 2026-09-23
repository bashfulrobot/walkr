package confluence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/bashfulrobot/walkr/internal/config"
	"github.com/bashfulrobot/walkr/internal/render"
	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

// PageClient is the slice of the Confluence API the publisher uses. *Client
// implements it, and tests substitute a fake.
type PageClient interface {
	CreatePage(ctx context.Context, p NewPage) (*Content, error)
	UpdatePage(ctx context.Context, u PageUpdate) error
	AddLabels(ctx context.Context, pageID string, labels ...string) error
	FindByLabel(ctx context.Context, spaceKey, label string) ([]Content, error)
	FindByTitle(ctx context.Context, spaceKey, title string) (*Content, error)
	PutAttachment(ctx context.Context, pageID, filename string, data []byte) (*Attachment, error)
}

// DiagramRenderer turns Mermaid source into PNG bytes.
type DiagramRenderer interface {
	Render(ctx context.Context, mermaid string) ([]byte, error)
}

// Publisher publishes one walkthrough as a section page, a tutorial page, and
// one child page per step.
type Publisher struct {
	Client   PageClient
	Site     config.Confluence // supplies PageURL
	Name     string            // target name, used in labels
	Target   config.Target
	Diagrams DiagramRenderer // nil shows Mermaid source in an expand
	DryRun   bool            // look pages up but write nothing
}

// PageResult is one published page.
type PageResult struct {
	Key     string // "section", "index", or the step ID
	Title   string
	ID      string
	URL     string
	Created bool
}

// Result summarizes a publish run.
type Result struct {
	Pages    []PageResult
	Warnings []string
}

// Tutorial returns the tutorial page, the entry point to share.
func (r *Result) Tutorial() PageResult {
	for _, p := range r.Pages {
		if p.Key == "index" {
			return p
		}
	}
	return PageResult{}
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slug lowercases s and collapses anything outside a-z0-9 to single hyphens,
// which is what Confluence labels allow.
func Slug(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// StepTitle is a step's Confluence page title. Titles are unique per space, so
// the walkthrough title prefixes it to avoid collisions between walkthroughs.
func StepTitle(walkthroughTitle string, s walkthrough.Step) string {
	return walkthroughTitle + ": " + render.PlainTitle(s.Title)
}

// Publish creates or updates every page. Pages are found first by label, then
// by exact title, because the label search index lags creation by many seconds
// and a fast rerun must not duplicate pages. Page IDs from the create calls
// feed the second pass, so links never depend on search.
func (p *Publisher) Publish(ctx context.Context, wt *walkthrough.Walkthrough) (*Result, error) {
	res := &Result{}
	slug := Slug(p.Name)
	if slug == "" {
		return nil, fmt.Errorf("target name %q has no usable characters for labels", p.Name)
	}
	title := wt.Manifest.Title
	if title == "" {
		title = p.Name
	}

	parent := p.Target.ParentID
	if sec := p.Target.Section; sec != "" {
		page, err := p.ensure(ctx, res, "section", sec, "", parent, sectionBody(), nil)
		if err != nil {
			return nil, err
		}
		parent = page.ID
	}

	index, err := p.ensure(ctx, res, "index", title, "walkr-"+slug+"-index", parent,
		"<p>Publishing in progress.</p>", []string{"walkr-" + slug})
	if err != nil {
		return nil, err
	}

	steps := make([]*Content, len(wt.Steps))
	titles := make([]string, len(wt.Steps))
	ids := map[string]string{}
	for i, s := range wt.Steps {
		titles[i] = StepTitle(title, s)
		steps[i], err = p.ensure(ctx, res, s.ID, titles[i], "walkr-"+slug+"-"+Slug(s.ID), index.ID,
			"<p>Publishing in progress.</p>", []string{"walkr-" + slug})
		if err != nil {
			return nil, err
		}
		ids[s.ID] = steps[i].ID
	}
	if p.DryRun {
		return res, nil
	}

	stepURL := func(id string) (string, bool) {
		pid, ok := ids[id]
		return p.Site.PageURL(pid), ok
	}

	for i, s := range wt.Steps {
		opts := render.StorageOptions{Index: i + 1, Total: len(wt.Steps), StepURL: stepURL}
		if i > 0 {
			opts.Prev = &render.NavLink{Title: render.PlainTitle(wt.Steps[i-1].Title), URL: p.Site.PageURL(steps[i-1].ID)}
		}
		if i < len(wt.Steps)-1 {
			opts.Next = &render.NavLink{Title: render.PlainTitle(wt.Steps[i+1].Title), URL: p.Site.PageURL(steps[i+1].ID)}
		}
		opts.Diagram = p.diagramFunc(ctx, res, steps[i].ID, s.ID)

		out, err := render.RenderStorage(s, wt.Glossary, opts)
		if err != nil {
			return nil, err
		}
		for _, w := range out.Warnings {
			res.Warnings = append(res.Warnings, s.ID+": "+w)
		}
		if err := p.update(ctx, steps[i], titles[i], out.Body); err != nil {
			return nil, fmt.Errorf("update %s: %w", s.ID, err)
		}
	}

	idx, err := render.RenderStorageIndex(wt.Manifest, wt.Steps, stepURL)
	if err != nil {
		return nil, err
	}
	for _, w := range idx.Warnings {
		res.Warnings = append(res.Warnings, "index: "+w)
	}
	if err := p.update(ctx, index, title, idx.Body); err != nil {
		return nil, fmt.Errorf("update tutorial page: %w", err)
	}
	return res, nil
}

// ensure returns the page for a title, creating it when it does not exist. An
// existing section page is left untouched, since it is shared and hand-edited.
func (p *Publisher) ensure(ctx context.Context, res *Result, key, title, label, parentID, body string, extraLabels []string) (*Content, error) {
	var page *Content
	if label != "" {
		found, err := p.Client.FindByLabel(ctx, p.Target.SpaceKey, label)
		if err != nil {
			return nil, fmt.Errorf("find %s by label: %w", key, err)
		}
		if len(found) > 0 {
			page = &found[0]
			if len(found) > 1 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %d pages carry label %s, using %s", key, len(found), label, page.ID))
			}
		}
	}
	if page == nil {
		found, err := p.Client.FindByTitle(ctx, p.Target.SpaceKey, title)
		if err != nil {
			return nil, fmt.Errorf("find %s by title: %w", key, err)
		}
		page = found
	}

	created := false
	if page == nil {
		created = true
		if p.DryRun {
			page = &Content{ID: "(new)", Title: title}
		} else {
			var err error
			page, err = p.Client.CreatePage(ctx, NewPage{
				SpaceKey: p.Target.SpaceKey, ParentID: parentID, Title: title, Body: body, Status: "current",
			})
			if err != nil {
				return nil, fmt.Errorf("create %s: %w", key, err)
			}
		}
	}
	if !p.DryRun && label != "" {
		labels := append([]string{label}, extraLabels...)
		if err := p.Client.AddLabels(ctx, page.ID, labels...); err != nil {
			return nil, fmt.Errorf("label %s: %w", key, err)
		}
	}
	res.Pages = append(res.Pages, PageResult{Key: key, Title: title, ID: page.ID, URL: p.Site.PageURL(page.ID), Created: created})
	return page, nil
}

func (p *Publisher) update(ctx context.Context, page *Content, title, body string) error {
	return p.Client.UpdatePage(ctx, PageUpdate{
		ID: page.ID, Title: title, Body: body,
		Version: page.Version.Number, Status: "current",
		Message: "walkr publish", Minor: true,
	})
}

// diagramFunc renders a Mermaid block to PNG and attaches it to the page. The
// file is named by content hash, so a diagram that did not change keeps its
// name. Any failure falls back to showing the source and records a warning.
func (p *Publisher) diagramFunc(ctx context.Context, res *Result, pageID, stepID string) func(string) (string, bool) {
	if p.Diagrams == nil || p.Target.Diagrams == "source" {
		return nil
	}
	done := map[string]bool{}
	return func(src string) (string, bool) {
		sum := sha256.Sum256([]byte(src))
		name := "diagram-" + hex.EncodeToString(sum[:])[:12] + ".png"
		if done[name] {
			return name, true
		}
		png, err := p.Diagrams.Render(ctx, src)
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: diagram not rendered, showing source: %v", stepID, err))
			return "", false
		}
		if _, err := p.Client.PutAttachment(ctx, pageID, name, png); err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: diagram upload failed, showing source: %v", stepID, err))
			return "", false
		}
		done[name] = true
		return name, true
	}
}

// sectionBody is the body of a newly created section page. Once it exists the
// publisher never rewrites it.
func sectionBody() string {
	return `<p>Guided tutorials, one page per concept.</p>` +
		`<ac:structured-macro ac:name="children" ac:schema-version="1"></ac:structured-macro>`
}
