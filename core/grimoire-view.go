package core

import (
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"

	"jewel-game/rl"
)

// The Grimoire screen: the meta currency, the entries to buy, and the return. The names of the
// currency and of the entities are one line each in the language file (meta.*), as they have
// none yet.

func metaName() string { return text.T("meta.currency", "Marks") }

func (v *QuestView) drawGrimoire(r rl.Rectangle) {
	h, p := v.Hero, &v.Profile
	ui.Panel(r)
	view := rl.Rectangle{X: r.X + 18, Y: r.Y + 24, Width: r.Width - 36, Height: r.Height - 40}
	cx, cw := r.X+30, r.Width-60-ui.BarWidth
	if narrow() {
		view = rl.Rectangle{X: r.X + 18, Y: r.Y + 30, Width: r.Width - 36, Height: r.Height - 46}
		cx, cw = r.X+38, r.Width-84
	}
	col := &ui.Column{X: cx, Y: v.grimoireScroll.Begin(view, uiScale()), W: cw, Gap: 8}
	defer func() { v.grimoireScroll.End(col.Y) }()

	title := col.Next(44)
	ui.TextFit(text.T("meta.grimoire", "The Grimoire"), title.X, title.Y, 38, title.Width, ui.Accent)
	col.Paragraph(text.T("meta.grimoire_about", "The book outlasts the run. What you write in it changes how every later run starts."), 17, ui.DimText)
	col.Line(text.Tf("meta.banked", "%[1]s banked: %[2]d", metaName(), p.Meta), 20, goldColor)
	if n := h.Forfeit(); n > 0 {
		col.Line(text.Tf("meta.unbanked", "%[1]s earned this act, not yet banked: %[2]d", metaName(), n), 17, ui.TextColor)
	}
	col.Line(text.Tf("meta.returns", "Returns: %d", p.Returns), 17, ui.DimText)
	if p.StageTwo() {
		col.Line(text.Tf("meta.lean", "Lean: %s", leanLabel(p.LeanName())), 17, ui.DimText)
	}
	label := text.T("meta.return_early", "Return now...")
	if h.Finished {
		label = text.T("meta.return", "Return...")
	}
	if ui.Button(col.Next(40), label, false) {
		v.confirmReturn()
	}

	col.Heading(text.T("meta.entries", "Entries"))
	hidden := 0
	for i := range rpg.Grimoire.Entries {
		e := &rpg.Grimoire.Entries[i]
		if !p.EntryShown(e) {
			hidden++
			continue
		}
		v.drawEntry(col, e)
	}
	if hidden > 0 {
		col.Paragraph(text.T("meta.more_later", "Some pages are still stuck together. Return more often."), 16, ui.DimText)
	}
}

func leanLabel(lean string) string {
	switch lean {
	case rpg.LEAN_TAKE:
		return text.T("meta.lean_take", "taking from them")
	case rpg.LEAN_RESIST:
		return text.T("meta.lean_resist", "resisting them")
	}
	return text.T("meta.lean_none", "undecided")
}

// drawEntry is one entry: name, kind, what it does, and its price or that it is owned.
func (v *QuestView) drawEntry(col *ui.Column, e *rpg.GrimoireEntry) {
	p := &v.Profile
	bh := float32(38)
	descH := ui.MeasureParagraph(e.Desc, 16, col.W-24)
	top := col.Y
	box := rl.Rectangle{X: col.X, Y: top, Width: col.W, Height: 12 + 30 + descH + 8 + bh + 12}
	plaque(box, false)
	inner := ui.Column{X: box.X + 12, Y: box.Y + 12, W: box.Width - 24, Gap: 6}
	name := inner.Next(26)
	ui.TextFit(e.Name, name.X, name.Y, 22, name.Width, ui.TextColor)
	if e.Lean != "" {
		tag := text.T("meta.kind_take", "Takes from them")
		if e.Lean == rpg.LEAN_RESIST {
			tag = text.T("meta.kind_resist", "Resists them")
		}
		ui.TextFit(tag, name.X, name.Y+24, 14, name.Width, ui.Accent)
		inner.Y += 4
	}
	inner.Paragraph(e.Desc, 16, ui.DimText)
	btn := inner.Next(bh)
	btn.Width = min(btn.Width, 260)
	switch {
	case p.Owns(e.ID):
		ui.ButtonIf(btn, text.T("meta.owned", "Written"), false)
	default:
		if ui.ButtonIf(btn, text.Tf("meta.buy", "Write it  (%[1]d %[2]s)", e.Cost, metaName()), p.CanBuy(e)) && p.Buy(e.ID) {
			v.saveProfile()
			ui.AddMessage("%s", text.Tf("meta.bought", "Written in the Grimoire: %s. It starts with your next run.", e.Name))
		}
	}
	box.Height = inner.Y - top + 6
	col.Y = top + box.Height + col.Gap
}

// confirmReturn asks before a return, saying exactly what it takes. At an ending it costs nothing
// that was earned; mid-run it also forfeits the meta currency the current act has not banked.
func (v *QuestView) confirmReturn() {
	h := v.Hero
	body := text.T("meta.return_lose", "The run ends. You lose your level, gold, gear, map, quests, academy and its staff, Orrery, standing, allegiance and companions, and start again at the first event, choosing a class.")
	if n := h.Forfeit(); n > 0 {
		body += "\n\n" + text.Tf("meta.return_forfeit", "This act is not over: the %[1]d %[2]s it has earned are forfeited.", n, metaName())
	} else if !h.Finished {
		body += "\n\n" + text.Tf("meta.return_nothing", "This act has earned no %s yet, so none is forfeited.", metaName())
	}
	body += "\n\n" + text.Tf("meta.return_keep", "Kept: %[1]d %[2]s banked, the Grimoire, the endings seen, and the count of returns.", v.Profile.Meta, metaName())
	v.show(Modal{
		Title: text.T("meta.return_title", "Return?"),
		Body:  body,
		// staying comes first, where a double click lands
		Buttons: []ModalButton{
			{text.T("meta.stay", "Stay"), nil},
			{text.T("meta.return_go", "Return"), v.doReturn},
		},
	})
}

// doReturn throws the run away and sends the player to the class choice. The next run's scenes,
// the entities' first, start once a class is picked.
func (v *QuestView) doReturn() {
	if v.Hero == nil {
		return
	}
	if n := rpg.Return(v.Hero); n > 0 {
		ui.AddMessage("%s", text.Tf("meta.forfeited", "Forfeited %[1]d %[2]s.", n, metaName()))
	}
	if err := rpg.DeleteRun(rpg.SavePath()); err != nil {
		ui.AddMessage("%s", text.Tf("log.save_delete", "Could not delete save: %v", err))
	}
	v.saveProfile()
	v.Hero = nil
	v.scenes.Stop()
	v.route = nil
	v.Screen = ScreenClass
}

func (v *QuestView) saveProfile() {
	if err := rpg.SaveProfile(rpg.ProfilePath(), v.Profile); err != nil {
		ui.AddMessage("%s", text.Tf("log.save_write", "Could not save: %v", err))
		v.saveErr = true
	}
}

// startRun makes the hero of a new run and plays its opening scenes.
func (v *QuestView) startRun(class string) {
	v.Hero = rpg.StartRun(class)
	v.Screen = ScreenWorld
	v.save()
	for _, id := range rpg.RunScenes() {
		v.scenes.Play(id)
	}
}
