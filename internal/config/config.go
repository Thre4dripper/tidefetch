// Package config loads and persists tidefetch settings.
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
	// Terminal UI preferences.
	Sidebar       bool `json:"sidebar"`
	CompactRows   bool `json:"compact_rows"`
	ConfirmRemove bool `json:"confirm_remove"`
	// ExtraSpawnArgs are appended to the aria2c command line when spawning.
	ExtraSpawnArgs []string `json:"extra_spawn_args,omitempty"`

	// Default per-download options used to prefill the Add form.
	DefaultSplit   string `json:"default_split"`
	DefaultMaxConn string `json:"default_max_conn"`

	// Web UI server settings (used by `tidefetch serve`).
	WebHost string `json:"web_host,omitempty"`
	WebPort int    `json:"web_port,omitempty"`
	// WebPasswordHash is a bcrypt hash of the web UI password.
	WebPasswordHash string `json:"web_password_hash,omitempty"`
}

// Dir returns the config directory (~/.config/tidefetch).
func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "tidefetch")
}

// DataDir returns the data directory (~/.local/share/tidefetch or platform equivalent).
func DataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "tidefetch")
}

// legacyDirs returns the pre-rename (aria2tui) config and data directories.
func legacyDirs() (cfgDir, dataDir string) {
	base, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(base, "aria2tui"), filepath.Join(home, ".local", "share", "aria2tui")
}

// migrateLegacy moves aria2tui config/data dirs to the tidefetch locations
// the first time the new paths are used.
func migrateLegacy() {
	oldCfg, oldData := legacyDirs()
	if _, err := os.Stat(Dir()); os.IsNotExist(err) {
		if _, err := os.Stat(oldCfg); err == nil {
			_ = os.Rename(oldCfg, Dir())
		}
	}
	if _, err := os.Stat(DataDir()); os.IsNotExist(err) {
		if _, err := os.Stat(oldData); err == nil {
			_ = os.MkdirAll(filepath.Dir(DataDir()), 0o755)
			_ = os.Rename(oldData, DataDir())
		}
	}
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
		Sidebar:        true,
		ConfirmRemove:  true,
		DefaultSplit:   "16",
		DefaultMaxConn: "16",
		WebHost:        "127.0.0.1",
		WebPort:        8210,
	}
}

func randomSecret() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "tidefetch"
	}
	return hex.EncodeToString(b)
}

// Load reads the config file, creating it with defaults on first run.
func Load() (*Config, error) {
	migrateLegacy()
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
	if cfg.WebHost == "" {
		cfg.WebHost = "127.0.0.1"
	}
	if cfg.WebPort <= 0 {
		cfg.WebPort = 8210
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
