// Package history persists a searchable record of finished downloads,
// with IDM-style automatic categories.
package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Entry is one recorded download.
type Entry struct {
	GID      string    `json:"gid"`
	Name     string    `json:"name"`
	URL      string    `json:"url,omitempty"`
	Dir      string    `json:"dir"`
	Size     int64     `json:"size"`
	Status   string    `json:"status"` // complete | error | removed
	Category string    `json:"category"`
	Added    time.Time `json:"added"`
	Finished time.Time `json:"finished"`
}

// Categories in display order.
var Categories = []string{"Video", "Audio", "Documents", "Archives", "Programs", "Images", "Torrents", "Other"}

var extCategory = map[string]string{
	// video
	".mp4": "Video", ".mkv": "Video", ".avi": "Video", ".mov": "Video", ".webm": "Video",
	".flv": "Video", ".wmv": "Video", ".m4v": "Video", ".ts": "Video", ".mpg": "Video", ".mpeg": "Video",
	// audio
	".mp3": "Audio", ".flac": "Audio", ".wav": "Audio", ".aac": "Audio", ".ogg": "Audio",
	".m4a": "Audio", ".opus": "Audio", ".wma": "Audio", ".aiff": "Audio",
	// documents
	".pdf": "Documents", ".doc": "Documents", ".docx": "Documents", ".xls": "Documents",
	".xlsx": "Documents", ".ppt": "Documents", ".pptx": "Documents", ".txt": "Documents",
	".epub": "Documents", ".mobi": "Documents", ".csv": "Documents", ".md": "Documents",
	// archives
	".zip": "Archives", ".rar": "Archives", ".7z": "Archives", ".tar": "Archives",
	".gz": "Archives", ".bz2": "Archives", ".xz": "Archives", ".zst": "Archives", ".tgz": "Archives",
	// programs
	".exe": "Programs", ".msi": "Programs", ".dmg": "Programs", ".pkg": "Programs",
	".deb": "Programs", ".rpm": "Programs", ".apk": "Programs", ".appimage": "Programs",
	".iso": "Programs", ".img": "Programs", ".bin": "Programs",
	// images
	".jpg": "Images", ".jpeg": "Images", ".png": "Images", ".gif": "Images",
	".webp": "Images", ".svg": "Images", ".bmp": "Images", ".tiff": "Images", ".heic": "Images",
	// torrents
	".torrent": "Torrents",
}

// Categorize maps a filename to an IDM-style category.
func Categorize(name string, isTorrent bool) string {
	if isTorrent {
		return "Torrents"
	}
	ext := strings.ToLower(filepath.Ext(name))
	if c, ok := extCategory[ext]; ok {
		return c
	}
	return "Other"
}

// Store is a JSON-file backed history collection. Safe for concurrent use.
type Store struct {
	mu      sync.Mutex
	path    string
	limit   int
	entries []Entry // newest first
	index   map[string]int
}

// Open loads (or initializes) the history store at path.
func Open(path string, limit int) (*Store, error) {
	s := &Store{path: path, limit: limit, index: map[string]int{}}
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &s.entries)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	sort.SliceStable(s.entries, func(i, j int) bool { return s.entries[i].Finished.After(s.entries[j].Finished) })
	s.reindex()
	return s, nil
}

func (s *Store) reindex() {
	s.index = make(map[string]int, len(s.entries))
	for i, e := range s.entries {
		s.index[e.GID] = i
	}
}

// Upsert inserts or updates an entry keyed by GID.
func (s *Store) Upsert(e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i, ok := s.index[e.GID]; ok {
		if e.Added.IsZero() {
			e.Added = s.entries[i].Added
		}
		s.entries[i] = e
		return
	}
	s.entries = append([]Entry{e}, s.entries...)
	if s.limit > 0 && len(s.entries) > s.limit {
		s.entries = s.entries[:s.limit]
	}
	s.reindex()
}

// Has reports whether a GID is already recorded with the same status.
func (s *Store) Has(gid, status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.index[gid]
	return ok && s.entries[i].Status == status
}

// Delete removes an entry by GID.
func (s *Store) Delete(gid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.index[gid]
	if !ok {
		return
	}
	s.entries = append(s.entries[:i], s.entries[i+1:]...)
	s.reindex()
}

// Clear removes every entry.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
	s.reindex()
}

// All returns a copy of all entries, newest first.
func (s *Store) All() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Save writes the store to disk.
func (s *Store) Save() error {
	s.mu.Lock()
	data, err := json.MarshalIndent(s.entries, "", " ")
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}
