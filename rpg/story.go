package rpg

import (
	"fmt"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// The story (world, foes, quests, characters, scenes and rule cards) is data, loaded from YAML files in
// assets/story. See docs/story.md for the format.

// StoryInfo is story.yaml: the pieces of the story that aren't a list of things.
type StoryInfo struct {
	Title		string	`yaml:"title"`
	MapName		string	`yaml:"map_name"`
	Start		string	`yaml:"start"`		// location a new hero starts in
	Blurb		string	`yaml:"blurb"`		// shown beside the class picker
	Opening		string	`yaml:"opening"`	// scene played when a hero is created
	FinalQuest	string	`yaml:"final_quest"`	// finishing it marks the story beaten
	Acts		[]Act	`yaml:"acts"`		// title cards for act 1, 2, ...; optional
	Endings		[]Ending	`yaml:"endings"`	// the endings an ending step can record on the profile
}

// Act is the title card shown when an act step moves the story into that act.
type Act struct {
	Title		string	`yaml:"title"`
	Subtitle	string	`yaml:"subtitle"`
}

// Ending is one way the story can end. Its id is stored on the profile, so it outlasts the run.
type Ending struct {
	ID		string	`yaml:"id"`
	Name	string	`yaml:"name"`
}

// Foe is a monster template. Stats grow by Growth/2 per level past the first.
type Foe struct {
	ID			string			`yaml:"id"`
	Name		string			`yaml:"name"`
	Shape		string			`yaml:"shape"`	// drawn when there's no image: blob, beast, humanoid, ghost, brute, dragon
	Color		[3]uint8		`yaml:"color,flow"`
	Image		string			`yaml:"image"`	// optional picture in assets/story/foes
	HPPct		int				`yaml:"hp_pct"`
	Spells		[]string		`yaml:"spells,flow"`
	Card		string			`yaml:"card"`	// rule card the foe fights on, default the standard duel
	RawStats	map[string]int	`yaml:"stats,flow"`
	RawGrowth	map[string]int	`yaml:"growth,flow"`
	Stats		[NUM_STATS]int	`yaml:"-"`
	Growth		[NUM_STATS]int	`yaml:"-"`
}

// Location is a stop on the world map. Roaming foes can be fought there, and roads between
// locations can be ambushed by the destination's foes.
type Location struct {
	ID			string		`yaml:"id"`
	Name		string		`yaml:"name"`
	Town		bool		`yaml:"town"`
	Open		bool		`yaml:"open"`		// on the map from the start
	Requires	[]string	`yaml:"requires,flow"`	// or opened once these hold
	X			float32		`yaml:"x"`			// 0..1 on the map
	Y			float32		`yaml:"y"`
	Level		int			`yaml:"level"`		// roaming foes' level
	Residue		string		`yaml:"residue"`	// color the site's chapters pay residue in: fire, water, air or earth
	Desc		string		`yaml:"desc"`
	Roads		[]string	`yaml:"roads,flow"`
	Foes		[]string	`yaml:"foes,flow"`
	Shop		[]string	`yaml:"shop,flow"`	// items the shop sells; none means every item
	Variants	[]Variant	`yaml:"variants"`	// the site as a holder, or a hostile society, changes it
	EnterScene	string		`yaml:"enter_scene"`	// played the first time the hero arrives
	People		[]Person	`yaml:"people"`
}

// Variant replaces some of a location's fields while it holds: when Holder holds the site (if
// set) and the If conditions hold (if any). The first variant that holds wins, and the fields it
// leaves out keep the location's own.
type Variant struct {
	Holder	string		`yaml:"holder"`	// a society id, or academy
	If		[]string	`yaml:"if,flow"`
	Desc	string		`yaml:"desc"`
	Foes	[]string	`yaml:"foes,flow"`
	Shop	[]string	`yaml:"shop,flow"`
}

// Person is someone to talk to at a location: a button that plays a scene.
type Person struct {
	Name		string		`yaml:"name"`
	Scene		string		`yaml:"scene"`
	Requires	[]string	`yaml:"requires,flow"`
}

// Quest asks for Count wins over a foe at a location. It is offered in the Giver town, and
// finishing it pays out and opens new locations.
type Quest struct {
	ID			string		`yaml:"id"`
	Title		string		`yaml:"title"`
	Main		bool		`yaml:"main"`
	Giver		string		`yaml:"giver"`
	Requires	[]string	`yaml:"requires,flow"`
	At			string		`yaml:"at"`
	Foe			string		`yaml:"foe"`
	FoeLevel	int			`yaml:"level"`
	Count		int			`yaml:"count"`
	Gold		int			`yaml:"gold"`
	XP			int			`yaml:"xp"`
	Item		string		`yaml:"item"`
	Card		string		`yaml:"card"`	// rule card its fights are on; wins over the foe's
	Opens		[]string	`yaml:"opens,flow"`
	Intro		string		`yaml:"intro"`
	Outro		string		`yaml:"outro"`
	OfferScene	string		`yaml:"offer_scene"`	// replaces the plain offer box; accept with accept_quest
	DoneScene	string		`yaml:"done_scene"`		// played after the quest is complete
}

// Character is a speaker in scenes.
type Character struct {
	ID			string		`yaml:"id"`
	Name		string		`yaml:"name"`
	Color		[3]uint8	`yaml:"color,flow"`
	Portrait	string		`yaml:"portrait"`	// image name in assets/story/portraits, default the ID
}

// Step is one line of a scene. Each step does one thing, picked by which fields are set.
type Step struct {
	// a line of dialogue; no Who means narration
	Who			string		`yaml:"who"`
	Mood		string		`yaml:"mood"`
	Text		string		`yaml:"text"`
	Choices		[]Choice	`yaml:"choices"`

	// presentation
	Background	string		`yaml:"background"`	// image in assets/story/backgrounds, or "none"
	FadeIn		float32		`yaml:"fade_in"`	// seconds
	FadeOut		float32		`yaml:"fade_out"`
	Wait		float32		`yaml:"wait"`
	Title		string		`yaml:"title"`		// full-screen title card
	Subtitle	string		`yaml:"subtitle"`

	// flow
	Goto		string		`yaml:"goto"`	// continue with another scene instead
	Run			string		`yaml:"run"`		// play another scene, then come back
	If			[]string	`yaml:"if,flow"`
	Then		[]Step		`yaml:"then"`
	Else		[]Step		`yaml:"else"`
	Duel		*DuelStep	`yaml:"duel"`
	Puzzle		*PuzzleStep	`yaml:"puzzle"`	// play a puzzle board, then a scene for the route taken
	End			bool		`yaml:"end"`	// the story is beaten

	// changes to the hero
	SetFlag		string		`yaml:"set_flag"`
	ClearFlag	string		`yaml:"clear_flag"`
	Gold		int			`yaml:"gold"`
	XP			int			`yaml:"xp"`
	Item		string		`yaml:"item"`
	AcceptQuest	string		`yaml:"accept_quest"`
	Unlock		string		`yaml:"unlock"`	// open a location
	Travel		string		`yaml:"travel"`	// move the hero there
	SetHolder	*HolderStep	`yaml:"set_holder"`	// change who holds a site
	Standing	map[string]int	`yaml:"standing,flow"`	// society → change in standing
	JoinSociety	string		`yaml:"join_society"`	// the lock: take an allegiance for the rest of the run
	Act			int			`yaml:"act"`		// move the story into this act, with its title card
	Ending		string		`yaml:"ending"`		// the story is beaten this way; recorded on the profile
	Favour		map[string]int	`yaml:"favour,flow"`	// companion → change in favour
	CompanionJoins	string	`yaml:"companion_joins"`
	CompanionLeaves	string	`yaml:"companion_leaves"`
}

// PuzzleStep plays a puzzle board. Written as just the puzzle id, every route carries on with
// the next step. Written out, a route may go to its own scene; routes it doesn't list carry on.
type PuzzleStep struct {
	ID		string				`yaml:"id"`
	Routes	map[string]string	`yaml:"routes,flow"`	// route id → scene
	Lose	string				`yaml:"lose"`	// scene when the board is left unsolved; default carries on
}

func (p *PuzzleStep) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		return n.Decode(&p.ID)
	}
	type plain PuzzleStep
	return n.Decode((*plain)(p))
}

// HolderStep says who holds a location from now on. An empty holder means no one does; the
// player's own society is "academy".
type HolderStep struct {
	At		string	`yaml:"at"`
	Holder	string	`yaml:"holder"`
}

type Choice struct {
	Text	string		`yaml:"text"`
	If		[]string	`yaml:"if,flow"`	// only offered when these hold
	Do		[]Step		`yaml:"do"`
	Goto	string		`yaml:"goto"`
}

type DuelStep struct {
	Foe		string	`yaml:"foe"`
	Level	int		`yaml:"level"`
	Win		string	`yaml:"win"`		// scene to go to on a win; default carries on
	Lose	string	`yaml:"lose"`	// scene to go to on a loss; default ends the scene
	Card	string	`yaml:"card"`	// rule card; wins over the quest's and the foe's
}

var (
	Story		StoryInfo
	Foes		[]Foe
	Locations	[]Location
	Quests		[]Quest
	Characters	[]Character
	Scenes		= map[string][]Step{}
)

func FoeByID(id string) *Foe {
	for i := range Foes {
		if Foes[i].ID == id {
			return &Foes[i]
		}
	}
	return nil
}

func LocationByID(id string) *Location {
	for i := range Locations {
		if Locations[i].ID == id {
			return &Locations[i]
		}
	}
	return nil
}

func QuestByID(id string) *Quest {
	for i := range Quests {
		if Quests[i].ID == id {
			return &Quests[i]
		}
	}
	return nil
}

func CharacterByID(id string) *Character {
	for i := range Characters {
		if Characters[i].ID == id {
			return &Characters[i]
		}
	}
	return nil
}

// Files lets the loader read from disk, the embedded copy, or test data alike.
type Files interface {
	ReadFile(name string) ([]byte, error)
	ReadDir(name string) ([]string, error)	// file names in a directory
}

type storyData struct {
	info		StoryInfo
	foes		[]Foe
	locations	[]Location
	quests		[]Quest
	characters	[]Character
	scenes		map[string][]Step
	cards		[]RuleCard
	orrery		[]OrreryNode
	facilities	[]Facility
	puzzles		[]Puzzle
	societies	[]Society
	companions	[]Companion
}

// LoadStory reads the story from dir and makes it current. A file that can't be read or parsed is
// an error and leaves the old story in place. Problems are content mistakes, like a quest naming a
// foe that doesn't exist; the story still loads with them, but they should be fixed.
func LoadStory(fs Files, dir string) (problems []string, err error) {
	var s storyData
	parse := func(name string, out any, required bool) error {
		data, err := fs.ReadFile(path.Join(dir, name))
		if err != nil {
			if required {
				return fmt.Errorf("reading %s: %w", name, err)
			}
			return nil
		}
		if err := yaml.Unmarshal(data, out); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	for _, f := range []struct {
		name		string
		out			any
		required	bool
	}{
		{ "story.yaml", &s.info, true },
		{ "foes.yaml", &s.foes, true },
		{ "locations.yaml", &s.locations, true },
		{ "quests.yaml", &s.quests, true },
		{ "characters.yaml", &s.characters, false },
		{ "cards.yaml", &s.cards, false },
		{ "orrery.yaml", &s.orrery, false },
		{ "facilities.yaml", &s.facilities, false },
		{ "societies.yaml", &s.societies, false },
		{ "companions.yaml", &s.companions, false },
	} {
		if err := parse(f.name, f.out, f.required); err != nil {
			return nil, err
		}
	}

	// every file in scenes/ adds its scenes
	s.scenes = map[string][]Step{}
	files, _ := fs.ReadDir(path.Join(dir, "scenes"))
	sort.Strings(files)
	for _, name := range files {
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		var scenes map[string][]Step
		if err := parse(path.Join("scenes", name), &scenes, true); err != nil {
			return nil, err
		}
		for id, steps := range scenes {
			if _, dup := s.scenes[id]; dup {
				problems = append(problems, fmt.Sprintf("scene %q is defined twice (again in scenes/%s)", id, name))
			}
			s.scenes[id] = steps
		}
	}

	// every file in puzzles/ is one puzzle board
	files, _ = fs.ReadDir(path.Join(dir, "puzzles"))
	sort.Strings(files)
	for _, name := range files {
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		var pz Puzzle
		if err := parse(path.Join("puzzles", name), &pz, true); err != nil {
			return nil, err
		}
		s.puzzles = append(s.puzzles, pz)
	}

	for i := range s.foes {
		f := &s.foes[i]
		var bad []string
		f.Stats, bad = statArray(f.RawStats)
		problems = append(problems, prefix("foe "+f.ID+" stats", bad)...)
		f.Growth, bad = statArray(f.RawGrowth)
		problems = append(problems, prefix("foe "+f.ID+" growth", bad)...)
		if f.HPPct == 0 {
			f.HPPct = 100
		}
	}
	problems = append(problems, linkOrrery(s.orrery)...)
	problems = append(problems, linkCompanions(s.companions)...)
	for i := range s.quests {
		if s.quests[i].Count == 0 {
			s.quests[i].Count = 1
		}
	}

	if len(s.foes) == 0 || len(s.locations) == 0 {
		return nil, fmt.Errorf("the story needs at least one foe and one location")
	}
	Story, Foes, Locations, Quests, Characters, Scenes, RuleCards, Orrery, Puzzles, Facilities = s.info, s.foes, s.locations, s.quests, s.characters, s.scenes, s.cards, s.orrery, s.puzzles, s.facilities
	Societies, Companions = s.societies, s.companions
	if LocationByID(Story.Start) == nil {
		problems = append(problems, fmt.Sprintf("story.yaml: start location %q doesn't exist, using %q", Story.Start, Locations[0].ID))
		Story.Start = Locations[0].ID
	}
	return append(problems, Validate()...), nil
}

func prefix(p string, list []string) []string {
	for i := range list {
		list[i] = p + ": " + list[i]
	}
	return list
}

func statArray(m map[string]int) (out [NUM_STATS]int, bad []string) {
	for name, n := range m {
		found := false
		for i, s := range StatNames {
			if strings.EqualFold(s, name) {
				out[i], found = n, true
			}
		}
		if !found {
			bad = append(bad, fmt.Sprintf("unknown stat %q", name))
		}
	}
	return out, bad
}

// Validate checks that everything the story refers to exists.
func Validate() []string {
	var p []string
	bad := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }
	scene := func(where, id string) {
		if id != "" && Scenes[id] == nil {
			bad("%s: unknown scene %q", where, id)
		}
	}
	conds := func(where string, list []string) {
		for _, c := range list {
			if msg := checkCond(c); msg != "" {
				bad("%s: %s", where, msg)
			}
		}
	}
	scene("story.yaml opening", Story.Opening)
	if Story.FinalQuest != "" && QuestByID(Story.FinalQuest) == nil {
		bad("story.yaml: unknown final_quest %q", Story.FinalQuest)
	}

	p = append(p, checkCards()...)
	p = append(p, checkOrrery()...)
	p = append(p, checkPuzzles()...)
	p = append(p, checkFacilities()...)
	p = append(p, checkSocieties()...)
	p = append(p, checkCompanions()...)
	for i, e := range Story.Endings {
		if e.ID == "" || slices.ContainsFunc(Story.Endings[:i], func(o Ending) bool { return o.ID == e.ID }) {
			bad("story.yaml: ending %d needs an id of its own", i+1)
		}
	}
	for _, f := range Foes {
		for _, s := range f.Spells {
			if SpellByID(s) == nil {
				bad("foe %s: unknown spell %q", f.ID, s)
			}
		}
		if f.Card != "" && RuleCardByID(f.Card) == nil {
			bad("foe %s: unknown card %q", f.ID, f.Card)
		}
	}
	for _, l := range Locations {
		where := "location " + l.ID
		for _, f := range l.Foes {
			if FoeByID(f) == nil {
				bad("%s: unknown foe %q", where, f)
			}
		}
		for _, r := range l.Roads {
			other := LocationByID(r)
			if other == nil {
				bad("%s: road to unknown location %q", where, r)
			} else if !slices.Contains(other.Roads, l.ID) {
				bad("%s: road to %s is one-way, add %s to its roads", where, r, l.ID)
			}
		}
		if manaByName(l.Residue) < 0 {
			bad("%s: residue %q should be fire, water, air or earth", where, l.Residue)
		}
		conds(where, l.Requires)
		items := func(where string, list []string) {
			for _, id := range list {
				if ItemByID(id) == nil {
					bad("%s: shop sells unknown item %q", where, id)
				}
			}
		}
		items(where, l.Shop)
		for i, v := range l.Variants {
			w := fmt.Sprintf("%s variant %d", where, i+1)
			if v.Holder != "" && !SocietyKnown(v.Holder) {
				bad("%s: unknown holder %q (add it to societies.yaml)", w, v.Holder)
			}
			for _, f := range v.Foes {
				if FoeByID(f) == nil {
					bad("%s: unknown foe %q", w, f)
				}
			}
			items(w, v.Shop)
			conds(w, v.If)
		}
		scene(where+" enter_scene", l.EnterScene)
		for _, person := range l.People {
			scene(where+" person "+person.Name, person.Scene)
			conds(where+" person "+person.Name, person.Requires)
		}
	}
	for _, q := range Quests {
		where := "quest " + q.ID
		if FoeByID(q.Foe) == nil {
			bad("%s: unknown foe %q", where, q.Foe)
		}
		if LocationByID(q.At) == nil {
			bad("%s: unknown location %q in at", where, q.At)
		}
		if g := LocationByID(q.Giver); g == nil {
			bad("%s: unknown giver %q", where, q.Giver)
		}
		if q.Item != "" && ItemByID(q.Item) == nil {
			bad("%s: unknown item %q", where, q.Item)
		}
		if q.Card != "" && RuleCardByID(q.Card) == nil {
			bad("%s: unknown card %q", where, q.Card)
		}
		for _, o := range q.Opens {
			if LocationByID(o) == nil {
				bad("%s: opens unknown location %q", where, o)
			}
		}
		conds(where, q.Requires)
		scene(where+" offer_scene", q.OfferScene)
		scene(where+" done_scene", q.DoneScene)
	}

	ids := make([]string, 0, len(Scenes))
	for id := range Scenes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		p = append(p, checkSteps("scene "+id, Scenes[id])...)
	}
	return p
}

func checkSteps(where string, steps []Step) []string {
	var p []string
	bad := func(format string, a ...any) { p = append(p, fmt.Sprintf(where+": "+format, a...)) }
	ref := func(id string) {
		if id != "" && Scenes[id] == nil {
			bad("unknown scene %q", id)
		}
	}
	for i, s := range steps {
		if s.Who != "" && s.Who != "hero" && CharacterByID(s.Who) == nil {
			bad("step %d: unknown character %q (add it to characters.yaml)", i+1, s.Who)
		}
		ref(s.Goto)
		ref(s.Run)
		for _, c := range s.If {
			if msg := checkCond(c); msg != "" {
				bad("step %d: %s", i+1, msg)
			}
		}
		p = append(p, checkSteps(fmt.Sprintf("%s step %d then", where, i+1), s.Then)...)
		p = append(p, checkSteps(fmt.Sprintf("%s step %d else", where, i+1), s.Else)...)
		for j, c := range s.Choices {
			ref(c.Goto)
			for _, cond := range c.If {
				if msg := checkCond(cond); msg != "" {
					bad("step %d choice %d: %s", i+1, j+1, msg)
				}
			}
			p = append(p, checkSteps(fmt.Sprintf("%s step %d choice %d", where, i+1, j+1), c.Do)...)
		}
		if d := s.Duel; d != nil {
			if FoeByID(d.Foe) == nil {
				bad("step %d: duel with unknown foe %q", i+1, d.Foe)
			}
			ref(d.Win)
			ref(d.Lose)
			if d.Card != "" && RuleCardByID(d.Card) == nil {
				bad("step %d: duel on unknown card %q", i+1, d.Card)
			}
		}
		if pz := s.Puzzle; pz != nil {
			if p := PuzzleByID(pz.ID); p == nil {
				bad("step %d: unknown puzzle %q", i+1, pz.ID)
			} else {
				for route, sc := range pz.Routes {
					if !slices.ContainsFunc(p.Routes, func(r PuzzleRoute) bool { return r.ID == route }) {
						bad("step %d: puzzle %s has no route %q", i+1, pz.ID, route)
					}
					ref(sc)
				}
			}
			ref(pz.Lose)
		}
		for soc := range s.Standing {
			if SocietyByID(soc) == nil {
				bad("step %d: standing with unknown society %q", i+1, soc)
			}
		}
		if s.JoinSociety != "" {
			if soc := SocietyByID(s.JoinSociety); soc == nil || !soc.Allegiance {
				bad("step %d: join_society %q should be a society with allegiance: true", i+1, s.JoinSociety)
			}
		}
		if s.Act < 0 {
			bad("step %d: act %d should be 1 or more", i+1, s.Act)
		}
		if s.Ending != "" && EndingByID(s.Ending) == nil {
			bad("step %d: unknown ending %q (add it to endings in story.yaml)", i+1, s.Ending)
		}
		for c := range s.Favour {
			if CompanionByID(c) == nil {
				bad("step %d: favour with unknown companion %q", i+1, c)
			}
		}
		for _, c := range []string{ s.CompanionJoins, s.CompanionLeaves } {
			if c != "" && CompanionByID(c) == nil {
				bad("step %d: unknown companion %q", i+1, c)
			}
		}
		if s.Item != "" && ItemByID(s.Item) == nil {
			bad("step %d: unknown item %q", i+1, s.Item)
		}
		if s.AcceptQuest != "" && QuestByID(s.AcceptQuest) == nil {
			bad("step %d: unknown quest %q", i+1, s.AcceptQuest)
		}
		if h := s.SetHolder; h != nil {
			if LocationByID(h.At) == nil {
				bad("step %d: set_holder at unknown location %q", i+1, h.At)
			}
			if h.Holder != "" && !SocietyKnown(h.Holder) {
				bad("step %d: holder %q is not a society (add it to societies.yaml)", i+1, h.Holder)
			}
		}
		for _, l := range []string{ s.Unlock, s.Travel } {
			if l != "" && LocationByID(l) == nil {
				bad("step %d: unknown location %q", i+1, l)
			}
		}
	}
	return p
}

// Conditions are strings like "quest:goblins" or "!flag:met_mayor". Kinds:
//   quest:ID   quest done        active:ID  quest in progress
//   flag:NAME  flag set          seen:ID    scene played before
//   item:ID    item owned        level:N    hero level at least N
//   class:NAME hero's class      at:ID      hero is at a location
//   act:N      act at least N    standing:SOCIETY>=N  standing compared with a number or a society's
//   holder:LOC=SOCIETY  who holds a site ("holder:LOC=" for no one)
//   hostile:SOCIETY     the society is hostile to the hero
//   allegiance:SOCIETY  the society joined at the lock ("allegiance:none" before it)
//   companion:ID        in the hero's company     favour:ID>=N  a companion's favour
//   returns:N  runs started over through the return, at least N (the profile)
//   ending:ID  the profile has seen that ending
// Numbers take a comparison in front, as in "act:<3" or "returns:=0"; without one, at least.
func checkCond(c string) string {
	kind, arg, ok := strings.Cut(strings.TrimPrefix(c, "!"), ":")
	if !ok || arg == "" {
		return fmt.Sprintf("condition %q should look like kind:value", c)
	}
	switch kind {
	case "quest", "active":
		if QuestByID(arg) == nil {
			return fmt.Sprintf("condition %q: unknown quest", c)
		}
	case "item":
		if ItemByID(arg) == nil {
			return fmt.Sprintf("condition %q: unknown item", c)
		}
	case "at":
		if LocationByID(arg) == nil {
			return fmt.Sprintf("condition %q: unknown location", c)
		}
	case "seen":
		if Scenes[arg] == nil {
			return fmt.Sprintf("condition %q: unknown scene", c)
		}
	case "level", "act", "returns":
		if _, _, ok := numberCond(arg); !ok {
			return fmt.Sprintf("condition %q: %s needs a number, optionally after >=, <=, >, < or =", c, kind)
		}
	case "class":
		if ClassByName(arg) == nil {
			return fmt.Sprintf("condition %q: unknown class", c)
		}
	case "standing", "favour":
		who, op, rhs, ok := splitCompare(arg)
		if !ok {
			return fmt.Sprintf("condition %q: should compare, as in %s:id>=3", c, kind)
		}
		known := func(id string) bool { return SocietyByID(id) != nil }
		if kind == "favour" {
			known = func(id string) bool { return CompanionByID(id) != nil }
		}
		if !known(who) {
			return fmt.Sprintf("condition %q: unknown %s %q", c, map[string]string{ "standing": "society", "favour": "companion" }[kind], who)
		}
		if _, err := strconv.Atoi(rhs); err != nil && (kind == "favour" || !known(rhs)) {
			return fmt.Sprintf("condition %q: %s%s should be followed by a number%s", c, who, op, map[bool]string{ true: " or a society" }[kind == "standing"])
		}
	case "holder":
		loc, holder, ok := strings.Cut(arg, "=")
		if !ok || LocationByID(loc) == nil {
			return fmt.Sprintf("condition %q: should be holder:location=society", c)
		}
		if holder != "" && !SocietyKnown(holder) {
			return fmt.Sprintf("condition %q: unknown society %q", c, holder)
		}
	case "hostile":
		if SocietyByID(arg) == nil {
			return fmt.Sprintf("condition %q: unknown society", c)
		}
	case "allegiance":
		if arg != "none" && SocietyByID(arg) == nil {
			return fmt.Sprintf("condition %q: unknown society", c)
		}
	case "companion":
		if CompanionByID(arg) == nil {
			return fmt.Sprintf("condition %q: unknown companion", c)
		}
	case "ending":
		if EndingByID(arg) == nil {
			return fmt.Sprintf("condition %q: unknown ending (add it to endings in story.yaml)", c)
		}
	case "flag":
	default:
		return fmt.Sprintf("condition %q: unknown kind %q", c, kind)
	}
	return ""
}

var compareOps = []string{ ">=", "<=", ">", "<", "=" }

// splitCompare splits "crown>=3" into its sides and the comparison.
func splitCompare(s string) (lhs, op, rhs string, ok bool) {
	i := strings.IndexAny(s, "<>=")
	if i <= 0 {
		return "", "", "", false
	}
	for _, o := range compareOps {
		if strings.HasPrefix(s[i:], o) {
			lhs, op, rhs = s[:i], o, s[i+len(o):]
			return lhs, op, rhs, rhs != ""
		}
	}
	return "", "", "", false
}

// numberCond reads "3" (at least 3) or "<3", "=0" and the like.
func numberCond(s string) (op string, n int, ok bool) {
	op = ">="
	for _, o := range compareOps {
		if strings.HasPrefix(s, o) {
			op, s = o, s[len(o):]
			break
		}
	}
	n, err := strconv.Atoi(s)
	return op, n, err == nil
}

func compare(a int, op string, b int) bool {
	switch op {
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case "<":
		return a < b
	}
	return a == b
}

// Meets reports whether every condition holds for the hero.
func (h *Hero) Meets(conds []string) bool {
	for _, c := range conds {
		if !h.meets(c) {
			return false
		}
	}
	return true
}

func (h *Hero) meets(c string) bool {
	neg := strings.HasPrefix(c, "!")
	kind, arg, _ := strings.Cut(strings.TrimPrefix(c, "!"), ":")
	num := func(have int) bool {
		op, n, _ := numberCond(arg)
		return compare(have, op, n)
	}
	var ok bool
	switch kind {
	case "quest":
		ok = slices.Contains(h.Done, arg)
	case "active":
		_, ok = h.Active[arg]
	case "flag":
		ok = h.Flags[arg]
	case "seen":
		ok = h.Seen[arg]
	case "item":
		ok = slices.Contains(h.Owned, arg)
	case "level":
		ok = num(h.Level)
	case "class":
		ok = strings.EqualFold(h.Class, arg)
	case "at":
		ok = h.Location == arg
	case "act":
		ok = num(h.CurrentAct())
	case "standing", "favour":
		who, op, rhs, _ := splitCompare(arg)
		counts := h.Standing
		if kind == "favour" {
			counts = h.Favour
		}
		n, err := strconv.Atoi(rhs)
		if err != nil {
			n = counts[rhs]	// another society's standing
		}
		ok = compare(counts[who], op, n)
	case "holder":
		loc, holder, _ := strings.Cut(arg, "=")
		ok = h.Holders[loc] == holder
	case "hostile":
		ok = h.Hostile(arg)
	case "allegiance":
		ok = h.Allegiance == arg || (arg == "none" && h.Allegiance == "")
	case "companion":
		ok = slices.Contains(h.Companions, arg)
	case "returns":
		ok = num(CurrentProfile.Returns)
	case "ending":
		ok = slices.Contains(CurrentProfile.Endings, arg)
	}
	return ok != neg
}

// GiveItem adds an item to the hero's inventory once.
func (h *Hero) GiveItem(id string) {
	if id != "" && !slices.Contains(h.Owned, id) {
		h.Owned = append(h.Owned, id)
	}
}
