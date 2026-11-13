package core

import (
	"jewel-game/rl"
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"
)

// drawAcademy is the academy: facilities to build, henchmen to hire, and chapters and the ledger.
func (v *QuestView) drawAcademy(r rl.Rectangle) {
	h := v.Hero
	ui.Panel(r)
	view := rl.Rectangle{X: r.X + 18, Y: r.Y + 20, Width: r.Width - 36, Height: r.Height - 36}
	cx, cw := r.X+30, r.Width-76
	if narrow() {
		view = rl.Rectangle{X: r.X + 18, Y: r.Y + 30, Width: r.Width - 36, Height: r.Height - 46}
		cx, cw = r.X+38, r.Width-84
	}
	top := v.academyScroll.Begin(view, uiScale())
	col := &ui.Column{X: cx, Y: top, W: cw, Gap: 8}
	defer func() { v.academyScroll.End(col.Y) }()

	title := col.Next(44)
	if narrow() {
		ui.TextFit(text.T("ui.academy", "Your Academy"), title.X, title.Y, 38, title.Width, ui.Accent)
		col.Line(text.Tf("ui.gold", "Gold %d", h.Gold), 24, goldColor)
	} else {
		ui.Text(text.T("ui.academy", "Your Academy"), title.X, title.Y, 38, ui.Accent)
		ui.Text(text.Tf("ui.gold", "Gold %d", h.Gold), title.X+title.Width-170, title.Y+8, 24, goldColor)
	}
	if !h.AcademyOpen() {
		col.Paragraph(text.T("ui.academy_locked", "The academy has not been founded yet."), 18, ui.DimText)
		return
	}
	col.Paragraph(text.T("ui.academy_help", "Gold builds facilities and pays your henchmen. Chapters and dues only move when you win duels."), 18, ui.TextColor)
	v.drawFacilities(col)
	v.drawHenchmen(col)
	v.drawLedger(col)
}

// costButton draws a button that pays gold, greyed in red when the hero can't afford it.
func costButton(btn rl.Rectangle, label string, cost, gold int) bool {
	if gold < cost {
		ui.DisabledButton(btn, label, tooDear)
		return false
	}
	return ui.Button(btn, label, false)
}

func (v *QuestView) drawFacilities(col *ui.Column) {
	h := v.Hero
	col.Heading(text.T("ui.facilities", "Facilities"))
	cols := 2
	if narrow() {
		cols = 1
	}
	// On a phone the button gets its own row, so every title has the plaque's full width.
	const btnW = 124
	about := func(f *rpg.Facility) string {
		lvl := h.FacilityLevel(f.ID)
		next := text.Tf("ui.facility_level", "Level %d / %d", lvl, len(f.Levels))
		if !h.FacilityAvailable(f) {
			next += ". " + text.T("ui.facility_locked", "Not available yet.")
		} else if lvl < len(f.Levels) {
			next += ". " + text.Tf("ui.facility_next", "Next: %s", rpg.LevelAbout(f.Levels[lvl]))
		}
		return next
	}
	var cells []rl.Rectangle
	for i := range rpg.Facilities {
		if i%cols == 0 {
			// a row is as tall as its longest plaque, whose lines wrap
			textW := (col.W-10*float32(cols-1))/float32(cols) - 40
			rowH := float32(0)
			for j := i; j < min(i+cols, len(rpg.Facilities)); j++ {
				g := &rpg.Facilities[j]
				rowH = max(rowH, 54+ui.MeasureParagraph(g.Desc, 16, textW)+ui.MeasureParagraph(about(g), 16, textW))
			}
			if narrow() {
				rowH += 44
			}
			cells = ui.Row(col.Next(rowH), cols, 10)
		}
		f := &rpg.Facilities[i]
		row := cells[i%cols]
		lvl := h.FacilityLevel(f.ID)
		plaque(row, lvl > 0)
		in := rl.Rectangle{X: row.X + 20, Y: row.Y + 12, Width: row.Width - 40}
		titleW := in.Width - btnW - 8
		btn := rl.Rectangle{X: in.X + in.Width - btnW, Y: row.Y + 10, Width: btnW, Height: 34}
		if narrow() {
			titleW = in.Width
			btn = rl.Rectangle{X: in.X, Y: row.Y + row.Height - 50, Width: in.Width, Height: 34}
		}
		ui.TextFit(f.Name, in.X, in.Y, 22, titleW, ui.TextColor)
		y := in.Y + 32
		y += ui.Wrapped(f.Desc, in.X, y, 16, in.Width, ui.DimText)
		ui.Wrapped(about(f), in.X, y, 16, in.Width, ui.TextColor)
		cost := h.NextCost(f)
		label := text.Tf("ui.build", "Build %d", cost)
		if lvl > 0 {
			label = text.Tf("ui.upgrade", "Upgrade %d", cost)
		}
		switch {
		case cost == 0:
			ui.Button(btn, text.T("ui.built", "Built"), true)
		case !h.FacilityAvailable(f):
			ui.ButtonIf(btn, label, false)
		case costButton(btn, label, cost, h.Gold):
			h.Build(f)
			v.save()
		}
	}
}

func (v *QuestView) drawHenchmen(col *ui.Column) {
	h := v.Hero
	col.Heading(text.Tf("ui.henchmen", "Henchmen  %d / %d", len(h.Henchmen), h.HenchmenCap()))
	if len(h.Henchmen) < h.HenchmenCap() {
		btn := col.Next(36)
		btn.Width = min(btn.Width, 200)
		if costButton(btn, text.Tf("ui.hire", "Hire %d", rpg.HIRE_GOLD), rpg.HIRE_GOLD, h.Gold) {
			h.Hire()
			v.save()
		}
	}
	sites := v.postSites()
	for i := range h.Henchmen {
		m := &h.Henchmen[i]
		box := col.Next(132)
		plaque(box, false)
		in := rl.Rectangle{X: box.X + 20, Y: box.Y + 12, Width: box.Width - 40, Height: box.Height - 24}
		ui.TextFit(text.Tf("ui.henchman", "Rank %d henchman", m.Rank), in.X, in.Y, 22, in.Width, ui.TextColor)
		status := text.T("ui.henchman_free", "Free")
		switch {
		case m.Report != nil:
			status = text.Tf("ui.henchman_report", "Report from %s", rpg.LocationByID(m.Report.At).Name)
		case m.Post != "":
			status = text.Tf("ui.henchman_posted", "At %s (%s), %d / %d wins", rpg.LocationByID(m.Post).Name, rpg.ManaLabel(rpg.SiteColor(m.Post)), m.Wins, rpg.CHAPTER_WINS)
		}
		gear := text.T("ui.gear_none", "No gear")
		if it := rpg.ItemByID(m.Gear); it != nil {
			gear = it.Label()
		}
		ui.TextFit(status, in.X, in.Y+28, 16, in.Width, ui.DimText)
		ui.TextFit(text.Tf("ui.henchman_gear", "Gear: %s", gear), in.X, in.Y+48, 16, in.Width, ui.DimText)
		btns := ui.Row(rl.Rectangle{X: in.X, Y: in.Y + 70, Width: in.Width, Height: 34}, 3, 8)
		if cost := rpg.TrainCost(m); cost == 0 {
			ui.ButtonIf(btns[0], text.T("ui.max_rank", "Top rank"), false)
		} else if costButton(btns[0], text.Tf("ui.train", "Train %d", cost), cost, h.Gold) {
			h.Train(i)
			v.save()
		}
		if ui.Button(btns[1], text.T("ui.gear_next", "Gear"), false) {
			h.Equip(i, nextGear(h, m.Gear))
			v.save()
		}
		switch {
		case m.Report != nil:
			if ui.Button(btns[2], text.T("ui.claim", "Claim"), false) {
				h.Claim(i)
				v.save()
			}
		case m.Post == "" && len(sites) > 0:
			if ui.Button(btns[2], text.T("ui.post", "Post"), false) {
				h.Post(i, sites[v.postAt%len(sites)].ID)
				v.save()
			}
		case m.Post != "":
			if ui.Button(btns[2], text.T("ui.recall", "Recall"), false) {
				h.Recall(i)
				v.save()
			}
		default:
			ui.ButtonIf(btns[2], text.T("ui.post", "Post"), false)
		}
		if m.Report != nil {
			col.Paragraph(m.Report.Line, 16, ui.TextColor)
			c := rpg.SiteColor(m.Report.At)
			col.Line(text.Tf("ui.report_pays", "Pays %d gold, %d %s residue.", m.Report.Gold, m.Report.Residue[c], rpg.ManaLabel(c)), 16, goldColor)
		}
	}
}

// nextGear is the item after cur among those the hero could hand a henchman, then none.
func nextGear(h *rpg.Hero, cur string) string {
	var free []string
	for _, id := range h.Owned {
		if id == cur || h.GearFree(id) {
			free = append(free, id)
		}
	}
	for i, id := range free {
		if id == cur {
			if i+1 < len(free) {
				return free[i+1]
			}
			return ""
		}
	}
	if len(free) > 0 {
		return free[0]
	}
	return ""
}

// postSites are the sites a henchman can be posted to: every location the hero has open.
func (v *QuestView) postSites() []*rpg.Location {
	var out []*rpg.Location
	for i := range rpg.Locations {
		if l := &rpg.Locations[i]; v.Hero.LocationOpen(l) {
			out = append(out, l)
		}
	}
	return out
}

func (v *QuestView) drawLedger(col *ui.Column) {
	h := v.Hero
	col.Heading(text.T("ui.chapters", "Chapters and ledger"))
	col.Paragraph(text.Tf("ui.chapters_help", "A posted henchman's chapter completes after %d of your wins at that site, and pays residue in its color. Recalling one loses its progress. Sites you hold pay %d dues on each win there.", rpg.CHAPTER_WINS, rpg.DUES_PER_WIN), 16, ui.DimText)
	if sites := v.postSites(); len(sites) > 0 {
		v.postAt = (v.postAt%len(sites) + len(sites)) % len(sites)
		site := sites[v.postAt]
		v.postAt += ui.Stepper(col.Next(34), text.T("ui.post_to_label", "Post to"), site.Name)
		col.Line(text.Tf("ui.post_pays", "%s chapters pay %s residue.", site.Name, rpg.ManaLabel(rpg.SiteColor(site.ID))), 16, ui.DimText)
	}
	held := false
	for i := range rpg.Locations {
		l := &rpg.Locations[i]
		holder, ok := h.Holders[l.ID]
		if !ok {
			continue
		}
		held = true
		if holder == rpg.OUR_SOCIETY {
			holder = text.T("ui.holder_ours", "your academy")
		} else {
			holder = rpg.SocietyName(holder)
		}
		col.Line(text.Tf("ui.ledger_row", "%s: %s, %d dues", l.Name, holder, h.Dues[l.ID]), 17, ui.TextColor)
	}
	if !held {
		col.Line(text.T("ui.ledger_none", "No site is held yet."), 17, ui.DimText)
	}
	btn := col.Next(36)
	btn.Width = min(btn.Width, 220)
	owed := h.DuesOwed()
	if ui.ButtonIf(btn, text.Tf("ui.claim_dues", "Claim %d dues", owed), owed > 0) {
		h.ClaimDues()
		v.save()
	}
}
