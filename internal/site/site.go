// Package site assembles a loaded walkthrough (internal/walkthrough) plus
// its rendered steps (internal/render) into a self-contained static site:
// index.html + copied vendor/asset files. No server required to view it.
package site

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bashfulrobot/walkr/assets"
	"github.com/bashfulrobot/walkr/internal/render"
	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

//go:embed page.html.tmpl
var pageTemplateSrc string

var pageTemplate = template.Must(template.New("page").Parse(pageTemplateSrc))

type stepView struct {
	ID        string
	ChapterNo string
	Kind      string
	TitleHTML template.HTML
	Summary   string
	BodyHTML  template.HTML
}

type railStep struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
}

type pageData struct {
	// AssetBase is the URL prefix for vendor/ and assets/: empty when they sit
	// beside index.html, or a relative path like "../../_walkr/" when shared.
	AssetBase            string
	Title, Tagline, Repo string
	Steps                []stepView
	StepsJSON            template.JS
	DeepDives            []render.DeepDive
}

// Options customizes a build.
type Options struct {
	// SharedAssets, when set, is the relative URL from the built index.html to
	// a directory written by WriteSharedAssets, with a trailing slash, for
	// example "../../_walkr/". The site then references vendor/ and assets/
	// there instead of carrying its own copies.
	SharedAssets string
}

// Build renders wt into outDir: index.html plus vendor/ and assets/ copied
// alongside it. outDir is created if it doesn't exist; existing contents
// are left in place except for files this call overwrites.
func Build(wt *walkthrough.Walkthrough, outDir string) error {
	return BuildWith(wt, outDir, Options{})
}

// WriteSharedAssets writes the vendored libraries and walkr's own CSS and JS
// into dir as vendor/ and assets/, ready to be shared by many sites.
func WriteSharedAssets(dir string) error {
	if err := copyEmbeddedDir(assets.Vendor, "vendor", filepath.Join(dir, "vendor")); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "style.css"), assets.StyleCSS, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "assets", "app.js"), assets.AppJS, 0o644)
}

// BuildWith is Build with options.
func BuildWith(wt *walkthrough.Walkthrough, outDir string, opts Options) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	data := pageData{
		AssetBase: opts.SharedAssets,
		Title:     wt.Manifest.Title,
		Tagline:   wt.Manifest.Tagline,
		Repo:      wt.Manifest.Repo,
	}

	stepIDs := make(map[string]bool, len(wt.Steps))
	for _, step := range wt.Steps {
		stepIDs[step.ID] = true
	}

	var rail []railStep
	for i, step := range wt.Steps {
		res, err := render.RenderStep(step, wt.Glossary, stepIDs)
		if err != nil {
			return fmt.Errorf("step %q: %w", step.ID, err)
		}
		titleHTML, err := render.RenderTitle(step.Title)
		if err != nil {
			return fmt.Errorf("step %q title: %w", step.ID, err)
		}
		data.Steps = append(data.Steps, stepView{
			ID:        step.ID,
			ChapterNo: fmt.Sprintf("%02d", i+1),
			Kind:      step.Kind,
			TitleHTML: template.HTML(titleHTML),
			Summary:   step.Summary,
			BodyHTML:  template.HTML(res.HTML),
		})
		data.DeepDives = append(data.DeepDives, res.DeepDives...)
		rail = append(rail, railStep{ID: step.ID, Title: step.Label, Kind: step.Kind})
	}

	railJSON, err := json.Marshal(rail)
	if err != nil {
		return err
	}
	data.StepsJSON = template.JS(railJSON)

	out, err := os.Create(filepath.Join(outDir, "index.html"))
	if err != nil {
		return err
	}
	defer out.Close()
	if err := pageTemplate.Execute(out, data); err != nil {
		return fmt.Errorf("rendering index.html: %w", err)
	}

	if opts.SharedAssets == "" {
		if err := WriteSharedAssets(outDir); err != nil {
			return err
		}
	}

	if err := copyMedia(wt.Dir, outDir); err != nil {
		return err
	}

	return nil
}

// copyMedia copies <walkthroughDir>/media/ to <outDir>/media/ when present.
// Absence is not an error — most walkthroughs have no author-supplied assets.
func copyMedia(walkthroughDir, outDir string) error {
	mediaDir := filepath.Join(walkthroughDir, "media")
	info, err := os.Stat(mediaDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: expected a directory", mediaDir)
	}
	return copyDir(mediaDir, filepath.Join(outDir, "media"))
}

// copyDir recursively copies a real on-disk directory tree, byte-for-byte,
// to destDir. Unlike copyEmbeddedDir this walks os.DirFS rather than an
// embed.FS, since media/ is author-supplied content on disk, not compiled
// into the binary.
func copyDir(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		dst, err := os.Create(target)
		if err != nil {
			return err
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		return err
	})
}

// copyEmbeddedDir copies an embed.FS subtree to a real directory on disk.
func copyEmbeddedDir(fsys fs.FS, srcDir, destDir string) error {
	return fs.WalkDir(fsys, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		src, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		dst, err := os.Create(target)
		if err != nil {
			return err
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		return err
	})
}
