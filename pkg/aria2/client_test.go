package aria2_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/turbostart/tidefetch/pkg/aria2"
)

// TestIntegration spins up a real aria2c daemon and exercises the client:
// dial, version, addUri, status polling, notifications, multicall, shutdown.
func TestIntegration(t *testing.T) {
	bin, err := exec.LookPath("aria2c")
	if err != nil {
		t.Skip("aria2c not installed; skipping integration test")
	}

	// Free port for RPC.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	dir := t.TempDir()
	secret := "testsecret"
	cmd := exec.Command(bin,
		"--enable-rpc=true",
		"--rpc-listen-all=false",
		fmt.Sprintf("--rpc-listen-port=%d", port),
		"--rpc-secret="+secret,
		"--dir="+dir,
		"--quiet=true",
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	// Local HTTP server with a 1 MiB payload.
	payload := make([]byte, 1<<20)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for RPC to come up.
	var c *aria2.Client
	url := fmt.Sprintf("ws://127.0.0.1:%d/jsonrpc", port)
	deadline := time.Now().Add(5 * time.Second)
	for {
		c, err = aria2.Dial(ctx, url, secret)
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	v, err := c.GetVersion(ctx)
	if err != nil {
		t.Fatalf("getVersion: %v", err)
	}
	t.Logf("aria2 version %s", v.Version)

	// Add a download and watch it complete.
	gid, err := c.AddURI(ctx, []string{srv.URL + "/blob.bin"}, aria2.Options{aria2.OptOut: "blob.bin"})
	if err != nil {
		t.Fatalf("addUri: %v", err)
	}
	if gid == "" {
		t.Fatal("empty gid")
	}

	var st aria2.Status
	for i := 0; i < 100; i++ {
		st, err = c.TellStatus(ctx, gid)
		if err != nil {
			t.Fatalf("tellStatus: %v", err)
		}
		if st.Status == aria2.StatusComplete || st.Status == aria2.StatusError {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if st.Status != aria2.StatusComplete {
		t.Fatalf("download did not complete: %s %s", st.Status, st.ErrorMessage)
	}
	if st.TotalLength.Int() != int64(len(payload)) {
		t.Fatalf("size mismatch: %d != %d", st.TotalLength.Int(), len(payload))
	}
	if got := st.Name(); got != "blob.bin" {
		t.Fatalf("name: %q", got)
	}
	data, err := os.ReadFile(filepath.Join(dir, "blob.bin"))
	if err != nil || len(data) != len(payload) {
		t.Fatalf("file on disk: %v len=%d", err, len(data))
	}

	// Notifications should include start + complete for our gid.
	seenStart, seenComplete := false, false
	timeout := time.After(2 * time.Second)
drain:
	for {
		select {
		case n := <-c.Notifications():
			if n.GID == gid && n.Method == aria2.EventStart {
				seenStart = true
			}
			if n.GID == gid && n.Method == aria2.EventComplete {
				seenComplete = true
			}
			if seenStart && seenComplete {
				break drain
			}
		case <-timeout:
			break drain
		}
	}
	if !seenStart || !seenComplete {
		t.Fatalf("notifications missing: start=%v complete=%v", seenStart, seenComplete)
	}

	// Multicall: version + stat in one round trip.
	results, errs, err := c.Multicall(ctx, []aria2.MulticallCall{
		{Method: "aria2.getVersion"},
		{Method: "aria2.getGlobalStat"},
	})
	if err != nil {
		t.Fatalf("multicall: %v", err)
	}
	if len(results) != 2 || errs[0] != nil || errs[1] != nil {
		t.Fatalf("multicall results: %v errs: %v", results, errs)
	}

	// Stopped list should contain our gid.
	stopped, err := c.TellStopped(ctx, 0, 10)
	if err != nil {
		t.Fatalf("tellStopped: %v", err)
	}
	found := false
	for _, s := range stopped {
		if s.GID == gid {
			found = true
		}
	}
	if !found {
		t.Fatal("gid not in stopped list")
	}

	// Global options round trip.
	if err := c.ChangeGlobalOption(ctx, aria2.Options{aria2.OptMaxOverallDownloadLimit: "500K"}); err != nil {
		t.Fatalf("changeGlobalOption: %v", err)
	}
	opts, err := c.GetGlobalOption(ctx)
	if err != nil || opts[aria2.OptMaxOverallDownloadLimit] != "512000" {
		t.Fatalf("global option: %v %q", err, opts[aria2.OptMaxOverallDownloadLimit])
	}

	if err := c.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
