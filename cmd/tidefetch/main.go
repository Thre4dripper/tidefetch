// Command tidefetch is a fast terminal UI and self-hosted web UI for the
// aria2 download engine — pause/resume, torrents, metalinks, queue control,
// per-download details, download history, live speed charts and more.
//
//	tidefetch              open the terminal UI
//	tidefetch serve        start the web UI server
//	tidefetch doctor       diagnose the local setup
//	tidefetch version      print the version
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Thre4dripper/tidefetch/internal/config"
	"github.com/Thre4dripper/tidefetch/internal/daemon"
	"github.com/Thre4dripper/tidefetch/internal/history"
	"github.com/Thre4dripper/tidefetch/internal/server"
	"github.com/Thre4dripper/tidefetch/internal/tui"
)

// Overridden at build time via -ldflags; this is only the fallback for
// `go install` and plain `go build`.
var version = "0.2.0"

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "serve":
			runServe(args[1:])
			return
		case "doctor":
			runDoctor()
			return
		case "version", "-version", "--version":
			fmt.Println("tidefetch", version)
			return
		case "help", "-h", "--help":
			usage()
			return
		}
	}
	runTUI(args)
}

func usage() {
	fmt.Fprintf(os.Stderr, `tidefetch — a TUI + web UI for the aria2 download engine

usage:
  tidefetch [flags] [URL ...]   open the terminal UI (URLs are queued on startup)
  tidefetch serve [flags]       start the web UI server
  tidefetch doctor              diagnose aria2, config and connectivity
  tidefetch version             print the version

tui flags:
`)
	fs := flag.NewFlagSet("tidefetch", flag.ContinueOnError)
	addCommonFlags(fs)
	fs.Bool("version", false, "print version and exit")
	fs.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nserve flags:\n")
	sfs, _ := serveFlags()
	sfs.PrintDefaults()
}

type commonFlags struct {
	url, secret, dir *string
	noSpawn          *bool
}

func addCommonFlags(fs *flag.FlagSet) commonFlags {
	return commonFlags{
		url:     fs.String("url", "", "aria2 RPC endpoint (e.g. ws://127.0.0.1:6800/jsonrpc); overrides config"),
		secret:  fs.String("secret", "", "aria2 RPC secret; overrides config"),
		dir:     fs.String("dir", "", "default download directory; overrides config"),
		noSpawn: fs.Bool("no-spawn", false, "never spawn a local aria2c daemon"),
	}
}

func (cf commonFlags) apply(cfg *config.Config) {
	if *cf.url != "" {
		cfg.RPCURL = *cf.url
	}
	if *cf.secret != "" {
		cfg.Secret = *cf.secret
	}
	if *cf.dir != "" {
		cfg.DownloadDir = *cf.dir
	}
	if *cf.noSpawn {
		cfg.AutoSpawn = false
	}
}

func runTUI(args []string) {
	fs := flag.NewFlagSet("tidefetch", flag.ExitOnError)
	cf := addCommonFlags(fs)
	flagVersion := fs.Bool("version", false, "print version and exit")
	fs.Usage = usage
	_ = fs.Parse(args)

	if *flagVersion {
		fmt.Println("tidefetch", version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("load config: %v", err)
	}
	cf.apply(cfg)

	res, err := daemon.Connect(context.Background(), cfg)
	if err != nil {
		fatal("%v", err)
	}

	hist, err := openHistory(cfg)
	if err != nil {
		fatal("open history: %v", err)
	}

	app := tui.New(cfg, hist, res, fs.Args())
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fatal("%v", err)
	}
}

func serveFlags() (*flag.FlagSet, *server.Flags) {
	fs := flag.NewFlagSet("tidefetch serve", flag.ExitOnError)
	sf := &server.Flags{}
	fs.StringVar(&sf.Host, "host", "", "listen address (default from config, 127.0.0.1)")
	fs.IntVar(&sf.Port, "port", 0, "listen port (default from config, 8210)")
	fs.StringVar(&sf.Password, "password", "", "set/replace the web UI password (stored as a bcrypt hash)")
	fs.BoolVar(&sf.NoAuth, "no-auth", false, "disable authentication (loopback binds, or behind your own auth proxy)")
	fs.StringVar(&sf.URL, "url", "", "aria2 RPC endpoint; overrides config")
	fs.StringVar(&sf.Secret, "secret", "", "aria2 RPC secret; overrides config")
	fs.StringVar(&sf.Dir, "dir", "", "default download directory; overrides config")
	fs.BoolVar(&sf.NoSpawn, "no-spawn", false, "never spawn a local aria2c daemon")
	return fs, sf
}

func runServe(args []string) {
	fs, sf := serveFlags()
	_ = fs.Parse(args)

	// Container-friendly: allow a direct value or a mounted secret file.
	if sf.Password == "" {
		password, err := passwordFromEnvironment()
		if err != nil {
			fatal("read web password: %v", err)
		}
		sf.Password = password
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("load config: %v", err)
	}
	if sf.URL != "" {
		cfg.RPCURL = sf.URL
	}
	if sf.Secret != "" {
		cfg.Secret = sf.Secret
	}
	if sf.Dir != "" {
		cfg.DownloadDir = sf.Dir
	}
	if sf.NoSpawn {
		cfg.AutoSpawn = false
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	res, err := daemon.Connect(ctx, cfg)
	if err != nil {
		fatal("%v", err)
	}

	hist, err := openHistory(cfg)
	if err != nil {
		fatal("open history: %v", err)
	}

	srv, err := server.New(cfg, hist, res, version, sf)
	if err != nil {
		fatal("%v", err)
	}
	if err := srv.Run(ctx); err != nil {
		fatal("%v", err)
	}
}

func passwordFromEnvironment() (string, error) {
	if password := os.Getenv("TIDEFETCH_PASSWORD"); password != "" {
		return password, nil
	}
	path := os.Getenv("TIDEFETCH_PASSWORD_FILE")
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func openHistory(cfg *config.Config) (*history.Store, error) {
	histPath := filepath.Join(config.DataDir(), "history.json")
	return history.Open(histPath, cfg.HistoryLimit)
}

func runDoctor() {
	ok := func(b bool) string {
		if b {
			return "ok"
		}
		return "FAIL"
	}

	fmt.Println("tidefetch doctor")
	fmt.Println("────────────────")

	cfg, err := config.Load()
	fmt.Printf("config   %-4s  %s\n", ok(err == nil), config.Path())
	if err != nil {
		fmt.Printf("         %v\n", err)
		return
	}

	bin := cfg.Aria2cPath
	var lookErr error
	if bin == "" {
		bin, lookErr = exec.LookPath("aria2c")
	}
	if lookErr != nil || bin == "" {
		fmt.Printf("aria2c   FAIL  not found in PATH — install it (brew install aria2 / apt install aria2)\n")
	} else {
		ver := ""
		if out, verr := exec.Command(bin, "--version").Output(); verr == nil {
			ver = strings.SplitN(string(out), "\n", 2)[0]
		}
		fmt.Printf("aria2c   ok    %s (%s)\n", bin, ver)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if c, derr := daemon.Redial(ctx, cfg.RPCURL, cfg.Secret); derr == nil {
		v, _ := c.GetVersion(ctx)
		fmt.Printf("rpc      ok    %s (aria2 %s)\n", cfg.RPCURL, v.Version)
		_ = c.Close()
	} else {
		fmt.Printf("rpc      —     %s not reachable (a daemon will be spawned on demand)\n", cfg.RPCURL)
	}

	st, serr := os.Stat(cfg.DownloadDir)
	writable := serr == nil && st.IsDir()
	if writable {
		probe := filepath.Join(cfg.DownloadDir, ".tidefetch-probe")
		if werr := os.WriteFile(probe, nil, 0o600); werr == nil {
			_ = os.Remove(probe)
		} else {
			writable = false
		}
	}
	fmt.Printf("dir      %-4s  %s\n", ok(writable), cfg.DownloadDir)
	fmt.Printf("data     ok    %s\n", config.DataDir())
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "tidefetch: "+format+"\n", args...)
	os.Exit(1)
}
