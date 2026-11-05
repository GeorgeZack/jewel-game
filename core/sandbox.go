package core

import (
	"fmt"
	"jewel-game/board"
	"jewel-game/entities"
	"jewel-game/text"
	"jewel-game/ui"

	"jewel-game/rl"
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
		{ Name: text.T("ui.preset_chill", "Chill"), Settings: chill, Speed: 1 },
		{ Name: text.T("ui.preset_playground", "Playground"), Settings: playground, Speed: 1 },
		{ Name: text.T("ui.preset_insane", "Insane"), Settings: insane, Speed: 1.5 },
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

	col.Heading(text.T("ui.presets", "Presets"))
	presets := Presets()
	for i, b := range ui.Row(col.Next(34), len(presets), 8) {
		if ui.Button(b, presets[i].Name, false) {
			s, tm.Speed, reset = presets[i].Settings, presets[i].Speed, true
		}
	}

	col.Heading(text.T("ui.board_heading", "Board (starts a new one)"))
	if d := ui.Stepper(col.Next(30), text.T("ui.width", "Width"), fmt.Sprint(s.Width)); d != 0 {
		s.Width, reset = clampInt(s.Width+d, 4, 16), true
	}
	if d := ui.Stepper(col.Next(30), text.T("ui.height", "Height"), fmt.Sprint(s.Height)); d != 0 {
		s.Height, reset = clampInt(s.Height+d, 4, 16), true
	}
	maxColors := max(3, len(dm.GemTypeOrder))
	if d := ui.Stepper(col.Next(30), text.T("ui.colors", "Colors"), fmt.Sprint(s.NumColors)); d != 0 {
		s.NumColors, reset = clampInt(s.NumColors+d, 3, maxColors), true
	}

	col.Heading(text.T("ui.rules", "Rules"))
	if d := ui.Stepper(col.Next(30), text.T("ui.locks_setting", "Locks"), lockLabel(s.MaxLocks)); d != 0 {
		s.MaxLocks = stepLocks(s.MaxLocks, d)
	}
	s.FreeSwap = ui.Toggle(col.Next(30), text.T("ui.free_swaps", "Free swaps"), s.FreeSwap)
	s.DiagonalSwap = ui.Toggle(col.Next(30), text.T("ui.diagonal_swaps", "Diagonal swaps"), s.DiagonalSwap)
	s.Overcharge = ui.Toggle(col.Next(30), text.T("ui.overcharged", "Overcharged specials"), s.Overcharge)
	s.ChaosGravity = ui.Toggle(col.Next(30), text.T("ui.chaos_gravity", "Chaos gravity"), s.ChaosGravity)
	if d := ui.Stepper(col.Next(30), text.T("ui.gravity_setting", "Gravity"), trDir(s.Gravity)); d != 0 {
		s.Gravity = board.Direction((int(s.Gravity) + d + 4) % 4)
	}
	rain := int(s.SpecialRate*100 + 0.5)
	if d := ui.Stepper(col.Next(30), text.T("ui.special_rain", "Special rain"), text.Tf("ui.percent", "%d%%", rain)); d != 0 {
		s.SpecialRate = float64(clampInt(rain+d*5, 0, 50)) / 100
	}
	if d := ui.Stepper(col.Next(30), text.T("ui.anim_speed", "Anim speed"), text.Tf("ui.speed", "%gx", tm.Speed)); d != 0 {
		tm.Speed = stepSpeed(tm.Speed, d)
	}

	col.Heading(text.T("ui.brush", "Paint brush"))
	col.Line(text.T("ui.brush_pick", "Pick one, then click or drag on gems."), 15, ui.DimText)
	col.Line(text.T("ui.brush_esc", "Esc puts the brush down."), 15, ui.DimText)
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
	if ui.Button(swatches[n], text.T("ui.brush_off", "Off"), tm.Brush == nil) {
		tm.Brush = nil
	}

	specials := []struct {
		label	string
		special	board.Special
	}{
		{ text.T("ui.brush_plain", "Plain"), board.None },
		{ text.T("special.bolt", "Bolt") + "-", board.BoltH },
		{ text.T("special.bolt", "Bolt") + "|", board.BoltV },
		{ text.T("special.bomb", "Bomb"), board.Bomb },
		{ text.T("special.prism", "Prism"), board.Prism },
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
		return text.T("ui.locks_unlimited", "unlimited")
	case 0:
		return text.T("ui.locks_off", "off")
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
