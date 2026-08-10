package mini

import (
	_ "embed"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// The Times' own faces (Cheltenham, Imperial, Franklin) are proprietary, so the
// page is set in PT Serif: a sturdy transitional serif that keeps its shape
// after 1-bit thresholding at small sizes.
var (
	//go:embed fonts/PTSerif-Regular.ttf
	serifTTF []byte
	//go:embed fonts/PTSerif-Bold.ttf
	serifBoldTTF []byte
)

var (
	serif     = mustParse(serifTTF)
	serifBold = mustParse(serifBoldTTF)
)

func mustParse(ttf []byte) *sfnt.Font {
	f, err := opentype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	return f
}

func face(f *sfnt.Font, size float64) font.Face {
	fc, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic(err)
	}
	return fc
}
