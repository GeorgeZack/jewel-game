package core

import (
	"fmt"
	"jewel-game/board"
	"jewel-game/rpg"
	"jewel-game/ui"
	"math"
	"math/rand"
	"time"

	"jewel-game/rl"
)

const (
	AI_THINK_TIME	= 0.7	// pause before the opponent acts, so its turn reads as a turn
	AI_SHOW_TIME	= 0.6	// how long its chosen swap is highlighted
	TRAVEL_TIME		= 0.6	// seconds per road
	AMBUSH_CHANCE	= 0.2
)

var (
	hpColor		= rl.Color{ R: 220, G: 60, B: 70, A: 255 }
	healColor	= rl.Color{ R: 80, G: 220, B: 120, A: 255 }
	goldColor	= rl.Color{ R: 255, G: 190, B: 40, A: 255 }
	xpColor		= rl.Color{ R: 190, G: 110, B: 255, A: 255 }
	manaColors	= [rpg.NUM_MANA]rl.Color{
		{ R: 215, G: 60, B: 50, A: 255 },
		{ R: 60, G: 115, B: 235, A: 255 },
		{ R: 205, G: 180, B: 40, A: 255 },
		{ R: 60, G: 175, B: 80, A: 255 },
	}
)

type Screen int

const (
	ScreenClass Screen = iota
	ScreenWorld
	ScreenHero
	ScreenSpells
	ScreenQuests
	ScreenShop
	ScreenCitadel
	ScreenDuel
)

type ModalButton struct {
	Label	string
	Action	func()
}

// Modal is a dialog that takes over input until a button is pressed.
type Modal struct {
	Title, Body	string
	Buttons		[]ModalButton
}

// QuestView runs the RPG: the world and its menus, and duels on a shared board.
type QuestView struct {
	DM			*DataManager
	TM			*TileManager
	Hero		*rpg.Hero
	Screen		Screen
	Duel		*rpg.Duel
	modals		[]Modal
	scenes		SceneRunner
	effects		Effects
	time		float64
	saveErr		bool
	storyErr	string	// why the story on disk didn't load
	storyIssues	int		// content mistakes found while loading
	lastWon		bool

	// the current duel
	foeID, duelAt	string
	settled			bool
	aiTimer			float32
	aiMove			[2]board.Point
	aiShowing		bool
	sideHit			[2]float32
	sideFlash		float32
	hpBars			[2]rl.Rectangle
	manaBars		[2]rl.Rectangle
	portraits		[2]rl.Vector2

	// walking the map
	route		[]string
	legFrom		string
	legT		float32
}

func NewQuestView(dm *DataManager) *QuestView {
	v := &QuestView{ DM: dm, TM: NewTileManager(rpg.BoardSettings(), dm) }
	v.scenes.v = v
	v.ReloadStory()
	v.TM.Types = dm.DuelGemTypes
	v.TM.Duel = true
	v.TM.OnStep = v.onStep
	v.TM.OnMoveDone = v.onMoveDone
	v.TM.OnShuffle = func() {
		if v.Duel != nil {
			v.Duel.ManaDrain()
		}
	}
	hero, err := rpg.LoadHero(rpg.SavePath())
	if err != nil {
		ui.AddMessage("Could not read save, starting fresh: %v", err)
	}
	v.Hero = hero
	if hero != nil {
		v.Screen = ScreenWorld
	}
	return v
}

func (v *QuestView) InDuel() bool	{ return v.Screen == ScreenDuel }
func (v *QuestView) HasModal() bool	{ return len(v.modals) > 0 || v.scenes.Showing() }

// ReloadStory reads assets/story again. If the files on disk are broken, the old story stays, or
// on startup the copy built into the game is used.
func (v *QuestView) ReloadStory() {
	problems, err := rpg.LoadStory(storyFiles{}, STORY_DIR)
	v.storyErr = ""
	if err != nil {
		v.storyErr = err.Error()
		ui.AddMessage("Story did not load: %v", err)
		if len(rpg.Locations) == 0 {
			problems, err = rpg.LoadStory(storyFiles{ embedded: true }, STORY_DIR)
			if err != nil {
				panic("the built-in story is broken: " + err.Error())
			}
		}
	}
	v.storyIssues = len(problems)
	for _, p := range problems {
		ui.AddMessage("Story problem: %s", p)
	}
	if v.storyErr == "" {
		ui.AddMessage("Story loaded: %d locations, %d quests, %d scenes, %d problems", len(rpg.Locations), len(rpg.Quests), len(rpg.Scenes), len(problems))
	}
	forgetStoryArt()
	if h := v.Hero; h != nil && rpg.LocationByID(h.Location) == nil {
		h.Location = rpg.Story.Start
		v.route = nil
	}
}

func (v *QuestView) show(m Modal) {
	if len(m.Buttons) == 0 {
		m.Buttons = []ModalButton{ { "Continue", nil } }
	}
	v.modals = append(v.modals, m)
}

func (v *QuestView) save() {
	if v.Hero == nil {
		return
	}
	if err := v.Hero.Save(rpg.SavePath()); err != nil {
		ui.AddMessage("Could not save: %v", err)
		v.saveErr = true
		return
	}
	v.saveErr = false
}

func (v *QuestView) Update() {
	dt := rl.GetFrameTime()
	v.time += float64(dt)
	v.effects.Update(dt)
	v.sideFlash = max(0, v.sideFlash-dt*2)
	for i := range v.sideHit {
		v.sideHit[i] = max(0, v.sideHit[i]-dt*3)
	}
	if v.scenes.WaitingForDuel() && v.Screen != ScreenDuel && len(v.modals) == 0 {
		v.scenes.DuelDone(v.lastWon)
	}
	if len(v.modals) > 0 {
		v.TM.Paused = true
		return
	}
	if v.scenes.Showing() {
		v.scenes.Update(dt)
		v.TM.Paused = true
		return
	}
	switch v.Screen {
	case ScreenWorld:
		v.updateTravel(dt)
	case ScreenDuel:
		v.updateDuel(dt)
	}
}

func (v *QuestView) updateTravel(dt float32) {
	if len(v.route) == 0 {
		return
	}
	v.legT += dt / TRAVEL_TIME
	if v.legT < 1 {
		return
	}
	v.legT = 0
	v.legFrom = v.route[0]
	v.Hero.Location = v.route[0]
	v.route = v.route[1:]
	v.save()
	here := rpg.LocationByID(v.Hero.Location)
	if here.EnterScene != "" && !v.Hero.Seen[here.EnterScene] {
		v.route = nil
		v.scenes.Play(here.EnterScene)
		return
	}
	if len(here.Foes) > 0 && rand.Float32() < AMBUSH_CHANCE {
		v.route = nil
		foe := rpg.FoeByID(here.Foes[rand.Intn(len(here.Foes))])
		v.show(Modal{
			Title:	"Ambush!",
			Body:	fmt.Sprintf("A %s jumps you on the road into %s.", foe.Name, here.Name),
			Buttons: []ModalButton{ { "Fight", func() { v.startDuel(foe.ID, roamLevel(here, v.Hero), here.ID) } } },
		})
	}
}

// roamLevel keeps ordinary foes near the hero without dropping below the region's level.
func roamLevel(l *rpg.Location, h *rpg.Hero) int {
	return max(l.Level, min(h.Level, l.Level+3))
}

func (v *QuestView) startDuel(foeID string, level int, at string) {
	v.foeID, v.duelAt = foeID, at
	v.settled = false
	v.TM.Reset(rpg.BoardSettings())
	d := rpg.NewDuel(rpg.NewHeroCombatant(v.Hero), rpg.NewFoeCombatant(rpg.FoeByID(foeID), level), time.Now().UnixNano())
	d.Board = v.TM.Board
	d.OnHit = v.onHit
	d.OnHeal = func(side, n int) {
		v.effects.Popup(fmt.Sprintf("+%d", n), barTop(v.hpBars[side]), 28, healColor)
	}
	v.Duel = d
	v.Screen = ScreenDuel
	v.aiShowing = false
	v.aiTimer = AI_THINK_TIME + 0.6	// let the board land
}

func (v *QuestView) onHit(side, n int) {
	v.sideHit[side] = 1
	v.effects.Popup(fmt.Sprintf("-%d", n), barTop(v.hpBars[side]), 34, hpColor)
	v.TM.Effects.shake = 0.35
	if side == rpg.Player {
		v.sideFlash = 1
	}
}

func (v *QuestView) onStep(res *board.StepResult) {
	if v.Duel == nil {
		return
	}
	g := v.Duel.ApplyStep(res)
	bar := v.manaBars[g.Side]
	for i, n := range g.Mana {
		if n > 0 {
			at := rl.Vector2{ X: bar.X + bar.Width*0.7, Y: bar.Y + float32(i)*bar.Height }
			v.effects.Popup(fmt.Sprintf("+%d %s", n, rpg.ManaNames[i]), at, 22, rl.ColorBrightness(manaColors[i], 0.3))
		}
	}
	at := rl.Vector2{ X: bar.X + bar.Width*0.25, Y: bar.Y + bar.Height*4 }
	if g.Gold > 0 {
		v.effects.Popup(fmt.Sprintf("+%d gold", g.Gold), at, 22, goldColor)
	}
	if g.XP > 0 {
		v.effects.Popup(fmt.Sprintf("+%d XP", g.XP), rl.Vector2{ X: at.X + 150, Y: at.Y }, 22, xpColor)
	}
	if g.Extra {
		r := v.TM.Rect()
		v.TM.Effects.Popup("Extra turn!", rl.Vector2{ X: r.X + r.Width/2, Y: r.Y + r.Height/2 }, 46, ui.Accent)
	}
}

func (v *QuestView) onMoveDone() {
	if v.Duel == nil {
		return
	}
	v.Duel.EndTurn()
	v.aiTimer = AI_THINK_TIME
	v.aiShowing = false
}

func (v *QuestView) myTurn() bool {
	return v.Duel != nil && !v.Duel.Over() && v.Duel.Turn == rpg.Player
}

func (v *QuestView) updateDuel(dt float32) {
	d := v.Duel
	v.TM.Paused = !v.myTurn()
	if v.TM.Busy() {
		return
	}
	if d.Over() {
		v.finishDuel()
		return
	}
	if v.myTurn() {
		for i, key := range []rl.Key{ rl.KeyOne, rl.KeyTwo, rl.KeyThree, rl.KeyFour, rl.KeyFive, rl.KeySix } {
			if rl.IsKeyPressed(key) && i < len(d.Sides[rpg.Player].Spells) {
				v.cast(d.Sides[rpg.Player].Spells[i])
			}
		}
		return
	}
	v.aiTimer -= dt
	if v.aiTimer > 0 {
		return
	}
	if v.aiShowing {
		v.aiShowing = false
		v.TM.PlaySwap(v.aiMove[0], v.aiMove[1])
		return
	}
	if s := d.ChooseSpell(); s != nil {
		v.cast(s)
		return
	}
	a, c, ok := d.ChooseMove()
	if !ok {
		d.EndTurn()
		v.aiTimer = AI_THINK_TIME
		return
	}
	v.aiMove = [2]board.Point{ a, c }
	v.aiShowing = true
	v.aiTimer = AI_SHOW_TIME
	v.TM.ShowMove(a, c)
}

// cast runs a spell for the side to move, then resolves whatever it did to the board.
func (v *QuestView) cast(s *rpg.Spell) {
	side := v.Duel.Turn
	if v.TM.Busy() || !v.Duel.Cast(s) {
		return
	}
	v.effects.Popup(s.Name+"!", rl.Vector2{ X: v.portraits[side].X, Y: v.portraits[side].Y - 40 }, 30, ui.Accent)
	r := v.TM.Rect()
	v.TM.Effects.Flash(flash{ kind: board.Prism, center: rl.Vector2{ X: r.X + r.Width/2, Y: r.Y + r.Height/2 }, radius: r.Width * 0.25, life: 0.3 })
	v.TM.RunMove()
}

func (v *QuestView) flee() {
	v.Duel.Winner = rpg.Enemy
}

func (v *QuestView) finishDuel() {
	if v.settled {
		return
	}
	v.settled = true
	d, h := v.Duel, v.Hero
	levelBefore := h.Level
	r := d.Settle(h)
	v.lastWon = r.Won
	back := []ModalButton{ { "Continue", func() { v.Screen = ScreenWorld; v.Duel = nil } } }
	if r.Won {
		v.show(Modal{
			Title:	"Victory!",
			Body:	fmt.Sprintf("%s is defeated. You collect %d gold and %d XP.", d.Sides[rpg.Enemy].Name, r.Gold, r.XP),
		})
		for _, q := range h.RecordKill(v.foeID, v.duelAt) {
			body := fmt.Sprintf("%s\n\nReward: %d gold, %d XP.", q.Outro, q.Gold, q.XP)
			if it := rpg.ItemByID(q.Item); it != nil {
				body += fmt.Sprintf(" You also gain %s (%s): equip it in the shop.", it.Name, it.Desc)
			}
			v.show(Modal{ Title: "Quest complete: " + q.Title, Body: body })
			v.scenes.Play(q.DoneScene)
		}
	} else {
		v.show(Modal{
			Title:	"Defeated",
			Body:	fmt.Sprintf("You limp away with %d gold and %d XP from the board. Try again when you're ready, nothing else was lost.", r.Gold, r.XP),
		})
	}
	if h.Level > levelBefore {
		v.show(Modal{
			Title:	fmt.Sprintf("Level %d!", h.Level),
			Body:	fmt.Sprintf("You have %d points to spend. Open Hero to raise your stats, and Spells to check for new ones.", h.Points),
		})
	}
	v.modals[len(v.modals)-1].Buttons = back
	v.save()
}

func barTop(r rl.Rectangle) rl.Vector2 {
	return rl.Vector2{ X: r.X + r.Width*0.7, Y: r.Y - 2 }
}

func (v *QuestView) DrawEffects() { v.effects.Draw() }

// DrawScreenFlash tints the screen red when the hero gets hit.
func (v *QuestView) DrawScreenFlash() {
	if v.sideFlash > 0 {
		rl.DrawRectangle(0, 0, int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()), rl.Fade(hpColor, 0.16*v.sideFlash))
	}
}

// DrawModal draws the front dialog, and the next one when that is dismissed. Without dialogs it
// draws the playing scene over full, the whole window.
func (v *QuestView) DrawModal(area, full rl.Rectangle) {
	if len(v.modals) == 0 {
		if v.scenes.Showing() {
			v.scenes.Draw(full, v.time)
		}
		return
	}
	m := v.modals[0]
	rl.DrawRectangleRec(rl.Rectangle{ X: -50, Y: -50, Width: 6000, Height: 4000 }, rl.Fade(rl.Black, 0.55))
	w, h := float32(640), float32(180)
	measure := ui.Column{ W: w - 60, Gap: 4 }
	measure.Y = ui.MeasureParagraph(m.Body, 20, measure.W)
	h += measure.Y
	box := rl.Rectangle{ X: area.X + (area.Width-w)/2, Y: area.Y + (area.Height-h)/2, Width: w, Height: h }
	ui.Panel(box)
	title := float32(38)
	for title > 20 && ui.TextWidth(m.Title, title) > w-40 {
		title--
	}
	ui.TextCentered(m.Title, box.X+w/2, box.Y+24, title, ui.Accent)
	col := ui.Column{ X: box.X + 30, Y: box.Y + 80, W: w - 60, Gap: 4 }
	col.Paragraph(m.Body, 20, ui.TextColor)
	row := ui.Row(rl.Rectangle{ X: box.X + 30, Y: box.Y + h - 66, Width: w - 60, Height: 44 }, len(m.Buttons), 12)
	for i, b := range m.Buttons {
		if ui.Button(row[i], b.Label, false) {
			v.modals = v.modals[1:]
			if b.Action != nil {
				b.Action()
			}
			break
		}
	}
}

// DrawLeft fills the left panel below the mode tabs.
func (v *QuestView) DrawLeft(col *ui.Column, r rl.Rectangle) {
	switch {
	case v.Hero == nil:
		col.Heading("Quest")
		col.Paragraph("Choose your class to begin. Your hero is saved automatically.", 16, ui.DimText)
		v.drawStoryStatus(col)
	case v.InDuel():
		v.drawSide(col, rpg.Player)
		col.Y += 4
		if ui.ButtonIf(col.Next(32), "Hint  [H]", v.myTurn() && !v.TM.Busy()) {
			v.TM.ShowHint()
		}
		if ui.ButtonIf(col.Next(32), "Flee (counts as a loss)", v.myTurn() && !v.TM.Busy()) {
			v.flee()
		}
	default:
		v.drawHeroSummary(col)
	}
}

func (v *QuestView) drawHeroSummary(col *ui.Column) {
	h := v.Hero
	c := rpg.NewHeroCombatant(h)
	col.Heading(fmt.Sprintf("Level %d %s", h.Level, h.Class))
	ui.Bar(col.Next(26), "HP", c.MaxHP, c.MaxHP, hpColor)
	ui.Bar(col.Next(24), "XP", h.XP, rpg.XPToNext(h.Level), xpColor)
	col.Line(fmt.Sprintf("Gold  %d", h.Gold), 20, goldColor)
	col.Line(fmt.Sprintf("Wins  %d    Losses  %d", h.Wins, h.Losses), 16, ui.DimText)
	if v.saveErr {
		col.Line("Saving failed, see F1 log", 16, hpColor)
	}
	v.drawStoryStatus(col)

	col.Heading("Menu")
	here := rpg.LocationByID(h.Location)
	nav := func(s Screen, label string, enabled bool) {
		r := col.Next(38)
		if enabled {
			if ui.Button(r, label, v.Screen == s) {
				v.Screen = s
			}
		} else {
			ui.ButtonIf(r, label, false)
		}
	}
	nav(ScreenWorld, "World map", true)
	label := "Hero"
	if h.Points > 0 {
		label = fmt.Sprintf("Hero  (%d points)", h.Points)
	}
	nav(ScreenHero, label, true)
	nav(ScreenSpells, "Spells", true)
	nav(ScreenQuests, fmt.Sprintf("Quests  (%d)", len(h.Active)), true)
	nav(ScreenShop, "Shop (in towns)", here.Town)
	nav(ScreenCitadel, "Citadel", true)
	if v.Screen == ScreenShop && !here.Town {
		v.Screen = ScreenWorld
	}

	col.Heading("How duels work")
	for _, s := range []string{
		"You and your foe take turns on one board.",
		"Colored gems fill your mana, for spells.",
		"Skulls hurt the foe. +5 skulls hit hard and blast the gems around them.",
		"Gold and stars pay out after the duel.",
		"Match 4 for an extra turn. Match 5 for a turn and a Wild, which completes any mana match and multiplies it.",
	} {
		col.Paragraph(s, 15, ui.DimText)
		col.Y += 3
	}
}

// drawStoryStatus warns about story files that didn't load or have mistakes.
func (v *QuestView) drawStoryStatus(col *ui.Column) {
	if v.storyErr != "" {
		col.Paragraph("Story files didn't load (F1 for details): "+v.storyErr, 14, hpColor)
	} else if v.storyIssues > 0 {
		col.Line(fmt.Sprintf("Story has %d problems, see F1", v.storyIssues), 15, hpColor)
	}
}

// drawSide draws one combatant: portrait, bars, status and spells.
func (v *QuestView) drawSide(col *ui.Column, side int) {
	d := v.Duel
	c := d.Sides[side]
	head := col.Next(30)
	nameCol := ui.TextColor
	if d.Turn == side && !d.Over() {
		nameCol = ui.Accent
		rl.DrawCircleV(rl.Vector2{ X: head.X + head.Width - 10, Y: head.Y + 13 }, 8, rl.Fade(ui.Accent, float32(0.55+0.45*math.Sin(v.time*6))))
	}
	ui.Text(fmt.Sprintf("%s  L%d", c.Name, c.Level), head.X, head.Y, 26, nameCol)

	stage := col.Next(130)
	at := rl.Vector2{ X: stage.X + stage.Width/2, Y: stage.Y + stage.Height/2 }
	v.portraits[side] = at
	if img := foeImage(c.Foe); img.ID != 0 {
		drawContain(img, rl.Rectangle{ X: at.X - 65, Y: stage.Y, Width: 130, Height: 130 })
	} else if c.Foe != nil {
		drawFoe(c.Foe.Shape, foeColor(c.Foe.Color), at, 52, v.time, v.sideHit[side], c.HP > 0)
	} else {
		cls := rpg.ClassByName(v.Hero.Class)
		drawFoe("hero", foeColor(cls.Color), at, 52, v.time, v.sideHit[side], c.HP > 0)
	}

	v.hpBars[side] = col.Next(28)
	ui.Bar(v.hpBars[side], "HP", c.HP, c.MaxHP, hpColor)
	for i := range c.Mana {
		r := col.Next(22)
		if i == 0 {
			v.manaBars[side] = r
		}
		ui.Bar(r, rpg.ManaNames[i], c.Mana[i], c.MaxMana[i], manaColors[i])
	}
	ui.Text(fmt.Sprintf("Gold %d", c.Gold), col.X, col.Y+2, 16, goldColor)
	ui.Text(fmt.Sprintf("XP %d", c.XP), col.X+150, col.Y+2, 16, xpColor)
	col.Y += 24

	status := ""
	for _, e := range c.Effects {
		status += fmt.Sprintf("%s(%d) ", rpg.EffectNames[e.Kind], e.Turns)
	}
	if status != "" {
		col.Paragraph(status, 15, ui.Accent)
	}

	col.Heading("Spells")
	for i, s := range c.Spells {
		v.drawSpell(col, c, s, i, side)
	}
}

func (v *QuestView) drawSpell(col *ui.Column, c *rpg.Combatant, s *rpg.Spell, i, side int) {
	r := col.Next(32)
	costW := float32(116)
	btn := rl.Rectangle{ X: r.X, Y: r.Y, Width: r.Width - costW, Height: r.Height }
	label := s.Name
	if side == rpg.Player {
		label = fmt.Sprintf("%d %s", i+1, s.Name)
	}
	if cd := c.Cooldowns[s.ID]; cd > 0 {
		label += fmt.Sprintf(" (%d)", cd)
	}
	enabled := side == rpg.Player && v.myTurn() && !v.TM.Busy() && c.CanCast(s)
	if ui.ButtonIf(btn, label, enabled) {
		v.cast(s)
	}
	x := btn.X + btn.Width + 8
	for m, n := range s.Cost {
		if n > 0 {
			col2 := rl.ColorBrightness(manaColors[m], 0.25)
			if c.Mana[m] < n {
				col2 = rl.Fade(col2, 0.45)
			}
			ui.Text(fmt.Sprint(n), x, r.Y+7, 18, col2)
			x += 27
		}
	}
	col.Paragraph(s.Desc, 13, ui.DimText)
}

// DrawRight fills the right panel: the foe in a duel, and context otherwise.
func (v *QuestView) DrawRight(r rl.Rectangle) {
	ui.Panel(r)
	col := ui.Column{ X: r.X + 18, Y: r.Y + 16, W: r.Width - 36, Gap: 6 }
	switch {
	case v.Hero == nil:
		col.Heading("A new hero")
		col.Paragraph(rpg.Story.Blurb, 18, ui.TextColor)
	case v.InDuel():
		v.drawSide(&col, rpg.Enemy)
		col.Heading("Battle log")
		bottom := r.Y + r.Height - 24
		for i, line := range v.Duel.Log {
			if col.Y > bottom {
				break
			}
			c := ui.DimText
			if i == 0 {
				c = ui.TextColor
			}
			col.Paragraph(line, 15, c)
		}
	default:
		v.drawLocation(&col)
	}
}

// foeImage is the foe's picture from assets/story/foes, if there is one.
func foeImage(f *rpg.Foe) rl.Texture2D {
	if f == nil {
		return rl.Texture2D{}
	}
	if f.Image != "" {
		return storyImage("foes", f.Image)
	}
	return storyImage("foes", f.ID)
}

func foeColor(c [3]uint8) rl.Color {
	return rl.Color{ R: c[0], G: c[1], B: c[2], A: 255 }
}

// drawFoe draws a portrait from simple shapes; hit makes it flash and shake.
func drawFoe(shape string, col rl.Color, c rl.Vector2, rad float32, t float64, hit float32, alive bool) {
	if !alive {
		ui.TextCentered("x_x", c.X, c.Y-20, 40, ui.DimText)
		return
	}
	c.Y += float32(math.Sin(t*2)) * 4
	c.X += (rand.Float32()*2 - 1) * 8 * hit
	col = lerp(col, rl.White, hit)
	alpha := float32(1)
	horn := func(lean, height float32) {
		for _, side := range []float32{ -1, 1 } {
			base := rl.Vector2{ X: c.X + side*rad*0.55, Y: c.Y - rad*0.7 }
			tip := rl.Vector2{ X: c.X + side*rad*lean, Y: c.Y - rad*height }
			inner := rl.Vector2{ X: c.X + side*rad*0.2, Y: c.Y - rad*0.9 }
			rl.DrawTriangle(base, tip, inner, col)
			rl.DrawTriangle(base, inner, tip, col)
		}
	}
	switch shape {
	case "blob":
		rl.DrawCircleV(c, rad, col)
		rl.DrawRectangleRec(rl.Rectangle{ X: c.X - rad, Y: c.Y + rad*0.5, Width: rad * 2, Height: rad * 0.5 }, col)
	case "brute":
		rl.DrawRectangleRounded(rl.Rectangle{ X: c.X - rad, Y: c.Y - rad*0.9, Width: rad * 2, Height: rad * 1.9 }, 0.25, 6, col)
	case "ghost":
		alpha = 0.75
		rl.DrawCircleV(c, rad, rl.Fade(col, alpha))
		for i := float32(-1); i <= 1; i++ {
			rl.DrawCircleV(rl.Vector2{ X: c.X + i*rad*0.62, Y: c.Y + rad*0.82 }, rad*0.36, rl.Fade(col, alpha))
		}
	case "beast":
		rl.DrawCircleV(c, rad*0.9, col)
		horn(0.7, 1.15)
	case "dragon":
		rl.DrawCircleV(c, rad, col)
		horn(0.95, 1.5)
	case "hero":
		rl.DrawCircleV(c, rad*0.8, col)
		rl.DrawRing(c, rad*0.8, rad*0.92, 0, 360, 36, ui.Accent)
	default:
		rl.DrawCircleV(c, rad*0.9, col)
		rl.DrawRectangleRounded(rl.Rectangle{ X: c.X - rad*0.9, Y: c.Y + rad*0.45, Width: rad * 1.8, Height: rad * 0.5 }, 0.4, 6, col)
	}
	mouse := rl.GetMousePosition()
	for _, side := range []float32{ -1, 1 } {
		eye := rl.Vector2{ X: c.X + side*rad*0.33, Y: c.Y - rad*0.12 }
		rl.DrawCircleV(eye, rad*0.19, rl.White)
		dx, dy := mouse.X-eye.X, mouse.Y-eye.Y
		if d := float32(math.Hypot(float64(dx), float64(dy))); d > 0 {
			dx, dy = dx/d*rad*0.08, dy/d*rad*0.08
		}
		rl.DrawCircleV(rl.Vector2{ X: eye.X + dx, Y: eye.Y + dy }, rad*0.09, rl.Black)
	}
	if shape != "hero" {
		mouthY := c.Y + rad*0.35
		for i := float32(-2); i < 2; i++ {
			x := c.X + i*rad*0.15
			a, b := rl.Vector2{ X: x, Y: mouthY }, rl.Vector2{ X: x + rad*0.15, Y: mouthY }
			tip := rl.Vector2{ X: x + rad*0.075, Y: mouthY + rad*0.18 }
			rl.DrawTriangle(a, tip, b, rl.White)
			rl.DrawTriangle(a, b, tip, rl.White)
		}
	}
}

func lerp(a, b rl.Color, t float32) rl.Color {
	l := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t) }
	return rl.Color{ R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: a.A }
}
