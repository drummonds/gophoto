// aim is to use just use a
// to display a simulated frame buffer
// and a mechanism to refresh every second
// THis is not using the panel mechanism
package main

import (
	"context"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/drummonds/gophoto/internal/frame"
)

const (
	hdDivider    = 5
	windowWidth  = 1920 / hdDivider
	windowHeight = 1080 / hdDivider
)

func handleExposeEvent(ctx context.Context, X *xgb.Conn, wid xproto.Window, mockFrameBuffer image.Image) {
	gc, _ := xproto.NewGcontextId(X)
	xproto.CreateGC(X, gc, xproto.Drawable(wid), 0, nil)
	// copy image to output pixel by pixel
	for y := 0; y < windowHeight; y++ {
		for x := 0; x < windowWidth; x++ {
			c := mockFrameBuffer.At(x, y)
			r, g, b, _ := c.RGBA()
			color := (r >> 8 << 16) | (g >> 8 << 8) | (b >> 8)
			xproto.ChangeGC(X, gc, xproto.GcForeground, []uint32{uint32(color)})
			xproto.PolyPoint(X, xproto.CoordModeOrigin, xproto.Drawable(wid), gc, []xproto.Point{{int16(x), int16(y)}})
		}
	}
}

func NewX(width, height int) (*xgb.Conn, xproto.Window, xproto.Atom, xproto.Atom, error) {
	X, err := xgb.NewConn()
	if err != nil {
		return nil, 0, 0, 0, err
	}

	screen := xproto.Setup(X).DefaultScreen(X)
	wid, _ := xproto.NewWindowId(X)
	xproto.CreateWindow(X, screen.RootDepth, wid, screen.Root,
		0, 0, uint16(width), uint16(height), 0,
		xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			0xffffffff,
			xproto.EventMaskExposure | xproto.EventMaskKeyPress | xproto.EventMaskStructureNotify,
		})

	// Set WM_PROTOCOLS to handle window close
	atomWmDeleteWindow, _ := xproto.InternAtom(X, false, uint16(len("WM_DELETE_WINDOW")), "WM_DELETE_WINDOW").Reply()
	atomWmProtocols, _ := xproto.InternAtom(X, false, uint16(len("WM_PROTOCOLS")), "WM_PROTOCOLS").Reply()
	xproto.ChangeProperty(X, xproto.PropModeReplace, wid, atomWmProtocols.Atom, xproto.AtomAtom, 32, 1, []byte{byte(atomWmDeleteWindow.Atom), 0, 0, 0})

	xproto.MapWindow(X, wid)
	return X, wid, atomWmDeleteWindow.Atom, atomWmProtocols.Atom, nil
}

func fatalError(err error) {
	log.Println(fmt.Errorf("Fatal error - won't return.\n%+v", err))
	time.Sleep(30 * time.Second)
	log.Fatal(err)
}

func main() {
	X, wid, atomWmDeleteWindow, atomWmProtocols, err := NewX(windowWidth, windowHeight)
	if err != nil {
		log.Fatal(err)
	}
	defer X.Close()

	ctx := context.Background()
	err = frame.NewPhotoPrism(ctx)
	if err != nil {
		fatalError(err)
	}

	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()
	var mockFrameBuffer image.Image
	// get first image
	mockFrameBuffer, err = frame.NewImage(ctx, image.Rect(0, 0, windowWidth, windowHeight))
	handleExposeEvent(ctx, X, wid, mockFrameBuffer)
	for {
		select {
		case <-ticker.C:
			// Refresh the image and redraw
			frame.GlobalPage.PhotoIndex++
			if frame.GlobalPage.PhotoIndex >= len(frame.GlobalPhotoList) {
				frame.GlobalPage.PhotoIndex = 0
			}
			mockFrameBuffer = frame.NewImage(ctx, image.Rect(0, 0, windowWidth, windowHeight))
			handleExposeEvent(ctx, X, wid, mockFrameBuffer)
			// drain the ticker channel
			for len(ticker.C) > 0 {
				<-ticker.C
			}

		default:
			ev, err := X.PollForEvent()
			if err != nil {
				// This is the correct way to handle connection errors
				log.Println(err)
				continue
			}
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case xproto.ExposeEvent:
				handleExposeEvent(ctx, X, wid, mockFrameBuffer)
			case xproto.ClientMessageEvent:
				if e.Type == atomWmProtocols && e.Data.Data32[0] == uint32(atomWmDeleteWindow) {
					return
				}
			case xproto.KeyPressEvent:
				return
			}
		}
	}
}
