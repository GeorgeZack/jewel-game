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
	case ScreenAcademy:
		v.drawAcademy(r)
	case ScreenOrrery:
		v.drawOrrery(r)
	case ScreenGrimoire:
		v.drawGrimoire(r)
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

// menuContent lays out a menu panel's inner column. On phones it starts a scroll view and pads the
// content clear of the ornate frame corners; the returned func ends the scroll.
func (v *QuestView) menuContent(r rl.Rectangle, gap float32) (*ui.Column, func()) {
	top, cx, cw := r.Y+24, r.X+30, r.Width-60
	if narrow() {
		top = v.menuScroll.Begin(rl.Rectangle{X: r.X + 18, Y: r.Y + 30, Width: r.Width - 36, Height: r.Height - 46}, uiScale())
		cx, cw = r.X+38, r.Width-84
	}
	col := &ui.Column{X: cx, Y: top, W: cw, Gap: gap}
	if narrow() {
		return col, func() { v.menuScroll.End(col.Y) }
	}
	return col, func() {}
}

func (v *QuestView) drawClassPicker(r rl.Rectangle) {
	ui.Panel(r)
	ui.TextCenteredFit(text.T("ui.choose_class", "Choose your discipline"), r.X+r.Width/2, r.Y+30, 42, r.Width-24, ui.Accent)
	grid := rl.Rectangle{X: r.X + 30, Y: r.Y + 100, Width: r.Width - 60, Height: r.Height - 130}
	w, h := (grid.Width-20)/2, (grid.Height-20)/2
	for i := range rpg.Classes {
		c := &rpg.Classes[i]
		box := rl.Rectangle{X: grid.X + float32(i%2)*(w+20), Y: grid.Y + float32(i/2)*(h+20), Width: w, Height: h}
		if card(box, false) {
			v.startRun(c.Name)
		}
		drawFoe("hero", foeColor(c.Color), rl.Vector2{X: box.X + box.Width/2, Y: box.Y + 58}, 36, v.time, 0, true)
		col := ui.Column{X: box.X + 20, Y: box.Y + 104, W: box.Width - 40, Gap: 4}
		name := col.Next(32)
		ui.TextCenteredFit(c.Label(), name.X+name.Width/2, name.Y, 28, name.Width, foeColor(c.Color))
		col.Paragraph(c.About(), 17, ui.TextColor)
		col.Y += 6
		// stats in two columns, left out if the card is too short for them
		rows := (len(c.Stats) + 1) / 2
		if col.Y+float32(rows)*(ui.MinText+6) > box.Y+box.Height-12 {
			continue
		}
		half := col.W / 2
		for stat, n := range c.Stats {
			x := col.X + float32(stat%2)*half
			y := col.Y + float32(stat/2)*(ui.MinText+6)
			ui.Text(text.Tf("ui.stat_line", "%s  %d", rpg.StatLabel(stat), n), x, y, ui.MinText, ui.DimText)
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
	mapLabelBoxes = mapLabelBoxes[:0]
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
	mapTitle := rpg.Story.MapName
	mx, my := r.X+16, r.Y+16
	if narrow() {
		mx, my = r.X+38, r.Y+30
	}
	ui.TextOutlined(mapTitle, mx, my, ui.FitSize(mapTitle, 28, r.X+r.Width-mx-16, 16), ui.Accent)
	v.drawHexEditor(r)
}

// mapLabelBoxes keeps place names from stacking on a small map. Cleared each time the map is drawn.
var mapLabelBoxes []rl.Rectangle

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
	label := text.Tf("ui.place_level", "%s (%d)", l.Name, l.Level)
	size := float32(17)
	if narrow() {
		label = l.Name
		size = 16
	}
	tw := ui.TextWidth(label, size)
	lab := rl.Rectangle{X: p.X - tw/2 - 2, Y: p.Y + 22, Width: tw + 4, Height: size + 2}
	skipLabel := false
	if narrow() && l.ID != h.Location {
		for _, o := range mapLabelBoxes {
			if lab.X < o.X+o.Width && lab.X+lab.Width > o.X && lab.Y < o.Y+o.Height && lab.Y+lab.Height > o.Y {
				skipLabel = true
				break
			}
		}
	}
	if !skipLabel {
		mapLabelBoxes = append(mapLabelBoxes, lab)
		ui.TextCenteredOutlined(label, p.X, p.Y+22, size, ui.TextColor)
	}
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
	l := h.AsHeld(rpg.LocationByID(h.Location)) // as its holder, or a hostile society, has it
	name := col.Next(36)
	ui.TextFit(l.Name, name.X, name.Y, 30, name.Width, ui.Accent)
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
				v.startDuel(q.Foe, q.FoeLevel, l.ID, q.Card)
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
				v.startDuel(id, level, l.ID, "")
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
	col, end := v.menuContent(r, 6)
	defer end()
	title := col.Next(44)
	class := text.Tf("ui.level_class", "Level %d %s", h.Level, rpg.ClassByName(h.Class).Label())
	if narrow() {
		// The New hero button would cover the title, so give it its own row.
		ui.TextFit(class, title.X, title.Y, 38, title.Width, ui.Accent)
		if ui.Button(col.Next(38), text.T("ui.new_hero", "New adept..."), false) {
			v.confirmNewHero()
		}
	} else {
		ui.Text(class, title.X, title.Y, 38, ui.Accent)
	}
	col.Paragraph(text.T("ui.hero_orrery", "Stats grow on the Orrery, bought with residue from good matches."), 17, ui.DimText)
	if ui.Button(col.Next(38), text.T("ui.open_orrery", "Open the Orrery"), false) {
		v.Screen = ScreenOrrery
	}
	col.Heading(text.T("ui.stats", "Stats"))
	for stat := range rpg.NUM_STATS {
		if !h.StatShown(stat) {
			continue
		}
		row := col.Next(50)
		ui.Text(text.Tf("ui.stat_line", "%s  %d", rpg.StatLabel(stat), c.Stats[stat]), row.X, row.Y+2, 22, ui.TextColor)
		ui.TextFit(rpg.StatBonus(stat, c.Stats[stat]), row.X, row.Y+28, 16, row.Width, ui.DimText)
	}
	col.Heading(text.T("ui.in_a_duel", "In a duel"))
	col.Paragraph(text.Tf("ui.duel_totals", "HP %d    Skull damage bonus %+d    Blocks %d    Gold +%d%%    XP +%d%%",
		c.MaxHP, c.SkullDamage+c.Stats[rpg.Battle]/4+c.Level/2, c.Armor+c.Stats[rpg.Morale]/10, c.GoldPct, c.XPPct), 17, ui.TextColor)
	col.Heading(text.T("ui.equipped", "Equipped"))
	for _, slot := range rpg.Slots {
		name := text.T("ui.empty_slot", "-")
		if it := rpg.ItemByID(h.Gear[slot]); it != nil {
			name = it.Label() + "  " + it.About()
		}
		col.Paragraph(text.Tf("ui.gear_line", "%s  %s", rpg.SlotLabel(slot), name), 17, ui.TextColor)
	}
	v.drawStanding(col)
	v.drawCompanions(col)
	if !narrow() && ui.Button(rl.Rectangle{X: r.X + r.Width - 210, Y: r.Y + 24, Width: 180, Height: 38}, text.T("ui.new_hero", "New adept..."), false) {
		v.confirmNewHero()
	}
}

func (v *QuestView) confirmNewHero() {
	v.show(Modal{
		Title: text.T("ui.start_over", "Start over?"),
		Body:  text.T("ui.start_over_body", "This deletes your current adept for good. The Grimoire and what it holds are kept."),
		Buttons: []ModalButton{
			{text.T("ui.delete_hero", "Delete adept"), func() {
				if err := rpg.DeleteRun(rpg.SavePath()); err != nil {
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

func (v *QuestView) drawSpellbook(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	col, end := v.menuContent(r, 6)
	defer end()
	title := col.Next(44)
	ui.Text(text.T("ui.spellbook", "Rites"), title.X, title.Y, 38, ui.Accent)
	col.Paragraph(text.Tf("ui.spellbook_help", "Prepared %d / %d. Click to swap. Your discipline's first rites come with levels; the rest are on the Orrery.", len(h.Spells), rpg.MAX_SPELLS), 18, ui.TextColor)
	col.Y += 8
	for i := range rpg.Spells {
		s := &rpg.Spells[i]
		// rites you know, and your own class's still to come
		if !h.Knows(s) && s.Class != h.Class {
			continue
		}
		rowH := float32(64)
		if narrow() {
			rowH = 84
		}
		row := col.Next(rowH)
		equipped := slices.Contains(h.Spells, s.ID)
		if !h.Knows(s) {
			label := text.Tf("ui.spell_locked", "%s  (level %d)", s.Label(), s.Level)
			if s.Level > rpg.KIT_LEVEL {
				label = text.Tf("ui.spell_locked_orrery", "%s  (level %d, on the Orrery)", s.Label(), s.Level)
			}
			ui.TextFit(label, row.X+14, row.Y+18, 20, row.Width-28, ui.DimText)
			continue
		}
		if card(row, equipped) {
			h.ToggleSpell(s.ID)
			v.save()
		}
		const size = 20
		costW := manaCostWidth(s.Cost, size)
		textW := row.Width - 28 - costW
		ui.TextFit(s.Label(), row.X+14, row.Y+8, 22, textW, ui.TextColor)
		aboutW := textW
		if narrow() {
			aboutW = row.Width - 28
		}
		ui.Wrapped(s.About(), row.X+14, row.Y+36, 16, aboutW, ui.DimText)
		if costW > 0 {
			v.drawManaCosts(row.X+row.Width-14-costW, row.Y+10, size, s.Cost, nil)
		}
	}
}

func (v *QuestView) drawQuestLog(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	col, end := v.menuContent(r, 6)
	defer end()
	title := col.Next(44)
	ui.Text(text.T("ui.quests", "Quests"), title.X, title.Y, 38, ui.Accent)
	col.Heading(text.T("ui.active", "Active"))
	if len(h.Active) == 0 {
		col.Paragraph(text.T("ui.no_quests", "None. Towns marked ! have work for you."), 18, ui.DimText)
	}
	for i := range rpg.Quests {
		q := &rpg.Quests[i]
		if n, ok := h.Active[q.ID]; ok {
			col.Line(q.Title, 22, ui.TextColor)
			col.Paragraph(text.Tf("ui.quest_task", "Defeat %s (L%d) at %s: %d / %d", rpg.FoeByID(q.Foe).Name, q.FoeLevel, rpg.LocationByID(q.At).Name, n, q.Count), 16, ui.DimText)
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
	tx, ty := r.X+30, r.Y+20
	view := rl.Rectangle{X: r.X + 18, Y: r.Y + 72, Width: r.Width - 36, Height: r.Height - 90}
	if narrow() {
		tx, ty = r.X+38, r.Y+30
		view = rl.Rectangle{X: r.X + 24, Y: r.Y + 86, Width: r.Width - 48, Height: r.Height - 104}
	}
	ui.Text(text.T("ui.shop", "Shop"), tx, ty, 38, ui.Accent)
	here := rpg.LocationByID(h.Location)
	ui.Text(text.Tf("ui.gold", "Gold %d", h.Gold), r.X+r.Width-200, r.Y+28, 24, goldColor)
	top := v.shopScroll.Begin(view, uiScale())
	col := ui.Column{X: view.X + 12, Y: top, W: view.Width - 40, Gap: 8}
	for _, slot := range rpg.Slots {
		col.Y += 8
		head := col.Next(30)
		ui.Text(rpg.SlotLabel(slot), head.X, head.Y, 24, ui.Accent)
		rl.DrawLineEx(rl.Vector2{X: head.X, Y: head.Y + 28}, rl.Vector2{X: head.X + head.Width, Y: head.Y + 28}, 1, ui.PanelEdge)
		for i := range rpg.Items {
			it := &rpg.Items[i]
			owned := slices.Contains(h.Owned, it.ID)
			if it.Slot != slot || (it.QuestOnly && !owned) || (!owned && !h.Sells(here, it.ID)) {
				continue
			}
			v.drawShopRow(&col, h, it)
		}
	}
	v.shopScroll.End(col.Y)
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

// tooDear colors a price the hero can't pay yet.
var tooDear = rl.Color{R: 220, G: 110, B: 100, A: 230}

func (v *QuestView) drawShopRow(col *ui.Column, h *rpg.Hero, it *rpg.Item) {
	worn := h.Gear[it.Slot] == it.ID
	owned := slices.Contains(h.Owned, it.ID)
	nameCol := ui.TextColor
	if it.Level > h.Level && !owned {
		nameCol = ui.DimText
	}
	icon := itemIcon(it.ID)
	if narrow() {
		// The name, description and button can't share one line on a phone.
		row := col.Next(104)
		tx := row.X
		if icon.ID != 0 {
			rl.DrawTexturePro(icon,
				rl.Rectangle{Width: float32(icon.Width), Height: float32(icon.Height)},
				rl.Rectangle{X: row.X, Y: row.Y + 2, Width: 56, Height: 56},
				rl.Vector2{}, 0, rl.White)
			tx = row.X + 68
		}
		tw := row.X + row.Width - tx
		ui.TextFit(it.Label(), tx, row.Y+4, 26, tw, nameCol)
		ui.TextFit(it.About(), tx, row.Y+36, 17, tw, ui.DimText)
		v.shopButton(rl.Rectangle{X: row.X, Y: row.Y + 62, Width: row.Width, Height: 38}, h, it, worn, owned)
		return
	}
	row := col.Next(64)
	if icon.ID != 0 {
		rl.DrawTexturePro(icon,
			rl.Rectangle{Width: float32(icon.Width), Height: float32(icon.Height)},
			rl.Rectangle{X: row.X, Y: row.Y + 6, Width: 52, Height: 52},
			rl.Vector2{}, 0, rl.White)
	}
	ui.TextFit(it.Label(), row.X+62, row.Y+6, 26, row.Width-62-160, nameCol)
	about := it.About()
	size := float32(18)
	maxW := row.Width - 62 - 160
	for size > 13 && ui.TextWidth(about, size) > maxW {
		size--
	}
	ui.Text(about, row.X+62, row.Y+36, size, ui.DimText)
	v.shopButton(rl.Rectangle{X: row.X + row.Width - 148, Y: row.Y + 12, Width: 140, Height: 40}, h, it, worn, owned)
}

// shopButton draws the buy/equip control for an item and applies it when clicked.
func (v *QuestView) shopButton(btn rl.Rectangle, h *rpg.Hero, it *rpg.Item, worn, owned bool) {
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
	case h.Gold < it.Price:
		ui.DisabledButton(btn, text.Tf("ui.buy", "Buy %d", it.Price), tooDear)
	default:
		if ui.Button(btn, text.Tf("ui.buy", "Buy %d", it.Price), false) {
			h.Buy(it)
			v.save()
		}
	}
}
