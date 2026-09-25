package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"unsafe"

	"git.bytestone.uk/hum3/gophoto/internal/fb"
	"golang.org/x/sys/unix"
)

// content is our static web server content.
//
//go:embed image template
var content embed.FS

// func HelloServer(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
// }

const helloPage = "<title>FrameBuffer</title><h1>Hello  from gophoto</h1>" +
	"<img src='static/image/P1120981.png' alt='Chimp' style='width:800px;'>"

func HelloServer(w http.ResponseWriter, r *http.Request) {
	if _, err := io.WriteString(w, helloPage); err != nil {
		log.Printf("hello page: %v", err)
	}
}

type Page struct {
	Title string
	Body  template.HTML
}

// Define the VIDIOC_G_FMT ioctl request code
const VIDIOC_G_FMT = 0xc0d05604

// Define V4L2_BUF_TYPE_VIDEO_CAPTURE
const V4L2_BUF_TYPE_VIDEO_CAPTURE uint32 = 1

// Define a struct to hold the format information
type v4l2_format struct {
	Type uint32
	Pad  [4]byte
	Fmt  [200]byte
}

// Get the frame buffer and add info to the string builder
func addFrameBufferInfo(sb *strings.Builder) {
	sb.WriteString("<hr><h2>Starting to get frame buffer Info </h2>")
	defer sb.WriteString("<p>Completed get frame buffer Info </p><hr>")

	sb.WriteString("<p>/dev/fb0 unix.Open</p>")
	fd, err := unix.Open("/dev/fb0", unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		fmt.Fprintf(sb, "<p>Error opening /dev/fb0: %v</p>", err)
		return
	}
	defer func() {
		if err := unix.Close(fd); err != nil {
			fmt.Fprintf(sb, "<p>Error closing /dev/fb0: %v</p>", err)
		}
	}()

	if int(uintptr(fd)) != fd {
		sb.WriteString("<p>Error: fd overflows</p>")
		return
	}

	d := &fb.Device{Fd: uintptr(fd)}

	sb.WriteString("<p>Call fb.FBIOGET_FSCREENINFO to get current screen info/p>")
	_, _, eno := unix.Syscall(unix.SYS_IOCTL, d.Fd, fb.FBIOGET_FSCREENINFO, uintptr(unsafe.Pointer(&d.FInfo)))
	if eno != 0 {
		fmt.Fprintf(sb, "<p>Error getting FBIOGET_FSCREENINFO: %v</p>", eno)
		return
	}

	sb.WriteString("<ul>")
	fmt.Fprintf(sb, "<li>Start of frame buffer memory: 0x%x</li>", d.FInfo.Smem_start)
	fmt.Fprintf(sb, "<li>Length of frame buffer memory: %d bytes</li>", d.FInfo.Smem_len)
	fmt.Fprintf(sb, "<li>Frame buffer type: %d</li>", d.FInfo.Type)
	fmt.Fprintf(sb, "<li>Type-dependent flags: %d</li>", d.FInfo.Type_aux)
	fmt.Fprintf(sb, "<li>Visual: %d</li>", d.FInfo.Visual)
	fmt.Fprintf(sb, "<li>XPanStep: %d</li>", d.FInfo.Xpanstep)
	fmt.Fprintf(sb, "<li>YPanStep: %d</li>", d.FInfo.Ypanstep)
	fmt.Fprintf(sb, "<li>YWrapStep: %d</li>", d.FInfo.Ywrapstep)
	fmt.Fprintf(sb, "<li>Line length: %d bytes</li>", d.FInfo.Line_length)
	fmt.Fprintf(sb, "<li>Memory mapped I/O start: 0x%x</li>", d.FInfo.Mmio_start)
	fmt.Fprintf(sb, "<li>Memory mapped I/O length: %d bytes</li>", d.FInfo.Mmio_len)
	fmt.Fprintf(sb, "<li>Accelerator: %d</li>", d.FInfo.Accel)
	fmt.Fprintf(sb, "<li>Capabilities: 0x%x</li>", d.FInfo.Capabilities)
	fmt.Fprintf(sb, "<li>Reserved[0]: %d</li>", d.FInfo.Reserved[0])
	fmt.Fprintf(sb, "<li>Reserved[1]: %d</li>", d.FInfo.Reserved[1])
	sb.WriteString("</ul>")

	var format v4l2_format
	format.Type = V4L2_BUF_TYPE_VIDEO_CAPTURE

	sb.WriteString("Using https://github.com/kraxel/fbida/blob/master/fbi.c as a source of information ")
	sb.WriteString("Lookg at querying format of the frame buffer.  Although this seems to be video capture ")
	sb.WriteString("it is not a video device.  It is a frame buffer device.  It is used to display graphics in memory ")
	sb.WriteString("<p>Call fb.FBIOGET_FSCREENINFO to get current screen info/p>")
	_, _, eno = unix.Syscall(unix.SYS_IOCTL, d.Fd, VIDIOC_G_FMT, uintptr(unsafe.Pointer(&format)))
	if eno != 0 {
		fmt.Fprintf(sb, "<p>Failed to get format: %v</p>", eno)
	} else {
		fmt.Fprintf(sb, "<p>Successfully got format. Type: %d</p>", format.Type)
	}
}

func diagHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(content, "template/base.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := &Page{Title: "FrameBuffer"}
	var sb strings.Builder
	sb.WriteString("<h1>Hello  from gophoto</h1>")
	addFrameBufferInfo(&sb)
	sb.WriteString("<title>FrameBuffer</title>")
	sb.WriteString("<img src='static/image/P1120981.png' alt='Chimp' style='width:800px;'>")
	page.Body = template.HTML(sb.String())

	err = tmpl.Execute(w, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func StartWebServer() {
	port := ":8080"
	log.Printf("Starting web server on port %s", port)

	http.HandleFunc("/", HelloServer)
	http.HandleFunc("/diag", diagHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(content))))
	// http.HandleFunc("/", index_handler)
	// http.HandleFunc("/about/", about_handler)
	// http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	log.Fatal(http.ListenAndServe(port, nil))
}
