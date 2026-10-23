package tray

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// square is a size×size image: blue on the left half, yellow on the right.
func square(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := color.RGBA{0, 0, 255, 255}
			if x >= size/2 {
				c = color.RGBA{255, 255, 0, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func pngBytes(t *testing.T, img image.Image) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// icoWith wraps image entries (raw PNG or BMP DIB data) into an ICO file.
func icoWith(sizes []int, data [][]byte) []byte {
	var b bytes.Buffer
	w := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }
	w(uint16(0))
	w(uint16(1))
	w(uint16(len(data)))
	offset := 6 + 16*len(data)
	for i, d := range data {
		s := byte(sizes[i])
		if sizes[i] >= 256 {
			s = 0
		}
		w([4]byte{s, s, 0, 0})
		w(uint16(1))
		w(uint16(32))
		w(uint32(len(d)))
		w(uint32(offset))
		offset += len(d)
	}
	for _, d := range data {
		b.Write(d)
	}
	return b.Bytes()
}

func write(t *testing.T, name string, data []byte) string {
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func checkHalves(t *testing.T, what string, img image.Image) {
	t.Helper()
	if img.Bounds().Dx() != iconSize || img.Bounds().Dy() != iconSize {
		t.Fatalf("%s : %v, attendu %d×%d", what, img.Bounds(), iconSize, iconSize)
	}
	left := color.RGBAModel.Convert(img.At(4, 4)).(color.RGBA)
	right := color.RGBAModel.Convert(img.At(iconSize-5, 4)).(color.RGBA)
	if left != (color.RGBA{0, 0, 255, 255}) || right != (color.RGBA{255, 255, 0, 255}) {
		t.Errorf("%s : gauche %v, droite %v", what, left, right)
	}
}

func TestLoadPNGScalesTo32(t *testing.T) {
	img, err := LoadImage(write(t, "a.png", pngBytes(t, square(256))))
	if err != nil {
		t.Fatal(err)
	}
	checkHalves(t, "png 256", img)
}

func TestLoadICOWithPNGEntryPicksTheLargest(t *testing.T) {
	ico := icoWith([]int{16, 256}, [][]byte{pngBytes(t, square(16)), pngBytes(t, square(256))})
	img, err := LoadImage(write(t, "a.ico", ico))
	if err != nil {
		t.Fatal(err)
	}
	checkHalves(t, "ico png", img)
}

func TestLoadICOWithBMPEntry(t *testing.T) {
	// Icon() writes a 32-bit BMP entry: read back our own disc.
	img, err := LoadImage(write(t, "disc.ico", Icon(Red)))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b := rgb(Red)
	if c := color.RGBAModel.Convert(img.At(iconSize/2, iconSize/2)).(color.RGBA); c != (color.RGBA{r, g, b, 255}) {
		t.Errorf("centre %v", c)
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Error("le coin devrait rester transparent")
	}
}

func TestLoadRejectsOtherFiles(t *testing.T) {
	for name, data := range map[string][]byte{"a.jpg": {1, 2, 3}, "bad.ico": {0, 0, 1, 0}, "bad.png": {1, 2}} {
		if _, err := LoadImage(write(t, name, data)); err == nil {
			t.Errorf("%s accepté", name)
		}
	}
}

// The state dot sits bottom-right; the rest of the image is untouched.
func TestBadgeKeepsTheImageAndShowsTheState(t *testing.T) {
	base := square(iconSize)
	for _, c := range []Color{Green, Orange, Grey, Red} {
		ico := Badged(base, c)
		img, err := LoadImage(write(t, "b.ico", ico))
		if err != nil {
			t.Fatal(err)
		}
		r, g, b := rgb(c)
		dot := color.RGBAModel.Convert(img.At(iconSize-dotRadius-1, iconSize-dotRadius-1)).(color.RGBA)
		if dot != (color.RGBA{r, g, b, 255}) {
			t.Errorf("%v : pastille %v", c, dot)
		}
		if got := color.RGBAModel.Convert(img.At(4, 4)).(color.RGBA); got != (color.RGBA{0, 0, 255, 255}) {
			t.Errorf("%v : l'image est modifiée hors de la pastille : %v", c, got)
		}
	}
}
