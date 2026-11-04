package rpg

import (
	"encoding/json"
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

// SavePath names where the hero is stored: a file on desktop, a localStorage key in a browser.
func SavePath() string {
	return savePath()
}

func (h *Hero) Save(path string) error {
	data, err := json.MarshalIndent(h, "", "\t")
	if err != nil {
		return err
	}
	return writeSave(path, data)
}

// DeleteSave removes the stored hero; deleting a missing save is not an error.
func DeleteSave(path string) error {
	return removeSave(path)
}

// LoadHero returns nil and no error when there is no save yet.
func LoadHero(path string) (*Hero, error) {
	data, err := readSave(path)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	h := &Hero{}
	if err := json.Unmarshal(data, h); err != nil {
		return nil, err
	}
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
	return h, nil
}
