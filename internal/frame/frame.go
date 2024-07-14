/*
A screen represents a complete rectangular area which presents data.

This might be a frame buffer or background image.

The screen is rendered by pasting panels on it.
This is done it two stages:

- Initial creation
- Rendering dynamic content
*/
package frame

import (
	"image"

	"image/color"
	"image/draw"

	_ "embed"
	_ "image/png"
)

type Panelled interface {
	Render(buffer *image.RGBA)
}

// This is the structure which holds the screen data.
type PictureFrame struct {
	// config
	Bounds      image.Rectangle
	W, H        int
	scaleFactor float64
	Buffer      *image.RGBA // This is what is output to the screen via the frame buffer
	BGColour    color.RGBA
	panels      []Panelled
}

// Create a new picture frame at a defined size, eg defined by browser window
// for frame size
func NewPictureFrame(bounds image.Rectangle) *PictureFrame {
	pf := new(PictureFrame)
	pf.Bounds = bounds
	pf.W = pf.Bounds.Max.X
	pf.H = pf.Bounds.Max.Y
	pf.BGColour = color.RGBA{R: 0xF4, G: 0xC7, B: 0xDF, A: 255}
	// Create intermediate buffer
	pf.Buffer = image.NewRGBA(pf.Bounds)
	pf.RepaintBackground()
	pf.panels = make([]Panelled, 0, 5)
	// Fill buffer background

	return pf
}

func (pf *PictureFrame) SetBGColour(r, g, b uint8) {
	pf.BGColour = color.RGBA{R: r, G: g, B: b, A: 255}
	pf.RepaintBackground()
}

func (pf *PictureFrame) RepaintBackground() {
	draw.Draw(pf.Buffer, pf.Bounds, &image.Uniform{pf.BGColour}, image.Point{}, draw.Src)
}

func (pf *PictureFrame) AddPanel(panel Panelled) error {
	pf.panels = append(pf.panels, panel)
	return nil
}

// Calls all the child panels to rerender them
func (pf *PictureFrame) Render() error {
	for _, panel := range pf.panels {
		// Recreate panel content if changed
		panel.Render(pf.Buffer)
		// apply panel content to buffer
		// pf.picture.Render(pf.buffer, photoRect)
	}
	return nil
}
