//go:build dev && !js

package rl

import (
	"fmt"
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
	f, err := os.Create(filepath.Join(shotDir, fmt.Sprintf("shot-%03d.png", shotN)))
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, s)
}

// windowSize lets JEWEL_WINDOW=1280x720 pick the starting window size.
func windowSize(w, h int) (int, int) {
	var ww, hh int
	if _, err := fmt.Sscanf(os.Getenv("JEWEL_WINDOW"), "%dx%d", &ww, &hh); err == nil && ww > 0 && hh > 0 {
		return ww, hh
	}
	return w, h
}
