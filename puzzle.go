package mini

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// dates=1 trims the archive index the API otherwise bundles with every puzzle.
const puzzleURL = "https://api.thewordfinder.com/crossword-solver/nyt-mini/%s?dates=1"

// structureFrom is the earliest puzzle the API ships grid geometry for. Older
// entries still carry clues and answers, but nothing that says where the
// blocks go, so there is no grid to draw.
const structureFrom = "2025-09-02"

type Puzzle struct {
	Date        time.Time
	Constructor string
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
	PuzzleDate string  `json:"puzzle_date"`
	Name       *string `json:"name"`
	Simplified struct {
		AcrossClues []apiClue `json:"acrossClues"`
		DownClues   []apiClue `json:"downClues"`
	} `json:"simplified"`
	Structure struct {
		Rows  int `json:"rows"`
		Cols  int `json:"cols"`
		Cells []struct {
			Row      int    `json:"r"`
			Col      int    `json:"c"`
			Type     string `json:"type"`
			Label    int    `json:"label"`
			Solution string `json:"solution"`
		} `json:"cells"`
	} `json:"structure"`
}

type apiClue struct {
	Number int    `json:"number"`
	Clue   string `json:"clue"`
	Answer string `json:"answer"`
}

func Fetch(date time.Time) (*Puzzle, error) {
	day := date.Format("2006-01-02")
	raw, err := cached(day, func() ([]byte, error) {
		raw, err := get(fmt.Sprintf(puzzleURL, day))
		if err != nil {
			return nil, err
		}
		// A date the API has no puzzle for still answers 200, with today's
		// puzzle in the body. Catch that before it reaches the cache.
		var probe struct {
			PuzzleDate string `json:"puzzle_date"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			return nil, fmt.Errorf("decoding puzzle: %w", err)
		}
		if probe.PuzzleDate != day {
			return nil, fmt.Errorf("no mini puzzle published for %s (the API answered with %s instead)", day, probe.PuzzleDate)
		}
		return raw, nil
	})
	if err != nil {
		return nil, err
	}

	var api apiResponse
	if err := json.Unmarshal(raw, &api); err != nil {
		return nil, fmt.Errorf("decoding puzzle: %w", err)
	}
	if len(api.Structure.Cells) == 0 {
		return nil, fmt.Errorf("the API has no grid layout for %s; only puzzles from %s onward can be drawn", day, structureFrom)
	}

	p := &Puzzle{
		Date:   date,
		Width:  api.Structure.Cols,
		Height: api.Structure.Rows,
		Cells:  make([]Cell, api.Structure.Rows*api.Structure.Cols),
	}
	if t, err := time.Parse("2006-01-02", api.PuzzleDate); err == nil {
		p.Date = t
	}
	if api.Name != nil {
		p.Constructor = *api.Name
	}

	for _, c := range api.Structure.Cells {
		i := c.Row*p.Width + c.Col
		if i < 0 || i >= len(p.Cells) {
			continue
		}
		cell := Cell{Block: c.Type == "block", Answer: c.Solution}
		if c.Label > 0 {
			cell.Label = strconv.Itoa(c.Label)
		}
		p.Cells[i] = cell
	}

	p.Across = convert(api.Simplified.AcrossClues)
	p.Down = convert(api.Simplified.DownClues)
	return p, nil
}

func (p *Puzzle) At(row, col int) Cell {
	i := row*p.Width + col
	if i < 0 || i >= len(p.Cells) {
		return Cell{Block: true}
	}
	return p.Cells[i]
}

func convert(in []apiClue) []Clue {
	out := make([]Clue, 0, len(in))
	for _, c := range in {
		out = append(out, Clue{Label: strconv.Itoa(c.Number), Text: c.Clue})
	}
	return out
}

// cached memoizes a day's JSON on disk. A published puzzle never changes, so
// re-rendering it at another width costs nothing.
func cached(day string, fetch func() ([]byte, error)) ([]byte, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return fetch()
	}
	dir = filepath.Join(dir, "nyt-mini-thermal-printer")
	path := filepath.Join(dir, "wordfinder-"+day+".json")

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

func get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "nyt-mini-thermal-printer/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("thewordfinder.com returned %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}
