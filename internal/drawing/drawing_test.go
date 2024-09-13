package drawing

import (
	"image"
	"testing"
)

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestScaleImageOuterEqual(t *testing.T) {

	result := ScaleImageOuter(image.Rectangle{image.Point{0, 0}, image.Point{1920, 1080}}, image.Point{1920, 1080}, image.Point{0, 0})
	want := image.Rectangle{image.Point{0, 0}, image.Point{1920, 1080}}
	if result != want {
		t.Fatalf(`ScaleImageOuter result = %v, want %v`, result, want)
	}
}

func TestScaleImageOuterWider(t *testing.T) {
	result := ScaleImageOuter(image.Rectangle{image.Point{0, 0}, image.Point{3840, 1080}}, image.Point{1920, 1080}, image.Point{60, 0})
	// Keep height same, offset x position
	want := image.Rectangle{image.Point{-900, 0}, image.Point{2880, 1080}}
	if result != want {
		t.Fatalf(`ScaleImageOuter result = %v, want %v`, result, want)
	}
}
