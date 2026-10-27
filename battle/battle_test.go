package battle

import (
	"jewel-game/board"
	"testing"
)

// roles: color 0 attack, 1 mana, 2 heal, 3 nothing
var testRoles = []Role{ RoleAttack, RoleMana, RoleHeal, RoleNone }

func step(chain int, colors ...int) *board.StepResult {
	res := &board.StepResult{ Chain: chain }
	for _, c := range colors {
		res.Cleared = append(res.Cleared, board.Cleared{ Gem: board.Gem{ Color: c } })
	}
	return res
}

func TestApplyStepUsesRolesAndChain(t *testing.T) {
	b := New(1)
	b.Player.HP = 50
	hp := b.Enemy.HP

	g := b.ApplyStep(step(1, 0, 0, 0, 1, 1, 1, 2, 2, 2, 3, 3, 3, -1), testRoles)
	if g.Damage != 9 || g.Mana != 3 || g.Heal != 6 {
		t.Fatalf("chain 1: got %+v", g)
	}
	if b.Enemy.HP != hp-9 || b.Player.Mana != 3 || b.Player.HP != 56 {
		t.Fatalf("state not updated: enemy %d, player %+v", b.Enemy.HP, b.Player)
	}

	g = b.ApplyStep(step(3, 0, 0, 0), testRoles)
	if g.Multiplier != 2 || g.Damage != 18 {
		t.Fatalf("chain 3 should double: got %+v", g)
	}
}

func TestHealAndManaCap(t *testing.T) {
	b := New(1)
	b.Player.Mana = b.Player.MaxMana - 1
	g := b.ApplyStep(step(1, 1, 1, 1, 2, 2, 2), testRoles)
	if g.Mana != 1 || g.Heal != 0 || b.Player.HP != b.Player.MaxHP {
		t.Fatalf("caps not applied: %+v", g)
	}
}

func TestEnemyActsOnCountdown(t *testing.T) {
	b := New(1)
	interval := b.Enemy.Interval
	for i := 1; i < interval; i++ {
		if b.EndTurn() {
			t.Fatalf("acted early on turn %d", i)
		}
	}
	if !b.EndTurn() {
		t.Fatal("didn't act when the countdown ran out")
	}
	hp := b.Player.HP
	a := b.EnemyAct()
	if b.Player.HP != hp-a.Damage || a.Damage == 0 {
		t.Fatalf("damage not dealt: %+v", a)
	}
	if b.Enemy.Countdown != interval {
		t.Fatalf("countdown not reset: %d", b.Enemy.Countdown)
	}
}

func TestDrainStealsMana(t *testing.T) {
	b := New(1)
	b.Enemy = NewEnemy(3)	// Wraith opens with Drain
	b.Player.Mana = 20
	a := b.EnemyAct()
	if a.Kind != Drain || a.Drained != a.Damage || b.Player.Mana != 20-a.Damage {
		t.Fatalf("drain: %+v, mana %d", a, b.Player.Mana)
	}
}

func TestVictoryPerkAndNextWave(t *testing.T) {
	b := New(1)
	b.ApplyStep(step(1, make([]int, 100)...), testRoles)
	if b.State != Victory || len(b.PerkChoices) != 3 {
		t.Fatalf("want victory with 3 perks, got state %v, %d perks", b.State, len(b.PerkChoices))
	}
	if b.EndTurn() {
		t.Fatal("dead enemy took a turn")
	}
	before := b.Player
	if !b.ChoosePerk(0) {
		t.Fatal("couldn't pick a perk")
	}
	if b.Wave != 2 || b.State != PlayerTurn || b.Enemy.HP != b.Enemy.MaxHP {
		t.Fatalf("next wave not started: wave %d state %v", b.Wave, b.State)
	}
	if len(b.Player.Perks) != len(before.Perks)+1 {
		t.Fatal("perk not recorded")
	}
}

func TestDefeat(t *testing.T) {
	b := New(1)
	b.Player.HP = 1
	b.EnemyAct()
	if b.State != Defeat {
		t.Fatalf("want defeat, got %v", b.State)
	}
	if g := b.ApplyStep(step(1, 0, 0, 0), testRoles); g.Damage != 0 {
		t.Fatal("dead player still dealt damage")
	}
}

func TestOncePerksDontRepeat(t *testing.T) {
	b := New(1)
	b.Mods.Diagonal, b.Mods.Overcharge = true, true
	for i := 0; i < 50; i++ {
		for _, p := range b.rollPerks(3) {
			if p.Name == "Slippery" || p.Name == "Overcharge" {
				t.Fatalf("offered %s twice", p.Name)
			}
		}
	}
}

func TestSkills(t *testing.T) {
	b := New(1)
	if b.Cast(Fireball) {
		t.Fatal("cast without mana")
	}
	b.Player.Mana = 30
	hp := b.Enemy.HP
	if !b.Cast(Fireball) || b.Enemy.HP != hp-b.Player.AttackPower*8 || b.Player.Mana != 30-Fireball.Cost() {
		t.Fatalf("fireball: enemy %d, mana %d", b.Enemy.HP, b.Player.Mana)
	}
}

func TestLaterLapsAreTougher(t *testing.T) {
	first, again := NewEnemy(1), NewEnemy(1+len(enemyKinds))
	if again.MaxHP <= first.MaxHP || again.Attack <= first.Attack || again.Name == first.Name {
		t.Fatalf("lap 2 not scaled: %+v vs %+v", again, first)
	}
}
