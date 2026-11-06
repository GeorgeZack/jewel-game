package ui

import "jewel-game/rl"

// Fantasy chrome sliced from one Retro Diffusion UI-panel sheet.
// A missing texture falls back to the flat rectangles.
var (
	themeReady    bool
	themePanel    rl.Texture2D
	themeRibbon   rl.Texture2D
	themeButton   rl.Texture2D
	themeButtonOn rl.Texture2D
	themeBar      rl.Texture2D
	themeBarFill  rl.Texture2D
)

func ensureTheme() {
	if themeReady {
		return
	}
	themeReady = true
	themePanel = loadTheme("_res/ui/panel.png")
	themeRibbon = loadTheme("_res/ui/ribbon.png")
	themeButton = loadTheme("_res/ui/button.png")
	themeButtonOn = loadTheme("_res/ui/button-on.png")
	themeBar = loadTheme("_res/ui/bar.png")
	themeBarFill = loadTheme("_res/ui/bar-fill.png")
}

func loadTheme(path string) rl.Texture2D {
	tex := rl.LoadTexture(path)
	if tex.ID != 0 {
		rl.SetTextureFilter(tex, rl.FilterPoint)
	}
	return tex
}

// Frame draws the stone window with no banner. Panel adds the banner on wide windows.
func Frame(r rl.Rectangle) {
	if !stone(r, false) {
		rl.DrawRectangleRounded(r, 0.04, 8, rl.Color{R: 32, G: 28, B: 58, A: 240})
		rl.DrawRectangleRoundedLines(r, 0.04, 8, 2, PanelEdge)
	}
}

func stone(r rl.Rectangle, banner bool) bool {
	ensureTheme()
	if !nine(themePanel, r, 24, rl.White) {
		return false
	}
	// The ribbon art includes the 9px of border it hangs from, so it sits 9px above the frame.
	if banner && themeRibbon.ID != 0 && r.Width >= 480 && r.Height >= 160 {
		rw, rh := float32(themeRibbon.Width), float32(themeRibbon.Height)
		rl.DrawTexturePro(themeRibbon,
			rl.Rectangle{Width: rw, Height: rh},
			rl.Rectangle{X: r.X + r.Width - rw - 36, Y: r.Y - 9, Width: rw, Height: rh},
			rl.Vector2{}, 0, rl.White)
	}
	return true
}

// nine draws tex as a 9-slice. corner is the source-pixel size of each corner.
func nine(tex rl.Texture2D, dst rl.Rectangle, corner int, tint rl.Color) bool {
	if tex.ID == 0 || dst.Width < 4 || dst.Height < 4 {
		return false
	}
	tw, th := float32(tex.Width), float32(tex.Height)
	c := float32(corner)
	if c*2 >= tw {
		c = tw/2 - 1
	}
	if c*2 >= th {
		c = th/2 - 1
	}
	dc := c
	if edge := min(dst.Width, dst.Height) * 0.22; dc > edge {
		dc = edge
	}
	if dc*2 > dst.Width {
		dc = dst.Width / 2
	}
	if dc*2 > dst.Height {
		dc = dst.Height / 2
	}
	sx := [3]float32{0, c, tw - c}
	sw := [3]float32{c, tw - 2*c, c}
	sy := [3]float32{0, c, th - c}
	sh := [3]float32{c, th - 2*c, c}
	dx := [3]float32{dst.X, dst.X + dc, dst.X + dst.Width - dc}
	dw := [3]float32{dc, dst.Width - 2*dc, dc}
	dy := [3]float32{dst.Y, dst.Y + dc, dst.Y + dst.Height - dc}
	dh := [3]float32{dc, dst.Height - 2*dc, dc}
	for j := range 3 {
		for i := range 3 {
			if sw[i] < 1 || sh[j] < 1 || dw[i] < 1 || dh[j] < 1 {
				continue
			}
			rl.DrawTexturePro(tex,
				rl.Rectangle{X: sx[i], Y: sy[j], Width: sw[i], Height: sh[j]},
				rl.Rectangle{X: dx[i], Y: dy[j], Width: dw[i], Height: dh[j]},
				rl.Vector2{}, 0, tint)
		}
	}
	return true
}
