package rpg

import (
	"fmt"
	"slices"

	"jewel-game/text"
)

// The academy: facilities built and upgraded with gold, henchmen hired to staff them, and field
// chapters those henchmen run at sites the hero has open. Nothing here moves on a clock: chapters
// and dues only advance when the hero wins a duel.

// Placeholder tuning for the academy prototype. Facility costs and perks are in facilities.yaml.
const (
	FOUNDING_FLAG		= "academy_founded"	// set by a scene; opens the academy
	OUR_SOCIETY			= "academy"			// the holder of a site the player's chapters took
	HENCHMEN_CAP		= 1		// henchmen before any facility raises the cap
	HIRE_GOLD			= 60	// gold to hire a henchman, at rank 1
	TRAIN_GOLD			= 50	// gold per rank to train a henchman up from that rank
	MAX_RANK			= 5
	CHAPTER_WINS		= 2		// hero wins at the posted site for its chapter to complete
	CHAPTER_GOLD		= 25	// dues a completed chapter pays per point of power
	CHAPTER_RESIDUE		= 3		// residue of the site's color per point of power
	GEAR_LEVELS			= 4		// item levels in a henchman's gear slot worth one point of power
	DUES_PER_WIN		= 4		// gold banked in the ledger for each later win at a held site
)

// Facility is a building in the academy: facilities.yaml. It becomes available when Requires
// holds, and each level is bought with gold in order, adding its perks to those below it.
type Facility struct {
	ID			string		`yaml:"id"`
	Name		string		`yaml:"name"`
	Desc		string		`yaml:"desc"`
	Requires	[]string	`yaml:"requires,flow"`
	Levels		[]FacilityLevel	`yaml:"levels"`
}

// FacilityLevel is one step of a facility: its gold cost and what it adds.
type FacilityLevel struct {
	Cost		int		`yaml:"cost"`
	GoldPct		int		`yaml:"gold_pct"`
	XPPct		int		`yaml:"xp_pct"`
	HP			int		`yaml:"hp"`
	StartMana	int		`yaml:"start_mana"`
	SkullDamage	int		`yaml:"skull_damage"`
	Armor		int		`yaml:"armor"`
	Henchmen	int		`yaml:"henchmen"`	// more henchmen allowed
	ChapterPct	int		`yaml:"chapter_pct"`	// more paid by every completed chapter, in percent
}

var Facilities []Facility

func FacilityByID(id string) *Facility {
	for i := range Facilities {
		if Facilities[i].ID == id {
			return &Facilities[i]
		}
	}
	return nil
}

func checkFacilities() []string {
	var p []string
	bad := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }
	seen := map[string]bool{}
	for _, f := range Facilities {
		where := "facility " + f.ID
		if f.ID == "" {
			bad("facilities.yaml: a facility has no id")
		} else if seen[f.ID] {
			bad("%s: defined twice", where)
		}
		seen[f.ID] = true
		if len(f.Levels) == 0 {
			bad("%s: needs at least one level", where)
		}
		for i, l := range f.Levels {
			if l.Cost <= 0 {
				bad("%s: level %d needs a gold cost", where, i+1)
			}
		}
		for _, c := range f.Requires {
			if msg := checkCond(c); msg != "" {
				bad("%s: %s", where, msg)
			}
		}
	}
	return p
}

// Henchman is unnamed hired staff, saved on the run. Post is the site of the chapter they are on,
// "" when free; Report is set when the chapter is complete and waits there to be claimed.
type Henchman struct {
	Rank	int
	Gear	string	`json:",omitempty"`	// item ID, one the hero owns
	Post	string	`json:",omitempty"`
	Wins	int		`json:",omitempty"`	// toward the chapter in progress
	Report	*Report	`json:",omitempty"`
}

// Report is what a completed chapter leaves for the hero to claim.
type Report struct {
	At		string
	Line	string
	Gold	int
	Residue	[NUM_MANA]int
}

// AcademyOpen reports whether the academy has been founded.
func (h *Hero) AcademyOpen() bool { return h.Flags[FOUNDING_FLAG] }

// FacilityLevel is how many levels of a facility the hero has built, 0 for none.
func (h *Hero) FacilityLevel(id string) int { return h.Facilities[id] }

// FacilityAvailable reports whether the story has made a facility available to build.
func (h *Hero) FacilityAvailable(f *Facility) bool {
	return h.AcademyOpen() && h.Meets(f.Requires)
}

// NextCost is the gold for a facility's next level, 0 when it is at the top.
func (h *Hero) NextCost(f *Facility) int {
	if n := h.FacilityLevel(f.ID); n < len(f.Levels) {
		return f.Levels[n].Cost
	}
	return 0
}

// Build buys a facility's next level.
func (h *Hero) Build(f *Facility) bool {
	cost := h.NextCost(f)
	if !h.FacilityAvailable(f) || cost == 0 || h.Gold < cost {
		return false
	}
	h.Gold -= cost
	if h.Facilities == nil {
		h.Facilities = map[string]int{}
	}
	h.Facilities[f.ID]++
	return true
}

// facilityPerks adds up every level built of every facility.
func (h *Hero) facilityPerks() (sum FacilityLevel) {
	for i := range Facilities {
		f := &Facilities[i]
		for _, l := range f.Levels[:min(h.FacilityLevel(f.ID), len(f.Levels))] {
			sum.GoldPct += l.GoldPct
			sum.XPPct += l.XPPct
			sum.HP += l.HP
			sum.StartMana += l.StartMana
			sum.SkullDamage += l.SkullDamage
			sum.Armor += l.Armor
			sum.Henchmen += l.Henchmen
			sum.ChapterPct += l.ChapterPct
		}
	}
	return sum
}

func (h *Hero) facilityBonus() Bonus {
	p := h.facilityPerks()
	return Bonus{ HP: p.HP, SkullDamage: p.SkullDamage, Armor: p.Armor, GoldPct: p.GoldPct, XPPct: p.XPPct, StartMana: p.StartMana }
}

// HenchmenCap is how many henchmen the academy can keep.
func (h *Hero) HenchmenCap() int { return HENCHMEN_CAP + h.facilityPerks().Henchmen }

// Hire takes on a henchman at rank 1.
func (h *Hero) Hire() bool {
	if !h.AcademyOpen() || len(h.Henchmen) >= h.HenchmenCap() || h.Gold < HIRE_GOLD {
		return false
	}
	h.Gold -= HIRE_GOLD
	h.Henchmen = append(h.Henchmen, Henchman{ Rank: 1 })
	return true
}

// TrainCost is the gold to raise a henchman one rank, 0 at the top.
func TrainCost(m *Henchman) int {
	if m.Rank >= MAX_RANK {
		return 0
	}
	return TRAIN_GOLD * m.Rank
}

func (h *Hero) Train(i int) bool {
	if i < 0 || i >= len(h.Henchmen) {
		return false
	}
	m := &h.Henchmen[i]
	cost := TrainCost(m)
	if cost == 0 || h.Gold < cost {
		return false
	}
	h.Gold -= cost
	m.Rank++
	return true
}

// GearFree reports whether an owned item is free to give a henchman: not worn by the hero or held
// by another henchman.
func (h *Hero) GearFree(id string) bool {
	if !slices.Contains(h.Owned, id) {
		return false
	}
	for _, worn := range h.Gear {
		if worn == id {
			return false
		}
	}
	for _, m := range h.Henchmen {
		if m.Gear == id {
			return false
		}
	}
	return true
}

// Equip gives henchman i an item the hero owns and isn't using; "" takes their gear back.
func (h *Hero) Equip(i int, item string) bool {
	if i < 0 || i >= len(h.Henchmen) || (item != "" && !h.GearFree(item)) {
		return false
	}
	h.Henchmen[i].Gear = item
	return true
}

// Power is what a henchman brings to a chapter: their rank, plus their gear.
func Power(m *Henchman) int {
	p := m.Rank
	if it := ItemByID(m.Gear); it != nil {
		p += it.Level / GEAR_LEVELS
	}
	return p
}

// SiteColor is the residue color a site's chapters pay, Fire if the site names none.
func SiteColor(loc string) int {
	if l := LocationByID(loc); l != nil {
		if c := manaByName(l.Residue); c >= 0 {
			return c
		}
	}
	return Fire
}

// Post sends a free henchman on a chapter at a site the hero has open.
func (h *Hero) Post(i int, loc string) bool {
	l := LocationByID(loc)
	if i < 0 || i >= len(h.Henchmen) || l == nil || !h.LocationOpen(l) || h.Henchmen[i].Post != "" {
		return false
	}
	h.Henchmen[i].Post, h.Henchmen[i].Wins = loc, 0
	return true
}

// Recall brings a henchman back from a chapter still in progress; its wins so far are lost.
func (h *Hero) Recall(i int) bool {
	if i < 0 || i >= len(h.Henchmen) || h.Henchmen[i].Post == "" || h.Henchmen[i].Report != nil {
		return false
	}
	h.Henchmen[i].Post, h.Henchmen[i].Wins = "", 0
	return true
}

// Held reports whether the player's society holds a site.
func (h *Hero) Held(loc string) bool { return h.Holders[loc] == OUR_SOCIETY }

// RecordWin moves the academy along after the hero wins a duel at loc: a held site banks dues in
// the ledger, and every chapter in progress there takes a step. It returns the chapters it completed.
func (h *Hero) RecordWin(loc string) (done []*Report) {
	if h.Held(loc) {
		if h.Dues == nil {
			h.Dues = map[string]int{}
		}
		h.Dues[loc] += DUES_PER_WIN
	}
	more := h.facilityPerks().ChapterPct
	for i := range h.Henchmen {
		m := &h.Henchmen[i]
		if m.Post != loc || m.Report != nil {
			continue
		}
		m.Wins++
		if m.Wins < CHAPTER_WINS {
			continue
		}
		p := Power(m)
		r := &Report{ At: m.Post, Gold: pct(CHAPTER_GOLD*p, more) }
		r.Residue[SiteColor(m.Post)] = pct(CHAPTER_RESIDUE*p, more)
		r.Line = reportLine(m, i)
		m.Report = r
		if h.Holders == nil {
			h.Holders = map[string]string{}
		}
		h.Holders[m.Post] = OUR_SOCIETY
		done = append(done, r)
	}
	return done
}

// reportLine is the short line a chapter's report reads, picked by how good the henchman is.
func reportLine(m *Henchman, i int) string {
	at := LocationByID(m.Post).Name
	lines := []string{
		text.Tf("academy.report_1", "Rank %d staff kept the books at %s. The site is ours.", m.Rank, at),
		text.Tf("academy.report_2", "A rank %d chapter at %s took oaths from the locals.", m.Rank, at),
		text.Tf("academy.report_3", "Our rank %d hand at %s turned a rival's ledger over to us.", m.Rank, at),
	}
	return lines[(m.Rank+i)%len(lines)]
}

// Claim banks henchman i's completed report and frees them.
func (h *Hero) Claim(i int) bool {
	if i < 0 || i >= len(h.Henchmen) || h.Henchmen[i].Report == nil {
		return false
	}
	m := &h.Henchmen[i]
	h.Gold += m.Report.Gold
	for c, n := range m.Report.Residue {
		h.Residue[c] += n
	}
	m.Post, m.Wins, m.Report = "", 0, nil
	return true
}

// DuesOwed is everything in the ledger.
func (h *Hero) DuesOwed() int {
	n := 0
	for _, d := range h.Dues {
		n += d
	}
	return n
}

// ClaimDues banks the ledger and empties it.
func (h *Hero) ClaimDues() int {
	n := h.DuesOwed()
	h.Gold += n
	h.Dues = nil
	return n
}

// Old runs: buildings and mounts are gone. Their gold is refunded at what they cost.
var oldPrices = map[string]int{
	"treasury": 200, "stables": 250, "barracks": 400, "library": 500, "temple": 600, "mage_tower": 800, "war_room": 1200,
	"pony": 150, "warhorse": 500, "griffin": 1400,
}

// fromBeforeAcademy converts a run saved before the academy: buildings and mounts are refunded,
// and the academy is founded if a quest the run has done would have founded it.
func (h *Hero) fromBeforeAcademy(buildings []string) {
	for _, b := range buildings {
		h.Gold += oldPrices[b]
	}
	h.Owned = slices.DeleteFunc(h.Owned, func(id string) bool {
		if ItemByID(id) == nil {
			h.Gold += oldPrices[id]
			return true
		}
		return false
	})
	for slot, id := range h.Gear {
		if ItemByID(id) == nil || !slices.Contains(Slots, slot) {
			delete(h.Gear, slot)
		}
	}
	for _, id := range h.Done {
		if q := QuestByID(id); q != nil && setsFlag(Scenes[q.DoneScene], FOUNDING_FLAG) {
			h.Flags[FOUNDING_FLAG] = true
		}
	}
}

func setsFlag(steps []Step, flag string) bool {
	for _, s := range steps {
		if s.SetFlag == flag {
			return true
		}
	}
	return false
}

// SetHolder records who holds a site; "" means no one.
func (h *Hero) SetHolder(loc, holder string) {
	if holder == "" {
		delete(h.Holders, loc)
		return
	}
	if h.Holders == nil {
		h.Holders = map[string]string{}
	}
	h.Holders[loc] = holder
}
