package ui

import "testing"

func TestWheelPixels(t *testing.T) {
	if wheelPixels(1) != 48 || wheelPixels(-1) != -48 {
		t.Fatal("a mouse notch should move about one row")
	}
	if wheelPixels(40) != 40 || wheelPixels(-12) != -12 {
		t.Fatal("a trackpad delta is already in pixels")
	}
}
