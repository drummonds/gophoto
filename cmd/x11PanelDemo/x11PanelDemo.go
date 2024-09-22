// aim is to use gophoto panel mechanism
// to display a simulated frame buffer
package main

import (
	"fmt"
	"image"
	_ "image/png"
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/drummonds/gophoto/internal/frame"
	"golang.org/x/image/draw"
)

const (
	hdDivider    = 5
	windowWidth  = 1920 / hdDivider
	windowHeight = 1080 / hdDivider
)

func handleExposeEvent(X *xgb.Conn, wid xproto.Window, img image.Image) {
	gc, _ := xproto.NewGcontextId(X)
	xproto.CreateGC(X, gc, xproto.Drawable(wid), 0, nil)

	for y := 0; y < windowHeight; y++ {
		for x := 0; x < windowWidth; x++ {
			c := img.At(x, y)
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

func main() {
	X, wid, atomWmDeleteWindow, atomWmProtocols, err := NewX(windowWidth, windowHeight)
	if err != nil {
		log.Fatal(err)
	}
	defer X.Close()

	// Get the image

	fmt.Print(("Hello\n"))
	mockFrameBuffer := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{1920, 1080}})

	pf := frame.NewPictureFrame(mockFrameBuffer.Bounds())

	// pf.SetupBoundedStaticImage()
	pf.SetupFullStaticImage()

	// Copy intermediate buffer to frame buffer
	pf.RenderPanels()
	draw.Draw(mockFrameBuffer, pf.Bounds, pf.Buffer, image.Point{}, draw.Src)

	for {
		ev, err := X.WaitForEvent()
		if err != nil {
			log.Println(err)
			continue
		}

		switch e := ev.(type) {
		case xproto.ExposeEvent:
			handleExposeEvent(X, wid, mockFrameBuffer)
		case xproto.ClientMessageEvent:
			if e.Type == atomWmProtocols && e.Data.Data32[0] == uint32(atomWmDeleteWindow) {
				return
			}
		case xproto.KeyPressEvent:
			return
		}
	}
}
