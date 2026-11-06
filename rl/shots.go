//go:build dev && !js

package rl

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Dev builds save a screenshot of the whole window every JEWEL_SHOT_EVERY seconds (default 2)
// into the folder JEWEL_SHOTS, if set. For checking layouts on a window you can't see.
var (
	shotDir   = os.Getenv("JEWEL_SHOTS")
	shotEvery = 2 * time.Second
	shotLast  time.Time
	shotN     int
)

func init() {
	if shotDir != "" {
		// a window on a hidden workspace gets no vsync, so draw on a timer instead
		ebiten.SetVsyncEnabled(false)
		ebiten.SetTPS(30)
	}
	if s, err := strconv.ParseFloat(os.Getenv("JEWEL_SHOT_EVERY"), 64); err == nil && s > 0 {
		shotEvery = time.Duration(s * float64(time.Second))
	}
}

func maybeShot(s *ebiten.Image) {
	if shotDir == "" || time.Since(shotLast) < shotEvery {
		return
	}
	shotLast = time.Now()
	shotN++
	b := s.Bounds()
	img := image.NewRGBA(b)
	s.ReadPixels(img.Pix)
	name := filepath.Join(shotDir, fmt.Sprintf("shot-%04d.png", shotN))
	go func() { // encode off the game loop so frames keep coming
		f, err := os.Create(name)
		if err != nil {
			return
		}
		defer f.Close()
		(&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(f, img)
	}()
}

// windowSize lets JEWEL_WINDOW=1280x720 pick the starting window size.
func windowSize(w, h int) (int, int) {
	var ww, hh int
	if _, err := fmt.Sscanf(os.Getenv("JEWEL_WINDOW"), "%dx%d", &ww, &hh); err == nil && ww > 0 && hh > 0 {
		return ww, hh
	}
	return w, h
}
