// Package aria2 is a standalone Go client for the aria2 JSON-RPC interface
// (https://aria2.github.io/manual/en/html/aria2c.html#rpc-interface).
//
// It speaks JSON-RPC 2.0 over a WebSocket connection, supports the complete
// aria2 method surface, typed results, system.multicall batching and
// asynchronous event notifications (download started / completed / errored…).
//
// It has no dependency on the TUI and can be imported on its own:
//
//	client, err := aria2.Dial(ctx, "ws://127.0.0.1:6800/jsonrpc", "secret")
//	gid, err := client.AddURI(ctx, []string{"https://example.org/file.iso"}, aria2.Options{"dir": "/tmp"})
package aria2

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Options is a set of aria2 input options (all values are strings, as aria2 expects).
type Options map[string]string

// Size is an int64 that unmarshals from aria2's quoted decimal strings ("12345").
type Size int64

// UnmarshalJSON implements json.Unmarshaler.
func (s *Size) UnmarshalJSON(b []byte) error {
	str := strings.Trim(string(b), `"`)
	if str == "" || str == "null" {
		*s = 0
		return nil
	}
	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("aria2: parse size %q: %w", str, err)
	}
	*s = Size(v)
	return nil
}

// Int returns the value as int64.
func (s Size) Int() int64 { return int64(s) }

// Bool is a bool that unmarshals from aria2's "true"/"false" strings.
type Bool bool

// UnmarshalJSON implements json.Unmarshaler.
func (b *Bool) UnmarshalJSON(data []byte) error {
	*b = Bool(strings.Trim(string(data), `"`) == "true")
	return nil
}

// Download states reported by aria2.
const (
	StatusActive   = "active"
	StatusWaiting  = "waiting"
	StatusPaused   = "paused"
	StatusError    = "error"
	StatusComplete = "complete"
	StatusRemoved  = "removed"
)

// Status is the result of aria2.tellStatus / tellActive / tellWaiting / tellStopped.
type Status struct {
	GID             string   `json:"gid"`
	Status          string   `json:"status"`
	TotalLength     Size     `json:"totalLength"`
	CompletedLength Size     `json:"completedLength"`
	UploadLength    Size     `json:"uploadLength"`
	Bitfield        string   `json:"bitfield"`
	DownloadSpeed   Size     `json:"downloadSpeed"`
	UploadSpeed     Size     `json:"uploadSpeed"`
	InfoHash        string   `json:"infoHash"`
	NumSeeders      Size     `json:"numSeeders"`
	Seeder          Bool     `json:"seeder"`
	PieceLength     Size     `json:"pieceLength"`
	NumPieces       Size     `json:"numPieces"`
	Connections     Size     `json:"connections"`
	ErrorCode       string   `json:"errorCode"`
	ErrorMessage    string   `json:"errorMessage"`
	FollowedBy      []string `json:"followedBy"`
	Following       string   `json:"following"`
	BelongsTo       string   `json:"belongsTo"`
	Dir             string   `json:"dir"`
	Files           []File   `json:"files"`
	BitTorrent      *BTInfo  `json:"bittorrent"`
	VerifiedLength  Size     `json:"verifiedLength"`
	VerifyPending   Bool     `json:"verifyIntegrityPending"`
}

// Name derives a human friendly display name for the download.
func (s Status) Name() string {
	if s.BitTorrent != nil && s.BitTorrent.Info.Name != "" {
		return s.BitTorrent.Info.Name
	}
	if len(s.Files) > 0 {
		f := s.Files[0]
		if f.Path != "" {
			if strings.HasPrefix(f.Path, "[METADATA]") {
				return f.Path
			}
			return baseName(f.Path)
		}
		if len(f.URIs) > 0 && f.URIs[0].URI != "" {
			return baseName(strings.SplitN(f.URIs[0].URI, "?", 2)[0])
		}
	}
	if s.InfoHash != "" {
		return s.InfoHash
	}
	return s.GID
}

func baseName(p string) string {
	p = strings.TrimRight(p, "/")
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		p = p[i+1:]
	}
	return p
}

// Progress returns completion in [0,1].
func (s Status) Progress() float64 {
	if s.TotalLength <= 0 {
		return 0
	}
	return float64(s.CompletedLength) / float64(s.TotalLength)
}

// PrimaryURI returns the first URI attached to the download, if any.
func (s Status) PrimaryURI() string {
	for _, f := range s.Files {
		for _, u := range f.URIs {
			if u.URI != "" {
				return u.URI
			}
		}
	}
	return ""
}

// IsTorrent reports whether the download is a BitTorrent transfer.
func (s Status) IsTorrent() bool { return s.BitTorrent != nil || s.InfoHash != "" }

// File is one file inside a download.
type File struct {
	Index           Size   `json:"index"`
	Path            string `json:"path"`
	Length          Size   `json:"length"`
	CompletedLength Size   `json:"completedLength"`
	Selected        Bool   `json:"selected"`
	URIs            []URI  `json:"uris"`
}

// URI is a source address with its status ("used" or "waiting").
type URI struct {
	URI    string `json:"uri"`
	Status string `json:"status"`
}

// BTInfo carries BitTorrent metadata.
type BTInfo struct {
	AnnounceList [][]string `json:"announceList"`
	Comment      string     `json:"comment"`
	CreationDate int64      `json:"creationDate"`
	Mode         string     `json:"mode"`
	Info         struct {
		Name string `json:"name"`
	} `json:"info"`
}

// Peer is a BitTorrent peer entry.
type Peer struct {
	PeerID        string `json:"peerId"`
	IP            string `json:"ip"`
	Port          Size   `json:"port"`
	Bitfield      string `json:"bitfield"`
	AmChoking     Bool   `json:"amChoking"`
	PeerChoking   Bool   `json:"peerChoking"`
	DownloadSpeed Size   `json:"downloadSpeed"`
	UploadSpeed   Size   `json:"uploadSpeed"`
	Seeder        Bool   `json:"seeder"`
}

// ServerGroup is the per-file server list from aria2.getServers.
type ServerGroup struct {
	Index   Size     `json:"index"`
	Servers []Server `json:"servers"`
}

// Server is one connected HTTP/FTP mirror.
type Server struct {
	URI           string `json:"uri"`
	CurrentURI    string `json:"currentUri"`
	DownloadSpeed Size   `json:"downloadSpeed"`
}

// GlobalStat is the result of aria2.getGlobalStat.
type GlobalStat struct {
	DownloadSpeed   Size `json:"downloadSpeed"`
	UploadSpeed     Size `json:"uploadSpeed"`
	NumActive       Size `json:"numActive"`
	NumWaiting      Size `json:"numWaiting"`
	NumStopped      Size `json:"numStopped"`
	NumStoppedTotal Size `json:"numStoppedTotal"`
}

// VersionInfo is the result of aria2.getVersion.
type VersionInfo struct {
	Version         string   `json:"version"`
	EnabledFeatures []string `json:"enabledFeatures"`
}

// SessionInfo is the result of aria2.getSessionInfo.
type SessionInfo struct {
	SessionID string `json:"sessionId"`
}

// Event names sent by aria2 as JSON-RPC notifications.
const (
	EventStart      = "aria2.onDownloadStart"
	EventPause      = "aria2.onDownloadPause"
	EventStop       = "aria2.onDownloadStop"
	EventComplete   = "aria2.onDownloadComplete"
	EventError      = "aria2.onDownloadError"
	EventBTComplete = "aria2.onBtDownloadComplete"
)

// Notification is an asynchronous event pushed by aria2.
type Notification struct {
	Method string
	GID    string
}

// gidParam is the notification params payload.
type gidParam struct {
	GID string `json:"gid"`
}

// parseNotificationGIDs extracts gids from a notification params array.
func parseNotificationGIDs(raw json.RawMessage) []string {
	var params []gidParam
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil
	}
	gids := make([]string, 0, len(params))
	for _, p := range params {
		gids = append(gids, p.GID)
	}
	return gids
}

// Common option keys, exported for convenience and typo safety.
const (
	OptDir                     = "dir"
	OptOut                     = "out"
	OptSplit                   = "split"
	OptMaxConnectionPerServer  = "max-connection-per-server"
	OptMaxDownloadLimit        = "max-download-limit"
	OptMaxUploadLimit          = "max-upload-limit"
	OptMaxOverallDownloadLimit = "max-overall-download-limit"
	OptMaxOverallUploadLimit   = "max-overall-upload-limit"
	OptMaxConcurrentDownloads  = "max-concurrent-downloads"
	OptMinSplitSize            = "min-split-size"
	OptPause                   = "pause"
	OptSelectFile              = "select-file"
	OptSeedRatio               = "seed-ratio"
	OptSeedTime                = "seed-time"
	OptBTMaxPeers              = "bt-max-peers"
	OptUserAgent               = "user-agent"
	OptReferer                 = "referer"
	OptHeader                  = "header"
	OptChecksum                = "checksum"
	OptContinue                = "continue"
	OptAllProxy                = "all-proxy"
)
