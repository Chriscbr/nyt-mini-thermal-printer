package main

import (
	"bufio"
	"flag"
	"fmt"
	"image/png"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "nyt-mini-thermal-printer:", err)
		os.Exit(1)
	}
}

func run() error {
	date := flag.String("date", "", "puzzle date as YYYY-MM-DD (default: today in US Eastern)")
	width := flag.Int("width", 384, "output width in pixels")
	out := flag.String("out", "", "output PNG path, or - for stdout (default: nyt-mini-YYYY-MM-DD.png)")
	answers := flag.Bool("answers", false, "append the answer key")
	cookie := flag.String("cookie", os.Getenv("NYT_S"), "NYT-S cookie value, required for dates other than today (env NYT_S)")
	flag.Parse()

	day, err := resolveDate(*date)
	if err != nil {
		return err
	}
	if *width < 64 {
		return fmt.Errorf("width must be at least 64 pixels")
	}

	puzzle, err := Fetch(day, *cookie)
	if err != nil {
		return err
	}

	img := Render(puzzle, Options{Width: *width, Answers: *answers})

	path := *out
	if path == "" {
		path = "nyt-mini-" + puzzle.Date.Format("2006-01-02") + ".png"
	}

	f := os.Stdout
	if path != "-" {
		if f, err = os.Create(path); err != nil {
			return err
		}
		defer f.Close()
	}
	w := bufio.NewWriter(f)
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(w, img); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if path != "-" {
		b := img.Bounds()
		fmt.Fprintf(os.Stderr, "wrote %s (%dx%d)\n", path, b.Dx(), b.Dy())
	}
	return nil
}

func resolveDate(s string) (time.Time, error) {
	if s != "" {
		return time.Parse("2006-01-02", s)
	}
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.Now(), nil
	}
	return time.Now().In(loc), nil
}
