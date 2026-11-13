// Package rpg holds the Puzzle Quest style rules: duels on a shared board, classes, spells,
// gear, the world and its quests. Like the board package it has no rendering.
package rpg

import "slices"

// Gem colors on a duel board. The first four are mana, and index Combatant.Mana and Stats.
const (
	Fire = iota
	Water
	Air
	Earth
	Skull
	Gold
	Star
	NUM_GEMS
)

const NUM_MANA = 4

var ManaNames = [NUM_MANA]string{ "Fire", "Water", "Air", "Earth" }

// Stats index: the masteries line up with the mana colors.
const (
	Battle = iota + NUM_MANA
	Cunning
	Morale
	NUM_STATS
)

var StatNames = [NUM_STATS]string{ "Fire", "Water", "Air", "Earth", "Battle", "Cunning", "Morale" }
var StatHelp = [NUM_STATS]string{
	"+1 max Fire mana. Every 6 adds +1 Fire per match.",
	"+1 max Water mana. Every 6 adds +1 Water per match.",
	"+1 max Air mana. Every 6 adds +1 Air per match.",
	"+1 max Earth mana. Every 6 adds +1 Earth per match.",
	"Every 4 adds +1 damage per skull match.",
	"+2% gold and XP. Decides who moves first.",
	"+3 max HP. Every 10 blocks 1 skull damage.",
}

const MAX_SPELLS = 6

type Class struct {
	Name	string
	Desc	string
	Stats	[NUM_STATS]int
	Color	[3]uint8
}

var Classes = []Class{
	{ "Knight", "Holy armor and steady damage. Forgiving, and hard to kill.",
		[NUM_STATS]int{ 2, 2, 1, 3, 5, 2, 6 }, [3]uint8{ 210, 215, 235 } },
	{ "Warrior", "Rage and big skull hits. Wins by ending fights early.",
		[NUM_STATS]int{ 4, 1, 2, 2, 7, 2, 3 }, [3]uint8{ 225, 90, 70 } },
	{ "Druid", "Poison, healing and control of the Earth and the board.",
		[NUM_STATS]int{ 1, 3, 2, 6, 2, 4, 3 }, [3]uint8{ 95, 190, 100 } },
	{ "Wizard", "Fragile, with the deepest mana and the strongest spells.",
		[NUM_STATS]int{ 5, 5, 4, 1, 1, 3, 2 }, [3]uint8{ 110, 130, 240 } },
}

func ClassByName(name string) *Class {
	for i := range Classes {
		if Classes[i].Name == name {
			return &Classes[i]
		}
	}
	return &Classes[0]
}

// Bonus is what gear, facilities and Orrery nodes add to a hero.
type Bonus struct {
	Stats		[NUM_STATS]int
	HP			int
	SkullDamage	int		// per skull match
	Armor		int		// less skull damage per match
	GoldPct		int
	XPPct		int
	StartMana	int		// of every color at the start of a duel
}

func (b *Bonus) add(o Bonus) {
	for i := range b.Stats {
		b.Stats[i] += o.Stats[i]
	}
	b.HP += o.HP
	b.SkullDamage += o.SkullDamage
	b.Armor += o.Armor
	b.GoldPct += o.GoldPct
	b.XPPct += o.XPPct
	b.StartMana += o.StartMana
}

// Hero is the saved character. Gear, spells and quests are stored by ID so saves survive retuning.
type Hero struct {
	Class		string
	Level		int
	XP			int
	Gold		int
	Stats		[NUM_STATS]int	// the class's; the Orrery adds to them
	Nodes		[]string		// Orrery nodes owned, the seat first. Nil in a run from before the Orrery
	Residue		[NUM_RESIDUE]int
	Spent		[NUM_RESIDUE]int	// residue paid for the nodes owned, from each pool, for a respec
	Gear		map[string]string	// slot → item ID
	Owned		[]string
	Spells		[]string		// equipped, at most MAX_SPELLS
	Facilities	map[string]int		`json:",omitempty"`	// facility ID → levels built
	Henchmen	[]Henchman		`json:",omitempty"`
	Dues		map[string]int		`json:",omitempty"`	// the ledger: site → dues not yet claimed
	Holders		map[string]string	`json:",omitempty"`	// site → who holds it; none means no one
	Location	string
	Active		map[string]int	// quest ID → kills so far
	Done		[]string
	Wins		int
	Losses		int
	Finished	bool			// the story has been beaten
	Flags		map[string]bool	`json:",omitempty"`	// story flags set by scenes
	Seen		map[string]bool	`json:",omitempty"`	// scenes played
}

func NewHero(class string) *Hero {
	h := &Hero{
		Class:		class,
		Level:		1,
		Gold:		50,
		Stats:		ClassByName(class).Stats,
		Gear:		map[string]string{},
		Location:	Story.Start,
		Active:		map[string]int{},
		Flags:		map[string]bool{},
		Seen:		map[string]bool{},
	}
	h.takeSeat()
	h.learnSpells()
	return h
}

// XPToNext is the XP needed to go from level to level+1.
func XPToNext(level int) int {
	return (60 + 10*level) * level
}

// AddXP banks XP and reports how many levels were gained.
func (h *Hero) AddXP(n int) int {
	h.XP += n
	gained := 0
	for h.XP >= XPToNext(h.Level) {
		h.XP -= XPToNext(h.Level)
		h.Level++
		gained++
	}
	if gained > 0 {
		h.learnSpells()
	}
	return gained
}

// learnSpells equips newly learned spells while there are free slots.
func (h *Hero) learnSpells() {
	for _, s := range h.KnownSpells() {
		if len(h.Spells) < MAX_SPELLS && !slices.Contains(h.Spells, s.ID) {
			h.Spells = append(h.Spells, s.ID)
		}
	}
}

// KnownSpells is every rite the hero's kit, Orrery and level have unlocked.
func (h *Hero) KnownSpells() []*Spell {
	var out []*Spell
	for i := range Spells {
		if s := &Spells[i]; h.Knows(s) {
			out = append(out, s)
		}
	}
	return out
}

// ToggleSpell equips or unequips a known spell.
func (h *Hero) ToggleSpell(id string) {
	if i := slices.Index(h.Spells, id); i >= 0 {
		h.Spells = slices.Delete(h.Spells, i, i+1)
	} else if len(h.Spells) < MAX_SPELLS {
		h.Spells = append(h.Spells, id)
	}
}

// TotalBonus adds up gear, facilities and Orrery nodes.
func (h *Hero) TotalBonus() Bonus {
	b := h.facilityBonus()
	for _, id := range h.Gear {
		if it := ItemByID(id); it != nil {
			b.add(it.Bonus)
		}
	}
	for _, id := range h.Nodes {
		if n := NodeByID(id); n != nil {
			b.add(Bonus{ Stats: n.Stats })
		}
	}
	return b
}

// Buy pays for an item and equips it. Items already owned swap in for free.
func (h *Hero) Buy(it *Item) bool {
	if !slices.Contains(h.Owned, it.ID) {
		if h.Gold < it.Price || h.Level < it.Level {
			return false
		}
		h.Gold -= it.Price
		h.Owned = append(h.Owned, it.ID)
	}
	for i := range h.Henchmen {
		if h.Henchmen[i].Gear == it.ID {
			h.Henchmen[i].Gear = ""
		}
	}
	h.Gear[it.Slot] = it.ID
	return true
}

const (
	SlotWeapon	= "Weapon"
	SlotArmor	= "Armor"
	SlotHelm	= "Helm"
	SlotTrinket	= "Trinket"
)

var Slots = []string{ SlotWeapon, SlotArmor, SlotHelm, SlotTrinket }

type Item struct {
	ID, Name	string
	Slot		string
	Level		int
	Price		int
	Desc		string
	Bonus		Bonus
	QuestOnly	bool
}

func stat(i, n int) (s [NUM_STATS]int) {
	s[i] = n
	return s
}

func allMastery(n int) [NUM_STATS]int {
	return [NUM_STATS]int{ n, n, n, n }
}

var Items = []Item{
	{ "rusty_blade", "Rusty Blade", SlotWeapon, 1, 40, "+1 skull damage", Bonus{ SkullDamage: 1 }, false },
	{ "oak_staff", "Oak Staff", SlotWeapon, 1, 60, "+2 all masteries", Bonus{ Stats: allMastery(2) }, false },
	{ "steel_sword", "Steel Sword", SlotWeapon, 4, 220, "+2 skull damage, +1 Battle", Bonus{ SkullDamage: 2, Stats: stat(Battle, 1) }, false },
	{ "ember_rod", "Ember Rod", SlotWeapon, 5, 280, "+6 Fire, +5 starting mana", Bonus{ Stats: stat(Fire, 6), StartMana: 5 }, false },
	{ "war_axe", "War Axe", SlotWeapon, 9, 650, "+4 skull damage", Bonus{ SkullDamage: 4 }, false },
	{ "tide_scepter", "Tide Scepter", SlotWeapon, 10, 700, "+8 Water, +8 Air", Bonus{ Stats: [NUM_STATS]int{ 0, 8, 8 } }, false },
	{ "kingsbane", "Kingsbane", SlotWeapon, 12, 0, "+6 skull damage, +4 Battle", Bonus{ SkullDamage: 6, Stats: stat(Battle, 4) }, true },

	{ "leather", "Leather Jerkin", SlotArmor, 1, 50, "+10 HP", Bonus{ HP: 10 }, false },
	{ "chain", "Chain Shirt", SlotArmor, 4, 240, "+20 HP, blocks 1", Bonus{ HP: 20, Armor: 1 }, false },
	{ "mage_robe", "Mage Robe", SlotArmor, 5, 260, "+4 all masteries", Bonus{ Stats: allMastery(4) }, false },
	{ "plate", "Plate Armor", SlotArmor, 9, 700, "+40 HP, blocks 2", Bonus{ HP: 40, Armor: 2 }, false },
	{ "barrow_mail", "Barrow Mail", SlotArmor, 6, 0, "+30 HP, blocks 2, +3 Morale", Bonus{ HP: 30, Armor: 2, Stats: stat(Morale, 3) }, true },

	{ "cap", "Padded Cap", SlotHelm, 1, 35, "+5 HP, +1 Morale", Bonus{ HP: 5, Stats: stat(Morale, 1) }, false },
	{ "circlet", "Silver Circlet", SlotHelm, 3, 150, "+10% XP", Bonus{ XPPct: 10 }, false },
	{ "great_helm", "Great Helm", SlotHelm, 7, 420, "+25 HP, blocks 1", Bonus{ HP: 25, Armor: 1 }, false },
	{ "crown", "Sage's Crown", SlotHelm, 10, 720, "+6 all masteries, +15% XP", Bonus{ Stats: allMastery(6), XPPct: 15 }, false },

	{ "lucky_coin", "Lucky Coin", SlotTrinket, 1, 45, "+15% gold", Bonus{ GoldPct: 15 }, false },
	{ "mana_charm", "Mana Charm", SlotTrinket, 3, 170, "+4 starting mana", Bonus{ StartMana: 4 }, false },
	{ "fox_tail", "Fox Tail", SlotTrinket, 6, 380, "+6 Cunning", Bonus{ Stats: stat(Cunning, 6) }, false },
	{ "heart_stone", "Heartstone", SlotTrinket, 8, 0, "+40 HP, +8 starting mana", Bonus{ HP: 40, StartMana: 8 }, true },
}

func ItemByID(id string) *Item {
	for i := range Items {
		if Items[i].ID == id {
			return &Items[i]
		}
	}
	return nil
}
