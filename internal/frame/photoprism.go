// Perhaps use MapFS for storing temp files
package frame

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/disintegration/gift"
	"github.com/drummonds/gophoto/internal/drawing"
	"github.com/drummonds/gophoto/internal/panel"
	"github.com/drummonds/photoprism-go-api/api"
	"golang.org/x/image/draw"
)

func GetClient() (*api.ClientWithResponses, error) {
	host := os.Getenv("PHOTOPRISM_DOMAIN")
	token := os.Getenv("PHOTOPRISM_TOKEN")
	provider := api.NewXAuthProvider(token)

	nc, err := api.NewClientWithResponses(host, api.WithRequestEditorFn(provider.Intercept))
	return nc, err
}

// Scale and image to centre and fit or fill
func ScaleImage(img image.Image, dstBounds image.Rectangle, fit bool) image.Image {
	// Calculate scaling factors
	srcBounds := img.Bounds()
	windowWidth := dstBounds.Dx()
	windowHeight := dstBounds.Dy()
	srcAspect := float64(srcBounds.Dx()) / float64(srcBounds.Dy())
	dstAspect := float64(windowWidth) / float64(windowHeight)

	var scaledWidth, scaledHeight int
	if fit {
		if srcAspect > dstAspect {
			scaledWidth = windowWidth
			scaledHeight = int(float64(windowWidth) / srcAspect)
		} else {
			scaledHeight = windowHeight
			scaledWidth = int(float64(windowHeight) * srcAspect)
		}
	} else {
		if srcAspect < dstAspect {
			scaledWidth = windowWidth
			scaledHeight = int(float64(windowWidth) / srcAspect)
		} else {
			scaledHeight = windowHeight
			scaledWidth = int(float64(windowHeight) * srcAspect)
		}
	}
	// Create a new RGBA image with the window dimensions
	scaled := image.NewRGBA(image.Rect(0, 0, windowWidth, windowHeight))

	// Calculate offset for centering
	offsetX := (windowWidth - scaledWidth) / 2
	offsetY := (windowHeight - scaledHeight) / 2

	// Scale and center the image
	draw.CatmullRom.Scale(scaled,
		image.Rect(offsetX, offsetY, offsetX+scaledWidth, offsetY+scaledHeight),
		img, srcBounds, draw.Over, nil)

	return scaled
}

func NewImage(ctx context.Context, bounds image.Rectangle) (image.Image, error) {
	log.Printf("Start newImage get and wait 3 sec")
	time.Sleep(3 * time.Second)
	rawImg, err := GetImage(ctx)
	if err != nil {
		return rawImg, err
	}
	log.Printf("got raw image")
	// handle scaling to mock frame buffer
	img := ScaleImage(rawImg, bounds, false)
	log.Printf("Scaled image")
	return img, err
}

// Search for first album
// then search for first 10 pictures in that album
// then retrun that as a list
func GetPhotoList(ctx context.Context) ([]string, error) {
	var albumUid string

	albumUid = os.Getenv("ALBUM_UID")

	// Get photos from album
	photoParams := api.SearchPhotosParams{Count: 20, S: &albumUid}
	photos, err := GlobalPage.Client.SearchPhotosWithResponse(ctx, &photoParams)
	if err != nil {
		return []string{}, err
	}
	if photos.HTTPResponse.StatusCode != 200 {
		return []string{}, fmt.Errorf("Problem with status %v\n", photos.HTTPResponse.StatusCode)
	}
	if len(*photos.JSON200) < 1 {
		return []string{}, fmt.Errorf("no photos to show")
	}
	photoList := make([]string, 0, len(*photos.JSON200))
	for _, photo := range *photos.JSON200 {
		log.Printf("Original name %s\n", *photo.OriginalName)
		photoList = append(photoList, *photo.UID)
	}
	return photoList, nil
}

func firstJpeg(files []api.EntityFile) api.EntityFile {
	for _, file := range files {
		if *file.Mime == "image/jpeg" {
			return file
		}
	}
	return api.EntityFile{}
}

// Returns a raw image, orientated correctly but not scaled
func GetImage(ctx context.Context) (image.Image, error) {
	var (
		body        []byte
		orientation int
		blank       image.Image
	)
	log.Printf("GetImage")
	// Get photo Id
	uid := GlobalPhotoList[GlobalPage.PhotoIndex]
	log.Printf("GetImage uid = %s, index = %v", uid, GlobalPage.PhotoIndex)
	// Get details by search
	// SearchPhotosWithResponse(ctx context.Context, params *SearchPhotosParams, reqEditors ...RequestEditorFn) (*SearchPhotosResponse, error)

	// Get details of photo but hash doesn't seem to work
	photo, err := GlobalPage.Client.GetPhotoWithResponse(ctx, uid)
	if err != nil {
		return blank, err
	}
	log.Printf("Got photo")
	files := photo.JSON200.Files
	log.Printf("Got files %v", len(*files))
	if len(*files) > 0 {
		fileEntity := firstJpeg(*files)
		orientation = *fileEntity.Orientation
		switch {
		case true: // Download raw file
			// now get actual data
			hash := *fileEntity.Hash
			// hash := GlobalPhotoList[GlobalPage.PhotoIndex]
			log.Printf("Get download")

			file, err := GlobalPage.Client.GetDownloadWithResponse(ctx, hash)
			if err != nil {
				return blank, err
			}
			status := file.HTTPResponse.StatusCode
			if status != 200 {
				return blank, fmt.Errorf("Problem with status downloading file %v\n", file.HTTPResponse.StatusCode)
			}
			body = file.Body
		case true: // Download thumbnail
			hash := *fileEntity.Hash
			token := os.Getenv("PHOTOPRISM_TOKEN")
			file, err := GlobalPage.Client.GetThumbWithResponse(ctx, hash, token, "tile_500")
			if err != nil {
				return blank, err
			}
			status := file.HTTPResponse.StatusCode
			if status != 200 {
				return blank, fmt.Errorf("Problem with status downloading file %v\n", file.HTTPResponse.StatusCode)
			}
			body = file.Body
		}
	}
	// Decode the test image using the imageorient.Decode function
	// to handle the image orientation correctly.
	// rawImg, _, err = imageorient.Decode(bytes.NewReader(body))
	// if err != nil {
	// 	log.Fatalf("imageorient.Decode failed: %v", err)
	// }
	log.Printf("Got download")
	rawImg, err := jpeg.Decode(bytes.NewReader(body))
	if err != nil {
		return blank, fmt.Errorf("error decoding JPEG: %v", err)
	}
	log.Printf("Decoded download")

	// Apply orientation based on the EXIF data
	g := gift.New()
	switch orientation {
	case 2:
		g.Add(gift.FlipHorizontal())
	case 3:
		g.Add(gift.Rotate180())
	case 4:
		g.Add(gift.FlipVertical())
	case 5:
		g.Add(gift.Rotate270())
		g.Add(gift.FlipHorizontal())
	case 6:
		g.Add(gift.Rotate270())
	case 7:
		g.Add(gift.Rotate90())
		g.Add(gift.FlipHorizontal())
	case 8:
		g.Add(gift.Rotate90())
	}
	log.Printf("Orientate")

	// Apply the orientation
	oriented := image.NewRGBA(g.Bounds(rawImg.Bounds()))
	log.Printf("Orientated")
	g.Draw(oriented, rawImg)
	log.Printf("Drawn")

	// Use 'oriented' for further processing
	return oriented, nil
}

type Page struct {
	Title      string
	ImageName  string
	Body       []byte
	Clock      string
	PhotoIndex int
	Client     *api.ClientWithResponses
}

var (
	GlobalPage      = Page{Title: "Album show", ImageName: "TBC"}
	GlobalPhotoList []string
)

func imageHandler(w http.ResponseWriter, r *http.Request) {

}

// Setup pictures to pull
func NewPhotoPrism(ctx context.Context) (err error) {
	log.Printf("Get client %s\n", time.Now().Format(time.RFC3339))
	GlobalPage.Client, err = GetClient()
	if err != nil {
		return err
	}
	log.Printf("Get photolist %s\n", time.Now().Format(time.RFC3339))
	GlobalPhotoList, err = GetPhotoList(ctx)
	log.Printf("Got photolist %s, err = %+v\n", time.Now().Format(time.RFC3339), err)
	return err
}

// Setup pictures to pull
func (pf *PictureFrame) SetupFullPhotoPrism() (err error) {
	fmt.Println(("Demo of showing pictures from an album"))
	ctx := context.Background()
	err = NewPhotoPrism(ctx)
	log.Printf("Done setup %s\n", time.Now().Format(time.RFC3339))
	return err
}

// Get latest picture and render
func (pf *PictureFrame) RenderPhotoPrism() error {
	fileToBeUploaded := "temp.jpg"
	f, err := os.Open(fileToBeUploaded)
	if err != nil {
		return err
	}
	defer f.Close()

	// fileInfo, _ := f.Stat()
	// var size int64 = fileInfo.Size()
	displayPhoto, imageFormat, err := image.Decode(f)
	if err != nil {
		log.Printf("can't decode photo image %s: |%+v|\n", imageFormat, err)
		displayPhoto, _, err = image.Decode(bytes.NewReader(displayPhotoPNG))
		if err != nil {
			return fmt.Errorf("can't decode embedded photo image %s: %v", imageFormat, err)
		}
	}
	GlobalPage.PhotoIndex = (GlobalPage.PhotoIndex + 1) % len(GlobalPhotoList)
	picture := panel.NewImagePanel(displayPhoto)
	photoRect := drawing.ScaleImageOuter(displayPhoto.Bounds(), pf.Bounds.Size(), image.Point{0, 0})
	// Now move image to center
	picture.Location = photoRect
	pf.AddPanel(picture)
	log.Printf("Picture location - %v", picture.Location)
	return nil
}
