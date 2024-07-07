package panel

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/fogleman/gg"
)

type Panel interface {
	Render(buffer *image.RGBA, location image.Rectangle)
}

type PlainPanel struct {
	// config
	img     draw.Image
	Bounds  image.Rectangle
	W, H    int
	buffer  *image.RGBA
	bgcolor color.RGBA
	g       *gg.Context
}

type ImagePanel struct {
	// config
	img     image.Image
	Bounds  image.Rectangle
	W, H    int
	buffer  *image.RGBA
	bgcolor color.RGBA
	g       *gg.Context
}

func NewPlainPanel(w, h int) *PlainPanel {
	p := new(PlainPanel)
	p.g = gg.NewContext(w, h)
	return p
}

// This does the initial rendering of the image to
// create the static image.  This is then copied
// during the rendering process
// Other panels might update the content on each render
func NewImagePanel(w, h int, img image.Image) *ImagePanel {
	p := new(ImagePanel)
	p.g = gg.NewContext(w, h)
	p.img = img
	// We do all prerendering into an *gg.Context, for which all drawing
	// operations are optimized in Go.
	// buffer := image.NewRGBA(bounds)
	// draw.Draw(buffer, bounds, &image.Uniform{bgcolor}, image.Point{}, draw.Src)

	// place the gopher in the bottom (centered)
	// borderTop := 50
	// photoRect := drawing.ScaleImage(displayPhoto.Bounds(), w, h-borderTop)
	// log.Printf("Bounds w %v, h %v and borderTop %v ", w, h, borderTop)
	// photoRect = photoRect.Add(image.Point{0, 0})
	// padX = 0         //photoRect.Size().X / 2
	// padY := borderTop //+ ((h/2)-photoRect.Size().Y)/2
	// photoRect = photoRect.Add(image.Point{padX, padY})
	// xdraw.BiLinear.Scale(buffer, photoRect, displayPhoto, displayPhoto.Bounds(), draw.Over, nil)

	return p
}

// // Draws on an image buffer the contents of the panel
// // the location size should be the same as the initial image
// func (p *PlainPanel)Render(buffer *image.RGBA, location image.Rectangle) struct {
// 	// draw.Draw(buffer, location, p.g.Image(), image.Point{}, draw.Src)
// }

func (p *ImagePanel) Render(buffer *image.RGBA, location image.Rectangle) {
	draw.Draw(buffer, location, p.g.Image(), image.Point{}, draw.Src)
}
