package tray

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidp57/process-monitor/internal/i18n"
)

var pngMagic = []byte("\x89PNG\r\n\x1a\n")

// LoadImage reads a .png or .ico file and scales it to the icon size. From
// an .ico it takes the largest image, stored as PNG or as a 32-bit BMP.
func LoadImage(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var img image.Image
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		img, err = png.Decode(bytes.NewReader(data))
	case ".ico":
		img, err = decodeICO(data)
	default:
		return nil, i18n.Errorf("icon.bad_type", path)
	}
	if err != nil {
		return nil, fmt.Errorf("%s : %w", path, err)
	}
	return scale(img, iconSize), nil
}

func decodeICO(data []byte) (image.Image, error) {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[0:]) != 0 || binary.LittleEndian.Uint16(data[2:]) != 1 {
		return nil, errors.New(i18n.T("icon.not_ico"))
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	if count == 0 || len(data) < 6+16*count {
		return nil, errors.New(i18n.T("icon.truncated"))
	}
	best, bestSize := -1, 0
	for i := 0; i < count; i++ {
		size := int(data[6+16*i])
		if size == 0 {
			size = 256
		}
		if size > bestSize {
			best, bestSize = i, size
		}
	}
	entry := data[6+16*best:]
	length := int(binary.LittleEndian.Uint32(entry[8:]))
	offset := int(binary.LittleEndian.Uint32(entry[12:]))
	if offset < 0 || length < 0 || offset+length > len(data) {
		return nil, errors.New(i18n.T("icon.outside"))
	}
	img := data[offset : offset+length]
	if bytes.HasPrefix(img, pngMagic) {
		return png.Decode(bytes.NewReader(img))
	}
	return decodeDIB(img)
}

// decodeDIB reads the 32-bit BMP stored in an ICO entry (rows bottom-up,
// height doubled for the AND mask, which the alpha channel makes useless).
func decodeDIB(b []byte) (image.Image, error) {
	if len(b) < 40 || binary.LittleEndian.Uint32(b) < 40 {
		return nil, errors.New(i18n.T("icon.unreadable"))
	}
	w := int(int32(binary.LittleEndian.Uint32(b[4:])))
	h := int(int32(binary.LittleEndian.Uint32(b[8:]))) / 2
	bpp := binary.LittleEndian.Uint16(b[14:])
	if bpp != 32 {
		return nil, i18n.Errorf("icon.depth", bpp)
	}
	px := b[binary.LittleEndian.Uint32(b):]
	if w <= 0 || h <= 0 || len(px) < w*h*4 {
		return nil, errors.New(i18n.T("icon.image_truncated"))
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	opaque := true // old icons leave the alpha byte at 0 everywhere
	for i := 3; i < w*h*4; i += 4 {
		if px[i] != 0 {
			opaque = false
			break
		}
	}
	for y := 0; y < h; y++ {
		row := px[(h-1-y)*w*4:]
		for x := 0; x < w; x++ {
			p := row[x*4:]
			a := p[3]
			if opaque {
				a = 255
			}
			img.Set(x, y, color.NRGBA{p[2], p[1], p[0], a})
		}
	}
	return img, nil
}

// scale resizes img to size×size: each target pixel averages the source
// pixels it covers (premultiplied, so transparent edges do not darken).
func scale(img image.Image, size int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, size, size))
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	for y := 0; y < size; y++ {
		y0, y1 := b.Min.Y+y*sh/size, b.Min.Y+max((y+1)*sh/size, y*sh/size+1)
		for x := 0; x < size; x++ {
			x0, x1 := b.Min.X+x*sw/size, b.Min.X+max((x+1)*sw/size, x*sw/size+1)
			var r, g, bl, a, n uint32
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := img.At(sx, sy).RGBA()
					r, g, bl, a, n = r+cr, g+cg, bl+cb, a+ca, n+1
				}
			}
			out.SetRGBA(x, y, color.RGBA{uint8(r / n >> 8), uint8(g / n >> 8), uint8(bl / n >> 8), uint8(a / n >> 8)})
		}
	}
	return out
}
