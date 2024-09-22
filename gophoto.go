// Program gophoto graphically shows the pictures on the Linux
// frame buffer, which is typically available via HDMI when running on a
// Raspberry Pi or a PC.
// This is a derivative work of  gokrazy/fbstatus
// Which is apache licensed
// Any of my work is MIT licensed
// V0.0.17 2024-07-07
package main

import (
	"context"
	"fmt"
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
	"github.com/go-ping/ping"

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

// Called once to set up newConsole
func newConsolePicture(devFrameBuffer draw.Image) (*ConsolePicture, error) {
	cp := new(ConsolePicture)
	cp.frameBuffer = devFrameBuffer

	cp.pf = frame.NewPictureFrame(cp.frameBuffer.Bounds())

	// cp.pf.SetupBoundedStaticImage()
	// cp.pf.SetupFullStaticImage()
	// err = cp.pf.SetupFullPhotoPrism()
	ctx := context.Background()
	err := frame.NewPhotoPrism(ctx)
	log.Printf("Done newConsolePicture %s\n", time.Now().Format(time.RFC3339))
	return cp, err
}

// repaint any live panels to buffer
// Call rerender
func (cp *ConsolePicture) render(ctx context.Context) error {
	log.Printf("Starting render %s\n", time.Now().Format(time.RFC3339))

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
	// cp.pf.Render()
	// Refresh the image and redraw
	frame.GlobalPage.PhotoIndex++
	if frame.GlobalPage.PhotoIndex >= len(frame.GlobalPhotoList) {
		frame.GlobalPage.PhotoIndex = 0
	}
	log.Printf("Get new image %s\n", time.Now().Format(time.RFC3339))
	img, err := frame.NewImage(ctx, cp.frameBuffer.Bounds())
	if err != nil {
		return err
	}
	log.Printf("Got new image %s\n", time.Now().Format(time.RFC3339))
	b := img.Bounds()
	m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
	cp.pf.Buffer = m

	log.Printf("%s Got image", time.Now().Format(time.RFC3339))

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
	log.Printf("%s Completed render %v ", time.Now().Format(time.RFC3339), cp.lastCopy)
	return nil
}

func gophoto(ctx context.Context) error {
	log.Printf("Starting gophoto %s\n", time.Now().Format(time.RFC3339))

	// Take over the frame buffer and cleanup afterwards
	cons, err := console.LeaseForGraphics()
	if err != nil {
		return err
	}
	log.Printf("Got console lease %s\n", time.Now().Format(time.RFC3339))
	defer func() {
		// Seems to generate VT_DISALLOCATE(2): device or resource busy
		if err := cons.Cleanup(); err != nil {
			log.Print(err)
		}
	}()
	log.Printf("Got console lease %s\n", time.Now().Format(time.RFC3339))
	dev, err := fb.Open("/dev/fb0")
	if err != nil {
		return err
	}
	log.Printf("Got FB %s\n", time.Now().Format(time.RFC3339))

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

	log.Printf("%s Start event loop ", time.Now().Format(time.RFC3339))
	for {
		if cons.Visible() {
			if err := ConsolePicture.render(ctx); err != nil {
				return err
			}
		}
		log.Printf("Start sleep")
		time.Sleep(15 * time.Second)
		log.Printf("End sleep")

		// select {
		// case <-ctx.Done():
		// 	// return to trigger the deferred cleanup function
		// 	return ctx.Err()

		// case <-cons.Redraw():
		// 	break // next iteration
		// }

	}
}

func fatalExit(err error) {
	// time.Sleep(30 * time.Second) // Don't hammer it with crashes
	log.Printf("(fmt) Fatal error - won't return.\n%+v", err)
	log.Printf("(log) Fatal error - won't return.\n%+v", err)
	os.Exit(125) // Don;t rerun
}

func enableUnprivilegedPing() error {
	return os.WriteFile("/proc/sys/net/ipv4/ping_group_range", []byte("0\t2147483647"), 0600)
}

func pingPhotoPrism() {
	pinger, err := ping.NewPinger("10.0.0.1")
	if err != nil {
		fatalExit(err)
	}
	pinger.Count = 3
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		fatalExit(err)
	}
	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats
	log.Printf("%+v\n", stats)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fatalExit(fmt.Errorf("Recovered in f %+v", r))
		}
	}()
	version := "GoPhoto V0.2.22"
	time.Sleep(5 * time.Second) // log not ready immed?
	log.Printf("%s 2024-09-22 %s\n", version, time.Now().Format(time.RFC3339))
	time.Sleep(5 * time.Second)
	log.Printf("Finished sleep %s\n", time.Now().Format(time.RFC3339))
	if err := enableUnprivilegedPing(); err != nil {
		fatalExit(err)
	}
	log.Printf("Updated ping privilege %s\n", time.Now().Format(time.RFC3339))
	pingPhotoPrism()
	log.Printf("%sa %s\n", version, time.Now().Format(time.RFC3339))
	for _, s := range []string{"ALBUM_UID", "PHOTOPRISM_DOMAIN", "PHOTOPRISM_TOKEN"} {
		log.Printf("Env: %s = %s\n", s, os.Getenv(s))
	}
	ctx := context.Background()

	// Cancel the context instead of exiting the program:
	ctx, canc := signal.NotifyContext(ctx, os.Interrupt)
	defer canc()
	if err := gophoto(ctx); err != nil {
		fatalExit(err)
	}
}
