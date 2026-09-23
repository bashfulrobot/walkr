//go:build integration

package confluence_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bashfulrobot/walkr/internal/config"
	"github.com/bashfulrobot/walkr/internal/confluence"
	"github.com/bashfulrobot/walkr/internal/secrets"
)

// Runs against real Confluence with the credentials in the global config.
// It uploads a generated PNG to the target's parent page and appends an image
// reference to the body, once. Run with:
//
//	go test -tags integration ./internal/confluence -run TestAttachmentRoundTrip -v
//
// WALKR_IT_TARGET picks the config target (default asana-templates).
// integrationClient builds a client from the global config, skipping the
// test when credentials are not available.
func integrationClient(t *testing.T) (*confluence.Client, config.Target) {
	t.Helper()
	ctx := context.Background()

	path, err := config.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Skipf("no usable config: %v", err)
	}
	name := os.Getenv("WALKR_IT_TARGET")
	if name == "" {
		name = "asana-templates"
	}
	tgt, err := cfg.Target(name)
	if err != nil {
		t.Skip(err)
	}
	saEnv := cfg.Confluence.Auth.ServiceAccountTokenEnv
	if os.Getenv(saEnv) == "" {
		t.Skipf("%s not set", saEnv)
	}

	op, err := secrets.NewOnePassword(ctx, secrets.New(os.Getenv(saEnv)), "integration-test")
	if err != nil {
		t.Fatal(err)
	}
	token, err := op.Resolve(ctx, cfg.Confluence.Auth.TokenRef)
	if err != nil {
		t.Fatal(err)
	}
	return confluence.New(cfg.Confluence.CloudID, cfg.Confluence.Email, token), tgt
}

// TestPageUpdateRoundTrip re-saves the target's parent page with its own body,
// proving the token can write page content and bump the version.
func TestPageUpdateRoundTrip(t *testing.T) {
	ctx := context.Background()
	c, tgt := integrationClient(t)

	page, err := c.Content(ctx, tgt.ParentID)
	if err != nil {
		t.Fatal(err)
	}
	err = c.UpdatePage(ctx, confluence.PageUpdate{
		ID: page.ID, Title: page.Title, Body: page.Body.Storage.Value,
		Version: page.Version.Number, Status: page.Status,
		Message: "walkr integration test: no-op update",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	after, err := c.Content(ctx, tgt.ParentID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("status %s, version %d -> %d", page.Status, page.Version.Number, after.Version.Number)
	if after.Version.Number <= page.Version.Number {
		t.Errorf("version did not advance: %d -> %d", page.Version.Number, after.Version.Number)
	}
}

func TestAttachmentRoundTrip(t *testing.T) {
	ctx := context.Background()
	c, tgt := integrationClient(t)

	const file = "walkr-api-spike.png"
	att, err := c.PutAttachment(ctx, tgt.ParentID, file, spikePNG(t))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	t.Logf("attachment %s, version %d", att.ID, att.Version.Number)

	page, err := c.Content(ctx, tgt.ParentID)
	if err != nil {
		t.Fatal(err)
	}
	ref := `<ri:attachment ri:filename="` + file + `" />`
	body := page.Body.Storage.Value
	if !strings.Contains(body, file) {
		body += `<p>API spike, image attached through the REST API:</p>` +
			`<ac:image ac:width="480">` + ref + `</ac:image>`
	}
	err = c.UpdatePage(ctx, confluence.PageUpdate{
		ID: page.ID, Title: page.Title, Body: body,
		Version: page.Version.Number, Status: page.Status,
		Message: "walkr integration test: attachment round trip",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	after, err := c.Content(ctx, tgt.ParentID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(after.Body.Storage.Value, file) {
		t.Fatalf("image reference missing after update; body:\n%s", after.Body.Storage.Value)
	}
	t.Logf("page now at version %d", after.Version.Number)
}

// spikePNG draws three labelled-by-colour boxes joined by a bar, enough to
// tell at a glance that the image rendered.
func spikePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 480, 160))
	fill := func(r image.Rectangle, c color.Color) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				img.Set(x, y, c)
			}
		}
	}
	fill(img.Bounds(), color.White)
	fill(image.Rect(20, 20, 140, 70), color.RGBA{0xB3, 0xD4, 0xFF, 0xFF})
	fill(image.Rect(180, 20, 300, 70), color.RGBA{0xAB, 0xF5, 0xD1, 0xFF})
	fill(image.Rect(340, 20, 460, 70), color.RGBA{0xFF, 0xF0, 0xB3, 0xFF})
	fill(image.Rect(60, 100, 420, 130), color.RGBA{0xC0, 0xB6, 0xF2, 0xFF})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestDraftLifecycle explores how v1 treats drafts: create, label, search,
// update. It asserts the safety property (a page created as a draft stays a
// draft through an update) and logs what CQL search can and cannot see. It
// leaves one new page in the target's space, delete it when done.
func TestDraftLifecycle(t *testing.T) {
	ctx := context.Background()
	c, tgt := integrationClient(t)

	stamp := time.Now().UTC().Format("20060102-150405")
	label := "walkr-it-" + stamp
	created, err := c.CreatePage(ctx, confluence.NewPage{
		SpaceKey: tgt.SpaceKey,
		Title:    "[walkr integration] draft lifecycle " + stamp,
		Body:     "<p>Created by the walkr integration test. Safe to delete.</p>",
		Status:   "draft",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Logf("created %s, status %q", created.ID, created.Status)

	got, err := c.Content(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("read back: status %q, version %d", got.Status, got.Version.Number)

	if err := c.AddLabels(ctx, created.ID, label); err != nil {
		t.Errorf("add label: %v", err)
	}
	found, err := c.FindByLabel(ctx, tgt.SpaceKey, label)
	t.Logf("label search: %d result(s), err %v (draft visibility to CQL)", len(found), err)

	err = c.UpdatePage(ctx, confluence.PageUpdate{
		ID: created.ID, Title: got.Title, Body: got.Body.Storage.Value + "<p>updated</p>",
		Version: got.Version.Number, Status: "draft",
		Message: "walkr integration test: draft update",
	})
	if err != nil {
		t.Fatalf("update as draft: %v", err)
	}
	after, err := c.Content(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("after update: status %q, version %d", after.Status, after.Version.Number)
	if created.Status == "draft" && after.Status != "draft" {
		t.Errorf("a draft became %q after update", after.Status)
	}
}

// TestPublishedLabelSearch creates a published page, labels it, and polls CQL
// label search until the page shows up, logging how long indexing took. It
// then updates the page and finds it again, the republish path. It leaves one
// published page in the target's space, delete it when done.
func TestPublishedLabelSearch(t *testing.T) {
	ctx := context.Background()
	c, tgt := integrationClient(t)

	stamp := time.Now().UTC().Format("20060102-150405")
	label := "walkr-it-" + stamp
	created, err := c.CreatePage(ctx, confluence.NewPage{
		SpaceKey: tgt.SpaceKey,
		Title:    "[walkr integration] published label search " + stamp,
		Body:     "<p>Created by the walkr integration test. Safe to delete.</p>",
		Status:   "current",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Logf("created %s, status %q, version %d", created.ID, created.Status, created.Version.Number)
	if err := c.AddLabels(ctx, created.ID, label); err != nil {
		t.Fatalf("add label: %v", err)
	}

	find := func(want string) bool {
		start := time.Now()
		for time.Since(start) < 90*time.Second {
			found, err := c.FindByLabel(ctx, tgt.SpaceKey, label)
			if err != nil {
				t.Fatalf("search: %v", err)
			}
			if len(found) == 1 && found[0].ID == created.ID {
				t.Logf("%s: found after %s", want, time.Since(start).Round(time.Second))
				return true
			}
			time.Sleep(3 * time.Second)
		}
		t.Errorf("%s: label %q not found within 90s", want, label)
		return false
	}
	if !find("after create") {
		return
	}

	err = c.UpdatePage(ctx, confluence.PageUpdate{
		ID: created.ID, Title: created.Title, Body: "<p>updated</p>",
		Version: created.Version.Number, Status: "current",
		Message: "walkr integration test: republish",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	after, err := c.Content(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("after update: status %q, version %d", after.Status, after.Version.Number)
	if after.Version.Number != created.Version.Number+1 {
		t.Errorf("version = %d, want %d", after.Version.Number, created.Version.Number+1)
	}
	find("after update")
}
