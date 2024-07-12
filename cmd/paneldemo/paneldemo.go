//	Aim is to mirror gophoto structure but instead of putting the frame buffer onto the linux console to display
//
// on a web browser as an image.
// For bonus points will refresh itself when the code changes
package main

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

	"github.com/drummonds/gophoto/internal/frame"
	"github.com/drummonds/gophoto/internal/panel"
)

type PictureFrame struct {
	devFrameBuffer draw.Image      // This is the end frame buffer (turned into PNG)
	bounds         image.Rectangle // this is the size of frame buffer and the intermediate buffer
	bgcolor        color.RGBA

	buffer  *image.RGBA //This is the intermediate buffer which is drawn on
	picture *panel.ImagePanel
}

//go:embed "P1120981.png"
var displayPhotoPNG []byte

func main() {
	fmt.Print(("Hello\n"))
	mockFrameBuffer := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{1960, 1080}})

	pf := frame.NewPictureFrame(mockFrameBuffer.Bounds())

	// borderTop := 30
	// margin := 20
	// get the main image
	displayPhoto, _, err := image.Decode(bytes.NewReader(displayPhotoPNG))
	if err != nil {
		panic("Can't find photo")
	}
	pf.AddPanel(displayPhoto)

	// pf.picture = panel.NewImagePanel(w-margin*2, h-borderTop-margin*2, displayPhoto)
	// // place the photo in the bottom (centered)
	// photoRect := drawing.ScaleImage(displayPhoto.Bounds(), w-margin*2, h-borderTop-margin*2)
	// // Now move image to center
	// padX := margin + (w-photoRect.Size().X)/2
	// padY := borderTop + ((h-borderTop)-photoRect.Size().Y)/2
	// photoRect = photoRect.Add(image.Point{padX, padY})
	// pf.picture.Render(pf.buffer, photoRect)
	// Copy intermediate buffer to frame buffer
	draw.Draw(mockFrameBuffer, pf.bounds, pf.buffer, image.Point{}, draw.Src)
	// Encode frame buffer as PNG and save
	f, _ := os.Create("framebuffer.png")
	png.Encode(f, mockFrameBuffer)
}
