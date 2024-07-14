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
	"image/draw"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"

	"github.com/drummonds/gophoto/internal/console"
	"github.com/drummonds/gophoto/internal/drawing"
	"github.com/drummonds/gophoto/internal/fb"
	"github.com/drummonds/gophoto/internal/fbimage"
	"github.com/drummonds/gophoto/internal/frame"
	"github.com/drummonds/gophoto/internal/panel"

	_ "embed"
	_ "image/png"
)

type ConsolePicture struct {
	// config
	frameBuffer draw.Image // This is what is output to the screen via the frame buffer
	pf          *frame.PictureFrame

	// state
	slowPathNotified     bool
	last                 [][][]string
	lastRender, lastCopy time.Duration
	renderCount          int
}

func setupBoundedStaticImage(cp *ConsolePicture) {
	borderTop := 30
	margin := 20
	// get the main image
	displayPhoto, _, err := image.Decode(bytes.NewReader(displayPhotoPNG))
	if err != nil {
		panic("Can't find photo")
	}
	picture := panel.NewImagePanel(displayPhoto)
	// picture.Resize ( pf.W-margin*2, pf.H-borderTop-margin*2,
	photoRect := drawing.ScaleImageInside(displayPhoto.Bounds(), cp.pf.W-margin*2, cp.pf.H-borderTop-margin*2)
	// Now move image to center
	padX := margin + (cp.pf.W-photoRect.Size().X)/2
	padY := borderTop + ((cp.pf.H-borderTop)-photoRect.Size().Y)/2
	picture.Location = photoRect.Add(image.Point{padX, padY})
	cp.pf.AddPanel(picture)
	// pf.SetBGColour(0xF4, 0xC7, 0xDF)
}

func setupFullStaticImage(cp *ConsolePicture) {
	// get the main image
	displayPhoto, _, err := image.Decode(bytes.NewReader(displayPhotoPNG))
	if err != nil {
		panic("Can't find photo")
	}
	picture := panel.NewImagePanel(displayPhoto)
	// picture.Resize ( pf.W-margin*2, pf.H-borderTop-margin*2,
	photoRect := drawing.ScaleImageOuter(displayPhoto.Bounds(), cp.pf.W, cp.pf.H, drawing.Bottom)
	// Now move image to center
	picture.Location = photoRect.Add(image.Point{0, 0})
	cp.pf.AddPanel(picture)
}

func newConsolePicture(devFrameBuffer draw.Image) (*ConsolePicture, error) {
	cp := new(ConsolePicture)
	cp.frameBuffer = devFrameBuffer

	cp.pf = frame.NewPictureFrame(cp.frameBuffer.Bounds())

	// setupBoundedStaticImage(cp)
	setupFullStaticImage(cp)

	return cp, nil
}

// repaint any live panels to buffer
func (cp *ConsolePicture) render(ctx context.Context) error {
	cp.renderCount += 1
	// copy buffer to screen

	// --------------------------------------------------------------------------------
	for idx := range cp.last {
		if idx == len(cp.last)-1 {
			break
		}
		cp.last[idx] = cp.last[idx+1]
	}

	t2 := time.Now()
	cp.pf.Render()
	cp.lastRender = time.Since(t2)

	t3 := time.Now()
	// NOTE: This code path is NOT using double buffering (which is done
	// using the pan ioctl when using the frame buffer), but in practice
	// updates seem smooth enough, most likely because we are only
	// updating timestamps.
	switch x := cp.frameBuffer.(type) {
	case *fbimage.BGR565:
		if cp.renderCount < 3 {
			log.Printf("framebuffer using pixel format BGR565")
		}
		drawing.CopyRGBAtoBGR565(x, cp.pf.Buffer)
	case *fbimage.BGRA:
		if cp.renderCount < 3 {
			log.Printf("framebuffer using pixel format BGRA")
		}
		drawing.CopyRGBAtoBGRA(x, cp.pf.Buffer)
	default:
		if !cp.slowPathNotified {
			if cp.renderCount < 3 {
				log.Printf("framebuffer not using pixel format BGR565, falling back to slow path for devFrameBuffer type %T", cp.frameBuffer)
			}
			cp.slowPathNotified = true
		}
		draw.Draw(cp.frameBuffer, cp.pf.Bounds, cp.pf.Buffer, image.Point{}, draw.Src)
	}
	cp.lastCopy = time.Since(t3)
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

	ConsolePicture, err := newConsolePicture(devFrameBuffer)
	if err != nil {
		return err
	}

	// Event loop, render every second
	tick := time.Tick(1 * time.Second)
	for {
		if cons.Visible() {
			if err := ConsolePicture.render(ctx); err != nil {
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
