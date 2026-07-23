// Command aria2tui is a fast, good looking terminal UI for the aria2 download
// utility — pause/resume, torrents, metalinks, queue control, per-download
// details, download history, live speed charts and more.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/turbostart/aria2c-tui/internal/config"
	"github.com/turbostart/aria2c-tui/internal/daemon"
	"github.com/turbostart/aria2c-tui/internal/history"
	"github.com/turbostart/aria2c-tui/internal/ui"
)

var version = "0.1.0"

func main() {
	var (
		flagURL     = flag.String("url", "", "aria2 RPC endpoint (e.g. ws://127.0.0.1:6800/jsonrpc); overrides config")
		flagSecret  = flag.String("secret", "", "aria2 RPC secret; overrides config")
		flagDir     = flag.String("dir", "", "default download directory; overrides config")
		flagNoSpawn = flag.Bool("no-spawn", false, "never spawn a local aria2c daemon")
		flagVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "aria2tui — a terminal UI for aria2\n\n")
		fmt.Fprintf(os.Stderr, "usage: %s [flags] [URL ...]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "URLs passed as arguments are queued immediately on startup.\n\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *flagVersion {
		fmt.Println("aria2tui", version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("load config: %v", err)
	}
	if *flagURL != "" {
		cfg.RPCURL = *flagURL
	}
	if *flagSecret != "" {
		cfg.Secret = *flagSecret
	}
	if *flagDir != "" {
		cfg.DownloadDir = *flagDir
	}
	if *flagNoSpawn {
		cfg.AutoSpawn = false
	}

	res, err := daemon.Connect(context.Background(), cfg)
	if err != nil {
		fatal("%v", err)
	}

	histPath := filepath.Join(config.DataDir(), "history.json")
	hist, err := history.Open(histPath, cfg.HistoryLimit)
	if err != nil {
		fatal("open history: %v", err)
	}

	app := ui.New(cfg, hist, res, flag.Args())
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fatal("%v", err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "aria2tui: "+format+"\n", args...)
	os.Exit(1)
}
