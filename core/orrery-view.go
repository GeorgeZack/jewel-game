package core

import (
	"slices"
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

// orreryPoint places a node on the disc: rings evenly out from the middle, leaving the middle
// empty so the innermost ring has room, and angles clockwise from the top.
func orreryPoint(c rl.Vector2, radius float32, rings int, n *rpg.OrreryNode) rl.Vector2 {
	rad := radius * float32(n.Ring+1) / float32(rings+1)
	a := n.Angle * math.Pi / 180
	return rl.Vector2{X: c.X + rad*float32(math.Sin(a)), Y: c.Y - rad*float32(math.Cos(a))}
}

// ORRERY_ZOOM is how far the phone zooms in on one region, so its nodes are big enough to tap.
const ORRERY_ZOOM = 1.6

// orreryRegions is each region with a seat, in the Orrery's order, as mana indices.
func orreryRegions() []int {
	var out []int
	for i := range rpg.Orrery {
		if n := &rpg.Orrery[i]; n.Seat != "" && !slices.Contains(out, n.Mana) {
			out = append(out, n.Mana)
		}
	}
	return out
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
	side := !narrow() && col.W > 560
	switch {
	case side:
		disc = min(col.W*0.6, room)
	case !narrow():
		disc = min(col.W, room-500)
	}
	disc = max(disc, 220)
	if narrow() {
		// a phone zooms in on one region at a time, so every node is a finger wide
		regions := orreryRegions()
		cells := ui.Row(col.Next(40), len(regions)+1, 4)
		if ui.Button(cells[0], text.T("ui.orrery_all", "All"), v.orreryZoom == 0) {
			v.orreryZoom = 0
		}
		for i, m := range regions {
			if ui.Button(cells[i+1], rpg.ManaLabel(m), v.orreryZoom == i+1) {
				v.orreryZoom = i + 1
			}
		}
	}
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

// clipLine cuts a segment to a rectangle, reporting false when none of it is inside.
func clipLine(a, b rl.Vector2, r rl.Rectangle) (rl.Vector2, rl.Vector2, bool) {
	t0, t1 := float32(0), float32(1)
	dx, dy := b.X-a.X, b.Y-a.Y
	for _, e := range [4][2]float32{{-dx, a.X - r.X}, {dx, r.X + r.Width - a.X}, {-dy, a.Y - r.Y}, {dy, r.Y + r.Height - a.Y}} {
		p, q := e[0], e[1]
		if p == 0 {
			if q < 0 {
				return a, b, false
			}
			continue
		}
		t := q / p
		if p < 0 {
			t0 = max(t0, t)
		} else {
			t1 = min(t1, t)
		}
	}
	if t0 > t1 {
		return a, b, false
	}
	return rl.Vector2{X: a.X + t0*dx, Y: a.Y + t0*dy}, rl.Vector2{X: a.X + t1*dx, Y: a.Y + t1*dy}, true
}

func (v *QuestView) drawOrreryDisc(area rl.Rectangle) {
	h := v.Hero
	c := rl.Vector2{X: area.X + area.Width/2, Y: area.Y + area.Height/2}
	nodeR := max(10, area.Width*0.021)
	radius := area.Width/2 - 2.2*nodeR - 4
	rings := 1
	for i := range rpg.Orrery {
		rings = max(rings, rpg.Orrery[i].Ring)
	}
	// on a phone zoomed on a region, everything scales about that region's middle
	zoom := float32(1)
	focus := c
	if regions := orreryRegions(); narrow() && v.orreryZoom > 0 && v.orreryZoom <= len(regions) {
		zoom = ORRERY_ZOOM
		seat := rpg.Orrery[slices.IndexFunc(rpg.Orrery, func(n rpg.OrreryNode) bool { return n.Seat != "" && n.Mana == regions[v.orreryZoom-1] })]
		a := seat.Angle * math.Pi / 180
		focus = rl.Vector2{X: c.X + 0.68*radius*float32(math.Sin(a)), Y: c.Y - 0.68*radius*float32(math.Cos(a))}
	}
	at := func(n *rpg.OrreryNode) rl.Vector2 {
		p := orreryPoint(c, radius, rings, n)
		return rl.Vector2{X: c.X + (p.X-focus.X)*zoom, Y: c.Y + (p.Y-focus.Y)*zoom}
	}
	nodeR *= zoom

	rl.DrawRectangleRounded(area, 0.04, 8, rl.Fade(rl.Black, 0.3))
	if zoom == 1 {
		rl.DrawCircleV(c, radius+2*nodeR, rl.Fade(rl.Black, 0.25))
		for ring := 1; ring <= rings; ring++ {
			rad := radius * float32(ring+1) / float32(rings+1)
			rl.DrawRing(c, rad-1, rad+1, 0, 360, 96, rl.Fade(ui.Accent, 0.2))
		}
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
			if a, b, ok := clipLine(at(n), at(o), area); ok {
				rl.DrawLineEx(a, b, 3, lc)
			}
		}
	}
	pulse := float32(0.6 + 0.4*math.Sin(v.time*4))
	for i := range rpg.Orrery {
		n := &rpg.Orrery[i]
		p := at(n)
		nr := nodeR
		switch {
		case n.Perk != nil:
			nr *= 1.5
		case n.Seat != "":
			nr *= 1.35
		}
		if p.X < area.X+nr || p.X > area.X+area.Width-nr || p.Y < area.Y+nr || p.Y > area.Y+area.Height-nr {
			continue
		}
		// every state keeps the region's color: owned is solid with a white rim, buyable a dark
		// region-tinted face with a pulsing rim, locked a dim face with a thin rim
		mc := residueColor(n.Mana)
		rim, face := rl.Fade(mc, 0.55), rl.ColorBrightness(mc, -0.8)
		rimW := float32(1)
		costCol := ui.DimText
		switch {
		case h.Owns(n.ID):
			rim, face, rimW = rl.White, mc, 3
		case h.Reaches(n):
			rim, face, rimW = rl.Fade(mc, pulse), rl.ColorBrightness(mc, -0.6), 4
			costCol = ui.TextColor
		}
		shape := func(r float32, col rl.Color) {
			if n.Rite != "" && n.Perk == nil {
				rl.DrawPoly(p, 4, r*1.25, 90, col)
			} else {
				rl.DrawCircleV(p, r, col)
			}
		}
		if n.Perk != nil {
			rl.DrawRing(p, nr+rimW+3, nr+rimW+5, 0, 360, 40, rl.Fade(mc, 0.8))
		}
		shape(nr+rimW, rim)
		shape(nr, face)
		if !h.Owns(n.ID) && n.Seat == "" {
			size := max(12, min(ui.MinText, nr*1.2))
			ui.TextCentered(text.Tf("ui.node_cost", "%d", n.Cost), p.X, p.Y-size/2, size, costCol)
		}
		if n.ID == v.orrerySel {
			rl.DrawRing(p, nr+6, nr+9, 0, 360, 32, ui.Accent)
		}
		// at least a finger wide, whatever the drawing
		hr := max(nr+4, 20)
		hit := rl.Rectangle{X: p.X - hr, Y: p.Y - hr, Width: 2 * hr, Height: 2 * hr}
		if ui.Clicked(hit) {
			v.orrerySel = n.ID
		}
	}
}

// nodeKind names what a node gives.
func nodeKind(n *rpg.OrreryNode) string {
	switch {
	case n.Seat != "":
		return text.T("ui.node_kind_seat", "Seat")
	case n.Perk != nil:
		return text.T("ui.node_kind_perk", "Perk")
	case n.Rite != "":
		return text.T("ui.node_kind_rite", "Rite")
	}
	return text.T("ui.node_kind_stats", "Stats")
}

func (v *QuestView) drawOrreryInfo(col *ui.Column, h *rpg.Hero) {
	col.Heading(text.T("ui.residue", "Residue"))
	row := col.Next(26)
	cells := ui.Row(row, rpg.NUM_RESIDUE, 6)
	for i, cell := range cells {
		rl.DrawCircleV(rl.Vector2{X: cell.X + 9, Y: cell.Y + 12}, 8, residueColor(i))
		ui.TextFit(text.Tf("ui.residue_pool", "%d", h.Residue[i]), cell.X+22, cell.Y+2, 20, cell.Width-22, ui.TextColor)
	}
	col.Paragraph(text.T("ui.orrery_help", "Fours, fives and cascades leave residue of their color. Nodes cost their region's color; colorless pays for any."), 15, ui.DimText)
	col.Paragraph(text.T("ui.orrery_legend", "Small: stats. Diamond: a rite. Large: a perk."), 15, ui.DimText)
	col.Line(text.Tf("ui.orrery_owned", "%d of %d nodes owned", len(h.Nodes), len(rpg.Orrery)), 15, ui.DimText)

	n := rpg.NodeByID(v.orrerySel)
	if n == nil && len(h.Nodes) > 0 {
		n = rpg.NodeByID(h.Nodes[0])
	}
	if n != nil {
		col.Heading(rpg.NodeLabel(n))
		col.Line(text.Tf("ui.node_region_kind", "%s  ·  %s region", nodeKind(n), residueLabel(n.Mana)), 16, residueColor(n.Mana))
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
				label = text.Tf("ui.node_far", "Not next to a node you own  (%d %s)", n.Cost, residueLabel(n.Mana))
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
	col.Paragraph(text.T("ui.respec_help", "Refund the residue your nodes took and go back to your seat."), 15, ui.DimText)
	label := text.Tf("ui.respec_button", "Respec  (%d gold)", h.RespecCost())
	if h.RespecCost() == 0 {
		label = text.T("ui.respec_free", "Respec  (free)")
	}
	if ui.ButtonIf(col.Next(38), label, h.Gold >= h.RespecCost() && len(h.Nodes) > 1) && h.Respec() {
		v.save()
	}
}
