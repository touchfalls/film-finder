package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

type movie struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
}

func (m movie) year() string {
	if len(m.ReleaseDate) >= 4 {
		return m.ReleaseDate[:4]
	}
	return "—"
}

func main() {
	n := flag.Int("n", 10, "number of actors to show")
	flag.Usage = func() { fmt.Fprintln(os.Stderr, "usage: film-finder [-n 10] <title>"); flag.PrintDefaults() }
	flag.Parse()
	query := strings.Join(flag.Args(), " ")
	if query == "" || *n <= 0 {
		flag.Usage()
		os.Exit(2)
	}
	loadDotenv()
	if err := run(query, *n); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(query string, n int) error {
	key := os.Getenv("TMDB_API_KEY")
	if key == "" {
		return errors.New("set TMDB_API_KEY in env or .env, get one at https://www.themoviedb.org/settings/api")
	}
	movies, err := searchTMDB(query, key)
	if err != nil {
		return err
	}
	if len(movies) == 0 {
		return fmt.Errorf("no films found for %q", query)
	}
	m, err := choose(movies)
	if err != nil {
		return err
	}

	page, filmURL, err := get(fmt.Sprintf("https://letterboxd.com/tmdb/%d/", m.ID), nil, 24*time.Hour)
	var he httpError
	switch {
	case err == nil && !strings.Contains(filmURL, "/film/"), errors.As(err, &he) && he.Code == 404:
		// unknown TMDB ids answer 200 without redirecting to /film/
		return fmt.Errorf("%s (%s) is not on Letterboxd", m.Title, m.year())
	case errors.As(err, &he) && he.Code == 403:
		return errors.New("blocked by Letterboxd (Cloudflare), try again later")
	case err != nil:
		return fmt.Errorf("letterboxd: %w", err)
	}
	f, err := parseFilm(page)
	if err != nil {
		return err
	}
	render(f, m.year(), filmURL, n)
	return nil
}

func searchTMDB(query, key string) ([]movie, error) {
	q := url.Values{"query": {query}, "language": {"en-US"}}
	h := http.Header{}
	if strings.HasPrefix(key, "eyJ") { // v4 read access token (JWT)
		h.Set("Authorization", "Bearer "+key)
	} else {
		q.Set("api_key", key)
	}
	body, _, err := get("https://api.themoviedb.org/3/search/movie?"+q.Encode(), h, time.Hour)
	var he httpError
	switch {
	case errors.As(err, &he) && he.Code == 401:
		return nil, errors.New("invalid TMDB_API_KEY")
	case errors.As(err, &he) && he.Code == 429:
		return nil, errors.New("TMDB rate limit, retry later")
	case err != nil:
		return nil, fmt.Errorf("tmdb: %w", err)
	}
	var r struct{ Results []movie }
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("tmdb: %w", err)
	}
	return r.Results[:min(10, len(r.Results))], nil
}

func choose(ms []movie) (movie, error) {
	if len(ms) == 1 {
		return ms[0], nil
	}
	for i, m := range ms {
		fmt.Printf("%2d) %s (%s)\n", i+1, m.Title, m.year())
	}
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("choose 1-%d: ", len(ms))
		if !sc.Scan() {
			return movie{}, errors.New("no choice made")
		}
		if i, err := strconv.Atoi(strings.TrimSpace(sc.Text())); err == nil && i >= 1 && i <= len(ms) {
			return ms[i-1], nil
		}
	}
}

func or(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func render(f Film, year, filmURL string, n int) {
	width := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}

	rating, runtime := "—", "—"
	if f.RatingCount > 0 {
		rating = fmt.Sprintf("%.2f / 5 (%d ratings)", f.Rating, f.RatingCount)
	}
	if f.Runtime > 0 {
		runtime = fmt.Sprintf("%d min", int(f.Runtime.Minutes()))
	}
	rows := [][2]string{
		{"Title", f.Title + " (" + year + ")"},
		{"Director", or(strings.Join(f.Directors, ", "))},
		{"Rating", rating},
		{"Genres", or(strings.Join(f.Genres, ", "))},
		{"Runtime", runtime},
		{"URL", filmURL},
	}
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	for _, r := range rows {
		t.AppendRow(table.Row{r[0], r[1]})
	}
	if card := t.Render(); text.LongestLineLen(card) <= width {
		fmt.Println(card)
	} else { // card table only when it fits the terminal
		for _, r := range rows {
			fmt.Printf("%s: %s\n", r[0], r[1])
		}
	}
	if f.Description != "" {
		fmt.Println("\n" + text.WrapSoft(f.Description, width) + "\n")
	}

	if len(f.Cast) == 0 {
		fmt.Println("no cast listed")
		return
	}
	ct := table.NewWriter()
	ct.SetStyle(table.StyleLight)
	ct.AppendHeader(table.Row{"#", "Actor", "Role"})
	for i, a := range f.Cast[:min(n, len(f.Cast))] {
		ct.AppendRow(table.Row{i + 1, a.Name, or(a.Role)})
	}
	// borders + "#" column take ~16 cols; split the rest between name and role
	colMax := max(10, (width-16)/2)
	ct.SetColumnConfigs([]table.ColumnConfig{{Number: 2, WidthMax: colMax}, {Number: 3, WidthMax: colMax}})
	fmt.Println(ct.Render())
}
