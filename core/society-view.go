package core

import (
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"
)

// drawStanding lists the hero's standing with each society, the allegiance taken at the lock, and
// which societies are hostile. Nothing shows in a story without societies.
func (v *QuestView) drawStanding(col *ui.Column) {
	h := v.Hero
	if len(rpg.Societies) == 0 {
		return
	}
	col.Heading(text.T("ui.standing", "Standing"))
	for _, s := range rpg.Societies {
		line := text.Tf("ui.standing_line", "%s  %+d", s.Name, h.Standing[s.ID])
		if h.Allegiance == s.ID {
			line += "  " + text.T("ui.standing_joined", "(joined)")
		}
		if h.Hostile(s.ID) {
			line += "  " + text.T("ui.standing_hostile", "(hostile)")
		}
		col.Paragraph(line, 17, ui.TextColor)
	}
}

// drawCompanions lists the companions in the hero's company, with their favour and what they
// bring, and lets the player pick the one who comes into fights. One button per companion, in a
// column, so it fits the phone as it is.
func (v *QuestView) drawCompanions(col *ui.Column) {
	h := v.Hero
	if len(h.Companions) == 0 {
		return
	}
	col.Heading(text.T("ui.companions", "Companions"))
	col.Paragraph(text.T("ui.companions_help", "One companion comes into your fights."), 15, ui.DimText)
	for _, id := range h.Companions {
		c := rpg.CompanionByID(id)
		if c == nil {
			continue
		}
		brings := ""
		if s := rpg.SpellByID(c.Rite); s != nil {
			brings = text.Tf("ui.companion_rite", "Rite: %s", s.Label())
		} else if c.Passive != nil {
			brings = rpg.PerkAbout(&c.Passive.Perk)
		}
		col.Paragraph(text.Tf("ui.companion_line", "%s  (favour %d)", c.Name, h.Favour[c.ID]), 18, ui.TextColor)
		col.Paragraph(brings, 15, ui.DimText)
		btn := col.Next(34)
		btn.Width = min(btn.Width, 240)
		chosen := h.Companion == c.ID
		label := text.T("ui.companion_bring", "Bring into fights")
		if chosen {
			label = text.T("ui.companion_brought", "Comes into fights")
		}
		if ui.Button(btn, label, chosen) {
			if chosen {
				h.Bring("")
			} else {
				h.Bring(c.ID)
			}
			v.save()
		}
	}
}
