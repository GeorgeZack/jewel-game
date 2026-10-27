package core

import (
	"fmt"
	"jewel-game/board"
	"jewel-game/entities"
	"jewel-game/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Preset struct {
	Name		string
	Settings	board.Settings
	Speed		float32
}

func Presets() []Preset {
	chill := board.DefaultSettings()
	chill.NumColors = 5
	chill.MaxLocks = -1
	chill.FreeSwap = true
	chill.DiagonalSwap = true
	chill.SpecialRate = 0.05

	playground := board.DefaultSettings()
	playground.MaxLocks = -1

	insane := board.DefaultSettings()
	insane.Width, insane.Height = 10, 10
	insane.MaxLocks = -1
	insane.FreeSwap = true
	insane.DiagonalSwap = true
	insane.Overcharge = true
	insane.ChaosGravity = true
	insane.SpecialRate = 0.25

	return []Preset{
		{ Name: "Chill", Settings: chill, Speed: 1 },
		{ Name: "Playground", Settings: playground, Speed: 1 },
		{ Name: "Insane", Settings: insane, Speed: 1.5 },
	}
}

var speeds = []float32{ 0.25, 0.5, 1, 1.5, 2, 3 }

const MAX_LOCK_SETTING = 20

// DrawSandboxPanel draws the sandbox controls and applies whatever the player changes.
func DrawSandboxPanel(tm *TileManager, dm *DataManager, r rl.Rectangle) {
	ui.Panel(r)
	col := ui.Column{ X: r.X + 18, Y: r.Y + 16, W: r.Width - 36, Gap: 6 }
	s := tm.Board.Settings
	reset := false

	col.Heading("Presets")
	presets := Presets()
	for i, b := range ui.Row(col.Next(34), len(presets), 8) {
		if ui.Button(b, presets[i].Name, false) {
			s, tm.Speed, reset = presets[i].Settings, presets[i].Speed, true
		}
	}

	col.Heading("Board (starts a new one)")
	if d := ui.Stepper(col.Next(30), "Width", fmt.Sprint(s.Width)); d != 0 {
		s.Width, reset = clampInt(s.Width+d, 4, 16), true
	}
	if d := ui.Stepper(col.Next(30), "Height", fmt.Sprint(s.Height)); d != 0 {
		s.Height, reset = clampInt(s.Height+d, 4, 16), true
	}
	maxColors := max(3, len(dm.GemTypeOrder))
	if d := ui.Stepper(col.Next(30), "Colors", fmt.Sprint(s.NumColors)); d != 0 {
		s.NumColors, reset = clampInt(s.NumColors+d, 3, maxColors), true
	}

	col.Heading("Rules")
	if d := ui.Stepper(col.Next(30), "Locks", lockLabel(s.MaxLocks)); d != 0 {
		s.MaxLocks = stepLocks(s.MaxLocks, d)
	}
	s.FreeSwap = ui.Toggle(col.Next(30), "Free swaps", s.FreeSwap)
	s.DiagonalSwap = ui.Toggle(col.Next(30), "Diagonal swaps", s.DiagonalSwap)
	s.Overcharge = ui.Toggle(col.Next(30), "Overcharged specials", s.Overcharge)
	s.ChaosGravity = ui.Toggle(col.Next(30), "Chaos gravity", s.ChaosGravity)
	if d := ui.Stepper(col.Next(30), "Gravity", s.Gravity.String()); d != 0 {
		s.Gravity = board.Direction((int(s.Gravity) + d + 4) % 4)
	}
	rain := int(s.SpecialRate*100 + 0.5)
	if d := ui.Stepper(col.Next(30), "Special rain", fmt.Sprintf("%d%%", rain)); d != 0 {
		s.SpecialRate = float64(clampInt(rain+d*5, 0, 50)) / 100
	}
	if d := ui.Stepper(col.Next(30), "Anim speed", fmt.Sprintf("%gx", tm.Speed)); d != 0 {
		tm.Speed = stepSpeed(tm.Speed, d)
	}

	col.Heading("Paint brush")
	col.Line("Pick one, then click or drag on gems.", 15, ui.DimText)
	col.Line("Esc puts the brush down.", 15, ui.DimText)
	drawBrushPicker(tm, dm, &col, s.NumColors)

	if reset {
		tm.Reset(s)
	} else {
		tm.Board.Settings = s
	}
}

func drawBrushPicker(tm *TileManager, dm *DataManager, col *ui.Column, numColors int) {
	types := dm.GetAllGemTypes()
	n := min(numColors, len(types))
	swatches := ui.Row(col.Next(40), n+1, 4)
	for i := 0; i < n; i++ {
		on := tm.Brush != nil && tm.Brush.Special != board.Prism && tm.Brush.Color == i
		if ui.Button(swatches[i], "", on) {
			special := board.None
			if tm.Brush != nil && tm.Brush.Special != board.Prism {
				special = tm.Brush.Special
			}
			tm.Brush = &Brush{ Color: i, Special: special }
		}
		r := swatches[i]
		entities.DrawGem(entities.GemLook{
			Type:	types[i],
			Gem:	board.Gem{ Color: i },
			Center:	rl.Vector2{ X: r.X + r.Width/2, Y: r.Y + r.Height/2 },
			Size:	min(r.Width, r.Height),
			Scale:	1,
			Alpha:	1,
			Time:	tm.time,
		})
	}
	if ui.Button(swatches[n], "Off", tm.Brush == nil) {
		tm.Brush = nil
	}

	specials := []struct {
		label	string
		special	board.Special
	}{
		{ "Plain", board.None },
		{ "Bolt-", board.BoltH },
		{ "Bolt|", board.BoltV },
		{ "Bomb", board.Bomb },
		{ "Prism", board.Prism },
	}
	for i, b := range ui.Row(col.Next(32), len(specials), 4) {
		sp := specials[i]
		if ui.Button(b, sp.label, tm.Brush != nil && tm.Brush.Special == sp.special) {
			if tm.Brush == nil {
				tm.Brush = &Brush{}
			}
			tm.Brush.Special = sp.special
		}
	}
}

func lockLabel(n int) string {
	switch n {
	case -1:
		return "unlimited"
	case 0:
		return "off"
	}
	return fmt.Sprint(n)
}

// stepLocks walks off, 1, 2 ... MAX_LOCK_SETTING, unlimited, and wraps around.
func stepLocks(n, d int) int {
	if n == -1 {
		n = MAX_LOCK_SETTING + 1
	}
	n = (n + d + MAX_LOCK_SETTING + 2) % (MAX_LOCK_SETTING + 2)
	if n == MAX_LOCK_SETTING+1 {
		return -1
	}
	return n
}

func stepSpeed(cur float32, d int) float32 {
	i := 2
	for j, s := range speeds {
		if s == cur {
			i = j
		}
	}
	return speeds[clampInt(i+d, 0, len(speeds)-1)]
}

func clampInt(n, lo, hi int) int {
	return max(lo, min(hi, n))
}
