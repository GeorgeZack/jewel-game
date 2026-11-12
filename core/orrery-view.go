package core

import (
	"math"

	"jewel-game/rl"
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"
)

var colorlessColor = rl.Color{R: 205, G: 200, B: 190, A: 255}

// residueColor is the color drawn for a residue pool: a mana color, or colorless.
func residueColor(i int) rl.Color {
	if i >= 0 && i < rpg.NUM_MANA {
		return manaColors[i]
	}
	return colorlessColor
}

func residueLabel(i int) string {
	if i < 0 || i > rpg.Colorless {
		return "?"
	}
	if i == rpg.Colorless {
		return text.T("ui.colorless", "Colorless")
	}
	return rpg.ManaLabel(i)
}

// orreryPoint places a node on the disc: rings evenly out from the middle, angles clockwise from
// the top.
func orreryPoint(c rl.Vector2, radius float32, rings int, n *rpg.OrreryNode) rl.Vector2 {
	rad := radius * float32(n.Ring) / float32(rings)
	a := n.Angle * math.Pi / 180
	return rl.Vector2{X: c.X + rad*float32(math.Sin(a)), Y: c.Y - rad*float32(math.Cos(a))}
}

// drawOrrery is the Orrery screen: the disc of nodes, the residue pools, the chosen node, and
// conversion and respec.
func (v *QuestView) drawOrrery(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	col, end := v.menuContent(r, 6)
	defer end()
	title := col.Next(44)
	ui.Text(text.T("ui.orrery", "The Orrery"), title.X, title.Y, 38, ui.Accent)

	// beside the disc when there's room, under it otherwise
	room := r.Y + r.Height - col.Y - 24
	disc := col.W
	side := !narrow() && col.W > 720
	switch {
	case side:
		disc = min(col.W*0.6, room)
	case !narrow():
		disc = min(col.W, room-500)
	}
	disc = max(disc, 220)
	area := col.Next(disc)
	area.Width = disc
	if !side {
		area.X += (col.W - disc) / 2
	}
	v.drawOrreryDisc(area)

	info := col
	if side {
		info = &ui.Column{X: area.X + disc + 24, Y: area.Y, W: col.W - disc - 24, Gap: col.Gap}
	}
	v.drawOrreryInfo(info, h)
	if side {
		col.Y = max(col.Y, info.Y)
	}
}

func (v *QuestView) drawOrreryDisc(area rl.Rectangle) {
	h := v.Hero
	c := rl.Vector2{X: area.X + area.Width/2, Y: area.Y + area.Height/2}
	nodeR := max(12, area.Width*0.04)
	radius := area.Width/2 - nodeR - 4
	rings := 1
	for i := range rpg.Orrery {
		rings = max(rings, rpg.Orrery[i].Ring)
	}
	rl.DrawCircleV(c, radius+nodeR, rl.Fade(rl.Black, 0.35))
	for ring := 1; ring <= rings; ring++ {
		rad := radius * float32(ring) / float32(rings)
		rl.DrawRing(c, rad-1, rad+1, 0, 360, 96, rl.Fade(ui.Accent, 0.25))
	}
	for i := range rpg.Orrery {
		n := &rpg.Orrery[i]
		for _, id := range n.Links {
			o := rpg.NodeByID(id)
			if o == nil {
				continue
			}
			lc := rl.Fade(ui.DimText, 0.4)
			if h.Owns(n.ID) && h.Owns(o.ID) {
				lc = ui.Accent
			} else if h.Owns(n.ID) || h.Owns(o.ID) {
				lc = rl.Fade(ui.TextColor, 0.7)
			}
			rl.DrawLineEx(orreryPoint(c, radius, rings, n), orreryPoint(c, radius, rings, o), 3, lc)
		}
	}
	pulse := float32(0.6 + 0.4*math.Sin(v.time*4))
	for i := range rpg.Orrery {
		n := &rpg.Orrery[i]
		p := orreryPoint(c, radius, rings, n)
		nr := nodeR
		if n.Seat != "" {
			nr *= 1.25
		}
		mc := residueColor(n.Mana)
		switch {
		case h.Owns(n.ID):
			rl.DrawCircleV(p, nr+2, rl.Black)
			rl.DrawCircleV(p, nr, mc)
		case h.Reaches(n):
			rl.DrawCircleV(p, nr+2, rl.Fade(mc, pulse))
			rl.DrawCircleV(p, nr-1, rl.Color{R: 30, G: 26, B: 22, A: 255})
		default:
			rl.DrawCircleV(p, nr+2, rl.Fade(rl.Black, 0.6))
			rl.DrawCircleV(p, nr, rl.Fade(mc, 0.25))
		}
		if !h.Owns(n.ID) && n.Seat == "" {
			ui.TextCentered(text.Tf("ui.node_cost", "%d", n.Cost), p.X, p.Y-ui.MinText/2, ui.MinText, ui.TextColor)
		}
		if n.ID == v.orrerySel {
			rl.DrawRing(p, nr+4, nr+7, 0, 360, 32, ui.Accent)
		}
		hit := rl.Rectangle{X: p.X - nr - 4, Y: p.Y - nr - 4, Width: 2*nr + 8, Height: 2*nr + 8}
		if ui.Clicked(hit) {
			v.orrerySel = n.ID
		}
	}
}

func (v *QuestView) drawOrreryInfo(col *ui.Column, h *rpg.Hero) {
	col.Heading(text.T("ui.residue", "Residue"))
	row := col.Next(26)
	cells := ui.Row(row, rpg.NUM_RESIDUE, 6)
	for i, cell := range cells {
		rl.DrawCircleV(rl.Vector2{X: cell.X + 9, Y: cell.Y + 12}, 8, residueColor(i))
		ui.TextFit(text.Tf("ui.residue_pool", "%d", h.Residue[i]), cell.X+22, cell.Y+2, 20, cell.Width-22, ui.TextColor)
	}
	col.Paragraph(text.T("ui.orrery_help", "Matches of four, five, and cascades leave residue of their color. Nodes cost their region's color; colorless pays for any. Buy nodes next to ones you own."), 15, ui.DimText)

	n := rpg.NodeByID(v.orrerySel)
	if n == nil && len(h.Nodes) > 0 {
		n = rpg.NodeByID(h.Nodes[0])
	}
	if n != nil {
		col.Heading(rpg.NodeLabel(n))
		col.Line(text.Tf("ui.node_region", "%s region", residueLabel(n.Mana)), 16, residueColor(n.Mana))
		if about := rpg.NodeAbout(n); about != "" {
			col.Paragraph(about, 17, ui.TextColor)
		}
		if s := rpg.SpellByID(n.Rite); s != nil && s.Level > h.Level {
			col.Line(text.Tf("ui.node_rite_level", "Castable from level %d", s.Level), 16, hpColor)
		}
		switch {
		case n.Seat != "":
			col.Line(text.Tf("ui.node_seat", "The %s's seat", rpg.ClassByName(n.Seat).Label()), 16, ui.DimText)
		case h.Owns(n.ID):
			col.Line(text.T("ui.node_owned", "Owned"), 16, ui.DimText)
		default:
			label := text.Tf("ui.node_buy", "Buy  (%d %s)", n.Cost, residueLabel(n.Mana))
			if !h.Reaches(n) {
				label = text.T("ui.node_far", "Not next to a node you own")
			}
			if ui.ButtonIf(col.Next(38), label, h.CanBuyNode(n)) && h.BuyNode(n.ID) {
				v.save()
			}
		}
	}

	col.Heading(text.T("ui.convert", "Convert"))
	col.Paragraph(text.Tf("ui.convert_help", "%d of a color becomes %d colorless.", rpg.CONVERT_STEP, rpg.CONVERT_STEP*rpg.CONVERT_PCT/100), 15, ui.DimText)
	for i, cell := range ui.Row(col.Next(38), rpg.NUM_MANA, 6) {
		if ui.ButtonIf(cell, rpg.ManaLabel(i), h.Residue[i] >= rpg.CONVERT_STEP) && h.Convert(i) {
			v.save()
		}
	}
	col.Heading(text.T("ui.respec", "Respec"))
	col.Paragraph(text.T("ui.respec_help", "Refund every node to its color's pool and go back to your seat."), 15, ui.DimText)
	if ui.ButtonIf(col.Next(38), text.Tf("ui.respec_button", "Respec  (%d gold)", rpg.RESPEC_GOLD), h.Gold >= rpg.RESPEC_GOLD && len(h.Nodes) > 1) && h.Respec() {
		v.save()
	}
}
