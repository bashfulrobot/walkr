package confluence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bashfulrobot/walkr/internal/config"
	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

// fakeConfluence is an in-memory PageClient. labelLag hides labels from
// FindByLabel, the way the real search index does right after a create.
type fakeConfluence struct {
	pages       map[string]*fakePage
	next        int
	labelLag    bool
	creates     int
	updates     int
	attachments []string
}

type fakePage struct {
	Content
	parent string
	labels []string
	body   string
}

func newFake() *fakeConfluence { return &fakeConfluence{pages: map[string]*fakePage{}, next: 100} }

func (f *fakeConfluence) CreatePage(_ context.Context, p NewPage) (*Content, error) {
	f.next++
	f.creates++
	id := fmt.Sprint(f.next)
	pg := &fakePage{parent: p.ParentID, body: p.Body}
	pg.ID, pg.Title, pg.Status = id, p.Title, "current"
	pg.Version.Number = 1
	f.pages[id] = pg
	c := pg.Content
	return &c, nil
}

func (f *fakeConfluence) UpdatePage(_ context.Context, u PageUpdate) error {
	pg, ok := f.pages[u.ID]
	if !ok {
		return errors.New("no such page")
	}
	if u.Version != pg.Version.Number {
		return fmt.Errorf("stale version %d, page is at %d", u.Version, pg.Version.Number)
	}
	f.updates++
	pg.Title, pg.body = u.Title, u.Body
	pg.Version.Number++
	return nil
}

func (f *fakeConfluence) AddLabels(_ context.Context, id string, labels ...string) error {
	pg := f.pages[id]
	for _, l := range labels {
		has := false
		for _, e := range pg.labels {
			has = has || e == l
		}
		if !has {
			pg.labels = append(pg.labels, l)
		}
	}
	return nil
}

func (f *fakeConfluence) FindByLabel(_ context.Context, _ string, label string) ([]Content, error) {
	if f.labelLag {
		return nil, nil
	}
	var out []Content
	for _, pg := range f.pages {
		for _, l := range pg.labels {
			if l == label {
				out = append(out, pg.Content)
			}
		}
	}
	return out, nil
}

func (f *fakeConfluence) FindByTitle(_ context.Context, _ string, title string) (*Content, error) {
	for _, pg := range f.pages {
		if pg.Title == title {
			c := pg.Content
			return &c, nil
		}
	}
	return nil, nil
}

func (f *fakeConfluence) PutAttachment(_ context.Context, pageID, name string, _ []byte) (*Attachment, error) {
	f.attachments = append(f.attachments, pageID+"/"+name)
	return &Attachment{ID: "att", Title: name}, nil
}

type fakeDiagrams struct{ err error }

func (d fakeDiagrams) Render(context.Context, string) ([]byte, error) {
	return []byte("PNG"), d.err
}

func testWalkthrough() *walkthrough.Walkthrough {
	step := func(id string, order int, body string) walkthrough.Step {
		return walkthrough.Step{
			ID: id, Title: "Step " + id, Label: id, Kind: "Kind", Order: order,
			Layout: walkthrough.LayoutOverview, Summary: "Lede " + id, Body: body,
		}
	}
	return &walkthrough.Walkthrough{
		Manifest: walkthrough.Manifest{Title: "Demo Tutorial", Tagline: "Field Guide"},
		Glossary: walkthrough.Glossary{},
		Steps: []walkthrough.Step{
			step("01-a", 1, "Go to [b]{step=02-b}.\n\n```mermaid title=\"d.mmd\"\ngraph TB\n  A --> B\n```\n"),
			step("02-b", 2, "Second."),
			step("03-c", 3, "Third."),
		},
	}
}

func newPublisher(f *fakeConfluence) *Publisher {
	return &Publisher{
		Client: f,
		Site:   config.Confluence{Site: "example.atlassian.net"},
		Name:   "demo",
		Target: config.Target{SpaceKey: "~me", ParentID: "1", Section: "Tutorials"},
	}
}

func byTitle(f *fakeConfluence, title string) *fakePage {
	for _, pg := range f.pages {
		if pg.Title == title {
			return pg
		}
	}
	return nil
}

func TestPublishBuildsSectionTutorialAndSteps(t *testing.T) {
	f := newFake()
	res, err := newPublisher(f).Publish(context.Background(), testWalkthrough())
	if err != nil {
		t.Fatal(err)
	}
	if f.creates != 5 {
		t.Fatalf("creates = %d, want 5 (section, tutorial, 3 steps)", f.creates)
	}

	section := byTitle(f, "Tutorials")
	tutorial := byTitle(f, "Demo Tutorial")
	step2 := byTitle(f, "Demo Tutorial: Step 02-b")
	if section == nil || tutorial == nil || step2 == nil {
		t.Fatalf("missing pages: %v %v %v", section, tutorial, step2)
	}
	if section.parent != "1" || tutorial.parent != section.ID || step2.parent != tutorial.ID {
		t.Errorf("tree wrong: section<-%s tutorial<-%s step<-%s", section.parent, tutorial.parent, step2.parent)
	}

	first := byTitle(f, "Demo Tutorial: Step 01-a")
	if !strings.Contains(first.body, "pageId="+step2.ID) {
		t.Errorf("step 1 does not link to the real ID of step 2 (%s):\n%s", step2.ID, first.body)
	}
	if !strings.Contains(step2.body, "Next: Step 03-c") || !strings.Contains(step2.body, "Previous: Step 01-a") {
		t.Errorf("step 2 pager missing:\n%s", step2.body)
	}
	if !strings.Contains(tutorial.body, "pageId="+first.ID) {
		t.Errorf("tutorial page does not list step 1:\n%s", tutorial.body)
	}
	if strings.Contains(step2.body, "Publishing in progress") {
		t.Error("placeholder body was not replaced")
	}
	if got := res.Tutorial().ID; got != tutorial.ID {
		t.Errorf("Result.Tutorial() = %s, want %s", got, tutorial.ID)
	}
	wantLabels := map[string]bool{"walkr-demo": true, "walkr-demo-02-b": true}
	for _, l := range step2.labels {
		delete(wantLabels, l)
	}
	if len(wantLabels) != 0 {
		t.Errorf("step 2 labels = %v", step2.labels)
	}
}

func TestPublishTwiceUpdatesInPlace(t *testing.T) {
	f := newFake()
	p := newPublisher(f)
	if _, err := p.Publish(context.Background(), testWalkthrough()); err != nil {
		t.Fatal(err)
	}
	creates := f.creates
	res, err := p.Publish(context.Background(), testWalkthrough())
	if err != nil {
		t.Fatal(err)
	}
	if f.creates != creates {
		t.Errorf("second run created %d page(s)", f.creates-creates)
	}
	for _, pg := range res.Pages {
		if pg.Created {
			t.Errorf("%s reported as created on a rerun", pg.Key)
		}
	}
}

func TestPublishRerunDuringIndexLagDoesNotDuplicate(t *testing.T) {
	f := newFake()
	p := newPublisher(f)
	if _, err := p.Publish(context.Background(), testWalkthrough()); err != nil {
		t.Fatal(err)
	}
	creates := f.creates
	f.labelLag = true // label search sees nothing, as in the first seconds after a create
	if _, err := p.Publish(context.Background(), testWalkthrough()); err != nil {
		t.Fatal(err)
	}
	if f.creates != creates {
		t.Errorf("rerun during lag created %d duplicate page(s)", f.creates-creates)
	}
}

func TestPublishRenamedStepIsFoundByLabel(t *testing.T) {
	f := newFake()
	p := newPublisher(f)
	if _, err := p.Publish(context.Background(), testWalkthrough()); err != nil {
		t.Fatal(err)
	}
	creates := f.creates
	wt := testWalkthrough()
	wt.Steps[1].Title = "A brand new title"
	if _, err := p.Publish(context.Background(), wt); err != nil {
		t.Fatal(err)
	}
	if f.creates != creates {
		t.Error("a renamed step created a new page instead of updating the old one")
	}
	if byTitle(f, "Demo Tutorial: A brand new title") == nil {
		t.Error("the page was not renamed")
	}
}

func TestPublishLeavesExistingSectionAlone(t *testing.T) {
	f := newFake()
	p := newPublisher(f)
	if _, err := p.Publish(context.Background(), testWalkthrough()); err != nil {
		t.Fatal(err)
	}
	section := byTitle(f, "Tutorials")
	section.body = "hand edited"
	if _, err := p.Publish(context.Background(), testWalkthrough()); err != nil {
		t.Fatal(err)
	}
	if section.body != "hand edited" {
		t.Errorf("section page was rewritten: %q", section.body)
	}
}

func TestPublishWithoutSectionPutsTutorialUnderParent(t *testing.T) {
	f := newFake()
	p := newPublisher(f)
	p.Target.Section = ""
	if _, err := p.Publish(context.Background(), testWalkthrough()); err != nil {
		t.Fatal(err)
	}
	if f.creates != 4 {
		t.Errorf("creates = %d, want 4", f.creates)
	}
	if got := byTitle(f, "Demo Tutorial").parent; got != "1" {
		t.Errorf("tutorial parent = %s, want 1", got)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	f := newFake()
	p := newPublisher(f)
	p.DryRun = true
	res, err := p.Publish(context.Background(), testWalkthrough())
	if err != nil {
		t.Fatal(err)
	}
	if f.creates+f.updates != 0 || len(f.pages) != 0 {
		t.Errorf("dry run wrote: creates %d updates %d pages %d", f.creates, f.updates, len(f.pages))
	}
	if len(res.Pages) != 5 || !res.Pages[0].Created {
		t.Errorf("plan = %+v", res.Pages)
	}
}

func TestDiagramsAreRenderedAndAttached(t *testing.T) {
	f := newFake()
	p := newPublisher(f)
	p.Diagrams = fakeDiagrams{}
	if _, err := p.Publish(context.Background(), testWalkthrough()); err != nil {
		t.Fatal(err)
	}
	first := byTitle(f, "Demo Tutorial: Step 01-a")
	if len(f.attachments) != 1 || !strings.HasPrefix(f.attachments[0], first.ID+"/diagram-") {
		t.Fatalf("attachments = %v", f.attachments)
	}
	if !strings.Contains(first.body, "ri:filename=\"diagram-") || strings.Contains(first.body, "Diagram source") {
		t.Errorf("page does not use the attached image:\n%s", first.body)
	}
}

func TestDiagramFailureFallsBackToSourceWithWarning(t *testing.T) {
	f := newFake()
	p := newPublisher(f)
	p.Diagrams = fakeDiagrams{err: errors.New("no chrome")}
	res, err := p.Publish(context.Background(), testWalkthrough())
	if err != nil {
		t.Fatal(err)
	}
	first := byTitle(f, "Demo Tutorial: Step 01-a")
	if !strings.Contains(first.body, "Diagram source") {
		t.Error("expected the source fallback")
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "no chrome") {
		t.Errorf("warnings = %v", res.Warnings)
	}
}

func TestSourceModeNeverRendersDiagrams(t *testing.T) {
	f := newFake()
	p := newPublisher(f)
	p.Diagrams = fakeDiagrams{}
	p.Target.Diagrams = "source"
	if _, err := p.Publish(context.Background(), testWalkthrough()); err != nil {
		t.Fatal(err)
	}
	if len(f.attachments) != 0 {
		t.Errorf("attachments = %v", f.attachments)
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"asana-templates": "asana-templates",
		"Asana Templates": "asana-templates",
		"01_Overview!":    "01-overview",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
