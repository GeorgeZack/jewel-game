package rpg

import (
	"fmt"
	"slices"
	"strings"
)

// The return, the meta currency and the Grimoire: what a profile carries from one run to the next
// (DESIGN.md 5.8). Meta stages 1 (patronage) and 2 (the price) are built; binding and the reckoning
// are not, but the profile keeps room for them (Profile.Bound).
//
// A run earns meta currency for first times. What it earns waits on the hero, unbanked, until the
// act ends; then it moves to the profile and can't be lost. A return before that forfeits it.

// ENTITIES is the built-in speaker for the entities behind the return, like "hero". Their name is
// a line in the language file (meta.entities), so it can change without touching the story.
const ENTITIES = "entities"

// Leans of a stage 2 Grimoire entry: it takes more from the entities, or resists them.
const (
	LEAN_TAKE	= "take"
	LEAN_RESIST	= "resist"
)

// GrimoireBook is grimoire.yaml: what the meta currency pays for, what pays it, and the entries.
type GrimoireBook struct {
	StageTwoReturns	int				`yaml:"stage_two_returns"`	// returns before stage 2 entries show
	Pay				MetaPay			`yaml:"pay"`
	Entries			[]GrimoireEntry	`yaml:"entries"`
}

// MetaPay is what each kind of first time pays, and the percent a repeat in a later run pays.
type MetaPay struct {
	Boss		int	`yaml:"boss"`		// a foe with boss: true defeated
	Route		int	`yaml:"route"`		// a puzzle board solved by a route
	Lock		int	`yaml:"lock"`		// a society joined at the lock
	Class		int	`yaml:"class"`		// the story finished as a class
	Ending		int	`yaml:"ending"`		// an ending reached
	RepeatPct	int	`yaml:"repeat_pct"`
}

// GrimoireEntry is one page of the Grimoire. Bought once with meta currency, it changes the start
// of every later run. Entries buy speed and options, not raw power.
type GrimoireEntry struct {
	ID			string			`yaml:"id"`
	Name		string			`yaml:"name"`
	Desc		string			`yaml:"desc"`
	Cost		int				`yaml:"cost"`
	Stage		int				`yaml:"stage"`	// 1, or 2 for the price; 0 reads as 1
	Lean		string			`yaml:"lean"`	// stage 2: take or resist

	// what a run starts with
	Gold		int				`yaml:"gold"`
	Flags		[]string		`yaml:"flags,flow"`
	Facilities	map[string]int	`yaml:"facilities,flow"`	// facility → levels built
	Nodes		[]string		`yaml:"nodes,flow"`		// Orrery nodes owned, kept through a respec
	Quests		[]string		`yaml:"quests,flow"`		// counted done: the fight is skipped, its places open, no reward
	Residue		map[string]int	`yaml:"residue,flow"`	// fire, water, air, earth, aether or colorless
}

var Grimoire GrimoireBook

func EntryByID(id string) *GrimoireEntry {
	for i := range Grimoire.Entries {
		if Grimoire.Entries[i].ID == id {
			return &Grimoire.Entries[i]
		}
	}
	return nil
}

func residueByName(name string) int {
	if strings.EqualFold(name, "colorless") {
		return Colorless
	}
	return manaByName(name)
}

func checkGrimoire() []string {
	var p []string
	bad := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }
	for i, e := range Grimoire.Entries {
		where := "grimoire entry " + e.ID
		if e.ID == "" || slices.ContainsFunc(Grimoire.Entries[:i], func(o GrimoireEntry) bool { return o.ID == e.ID }) {
			bad("grimoire.yaml: entry %d needs an id of its own", i+1)
		}
		if e.Stage > 2 || e.Stage < 0 {
			bad("%s: stage %d should be 1 or 2 (later stages are not built)", where, e.Stage)
		}
		if e.Stage == 2 && e.Lean != LEAN_TAKE && e.Lean != LEAN_RESIST {
			bad("%s: a stage 2 entry needs lean: take or lean: resist", where)
		}
		if e.Stage < 2 && e.Lean != "" {
			bad("%s: only stage 2 entries have a lean", where)
		}
		for f := range e.Facilities {
			if FacilityByID(f) == nil {
				bad("%s: unknown facility %q", where, f)
			}
		}
		for _, n := range e.Nodes {
			if NodeByID(n) == nil {
				bad("%s: unknown Orrery node %q", where, n)
			}
		}
		for _, q := range e.Quests {
			if QuestByID(q) == nil {
				bad("%s: unknown quest %q", where, q)
			}
		}
		for r := range e.Residue {
			if residueByName(r) < 0 {
				bad("%s: residue %q should be a mana color or colorless", where, r)
			}
		}
	}
	return p
}

// StageTwo reports whether the profile has returned often enough for the price (stage 2).
func (p *Profile) StageTwo() bool { return p.Returns >= Grimoire.StageTwoReturns }

// EntryShown reports whether the Grimoire shows an entry: stage 2 entries only from stage 2.
func (p *Profile) EntryShown(e *GrimoireEntry) bool { return e.Stage < 2 || p.StageTwo() }

func (p *Profile) Owns(id string) bool { return slices.Contains(p.Grimoire, id) }

// CanBuy reports whether the profile can buy an entry now.
func (p *Profile) CanBuy(e *GrimoireEntry) bool {
	return e != nil && p.EntryShown(e) && !p.Owns(e.ID) && p.Meta >= e.Cost
}

// Buy pays for a Grimoire entry. Its effect starts with the next run. A stage 2 entry moves the
// lean: up for one that takes from the entities, down for one that resists them.
func (p *Profile) Buy(id string) bool {
	e := EntryByID(id)
	if !p.CanBuy(e) {
		return false
	}
	p.Meta -= e.Cost
	p.Grimoire = append(p.Grimoire, id)
	switch e.Lean {
	case LEAN_TAKE:
		p.Lean++
	case LEAN_RESIST:
		p.Lean--
	}
	return true
}

// LeanName is the profile's lean as the lean: condition reads it: take, resist or none.
func (p *Profile) LeanName() string {
	switch {
	case p.Lean > 0:
		return LEAN_TAKE
	case p.Lean < 0:
		return LEAN_RESIST
	}
	return "none"
}

// StartRun makes the hero of a new run: a fresh hero of the class, with what the profile's
// Grimoire gives at the start.
func StartRun(class string) *Hero {
	h := NewHero(class)
	for _, id := range CurrentProfile.Grimoire {
		if e := EntryByID(id); e != nil {
			h.applyEntry(e)
		}
	}
	return h
}

func (h *Hero) applyEntry(e *GrimoireEntry) {
	h.Gold += e.Gold
	for _, f := range e.Flags {
		h.Flags[f] = true
	}
	for f, n := range e.Facilities {
		if fac := FacilityByID(f); fac != nil {
			if h.Facilities == nil {
				h.Facilities = map[string]int{}
			}
			h.Facilities[f] = min(len(fac.Levels), max(h.Facilities[f], n))
		}
	}
	for _, id := range e.Nodes {
		if NodeByID(id) != nil && !slices.Contains(h.Gifts, id) {
			h.Gifts = append(h.Gifts, id)
			if !h.Owns(id) {
				h.Nodes = append(h.Nodes, id)
			}
		}
	}
	h.learnSpells()
	for _, q := range e.Quests {
		if QuestByID(q) != nil && !slices.Contains(h.Done, q) {
			h.Done = append(h.Done, q)
		}
	}
	for r, n := range e.Residue {
		if i := residueByName(r); i >= 0 {
			h.Residue[i] += n
		}
	}
}

// earn pays meta currency for a first time, named by key ("boss:hollow_king"). It pays once a
// run: in full the first time the profile does it, RepeatPct of that on later runs. It waits on
// the hero until the act's end banks it.
func (h *Hero) earn(key string, full int) int {
	if full <= 0 || slices.Contains(h.MetaEarned, key) {
		return 0
	}
	n := full
	if CurrentProfile.Firsts[key] > 0 {
		n = full * Grimoire.Pay.RepeatPct / 100
	}
	h.MetaEarned = append(h.MetaEarned, key)
	h.MetaPending = append(h.MetaPending, key)
	h.MetaUnbanked += n
	return n
}

// bank moves what this act earned to the profile, where a return can't take it.
func (h *Hero) bank() {
	p := CurrentProfile
	p.Meta += h.MetaUnbanked
	if len(h.MetaPending) > 0 && p.Firsts == nil {
		p.Firsts = map[string]int{}
	}
	for _, k := range h.MetaPending {
		p.Firsts[k]++
	}
	h.MetaUnbanked, h.MetaPending = 0, nil
}

// finish pays for the story finished as this class, and for an ending if it has one, then banks
// the last act.
func (h *Hero) finish(ending string) {
	h.earn("class:"+h.Class, Grimoire.Pay.Class)
	if ending != "" {
		h.earn("ending:"+ending, Grimoire.Pay.Ending)
	}
	h.bank()
}

// Forfeit is what a return now would lose: the meta currency this act has earned and not banked.
func (h *Hero) Forfeit() int {
	if h == nil {
		return 0
	}
	return h.MetaUnbanked
}

// Return ends the run, at an ending or early. The hero is to be thrown away: everything on it goes,
// including what this act earned and hasn't banked. The profile keeps the rest, counts the return,
// and the tutorial is behind it. It reports what was forfeited.
func Return(h *Hero) (forfeit int) {
	p := CurrentProfile
	forfeit = h.Forfeit()
	p.Returns++
	p.TutorialDone = true
	return forfeit
}

// RunScenes are the scenes a new run plays once its class is chosen, in order: the entities'
// scene once the profile has returned, the tutorial on its first run, then the first event. A
// hero deleted from the title screen is not a return, but its next run is still a later run.
func RunScenes() []string {
	var out []string
	if CurrentProfile.Returns > 0 && Story.ReturnScene != "" {
		out = append(out, Story.ReturnScene)
	}
	if !CurrentProfile.TutorialDone && Story.Tutorial != "" {
		out = append(out, Story.Tutorial)
	}
	return append(out, Story.FirstEventScene())
}

// FirstEventScene is the scene every run starts at: first_event, or the opening in a story from
// before the return.
func (s *StoryInfo) FirstEventScene() string {
	if s.FirstEvent != "" {
		return s.FirstEvent
	}
	return s.Opening
}
