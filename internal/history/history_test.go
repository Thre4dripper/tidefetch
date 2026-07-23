package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCategorize(t *testing.T) {
	cases := []struct {
		name    string
		torrent bool
		want    string
	}{
		{"movie.mkv", false, "Video"},
		{"song.FLAC", false, "Audio"},
		{"book.pdf", false, "Documents"},
		{"archive.tar.gz", false, "Archives"},
		{"tool.dmg", false, "Programs"},
		{"photo.jpeg", false, "Images"},
		{"anything.xyz", false, "Other"},
		{"ubuntu.iso", true, "Torrents"},
	}
	for _, c := range cases {
		if got := Categorize(c.name, c.torrent); got != c.want {
			t.Errorf("Categorize(%q, %v) = %q, want %q", c.name, c.torrent, got, c.want)
		}
	}
}

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	s, err := Open(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i, name := range []string{"a.zip", "b.zip", "c.zip", "d.zip"} {
		s.Upsert(Entry{GID: name, Name: name, Status: "complete",
			Category: Categorize(name, false), Finished: now.Add(time.Duration(i) * time.Second)})
	}
	if got := len(s.All()); got != 3 {
		t.Fatalf("limit not applied: %d", got)
	}
	// newest first
	if s.All()[0].GID != "d.zip" {
		t.Fatalf("order: %v", s.All()[0].GID)
	}
	// upsert same gid keeps count
	s.Upsert(Entry{GID: "d.zip", Name: "d.zip", Status: "error", Finished: now})
	if got := len(s.All()); got != 3 {
		t.Fatalf("upsert duplicated: %d", got)
	}
	if !s.Has("d.zip", "error") {
		t.Fatal("Has after update")
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(s2.All()); got != 3 {
		t.Fatalf("reload: %d", got)
	}
	s2.Delete("d.zip")
	if s2.Has("d.zip", "error") {
		t.Fatal("delete failed")
	}
	s2.Clear()
	if len(s2.All()) != 0 {
		t.Fatal("clear failed")
	}
}
