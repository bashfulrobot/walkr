package confluence

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bashfulrobot/walkr/internal/config"
	"github.com/bashfulrobot/walkr/internal/secrets"
)

type fakeResolver struct {
	val secrets.Secret
	err error
	got string
}

func (f *fakeResolver) Resolve(_ context.Context, ref string) (secrets.Secret, error) {
	f.got = ref
	return f.val, f.err
}

func testConfig() *config.Config {
	c, err := config.Parse([]byte(`
confluence:
  cloud_id: cloud
  email: me@example.com
  auth:
    token_ref: op://vault/item/token
  targets:
    demo:
      space_key: "DEMO"
      parent_id: "55"
`), "test")
	if err != nil {
		panic(err)
	}
	return c
}

func TestCheckReportsSpacesAndParent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/space":
			_, _ = w.Write([]byte(`{"results":[{"id":1,"key":"K"}]}`))
		case "/content/55":
			_, _ = w.Write([]byte(`{"id":"55","status":"current","title":"Home","version":{"number":2}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	res := &fakeResolver{val: secrets.New(canary)}
	var out bytes.Buffer
	if err := Check(context.Background(), &out, testConfig(), "demo", res, WithBaseURL(srv.URL)); err != nil {
		t.Fatal(err)
	}
	if res.got != "op://vault/item/token" {
		t.Errorf("resolved %q", res.got)
	}
	for _, want := range []string{"spaces:  ok", `parent:  ok, "Home"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), canary) {
		t.Errorf("output leaked the token:\n%s", out.String())
	}
}

func TestCheckSurfacesAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	var out bytes.Buffer
	err := Check(context.Background(), &out, testConfig(), "", &fakeResolver{val: secrets.New(canary)}, WithBaseURL(srv.URL))
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 401 {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), canary) || strings.Contains(out.String(), canary) {
		t.Error("token leaked")
	}
}

func TestCheckResolverFailure(t *testing.T) {
	res := &fakeResolver{err: errors.New("vault not shared with service account")}
	err := Check(context.Background(), &bytes.Buffer{}, testConfig(), "", res)
	if err == nil || !strings.Contains(err.Error(), "vault not shared") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckUnknownTarget(t *testing.T) {
	err := Check(context.Background(), &bytes.Buffer{}, testConfig(), "nope", &fakeResolver{})
	if err == nil || !strings.Contains(err.Error(), "demo") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckRequiresConfig(t *testing.T) {
	empty, _ := config.Parse(nil, "test")
	if err := Check(context.Background(), &bytes.Buffer{}, empty, "", &fakeResolver{}); err == nil {
		t.Fatal("expected a config error")
	}
}
