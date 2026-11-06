package core

import (
	"jewel-game/rl"
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"
)

// titleScreen is shown at launch, before the game's panels.
type titleScreen struct {
	on      bool
	help    bool // showing How to play
	confirm bool // asking before New game replaces a saved hero
}

// drawTitle draws the title screen in panel space, w by h, and handles its buttons.
func (c *GameController) drawTitle(w, h float32) {
	t := &c.title
	q := c.Quest
	box := rl.Rectangle{X: w/2 - 360, Y: max(20, h/2-270), Width: 720, Height: min(540, h-40)}
	ui.Panel(box)
	col := ui.Column{X: box.X + 50, Y: box.Y + 36, W: box.Width - 100, Gap: 10}
	if t.help {
		c.drawHowToPlay(&col, box)
		return
	}

	head := col.Next(64)
	ui.TextCentered(rpg.Story.Title, head.X+head.Width/2, head.Y, 56, ui.Accent)
	sub := col.Next(28)
	ui.TextCentered(text.T("ui.title_sub", "A match-3 adventure"), sub.X+sub.Width/2, sub.Y, 22, ui.DimText)
	col.Y += 12
	col.Paragraph(rpg.Story.Blurb, 20, ui.TextColor)
	col.Y += 18

	button := func(label string) bool {
		r := col.Next(50)
		r.X, r.Width = r.X+110, r.Width-220
		return ui.Button(r, label, false)
	}
	switch {
	case t.confirm:
		col.Paragraph(text.T("ui.title_confirm", "Start over? This deletes your current hero for good."), 20, ui.Accent)
		// Keep comes first, where New game was, so a double click can't delete the hero
		if button(text.T("ui.keep_hero", "Keep my hero")) {
			t.confirm = false
		}
		if button(text.T("ui.delete_hero", "Delete hero")) {
			if err := rpg.DeleteSave(rpg.SavePath()); err != nil {
				ui.AddMessage("%s", text.Tf("log.save_delete", "Could not delete save: %v", err))
			}
			q.Hero = nil
			q.scenes.Stop()
			q.Screen = ScreenClass
			t.on, t.confirm = false, false
		}
	default:
		if q.Hero != nil {
			label := text.Tf("ui.continue_hero", "Continue  (Level %d %s)", q.Hero.Level, rpg.ClassByName(q.Hero.Class).Label())
			if button(label) || rl.IsKeyPressed(rl.KeyEnter) {
				t.on = false
			}
		}
		if button(text.T("ui.new_game", "New game")) {
			if q.Hero != nil {
				t.confirm = true
			} else {
				q.Screen = ScreenClass
				t.on = false
			}
		}
		if button(text.T("ui.how_to_play", "How to play")) {
			t.help = true
		}
	}

	foot := box.Y + box.Height - 40
	ui.TextCentered(text.T("ui.title_foot", "Vertical slice. Some art is placeholder."), box.X+box.Width/2, foot, 16, ui.DimText)
}

func (c *GameController) drawHowToPlay(col *ui.Column, box rl.Rectangle) {
	head := col.Next(50)
	ui.TextCentered(text.T("ui.how_to_play", "How to play"), head.X+head.Width/2, head.Y, 40, ui.Accent)
	back := rl.Rectangle{X: box.X + box.Width/2 - 100, Y: box.Y + box.Height - 70, Width: 200, Height: 46}
	view := rl.Rectangle{X: box.X + 20, Y: col.Y, Width: box.Width - 40, Height: back.Y - 12 - col.Y}
	col.Y = c.helpScroll.Begin(view, uiScale())
	para := func(s string) {
		col.Paragraph(s, 18, ui.TextColor)
		col.Y += 4
	}
	col.Heading(text.T("ui.how_duels", "How duels work"))
	para(text.T("ui.duel_1", "You and your foe take turns on one board."))
	para(text.T("ui.duel_2", "Colored gems fill your mana, for spells."))
	para(text.T("ui.duel_3", "Skulls hurt the foe. +5 skulls hit hard and blast the gems around them."))
	para(text.T("ui.duel_4", "Gold and stars pay out after the duel."))
	para(text.T("ui.duel_5", "Match 4 for an extra turn. Match 5 for a turn and a Wild, which completes any mana match and multiplies it."))
	col.Heading(text.T("ui.specials", "Specials"))
	para(text.T("ui.help_bolt", "4 in a row makes a Bolt. It clears its whole row or column."))
	para(text.T("ui.help_bomb", "An L or T shape makes a Bomb. It blasts the 3x3 around it."))
	para(text.T("ui.help_prism", "5 in a row makes a Prism. Swap it with any gem to clear that color."))
	col.Heading(text.T("ui.combos", "Combos"))
	para(text.T("ui.help_combo_intro", "Swap two specials into each other:"))
	para(text.T("ui.help_bolt_bolt", "Bolt + Bolt: a full cross."))
	para(text.T("ui.help_bolt_bomb", "Bolt + Bomb: a cross three lines wide."))
	para(text.T("ui.help_bomb_bomb", "Bomb + Bomb: a 5x5 blast."))
	para(text.T("ui.help_prism_mix", "Prism + Bolt/Bomb: every gem of that color turns into one and goes off."))
	para(text.T("ui.help_prism_prism", "Prism + Prism: clears the board."))
	col.Heading(text.T("ui.controls", "Controls"))
	para(text.T("ui.help_controls", "Drag a gem, or click two gems, to swap them. Keys 1-6 cast spells, H shows a hint."))
	c.helpScroll.End(col.Y)
	if ui.Button(back, text.T("ui.back", "Back"), false) || rl.IsKeyPressed(rl.KeyEscape) {
		c.title.help = false
	}
}
