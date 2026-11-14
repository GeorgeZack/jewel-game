package rpg

import (
	"fmt"
	"slices"
	"strings"

	"jewel-game/text"
)

// Societies, standing, allegiance, companions and acts: the parts of the story format the
// campaign's branches are written in (DESIGN.md 5.5 and 5.7). All of it is per run and saved on the
// hero, except the profile, which outlasts runs.

// Society is a faction: societies.yaml. A site's holder is a society id, or OUR_SOCIETY for the
// player's own, which needs no entry.
type Society struct {
	ID			string		`yaml:"id"`
	Name		string		`yaml:"name"`
	Desc		string		`yaml:"desc"`
	Allegiance	bool		`yaml:"allegiance"`	// can be joined at the lock with join_society
	Hostile		[]string	`yaml:"hostile,flow"`	// hostile to the hero while these all hold; empty never
}

// Companion is a named character who can travel with the hero: companions.yaml. One companion
// at a time comes into a fight and brings a rite or a passive, not a second combatant.
type Companion struct {
	ID			string		`yaml:"id"`
	Name		string		`yaml:"name"`
	Desc		string		`yaml:"desc"`
	Society		string		`yaml:"society"`	// the society they lean to; empty for none
	Rite		string		`yaml:"rite"`		// a spell the hero can cast in fights they come to
	Cooldown	int			`yaml:"cooldown"`	// turns between casts of the rite; default the spell's own
	Passive		*CompanionPerk	`yaml:"passive"`	// or a perk on the hero's side of the duel

	spell	*Spell	// the rite with the companion's cooldown
}

// CompanionPerk is a perk with its color named, since a companion has no Orrery region to take it from.
type CompanionPerk struct {
	Color	string	`yaml:"color"`
	Perk	`yaml:",inline"`
}

var (
	Societies	[]Society
	Companions	[]Companion

	// CurrentProfile is the profile conditions such as returns: and ending: read. The game points
	// it at the loaded profile; until then it is empty.
	CurrentProfile = &Profile{}
)

func SocietyByID(id string) *Society {
	for i := range Societies {
		if Societies[i].ID == id {
			return &Societies[i]
		}
	}
	return nil
}

// SocietyKnown reports whether id can hold a site: a society in the data, or the player's own.
func SocietyKnown(id string) bool { return id == OUR_SOCIETY || SocietyByID(id) != nil }

// SocietyName is what the player reads for a holder.
func SocietyName(id string) string {
	if s := SocietyByID(id); s != nil {
		return s.Name
	}
	return id
}

// Allegiances are the societies that can be joined at the lock: the branches of the campaign.
func Allegiances() []string {
	var out []string
	for _, s := range Societies {
		if s.Allegiance {
			out = append(out, s.ID)
		}
	}
	return out
}

func CompanionByID(id string) *Companion {
	for i := range Companions {
		if Companions[i].ID == id {
			return &Companions[i]
		}
	}
	return nil
}

func EndingByID(id string) *Ending {
	for i := range Story.Endings {
		if Story.Endings[i].ID == id {
			return &Story.Endings[i]
		}
	}
	return nil
}

// linkCompanions fills in a companion's passive color and rite, as linkOrrery does for nodes.
func linkCompanions(list []Companion) (problems []string) {
	for i := range list {
		c := &list[i]
		c.spell = nil
		if s := SpellByID(c.Rite); s != nil {
			copied := *s
			if c.Cooldown > 0 {
				copied.Cooldown = c.Cooldown
			}
			c.spell = &copied
		}
		if p := c.Passive; p != nil {
			p.Node, p.Perk.Color, p.ToMana = "companion:"+c.ID, manaByName(p.Color), manaByName(p.To)
		}
	}
	return nil
}

func checkSocieties() []string {
	var p []string
	bad := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }
	seen := map[string]bool{}
	for _, s := range Societies {
		where := "society " + s.ID
		switch {
		case s.ID == "":
			bad("societies.yaml: a society has no id")
		case s.ID == OUR_SOCIETY:
			bad("%s: %q is the player's own society and can't be defined", where, OUR_SOCIETY)
		case seen[s.ID]:
			bad("%s: defined twice", where)
		}
		seen[s.ID] = true
		for _, c := range s.Hostile {
			if strings.Contains(c, "hostile:") {
				// it would ask itself
				bad("%s: hostile conditions can't test hostile:", where)
			} else if msg := checkCond(c); msg != "" {
				bad("%s: %s", where, msg)
			}
		}
	}
	return p
}

func checkCompanions() []string {
	var p []string
	bad := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }
	seen := map[string]bool{}
	for _, c := range Companions {
		where := "companion " + c.ID
		if c.ID == "" {
			bad("companions.yaml: a companion has no id")
		} else if seen[c.ID] {
			bad("%s: defined twice", where)
		}
		seen[c.ID] = true
		if c.Society != "" && SocietyByID(c.Society) == nil {
			bad("%s: unknown society %q", where, c.Society)
		}
		if (c.Rite == "") == (c.Passive == nil) {
			bad("%s: needs a rite or a passive, one of them", where)
		}
		if c.Rite != "" {
			if s := SpellByID(c.Rite); s == nil || s.Class == "" {
				bad("%s: rite %q should be a hero spell", where, c.Rite)
			}
		}
		if c.Cooldown < 0 {
			bad("%s: cooldown %d should be 0 or more", where, c.Cooldown)
		}
		if pk := c.Passive; pk != nil {
			if pk.Perk.Color < 0 {
				bad("%s: passive color %q should be fire, water, air, earth or aether", where, pk.Color)
			}
			for _, msg := range checkPerk(&pk.Perk) {
				bad("%s: passive: %s", where, msg)
			}
		}
	}
	return p
}

// Hostile reports whether a society is hostile to the hero now.
func (h *Hero) Hostile(id string) bool {
	s := SocietyByID(id)
	return s != nil && len(s.Hostile) > 0 && h.Meets(s.Hostile)
}

// CurrentAct is the act the story is in; a run that has never had an act step is in the first.
func (h *Hero) CurrentAct() int { return max(1, h.Act) }

// AsHeld is a location as the hero finds it now: the first of its variants that holds replaces
// the fields it names. The result is a copy; the location itself is never changed.
func (h *Hero) AsHeld(l *Location) *Location {
	out := *l
	for _, v := range l.Variants {
		if (v.Holder != "" && h.Holders[l.ID] != v.Holder) || !h.Meets(v.If) {
			continue
		}
		if v.Desc != "" {
			out.Desc = v.Desc
		}
		if v.Foes != nil {
			out.Foes = v.Foes
		}
		if v.Shop != nil {
			out.Shop = v.Shop
		}
		break
	}
	return &out
}

// Sells reports whether a location's shop, as the hero finds it now, sells an item. A shop that
// lists no stock sells everything.
func (h *Hero) Sells(l *Location, item string) bool {
	stock := h.AsHeld(l).Shop
	return len(stock) == 0 || slices.Contains(stock, item)
}

// CardFor is the rule card a fight is on: the first id given that names a card. Callers pass the
// duel step's, then the quest's, then the foe's; none is the standard duel.
func CardFor(ids ...string) *RuleCard {
	for _, id := range ids {
		if c := RuleCardByID(id); c != nil {
			return c
		}
	}
	return nil
}

// Join takes an allegiance at the lock. It is permanent for the run, so a second join is ignored,
// and companions who lean to another society leave.
func (h *Hero) Join(society string) bool {
	if h.Allegiance != "" || SocietyByID(society) == nil {
		return false
	}
	h.Allegiance = society
	h.earn("lock:"+society, Grimoire.Pay.Lock)
	h.Companions = slices.DeleteFunc(h.Companions, func(id string) bool {
		c := CompanionByID(id)
		return c != nil && c.Society != "" && c.Society != society
	})
	if !slices.Contains(h.Companions, h.Companion) {
		h.Companion = ""
	}
	return true
}

// CompanionJoins adds a companion to the hero's company. One who leans to a society other than the
// hero's allegiance won't join after the lock.
func (h *Hero) CompanionJoins(id string) bool {
	c := CompanionByID(id)
	if c == nil || slices.Contains(h.Companions, id) || (h.Allegiance != "" && c.Society != "" && c.Society != h.Allegiance) {
		return false
	}
	h.Companions = append(h.Companions, id)
	return true
}

func (h *Hero) CompanionLeaves(id string) {
	h.Companions = slices.DeleteFunc(h.Companions, func(c string) bool { return c == id })
	if h.Companion == id {
		h.Companion = ""
	}
}

// Bring picks the companion who comes into fights; "" for none.
func (h *Hero) Bring(id string) bool {
	if id != "" && !slices.Contains(h.Companions, id) {
		return false
	}
	h.Companion = id
	return true
}

// FightCompanion is the companion who comes into the hero's fights, nil for none.
func (h *Hero) FightCompanion() *Companion {
	if !slices.Contains(h.Companions, h.Companion) {
		return nil
	}
	return CompanionByID(h.Companion)
}

// addCompanion gives the hero's side of a duel what the companion brings. A rite the hero already
// has equipped is replaced by the companion's, so the two never share a cooldown.
func (c *Combatant) addCompanion(comp *Companion) {
	if comp == nil {
		return
	}
	if s := comp.spell; s != nil {
		c.Spells = slices.DeleteFunc(c.Spells, func(o *Spell) bool { return o.ID == s.ID })
		c.Spells = append(c.Spells, s)
	}
	if p := comp.Passive; p != nil {
		c.Perks = append(c.Perks, &p.Perk)
	}
}

// Applied is what a step did that the game has to show or save.
type Applied struct {
	Levels	int		// levels gained
	Act		*Act	// the title card of an act the story just moved into
	ActN	int
	Ending	string	// an ending recorded on the profile; save the profile
	Banked	bool	// an act ended or the story did: meta currency moved to the profile; save it
}

// Apply makes a scene step's changes to the hero: flags, rewards, quests, places, holders,
// standing, allegiance, acts, endings and companions. Presentation and flow are the scene
// player's. Conditions (if) are the caller's to check first.
func (h *Hero) Apply(st *Step) (a Applied) {
	if st.SetFlag != "" {
		h.Flags[st.SetFlag] = true
	}
	if st.ClearFlag != "" {
		delete(h.Flags, st.ClearFlag)
	}
	if st.Gold != 0 {
		h.Gold = max(0, h.Gold+st.Gold)
	}
	if st.XP > 0 {
		a.Levels = h.AddXP(st.XP)
	}
	h.GiveItem(st.Item)
	if q := QuestByID(st.AcceptQuest); q != nil {
		if _, active := h.Active[q.ID]; !active && !slices.Contains(h.Done, q.ID) {
			h.Accept(q)
		}
	}
	if st.Unlock != "" {
		h.Flags["unlocked:"+st.Unlock] = true
	}
	if st.Travel != "" && LocationByID(st.Travel) != nil {
		h.Location = st.Travel
	}
	if hs := st.SetHolder; hs != nil && LocationByID(hs.At) != nil {
		h.SetHolder(hs.At, hs.Holder)
	}
	for soc, n := range st.Standing {
		if h.Standing == nil {
			h.Standing = map[string]int{}
		}
		h.Standing[soc] += n
	}
	if st.JoinSociety != "" {
		h.Join(st.JoinSociety)
	}
	for c, n := range st.Favour {
		if h.Favour == nil {
			h.Favour = map[string]int{}
		}
		h.Favour[c] += n
	}
	if st.CompanionJoins != "" {
		h.CompanionJoins(st.CompanionJoins)
	}
	if st.CompanionLeaves != "" {
		h.CompanionLeaves(st.CompanionLeaves)
	}
	if st.Act > 0 && st.Act != h.CurrentAct() {
		// the act before it is over: what it earned is safe from a return now
		h.bank()
		a.Banked = true
		h.Act = st.Act
		a.ActN = st.Act
		a.Act = &Act{ Title: text.Tf("ui.act_title", "Act %d", st.Act) }
		if st.Act <= len(Story.Acts) {
			a.Act = &Story.Acts[st.Act-1]
		}
	}
	if st.End || st.Ending != "" {
		h.finish(st.Ending)	// pays each first once a run, so a second end step is harmless
		a.Banked = true
		h.Finished = true
	}
	if st.Ending != "" {
		if !slices.Contains(CurrentProfile.Endings, st.Ending) {
			CurrentProfile.Endings = append(CurrentProfile.Endings, st.Ending)
		}
		a.Ending = st.Ending
	}
	return a
}
