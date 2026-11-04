package core

import (
	"fmt"
	"jewel-game/rpg"
	"jewel-game/ui"
	"slices"

	"jewel-game/rl"
)

// DrawCenter draws the screen between the panels. During a duel that space is the board.
func (v *QuestView) DrawCenter(r rl.Rectangle) {
	switch v.Screen {
	case ScreenClass:
		v.drawClassPicker(r)
	case ScreenWorld:
		v.drawWorld(r)
	case ScreenHero:
		v.drawHeroSheet(r)
	case ScreenSpells:
		v.drawSpellbook(r)
	case ScreenQuests:
		v.drawQuestLog(r)
	case ScreenShop:
		v.drawShop(r)
	case ScreenCitadel:
		v.drawCitadel(r)
	}
}

// card draws a hoverable box and reports a click on it.
func card(r rl.Rectangle, selected bool) bool {
	edge := ui.PanelEdge
	if selected || ui.Hover(r) {
		edge = ui.Accent
	}
	rl.DrawRectangleRounded(r, 0.06, 8, rl.Color{ R: 32, G: 28, B: 58, A: 240 })
	rl.DrawRectangleRoundedLines(r, 0.06, 8, 2, edge)
	return ui.Clicked(r)
}

func (v *QuestView) drawClassPicker(r rl.Rectangle) {
	ui.Panel(r)
	ui.TextCentered("Choose your class", r.X+r.Width/2, r.Y+30, 42, ui.Accent)
	grid := rl.Rectangle{ X: r.X + 30, Y: r.Y + 100, Width: r.Width - 60, Height: r.Height - 130 }
	w, h := (grid.Width-20)/2, (grid.Height-20)/2
	for i := range rpg.Classes {
		c := &rpg.Classes[i]
		box := rl.Rectangle{ X: grid.X + float32(i%2)*(w+20), Y: grid.Y + float32(i/2)*(h+20), Width: w, Height: h }
		if card(box, false) {
			v.Hero = rpg.NewHero(c.Name)
			v.Screen = ScreenWorld
			v.save()
			v.scenes.Play(rpg.Story.Opening)
		}
		drawFoe("hero", foeColor(c.Color), rl.Vector2{ X: box.X + box.Width/2, Y: box.Y + 70 }, 42, v.time, 0, true)
		col := ui.Column{ X: box.X + 20, Y: box.Y + 125, W: box.Width - 40, Gap: 4 }
		name := col.Next(32)
		ui.TextCentered(c.Name, name.X+name.Width/2, name.Y, 28, foeColor(c.Color))
		col.Paragraph(c.Desc, 17, ui.TextColor)
		col.Y += 6
		for stat, n := range c.Stats {
			col.Line(fmt.Sprintf("%-8s %2d", rpg.StatNames[stat], n), 15, ui.DimText)
		}
	}
}

func mapPoint(r rl.Rectangle, l *rpg.Location) rl.Vector2 {
	return rl.Vector2{ X: r.X + 50 + l.X*(r.Width-100), Y: r.Y + 60 + l.Y*(r.Height-120) }
}

func (v *QuestView) drawWorld(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	rl.DrawRectangleRounded(rl.Rectangle{ X: r.X + 12, Y: r.Y + 12, Width: r.Width - 24, Height: r.Height - 24 }, 0.03, 8, rl.Color{ R: 40, G: 54, B: 44, A: 150 })
	ui.Text(rpg.Story.MapName, r.X+30, r.Y+24, 30, ui.Accent)

	for i := range rpg.Locations {
		l := &rpg.Locations[i]
		for _, id := range l.Roads {
			o := rpg.LocationByID(id)
			if l.ID < o.ID {
				col := rl.Fade(rl.Beige, 0.7)
				if !h.LocationOpen(l) || !h.LocationOpen(o) {
					col = rl.Fade(rl.Beige, 0.12)
				}
				rl.DrawLineEx(mapPoint(r, l), mapPoint(r, o), 4, col)
			}
		}
	}
	for i := range rpg.Locations {
		v.drawMapNode(r, &rpg.Locations[i])
	}

	// the hero token, mid-road if walking
	at := mapPoint(r, rpg.LocationByID(h.Location))
	if len(v.route) > 0 {
		to := mapPoint(r, rpg.LocationByID(v.route[0]))
		at = rl.Vector2{ X: at.X + (to.X-at.X)*v.legT, Y: at.Y + (to.Y-at.Y)*v.legT }
	}
	drawFoe("hero", foeColor(rpg.ClassByName(h.Class).Color), rl.Vector2{ X: at.X, Y: at.Y - 34 }, 20, v.time, 0, true)
}

func (v *QuestView) drawMapNode(r rl.Rectangle, l *rpg.Location) {
	h := v.Hero
	p := mapPoint(r, l)
	if !h.LocationOpen(l) {
		rl.DrawCircleV(p, 14, rl.Color{ R: 40, G: 40, B: 50, A: 220 })
		ui.TextCentered("?", p.X, p.Y-10, 20, ui.DimText)
		return
	}
	col := rl.Color{ R: 180, G: 70, B: 70, A: 255 }
	if l.Town {
		col = rl.Color{ R: 80, G: 130, B: 220, A: 255 }
	}
	hit := rl.Rectangle{ X: p.X - 24, Y: p.Y - 24, Width: 48, Height: 48 }
	if ui.Hover(hit) {
		rl.DrawCircleV(p, 25, rl.Fade(ui.Accent, 0.4))
	}
	if l.ID == h.Location {
		rl.DrawRing(p, 21, 25, 0, 360, 36, ui.Accent)
	}
	if l.Town {
		rl.DrawRectangleRounded(rl.Rectangle{ X: p.X - 17, Y: p.Y - 17, Width: 34, Height: 34 }, 0.25, 6, col)
	} else {
		rl.DrawPoly(p, 6, 19, 0, col)
	}
	ui.TextCentered(fmt.Sprintf("%s (%d)", l.Name, l.Level), p.X, p.Y+26, 15, ui.TextColor)
	if len(h.OfferedQuests(l.ID)) > 0 || len(h.QuestsHere(l.ID)) > 0 {
		ui.TextCentered("!", p.X+18, p.Y-36, 30, ui.Accent)
	}
	if ui.Clicked(hit) && l.ID != h.Location && len(v.route) == 0 {
		if route := h.Route(l.ID); route != nil {
			v.route, v.legFrom, v.legT = route, h.Location, 0
		}
	}
}

// drawLocation is the right panel away from duels: what the hero can do here.
func (v *QuestView) drawLocation(col *ui.Column) {
	h := v.Hero
	l := rpg.LocationByID(h.Location)
	name := col.Next(36)
	ui.Text(l.Name, name.X, name.Y, 30, ui.Accent)
	col.Paragraph(l.Desc, 16, ui.TextColor)
	travelling := len(v.route) > 0
	if travelling {
		col.Line("On the road...", 18, ui.DimText)
		return
	}

	here := h.QuestsHere(l.ID)
	if len(here) > 0 {
		col.Heading("Quest targets")
		for _, q := range here {
			foe := rpg.FoeByID(q.Foe)
			if ui.Button(col.Next(40), fmt.Sprintf("%s  L%d", foe.Name, q.FoeLevel), false) {
				v.Screen = ScreenWorld
				v.startDuel(q.Foe, q.FoeLevel, l.ID)
			}
			col.Line(fmt.Sprintf("%s  %d / %d", q.Title, h.Active[q.ID], q.Count), 14, ui.DimText)
		}
	}
	if len(l.Foes) > 0 {
		col.Heading("Roaming foes")
		level := roamLevel(l, h)
		for _, id := range l.Foes {
			foe := rpg.FoeByID(id)
			if ui.Button(col.Next(36), fmt.Sprintf("%s  L%d", foe.Name, level), false) {
				v.startDuel(id, level, l.ID)
			}
		}
	}
	if offers := h.OfferedQuests(l.ID); len(offers) > 0 {
		col.Heading("Quests offered")
		for _, q := range offers {
			q := q
			label := q.Title
			if q.Main {
				label = "* " + label
			}
			if ui.Button(col.Next(40), label, false) {
				v.offerQuest(q)
			}
		}
	}
	var people []rpg.Person
	for _, p := range l.People {
		if h.Meets(p.Requires) {
			people = append(people, p)
		}
	}
	if len(people) > 0 {
		col.Heading("People")
		for _, p := range people {
			if ui.Button(col.Next(36), "Talk to "+p.Name, false) {
				v.scenes.Play(p.Scene)
			}
		}
	}
	if l.Town {
		col.Heading("Town")
		if ui.Button(col.Next(38), "Visit the shop", v.Screen == ScreenShop) {
			v.Screen = ScreenShop
		}
	}
	col.Heading("Travel")
	col.Paragraph("Click a location on the map. Roads can be ambushed.", 15, ui.DimText)
}

func (v *QuestView) offerQuest(q *rpg.Quest) {
	if q.OfferScene != "" {
		v.scenes.Play(q.OfferScene)
		return
	}
	foe := rpg.FoeByID(q.Foe)
	at := rpg.LocationByID(q.At)
	body := fmt.Sprintf("%s\n\nDefeat %d x %s (L%d) at %s. Reward: %d gold, %d XP.", q.Intro, q.Count, foe.Name, q.FoeLevel, at.Name, q.Gold, q.XP)
	v.show(Modal{
		Title:	q.Title,
		Body:	body,
		Buttons: []ModalButton{
			{ "Accept", func() { v.Hero.Accept(q); v.save() } },
			{ "Not yet", nil },
		},
	})
}

func (v *QuestView) drawHeroSheet(r rl.Rectangle) {
	h := v.Hero
	c := rpg.NewHeroCombatant(h)
	ui.Panel(r)
	col := ui.Column{ X: r.X + 30, Y: r.Y + 24, W: r.Width - 60, Gap: 6 }
	title := col.Next(44)
	ui.Text(fmt.Sprintf("Level %d %s", h.Level, h.Class), title.X, title.Y, 38, ui.Accent)
	col.Line(fmt.Sprintf("Points to spend: %d", h.Points), 22, ui.TextColor)
	col.Heading("Stats")
	cls := rpg.ClassByName(h.Class)
	for stat := range rpg.NUM_STATS {
		row := col.Next(54)
		ui.Text(fmt.Sprintf("%-8s %3d", rpg.StatNames[stat], c.Stats[stat]), row.X, row.Y+2, 22, ui.TextColor)
		ui.Text(rpg.StatHelp[stat], row.X, row.Y+30, 15, ui.DimText)
		btn := rl.Rectangle{ X: row.X + row.Width - 130, Y: row.Y + 4, Width: 130, Height: 38 }
		if ui.ButtonIf(btn, fmt.Sprintf("+1  (cost %d)", cls.Cost[stat]), h.Points >= cls.Cost[stat]) {
			h.RaiseStat(stat)
			v.save()
		}
	}
	col.Heading("In a duel")
	col.Line(fmt.Sprintf("HP %d    Skull damage bonus %+d    Blocks %d    Gold +%d%%    XP +%d%%",
		c.MaxHP, c.SkullDamage+c.Stats[rpg.Battle]/4+c.Level/2, c.Armor+c.Stats[rpg.Morale]/10, c.GoldPct, c.XPPct), 17, ui.TextColor)
	col.Heading("Equipped")
	for _, slot := range rpg.Slots {
		name := "-"
		if it := rpg.ItemByID(h.Gear[slot]); it != nil {
			name = it.Name + "  " + it.Desc
		}
		col.Line(fmt.Sprintf("%-8s %s", slot, name), 17, ui.TextColor)
	}
	if ui.Button(rl.Rectangle{ X: r.X + r.Width - 210, Y: r.Y + 24, Width: 180, Height: 38 }, "New hero...", false) {
		v.show(Modal{
			Title:	"Start over?",
			Body:	"This deletes your current hero for good.",
			Buttons: []ModalButton{
				{ "Delete hero", func() {
					if err := rpg.DeleteSave(rpg.SavePath()); err != nil {
						ui.AddMessage("Could not delete save: %v", err)
					}
					v.Hero = nil
					v.scenes.Stop()
					v.Screen = ScreenClass
				} },
				{ "Keep playing", nil },
			},
		})
	}
}

func (v *QuestView) drawSpellbook(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	col := ui.Column{ X: r.X + 30, Y: r.Y + 24, W: r.Width - 60, Gap: 6 }
	title := col.Next(44)
	ui.Text("Spellbook", title.X, title.Y, 38, ui.Accent)
	col.Line(fmt.Sprintf("Equipped %d / %d. Click to swap. New spells come with levels.", len(h.Spells), rpg.MAX_SPELLS), 18, ui.TextColor)
	col.Y += 8
	for i := range rpg.Spells {
		s := &rpg.Spells[i]
		if s.Class != h.Class {
			continue
		}
		row := col.Next(64)
		known := s.Level <= h.Level
		equipped := slices.Contains(h.Spells, s.ID)
		if !known {
			ui.Text(fmt.Sprintf("%s  (level %d)", s.Name, s.Level), row.X+14, row.Y+18, 20, ui.DimText)
			continue
		}
		if card(row, equipped) {
			h.ToggleSpell(s.ID)
			v.save()
		}
		ui.Text(s.Name, row.X+14, row.Y+8, 22, ui.TextColor)
		ui.Text(s.Desc, row.X+14, row.Y+36, 16, ui.DimText)
		x := row.X + row.Width - 130
		for m, n := range s.Cost {
			if n > 0 {
				ui.Text(fmt.Sprint(n), x, row.Y+10, 20, rl.ColorBrightness(manaColors[m], 0.25))
				x += 30
			}
		}
	}
}

func (v *QuestView) drawQuestLog(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	col := ui.Column{ X: r.X + 30, Y: r.Y + 24, W: r.Width - 60, Gap: 6 }
	title := col.Next(44)
	ui.Text("Quests", title.X, title.Y, 38, ui.Accent)
	col.Heading("Active")
	if len(h.Active) == 0 {
		col.Line("None. Towns marked ! have work for you.", 18, ui.DimText)
	}
	for i := range rpg.Quests {
		q := &rpg.Quests[i]
		if n, ok := h.Active[q.ID]; ok {
			col.Line(q.Title, 22, ui.TextColor)
			col.Line(fmt.Sprintf("Defeat %s (L%d) at %s: %d / %d", rpg.FoeByID(q.Foe).Name, q.FoeLevel, rpg.LocationByID(q.At).Name, n, q.Count), 16, ui.DimText)
			col.Y += 4
		}
	}
	col.Heading("Completed")
	for _, id := range h.Done {
		if q := rpg.QuestByID(id); q != nil {
			col.Line(q.Title, 16, ui.DimText)
		}
	}
}

func (v *QuestView) drawShop(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	title := rl.Rectangle{ X: r.X + 30, Y: r.Y + 20, Width: r.Width - 60, Height: 44 }
	ui.Text("Shop", title.X, title.Y, 38, ui.Accent)
	ui.Text(fmt.Sprintf("Gold %d", h.Gold), title.X+title.Width-170, title.Y+8, 24, goldColor)
	col := ui.Column{ X: r.X + 30, Y: r.Y + 70, W: r.Width - 60, Gap: 3 }
	for _, slot := range rpg.Slots {
		col.Heading(slot)
		if slot == rpg.SlotMount && !h.Has("stables") {
			col.Line("Build Stables in your citadel to buy mounts.", 16, ui.DimText)
			continue
		}
		for i := range rpg.Items {
			it := &rpg.Items[i]
			owned := slices.Contains(h.Owned, it.ID)
			if it.Slot != slot || (it.QuestOnly && !owned) {
				continue
			}
			row := col.Next(32)
			worn := h.Gear[slot] == it.ID
			nameCol := ui.TextColor
			if it.Level > h.Level {
				nameCol = ui.DimText
			}
			ui.Text(it.Name, row.X, row.Y+6, 18, nameCol)
			ui.Text(it.Desc, row.X+190, row.Y+8, 14, ui.DimText)
			btn := rl.Rectangle{ X: row.X + row.Width - 130, Y: row.Y, Width: 130, Height: 30 }
			switch {
			case worn:
				ui.Button(btn, "Equipped", true)
			case owned:
				if ui.Button(btn, "Equip", false) {
					h.Buy(it)
					v.save()
				}
			case it.Level > h.Level:
				ui.ButtonIf(btn, fmt.Sprintf("Level %d", it.Level), false)
			default:
				if ui.ButtonIf(btn, fmt.Sprintf("Buy %d", it.Price), h.Gold >= it.Price) {
					h.Buy(it)
					v.save()
				}
			}
		}
	}
}

func (v *QuestView) drawCitadel(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	col := ui.Column{ X: r.X + 30, Y: r.Y + 24, W: r.Width - 60, Gap: 8 }
	title := col.Next(44)
	ui.Text("Your Citadel", title.X, title.Y, 38, ui.Accent)
	ui.Text(fmt.Sprintf("Gold %d", h.Gold), title.X+title.Width-170, title.Y+8, 24, goldColor)
	col.Paragraph("Spend your gold on buildings. Their bonuses last forever and apply in every duel.", 18, ui.TextColor)
	col.Y += 8
	for i := range rpg.Buildings {
		b := &rpg.Buildings[i]
		row := col.Next(72)
		card(row, h.Has(b.ID))
		ui.Text(b.Name, row.X+16, row.Y+10, 24, ui.TextColor)
		ui.Text(b.Desc, row.X+16, row.Y+42, 16, ui.DimText)
		btn := rl.Rectangle{ X: row.X + row.Width - 160, Y: row.Y + 16, Width: 140, Height: 40 }
		if h.Has(b.ID) {
			ui.Button(btn, "Built", true)
		} else if ui.ButtonIf(btn, fmt.Sprintf("Build %d", b.Price), h.Gold >= b.Price) {
			h.Build(b)
			v.save()
		}
	}
}
