// Perhaps use MapFS for storing temp files
package frame

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"

	"github.com/drummonds/gophoto/internal/drawing"
	"github.com/drummonds/gophoto/internal/panel"
	"github.com/drummonds/photoprism-go-api/api"
)

func getClient() *api.ClientWithResponses {
	host := os.Getenv("PHOTOPRISM_DOMAIN")
	token := os.Getenv("PHOTOPRISM_TOKEN")
	provider := api.NewXAuthProvider(token)

	nc, err := api.NewClientWithResponses(host, api.WithRequestEditorFn(provider.Intercept))
	if err != nil {
		panic(err)
	}
	return nc
}

// Search for first album
// then search for first 10 pictures in that album
// then return that as a list
func getPhotoList(ctx context.Context) []string {
	albumUid := os.Getenv("ALBUM_UID")

	// Get photos from album
	photoParams := api.SearchPhotosParams{Count: 1000, S: &albumUid}
	photos, err := GlobalPage.Client.SearchPhotosWithResponse(ctx, &photoParams)
	if err != nil {
		panic(err)
	}
	if photos.HTTPResponse.StatusCode != 200 {
		panic(fmt.Errorf("Problem with status %v\n", photos.HTTPResponse.StatusCode))
	}
	if len(*photos.JSON200) < 1 {
		panic("no photos to show")
	}
	photoList := make([]string, 0, 10)
	for _, photo := range *photos.JSON200 {
		fmt.Printf("%+v\n", photo.OriginalName)
		photoList = append(photoList, *photo.UID)
	}
	return photoList
}

// Downloads global image as temp.jpg
func getImage(ctx context.Context) (b []byte, err error) {
	// Get album Id
	uid := GlobalPhotoList[GlobalPage.PhotoIndex]
	// Get details by search
	// SearchPhotosWithResponse(ctx context.Context, params *SearchPhotosParams, reqEditors ...RequestEditorFn) (*SearchPhotosResponse, error)

	// Get details of photo but hash doesn't seem to work
	photo, err := GlobalPage.Client.GetPhotoWithResponse(ctx, uid)
	if err != nil {
		return b, err
	}
	files := photo.JSON200.Files
	if len(*files) > 0 {
		fileEntity := (*files)[0]
		switch {
		case true: // Download raw file
			// now get actual data
			hash := *fileEntity.Hash
			// hash := GlobalPhotoList[GlobalPage.PhotoIndex]

			file, err := GlobalPage.Client.GetDownloadWithResponse(ctx, hash)
			if err != nil {
				return b, err
			}
			status := file.HTTPResponse.StatusCode
			if status != 200 {
				return b, fmt.Errorf("Problem with status downloading file %v\n", file.HTTPResponse.StatusCode)
			}
			return file.Body, nil
		case true: // Download thumbnail
			hash := *fileEntity.Hash
			token := os.Getenv("PHOTOPRISM_TOKEN")
			file, err := GlobalPage.Client.GetThumbWithResponse(ctx, hash, token, "fit_3840")
			if err != nil {
				return b, err
			}
			status := file.HTTPResponse.StatusCode
			if status != 200 {
				return b, fmt.Errorf("Problem with status downloading file %v\n", file.HTTPResponse.StatusCode)
			}
			return file.Body, nil
		}
	}
	return b, fmt.Errorf("no image files found for album")
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
func (pf *PictureFrame) SetupFullPhotoPrism() {
	fmt.Println(("Demo of showing pictures from an album"))
	GlobalPage.Client = getClient()
	ctx := context.Background()
	GlobalPhotoList = getPhotoList(ctx)
}

// Get latest picture and render
func (pf *PictureFrame) RenderPhotoPrism() {
	ctx := context.Background()
	// pf.picture.Render(pf.buffer, photoRect)

	// http.ServeFile(w, r, "temp.jpg")
	// get the main image
	thisImage, err := getImage(ctx)
	if err != nil {
		panic(fmt.Errorf("can't get find image: %v", err))
	}
	// panics if this image is attempted to decode
	displayPhoto, imageFormat, err := image.Decode(bytes.NewReader(thisImage))
	if err != nil {
		fmt.Printf("can't decode photo image %s: |%+v|", imageFormat, err)
		displayPhoto, _, err = image.Decode(bytes.NewReader(displayPhotoPNG))
		if err != nil {
			panic(fmt.Errorf("can't decode embedded photo image %s: %v", imageFormat, err))
		}
	}
	GlobalPage.PhotoIndex = (GlobalPage.PhotoIndex + 1) % len(GlobalPhotoList)
	picture := panel.NewImagePanel(displayPhoto)
	photoRect := drawing.ScaleImageOuter(displayPhoto.Bounds(), pf.Bounds.Size(), image.Point{0, 0})
	// Now move image to center
	picture.Location = photoRect
	pf.AddPanel(picture)
	log.Printf("Picture location - %v", picture.Location)

}
