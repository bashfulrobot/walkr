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
	Title, Tagline, Repo string
	Steps                []stepView
	StepsJSON            template.JS
	DeepDives            []render.DeepDive
}

// Build renders wt into outDir: index.html plus vendor/ and assets/ copied
// alongside it. outDir is created if it doesn't exist; existing contents
// are left in place except for files this call overwrites.
func Build(wt *walkthrough.Walkthrough, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	data := pageData{
		Title:   wt.Manifest.Title,
		Tagline: wt.Manifest.Tagline,
		Repo:    wt.Manifest.Repo,
	}

	var rail []railStep
	for i, step := range wt.Steps {
		res, err := render.RenderStep(step, wt.Glossary)
		if err != nil {
			return fmt.Errorf("step %q: %w", step.ID, err)
		}
		titleHTML, err := render.RenderTitle(step.Title)
		if err != nil {
			return fmt.Errorf("step %q title: %w", step.ID, err)
		}
		data.Steps = append(data.Steps, stepView{
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

	if err := copyEmbeddedDir(assets.Vendor, "vendor", filepath.Join(outDir, "vendor")); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outDir, "assets"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "assets", "style.css"), assets.StyleCSS, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "assets", "app.js"), assets.AppJS, 0o644); err != nil {
		return err
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
