package tray

import (
	"bytes"
	"encoding/binary"
)

const iconSize = 32

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

// Icon draws a filled disc of the given colour with a dark rim, as a
// 32×32 32-bit ICO file (the format the notification area takes).
func Icon(c Color) []byte {
	r, g, b := rgb(c)
	pixels := make([]byte, 0, iconSize*iconSize*4)
	center := float64(iconSize-1) / 2
	for row := iconSize - 1; row >= 0; row-- { // BMP rows go bottom-up
		for col := 0; col < iconSize; col++ {
			dx, dy := float64(col)-center, float64(row)-center
			d2 := dx*dx + dy*dy
			switch {
			case d2 > 15*15:
				pixels = append(pixels, 0, 0, 0, 0)
			case d2 > 13*13:
				pixels = append(pixels, b/2, g/2, r/2, 255)
			default:
				pixels = append(pixels, b, g, r, 255)
			}
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
