package core

import (
	"jewel-game/rpg"
	"jewel-game/text"
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
func plaque(r rl.Rectangle, selected bool) {
	ui.Frame(r)
	if selected || ui.Hover(r) {
		rl.DrawRectangleRoundedLines(r, 0.06, 8, 2, ui.Accent)
	}
}

func card(r rl.Rectangle, selected bool) bool {
	plaque(r, selected)
	return ui.Clicked(r)
}

func (v *QuestView) drawClassPicker(r rl.Rectangle) {
	ui.Panel(r)
	ui.TextCentered(text.T("ui.choose_class", "Choose your class"), r.X+r.Width/2, r.Y+30, 42, ui.Accent)
	grid := rl.Rectangle{X: r.X + 30, Y: r.Y + 100, Width: r.Width - 60, Height: r.Height - 130}
	w, h := (grid.Width-20)/2, (grid.Height-20)/2
	for i := range rpg.Classes {
		c := &rpg.Classes[i]
		box := rl.Rectangle{X: grid.X + float32(i%2)*(w+20), Y: grid.Y + float32(i/2)*(h+20), Width: w, Height: h}
		if card(box, false) {
			v.Hero = rpg.NewHero(c.Name)
			v.Screen = ScreenWorld
			v.save()
			v.scenes.Play(rpg.Story.Opening)
		}
		drawFoe("hero", foeColor(c.Color), rl.Vector2{X: box.X + box.Width/2, Y: box.Y + 70}, 42, v.time, 0, true)
		col := ui.Column{X: box.X + 20, Y: box.Y + 125, W: box.Width - 40, Gap: 4}
		name := col.Next(32)
		ui.TextCentered(c.Label(), name.X+name.Width/2, name.Y, 28, foeColor(c.Color))
		col.Paragraph(c.About(), 17, ui.TextColor)
		col.Y += 6
		for stat, n := range c.Stats {
			col.Line(text.Tf("ui.stat_line", "%s  %d", rpg.StatLabel(stat), n), 15, ui.DimText)
		}
	}
}

func (v *QuestView) drawWorld(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	drawMapBackground(r)

	if v.hex.on {
		for i := range rpg.Locations {
			l := &rpg.Locations[i]
			for _, id := range l.Roads {
				o := rpg.LocationByID(id)
				if o == nil || l.ID >= o.ID {
					continue
				}
				col := rl.Fade(rl.Beige, 0.85)
				if !h.LocationOpen(l) || !h.LocationOpen(o) {
					col = rl.Fade(rl.Beige, 0.18)
				}
				pts := roadScreen(r, l.ID, o.ID)
				for s := 1; s < len(pts); s++ {
					rl.DrawLineEx(pts[s-1], pts[s], 7, rl.Fade(rl.Black, 0.55))
					rl.DrawLineEx(pts[s-1], pts[s], 4, col)
				}
				for s := 1; s < len(pts)-1; s++ {
					rl.DrawCircleV(pts[s], 7, rl.Black)
					rl.DrawCircleV(pts[s], 5, rl.Beige)
				}
			}
		}
	}
	for i := range rpg.Locations {
		v.drawMapNode(r, &rpg.Locations[i])
	}

	// the hero token, mid-road if walking
	at := placePoint(r, h.Location)
	if len(v.route) > 0 {
		from := v.legFrom
		if from == "" {
			from = h.Location
		}
		at = pointAlong(roadScreen(r, from, v.route[0]), v.legT)
	}
	drawFoe("hero", foeColor(rpg.ClassByName(h.Class).Color), rl.Vector2{X: at.X, Y: at.Y - 34}, 20, v.time, 0, true)
	ui.TextOutlined(rpg.Story.MapName, r.X+30, r.Y+24, 30, ui.Accent)
	v.drawHexEditor(r)
}

func (v *QuestView) drawMapNode(r rl.Rectangle, l *rpg.Location) {
	h := v.Hero
	p := placePoint(r, l.ID)
	if !h.LocationOpen(l) {
		if v.hex.on {
			rl.DrawCircleV(p, 14, rl.Color{R: 40, G: 40, B: 50, A: 220})
		}
		ui.TextCenteredOutlined("?", p.X, p.Y-10, 20, ui.DimText)
		return
	}
	hit := rl.Rectangle{X: p.X - 24, Y: p.Y - 24, Width: 48, Height: 48}
	if v.hex.on {
		col := rl.Color{R: 180, G: 70, B: 70, A: 255}
		if l.Town {
			col = rl.Color{R: 80, G: 130, B: 220, A: 255}
		}
		if ui.Hover(hit) {
			rl.DrawCircleV(p, 25, rl.Fade(ui.Accent, 0.4))
		}
		if l.ID == h.Location {
			rl.DrawRing(p, 21, 25, 0, 360, 36, ui.Accent)
		}
		if l.Town {
			rl.DrawRectangleRounded(rl.Rectangle{X: p.X - 17, Y: p.Y - 17, Width: 34, Height: 34}, 0.25, 6, col)
		} else {
			rl.DrawPoly(p, 6, 19, 0, col)
		}
	}
	ui.TextCenteredOutlined(text.Tf("ui.place_level", "%s (%d)", l.Name, l.Level), p.X, p.Y+26, 15, ui.TextColor)
	if len(h.OfferedQuests(l.ID)) > 0 || len(h.QuestsHere(l.ID)) > 0 {
		ui.TextCenteredOutlined("!", p.X+18, p.Y-36, 30, ui.Accent)
	}
	if v.hex.on {
		return
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
		col.Line(text.T("ui.on_the_road", "On the road..."), 18, ui.DimText)
		return
	}

	here := h.QuestsHere(l.ID)
	if len(here) > 0 {
		col.Heading(text.T("ui.quest_targets", "Quest targets"))
		for _, q := range here {
			foe := rpg.FoeByID(q.Foe)
			if ui.Button(col.Next(40), text.Tf("ui.foe_level", "%s  L%d", foe.Name, q.FoeLevel), false) {
				v.Screen = ScreenWorld
				v.startDuel(q.Foe, q.FoeLevel, l.ID)
			}
			col.Line(text.Tf("ui.quest_progress", "%s  %d / %d", q.Title, h.Active[q.ID], q.Count), 14, ui.DimText)
		}
	}
	if len(l.Foes) > 0 {
		col.Heading(text.T("ui.roaming", "Roaming foes"))
		level := roamLevel(l, h)
		for _, id := range l.Foes {
			foe := rpg.FoeByID(id)
			if ui.Button(col.Next(36), text.Tf("ui.foe_level", "%s  L%d", foe.Name, level), false) {
				v.startDuel(id, level, l.ID)
			}
		}
	}
	if offers := h.OfferedQuests(l.ID); len(offers) > 0 {
		col.Heading(text.T("ui.offered", "Quests offered"))
		for _, q := range offers {
			q := q
			label := q.Title
			if q.Main {
				label = text.Tf("ui.main_mark", "* %s", q.Title)
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
		col.Heading(text.T("ui.people", "People"))
		for _, p := range people {
			if ui.Button(col.Next(36), text.Tf("ui.talk_to", "Talk to %s", p.Name), false) {
				v.scenes.Play(p.Scene)
			}
		}
	}
	if l.Town {
		col.Heading(text.T("ui.town", "Town"))
		if ui.Button(col.Next(38), text.T("ui.visit_shop", "Visit the shop"), v.Screen == ScreenShop) {
			v.Screen = ScreenShop
		}
	}
	col.Heading(text.T("ui.travel", "Travel"))
	col.Paragraph(text.T("ui.travel_help", "Click a location on the map. Roads can be ambushed."), 15, ui.DimText)
}

func (v *QuestView) offerQuest(q *rpg.Quest) {
	if q.OfferScene != "" {
		v.scenes.Play(q.OfferScene)
		return
	}
	foe := rpg.FoeByID(q.Foe)
	at := rpg.LocationByID(q.At)
	body := text.Tf("ui.quest_offer", "%s\n\nDefeat %d x %s (L%d) at %s. Reward: %d gold, %d XP.", q.Intro, q.Count, foe.Name, q.FoeLevel, at.Name, q.Gold, q.XP)
	v.show(Modal{
		Title: q.Title,
		Body:  body,
		Buttons: []ModalButton{
			{text.T("ui.accept", "Accept"), func() { v.Hero.Accept(q); v.save() }},
			{text.T("ui.not_yet", "Not yet"), nil},
		},
	})
}

func (v *QuestView) drawHeroSheet(r rl.Rectangle) {
	h := v.Hero
	c := rpg.NewHeroCombatant(h)
	ui.Panel(r)
	col := ui.Column{X: r.X + 30, Y: r.Y + 24, W: r.Width - 60, Gap: 6}
	title := col.Next(44)
	ui.Text(text.Tf("ui.level_class", "Level %d %s", h.Level, rpg.ClassByName(h.Class).Label()), title.X, title.Y, 38, ui.Accent)
	col.Line(text.Tf("ui.points", "Points to spend: %d", h.Points), 22, ui.TextColor)
	col.Heading(text.T("ui.stats", "Stats"))
	cls := rpg.ClassByName(h.Class)
	for stat := range rpg.NUM_STATS {
		row := col.Next(54)
		ui.Text(text.Tf("ui.stat_line", "%s  %d", rpg.StatLabel(stat), c.Stats[stat]), row.X, row.Y+2, 22, ui.TextColor)
		ui.Text(rpg.StatBonus(stat, c.Stats[stat]), row.X, row.Y+30, 15, ui.DimText)
		btn := rl.Rectangle{X: row.X + row.Width - 130, Y: row.Y + 4, Width: 130, Height: 38}
		if ui.ButtonIf(btn, text.Tf("ui.raise_stat", "+1  (cost %d)", cls.Cost[stat]), h.Points >= cls.Cost[stat]) {
			h.RaiseStat(stat)
			v.save()
		}
	}
	col.Heading(text.T("ui.in_a_duel", "In a duel"))
	col.Line(text.Tf("ui.duel_totals", "HP %d    Skull damage bonus %+d    Blocks %d    Gold +%d%%    XP +%d%%",
		c.MaxHP, c.SkullDamage+c.Stats[rpg.Battle]/4+c.Level/2, c.Armor+c.Stats[rpg.Morale]/10, c.GoldPct, c.XPPct), 17, ui.TextColor)
	col.Heading(text.T("ui.equipped", "Equipped"))
	for _, slot := range rpg.Slots {
		name := text.T("ui.empty_slot", "-")
		if it := rpg.ItemByID(h.Gear[slot]); it != nil {
			name = it.Label() + "  " + it.About()
		}
		col.Line(text.Tf("ui.gear_line", "%s  %s", rpg.SlotLabel(slot), name), 17, ui.TextColor)
	}
	if ui.Button(rl.Rectangle{X: r.X + r.Width - 210, Y: r.Y + 24, Width: 180, Height: 38}, text.T("ui.new_hero", "New hero..."), false) {
		v.show(Modal{
			Title: text.T("ui.start_over", "Start over?"),
			Body:  text.T("ui.start_over_body", "This deletes your current hero for good."),
			Buttons: []ModalButton{
				{text.T("ui.delete_hero", "Delete hero"), func() {
					if err := rpg.DeleteSave(rpg.SavePath()); err != nil {
						ui.AddMessage("%s", text.Tf("log.save_delete", "Could not delete save: %v", err))
					}
					v.Hero = nil
					v.scenes.Stop()
					v.Screen = ScreenClass
				}},
				{text.T("ui.keep_playing", "Keep playing"), nil},
			},
		})
	}
}

func (v *QuestView) drawSpellbook(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	col := ui.Column{X: r.X + 30, Y: r.Y + 24, W: r.Width - 60, Gap: 6}
	title := col.Next(44)
	ui.Text(text.T("ui.spellbook", "Spellbook"), title.X, title.Y, 38, ui.Accent)
	col.Line(text.Tf("ui.spellbook_help", "Equipped %d / %d. Click to swap. New spells come with levels.", len(h.Spells), rpg.MAX_SPELLS), 18, ui.TextColor)
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
			ui.Text(text.Tf("ui.spell_locked", "%s  (level %d)", s.Label(), s.Level), row.X+14, row.Y+18, 20, ui.DimText)
			continue
		}
		if card(row, equipped) {
			h.ToggleSpell(s.ID)
			v.save()
		}
		ui.Text(s.Label(), row.X+14, row.Y+8, 22, ui.TextColor)
		ui.Text(s.About(), row.X+14, row.Y+36, 16, ui.DimText)
		const size = 20
		if w := manaCostWidth(s.Cost, size); w > 0 {
			v.drawManaCosts(row.X+row.Width-14-w, row.Y+10, size, s.Cost, nil)
		}
	}
}

func (v *QuestView) drawQuestLog(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	col := ui.Column{X: r.X + 30, Y: r.Y + 24, W: r.Width - 60, Gap: 6}
	title := col.Next(44)
	ui.Text(text.T("ui.quests", "Quests"), title.X, title.Y, 38, ui.Accent)
	col.Heading(text.T("ui.active", "Active"))
	if len(h.Active) == 0 {
		col.Line(text.T("ui.no_quests", "None. Towns marked ! have work for you."), 18, ui.DimText)
	}
	for i := range rpg.Quests {
		q := &rpg.Quests[i]
		if n, ok := h.Active[q.ID]; ok {
			col.Line(q.Title, 22, ui.TextColor)
			col.Line(text.Tf("ui.quest_task", "Defeat %s (L%d) at %s: %d / %d", rpg.FoeByID(q.Foe).Name, q.FoeLevel, rpg.LocationByID(q.At).Name, n, q.Count), 16, ui.DimText)
			col.Y += 4
		}
	}
	col.Heading(text.T("ui.completed", "Completed"))
	for _, id := range h.Done {
		if q := rpg.QuestByID(id); q != nil {
			col.Line(q.Title, 16, ui.DimText)
		}
	}
}

func (v *QuestView) drawShop(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	ui.Text(text.T("ui.shop", "Shop"), r.X+30, r.Y+20, 38, ui.Accent)
	ui.Text(text.Tf("ui.gold", "Gold %d", h.Gold), r.X+r.Width-200, r.Y+28, 24, goldColor)

	view := rl.Rectangle{X: r.X + 18, Y: r.Y + 72, Width: r.Width - 36, Height: r.Height - 90}
	track := rl.Rectangle{X: view.X + view.Width - 16, Y: view.Y + 4, Width: 12, Height: view.Height - 8}
	v.dragShopScroll(view, track)
	if ui.Hover(view) {
		v.shopScroll -= shopWheel(rl.Wheel()) / uiScale()
	}
	if v.shopScroll < 0 {
		v.shopScroll = 0
	}
	if v.shopScroll > v.shopMax {
		v.shopScroll = v.shopMax
	}
	s := uiScale()
	rl.BeginScissorMode(int32(view.X*s), int32(view.Y*s), int32(view.Width*s), int32(view.Height*s))
	col := ui.Column{X: view.X + 12, Y: view.Y - v.shopScroll, W: view.Width - 40, Gap: 8}
	top := col.Y
	for _, slot := range rpg.Slots {
		col.Y += 8
		head := col.Next(30)
		ui.Text(rpg.SlotLabel(slot), head.X, head.Y, 24, ui.Accent)
		rl.DrawLineEx(rl.Vector2{X: head.X, Y: head.Y + 28}, rl.Vector2{X: head.X + head.Width, Y: head.Y + 28}, 1, ui.PanelEdge)
		if slot == rpg.SlotMount && !h.Has("stables") {
			col.Line(text.T("ui.need_stables", "Build Stables in your citadel to buy mounts."), 22, ui.DimText)
			continue
		}
		for i := range rpg.Items {
			it := &rpg.Items[i]
			owned := slices.Contains(h.Owned, it.ID)
			if it.Slot != slot || (it.QuestOnly && !owned) {
				continue
			}
			v.drawShopRow(&col, h, it)
		}
	}
	v.shopMax = max(0, col.Y-top-view.Height)
	rl.EndScissorMode()
	v.drawShopScroll(track)
}

// shopWheel turns one frame of wheel input into pixels of the list.
// A mouse notch on the desktop build is about ±1. A trackpad, and the browser,
// already report how far the fingers moved, often as a burst of pixel deltas.
// Multiplying those by a row height makes a two-finger swipe race through the list.
func shopWheel(dy float32) float32 {
	if dy > 3 || dy < -3 {
		return dy
	}
	return dy * 48
}

// dragShopScroll moves the list when the thumb or its track is dragged.
func (v *QuestView) dragShopScroll(view, track rl.Rectangle) {
	if v.shopMax <= 0 {
		v.shopGrab = false
		return
	}
	thumb := shopThumb(track, v.shopScroll, v.shopMax)
	if !rl.IsMouseButtonDown(rl.MouseLeftButton) {
		v.shopGrab = false
		return
	}
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && ui.Hover(track) {
		v.shopGrab = true
		if !ui.Hover(thumb) {
			// Jump the thumb so the click lands on it.
			frac := (rl.GetMousePosition().Y - track.Y - thumb.Height/2) / (track.Height - thumb.Height)
			v.shopScroll = max(0, min(1, frac)) * v.shopMax
		}
	}
	if !v.shopGrab {
		return
	}
	thumb = shopThumb(track, v.shopScroll, v.shopMax)
	frac := (rl.GetMousePosition().Y - track.Y - thumb.Height/2) / (track.Height - thumb.Height)
	v.shopScroll = max(0, min(1, frac)) * v.shopMax
}

func shopThumb(track rl.Rectangle, scroll, maxScroll float32) rl.Rectangle {
	span := track.Height
	// The thumb shows how much of the list is on screen. The list is the
	// visible height plus whatever is still scrolled off.
	visible := span
	total := span + maxScroll
	h := max(28, span*visible/total)
	if h > span {
		h = span
	}
	travel := span - h
	y := track.Y
	if maxScroll > 0 {
		y += travel * (scroll / maxScroll)
	}
	return rl.Rectangle{X: track.X, Y: y, Width: track.Width, Height: h}
}

func (v *QuestView) drawShopScroll(track rl.Rectangle) {
	if v.shopMax <= 0 {
		return
	}
	rl.DrawRectangleRounded(track, 0.8, 6, rl.Color{R: 16, G: 14, B: 28, A: 220})
	thumb := shopThumb(track, v.shopScroll, v.shopMax)
	col := ui.Accent
	if ui.Hover(thumb) || v.shopGrab {
		col = rl.Color{R: 255, G: 214, B: 120, A: 255}
	}
	rl.DrawRectangleRounded(thumb, 0.8, 6, col)
}

var itemIcons map[string]rl.Texture2D

func itemIcon(id string) rl.Texture2D {
	if itemIcons == nil {
		itemIcons = map[string]rl.Texture2D{}
	}
	if tex, ok := itemIcons[id]; ok {
		return tex
	}
	tex := rl.LoadTexture("_res/items/" + id + ".png")
	if tex.ID != 0 {
		rl.SetTextureFilter(tex, rl.FilterPoint)
	}
	itemIcons[id] = tex
	return tex
}

func (v *QuestView) drawShopRow(col *ui.Column, h *rpg.Hero, it *rpg.Item) {
	row := col.Next(64)
	worn := h.Gear[it.Slot] == it.ID
	owned := slices.Contains(h.Owned, it.ID)
	nameCol := ui.TextColor
	if it.Level > h.Level && !owned {
		nameCol = ui.DimText
	}
	if icon := itemIcon(it.ID); icon.ID != 0 {
		rl.DrawTexturePro(icon,
			rl.Rectangle{Width: float32(icon.Width), Height: float32(icon.Height)},
			rl.Rectangle{X: row.X, Y: row.Y + 6, Width: 52, Height: 52},
			rl.Vector2{}, 0, rl.White)
	}
	ui.Text(it.Label(), row.X+62, row.Y+6, 26, nameCol)
	about := it.About()
	size := float32(18)
	maxW := row.Width - 62 - 160
	for size > 13 && ui.TextWidth(about, size) > maxW {
		size--
	}
	ui.Text(about, row.X+62, row.Y+36, size, ui.DimText)
	btn := rl.Rectangle{X: row.X + row.Width - 148, Y: row.Y + 12, Width: 140, Height: 40}
	switch {
	case worn:
		ui.Button(btn, text.T("ui.equipped_tag", "Equipped"), true)
	case owned:
		if ui.Button(btn, text.T("ui.equip", "Equip"), false) {
			h.Buy(it)
			v.save()
		}
	case it.Level > h.Level:
		ui.ButtonIf(btn, text.Tf("ui.need_level", "Level %d", it.Level), false)
	default:
		if ui.ButtonIf(btn, text.Tf("ui.buy", "Buy %d", it.Price), h.Gold >= it.Price) {
			h.Buy(it)
			v.save()
		}
	}
}

// citadelSprite is the hex picture for a building in the yard.
func citadelSprite(id string) hexRef {
	switch id {
	case "treasury":
		return hx(hexSheetTown, 4, 5)
	case "stables":
		return hx(hexSheetTown, 0, 4)
	case "barracks":
		return hx(hexSheetTown, 5, 2)
	case "library":
		return hx(hexSheetTown, 4, 0)
	case "temple":
		return hx(hexSheetOcean, 0, 1)
	case "mage_tower":
		return hx(hexSheetTown, 4, 2)
	case "war_room":
		return hx(hexSheetTown, 3, 4)
	default:
		return hx(hexSheetTown, 2, 0)
	}
}

func (v *QuestView) drawCitadelYard(col *ui.Column, h *rpg.Hero) {
	n := len(rpg.Buildings)
	if n == 0 {
		return
	}
	gap := float32(8)
	rowH := float32(128)
	per := n
	if slot := (col.W - gap*float32(n-1)) / float32(n); slot < 96 && n > 1 {
		per = (n + 1) / 2
	}
	loadHexSheets()
	for start := 0; start < n; start += per {
		end := min(start+per, n)
		count := end - start
		row := col.Next(rowH)
		slotW := (row.Width - gap*float32(count-1)) / float32(count)
		rl.DrawRectangleRounded(row, 0.04, 6, rl.Color{R: 36, G: 58, B: 40, A: 160})
		for i := start; i < end; i++ {
			b := &rpg.Buildings[i]
			x := row.X + float32(i-start)*(slotW+gap)
			slot := rl.Rectangle{X: x, Y: row.Y + 4, Width: slotW, Height: row.Height - 8}
			built := h.Has(b.ID)
			scale := min(slotW*0.85, 78) / hexCell
			foot := rl.Vector2{X: slot.X + slot.Width/2, Y: slot.Y + slot.Height - 28}
			tint := rl.White
			if !built {
				tint = rl.Fade(rl.White, 0.28)
			}
			drawHexTile(citadelSprite(b.ID), foot, scale, tint)
			if !built {
				ui.TextCentered(text.Tf("ui.build", "Build %d", b.Price), slot.X+slot.Width/2, slot.Y+2, 13, goldColor)
			}
			ui.TextCentered(b.Label(), slot.X+slot.Width/2, slot.Y+slot.Height-22, 14, ui.TextColor)
			if !built && ui.Clicked(slot) && h.Gold >= b.Price {
				h.Build(b)
				v.save()
			}
		}
	}
}

func (v *QuestView) drawCitadel(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	col := ui.Column{X: r.X + 30, Y: r.Y + 24, W: r.Width - 60, Gap: 8}
	title := col.Next(44)
	ui.Text(text.T("ui.citadel", "Your Citadel"), title.X, title.Y, 38, ui.Accent)
	ui.Text(text.Tf("ui.gold", "Gold %d", h.Gold), title.X+title.Width-170, title.Y+8, 24, goldColor)
	col.Paragraph(text.T("ui.citadel_help", "Spend your gold on buildings. Their bonuses last forever and apply in every duel."), 18, ui.TextColor)
	col.Y += 6
	v.drawCitadelYard(&col, h)
	col.Y += 10
	for i := range rpg.Buildings {
		b := &rpg.Buildings[i]
		row := col.Next(72)
		plaque(row, h.Has(b.ID))
		ui.Text(b.Label(), row.X+16, row.Y+10, 24, ui.TextColor)
		ui.Text(b.About(), row.X+16, row.Y+42, 16, ui.DimText)
		btn := rl.Rectangle{X: row.X + row.Width - 160, Y: row.Y + 16, Width: 140, Height: 40}
		if h.Has(b.ID) {
			ui.Button(btn, text.T("ui.built", "Built"), true)
		} else if ui.ButtonIf(btn, text.Tf("ui.build", "Build %d", b.Price), h.Gold >= b.Price) {
			h.Build(b)
			v.save()
		}
	}
}
