package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type httpError struct{ Code int }

func (e httpError) Error() string { return fmt.Sprintf("HTTP %d", e.Code) }

// cacheEntry is a DNS-style record: valid until Expires, then refetched.
// URL itself is not stored: for v3 keys it contains the TMDB api_key.
type cacheEntry struct {
	FinalURL string
	Expires  time.Time
	Body     []byte
}

var (
	client   = &http.Client{Timeout: 15 * time.Second}
	cacheDir = func() string { d, _ := os.UserCacheDir(); return filepath.Join(d, "film-finder") }()
)

func cachePath(url string) string {
	h := sha256.Sum256([]byte(url))
	return filepath.Join(cacheDir, hex.EncodeToString(h[:])+".json")
}

func cacheGet(url string) (cacheEntry, bool) {
	var e cacheEntry
	b, err := os.ReadFile(cachePath(url))
	if err != nil || json.Unmarshal(b, &e) != nil || !time.Now().Before(e.Expires) {
		return e, false
	}
	return e, true
}

// cachePut is best effort: a failure just means the next run hits the network.
func cachePut(url string, e cacheEntry) {
	b, _ := json.Marshal(e)
	if os.MkdirAll(cacheDir, 0o755) != nil {
		return
	}
	tmp, err := os.CreateTemp(cacheDir, "tmp-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(b)
	if cerr := tmp.Close(); werr != nil || cerr != nil || os.Rename(tmp.Name(), cachePath(url)) != nil {
		os.Remove(tmp.Name())
	}
}

// get fetches url (following redirects), serving from cache while the TTL holds.
func get(url string, header http.Header, ttl time.Duration) ([]byte, string, error) {
	if e, ok := cacheGet(url); ok {
		return e.Body, e.FinalURL, nil
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (film-finder)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", httpError{resp.StatusCode}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	final := resp.Request.URL.String()
	cachePut(url, cacheEntry{FinalURL: final, Expires: time.Now().Add(ttl), Body: body})
	return body, final, nil
}
