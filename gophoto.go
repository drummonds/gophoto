// Program gophoto graphically shows the pictures on the Linux
// frame buffer, which is typically available via HDMI when running on a
// Raspberry Pi or a PC.
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"io/ioutil"
	"log"
	"math"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/drummonds/gophoto/internal/console"
	"github.com/drummonds/gophoto/internal/drawing"
	"github.com/drummonds/gophoto/internal/fb"
	"github.com/drummonds/gophoto/internal/fbimage"
	"github.com/drummonds/gophoto/internal/panel"
	"github.com/fogleman/gg"
	"github.com/gokrazy/gokrazy"
	"github.com/golang/freetype/truetype"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"

	_ "embed"
	_ "image/png"
)

func uptime() (string, error) {
	file, err := os.Open("/proc/uptime")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.Split(text, " ")
		dur, err := time.ParseDuration(parts[0] + "s")
		if err != nil {
			return "", err
		}
		return dur.Round(time.Second).String(), nil
	}
	return "", fmt.Errorf("BUG: parse /proc/uptime")
}

type PictureFrame struct {
	// config
	img          draw.Image
	bounds       image.Rectangle
	w, h         int
	scaleFactor  float64
	buffer       *image.RGBA
	files        map[string]*os.File
	bgcolor      color.RGBA
	hostname     string
	g            *gg.Context
	pictureTitle *gg.Context
	oldPicture   *gg.Context
	picture      *panel.ImagePanel

	// state
	slowPathNotified     bool
	last                 [][][]string
	lastRender, lastCopy time.Duration
}

func newPictureFrame(img draw.Image) (*PictureFrame, error) {
	// img is the frame buffer on which we need to draw
	// all the panels
	// Todo
	// var padX int
	pf := new(PictureFrame)
	pf.bounds = img.Bounds()
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

	bgcolor := color.RGBA{R: 0xCC, G: 0xB7, B: 0x7D, A: 255}

	// We do all rendering into an *image.RGBA buffer, for which all drawing
	// operations are optimized in Go. Only at the very end do we copy the
	// buffer contents to the framebuffer (BGR565 or BGRA)
	buffer := image.NewRGBA(pf.bounds)
	draw.Draw(buffer, pf.bounds, &image.Uniform{bgcolor}, image.Point{}, draw.Src)

	// place the photo in the bottom (centered)
	photoRect := drawing.ScaleImage(displayPhoto.Bounds(), w, h-borderTop)
	log.Printf("Bounds w %v, h %v and borderTop %v ", w, h, borderTop)
	// photoRect = photoRect.Add(image.Point{0, 0})
	// padX = 0         //photoRect.Size().X / 2
	// padY := borderTop //+ ((h/2)-photoRect.Size().Y)/2
	// photoRect = photoRect.Add(image.Point{padX, padY})

	t1 := time.Now()
	xdraw.BiLinear.Scale(buffer, photoRect, displayPhoto, displayPhoto.Bounds(), draw.Over, nil)
	log.Printf("Destination photorect %v", photoRect)
	log.Printf("Displayphoto bound %v", displayPhoto.Bounds())
	log.Printf("Photo scaled in %v", time.Since(t1))

	g := gg.NewContext(w/2, h/2)
	pf.pictureTitle = gg.NewContext(w, borderTop)
	pf.oldPicture = gg.NewContext(w, h-borderTop)

	// draw textual information in a block of key: value details
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}

	size := float64(16)
	size *= scaleFactor
	face := truetype.NewFace(font, &truetype.Options{Size: size})
	g.SetFontFace(face)

	italicfont, err := truetype.Parse(goitalic.TTF)
	if err != nil {
		return nil, err
	}
	italicface := truetype.NewFace(italicfont, &truetype.Options{Size: 2 * size})
	pf.pictureTitle.SetFontFace(italicface)

	{
		r, g, b, a := bgcolor.RGBA()
		pf.pictureTitle.SetRGBA(
			float64(r)/0xffff,
			float64(g)/0xffff,
			float64(b)/0xffff,
			float64(a)/0xffff)
	}
	//todo
	// pictureTitle.Clear()
	// pictureTitle.SetRGB(1, 1, 1)
	// padX = ((w / 2) - int(66*scaleFactor)) / 2
	// pictureTitle.DrawString("Tay still going V0.0.15!", float64(padX)-(30*scaleFactor), 1*scaleFactor)

	pf.hostname, err = os.Hostname()
	if err != nil {
		log.Print(err)
	}

	return pf, nil
}

func (d *PictureFrame) render(ctx context.Context) error {
	const lineSpacing = 1.5

	// --------------------------------------------------------------------------------
	contents := make(map[string][]byte)
	for path, fl := range d.files {
		if _, err := fl.Seek(0, io.SeekStart); err != nil {
			return err
		}
		b, err := ioutil.ReadAll(fl)
		if err != nil {
			return err
		}
		contents[path] = b
	}

	em, _ := d.g.MeasureString("m")

	for idx := range d.last {
		if idx == len(d.last)-1 {
			break
		}
		d.last[idx] = d.last[idx+1]
	}

	t2 := time.Now()
	{
		r, gg, b, a := d.bgcolor.RGBA()
		d.g.SetRGBA(
			float64(r)/0xffff,
			float64(gg)/0xffff,
			float64(b)/0xffff,
			float64(a)/0xffff)
	}
	d.g.Clear()
	d.g.SetRGB(1, 1, 1)
	// The constant time update keeps an animation going
	lines := []string{
		"host “" + d.hostname + "” (" + gokrazy.Model() + ")",
		"time: " + time.Now().Format(time.RFC3339),
	}
	//??
	if d.lastRender > 0 || d.lastCopy > 0 {
		last := len(lines) - 1
		lines[last] += fmt.Sprintf(", fb: draw %v, cp %v",
			d.lastRender.Round(time.Millisecond),
			d.lastCopy.Round(time.Millisecond))
	}
	texty := int(6 * em)

	for _, line := range lines {
		d.g.DrawString(line, 3*em, float64(texty))
		texty += int(d.g.FontHeight() * lineSpacing)
	}
	// leftHalf := image.Rect(0, d.h/2-50, d.w/2, d.h/2)
	// draw.Draw(d.buffer, leftHalf, d.g.Image(), image.Point{}, draw.Src)

	photoLocation := image.Rect(0, 0, d.w/2, d.h/2)
	// draw.Draw(d.buffer, photoLocation, d.oldPicture.Image(), image.Point{}, draw.Src)
	d.picture.Render(d.buffer, photoLocation)

	d.lastRender = time.Since(t2)

	t3 := time.Now()
	// NOTE: This code path is NOT using double buffering (which is done
	// using the pan ioctl when using the frame buffer), but in practice
	// updates seem smooth enough, most likely because we are only
	// updating timestamps.
	switch x := d.img.(type) {
	case *fbimage.BGR565:
		drawing.CopyRGBAtoBGR565(x, d.buffer)
	case *fbimage.BGRA:
		drawing.CopyRGBAtoBGRA(x, d.buffer)
	default:
		if !d.slowPathNotified {
			log.Printf("framebuffer not using pixel format BGR565, falling back to slow path for img type %T", d.img)
			d.slowPathNotified = true
		}
		draw.Draw(d.img, d.bounds, d.buffer, image.Point{}, draw.Src)
	}
	d.lastCopy = time.Since(t3)
	return nil
}

func gophoto() error {
	ctx := context.Background()

	// Cancel the context instead of exiting the program:
	ctx, canc := signal.NotifyContext(ctx, os.Interrupt)
	defer canc()

	// Take over the frame buffer
	cons, err := console.LeaseForGraphics()
	if err != nil {
		return err
	}
	defer func() {
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
	if err := gophoto(); err != nil {
		log.Fatal(err)
	}
}
