package core

import (
	"fmt"
	"jewel-game/battle"
	"jewel-game/board"
	"jewel-game/entities"
	"jewel-game/ui"
	"math"
	"math/rand"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const ENEMY_DELAY = 0.6	// pause before the enemy acts, so its turn reads as a turn

var (
	hpColor		= rl.Color{ R: 220, G: 60, B: 70, A: 255 }
	manaColor	= rl.Color{ R: 70, G: 120, B: 240, A: 255 }
	healColor	= rl.Color{ R: 80, G: 220, B: 120, A: 255 }
)

// BattleView runs battle mode: the board is the weapon, the side panels are the fight.
type BattleView struct {
	Battle		*battle.Battle
	TM			*TileManager
	DataManager	*DataManager
	effects		Effects		// popups in panel space
	enemyTimer	float32
	enemyHit	float32		// 1 → 0 white flash when hurt
	enemyLunge	float32		// 1 → 0 when attacking
	playerHit	float32		// 1 → 0 red screen flash
	time		float64
	// where things were last drawn (panel space) so popups can point at them
	enemyAt		rl.Vector2
	hpBar		rl.Rectangle
	manaBar		rl.Rectangle
}

func NewBattleView(dm *DataManager) *BattleView {
	bv := &BattleView{ DataManager: dm }
	bv.TM = NewTileManager(board.DefaultSettings(), dm)
	bv.TM.OnStep = bv.onStep
	bv.TM.OnMoveDone = bv.onMoveDone
	bv.Restart()
	return bv
}

func (bv *BattleView) Restart() {
	bv.Battle = battle.New(time.Now().UnixNano())
	bv.newBoard()
}

func (bv *BattleView) newBoard() {
	bv.TM.Reset(bv.Battle.BoardSettings(max(3, len(bv.DataManager.GemTypeOrder))))
	bv.enemyTimer = 0
}

func (bv *BattleView) roles() []battle.Role {
	types := bv.DataManager.GetAllGemTypes()
	roles := make([]battle.Role, len(types))
	for i, gt := range types {
		roles[i] = battle.ParseRole(gt.Role)
	}
	return roles
}

func (bv *BattleView) onStep(res *board.StepResult) {
	g := bv.Battle.ApplyStep(res, bv.roles())
	if g.Damage > 0 {
		bv.enemyHit = 1
		text := fmt.Sprintf("-%d", g.Damage)
		if g.Multiplier > 1 {
			text += fmt.Sprintf("  x%g", g.Multiplier)
		}
		bv.effects.Popup(text, bv.enemyAt, 34, hpColor)
	}
	if g.Heal > 0 {
		bv.effects.Popup(fmt.Sprintf("+%d HP", g.Heal), barTop(bv.hpBar), 22, healColor)
	}
	if g.Mana > 0 {
		bv.effects.Popup(fmt.Sprintf("+%d MP", g.Mana), barTop(bv.manaBar), 22, manaColor)
	}
}

func (bv *BattleView) onMoveDone() {
	if bv.Battle.EndTurn() {
		bv.enemyTimer = ENEMY_DELAY
	}
}

func (bv *BattleView) Update() {
	dt := rl.GetFrameTime()
	bv.time += float64(dt)
	bv.enemyHit = max(0, bv.enemyHit-dt*3)
	bv.enemyLunge = max(0, bv.enemyLunge-dt*3)
	bv.playerHit = max(0, bv.playerHit-dt*2)
	bv.effects.Update(dt)

	if bv.enemyTimer > 0 {
		bv.enemyTimer -= dt
		if bv.enemyTimer <= 0 {
			bv.enemyTurn()
		}
	}
	bv.TM.Paused = bv.Battle.State != battle.PlayerTurn || bv.enemyTimer > 0

	if bv.TM.Busy() {
		return
	}
	switch bv.Battle.State {
	case battle.PlayerTurn:
		for i, key := range []int32{ rl.KeyOne, rl.KeyTwo, rl.KeyThree } {
			if rl.IsKeyPressed(key) && i < len(battle.Skills) {
				bv.Cast(battle.Skills[i])
			}
		}
	case battle.Victory:
		for i, key := range []int32{ rl.KeyOne, rl.KeyTwo, rl.KeyThree } {
			if rl.IsKeyPressed(key) {
				bv.choosePerk(i)
			}
		}
	case battle.Defeat:
		if rl.IsKeyPressed(rl.KeyR) || rl.IsKeyPressed(rl.KeyEnter) {
			bv.Restart()
		}
	}
}

func (bv *BattleView) enemyTurn() {
	a := bv.Battle.EnemyAct()
	bv.enemyLunge, bv.playerHit = 1, 1
	bv.TM.Effects.shake = 0.5
	bv.effects.Popup(fmt.Sprintf("-%d", a.Damage), barTop(bv.hpBar), 30, hpColor)
	switch a.Kind {
	case battle.Drain:
		if a.Drained > 0 {
			bv.effects.Popup(fmt.Sprintf("-%d MP", a.Drained), barTop(bv.manaBar), 24, manaColor)
		}
	case battle.Quake:
		b := bv.TM.Board
		b.Settings.Gravity = board.Direction((int(b.Settings.Gravity) + 1 + rand.Intn(3)) % 4)
		r := bv.TM.Rect()
		bv.TM.Effects.Popup("Gravity shifts "+b.Settings.Gravity.String()+"!", rl.Vector2{ X: r.X + r.Width/2, Y: r.Y + r.Height/2 }, 36, ui.Accent)
	}
}

// Cast uses a skill if the board is settled and there's mana for it.
func (bv *BattleView) Cast(s battle.Skill) {
	if bv.TM.Busy() || bv.TM.Paused || !bv.Battle.Cast(s) {
		return
	}
	switch s {
	case battle.Fireball:
		bv.enemyHit = 1
		bv.effects.Popup(fmt.Sprintf("-%d", bv.Battle.Player.AttackPower*8), bv.enemyAt, 38, rl.Orange)
	case battle.Mend:
		bv.effects.Popup("Mended", barTop(bv.hpBar), 22, healColor)
	case battle.Conjure:
		bv.conjureBomb()
	}
}

func (bv *BattleView) conjureBomb() {
	b := bv.TM.Board
	var spots []board.Point
	for y, row := range b.Cells {
		for x, g := range row {
			if g != nil && !g.Locked && g.Special == board.None {
				spots = append(spots, board.Point{ X: x, Y: y })
			}
		}
	}
	if len(spots) == 0 {
		return
	}
	p := spots[rand.Intn(len(spots))]
	b.Paint(p, b.At(p).Color, board.Bomb)
	bv.TM.Effects.Popup("Bomb!", bv.TM.pointCenter(p), 26, ui.Accent)
	bv.TM.Effects.Burst(bv.TM.pointCenter(p), rl.Orange, 20, bv.TM.CellSize*5)
}

func (bv *BattleView) choosePerk(i int) {
	if bv.Battle.ChoosePerk(i) {
		bv.newBoard()
	}
}

func barTop(r rl.Rectangle) rl.Vector2 {
	return rl.Vector2{ X: r.X + r.Width*0.7, Y: r.Y - 2 }
}

// DrawPlayer fills the left panel below the mode tabs.
func (bv *BattleView) DrawPlayer(col *ui.Column) {
	p := bv.Battle.Player
	col.Heading("Hero")
	bv.hpBar = col.Next(28)
	ui.Bar(bv.hpBar, "HP", p.HP, p.MaxHP, hpColor)
	bv.manaBar = col.Next(24)
	ui.Bar(bv.manaBar, "Mana", p.Mana, p.MaxMana, manaColor)

	col.Heading("Gems")
	types := bv.DataManager.GetAllGemTypes()
	for _, role := range []battle.Role{ battle.RoleAttack, battle.RoleMana, battle.RoleHeal } {
		for i, gt := range types {
			if battle.ParseRole(gt.Role) != role {
				continue
			}
			bv.drawGemLegend(col, i, gt, role)
		}
	}
	col.Paragraph("Chains multiply everything: x1.5, x2, x2.5 ...", 15, ui.DimText)

	col.Heading("Skills (free action)")
	bv.drawSkills(col)
	bv.drawPerks(col)
}

func (bv *BattleView) drawGemLegend(col *ui.Column, i int, gt *entities.GemType, role battle.Role) {
	p := bv.Battle.Player
	r := col.Next(30)
	entities.DrawGem(entities.GemLook{
		Type:	gt,
		Gem:	board.Gem{ Color: i },
		Center:	rl.Vector2{ X: r.X + 14, Y: r.Y + 15 },
		Size:	34,
		Scale:	1,
		Alpha:	1,
		Time:	bv.time,
	})
	text := map[battle.Role]string{
		battle.RoleAttack:	fmt.Sprintf("Attack  %d dmg each", p.AttackPower),
		battle.RoleMana:	fmt.Sprintf("Mana    %d MP each", p.ManaPower),
		battle.RoleHeal:	fmt.Sprintf("Heal    %d HP each", p.HealPower),
	}[role]
	ui.Text(text, r.X+40, r.Y+6, 18, ui.TextColor)
}

func (bv *BattleView) drawSkills(col *ui.Column) {
	p := bv.Battle.Player
	for i, s := range battle.Skills {
		r := col.Next(34)
		btn := rl.Rectangle{ X: r.X, Y: r.Y, Width: 168, Height: r.Height }
		label := fmt.Sprintf("%d %s %d", i+1, s.Name(), s.Cost())
		if ui.ButtonIf(btn, label, bv.Battle.CanCast(s) && !bv.TM.Busy() && !bv.TM.Paused) {
			bv.Cast(s)
		}
		ui.Text(s.Desc(p), btn.X+btn.Width+10, r.Y+9, 15, ui.DimText)
	}
}

func (bv *BattleView) drawPerks(col *ui.Column) {
	p := bv.Battle.Player
	col.Heading("Perks")
	if len(p.Perks) == 0 {
		col.Paragraph("Beat an enemy to pick your first perk.", 15, ui.DimText)
	} else {
		col.Paragraph(strings.Join(p.Perks, ", "), 16, ui.TextColor)
	}
	col.Line(fmt.Sprintf("Locks %d / %d    Turn %d", bv.TM.Board.LockCount(), bv.Battle.Mods.Locks, bv.Battle.Turns), 16, ui.DimText)
}

// DrawEnemy fills the right panel.
func (bv *BattleView) DrawEnemy(r rl.Rectangle) {
	ui.Panel(r)
	e := bv.Battle.Enemy
	col := ui.Column{ X: r.X + 18, Y: r.Y + 16, W: r.Width - 36, Gap: 6 }
	col.Heading(fmt.Sprintf("Wave %d", bv.Battle.Wave))
	name := col.Next(40)
	ui.TextCentered(e.Name, name.X+name.Width/2, name.Y, 36, e.Color)

	stage := col.Next(230)
	bv.enemyAt = rl.Vector2{ X: stage.X + stage.Width/2, Y: stage.Y + stage.Height/2 }
	bv.drawMonster(bv.enemyAt, 85)

	ui.Bar(col.Next(28), "HP", e.HP, e.MaxHP, hpColor)

	if e.HP > 0 {
		intent := col.Next(34)
		turns := "next turn"
		if e.Countdown > 1 {
			turns = fmt.Sprintf("in %d turns", e.Countdown)
		}
		urgent := ui.TextColor
		if e.Countdown <= 1 {
			urgent = hpColor
		}
		ui.Text(fmt.Sprintf("%s %s", e.Intent().Short(), turns), intent.X, intent.Y+6, 20, urgent)
		// countdown pips
		for i := 0; i < e.Interval; i++ {
			c := rl.Vector2{ X: intent.X + intent.Width - 10 - float32(i)*22, Y: intent.Y + 17 }
			if i < e.Countdown {
				rl.DrawCircleV(c, 7, urgent)
			} else {
				rl.DrawCircleLines(int32(c.X), int32(c.Y), 7, ui.DimText)
			}
		}
	}

	col.Heading("Battle log")
	bottom := r.Y + r.Height - 24
	for i, line := range bv.Battle.Log {
		if col.Y > bottom {
			break
		}
		c := ui.TextColor
		if i > 0 {
			c = ui.DimText
		}
		col.Paragraph(line, 15, c)
	}
}

func (bv *BattleView) drawMonster(c rl.Vector2, rad float32) {
	e := bv.Battle.Enemy
	if e.HP <= 0 {
		ui.TextCentered("x_x", c.X, c.Y-20, 40, ui.DimText)
		return
	}
	c.Y += float32(math.Sin(bv.time*2)) * 6
	c.X += (rand.Float32()*2 - 1) * 10 * bv.enemyHit
	rad *= 1 + 0.18*bv.enemyLunge
	col := lerp(e.Color, rl.White, bv.enemyHit)
	alpha := float32(1)
	if strings.HasSuffix(e.Name, "Wraith") {
		alpha = 0.75
	}

	switch {
	case strings.HasSuffix(e.Name, "Golem"):
		body := rl.Rectangle{ X: c.X - rad, Y: c.Y - rad*0.9, Width: rad * 2, Height: rad * 1.9 }
		rl.DrawRectangleRounded(body, 0.25, 6, rl.Fade(col, alpha))
	case strings.HasSuffix(e.Name, "Slime"):
		rl.DrawCircleV(c, rad, rl.Fade(col, alpha))
		rl.DrawRectangleRec(rl.Rectangle{ X: c.X - rad, Y: c.Y + rad*0.55, Width: rad * 2, Height: rad * 0.46 }, rl.Fade(col, alpha))
	default:
		rl.DrawCircleV(c, rad, rl.Fade(col, alpha))
	}
	if strings.HasSuffix(e.Name, "Goblin") || strings.HasSuffix(e.Name, "Dragon") {
		for _, side := range []float32{ -1, 1 } {
			base := rl.Vector2{ X: c.X + side*rad*0.55, Y: c.Y - rad*0.7 }
			tip := rl.Vector2{ X: c.X + side*rad*0.85, Y: c.Y - rad*1.35 }
			inner := rl.Vector2{ X: c.X + side*rad*0.2, Y: c.Y - rad*0.9 }
			rl.DrawTriangle(base, tip, inner, rl.Fade(col, alpha))
			rl.DrawTriangle(base, inner, tip, rl.Fade(col, alpha))
		}
	}
	rl.DrawCircleV(rl.Vector2{ X: c.X - rad*0.35, Y: c.Y - rad*0.4 }, rad*0.22, rl.Fade(rl.White, 0.18))

	// eyes follow the mouse
	mouse := rl.GetMousePosition()
	for _, side := range []float32{ -1, 1 } {
		eye := rl.Vector2{ X: c.X + side*rad*0.35, Y: c.Y - rad*0.12 }
		rl.DrawCircleV(eye, rad*0.2, rl.White)
		dx, dy := mouse.X-eye.X, mouse.Y-eye.Y
		d := float32(math.Hypot(float64(dx), float64(dy)))
		if d > 0 {
			dx, dy = dx/d*rad*0.08, dy/d*rad*0.08
		}
		rl.DrawCircleV(rl.Vector2{ X: eye.X + dx, Y: eye.Y + dy }, rad*0.09, rl.Black)
	}

	// grin turns to fangs when it's about to act
	mouthY := c.Y + rad*0.35
	if e.Countdown <= 1 {
		for i := float32(-2); i < 2; i++ {
			x := c.X + i*rad*0.15
			a, b := rl.Vector2{ X: x, Y: mouthY }, rl.Vector2{ X: x + rad*0.15, Y: mouthY }
			tip := rl.Vector2{ X: x + rad*0.075, Y: mouthY + rad*0.18 }
			rl.DrawTriangle(a, tip, b, rl.White)
			rl.DrawTriangle(a, b, tip, rl.White)
		}
	} else {
		rl.DrawRing(rl.Vector2{ X: c.X, Y: mouthY - rad*0.1 }, rad*0.22, rad*0.27, 20, 160, 12, rl.Black)
	}
}

// DrawPanelEffects draws popups over the panels; call inside the panel camera.
func (bv *BattleView) DrawPanelEffects() {
	bv.effects.Draw()
}

// DrawScreenFlash tints the screen red when the player gets hit.
func (bv *BattleView) DrawScreenFlash() {
	if bv.playerHit > 0 {
		rl.DrawRectangle(0, 0, int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()), rl.Fade(hpColor, 0.22*bv.playerHit))
	}
}

// DrawOverlay shows the perk picker after a win and the defeat screen. Screen space.
func (bv *BattleView) DrawOverlay() {
	st := bv.Battle.State
	if st == battle.PlayerTurn || bv.TM.Busy() {
		return
	}
	w, h := float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, int32(w), int32(h), rl.Fade(rl.Black, 0.55))

	if st == battle.Defeat {
		box := rl.Rectangle{ X: w/2 - 300, Y: h/2 - 140, Width: 600, Height: 280 }
		ui.Panel(box)
		ui.TextCentered("Defeated", w/2, box.Y+30, 52, hpColor)
		ui.TextCentered(fmt.Sprintf("You reached wave %d in %d turns.", bv.Battle.Wave, bv.Battle.Turns), w/2, box.Y+105, 22, ui.TextColor)
		if ui.Button(rl.Rectangle{ X: w/2 - 120, Y: box.Y + 170, Width: 240, Height: 50 }, "Try again  [R]", false) {
			bv.Restart()
		}
		return
	}

	choices := bv.Battle.PerkChoices
	cardW, gap := float32(300), float32(24)
	total := float32(len(choices))*cardW + float32(len(choices)-1)*gap
	box := rl.Rectangle{ X: w/2 - total/2 - 30, Y: h/2 - 160, Width: total + 60, Height: 320 }
	ui.Panel(box)
	ui.TextCentered("Victory!", w/2, box.Y+24, 48, ui.Accent)
	ui.TextCentered("Choose a perk", w/2, box.Y+82, 22, ui.TextColor)
	for i, p := range choices {
		card := rl.Rectangle{ X: box.X + 30 + float32(i)*(cardW+gap), Y: box.Y + 130, Width: cardW, Height: 150 }
		hover := rl.CheckCollisionPointRec(rl.GetMousePosition(), card)
		edge := ui.PanelEdge
		if hover {
			edge = ui.Accent
		}
		rl.DrawRectangleRounded(card, 0.08, 8, rl.Color{ R: 40, G: 34, B: 70, A: 255 })
		rl.DrawRectangleRoundedLines(card, 0.08, 8, 3, edge)
		col := ui.Column{ X: card.X + 18, Y: card.Y + 18, W: card.Width - 36, Gap: 4 }
		col.Line(fmt.Sprintf("%d  %s", i+1, p.Name), 26, ui.Accent)
		col.Y += 10
		col.Paragraph(p.Desc, 18, ui.TextColor)
		if hover && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			bv.choosePerk(i)
		}
	}
}

func lerp(a, b rl.Color, t float32) rl.Color {
	l := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t) }
	return rl.Color{ R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: a.A }
}
