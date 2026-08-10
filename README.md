# nyt-mini-thermal-printer

Renders the day's NYT Mini crossword as a 1-bit PNG that has been optimized for a thermal printer. It defaults to rendering to 384px wide by default for the "Cat Printer" which takes 58mm paper, but the output width can be customized for printers with higher DPIs.

## Build

```
go build -o nyt-mini .
```

## Usage

```
./nyt-mini                      # today's mini -> nyt-mini-YYYY-MM-DD.png
./nyt-mini -answers             # append the answer key
./nyt-mini -width 576           # different printer width
./nyt-mini -out - | your-printer-tool
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-date` | today, US Eastern | puzzle date, `YYYY-MM-DD` |
| `-width` | `384` | output width in pixels; everything scales from it |
| `-out` | `nyt-mini-<date>.png` | output path, or `-` for stdout |
| `-answers` | off | append a filled answer grid |
| `-cookie` | `$NYT_S` | NYT-S cookie value |

## Development

### About the NYT endpoint

Puzzles come from `https://www.nytimes.com/svc/crosswords/v6/puzzle/mini/<date>.json`.

Today's mini is free, but NYT load-balances that endpoint across two backends
and only one serves anonymous callers, so a plain request returns 403 much of
the time. The tool retries up to 8 times with backoff, which reliably gets
through. Older dates are subscriber-only and always 403 without credentials —
for those, pass your `NYT-S` cookie:

```
NYT_S='...' ./nyt-mini -date 2026-08-01
```

Grab the value from DevTools → Application → Cookies on nytimes.com while
logged in (the base64-looking string, no `NYT-S=` prefix).

Fetched JSON is cached under `~/Library/Caches/nyt-mini-thermal-printer` (or
`$XDG_CACHE_HOME`), so re-rendering a day at a different width never re-hits the
network. Delete a file there to force a refetch.

### Output

A palette PNG with two colors, so it is genuinely 1 bit per pixel — feed it
straight to a printer tool without an extra dithering or threshold pass.

### Fonts

The Times sets its headlines in Cheltenham and its body text in Imperial, both
proprietary. This embeds PT Serif (SIL OFL, license in `fonts/`) instead — a
transitional serif sturdy enough to survive 1-bit thresholding at small sizes.
Type sizes live at the top of `Render` in `render.go` if you want them bigger
or smaller still.
