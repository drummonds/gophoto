package web

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"unsafe"

	"github.com/drummonds/gophoto/internal/fb"
	"golang.org/x/sys/unix"
)

// content is our static web server content.
//
//go:embed image template
var content embed.FS

// func HelloServer(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
// }

func HelloServer(w http.ResponseWriter, r *http.Request) {
	// MAIN SECTION HTML CODE
	fmt.Fprintf(w, "")
	fmt.Fprintf(w, "<h1>Hello  from gophoto</h1>")
	fmt.Fprintf(w, "<title>FrameBuffer</title>")
	fmt.Fprintf(w, "<img src='static/image/P1120981.png' alt='Chimp' style='width:800px;'>")
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
		sb.WriteString(fmt.Sprintf("<p>Error opening /dev/fb0: %v</p>", err))
		return
	}
	defer unix.Close(fd)

	if int(uintptr(fd)) != fd {
		sb.WriteString("<p>Error: fd overflows</p>")
		return
	}

	d := &fb.Device{Fd: uintptr(fd)}

	sb.WriteString("<p>Call fb.FBIOGET_FSCREENINFO to get current screen info/p>")
	_, _, eno := unix.Syscall(unix.SYS_IOCTL, d.Fd, fb.FBIOGET_FSCREENINFO, uintptr(unsafe.Pointer(&d.FInfo)))
	if eno != 0 {
		sb.WriteString(fmt.Sprintf("<p>Error getting FBIOGET_FSCREENINFO: %v</p>", eno))
		return
	}

	sb.WriteString("<ul>")
	sb.WriteString(fmt.Sprintf("<li>Start of frame buffer memory: 0x%x</li>", d.FInfo.Smem_start))
	sb.WriteString(fmt.Sprintf("<li>Length of frame buffer memory: %d bytes</li>", d.FInfo.Smem_len))
	sb.WriteString(fmt.Sprintf("<li>Frame buffer type: %d</li>", d.FInfo.Type))
	sb.WriteString(fmt.Sprintf("<li>Type-dependent flags: %d</li>", d.FInfo.Type_aux))
	sb.WriteString(fmt.Sprintf("<li>Visual: %d</li>", d.FInfo.Visual))
	sb.WriteString(fmt.Sprintf("<li>XPanStep: %d</li>", d.FInfo.Xpanstep))
	sb.WriteString(fmt.Sprintf("<li>YPanStep: %d</li>", d.FInfo.Ypanstep))
	sb.WriteString(fmt.Sprintf("<li>YWrapStep: %d</li>", d.FInfo.Ywrapstep))
	sb.WriteString(fmt.Sprintf("<li>Line length: %d bytes</li>", d.FInfo.Line_length))
	sb.WriteString(fmt.Sprintf("<li>Memory mapped I/O start: 0x%x</li>", d.FInfo.Mmio_start))
	sb.WriteString(fmt.Sprintf("<li>Memory mapped I/O length: %d bytes</li>", d.FInfo.Mmio_len))
	sb.WriteString(fmt.Sprintf("<li>Accelerator: %d</li>", d.FInfo.Accel))
	sb.WriteString(fmt.Sprintf("<li>Capabilities: 0x%x</li>", d.FInfo.Capabilities))
	sb.WriteString(fmt.Sprintf("<li>Reserved[0]: %d</li>", d.FInfo.Reserved[0]))
	sb.WriteString(fmt.Sprintf("<li>Reserved[1]: %d</li>", d.FInfo.Reserved[1]))
	sb.WriteString("</ul>")

	var format v4l2_format
	format.Type = V4L2_BUF_TYPE_VIDEO_CAPTURE

	sb.WriteString("Using https://github.com/kraxel/fbida/blob/master/fbi.c as a source of information ")
	sb.WriteString("Lookg at querying format of the frame buffer.  Although this seems to be video capture ")
	sb.WriteString("it is not a video device.  It is a frame buffer device.  It is used to display graphics in memory ")
	sb.WriteString("<p>Call fb.FBIOGET_FSCREENINFO to get current screen info/p>")
	_, _, err = unix.Syscall(unix.SYS_IOCTL, d.Fd, VIDIOC_G_FMT, uintptr(unsafe.Pointer(&format)))
	if err != nil {
		sb.WriteString(fmt.Sprintf("<p>Failed to get format: %v</p>", err))
	} else {
		sb.WriteString(fmt.Sprintf("<p>Successfully got format. Type: %d</p>", format.Type))
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
	http.ListenAndServe(port, nil)
}
