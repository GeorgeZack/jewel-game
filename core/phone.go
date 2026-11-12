package core

import (
	"jewel-game/rl"
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"
	"jewel-game/utils"
)

// A window this narrow, or taller than it is wide, cannot hold both side panels and a board.
func narrowSize(w, h int) bool {
	return w < 1100 || h > w+40
}

func narrow() bool {
	return narrowSize(rl.GetScreenWidth(), rl.GetScreenHeight())
}

// uiScaleFor is how many screen pixels one panel-space pixel covers.
// On a phone the layout is already in screen pixels, so the scale stays 1.
// Shrinking it made every label a fraction of its size and the glyphs went soft.
func uiScaleFor(w, h int) float32 {
	if narrowSize(w, h) {
		return 1
	}
	s := float32(h) / DESIGN_HEIGHT
	s = min(s, float32(w)/(2*utils.SIDE_PANEL_WIDTH+620))
	return max(0.5, min(1.4, s))
}

func uiScale() float32 {
	return uiScaleFor(rl.GetScreenWidth(), rl.GetScreenHeight())
}

// phoneFoeH is the height of the foe panel in a phone duel: enough for its name, HP, mana and a
// scrolling list of its moves and the battle log, so the phone sees what the desktop panel does.
const phoneFoeH = 160

// phoneBoardRect is the board in screen pixels. Portrait puts a dock under it.
// Landscape puts the dock beside it, so the gems stay large enough to tap.
func phoneBoardRect(sw, sh float32) rl.Rectangle {
	if sw > sh {
		dock := max(float32(220), min(320, sw*0.42))
		size := min(sh-12, sw-dock-16)
		if size < 48 {
			size = 48
		}
		return rl.Rectangle{X: 6, Y: (sh - size) / 2, Width: size, Height: size}
	}
	top := float32(6 + phoneFoeH + 22)
	dock := sh * 0.36
	if dock < 240 {
		dock = 240
	}
	if dock > 380 {
		dock = 380
	}
	size := min(sw-12, sh-top-dock)
	if size < 48 {
		size = 48
	}
	return rl.Rectangle{X: (sw - size) / 2, Y: top + (sh-top-dock-size)/2, Width: size, Height: size}
}

func (c *GameController) drawNarrow(w, h float32) {
	q := c.Quest
	if q.InDuel() {
		c.drawPhoneDuel(w, h)
		return
	}
	navH := float32(0)
	if q.Hero != nil {
		navH = c.drawPhoneNav(w)
	}
	content := rl.Rectangle{X: 8, Y: 8 + navH, Width: w - 16, Height: h - 16 - navH}
	if content.Height < 40 {
		content.Height = 40
	}
	if q.Hero != nil && q.Screen == ScreenWorld {
		if w > h {
			mapW := content.Width * 0.62
			q.DrawCenter(rl.Rectangle{X: content.X, Y: content.Y, Width: mapW - 14, Height: content.Height})
			q.DrawRight(rl.Rectangle{X: content.X + mapW, Y: content.Y, Width: content.Width - mapW, Height: content.Height})
			return
		}
		mapH := content.Height * 0.56
		q.DrawCenter(rl.Rectangle{X: content.X, Y: content.Y, Width: content.Width, Height: mapH - 14})
		q.DrawRight(rl.Rectangle{X: content.X, Y: content.Y + mapH, Width: content.Width, Height: content.Height - mapH})
		return
	}
	q.DrawCenter(content)
}

// drawPhoneNav is the row of screens. It returns the height it used, including the gap below it.
func (c *GameController) drawPhoneNav(w float32) float32 {
	q := c.Quest
	h := q.Hero
	type item struct {
		s       Screen
		label   string
		enabled bool
	}
	here := rpg.LocationByID(h.Location)
	items := []item{
		{ScreenWorld, text.T("ui.map_short", "Map"), true},
		{ScreenHero, text.T("ui.hero", "Hero"), true},
		{ScreenOrrery, text.T("ui.orrery_nav", "Orrery"), true},
		{ScreenSpells, text.T("ui.spells", "Spells"), true},
		{ScreenQuests, text.T("ui.quests_short", "Quests"), true},
		{ScreenShop, text.T("ui.shop", "Shop"), here.Town},
		{ScreenCitadel, text.T("ui.citadel_short", "Citadel"), true},
	}
	cols := len(items)
	btnW := (w - 16 - float32(cols-1)*6) / float32(cols)
	if btnW < 78 {
		cols = 4
		btnW = (w - 16 - 3*6) / 4
	}
	rows := (len(items) + cols - 1) / cols
	bh := float32(36)
	for i, it := range items {
		col := i % cols
		row := i / cols
		r := rl.Rectangle{X: 8 + float32(col)*(btnW+6), Y: 8 + float32(row)*(bh+6), Width: btnW, Height: bh}
		if it.enabled {
			if ui.Button(r, it.label, q.Screen == it.s) {
				q.Screen = it.s
			}
		} else {
			ui.ButtonIf(r, it.label, false)
		}
	}
	if q.Screen == ScreenShop && !here.Town {
		q.Screen = ScreenWorld
	}
	return float32(rows)*(bh+6) + 4
}

func (c *GameController) drawPhoneDuel(w, h float32) {
	q := c.Quest
	s := uiScale()
	b := c.boardArea()
	if w > h {
		dock := rl.Rectangle{X: (b.X+b.Width)/s + 8, Y: 8, Width: w - (b.X+b.Width)/s - 16, Height: h - 16}
		eh := min(float32(phoneFoeH), dock.Height*0.5)
		q.drawPhoneEnemy(rl.Rectangle{X: dock.X, Y: dock.Y, Width: dock.Width, Height: eh})
		q.drawPhoneDock(rl.Rectangle{X: dock.X, Y: dock.Y + eh + 6, Width: dock.Width, Height: dock.Height - eh - 6})
		return
	}
	q.drawPhoneEnemy(rl.Rectangle{X: 8, Y: 6, Width: w - 16, Height: phoneFoeH})
	top := (b.Y + b.Height) / s
	q.drawPhoneDock(rl.Rectangle{X: 8, Y: top + 8, Width: w - 16, Height: h - top - 16})
}

func (v *QuestView) drawPhoneEnemy(r rl.Rectangle) {
	if v.Duel == nil || r.Height < 8 {
		return
	}
	ui.Panel(r)
	e := v.Duel.Sides[rpg.Enemy]
	col := ui.Column{X: r.X + 20, Y: r.Y + 12, W: r.Width - 40, Gap: 4}
	v.portraits[rpg.Enemy] = rl.Vector2{X: r.X + 30, Y: col.Y + 14}
	name := col.Next(28)
	ui.TextFit(text.Tf("ui.level_short", "%s  L%d", e.Name, e.Level), name.X, name.Y, 22, name.Width, ui.TextColor)
	bar := col.Next(24)
	v.hpBars[rpg.Enemy] = bar
	ui.Bar(bar, text.T("ui.hp", "HP"), e.HP, e.MaxHP, hpColor)
	mana := col.Next(20)
	v.drawPhoneMana(mana, e)
	status := ""
	for _, ef := range e.Effects {
		status += text.Tf("ui.effect", "%s(%d) ", rpg.EffectLabel(ef.Kind), ef.Turns)
	}
	if status != "" {
		col.Paragraph(status, 15, ui.Accent)
	}
	// The foe's moves and the battle log scroll, matching what the desktop's right panel shows.
	view := rl.Rectangle{X: r.X + 14, Y: col.Y + 2, Width: r.Width - 28, Height: r.Y + r.Height - 6 - (col.Y + 2)}
	if view.Height < 8 {
		return
	}
	col.Y = v.rightScroll.Begin(view, uiScale())
	for i, s := range e.Spells {
		v.drawFoeMove(&col, e, s, i)
	}
	col.Heading(text.T("ui.battle_log", "Battle log"))
	for _, line := range v.Duel.Log {
		col.Paragraph(line.Text(), 13, ui.DimText)
	}
	v.rightScroll.End(col.Y)
}

// drawFoeMove is one compact row of a foe's spell: its name and cost, dimmed when it can't cast it.
func (v *QuestView) drawFoeMove(col *ui.Column, c *rpg.Combatant, s *rpg.Spell, i int) {
	r := col.Next(28)
	const size = 16
	costW := manaCostWidth(s.Cost, size)
	ink := ui.DimText
	if c.CanCast(s) {
		ink = ui.TextColor
	}
	label := s.Label()
	if cd := c.Cooldowns[s.ID]; cd > 0 {
		label += text.Tf("ui.cooldown", " (%d)", cd)
	}
	ui.TextFit(label, r.X+4, r.Y+4, size, r.Width-12-costW, ink)
	if costW > 0 {
		v.drawManaCosts(r.X+r.Width-costW, r.Y+4, size, s.Cost, &c.Mana)
	}
}

func (v *QuestView) drawPhoneDock(r rl.Rectangle) {
	if v.Duel == nil || r.Height < 20 {
		return
	}
	ui.Panel(r)
	p := v.Duel.Sides[rpg.Player]
	col := ui.Column{X: r.X + 20, Y: r.Y + 12, W: r.Width - 40, Gap: 4}
	v.portraits[rpg.Player] = rl.Vector2{X: r.X + r.Width/2, Y: r.Y + 20}
	ui.TextFit(text.Tf("ui.level_short", "%s  L%d", p.Name, p.Level), col.X, col.Y, 22, col.W, ui.Accent)
	col.Y += 28
	bar := col.Next(26)
	v.hpBars[rpg.Player] = bar
	ui.Bar(bar, text.T("ui.hp", "HP"), p.HP, p.MaxHP, hpColor)
	mana := col.Next(24)
	v.drawPhoneMana(mana, p)
	status := ""
	for _, ef := range p.Effects {
		status += text.Tf("ui.effect", "%s(%d) ", rpg.EffectLabel(ef.Kind), ef.Turns)
	}
	if status != "" {
		col.Paragraph(status, 15, ui.Accent)
	}
	btns := col.Next(44)
	gap := float32(8)
	half := (btns.Width - gap) / 2
	mine := v.myTurn() && !v.TM.Busy()
	if ui.ButtonIf(rl.Rectangle{X: btns.X, Y: btns.Y, Width: half, Height: btns.Height}, text.T("ui.hint", "Hint"), mine) {
		v.TM.ShowHint()
	}
	if ui.ButtonIf(rl.Rectangle{X: btns.X + half + gap, Y: btns.Y, Width: half, Height: btns.Height}, text.T("ui.flee_short", "Flee"), mine) {
		v.flee()
	}
	view := rl.Rectangle{X: r.X + 6, Y: col.Y, Width: r.Width - 12, Height: r.Y + r.Height - 8 - col.Y}
	if view.Height < 8 {
		return
	}
	col.Y = v.dockScroll.Begin(view, uiScale())
	if v.dockScroll.Overflows() {
		col.W -= 14
	}
	for i, s := range p.Spells {
		v.drawSpell(&col, p, s, i, rpg.Player)
	}
	v.dockScroll.End(col.Y)
}

// drawPhoneMana draws the four mana pools as short bars across one row.
func (v *QuestView) drawPhoneMana(r rl.Rectangle, c *rpg.Combatant) {
	const gap = 4
	bw := (r.Width - 3*gap) / rpg.NUM_MANA
	for i := range c.Mana {
		box := rl.Rectangle{X: r.X + float32(i)*(bw+gap), Y: r.Y, Width: bw, Height: r.Height}
		rl.DrawRectangleRounded(box, 0.4, 4, rl.Color{R: 0, G: 0, B: 0, A: 140})
		if c.MaxMana[i] > 0 && c.Mana[i] > 0 {
			fill := box
			fill.Width = box.Width * float32(min(c.Mana[i], c.MaxMana[i])) / float32(c.MaxMana[i])
			rl.DrawRectangleRounded(fill, 0.4, 4, manaColors[i])
		}
		label := text.Tf("ui.mana_frac", "%d", c.Mana[i])
		ink := ui.TextColor
		lum := 0.3*float32(manaColors[i].R) + 0.59*float32(manaColors[i].G) + 0.11*float32(manaColors[i].B)
		if c.Mana[i] > 0 && lum > 160 {
			ink = rl.Color{R: 28, G: 20, B: 24, A: 255}
		}
		ui.TextCentered(label, box.X+box.Width/2, box.Y+(box.Height-16)/2, 16, ink)
	}
}
