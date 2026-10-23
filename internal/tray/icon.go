package tray

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
)

const (
	iconSize  = 32
	dotRadius = 6 // the state dot drawn over a custom image
)

func rgb(c Color) (r, g, b byte) {
	switch c {
	case Green:
		return 0x2e, 0xa0, 0x43
	case Orange:
		return 0xe0, 0x8a, 0x00
	case Grey:
		return 0x8a, 0x8a, 0x8a
	default:
		return 0xd0, 0x2c, 0x2c
	}
}

// disc paints a filled disc of colour c with a darker rim onto img,
// centred on (cx, cy): inside radius r the colour, up to r+2 the rim.
func disc(img *image.RGBA, cx, cy, r float64, c Color) {
	red, green, blue := rgb(c)
	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			switch d2 := dx*dx + dy*dy; {
			case d2 <= r*r:
				img.SetRGBA(x, y, color.RGBA{red, green, blue, 255})
			case d2 <= (r+2)*(r+2):
				img.SetRGBA(x, y, color.RGBA{red / 2, green / 2, blue / 2, 255})
			}
		}
	}
}

// Icon draws the default icon: a disc of the state's colour, as a 32×32
// 32-bit ICO file (the format the notification area takes).
func Icon(c Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	center := float64(iconSize-1) / 2
	disc(img, center, center, 13, c)
	return encodeICO(img)
}

// Badged draws a custom image with a small dot of the state's colour in its
// bottom-right corner. base must be iconSize×iconSize (see LoadImage).
func Badged(base image.Image, c Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			img.Set(x, y, base.At(base.Bounds().Min.X+x, base.Bounds().Min.Y+y))
		}
	}
	at := float64(iconSize - dotRadius - 1)
	disc(img, at, at, dotRadius, c)
	return encodeICO(img)
}

// encodeICO writes img as a one-image ICO holding a 32-bit BMP.
func encodeICO(img *image.RGBA) []byte {
	pixels := make([]byte, 0, iconSize*iconSize*4)
	for y := iconSize - 1; y >= 0; y-- { // BMP rows go bottom-up
		for x := 0; x < iconSize; x++ {
			c := img.RGBAAt(x, y)
			pixels = append(pixels, c.B, c.G, c.R, c.A)
		}
	}
	mask := make([]byte, iconSize*iconSize/8) // AND mask unused: the alpha channel decides

	var bmp bytes.Buffer
	le := func(v any) { _ = binary.Write(&bmp, binary.LittleEndian, v) }
	le(uint32(40))          // BITMAPINFOHEADER size
	le(int32(iconSize))     // width
	le(int32(iconSize * 2)) // height: colour + mask, as ICO wants
	le(uint16(1))           // planes
	le(uint16(32))          // bits per pixel
	le(uint32(0))           // BI_RGB
	le(uint32(len(pixels) + len(mask)))
	le([4]int32{}) // resolution, palette: unused
	bmp.Write(pixels)
	bmp.Write(mask)

	var ico bytes.Buffer
	w := func(v any) { _ = binary.Write(&ico, binary.LittleEndian, v) }
	w(uint16(0)) // reserved
	w(uint16(1)) // type: icon
	w(uint16(1)) // one image
	w([4]byte{iconSize, iconSize, 0, 0})
	w(uint16(1))         // planes
	w(uint16(32))        // bits per pixel
	w(uint32(bmp.Len())) // image size
	w(uint32(6 + 16))    // image offset: after this header and its one entry
	ico.Write(bmp.Bytes())
	return ico.Bytes()
}
