package main

import (
	"os"
	"testing"
	"time"
)

func TestParseFilm(t *testing.T) {
	page, err := os.ReadFile("testdata/the-matrix.html")
	if err != nil {
		t.Fatal(err)
	}
	f, err := parseFilm(page)
	if err != nil {
		t.Fatal(err)
	}
	if f.Title != "The Matrix" || f.Runtime != 136*time.Minute || f.Rating <= 0 || f.RatingCount <= 0 {
		t.Errorf("bad film: %+v", f)
	}
	if len(f.Directors) != 2 || len(f.Genres) == 0 || f.Description == "" {
		t.Errorf("missing fields: %+v", f)
	}
	if len(f.Cast) < 10 || f.Cast[0] != (Actor{"Keanu Reeves", "Neo"}) {
		t.Errorf("bad cast: %v", f.Cast)
	}
	if _, err := parseFilm([]byte("<html></html>")); err == nil {
		t.Error("expected error on page without JSON-LD")
	}
}

func TestCacheTTL(t *testing.T) {
	cacheDir = t.TempDir()
	cachePut("fresh", cacheEntry{FinalURL: "f", Expires: time.Now().Add(time.Hour), Body: []byte("x")})
	cachePut("stale", cacheEntry{Expires: time.Now().Add(-time.Second), Body: []byte("y")})
	if e, ok := cacheGet("fresh"); !ok || string(e.Body) != "x" || e.FinalURL != "f" {
		t.Errorf("fresh entry not served: %v %+v", ok, e)
	}
	if _, ok := cacheGet("stale"); ok {
		t.Error("expired entry served")
	}
	if _, ok := cacheGet("missing"); ok {
		t.Error("missing entry served")
	}
}

func TestParseEnvLine(t *testing.T) {
	for line, want := range map[string][2]string{
		"TMDB_API_KEY=abc":          {"TMDB_API_KEY", "abc"},
		`export TMDB_API_KEY="a b"`: {"TMDB_API_KEY", "a b"},
		" K = 'x=y' ":               {"K", "x=y"},
	} {
		if k, v, ok := parseEnvLine(line); !ok || k != want[0] || v != want[1] {
			t.Errorf("%q -> %q %q %v", line, k, v, ok)
		}
	}
	for _, line := range []string{"", "# c", "noequals", "=v"} {
		if _, _, ok := parseEnvLine(line); ok {
			t.Errorf("%q should be skipped", line)
		}
	}
}
