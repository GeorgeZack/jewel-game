//go:build !dev || js

package rl

import "github.com/hajimehoshi/ebiten/v2"

func maybeShot(*ebiten.Image) {}

func windowSize(w, h int) (int, int) { return w, h }
