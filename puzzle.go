package mini

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const puzzleURL = "https://www.nytimes.com/svc/crosswords/v6/puzzle/mini/%s.json"

type Puzzle struct {
	Date        time.Time
	Constructor string
	Copyright   string
	Width       int
	Height      int
	Cells       []Cell
	Across      []Clue
	Down        []Clue
}

type Cell struct {
	Block  bool
	Label  string
	Answer string
}

type Clue struct {
	Label string
	Text  string
}

type apiResponse struct {
	Body []struct {
		Cells []struct {
			Answer string `json:"answer"`
			Label  string `json:"label"`
			Type   int    `json:"type"`
		} `json:"cells"`
		ClueLists []struct {
			Name  string `json:"name"`
			Clues []int  `json:"clues"`
		} `json:"clueLists"`
		Clues []struct {
			Direction string `json:"direction"`
			Label     string `json:"label"`
			Text      []struct {
				Plain string `json:"plain"`
			} `json:"text"`
		} `json:"clues"`
		Dimensions struct {
			Height int `json:"height"`
			Width  int `json:"width"`
		} `json:"dimensions"`
	} `json:"body"`
	Constructors    []string `json:"constructors"`
	Copyright       string   `json:"copyright"`
	PublicationDate string   `json:"publicationDate"`
}

// NYT load-balances this endpoint across two backends and only one of them
// serves anonymous callers, so an unauthenticated 403 is usually transient.
const maxAttempts = 8

func Fetch(date time.Time, cookie string) (*Puzzle, error) {
	day := date.Format("2006-01-02")
	raw, err := cached(day, func() ([]byte, error) {
		return get(fmt.Sprintf(puzzleURL, day), cookie, day)
	})
	if err != nil {
		return nil, err
	}

	var api apiResponse
	if err := json.Unmarshal(raw, &api); err != nil {
		return nil, fmt.Errorf("decoding puzzle: %w", err)
	}
	if len(api.Body) == 0 {
		return nil, fmt.Errorf("puzzle for %s has no body", day)
	}
	body := api.Body[0]

	p := &Puzzle{
		Date:      date,
		Copyright: api.Copyright,
		Width:     body.Dimensions.Width,
		Height:    body.Dimensions.Height,
	}
	if len(api.Constructors) > 0 {
		p.Constructor = api.Constructors[0]
		for _, c := range api.Constructors[1:] {
			p.Constructor += ", " + c
		}
	}
	if t, err := time.Parse("2006-01-02", api.PublicationDate); err == nil {
		p.Date = t
	}

	for _, c := range body.Cells {
		p.Cells = append(p.Cells, Cell{Block: c.Type == 0, Label: c.Label, Answer: c.Answer})
	}

	for _, list := range body.ClueLists {
		var out []Clue
		for _, i := range list.Clues {
			if i < 0 || i >= len(body.Clues) {
				continue
			}
			c := body.Clues[i]
			text := ""
			if len(c.Text) > 0 {
				text = c.Text[0].Plain
			}
			out = append(out, Clue{Label: c.Label, Text: text})
		}
		switch list.Name {
		case "Across":
			p.Across = out
		case "Down":
			p.Down = out
		}
	}
	return p, nil
}

// cached memoizes a day's JSON on disk. A published puzzle never changes, so
// re-rendering it at another width costs nothing and spares the flaky endpoint.
func cached(day string, fetch func() ([]byte, error)) ([]byte, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return fetch()
	}
	dir = filepath.Join(dir, "nyt-mini-thermal-printer")
	path := filepath.Join(dir, "mini-"+day+".json")

	if raw, err := os.ReadFile(path); err == nil {
		return raw, nil
	}
	raw, err := fetch()
	if err != nil {
		return nil, err
	}
	if os.MkdirAll(dir, 0o755) == nil {
		os.WriteFile(path, raw, 0o644)
	}
	return raw, nil
}

func get(url, cookie, day string) ([]byte, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	for attempt := 1; ; attempt++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "nyt-mini-thermal-printer/1.0")
		if cookie != "" {
			req.Header.Set("Cookie", "NYT-S="+cookie)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			return body, readErr
		case resp.StatusCode == http.StatusNotFound:
			return nil, fmt.Errorf("no mini puzzle published for %s", day)
		case resp.StatusCode == http.StatusForbidden && attempt >= maxAttempts:
			return nil, fmt.Errorf("nytimes.com refused %s after %d attempts; today's mini is free but older ones are not, so pass -cookie or set NYT_S to your NYT-S cookie value", day, attempt)
		case resp.StatusCode == http.StatusForbidden:
		default:
			return nil, fmt.Errorf("nytimes.com returned %s for %s", resp.Status, day)
		}
		time.Sleep(time.Duration(attempt) * 400 * time.Millisecond)
	}
}

func (p *Puzzle) At(row, col int) Cell {
	i := row*p.Width + col
	if i < 0 || i >= len(p.Cells) {
		return Cell{Block: true}
	}
	return p.Cells[i]
}
