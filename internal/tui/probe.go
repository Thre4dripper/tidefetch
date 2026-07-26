package tui

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// probeResult is what we learn about a URL before downloading it —
// IDM-style link inspection: resumability, size, server-provided filename.
type probeResult struct {
	url         string
	finalURL    string
	filename    string
	contentType string
	size        int64
	resumable   bool
	how         string // what proved resumability
	err         error
}

// probeMsg delivers an asynchronous probe outcome to the UI.
type probeMsg struct{ res probeResult }

// probeCmd inspects a URL with a HEAD request, falling back to a 1-byte
// ranged GET (some servers reject HEAD but still support ranges).
func probeCmd(url string) tea.Cmd {
	return func() tea.Msg { return probeMsg{res: probeURL(url)} }
}

func probeURL(url string) probeResult {
	res := probeResult{url: url}
	if strings.HasPrefix(strings.ToLower(url), "magnet:") {
		res.err = fmt.Errorf("magnet links cannot be probed")
		return res
	}
	client := &http.Client{Timeout: 8 * time.Second}

	// 1. HEAD
	if req, err := http.NewRequest(http.MethodHead, url, nil); err == nil {
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
			if resp.StatusCode < 400 {
				res.absorb(resp)
				if strings.Contains(strings.ToLower(resp.Header.Get("Accept-Ranges")), "bytes") {
					res.resumable = true
					res.how = "accept-ranges"
					return res
				}
			}
		}
	}

	// 2. GET with Range: bytes=0-0 — a 206 response proves resumability.
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		res.err = err
		return res
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	if err != nil {
		if res.finalURL == "" {
			res.err = err
		}
		return res
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		if res.finalURL == "" {
			res.err = fmt.Errorf("server answered %s", resp.Status)
		}
		return res
	}
	res.absorb(resp)
	if resp.StatusCode == http.StatusPartialContent {
		res.resumable = true
		res.how = "range request"
		// Content-Range: bytes 0-0/123456 carries the real total.
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			if i := strings.LastIndex(cr, "/"); i >= 0 {
				if n, err := strconv.ParseInt(cr[i+1:], 10, 64); err == nil {
					res.size = n
				}
			}
		}
	}
	return res
}

// absorb pulls metadata out of a response.
func (r *probeResult) absorb(resp *http.Response) {
	r.finalURL = resp.Request.URL.String()
	if r.contentType == "" {
		r.contentType, _, _ = strings.Cut(resp.Header.Get("Content-Type"), ";")
	}
	if r.size == 0 {
		if n, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64); err == nil && n > 1 {
			r.size = n
		}
	}
	if r.filename == "" {
		if cd := resp.Header.Get("Content-Disposition"); cd != "" {
			if _, params, err := mime.ParseMediaType(cd); err == nil {
				if fn := params["filename"]; fn != "" {
					r.filename = fn
				}
			}
		}
	}
	if r.filename == "" {
		p := resp.Request.URL.Path
		if i := strings.LastIndex(p, "/"); i >= 0 && i+1 < len(p) {
			r.filename = p[i+1:]
		}
	}
}

// render formats the probe outcome as a short status line.
func (r *probeResult) render(width int) string {
	if r.err != nil {
		return styleBad.Render("⌕ " + truncate(r.err.Error(), width-3))
	}
	var parts []string
	if r.filename != "" {
		parts = append(parts, styleTitle.Render(truncate(r.filename, 40)))
	}
	if r.size > 0 {
		parts = append(parts, styleText.Render(humanBytes(r.size)))
	}
	if r.contentType != "" {
		parts = append(parts, styleDim.Render(r.contentType))
	}
	if r.resumable {
		parts = append(parts, styleGood.Render("resumable ✓ ("+r.how+")"))
	} else {
		parts = append(parts, styleWarn.Render("resume unsupported ✗"))
	}
	return truncate(styleCyan.Render("⌕ ")+strings.Join(parts, styleFaint.Render(" · ")), width)
}
