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

	"image/draw"
	"image/png"

	_ "embed"
	_ "image/png"

	"github.com/drummonds/gophoto/internal/drawing"
	"github.com/drummonds/gophoto/internal/frame"
	"github.com/drummonds/gophoto/internal/panel"
)

//go:embed "P1120981.png"
var displayPhotoPNG []byte

func main() {
	fmt.Print(("Hello\n"))
	mockFrameBuffer := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{1960, 1080}})

	pf := frame.NewPictureFrame(mockFrameBuffer.Bounds())

	borderTop := 30
	margin := 20
	// get the main image
	displayPhoto, _, err := image.Decode(bytes.NewReader(displayPhotoPNG))
	if err != nil {
		panic("Can't find photo")
	}
	picture := panel.NewImagePanel(displayPhoto)
	// picture.Resize ( pf.W-margin*2, pf.H-borderTop-margin*2,
	photoRect := drawing.ScaleImageInside(displayPhoto.Bounds(), pf.W-margin*2, pf.H-borderTop-margin*2)
	// Now move image to center
	padX := margin + (pf.W-photoRect.Size().X)/2
	padY := borderTop + ((pf.H-borderTop)-photoRect.Size().Y)/2
	picture.Location = photoRect.Add(image.Point{padX, padY})
	pf.AddPanel(picture)
	// pf.SetBGColour(0xF4, 0xC7, 0xDF)

	// Copy intermediate buffer to frame buffer
	pf.Render()
	draw.Draw(mockFrameBuffer, pf.Bounds, pf.Buffer, image.Point{}, draw.Src)
	// Encode frame buffer as PNG and save
	f, _ := os.Create("framebuffer.png")
	png.Encode(f, mockFrameBuffer)
}
