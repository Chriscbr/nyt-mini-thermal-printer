package mini

import (
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// maxCanvas caps the working canvas; the image is cropped to its real height.
const maxCanvas = 8000

type Options struct {
	Width   int
	Answers bool
}

type canvas struct {
	img *image.Gray
	w   int
	y   int
	pad int
	s   float64 // scale relative to the 384px reference width
}

func Render(p *Puzzle, opt Options) image.Image {
	s := float64(opt.Width) / 384.0
	c := &canvas{
		img: image.NewGray(image.Rect(0, 0, opt.Width, maxCanvas)),
		w:   opt.Width,
		pad: round(10 * s),
		s:   s,
	}
	for i := range c.img.Pix {
		c.img.Pix[i] = 0xff
	}

	title := face(serifBold, 46*s)
	meta := face(serif, 20*s)
	section := face(serifBold, 28*s)
	clue := face(serif, 23*s)
	clueBold := face(serifBold, 23*s)

	c.y = c.pad
	c.centeredTracked(title, "NYT MINI", round(4*s))
	c.gap(3 * s)
	c.rule(max(round(1*s), 1))
	c.gap(5 * s)
	c.centered(meta, p.Date.Format("Monday, January 2, 2006"))
	if p.Constructor != "" {
		c.centered(meta, "By "+p.Constructor)
	}

	c.gap(10 * s)
	c.grid(p, c.w-2*c.pad, false)
	c.gap(14 * s)

	c.clueSection(section, clueBold, clue, "ACROSS", p.Across)
	c.gap(10 * s)
	c.clueSection(section, clueBold, clue, "DOWN", p.Down)

	if opt.Answers {
		c.gap(16 * s)
		c.rule(round(2 * s))
		c.gap(10 * s)
		c.centered(section, "ANSWERS")
		c.gap(8 * s)
		c.grid(p, round(float64(c.w-2*c.pad)*0.62), true)
	}

	c.y += c.pad
	return threshold(c.img.SubImage(image.Rect(0, 0, c.w, min(c.y, maxCanvas))).(*image.Gray))
}

func (c *canvas) gap(px float64) { c.y += round(px) }

// draw renders text with its left edge at x sitting on the given baseline.
func (c *canvas) draw(f font.Face, x, baseline int, text string) {
	d := &font.Drawer{
		Dst:  c.img,
		Src:  image.NewUniform(color.Gray{Y: 0}),
		Face: f,
		Dot:  fixed.P(x, baseline),
	}
	d.DrawString(text)
}

// drawAt renders text into a box whose top edge is y, without moving the cursor.
func (c *canvas) drawAt(f font.Face, x, y int, text string) {
	c.draw(f, x, y+f.Metrics().Ascent.Ceil(), text)
}

// line draws a single string with its left edge at x and advances the cursor.
func (c *canvas) line(f font.Face, x int, text string) {
	c.drawAt(f, x, c.y, text)
	c.y += f.Metrics().Height.Ceil()
}

func (c *canvas) centered(f font.Face, text string) {
	w := font.MeasureString(f, text).Ceil()
	c.line(f, max((c.w-w)/2, c.pad), text)
}

// centeredTracked centers text with extra space between letters. Drawing rune
// by rune drops kerning, which is what you want for spaced-out display caps.
func (c *canvas) centeredTracked(f font.Face, text string, tracking int) {
	width, n := 0, 0
	for _, r := range text {
		adv, ok := f.GlyphAdvance(r)
		if !ok {
			continue
		}
		width += adv.Ceil()
		n++
	}
	if n > 1 {
		width += tracking * (n - 1)
	}

	x := max((c.w-width)/2, c.pad)
	baseline := c.y + f.Metrics().Ascent.Ceil()
	for _, r := range text {
		adv, ok := f.GlyphAdvance(r)
		if !ok {
			continue
		}
		c.draw(f, x, baseline, string(r))
		x += adv.Ceil() + tracking
	}
	c.y += f.Metrics().Height.Ceil()
}

func (c *canvas) rule(thickness int) {
	c.fill(c.pad, c.y, c.w-2*c.pad, thickness)
	c.y += thickness
}

func (c *canvas) fill(x, y, w, h int) {
	r := image.Rect(x, y, x+w, y+h).Intersect(c.img.Bounds())
	for yy := r.Min.Y; yy < r.Max.Y; yy++ {
		row := c.img.Pix[yy*c.img.Stride+r.Min.X : yy*c.img.Stride+r.Max.X]
		for i := range row {
			row[i] = 0
		}
	}
}

func (c *canvas) clueSection(heading, num, body font.Face, name string, clues []Clue) {
	c.line(heading, c.pad, name)
	c.gap(3 * c.s)

	indent := 0
	for _, cl := range clues {
		if w := font.MeasureString(num, cl.Label+".").Ceil(); w > indent {
			indent = w
		}
	}
	indent += round(6 * c.s)

	x := c.pad + indent
	avail := c.w - c.pad - x
	for _, cl := range clues {
		top := c.y
		for i, ln := range wrap(body, smartQuotes(cl.Text), avail) {
			if i == 0 {
				c.drawAt(num, c.pad, top, cl.Label+".")
			}
			c.line(body, x, ln)
		}
		c.gap(3 * c.s)
	}
}

func (c *canvas) grid(p *Puzzle, maxWidth int, answers bool) {
	lw := max(round(3*c.s), 1)
	cell := (maxWidth - lw) / p.Width
	total := cell*p.Width + lw
	ox := (c.w - total) / 2
	oy := c.y

	labels := face(serif, float64(cell)*0.31)
	letters := face(serifBold, float64(cell)*0.72)

	for row := 0; row < p.Height; row++ {
		for col := 0; col < p.Width; col++ {
			cl := p.At(row, col)
			x, y := ox+col*cell, oy+row*cell
			if cl.Block {
				c.fill(x, y, cell+lw, cell+lw)
				continue
			}
			if answers {
				b, _ := font.BoundString(letters, cl.Answer)
				bw, bh := (b.Max.X - b.Min.X).Ceil(), (b.Max.Y - b.Min.Y).Ceil()
				c.draw(letters, x+(cell-bw)/2-b.Min.X.Floor(), y+(cell+bh)/2-b.Max.Y.Ceil(), cl.Answer)
			} else if cl.Label != "" {
				c.drawAt(labels, x+lw+round(3*c.s), y+lw+round(1*c.s), cl.Label)
			}
		}
	}

	for row := 0; row <= p.Height; row++ {
		c.fill(ox, oy+row*cell, total, lw)
	}
	for col := 0; col <= p.Width; col++ {
		c.fill(ox+col*cell, oy, lw, cell*p.Height+lw)
	}
	c.y = oy + cell*p.Height + lw
}

// smartQuotes upgrades the API's straight quotes to typographic ones. Double
// quotes alternate open/close; apostrophes always close, which is right for
// contractions and possessives and wrong only for rare leading single quotes.
func smartQuotes(text string) string {
	var b strings.Builder
	open := true
	for _, r := range text {
		switch r {
		case '"':
			if open {
				b.WriteRune('“')
			} else {
				b.WriteRune('”')
			}
			open = !open
		case '\'':
			b.WriteRune('’')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func wrap(f font.Face, text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if font.MeasureString(f, line+" "+w).Ceil() <= width {
			line += " " + w
		} else {
			lines = append(lines, line)
			line = w
		}
	}
	return append(lines, line)
}

// threshold converts the antialiased grayscale canvas to a 1-bit-per-pixel image.
func threshold(src *image.Gray) *image.Paletted {
	b := src.Bounds()
	dst := image.NewPaletted(b, color.Palette{color.Gray{Y: 0xff}, color.Gray{Y: 0}})
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if src.GrayAt(x, y).Y < 0xa0 {
				dst.SetColorIndex(x, y, 1)
			}
		}
	}
	return dst
}

func round(f float64) int { return int(f + 0.5) }
