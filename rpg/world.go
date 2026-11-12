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
// bare Hero; LoadRun migrates them.
const SAVE_VERSION = 1

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

// readOrBackUp reads a save and parses it. A save that can't be parsed is copied to "<path>.bak",
// since the next save overwrites it.
func readOrBackUp(path string, parse func([]byte) error) error {
	data, err := readSave(path)
	if err != nil || data == nil {
		return err
	}
	if err := parse(data); err != nil {
		writeSave(path + ".bak", data)
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
	err := readOrBackUp(path, func(data []byte) error {
		version, err := saveVersion(data, SAVE_VERSION)
		if err != nil {
			return err
		}
		if version == 0 {
			// version 0 was a bare hero
			h = &Hero{}
			return json.Unmarshal(data, h)
		}
		var f runFile
		err = json.Unmarshal(data, &f)
		h = f.Run
		return err
	})
	if err != nil {
		return nil, err
	}
	if h != nil {
		h.fixUp()
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
