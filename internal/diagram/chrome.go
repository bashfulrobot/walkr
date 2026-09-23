// Package diagram renders Mermaid source to PNG with a local headless Chrome,
// using the Mermaid build already embedded in walkr. Nothing leaves the machine.
package diagram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	cruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/bashfulrobot/walkr/assets"
)

// ErrNoBrowser means no Chrome or Chromium could be found.
var ErrNoBrowser = errors.New("no Chrome or Chromium found (set WALKR_CHROME to its path)")

const renderTimeout = 45 * time.Second

// pageHTML hosts one diagram. The theme is a muted two-tone, matching the
// palette of the Confluence pages the images sit on.
const pageHTML = `<!doctype html>
<html><head><meta charset="utf-8">
<style>html,body{margin:0;background:#fff}#d{display:inline-block;padding:24px;background:#fff}</style>
</head><body><div id="d"></div>
<script src="/mermaid.min.js"></script>
<script>
mermaid.initialize({
  startOnLoad: false,
  theme: 'base',
  fontFamily: '-apple-system, "Segoe UI", Helvetica, Arial, sans-serif',
  flowchart: { useMaxWidth: false, htmlLabels: true },
  themeVariables: {
    background: '#ffffff', primaryColor: '#DEEBFF', primaryBorderColor: '#B3BAC5',
    primaryTextColor: '#172B4D', secondaryColor: '#F4F5F7', tertiaryColor: '#F4F5F7',
    lineColor: '#6B778C', textColor: '#172B4D', edgeLabelBackground: '#ffffff', fontSize: '15px'
  }
});
window.renderDiagram = async (src) => {
  const { svg } = await mermaid.render('walkr-diagram', src);
  document.getElementById('d').innerHTML = svg;
  const r = document.getElementById('d').getBoundingClientRect();
  return [r.width, r.height];
};
</script></body></html>`

// Chrome renders diagrams through one shared headless browser.
type Chrome struct {
	execPath string

	once   sync.Once
	err    error
	alloc  context.Context
	cancel context.CancelFunc
	srv    *http.Server
	url    string
}

// NewChrome locates a browser. It does not start it until the first Render.
func NewChrome() (*Chrome, error) {
	path, err := findBrowser()
	if err != nil {
		return nil, err
	}
	return &Chrome{execPath: path}, nil
}

// Path is the browser executable in use.
func (c *Chrome) Path() string { return c.execPath }

func findBrowser() (string, error) {
	if p := os.Getenv("WALKR_CHROME"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("WALKR_CHROME %q: %w", p, err)
		}
		return p, nil
	}
	var candidates []string
	if runtime.GOOS == "darwin" {
		candidates = append(candidates,
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", ErrNoBrowser
}

func (c *Chrome) start() error {
	c.once.Do(func() {
		mermaid, err := fs.ReadFile(assets.Vendor, "vendor/mermaid.min.js")
		if err != nil {
			c.err = fmt.Errorf("embedded mermaid: %w", err)
			return
		}
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			c.err = err
			return
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(pageHTML))
		})
		mux.HandleFunc("/mermaid.min.js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write(mermaid)
		})
		c.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
		go func() { _ = c.srv.Serve(ln) }()
		c.url = "http://" + ln.Addr().String() + "/"

		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(c.execPath),
			chromedp.Flag("hide-scrollbars", true),
		)
		c.alloc, c.cancel = chromedp.NewExecAllocator(context.Background(), opts...)
	})
	return c.err
}

// Render returns the diagram as a PNG at twice the CSS pixel size.
func (c *Chrome) Render(ctx context.Context, mermaid string) ([]byte, error) {
	if err := c.start(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()
	tab, cancelTab := chromedp.NewContext(c.alloc)
	defer cancelTab()
	stop := context.AfterFunc(ctx, cancelTab)
	defer stop()

	src, err := json.Marshal(mermaid)
	if err != nil {
		return nil, err
	}
	var size []float64
	var png []byte
	err = chromedp.Run(tab,
		chromedp.Navigate(c.url),
		chromedp.Evaluate(`window.renderDiagram(`+string(src)+`)`, &size,
			func(p *cruntime.EvaluateParams) *cruntime.EvaluateParams { return p.WithAwaitPromise(true) }),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(size) != 2 || size[0] < 1 || size[1] < 1 {
				return fmt.Errorf("diagram has no size: %v", size)
			}
			w, h := int64(math.Ceil(size[0])), int64(math.Ceil(size[1]))
			if err := emulation.SetDeviceMetricsOverride(w, h, 2, false).Do(ctx); err != nil {
				return err
			}
			var shotErr error
			png, shotErr = page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatPng).
				WithClip(&page.Viewport{X: 0, Y: 0, Width: float64(w), Height: float64(h), Scale: 1}).
				Do(ctx)
			return shotErr
		}),
	)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("render timed out or was canceled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("render diagram: %w", err)
	}
	return png, nil
}

// Close stops the browser and the local server.
func (c *Chrome) Close() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.srv != nil {
		_ = c.srv.Close()
	}
}
