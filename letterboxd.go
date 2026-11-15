package main

import (
	"encoding/json"
	"errors"
	"html"
	"regexp"
	"strings"
	"time"
)

type Actor struct{ Name, Role string }

type Film struct {
	Title, Description string
	Directors, Genres  []string
	Runtime            time.Duration
	Rating             float64
	RatingCount        int
	Cast               []Actor
}

var (
	ldRe = regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`)
	// shortcut: regex по вёрстке, сломается при редизайне — тест на testdata это поймает
	castRe = regexp.MustCompile(`<a (?:title="([^"]*)" )?href="/actor/[^"]+/"[^>]*>([^<]+)</a>`)
)

func parseFilm(page []byte) (Film, error) {
	m := ldRe.FindSubmatch(page)
	if m == nil {
		return Film{}, errors.New("unexpected page format: no JSON-LD")
	}
	raw := strings.NewReplacer("/* <![CDATA[ */", "", "/* ]]> */", "").Replace(string(m[1]))
	var ld struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Duration    string `json:"duration"`
		Genre       []string
		Director    []struct{ Name string }
		Rating      *struct {
			RatingValue float64
			RatingCount int
		} `json:"aggregateRating"`
	}
	if err := json.Unmarshal([]byte(raw), &ld); err != nil {
		return Film{}, errors.New("unexpected page format: " + err.Error())
	}
	f := Film{Title: html.UnescapeString(ld.Name), Description: html.UnescapeString(ld.Description), Genres: ld.Genre}
	for _, d := range ld.Director {
		f.Directors = append(f.Directors, html.UnescapeString(d.Name))
	}
	if ld.Rating != nil {
		f.Rating, f.RatingCount = ld.Rating.RatingValue, ld.Rating.RatingCount
	}
	f.Runtime, _ = time.ParseDuration(strings.ToLower(strings.TrimPrefix(ld.Duration, "PT"))) // missing → 0
	for _, c := range castRe.FindAllSubmatch(page, -1) {
		f.Cast = append(f.Cast, Actor{Name: html.UnescapeString(string(c[2])), Role: html.UnescapeString(string(c[1]))})
	}
	return f, nil
}
