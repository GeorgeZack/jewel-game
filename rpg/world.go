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

// SavePath names where the save is stored: a file on desktop, a localStorage key in a browser.
func SavePath() string {
	return savePath()
}

// SAVE_VERSION is written into every save. Saves from before versioning have none and hold a
// bare Hero; LoadSave migrates them.
const SAVE_VERSION = 1

// Profile is what outlasts a run. Nothing fills it in yet.
type Profile struct {
	Grimoire	[]string		`json:",omitempty"`	// entry IDs bought
	Endings		[]string		`json:",omitempty"`	// endings seen
	Returns		int								// runs started over through the return
	TutorialDone	bool
}

// SaveFile is the whole save: the profile, and the run in progress (nil when there is none).
type SaveFile struct {
	Version	int
	Profile	Profile
	Run		*Hero	`json:",omitempty"`
}

func (s *SaveFile) Write(path string) error {
	s.Version = SAVE_VERSION
	data, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return err
	}
	return writeSave(path, data)
}

// LoadSave returns an empty save and no error when there is none yet. A save it can't read, or
// one from a newer version of the game, is an error; the empty save it returns starts fresh.
func LoadSave(path string) (*SaveFile, error) {
	data, err := readSave(path)
	if err != nil || data == nil {
		return &SaveFile{ Version: SAVE_VERSION }, err
	}
	var head struct{ Version int }
	if err := json.Unmarshal(data, &head); err != nil {
		return &SaveFile{ Version: SAVE_VERSION }, err
	}
	s := &SaveFile{}
	switch {
	case head.Version == 0:
		// version 0 was a bare hero, with no profile
		h := &Hero{}
		if err := json.Unmarshal(data, h); err != nil {
			return &SaveFile{ Version: SAVE_VERSION }, err
		}
		s.Run = h
	case head.Version > SAVE_VERSION:
		return &SaveFile{ Version: SAVE_VERSION }, fmt.Errorf("save is from a newer version (%d)", head.Version)
	default:
		if err := json.Unmarshal(data, s); err != nil {
			return &SaveFile{ Version: SAVE_VERSION }, err
		}
	}
	s.Version = SAVE_VERSION
	if s.Run != nil {
		s.Run.fixUp()
	}
	return s, nil
}

// DeleteRun forgets the run in progress and keeps the profile.
func DeleteRun(path string) error {
	s, err := LoadSave(path)
	if err != nil {
		return removeSave(path)
	}
	s.Run = nil
	return s.Write(path)
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
}
