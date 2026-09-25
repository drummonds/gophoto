//	Aim is to mirror gophoto structure but instead of putting the frame buffer onto the linux console to display
//
// on a web browser as an image.
// For bonus points will refresh itself when the code changes
package main

import (
	"fmt"
	"image"
	"log"
	"os"

	"image/draw"
	"image/png"

	_ "embed"
	_ "image/png"

	"git.bytestone.uk/hum3/gophoto/internal/frame"
)

func main() {
	fmt.Print(("Hello\n"))
	mockFrameBuffer := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{1920, 1080}})

	pf := frame.NewPictureFrame(mockFrameBuffer.Bounds())

	// pf.SetupBoundedStaticImage()
	pf.SetupFullStaticImage()

	// Copy intermediate buffer to frame buffer
	pf.RenderPanels()
	draw.Draw(mockFrameBuffer, pf.Bounds, pf.Buffer, image.Point{}, draw.Src)
	// Encode frame buffer as PNG and save
	f, err := os.Create("framebuffer.png")
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(f, mockFrameBuffer); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}
