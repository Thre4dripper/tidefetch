// Package daemon locates, spawns and connects to a local aria2c process.
package daemon

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Thre4dripper/tidefetch/internal/config"
	"github.com/Thre4dripper/tidefetch/pkg/aria2"
)

// Result describes how the RPC connection was established.
type Result struct {
	Client  *aria2.Client
	Version string
	// Spawned is true when this process started the aria2c daemon.
	Spawned bool
	// PID of the spawned daemon (0 if attached to an existing one).
	PID int
	// URL actually connected to (may differ from config when port was busy).
	URL string
}

// Connect attaches to a running aria2 RPC endpoint, or — when allowed —
// spawns a local aria2c daemon with sane defaults and connects to it.
func Connect(ctx context.Context, cfg *config.Config) (*Result, error) {
	// 1. Try the configured endpoint first: maybe a daemon is already running.
	if c, err := tryDial(ctx, cfg.RPCURL, cfg.Secret); err == nil {
		v, _ := c.GetVersion(ctx)
		return &Result{Client: c, Version: v.Version, URL: cfg.RPCURL}, nil
	}

	if !cfg.AutoSpawn {
		return nil, fmt.Errorf("cannot reach aria2 RPC at %s and auto-spawn is disabled", cfg.RPCURL)
	}

	bin := cfg.Aria2cPath
	if bin == "" {
		var err error
		bin, err = exec.LookPath("aria2c")
		if err != nil {
			return nil, fmt.Errorf("aria2c binary not found in PATH — install it first (brew install aria2 / apt install aria2)")
		}
	}

	port := portOf(cfg.RPCURL)
	if port == 0 || !portFree(port) {
		var err error
		port, err = freePort()
		if err != nil {
			return nil, fmt.Errorf("no free port for aria2 RPC: %w", err)
		}
	}

	dataDir := config.DataDir()
	_ = os.MkdirAll(dataDir, 0o755)
	sessionFile := filepath.Join(dataDir, "session.aria2")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		_ = os.WriteFile(sessionFile, nil, 0o600)
	}
	_ = os.MkdirAll(cfg.DownloadDir, 0o755)

	args := []string{
		"--enable-rpc=true",
		"--rpc-listen-all=false",
		fmt.Sprintf("--rpc-listen-port=%d", port),
		"--rpc-secret=" + cfg.Secret,
		"--dir=" + cfg.DownloadDir,
		"--continue=true",
		"--input-file=" + sessionFile,
		"--save-session=" + sessionFile,
		"--save-session-interval=20",
		"--auto-save-interval=20",
		"--max-concurrent-downloads=5",
		"--file-allocation=none",
		"--bt-save-metadata=true",
		"--follow-torrent=true",
		"--summary-interval=0",
		"--quiet=true",
	}
	args = append(args, cfg.ExtraSpawnArgs...)

	cmd := exec.Command(bin, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("spawn aria2c: %w", err)
	}
	// Detach: we do not reap it; it should outlive the UI unless shut down via RPC.
	go func() { _ = cmd.Wait() }()

	rpcURL := fmt.Sprintf("ws://127.0.0.1:%d/jsonrpc", port)
	deadline := time.Now().Add(6 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		c, err := tryDial(ctx, rpcURL, cfg.Secret)
		if err == nil {
			v, _ := c.GetVersion(ctx)
			return &Result{Client: c, Version: v.Version, Spawned: true, PID: cmd.Process.Pid, URL: rpcURL}, nil
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	return nil, fmt.Errorf("spawned aria2c but RPC never came up: %w", lastErr)
}

// Redial re-establishes a client connection to url.
func Redial(ctx context.Context, url, secret string) (*aria2.Client, error) {
	return tryDial(ctx, url, secret)
}

func tryDial(ctx context.Context, url, secret string) (*aria2.Client, error) {
	dctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	c, err := aria2.Dial(dctx, url, secret)
	if err != nil {
		return nil, err
	}
	// Validate the secret actually works.
	if _, err := c.GetVersion(dctx); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func portOf(rpcURL string) int {
	u, err := url.Parse(rpcURL)
	if err != nil {
		return 0
	}
	p, _ := strconv.Atoi(u.Port())
	return p
}

func portFree(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
