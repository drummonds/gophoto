package panel

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/fogleman/gg"
	xdraw "golang.org/x/image/draw"
)

type PlainPanel struct {
	// config
	img      draw.Image
	Bounds   image.Rectangle
	W, H     int
	buffer   *image.RGBA
	bgcolor  color.RGBA
	g        *gg.Context
	Location image.Rectangle // Where panel is to be rendered
}

type ImagePanel struct {
	// config
	img      image.Image
	Bounds   image.Rectangle
	W, H     int
	buffer   *image.RGBA
	bgcolor  color.RGBA
	g        *gg.Context
	Location image.Rectangle // Where panel is to be rendered
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
func NewImagePanel(img image.Image) *ImagePanel {
	p := new(ImagePanel)
	p.img = img
	p.Resize(p.img.Bounds())
	return p
}

// // the location size should be the same as the initial image
func (p *ImagePanel) Resize(bounds image.Rectangle) {
	p.Bounds = bounds
	p.W = p.Bounds.Max.X
	p.H = p.Bounds.Max.Y
	p.g = gg.NewContext(p.W, p.H)
}

// // Draws on an image buffer the contents of the panel
// // the location size should be the same as the initial image
func (p *ImagePanel) Render(buffer *image.RGBA) {
	draw.Draw(buffer, p.Location, p.g.Image(), image.Point{0, 0}, draw.Src)
	xdraw.BiLinear.Scale(buffer, p.Location, p.img, p.img.Bounds(), draw.Over, nil)
}
