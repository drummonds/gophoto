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
	"bytes"
	"fmt"
	"image"
	"os"

	"image/color"
	"image/draw"
	"image/png"

	_ "embed"
	_ "image/png"
	"github.com/drummonds/gophoto/internal/panel"
}

// This is the structure which holds the screen data.
type PictureFrame struct {
	// config
	bounds         image.Rectangle
	w, h           int
	scaleFactor    float64
	buffer         *image.RGBA // This is what is output to the screen via the frame buffer
	bgcolor        color.RGBA
	panels []*panel.Panel;
}

// Create a new picture frame at a defined size, eg defined by browser window
// for frame size
func NewPlainPictureFrame(bounds image.Rectangle) *PictureFrame {
	pf := new(PictureFrame)
	pf.bounds = bounds
	w := pf.bounds.Max.X
	h := pf.bounds.Max.Y
	pf.bgcolor = color.RGBA{R: 0xF4, G: 0xC7, B: 0xDF, A: 255}
	// Create intermediate buffer
	pf.buffer = image.NewRGBA(pf.bounds)
	draw.Draw(pf.buffer, pf.bounds, &image.Uniform{pf.bgcolor}, image.Point{}, draw.Src)
	pf.panels = make([]*panel.Panel, 0, 5)
		// Fill buffer background

	return pf
}

func (pf *PictureFrame) AddPanel(panel *panel.PanelFrame) err error {
	pf.panels = pf.panels.append(panel)
	return nil
}

// Calls all the child panels to rerender them
func (pf *PictureFrame) Render() err error {
	for _,panel := range pf.panels {
		panel.Render()
	}
	pf.panels = pf.panels.append(panel)
	return nil
}
