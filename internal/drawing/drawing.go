package drawing

import (
	"image"
	"image/color"
	_ "net/http/pprof"

	"github.com/drummonds/gophoto/internal/fbimage"

	_ "embed"
	_ "image/png"
)

// copyRGBAtoBGR565 is an inlined version of the hot pixel copying loop for the
// special case of copying from an *image.RGBA to an *fbimage.BGR565.
//
// This specialization brings down copying time to 137ms (from 1.8s!) on the
// Raspberry Pi 4.
func CopyRGBAtoBGR565(dst *fbimage.BGR565, src *image.RGBA) {
	bounds := dst.Bounds()
	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			var c color.NRGBA

			i := src.PixOffset(x, y)
			// Small cap improves performance, see https://golang.org/issue/27857
			s := src.Pix[i : i+4 : i+4]
			switch s[3] {
			case 0xff:
				c = color.NRGBA{s[0], s[1], s[2], 0xff}
			case 0:
				c = color.NRGBA{0, 0, 0, 0}
			default:
				r := uint32(s[0])
				r |= r << 8
				g := uint32(s[1])
				g |= g << 8
				b := uint32(s[2])
				b |= b << 8
				a := uint32(s[3])
				a |= a << 8

				// Since Color.RGBA returns an alpha-premultiplied color, we
				// should have r <= a && g <= a && b <= a.
				r = (r * 0xffff) / a
				g = (g * 0xffff) / a
				b = (b * 0xffff) / a
				c = color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
			}

			pix := dst.Pix[dst.PixOffset(x, y):]
			pix[0] = (c.B >> 3) | ((c.G >> 2) << 5)
			pix[1] = (c.G >> 5) | ((c.R >> 3) << 3)
		}
	}
}

// copyRGBAtoBGRA is an inlined version of the hot pixel copying loop for the
// special case of copying from an *image.RGBA to an *fbimage.BGRA.
//
// This specialization brings down copying time to 5ms (from 60-70ms) on an
// amd64 qemu VM with virtio VGA.
func CopyRGBAtoBGRA(dst *fbimage.BGRA, src *image.RGBA) {
	for i := 0; i < len(src.Pix); i += 4 {
		s := src.Pix[i : i+4 : i+4]
		d := dst.Pix[i : i+4 : i+4]
		d[0], d[1], d[2], d[3] = s[2], s[1], s[0], s[3]
	}
}

// Calculated linear scaling of an rectangle from its original size to
// a max width and max height of a desired output.
// The whole picture is scaled inside the rectangle with blank space to
// right and top
func ScaleImageInside(bounds image.Rectangle, maxW, maxH int) image.Rectangle {
	imgW := bounds.Max.X
	imgH := bounds.Max.Y
	ratio := float64(maxW) / float64(imgW)
	if r := float64(maxH) / float64(imgH); r < ratio {
		ratio = r
	}
	scaledW := int(ratio * float64(imgW))
	scaledH := int(ratio * float64(imgH))
	return image.Rect(0, 0, scaledW, scaledH)
}

// Calculated linear scaling of an rectangle from its original size to
// a max width and max height of a desired output.
// The whole picture is scaled inside the rectangle with blank space to
// right and top.  It is assumed the image is not rescaled when it is drawn
//
// Parameters
// - bounds is the size of the image
// - maxW and maxH is frame size needs to be mapped to
// - anchor which side of picture is to be anchored (might be better as a point)
func ScaleImageOuter(bounds image.Rectangle, target, clip image.Point) image.Rectangle {
	imgW := bounds.Max.X
	imgH := bounds.Max.Y
	// Ratio of screen to image, <1 means reduce image
	ratioH := float64(target.Y) / float64(imgH)
	ratioW := float64(target.X) / float64(imgW)
	var ratio float64
	scaledW := int(ratio * float64(imgW))
	scaledH := int(ratio * float64(imgH))
	top := 0
	left := 0
	switch {
	case ratioH > ratioW: // Scaling to expand width
		ratio = ratioH
		left += clip.X + (imgW-scaledW)/2
	case ratioW > ratioH: // Scaling to expand height
		ratio = ratioW
		top += clip.Y + (imgH-scaledH)/2
	}
	return image.Rect(left, top, scaledW+left, scaledH+top)
}

var ColourNameToRGBA = map[string]color.NRGBA{
	"darkgray": {R: 0x55, G: 0x57, B: 0x53},
	"red":      {R: 0xEF, G: 0x29, B: 0x29},
	"green":    {R: 0x8A, G: 0xE2, B: 0x34},
	"yellow":   {R: 0xFC, G: 0xE9, B: 0x4F},
	"blue":     {R: 0x72, G: 0x9F, B: 0xCF},
	"magenta":  {R: 0xEE, G: 0x38, B: 0xDA},
	"cyan":     {R: 0x34, G: 0xE2, B: 0xE2},
	"white":    {R: 0xEE, G: 0xEE, B: 0xEC},
}
