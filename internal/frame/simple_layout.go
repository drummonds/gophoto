// Some standard layouts

package frame

import (
	"bytes"
	"image"
	"log"

	_ "embed"
	_ "image/png"

	"github.com/drummonds/gophoto/internal/drawing"
	"github.com/drummonds/gophoto/internal/panel"
)

// gokrazy
//
//go:embed "P1120981.png"
var displayPhotoPNG []byte

func (pf *PictureFrame) SetupBoundedStaticImage() {
	borderTop := 30
	margin := 20
	// get the main image
	displayPhoto, _, err := image.Decode(bytes.NewReader(displayPhotoPNG))
	if err != nil {
		panic("Can't find photo")
	}
	picture := panel.NewImagePanel(displayPhoto)
	// picture.Resize ( pf.Bounds.Max.X-margin*2, pf.Bounds.Max.Y-borderTop-margin*2,
	photoRect := drawing.ScaleImageInside(displayPhoto.Bounds(), pf.Bounds.Max.X-margin*2, pf.Bounds.Max.Y-borderTop-margin*2)
	// Now move image to center
	padX := margin + (pf.Bounds.Max.X-photoRect.Size().X)/2
	padY := borderTop + ((pf.Bounds.Max.X-borderTop)-photoRect.Size().Y)/2
	picture.Location = photoRect.Add(image.Point{padX, padY})
	pf.AddPanel(picture)
	// pf.SetBGColour(0xF4, 0xC7, 0xDF)
	log.Printf("Picture location - %v", picture.Location)
}

func (pf *PictureFrame) SetupFullStaticImage() {
	// get the main image
	displayPhoto, _, err := image.Decode(bytes.NewReader(displayPhotoPNG))
	if err != nil {
		panic("Can't find photo")
	}
	picture := panel.NewImagePanel(displayPhoto)
	photoRect := drawing.ScaleImageOuter(displayPhoto.Bounds(), pf.Bounds.Size(), image.Point{0, 0})
	// Now move image to center
	picture.Location = photoRect
	pf.AddPanel(picture)
	log.Printf("Picture location - %v", picture.Location)
}
