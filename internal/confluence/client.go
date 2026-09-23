// Package confluence is a small client for the Confluence Cloud REST API
// (v1 content endpoints), reached through the api.atlassian.com gateway,
// which is the only host a scoped API token authenticates against.
package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bashfulrobot/walkr/internal/secrets"
)

const gatewayBase = "https://api.atlassian.com/ex/confluence"

// Client calls the Confluence REST API as one user.
type Client struct {
	base  string
	email string
	token secrets.Secret
	hc    *http.Client
}

// Option customizes a Client.
type Option func(*Client)

// WithBaseURL overrides the REST base URL (used by tests).
func WithBaseURL(u string) Option { return func(c *Client) { c.base = strings.TrimRight(u, "/") } }

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.hc = hc } }

// New builds a client for the site identified by cloudID.
func New(cloudID, email string, token secrets.Secret, opts ...Option) *Client {
	c := &Client{
		base:  gatewayBase + "/" + cloudID + "/wiki/rest/api",
		email: email,
		token: token,
		hc:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// APIError is a non-2xx response. It never carries request headers.
type APIError struct {
	Method string
	Path   string
	Status int
	Body   string
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("%s %s: HTTP %d", e.Method, e.Path, e.Status)
	if e.Body != "" {
		msg += ": " + e.Body
	}
	return msg
}

// Space is the subset of a space walkr reads.
type Space struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Content is the subset of a page walkr reads.
type Content struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Title   string `json:"title"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
}

// Spaces lists up to limit spaces. It is the cheapest authenticated read and
// so doubles as a credential check.
func (c *Client) Spaces(ctx context.Context, limit int) ([]Space, error) {
	var out struct {
		Results []Space `json:"results"`
	}
	q := url.Values{"limit": {fmt.Sprint(limit)}}
	if err := c.get(ctx, "/space", q, &out); err != nil {
		return nil, err
	}
	return out.Results, nil
}

// Content fetches one page or blog post by ID, in any status.
func (c *Client) Content(ctx context.Context, id string) (*Content, error) {
	var out Content
	q := url.Values{"status": {"any"}, "expand": {"version,body.storage"}}
	if err := c.get(ctx, "/content/"+url.PathEscape(id), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	if len(q) > 0 {
		req.URL.RawQuery = q.Encode()
	}
	return c.do(req, path, out)
}

func (c *Client) do(req *http.Request, path string, out any) error {
	req.SetBasicAuth(c.email, c.token.Reveal())
	req.Header.Set("Accept", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		// net/http errors carry the URL, never headers.
		return fmt.Errorf("%s %s: %w", req.Method, path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("%s %s: read body: %w", req.Method, path, err)
	}
	if resp.StatusCode/100 != 2 {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		return &APIError{Method: req.Method, Path: path, Status: resp.StatusCode, Body: snippet}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s %s: decode response: %w", req.Method, path, err)
	}
	return nil
}
