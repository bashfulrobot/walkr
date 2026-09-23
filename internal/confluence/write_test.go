package confluence

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPutAttachmentCreatesWhenAbsent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/content/9/child/attachment":
			_, _ = w.Write([]byte(`{"results":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/content/9/child/attachment":
			if r.Header.Get("X-Atlassian-Token") != "no-check" {
				t.Error("missing X-Atlassian-Token")
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			f, hdr, err := r.FormFile("file")
			if err != nil {
				t.Fatal(err)
			}
			b, _ := io.ReadAll(f)
			if hdr.Filename != "d.png" || string(b) != "PNGDATA" {
				t.Errorf("file = %q %q", hdr.Filename, b)
			}
			_, _ = w.Write([]byte(`{"results":[{"id":"att1","title":"d.png","version":{"number":1}}]}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
	})
	att, err := c.PutAttachment(context.Background(), "9", "d.png", []byte("PNGDATA"))
	if err != nil || att.ID != "att1" {
		t.Fatalf("att = %+v, err %v", att, err)
	}
}

func TestPutAttachmentUpdatesWhenPresent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"results":[{"id":"att7","title":"d.png","version":{"number":2}}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/content/9/child/attachment/att7/data":
			_, _ = w.Write([]byte(`{"id":"att7","title":"d.png","version":{"number":3}}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
	})
	att, err := c.PutAttachment(context.Background(), "9", "d.png", []byte("x"))
	if err != nil || att.Version.Number != 3 {
		t.Fatalf("att = %+v, err %v", att, err)
	}
}

func TestUpdatePageSendsStorageBodyAndDraftQuery(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/content/5" || r.URL.Query().Get("status") != "draft" {
			t.Errorf("request = %s %s", r.Method, r.URL)
		}
		var got struct {
			Title   string `json:"title"`
			Status  string `json:"status"`
			Version struct {
				Number  int    `json:"number"`
				Message string `json:"message"`
			} `json:"version"`
			Body struct {
				Storage struct {
					Value          string `json:"value"`
					Representation string `json:"representation"`
				} `json:"storage"`
			} `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.Version.Number != 3 || got.Version.Message != "m" || got.Status != "draft" ||
			got.Body.Storage.Representation != "storage" || !strings.Contains(got.Body.Storage.Value, "<p>hi</p>") {
			t.Errorf("payload = %+v", got)
		}
		_, _ = w.Write([]byte(`{}`))
	})
	err := c.UpdatePage(context.Background(), PageUpdate{
		ID: "5", Title: "T", Body: "<p>hi</p>", Version: 3, Status: "draft", Message: "m",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreatePageSendsSpaceAncestorAndStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/content" {
			t.Errorf("request = %s %s", r.Method, r.URL)
		}
		var got map[string]any
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		space := got["space"].(map[string]any)
		anc := got["ancestors"].([]any)[0].(map[string]any)
		if got["status"] != "draft" || space["key"] != "K" || anc["id"] != "77" || got["title"] != "T" {
			t.Errorf("payload = %v", got)
		}
		_, _ = w.Write([]byte(`{"id":"900","status":"draft","title":"T","version":{"number":1}}`))
	})
	p, err := c.CreatePage(context.Background(), NewPage{
		SpaceKey: "K", ParentID: "77", Title: "T", Body: "<p>x</p>", Status: "draft",
	})
	if err != nil || p.ID != "900" || p.Status != "draft" {
		t.Fatalf("page = %+v, err %v", p, err)
	}
}

func TestAddLabelsSendsGlobalPrefix(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/content/9/label" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var got []map[string]string
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0]["prefix"] != "global" || got[1]["name"] != "b" {
			t.Errorf("payload = %v", got)
		}
		_, _ = w.Write([]byte(`{}`))
	})
	if err := c.AddLabels(context.Background(), "9", "a", "b"); err != nil {
		t.Fatal(err)
	}
}

func TestFindByLabelBuildsCQL(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		if r.URL.Path != "/content/search" || !strings.Contains(cql, `label = "walkr-x"`) || !strings.Contains(cql, `space = "K"`) {
			t.Errorf("request = %s", r.URL)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"5","title":"P","status":"current"}]}`))
	})
	got, err := c.FindByLabel(context.Background(), "K", "walkr-x")
	if err != nil || len(got) != 1 || got[0].ID != "5" {
		t.Fatalf("got %+v, err %v", got, err)
	}
}

func TestUpdatePageBumpsVersionForPublishedPages(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var got struct {
			Version struct {
				Number int `json:"number"`
			} `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.Version.Number != 4 || r.URL.Query().Get("status") != "" {
			t.Errorf("version = %d, query = %q", got.Version.Number, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{}`))
	})
	err := c.UpdatePage(context.Background(), PageUpdate{ID: "5", Title: "T", Body: "b", Version: 3, Status: "current"})
	if err != nil {
		t.Fatal(err)
	}
}
