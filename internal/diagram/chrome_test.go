package diagram

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"strings"
	"testing"
)

// These tests drive a real headless Chrome, so they skip on machines without
// one. They make no network calls.
func newTestChrome(t *testing.T) *Chrome {
	t.Helper()
	c, err := NewChrome()
	if errors.Is(err, ErrNoBrowser) {
		t.Skip("no Chrome or Chromium installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

const sample = "graph TB\n  PORT[Customer portfolio] --> CS[CS project]\n  PORT --> PS[PS projects]\n  CS -.->|flagged| RAID[RAID view]\n"

func TestRenderProducesAPNGAtTwiceScale(t *testing.T) {
	c := newTestChrome(t)
	data, err := c.Render(context.Background(), sample)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() < 300 || b.Dy() < 200 {
		t.Errorf("image is only %dx%d, expected a 2x render of a real diagram", b.Dx(), b.Dy())
	}
	t.Logf("rendered %dx%d px, %d bytes, browser %s", b.Dx(), b.Dy(), len(data), c.Path())
}

func TestRenderReusesOneBrowserForManyDiagrams(t *testing.T) {
	c := newTestChrome(t)
	for i := 0; i < 3; i++ {
		if _, err := c.Render(context.Background(), sample); err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
	}
}

func TestRenderReportsMermaidSyntaxErrors(t *testing.T) {
	c := newTestChrome(t)
	_, err := c.Render(context.Background(), "graph TB\n  this is ((( not valid")
	if err == nil {
		t.Fatal("expected an error for invalid Mermaid")
	}
	if !strings.Contains(err.Error(), "render diagram") {
		t.Errorf("err = %v", err)
	}
}

func TestNewChromeHonorsMissingOverride(t *testing.T) {
	t.Setenv("WALKR_CHROME", "/definitely/not/a/browser")
	if _, err := NewChrome(); err == nil {
		t.Fatal("expected an error for a bad WALKR_CHROME")
	}
}
