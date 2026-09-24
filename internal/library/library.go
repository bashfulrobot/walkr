// Package library builds a whole tree of walkthroughs at once. Every
// directory named .walkr that holds a steps/ directory is one tutorial, and its
// site is written next to it as site/. By default all sites share one copy of
// the vendored libraries and stylesheet in <root>/_walkr, which keeps a large
// collection small. The same code powers a freshness check, so a repository can
// verify that its committed sites match their sources.
package library

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bashfulrobot/walkr/internal/site"
	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

const (
	// SharedDir holds the assets every site references, at the library root.
	SharedDir = "_walkr"
	// SiteDir is the generated output directory beside each .walkr.
	SiteDir = "site"
	// SourceDir is the directory name that marks a tutorial.
	SourceDir = ".walkr"
)

// Options customizes a build or a check.
type Options struct {
	// PerSite gives every site its own copy of the assets, like walkr build,
	// instead of one shared copy in SharedDir.
	PerSite bool
}

// Tutorial is one discovered walkthrough.
type Tutorial struct {
	Rel     string // tutorial directory relative to the root, slash separated
	WalkDir string // absolute path of its .walkr directory
}

// Failure is a tutorial that could not be built.
type Failure struct {
	Rel string
	Err error
}

// Report summarizes BuildAll.
type Report struct {
	Built    int
	Failures []Failure
}

// Discover finds every tutorial under root, sorted by path. It skips .git,
// node_modules, generated site and shared directories, and any other hidden
// directory, so nothing generated or vendored is ever mistaken for a source.
func Discover(root string) ([]Tutorial, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var out []Tutorial
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != absRoot {
			switch {
			case name == SourceDir:
				if info, statErr := os.Stat(filepath.Join(path, "steps")); statErr == nil && info.IsDir() {
					rel, relErr := filepath.Rel(absRoot, filepath.Dir(path))
					if relErr != nil {
						return relErr
					}
					out = append(out, Tutorial{Rel: filepath.ToSlash(rel), WalkDir: path})
				}
				return filepath.SkipDir
			case name == "node_modules" || name == SiteDir || name == SharedDir || strings.HasPrefix(name, "."):
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, nil
}

// BuildAll builds every tutorial found under root into outRoot, which is root
// itself for a normal build. Each site directory is removed and rebuilt, so
// output never keeps files from an older layout. One tutorial failing does not
// stop the rest.
func BuildAll(root, outRoot string, opt Options) (*Report, error) {
	tutorials, err := Discover(root)
	if err != nil {
		return nil, err
	}
	if len(tutorials) == 0 {
		return nil, fmt.Errorf("no tutorials found under %s (a tutorial is a %s directory containing steps/)", root, SourceDir)
	}
	absOut, err := filepath.Abs(outRoot)
	if err != nil {
		return nil, err
	}

	shared := filepath.Join(absOut, SharedDir)
	if err := os.RemoveAll(shared); err != nil {
		return nil, err
	}
	if !opt.PerSite {
		if err := site.WriteSharedAssets(shared); err != nil {
			return nil, err
		}
	}

	rep := &Report{}
	for _, t := range tutorials {
		if err := buildOne(t, absOut, shared, opt); err != nil {
			rep.Failures = append(rep.Failures, Failure{Rel: t.Rel, Err: err})
			continue
		}
		rep.Built++
	}
	return rep, nil
}

func buildOne(t Tutorial, absOut, shared string, opt Options) error {
	wt, err := walkthrough.Load(t.WalkDir)
	if err != nil {
		return err
	}
	siteDir := filepath.Join(absOut, filepath.FromSlash(t.Rel), SiteDir)
	if err := os.RemoveAll(siteDir); err != nil {
		return err
	}
	var opts site.Options
	if !opt.PerSite {
		rel, err := filepath.Rel(siteDir, shared)
		if err != nil {
			return err
		}
		opts.SharedAssets = filepath.ToSlash(rel) + "/"
	}
	return site.BuildWith(wt, siteDir, opts)
}

// DiffKind says how a committed file differs from a fresh build.
type DiffKind string

const (
	// Missing means a fresh build produces it but the tree does not have it.
	Missing DiffKind = "missing"
	// Stale means the tree has it but the content differs from a fresh build.
	Stale DiffKind = "stale"
	// Extra means the tree has it but a fresh build would not produce it.
	Extra DiffKind = "extra"
)

// Diff is one difference, with a path relative to the root.
type Diff struct {
	Path string
	Kind DiffKind
}

// CheckResult is the outcome of Check.
type CheckResult struct {
	Diffs    []Diff
	Failures []Failure
}

// OK reports whether the tree matches a fresh build exactly.
func (r *CheckResult) OK() bool { return len(r.Diffs) == 0 && len(r.Failures) == 0 }

// Check builds everything into a temporary directory and compares it with what
// is on disk under root. It never modifies root.
func Check(root string, opt Options) (*CheckResult, error) {
	tmp, err := os.MkdirTemp("", "walkr-check-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	rep, err := BuildAll(root, tmp, opt)
	if err != nil {
		return nil, err
	}
	res := &CheckResult{Failures: rep.Failures}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	tutorials, err := Discover(root)
	if err != nil {
		return nil, err
	}

	if opt.PerSite {
		if _, statErr := os.Stat(filepath.Join(absRoot, SharedDir)); statErr == nil {
			res.Diffs = append(res.Diffs, Diff{Path: SharedDir, Kind: Extra})
		}
	} else {
		res.Diffs = append(res.Diffs, compareTrees(filepath.Join(tmp, SharedDir), filepath.Join(absRoot, SharedDir), SharedDir)...)
	}
	failed := map[string]bool{}
	for _, f := range rep.Failures {
		failed[f.Rel] = true
	}
	for _, t := range tutorials {
		if failed[t.Rel] {
			continue
		}
		rel := t.Rel + "/" + SiteDir
		res.Diffs = append(res.Diffs, compareTrees(
			filepath.Join(tmp, filepath.FromSlash(t.Rel), SiteDir),
			filepath.Join(absRoot, filepath.FromSlash(t.Rel), SiteDir),
			rel)...)
	}
	sort.Slice(res.Diffs, func(i, j int) bool { return res.Diffs[i].Path < res.Diffs[j].Path })
	return res, nil
}

// compareTrees reports how the tree at actual differs from the tree at want.
// prefix names the directory in the reported paths. A directory that is absent
// entirely is one Missing entry, not one per file.
func compareTrees(want, actual, prefix string) []Diff {
	if _, err := os.Stat(actual); errors.Is(err, fs.ErrNotExist) {
		return []Diff{{Path: prefix, Kind: Missing}}
	}
	var diffs []Diff
	wantFiles := map[string]bool{}
	_ = filepath.WalkDir(want, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(want, path)
		rel = filepath.ToSlash(rel)
		wantFiles[rel] = true
		got, readErr := os.ReadFile(filepath.Join(actual, filepath.FromSlash(rel)))
		if readErr != nil {
			diffs = append(diffs, Diff{Path: prefix + "/" + rel, Kind: Missing})
			return nil
		}
		expected, _ := os.ReadFile(path)
		if !bytes.Equal(got, expected) {
			diffs = append(diffs, Diff{Path: prefix + "/" + rel, Kind: Stale})
		}
		return nil
	})
	_ = filepath.WalkDir(actual, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(actual, path)
		rel = filepath.ToSlash(rel)
		if !wantFiles[rel] {
			diffs = append(diffs, Diff{Path: prefix + "/" + rel, Kind: Extra})
		}
		return nil
	})
	return diffs
}
