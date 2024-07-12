// Program gophoto graphically shows the pictures on the Linux
// frame buffer, which is typically available via HDMI when running on a
// Raspberry Pi or a PC.
// This is a derivative work of  gokrazy/fbstatus
// Which is apache licensed
// Any of my work is MIT licensed
// V0.0.17 2024-07-07
package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"

	"github.com/drummonds/gophoto/internal/console"
	"github.com/drummonds/gophoto/internal/drawing"
	"github.com/drummonds/gophoto/internal/fb"
	"github.com/drummonds/gophoto/internal/fbimage"
	"github.com/drummonds/gophoto/internal/panel"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font/gofont/goregular"

	_ "embed"
	_ "image/png"
)

type PictureFrame struct {
	// config
	bounds      image.Rectangle
	w, h        int
	scaleFactor float64
	buffer      *image.RGBA // This is what is output to the screen via the frame buffer
	bgcolor     color.RGBA
	g           *gg.Context
	picture     *panel.ImagePanel

	// state
	slowPathNotified     bool
	last                 [][][]string
	lastRender, lastCopy time.Duration
}

func newPictureFrame(devFrameBuffer draw.Image) (*PictureFrame, error) {
	// devFrameBuffer is the frame buffer on which we need to draw
	// all the panels
	// Todo
	// var padX int
	pf := new(PictureFrame)
	pf.devFrameBuffer = devFrameBuffer
	pf.bounds = devFrameBuffer.Bounds()
	w := pf.bounds.Max.X
	h := pf.bounds.Max.Y
	borderTop := 50

	scaleFactor := math.Floor(float64(w) / 1024)
	if scaleFactor < 1 {
		scaleFactor = 1
	}
	log.Printf("font scale factor: %.f", scaleFactor)

	// get the main image
	displayPhoto, _, err := image.Decode(bytes.NewReader(displayPhotoPNG))
	if err != nil {
		return nil, err
	}
	pf.picture = panel.NewImagePanel(w, h-borderTop, displayPhoto)

	pf.bgcolor = color.RGBA{R: 0xCC, G: 0xB7, B: 0x7D, A: 255}

	// We do all rendering into an *image.RGBA buffer, for which all drawing
	// operations are optimized in Go. Only at the very end do we copy the
	// buffer contents to the framebuffer (BGR565 or BGRA)
	pf.buffer = image.NewRGBA(pf.bounds)
	draw.Draw(pf.buffer, pf.bounds, &image.Uniform{pf.bgcolor}, image.Point{}, draw.Src)

	// place the photo in the bottom (centered)
	photoRect := drawing.ScaleImage(displayPhoto.Bounds(), w, h-borderTop)
	log.Printf("Bounds w %v, h %v and borderTop %v ", w, h, borderTop)
	// Now move image to center
	padX := photoRect.Size().X / 2
	padY := borderTop + ((h-borderTop)-photoRect.Size().Y)/2
	photoRect = photoRect.Add(image.Point{padX, padY})

	t1 := time.Now()
	xdraw.BiLinear.Scale(pf.buffer, photoRect, displayPhoto, displayPhoto.Bounds(), draw.Over, nil)
	log.Printf("Destination photorect %v", photoRect)
	log.Printf("Displayphoto bound %v", displayPhoto.Bounds())
	log.Printf("Photo scaled in %v", time.Since(t1))

	pf.g = gg.NewContext(w/2, h/2)

	// draw textual information in a block of key: value details
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}

	size := float64(16)
	size *= scaleFactor
	face := truetype.NewFace(font, &truetype.Options{Size: size})
	pf.g.SetFontFace(face)

	// padX = ((w / 2) - int(66*scaleFactor)) / 2
	// pictureTitle.DrawString("Tay still going V0.0.15!", float64(padX)-(30*scaleFactor), 1*scaleFactor)

	return pf, nil
}

func (d *PictureFrame) render(ctx context.Context) error {
	// repaint any live panels to buffer
	// copy buffer to screen

	// --------------------------------------------------------------------------------
	for idx := range d.last {
		if idx == len(d.last)-1 {
			break
		}
		d.last[idx] = d.last[idx+1]
	}

	t2 := time.Now()
	{
		r, g, b, a := d.bgcolor.RGBA()
		d.g.SetRGBA(
			float64(r)/0xffff,
			float64(g)/0xffff,
			float64(b)/0xffff,
			float64(a)/0xffff)
	}
	d.g.Clear()
	d.g.SetRGB(1, 1, 1)
	// The constant time update keeps an animation going
	//??

	leftHalf := image.Rect(0, d.h/2-50, d.w/2, d.h/2)
	log.Printf("About to draw %T", d.g)
	draw.Draw(d.buffer, leftHalf, d.g.Image(), image.Point{}, draw.Src)
	log.Print("Has drawn")

	// photoLocation := image.Rect(0, 0, d.w/2, d.h/2)
	// d.picture.Render(d.buffer, photoLocation)

	d.lastRender = time.Since(t2)

	t3 := time.Now()
	// NOTE: This code path is NOT using double buffering (which is done
	// using the pan ioctl when using the frame buffer), but in practice
	// updates seem smooth enough, most likely because we are only
	// updating timestamps.
	switch x := d.devFrameBuffer.(type) {
	case *fbimage.BGR565:
		drawing.CopyRGBAtoBGR565(x, d.buffer)
	case *fbimage.BGRA:
		drawing.CopyRGBAtoBGRA(x, d.buffer)
	default:
		if !d.slowPathNotified {
			log.Printf("framebuffer not using pixel format BGR565, falling back to slow path for devFrameBuffer type %T", d.devFrameBuffer)
			d.slowPathNotified = true
		}
		draw.Draw(d.devFrameBuffer, d.bounds, d.buffer, image.Point{}, draw.Src)
	}
	d.lastCopy = time.Since(t3)
	return nil
}

func gophoto(ctx context.Context) error {

	// Take over the frame buffer and cleanup afterwards
	cons, err := console.LeaseForGraphics()
	if err != nil {
		return err
	}
	defer func() {
		// Seems to generate VT_DISALLOCATE(2): device or resource busy
		if err := cons.Cleanup(); err != nil {
			log.Print(err)
		}
	}()

	dev, err := fb.Open("/dev/fb0")
	if err != nil {
		return err
	}

	if info, err := dev.VarScreeninfo(); err == nil {
		log.Printf("framebuffer screeninfo: %+v", info)
	}

	devFrameBuffer, err := dev.Image()
	if err != nil {
		return err
	}

	pictureFrame, err := newPictureFrame(devFrameBuffer)
	if err != nil {
		return err
	}

	// Event loop, render every second
	tick := time.Tick(1 * time.Second)
	for {
		if cons.Visible() {
			if err := pictureFrame.render(ctx); err != nil {
				return err
			}
		}

		select {
		case <-ctx.Done():
			// return to trigger the deferred cleanup function
			return ctx.Err()

		case <-cons.Redraw():
			break // next iteration

		case <-tick:
			break
		}
	}
}

// gokrazy
//
//go:embed "P1120981.png"
var displayPhotoPNG []byte

func main() {
	ctx := context.Background()

	// Cancel the context instead of exiting the program:
	ctx, canc := signal.NotifyContext(ctx, os.Interrupt)
	defer canc()
	if err := gophoto(ctx); err != nil {
		log.Fatal(err)
	}
}
