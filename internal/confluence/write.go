package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
)

// Attachment is a file attached to a page.
type Attachment struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
}

// PutAttachment uploads data as filename on the page, adding a new version
// when a file with that name is already attached. Reference the file from
// page storage with <ac:image><ri:attachment ri:filename="..."/></ac:image>.
func (c *Client) PutAttachment(ctx context.Context, pageID, filename string, data []byte) (*Attachment, error) {
	existing, err := c.attachmentByName(ctx, pageID, filename)
	if err != nil {
		return nil, err
	}
	path := "/content/" + url.PathEscape(pageID) + "/child/attachment"
	if existing != nil {
		path += "/" + url.PathEscape(existing.ID) + "/data"
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(data); err != nil {
		return nil, err
	}
	if err := mw.WriteField("minorEdit", "true"); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check")

	// Create returns {"results":[att]}, update returns the attachment itself.
	var out struct {
		Attachment
		Results []Attachment `json:"results"`
	}
	if err := c.do(req, path, &out); err != nil {
		return nil, err
	}
	if len(out.Results) > 0 {
		return &out.Results[0], nil
	}
	return &out.Attachment, nil
}

func (c *Client) attachmentByName(ctx context.Context, pageID, filename string) (*Attachment, error) {
	var out struct {
		Results []Attachment `json:"results"`
	}
	q := url.Values{"filename": {filename}}
	if err := c.get(ctx, "/content/"+url.PathEscape(pageID)+"/child/attachment", q, &out); err != nil {
		return nil, err
	}
	for i := range out.Results {
		if out.Results[i].Title == filename {
			return &out.Results[i], nil
		}
	}
	return nil, nil
}

// PageUpdate replaces a page's title and storage-format body.
type PageUpdate struct {
	ID      string
	Title   string
	Body    string // Confluence storage format
	Version int    // the page's current version number, as last read
	Status  string // "current" or "draft"
	Message string // optional version message
}

// UpdatePage replaces a page's content.
//
// Published pages are saved as Version+1. Drafts are not versioned, so Confluence
// requires the number to stay at the current one.
func (c *Client) UpdatePage(ctx context.Context, u PageUpdate) error {
	next := u.Version + 1
	if u.Status == "draft" {
		next = u.Version
	}
	version := map[string]any{"number": next}
	if u.Message != "" {
		version["message"] = u.Message
	}
	payload, err := json.Marshal(map[string]any{
		"id":      u.ID,
		"type":    "page",
		"status":  u.Status,
		"title":   u.Title,
		"version": version,
		"body": map[string]any{
			"storage": map[string]any{"value": u.Body, "representation": "storage"},
		},
	})
	if err != nil {
		return fmt.Errorf("encode page update: %w", err)
	}

	path := "/content/" + url.PathEscape(u.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.base+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	if u.Status == "draft" {
		req.URL.RawQuery = url.Values{"status": {"draft"}}.Encode()
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, path, nil)
}

// NewPage describes a page to create.
type NewPage struct {
	SpaceKey string
	ParentID string // optional ancestor page
	Title    string
	Body     string // Confluence storage format
	Status   string // "current" publishes, "draft" does not
}

// CreatePage creates a page and returns it.
func (c *Client) CreatePage(ctx context.Context, p NewPage) (*Content, error) {
	payload := map[string]any{
		"type":   "page",
		"status": p.Status,
		"title":  p.Title,
		"space":  map[string]any{"key": p.SpaceKey},
		"body": map[string]any{
			"storage": map[string]any{"value": p.Body, "representation": "storage"},
		},
	}
	if p.ParentID != "" {
		payload["ancestors"] = []map[string]any{{"id": p.ParentID}}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode new page: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/content", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	var out Content
	if err := c.do(req, "/content", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddLabels attaches global labels to a page. Labels are lowercase with no
// spaces; Confluence rejects anything else.
func (c *Client) AddLabels(ctx context.Context, pageID string, labels ...string) error {
	body := make([]map[string]string, 0, len(labels))
	for _, l := range labels {
		body = append(body, map[string]string{"prefix": "global", "name": l})
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode labels: %w", err)
	}
	path := "/content/" + url.PathEscape(pageID) + "/label"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, path, nil)
}

// FindByLabel returns pages in the space carrying the label. CQL search only
// sees indexed content, so an unpublished draft may not appear.
func (c *Client) FindByLabel(ctx context.Context, spaceKey, label string) ([]Content, error) {
	var out struct {
		Results []Content `json:"results"`
	}
	cql := fmt.Sprintf("label = %q AND space = %q AND type = page", label, spaceKey)
	q := url.Values{"cql": {cql}, "expand": {"version"}}
	if err := c.get(ctx, "/content/search", q, &out); err != nil {
		return nil, err
	}
	return out.Results, nil
}
