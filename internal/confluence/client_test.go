package confluence

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bashfulrobot/walkr/internal/secrets"
)

const canary = "tok-canary-should-never-appear"

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New("cloud", "me@example.com", secrets.New(canary), WithBaseURL(srv.URL))
}

func TestSpacesSendsBasicAuthAndDecodes(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "me@example.com" || pass != canary {
			t.Errorf("bad basic auth: %q %q %v", user, pass, ok)
		}
		if r.URL.Path != "/space" || r.URL.Query().Get("limit") != "3" {
			t.Errorf("request = %s", r.URL)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":9,"key":"ENG","name":"Engineering","type":"global"}]}`))
	})
	got, err := c.Spaces(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Key != "ENG" {
		t.Fatalf("got %+v", got)
	}
}

func TestContentRequestsAnyStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/content/123" || r.URL.Query().Get("status") != "any" {
			t.Errorf("request = %s", r.URL)
		}
		_, _ = w.Write([]byte(`{"id":"123","type":"page","status":"draft","title":"T","version":{"number":4}}`))
	})
	got, err := c.Content(context.Background(), "123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "T" || got.Status != "draft" || got.Version.Number != 4 {
		t.Fatalf("got %+v", got)
	}
}

func TestErrorIsTypedAndNeverLeaksToken(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
	_, err := c.Spaces(context.Background(), 1)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), canary) {
		t.Errorf("error leaked the token: %v", err)
	}
}

func TestTransportErrorNeverLeaksToken(t *testing.T) {
	c := New("cloud", "me@example.com", secrets.New(canary), WithBaseURL("http://127.0.0.1:1"))
	_, err := c.Spaces(context.Background(), 1)
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if strings.Contains(err.Error(), canary) {
		t.Errorf("error leaked the token: %v", err)
	}
}

func TestDefaultBaseUsesGatewayAndCloudID(t *testing.T) {
	c := New("abc-123", "e", secrets.New("t"))
	want := "https://api.atlassian.com/ex/confluence/abc-123/wiki/rest/api"
	if c.base != want {
		t.Fatalf("base = %q, want %q", c.base, want)
	}
}
