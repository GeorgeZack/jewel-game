package rpg

import (
	"encoding/json"
	"fmt"
	"slices"
)

// LocationOpen reports whether the hero can travel to a location.
func (h *Hero) LocationOpen(l *Location) bool {
	if l.Open || h.Flags["unlocked:"+l.ID] || (len(l.Requires) > 0 && h.Meets(l.Requires)) {
		return true
	}
	for _, id := range h.Done {
		if q := QuestByID(id); q != nil && slices.Contains(q.Opens, l.ID) {
			return true
		}
	}
	return false
}

// Route is the shortest open road from the hero's location, excluding the start. Nil if none.
func (h *Hero) Route(to string) []string {
	prev := map[string]string{ h.Location: "" }
	queue := []string{ h.Location }
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			var path []string
			for ; cur != h.Location; cur = prev[cur] {
				path = append([]string{ cur }, path...)
			}
			return path
		}
		for _, next := range LocationByID(cur).Roads {
			if _, seen := prev[next]; !seen && h.LocationOpen(LocationByID(next)) {
				prev[next] = cur
				queue = append(queue, next)
			}
		}
	}
	return nil
}

// OfferedQuests lists quests a town can give the hero right now.
func (h *Hero) OfferedQuests(town string) []*Quest {
	var out []*Quest
	for i := range Quests {
		q := &Quests[i]
		_, active := h.Active[q.ID]
		if q.Giver != town || active || slices.Contains(h.Done, q.ID) {
			continue
		}
		if h.Meets(q.Requires) {
			out = append(out, q)
		}
	}
	return out
}

func (h *Hero) Accept(q *Quest) {
	h.Active[q.ID] = 0
}

// QuestsHere lists active quests with targets at a location.
func (h *Hero) QuestsHere(loc string) []*Quest {
	var out []*Quest
	for i := range Quests {
		if q := &Quests[i]; q.At == loc {
			if _, ok := h.Active[q.ID]; ok {
				out = append(out, q)
			}
		}
	}
	return out
}

// RecordKill advances quests hunting a foe at a location, paying out any it completes.
func (h *Hero) RecordKill(foe, loc string) (finished []*Quest) {
	for i := range Quests {
		q := &Quests[i]
		if _, ok := h.Active[q.ID]; !ok || q.Foe != foe || q.At != loc {
			continue
		}
		h.Active[q.ID]++
		if h.Active[q.ID] < q.Count {
			continue
		}
		delete(h.Active, q.ID)
		h.Done = append(h.Done, q.ID)
		h.Gold += q.Gold
		h.AddXP(q.XP)
		h.GiveItem(q.Item)
		if q.ID == Story.FinalQuest {
			h.Finished = true
		}
		finished = append(finished, q)
	}
	return finished
}

// SavePath names where the run is stored: a file on desktop, a localStorage key in a browser.
func SavePath() string {
	return savePath()
}

// ProfilePath names where the profile is stored, beside the run.
func ProfilePath() string {
	return profilePath()
}

// SAVE_VERSION is written into every run save. Saves from before versioning have none and hold a
// bare Hero; LoadRun migrates them. Version 1 is from before the academy, with buildings and mounts.
// Version 2 is from the 24-node prototype Orrery. Version 3 is from before Aether, with four mana
// colors in every stat and residue array.
const SAVE_VERSION = 4

// PROFILE_VERSION is written into every profile save.
const PROFILE_VERSION = 1

// Profile is what outlasts a run. It is saved apart from the run, so deleting a hero keeps it.
// Nothing fills it in yet.
type Profile struct {
	Grimoire	[]string		`json:",omitempty"`	// entry IDs bought
	Endings		[]string		`json:",omitempty"`	// endings seen
	Returns		int								// runs started over through the return
	TutorialDone	bool
}

type runFile struct {
	Version	int
	Run		*Hero
}

type profileFile struct {
	Version	int
	Profile	Profile
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err
	}
	return writeSave(path, data)
}

// SaveRun writes the run in progress.
func SaveRun(path string, h *Hero) error {
	return writeJSON(path, runFile{ SAVE_VERSION, h })
}

// SaveProfile writes the profile.
func SaveProfile(path string, p Profile) error {
	return writeJSON(path, profileFile{ PROFILE_VERSION, p })
}

// backUp copies save data to "<path>.bak", for a save the next write would lose: one that
// can't be parsed, or one about to be converted.
func backUp(path string, data []byte) {
	writeSave(path + ".bak", data)
}

// readOrBackUp reads a save and parses it. A save that can't be parsed is backed up.
func readOrBackUp(path string, parse func([]byte) error) error {
	data, err := readSave(path)
	if err != nil || data == nil {
		return err
	}
	if err := parse(data); err != nil {
		backUp(path, data)
		return err
	}
	return nil
}

func saveVersion(data []byte, newest int) (int, error) {
	var head struct{ Version int }
	if err := json.Unmarshal(data, &head); err != nil {
		return 0, err
	}
	if head.Version > newest {
		return 0, fmt.Errorf("save is from a newer version (%d)", head.Version)
	}
	return head.Version, nil
}

// LoadRun returns nil and no error when there is no run saved. A save it can't read, or one from
// a newer version of the game, is an error, and the nil it returns starts fresh.
func LoadRun(path string) (*Hero, error) {
	var h *Hero
	var buildings []string
	old, smallOrrery, fourMana := false, false, false
	err := readOrBackUp(path, func(data []byte) error {
		version, err := saveVersion(data, SAVE_VERSION)
		if err != nil {
			return err
		}
		// version 0 was a bare hero; later versions wrap it
		run := data
		if version > 0 {
			var f struct{ Run json.RawMessage }
			if err := json.Unmarshal(data, &f); err != nil {
				return err
			}
			run = f.Run
		}
		if len(run) == 0 || string(run) == "null" {
			return nil
		}
		h = &Hero{}
		if err := json.Unmarshal(run, h); err != nil {
			return err
		}
		if version < SAVE_VERSION {
			// an older run: it is converted below, so keep the original
			backUp(path, data)
			smallOrrery, fourMana = version < 3, true
		}
		if version < 2 {
			var b struct{ Buildings []string }
			json.Unmarshal(run, &b)
			buildings, old = b.Buildings, true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if h != nil {
		if fourMana {
			h.fromFourMana()
		}
		h.fixUp()
		if old {
			h.fromBeforeAcademy(buildings)
		}
		if smallOrrery {
			h.fromSmallOrrery()
		}
	}
	return h, nil
}

// LoadProfile returns an empty profile and no error when there is none yet.
func LoadProfile(path string) (Profile, error) {
	var f profileFile
	err := readOrBackUp(path, func(data []byte) error {
		if _, err := saveVersion(data, PROFILE_VERSION); err != nil {
			return err
		}
		return json.Unmarshal(data, &f)
	})
	if err != nil {
		return Profile{}, err
	}
	return f.Profile, nil
}

// DeleteRun forgets the run in progress. The profile is in its own save and is not touched.
func DeleteRun(path string) error {
	return removeSave(path)
}

// fromFourMana converts a run saved before Aether: its arrays were read into the front of the
// longer ones, so everything from Aether's index on moves up one, and Aether starts at zero.
func (h *Hero) fromFourMana() {
	widen := func(a []int) {
		copy(a[Aether+1:], a[Aether:len(a)-1])
		a[Aether] = 0
	}
	widen(h.Stats[:])
	widen(h.Residue[:])
	widen(h.Spent[:])
}

// fixUp fills in what an older or hand-edited save left out.
func (h *Hero) fixUp() {
	if h.Gear == nil {
		h.Gear = map[string]string{}
	}
	if h.Active == nil {
		h.Active = map[string]int{}
	}
	if h.Flags == nil {
		h.Flags = map[string]bool{}
	}
	if h.Seen == nil {
		h.Seen = map[string]bool{}
	}
	if LocationByID(h.Location) == nil {
		h.Location = Story.Start
	}
	if h.Nodes == nil {
		// a run from before the Orrery: stats raised with level-up points go back to the
		// class's, unspent points are dropped, and each level pays colorless residue instead
		h.Stats = ClassByName(h.Class).Stats
		h.Residue[Colorless] += (h.Level - 1) * OLD_RUN_RESIDUE
		h.takeSeat()
		h.Spells = slices.DeleteFunc(h.Spells, func(id string) bool {
			s := SpellByID(id)
			return s == nil || !h.Knows(s)
		})
		h.learnSpells()
	}
}
