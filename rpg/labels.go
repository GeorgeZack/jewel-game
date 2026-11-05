package rpg

import (
	"strings"

	"jewel-game/text"
)

func slug(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", "_"))
}

// Label and About are the words shown for a thing. The name stored on the
// value stays the id (saves and story files keep using it).

func (c *Class) Label() string { return text.T("class."+slug(c.Name)+".name", c.Name) }
func (c *Class) About() string { return text.T("class."+slug(c.Name)+".desc", c.Desc) }

func (s *Spell) Label() string { return text.T("spell."+s.ID+".name", s.Name) }
func (s *Spell) About() string { return text.T("spell."+s.ID+".desc", s.Desc) }

func (it *Item) Label() string { return text.T("item."+it.ID+".name", it.Name) }
func (it *Item) About() string { return text.T("item."+it.ID+".desc", it.Desc) }

func (b *Building) Label() string { return text.T("building."+b.ID+".name", b.Name) }
func (b *Building) About() string { return text.T("building."+b.ID+".desc", b.Desc) }

func StatLabel(i int) string { return text.T("stat."+slug(StatNames[i])+".name", StatNames[i]) }
func StatAbout(i int) string { return text.T("stat."+slug(StatNames[i])+".help", StatHelp[i]) }

// StatBonus is what a stat of n is worth right now, including gear.
func StatBonus(i, n int) string {
	key := "stat." + slug(StatNames[i]) + ".now"
	switch i {
	case Fire, Water, Air, Earth:
		return text.Tf(key, "+%d max "+StatNames[i]+" mana, +%d per match", n, n/6)
	case Battle:
		return text.Tf(key, "+%d damage per skull match", n/4)
	case Cunning:
		return text.Tf(key, "+%d%% gold and XP", n*2)
	case Morale:
		return text.Tf(key, "+%d max HP, blocks %d", n*3, n/10)
	default:
		return ""
	}
}
func ManaLabel(i int) string { return StatLabel(i) }

func EffectLabel(k EffectKind) string {
	return text.T("effect."+slug(EffectNames[k]), EffectNames[k])
}

func SlotLabel(slot string) string { return text.T("slot."+slug(slot), slot) }
