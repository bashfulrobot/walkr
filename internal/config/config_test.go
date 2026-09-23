package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const valid = `
confluence:
  cloud_id: 11111111-2222-3333-4444-555555555555
  email: someone@example.com
  auth:
    token_ref: op://vault/item/token
  targets:
    demo:
      space_key: "DEMO"
      parent_id: "7"
      diagrams: png
`

func TestParseValid(t *testing.T) {
	c, err := Parse([]byte(valid), "test")
	if err != nil {
		t.Fatal(err)
	}
	if c.Confluence.Email != "someone@example.com" {
		t.Errorf("email = %q", c.Confluence.Email)
	}
	if got := c.Confluence.Auth.ServiceAccountTokenEnv; got != DefaultServiceAccountTokenEnv {
		t.Errorf("default sa env = %q", got)
	}
	tgt, err := c.Target("demo")
	if err != nil || tgt.SpaceKey != "DEMO" || tgt.ParentID != "7" {
		t.Errorf("target = %+v, err %v", tgt, err)
	}
	if err := c.RequireConfluence(); err != nil {
		t.Errorf("RequireConfluence: %v", err)
	}
}

func TestParseRejectsLiteralTokenKey(t *testing.T) {
	raw := strings.Replace(valid, "token_ref: op://vault/item/token", "token: not-a-real-token-value", 1)
	_, err := Parse([]byte(raw), "test")
	if err == nil {
		t.Fatal("expected an error for a literal token key")
	}
	if strings.Contains(err.Error(), "not-a-real-token-value") {
		t.Errorf("error echoes the value: %v", err)
	}
}

func TestParseRejectsNonOpTokenRef(t *testing.T) {
	raw := strings.Replace(valid, "op://vault/item/token", "not-a-real-token-value", 1)
	_, err := Parse([]byte(raw), "test")
	if err == nil {
		t.Fatal("expected an error for a non-op:// token_ref")
	}
	if strings.Contains(err.Error(), "not-a-real-token-value") {
		t.Errorf("error echoes the value: %v", err)
	}
}

func TestParseRejectsBadDiagramsMode(t *testing.T) {
	raw := strings.Replace(valid, "diagrams: png", "diagrams: svg", 1)
	if _, err := Parse([]byte(raw), "test"); err == nil {
		t.Fatal("expected an error for diagrams: svg")
	}
}

func TestParseEmptyFile(t *testing.T) {
	c, err := Parse(nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.RequireConfluence(); err == nil {
		t.Error("empty config should fail RequireConfluence")
	}
}

func TestTargetUnknownListsNames(t *testing.T) {
	c, _ := Parse([]byte(valid), "test")
	_, err := c.Target("nope")
	if err == nil || !strings.Contains(err.Error(), "demo") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadMissingFileNamesPath(t *testing.T) {
	p := filepath.Join(t.TempDir(), "absent.yaml")
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), p) {
		t.Fatalf("err = %v", err)
	}
}

func TestDefaultPathHonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	got, err := DefaultPath()
	if err != nil || got != filepath.Join("/xdg", "walkr", "config.yaml") {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestLoadRoundTripFromDisk(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err != nil {
		t.Fatal(err)
	}
}
