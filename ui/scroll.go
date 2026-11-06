package ui

import "jewel-game/rl"

// clip, when set, limits the mouse to a scrolled view, so rows scrolled out of sight can't be clicked.
var clip *rl.Rectangle

func inClip() bool {
	return clip == nil || rl.CheckCollisionPointRec(rl.GetMousePosition(), *clip)
}

// Scroll is a vertical scroll view with a draggable bar. Draw the content between Begin and
// End, offset by what Begin returns.
type Scroll struct {
	Pos, Max float32
	grab     bool
	view     rl.Rectangle
	top      float32
}

// BarWidth is the room to leave at the right of the content for the scroll bar.
const BarWidth = 16

// Begin handles the wheel and the bar, clips drawing to view, and returns the y to start the
// content at. scale is the panel scale, to clip in screen space.
func (s *Scroll) Begin(view rl.Rectangle, scale float32) float32 {
	s.view = view
	s.drag()
	if Hover(view) {
		s.Pos -= wheelPixels(rl.Wheel()) / scale
	}
	s.Pos = max(0, min(s.Pos, s.Max))
	rl.BeginScissorMode(int32(view.X*scale), int32(view.Y*scale), int32(view.Width*scale), int32(view.Height*scale))
	clip = &s.view
	s.top = view.Y - s.Pos
	return s.top
}

// End takes the y the content reached, stops clipping and draws the bar if the content overflows.
func (s *Scroll) End(bottom float32) {
	s.Max = max(0, bottom-s.top-s.view.Height)
	rl.EndScissorMode()
	clip = nil
	if s.Max <= 0 {
		return
	}
	track := s.track()
	rl.DrawRectangleRounded(track, 0.8, 6, rl.Color{R: 16, G: 14, B: 28, A: 220})
	col := Accent
	if s.grab || Hover(s.thumb(track)) {
		col = rl.Color{R: 255, G: 214, B: 120, A: 255}
	}
	rl.DrawRectangleRounded(s.thumb(track), 0.8, 6, col)
}

// Overflows is true when the content didn't fit last frame, so callers can make room for the bar.
func (s *Scroll) Overflows() bool { return s.Max > 0 }

func (s *Scroll) track() rl.Rectangle {
	v := s.view
	return rl.Rectangle{X: v.X + v.Width - 20, Y: v.Y + 4, Width: 10, Height: v.Height - 8}
}

func (s *Scroll) thumb(track rl.Rectangle) rl.Rectangle {
	total := track.Height + s.Max
	h := min(track.Height, max(28, track.Height*track.Height/total))
	y := track.Y
	if s.Max > 0 {
		y += (track.Height - h) * (s.Pos / s.Max)
	}
	return rl.Rectangle{X: track.X, Y: y, Width: track.Width, Height: h}
}

// drag moves the content when the thumb or its track is dragged.
func (s *Scroll) drag() {
	if s.Max <= 0 || !rl.IsMouseButtonDown(rl.MouseLeftButton) {
		s.grab = false
		return
	}
	track := s.track()
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && Hover(track) {
		s.grab = true
		clickTaken = true
	}
	if !s.grab {
		return
	}
	thumb := s.thumb(track)
	frac := (rl.GetMousePosition().Y - track.Y - thumb.Height/2) / (track.Height - thumb.Height)
	s.Pos = max(0, min(1, frac)) * s.Max
}

// wheelPixels turns one frame of wheel input into pixels.
// A mouse notch on the desktop build is about ±1. A trackpad, and the browser,
// already report how far the fingers moved, often as a burst of pixel deltas.
func wheelPixels(dy float32) float32 {
	if dy > 3 || dy < -3 {
		return dy
	}
	return dy * 48
}
