// Package config loads and persists aria2tui settings.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the persisted application configuration.
type Config struct {
	// RPCURL is the aria2 JSON-RPC websocket endpoint.
	RPCURL string `json:"rpc_url"`
	// Secret is the --rpc-secret token.
	Secret string `json:"secret"`
	// Aria2cPath overrides the aria2c binary location (empty = $PATH lookup).
	Aria2cPath string `json:"aria2c_path,omitempty"`
	// AutoSpawn starts a local aria2c daemon when no RPC endpoint answers.
	AutoSpawn bool `json:"auto_spawn"`
	// DownloadDir is the default destination directory.
	DownloadDir string `json:"download_dir"`
	// PollMS is the UI refresh interval in milliseconds.
	PollMS int `json:"poll_ms"`
	// HistoryLimit caps the number of retained history entries.
	HistoryLimit int `json:"history_limit"`
	// ExtraSpawnArgs are appended to the aria2c command line when spawning.
	ExtraSpawnArgs []string `json:"extra_spawn_args,omitempty"`

	// Default per-download options used to prefill the Add form.
	DefaultSplit   string `json:"default_split"`
	DefaultMaxConn string `json:"default_max_conn"`
}

// Dir returns the config directory (~/.config/aria2tui).
func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "aria2tui")
}

// DataDir returns the data directory (~/.local/share/aria2tui or platform equivalent).
func DataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "aria2tui")
}

// Path returns the config file path.
func Path() string { return filepath.Join(Dir(), "config.json") }

// Default builds a fresh config with a random RPC secret.
func Default() *Config {
	home, _ := os.UserHomeDir()
	dl := filepath.Join(home, "Downloads")
	return &Config{
		RPCURL:         "ws://127.0.0.1:6800/jsonrpc",
		Secret:         randomSecret(),
		AutoSpawn:      true,
		DownloadDir:    dl,
		PollMS:         700,
		HistoryLimit:   2000,
		DefaultSplit:   "16",
		DefaultMaxConn: "16",
	}
}

func randomSecret() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "aria2tui"
	}
	return hex.EncodeToString(b)
}

// Load reads the config file, creating it with defaults on first run.
func Load() (*Config, error) {
	p := Path()
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		cfg := Default()
		if err := cfg.Save(); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.PollMS < 200 {
		cfg.PollMS = 700
	}
	if cfg.HistoryLimit <= 0 {
		cfg.HistoryLimit = 2000
	}
	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0o600)
}
