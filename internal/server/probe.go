package server

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

// GET /api/probe?url= — IDM-style pre-download link inspection, executed on
// the server so it shares the aria2 daemon's network view.
func (s *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("url")
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("probe supports http(s) URLs"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	res, err := probeURL(ctx, raw)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ProbeResult describes what the remote server offers for a URL.
type ProbeResult struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
	Resumable   bool   `json:"resumable"`
	Via         string `json:"via"` // accept-ranges | range-request | none
	FinalURL    string `json:"finalUrl"`
}

func probeURL(ctx context.Context, raw string) (*ProbeResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "tidefetch/probe")

	out := &ProbeResult{FinalURL: raw}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		fill(out, resp)
		if resp.Header.Get("Accept-Ranges") == "bytes" {
			out.Resumable = true
			out.Via = "accept-ranges"
			return out, nil
		}
	}

	// Fallback: 1-byte ranged GET tells the truth even when HEAD lies.
	greq, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return out, nil
	}
	greq.Header.Set("User-Agent", "tidefetch/probe")
	greq.Header.Set("Range", "bytes=0-0")
	gresp, gerr := client.Do(greq)
	if gerr != nil {
		if out.Via == "" {
			out.Via = "none"
		}
		return out, nil
	}
	defer gresp.Body.Close()
	_, _ = io.CopyN(io.Discard, gresp.Body, 2)

	if out.Filename == "" || out.Size == 0 {
		fill(out, gresp)
	}
	if gresp.StatusCode == http.StatusPartialContent {
		out.Resumable = true
		out.Via = "range-request"
		// Content-Range: bytes 0-0/12345 → total size.
		if cr := gresp.Header.Get("Content-Range"); cr != "" {
			if i := strings.LastIndex(cr, "/"); i >= 0 {
				if n, err := strconv.ParseInt(cr[i+1:], 10, 64); err == nil {
					out.Size = n
				}
			}
		}
	} else if out.Via == "" {
		out.Via = "none"
	}
	return out, nil
}

func fill(out *ProbeResult, resp *http.Response) {
	out.FinalURL = resp.Request.URL.String()
	if out.Size == 0 && resp.ContentLength > 0 {
		out.Size = resp.ContentLength
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		out.ContentType = strings.SplitN(ct, ";", 2)[0]
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if fn := params["filename"]; fn != "" {
				out.Filename = fn
			}
		}
	}
	if out.Filename == "" {
		out.Filename = path.Base(resp.Request.URL.Path)
		if out.Filename == "/" || out.Filename == "." {
			out.Filename = ""
		}
	}
}
