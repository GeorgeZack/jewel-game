package core

import (
	"fmt"
	"jewel-game/board"
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"
	"math"
	"math/rand"
	"time"

	"jewel-game/rl"
)

const (
	AI_THINK_TIME = 0.7 // pause before the opponent acts, so its turn reads as a turn
	AI_SHOW_TIME  = 0.6 // how long its chosen swap is highlighted
	TRAVEL_TIME   = 0.6 // seconds per road
	AMBUSH_CHANCE = 0.2
)

var (
	hpColor    = rl.Color{R: 220, G: 60, B: 70, A: 255}
	healColor  = rl.Color{R: 80, G: 220, B: 120, A: 255}
	goldColor  = rl.Color{R: 224, G: 120, B: 32, A: 255}
	xpColor    = rl.Color{R: 190, G: 110, B: 255, A: 255}
	manaColors = [rpg.NUM_MANA]rl.Color{
		{R: 215, G: 60, B: 50, A: 255},
		{R: 60, G: 115, B: 235, A: 255},
		{R: 245, G: 220, B: 90, A: 255},
		{R: 60, G: 175, B: 80, A: 255},
	}
	// manaGems are the stills drawn beside a spell's cost. Same order as manaColors.
	manaGems = [rpg.NUM_MANA]string{"Rupee", "Sapphire", "Crystal", "Emerald"}
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
	Label  string
	Action func()
}

// Modal is a dialog that takes over input until a button is pressed.
type Modal struct {
	Title, Body string
	Buttons     []ModalButton
}

// QuestView runs the RPG: the world and its menus, and duels on a shared board.
type QuestView struct {
	DM          *DataManager
	TM          *TileManager
	Hero        *rpg.Hero
	Screen      Screen
	Duel        *rpg.Duel
	modals      []Modal
	scenes      SceneRunner
	effects     Effects
	time        float64
	saveErr     bool
	storyErr    string // why the story on disk didn't load
	storyIssues int    // content mistakes found while loading
	lastWon     bool

	// the current duel
	foeID, duelAt string
	settled       bool
	aiTimer       float32
	aiMove        [2]board.Point
	aiShowing     bool
	sideHit       [2]float32
	blood         [2]float32 // how much blood is still on the portrait
	sideFlash     float32
	hpBars        [2]rl.Rectangle
	manaBars      [2]rl.Rectangle
	portraits     [2]rl.Vector2

	// walking the map
	route   []string
	legFrom string
	legT    float32

	hex hexEditor

	menuScroll  ui.Scroll
	shopScroll ui.Scroll
	leftScroll ui.Scroll
	rightScroll ui.Scroll
	dockScroll  ui.Scroll
	modalScroll ui.Scroll
	modalKey    string
}

func NewQuestView(dm *DataManager) *QuestView {
	v := &QuestView{DM: dm, TM: NewTileManager(rpg.BoardSettings(), dm)}
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
		ui.AddMessage("%s", text.Tf("log.save_read", "Could not read save, starting fresh: %v", err))
	}
	v.Hero = hero
	if hero != nil {
		v.Screen = ScreenWorld
	}
	return v
}

func (v *QuestView) InDuel() bool   { return v.Screen == ScreenDuel }
func (v *QuestView) HasModal() bool { return len(v.modals) > 0 || v.scenes.Showing() }

// ReloadStory reads assets/story again. If the files on disk are broken, the old story stays, or
// on startup the copy built into the game is used.
func (v *QuestView) ReloadStory() {
	problems, err := rpg.LoadStory(storyFiles{}, storyRoot)
	v.storyErr = ""
	if err != nil {
		v.storyErr = err.Error()
		ui.AddMessage("%s", text.Tf("log.story_load", "Story did not load: %v", err))
		if len(rpg.Locations) == 0 {
			problems, err = rpg.LoadStory(storyFiles{embedded: true}, defaultStoryDir)
			if err != nil {
				panic("the built-in story is broken: " + err.Error())
			}
		}
	}
	v.storyIssues = len(problems)
	for _, p := range problems {
		ui.AddMessage("%s", text.Tf("log.story_problem", "Story problem: %s", p))
	}
	if v.storyErr == "" {
		ui.AddMessage("%s", text.Tf("log.story_loaded", "Story loaded: %d locations, %d quests, %d scenes, %d problems", len(rpg.Locations), len(rpg.Quests), len(rpg.Scenes), len(problems)))
	}
	forgetStoryArt()
	if h := v.Hero; h != nil && rpg.LocationByID(h.Location) == nil {
		h.Location = rpg.Story.Start
		v.route = nil
	}
}

func (v *QuestView) show(m Modal) {
	if len(m.Buttons) == 0 {
		m.Buttons = []ModalButton{{text.T("ui.continue", "Continue"), nil}}
	}
	v.modals = append(v.modals, m)
}

func (v *QuestView) save() {
	if v.Hero == nil {
		return
	}
	if err := v.Hero.Save(rpg.SavePath()); err != nil {
		ui.AddMessage("%s", text.Tf("log.save_write", "Could not save: %v", err))
		v.saveErr = true
		return
	}
	v.saveErr = false
}

func (v *QuestView) Update() {
	noteMapSave()
	v.devTour()
	dt := rl.GetFrameTime()
	v.time += float64(dt)
	v.effects.Update(dt)
	v.sideFlash = max(0, v.sideFlash-dt*2)
	for i := range v.sideHit {
		v.sideHit[i] = max(0, v.sideHit[i]-dt*3)
		v.blood[i] = max(0, v.blood[i]-dt*0.4)
	}
	if v.scenes.WaitingForDuel() && v.Screen != ScreenDuel && len(v.modals) == 0 {
		v.scenes.DuelDone(v.lastWon)
	}
	v.thankOnce()
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
		v.updateHexEdit()
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
			Title:   text.T("ui.ambush", "Ambush!"),
			Body:    text.Tf("ui.ambush_body", "A %s jumps you on the road into %s.", foe.Name, here.Name),
			Buttons: []ModalButton{{text.T("ui.fight", "Fight"), func() { v.startDuel(foe.ID, roamLevel(here, v.Hero), here.ID) }}},
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
	v.sideHit = [2]float32{}
	v.blood = [2]float32{}
	v.sideFlash = 0
	v.TM.Reset(rpg.BoardSettings())
	d := rpg.NewDuel(rpg.NewHeroCombatant(v.Hero), rpg.NewFoeCombatant(rpg.FoeByID(foeID), level), time.Now().UnixNano())
	d.Board = v.TM.Board
	d.OnHit = v.onHit
	d.OnHeal = func(side, n int) {
		at := v.portraits[side]
		if at.X == 0 && at.Y == 0 {
			at = barTop(v.hpBars[side])
		}
		v.effects.Popup(text.Tf("ui.plus", "+%d", n), at, 30, healColor)
		v.effects.Burst(at, healColor, 16, 130)
		v.effects.Flash(flash{center: at, radius: 64, life: 0.45, col: healColor})
	}
	v.Duel = d
	v.Screen = ScreenDuel
	v.aiShowing = false
	v.aiTimer = AI_THINK_TIME + 0.6 // let the board land
}

func (v *QuestView) onHit(side, n int) {
	v.sideHit[side] = 1
	v.blood[side] = min(1, v.blood[side]+0.25+float32(n)/40)
	v.effects.Popup(text.Tf("ui.minus", "-%d", n), barTop(v.hpBars[side]), 22, hpColor)
	v.effects.Burst(v.portraits[side], rl.Color{R: 150, G: 16, B: 24, A: 255}, 8+min(n, 14), 140)
	v.TM.Effects.Slash(v.TM.Rect(), min(1, float32(n)/12))
	if side == rpg.Player {
		v.sideFlash = 1
	}
}

func (v *QuestView) onStep(res *board.StepResult) {
	if v.Duel == nil {
		return
	}
	g := v.Duel.ApplyStep(res)
	for _, c := range res.Cleared {
		if c.Gem.Color == rpg.Skull {
			v.TM.Effects.Burst(v.TM.pointCenter(c.Pos), hpColor, 8, v.TM.CellSize*6)
		}
	}
	for i, n := range g.Mana {
		col := rl.ColorBrightness(manaColors[i], 0.35)
		v.floatGain(res.Cleared, i, n, 24, col, func(n int) string {
			return text.Tf("ui.plus_mana", "+%d %s", n, rpg.ManaLabel(i))
		})
	}
	v.floatGain(res.Cleared, rpg.Gold, g.Gold, 24, goldColor, func(n int) string {
		return text.Tf("ui.plus_gold", "+%d gold", n)
	})
	v.floatGain(res.Cleared, rpg.Star, g.XP, 24, xpColor, func(n int) string {
		return text.Tf("ui.plus_xp", "+%d XP", n)
	})
	v.floatGain(res.Cleared, rpg.Skull, g.Damage, 36, hpColor, func(n int) string {
		return text.Tf("ui.minus", "-%d", n)
	})
	if g.Extra {
		r := v.TM.Rect()
		v.TM.Effects.Popup(text.T("ui.extra_turn", "Extra turn!"), rl.Vector2{X: r.X + r.Width/2, Y: r.Y + r.Height/2}, 46, ui.Accent)
	}
}

// floatGain raises one number from each cluster of color. The total is split by how many gems are in each cluster.
func (v *QuestView) floatGain(cleared []board.Cleared, color, total int, size float32, col rl.Color, label func(int) string) {
	if total <= 0 {
		return
	}
	groups := colorClusters(cleared, color)
	if len(groups) == 0 {
		r := v.TM.Rect()
		v.TM.Effects.Popup(label(total), rl.Vector2{X: r.X + r.Width/2, Y: r.Y + r.Height/2}, size, col)
		return
	}
	weights := make([]int, len(groups))
	for i, g := range groups {
		weights[i] = g.weight
	}
	parts := splitAmounts(total, weights)
	for i, g := range groups {
		if parts[i] <= 0 {
			continue
		}
		var sum rl.Vector2
		for _, p := range g.cells {
			c := v.TM.pointCenter(p)
			sum.X += c.X
			sum.Y += c.Y
		}
		n := float32(len(g.cells))
		sz := size
		if color == rpg.Skull {
			sz = min(54, size+float32(parts[i])/3)
		}
		v.TM.Effects.Popup(label(parts[i]), rl.Vector2{X: sum.X / n, Y: sum.Y / n}, sz, col)
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
	auto := devAutoplay()
	v.TM.Paused = !v.myTurn() || auto
	if v.TM.Busy() {
		return
	}
	if d.Over() {
		v.finishDuel()
		return
	}
	if v.myTurn() && !auto {
		for i, key := range []rl.Key{rl.KeyOne, rl.KeyTwo, rl.KeyThree, rl.KeyFour, rl.KeyFive, rl.KeySix} {
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
	v.aiMove = [2]board.Point{a, c}
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
	v.effects.Popup(text.Tf("ui.cast_popup", "%s!", s.Label()), rl.Vector2{X: v.portraits[side].X, Y: v.portraits[side].Y - 40}, 30, ui.Accent)
	r := v.TM.Rect()
	v.TM.Effects.Flash(flash{kind: board.Prism, center: rl.Vector2{X: r.X + r.Width/2, Y: r.Y + r.Height/2}, radius: r.Width * 0.25, life: 0.3})
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
	back := []ModalButton{{text.T("ui.continue", "Continue"), func() { v.Screen = ScreenWorld; v.Duel = nil }}}
	if r.Won {
		v.show(Modal{
			Title: text.T("ui.victory", "Victory!"),
			Body:  text.Tf("ui.victory_body", "%s is defeated. You collect %d gold and %d XP.", d.Sides[rpg.Enemy].Name, r.Gold, r.XP),
		})
		for _, q := range h.RecordKill(v.foeID, v.duelAt) {
			body := text.Tf("ui.quest_done_body", "%s\n\nReward: %d gold, %d XP.", q.Outro, q.Gold, q.XP)
			if it := rpg.ItemByID(q.Item); it != nil {
				body += text.Tf("ui.quest_done_item", " You also gain %s (%s): equip it in the shop.", it.Label(), it.About())
			}
			v.show(Modal{Title: text.Tf("ui.quest_done", "Quest complete: %s", q.Title), Body: body})
			v.scenes.Play(q.DoneScene)
		}
	} else {
		v.show(Modal{
			Title: text.T("ui.defeated", "Defeated"),
			Body:  text.Tf("ui.defeated_body", "You limp away with %d gold and %d XP from the board. Try again when you're ready, nothing else was lost.", r.Gold, r.XP),
		})
	}
	if h.Level > levelBefore {
		v.show(Modal{
			Title: text.Tf("ui.level_up", "Level %d!", h.Level),
			Body:  text.Tf("ui.level_up_body", "You have %d points to spend. Open Hero to raise your stats, and Spells to check for new ones.", h.Points),
		})
	}
	v.modals[len(v.modals)-1].Buttons = back
	v.save()
}

func barTop(r rl.Rectangle) rl.Vector2 {
	return rl.Vector2{X: r.X + r.Width*0.7, Y: r.Y - 2}
}

func (v *QuestView) DrawEffects() {
	v.effects.DrawFlashes(rl.Vector2{})
	v.effects.Draw()
}

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
	if key := m.Title + "\n" + m.Body; key != v.modalKey {
		v.modalKey = key
		v.modalScroll = ui.Scroll{}
	}
	rl.DrawRectangleRec(rl.Rectangle{X: -50, Y: -50, Width: 6000, Height: 4000}, rl.Fade(rl.Black, 0.55))
	pad := float32(28)
	bodySize := float32(20)
	if narrow() {
		pad = 16
		bodySize = 18
	}
	w := min(float32(640), full.Width-24)
	bodyW := w - pad*2
	bodyH := ui.MeasureParagraph(m.Body, bodySize, bodyW)
	nBtn := max(1, len(m.Buttons))
	btnH := float32(44)
	btnBlock := btnH
	if narrow() && nBtn > 1 {
		btnBlock = float32(nBtn)*btnH + float32(nBtn-1)*8
	}
	h := 24 + 36 + 12 + bodyH + 16 + btnBlock + 16
	host := area
	if narrow() {
		host = full
	}
	if h > full.Height-16 {
		h = full.Height - 16
	}
	box := rl.Rectangle{X: host.X + (host.Width-w)/2, Y: host.Y + (host.Height-h)/2, Width: w, Height: h}
	if box.Y < 8 {
		box.Y = 8
	}
	if box.Y+box.Height > full.Height-8 {
		box.Y = max(float32(8), full.Height-8-box.Height)
	}
	ui.Panel(box)
	ui.TextCenteredFit(m.Title, box.X+w/2, box.Y+16, 34, w-pad*2, ui.Accent)
	btnTop := box.Y + box.Height - 16 - btnBlock
	view := rl.Rectangle{X: box.X + pad, Y: box.Y + 58, Width: bodyW, Height: btnTop - 12 - (box.Y + 58)}
	if view.Height < 36 {
		view.Height = 36
	}
	col := ui.Column{X: view.X, Y: view.Y, W: view.Width, Gap: 4}
	col.Y = v.modalScroll.Begin(view, uiScale())
	if v.modalScroll.Overflows() {
		col.W -= 14
	}
	col.Paragraph(m.Body, bodySize, ui.TextColor)
	v.modalScroll.End(col.Y)
	var row []rl.Rectangle
	if narrow() && nBtn > 1 {
		row = make([]rl.Rectangle, nBtn)
		for i := range row {
			row[i] = rl.Rectangle{X: box.X + pad, Y: btnTop + float32(i)*(btnH+8), Width: bodyW, Height: btnH}
		}
	} else {
		row = ui.Row(rl.Rectangle{X: box.X + pad, Y: btnTop, Width: bodyW, Height: btnH}, nBtn, 12)
	}
	for i, b := range m.Buttons {
		if ui.Button(row[i], b.Label, false) {
			v.modals = v.modals[1:]
			v.TM.HoldInput()
			if b.Action != nil {
				b.Action()
			}
			break
		}
	}
}

// DrawLeft fills the left panel below the mode tabs.
// Everything below the title scrolls when it doesn't fit, as in a duel with many spells.
func (v *QuestView) DrawLeft(col *ui.Column, r rl.Rectangle) {
	view := rl.Rectangle{X: r.X + 4, Y: col.Y, Width: r.Width - 8, Height: r.Y + r.Height - 14 - col.Y}
	col.Y = v.leftScroll.Begin(view, uiScale())
	if v.leftScroll.Overflows() {
		col.W -= 14
	}
	bottom := col.Y + view.Height
	defer func() { v.leftScroll.End(col.Y) }()
	switch {
	case v.Hero == nil:
		col.Heading(text.T("ui.quest_heading", "Quest"))
		col.Paragraph(text.T("ui.quest_start", "Choose your class to begin. Your hero is saved automatically."), 16, ui.DimText)
		v.drawStoryStatus(col)
	case v.InDuel():
		v.drawSide(col, rpg.Player)
		col.Y += 4
		if ui.ButtonIf(col.Next(32), text.T("ui.hint_duel", "Hint  [H]"), v.myTurn() && !v.TM.Busy()) {
			v.TM.ShowHint()
		}
		if ui.ButtonIf(col.Next(32), text.T("ui.flee", "Flee (counts as a loss)"), v.myTurn() && !v.TM.Busy()) {
			v.flee()
		}
	default:
		v.drawHeroSummary(col, bottom)
	}
}

// drawHeroSummary fills the column down to bottom, dropping guide lines that don't fit.
func (v *QuestView) drawHeroSummary(col *ui.Column, bottom float32) {
	h := v.Hero
	c := rpg.NewHeroCombatant(h)
	col.Heading(text.Tf("ui.level_class", "Level %d %s", h.Level, rpg.ClassByName(h.Class).Label()))
	ui.Bar(col.Next(26), text.T("ui.hp", "HP"), c.MaxHP, c.MaxHP, hpColor)
	ui.Bar(col.Next(24), text.T("ui.xp", "XP"), h.XP, rpg.XPToNext(h.Level), xpColor)
	col.Line(text.Tf("ui.gold_gap", "Gold  %d", h.Gold), 20, goldColor)
	col.Line(text.Tf("ui.wins", "Wins  %d    Losses  %d", h.Wins, h.Losses), 16, ui.DimText)
	if v.saveErr {
		col.Line(text.T("ui.save_failed_short", "Saving failed, see F1 log"), 16, hpColor)
	}
	v.drawStoryStatus(col)

	col.Heading(text.T("ui.menu", "Menu"))
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
	nav(ScreenWorld, text.T("ui.world_map", "World map"), true)
	label := text.T("ui.hero", "Hero")
	if h.Points > 0 {
		label = text.Tf("ui.hero_points", "Hero  (%d points)", h.Points)
	}
	nav(ScreenHero, label, true)
	nav(ScreenSpells, text.T("ui.spells", "Spells"), true)
	nav(ScreenQuests, text.Tf("ui.quests_count", "Quests  (%d)", len(h.Active)), true)
	nav(ScreenShop, text.T("ui.shop_towns", "Shop (in towns)"), here.Town)
	nav(ScreenCitadel, text.T("ui.citadel_nav", "Citadel"), true)
	if v.Screen == ScreenShop && !here.Town {
		v.Screen = ScreenWorld
	}

	guide := []string{
		text.T("ui.duel_1", "You and your foe take turns on one board."),
		text.T("ui.duel_2", "Colored gems fill your mana, for spells."),
		text.T("ui.duel_3", "Skulls hurt the foe. +5 skulls hit hard and blast the gems around them."),
		text.T("ui.duel_4", "Gold and stars pay out after the duel."),
		text.T("ui.duel_5", "Match 4 for an extra turn. Match 5 for a turn and a Wild, which completes any mana match and multiplies it."),
	}
	if col.Y+40+ui.MeasureParagraph(guide[0], 15, col.W) > bottom {
		return
	}
	col.Heading(text.T("ui.how_duels", "How duels work"))
	for _, s := range guide {
		if col.Y+ui.MeasureParagraph(s, 15, col.W) > bottom {
			break
		}
		col.Paragraph(s, 15, ui.DimText)
		col.Y += 3
	}
}

// drawStoryStatus warns about story files that didn't load or have mistakes.
func (v *QuestView) drawStoryStatus(col *ui.Column) {
	if v.storyErr != "" {
		col.Paragraph(text.Tf("ui.story_broken", "Story files didn't load (F1 for details): %s", v.storyErr), 14, hpColor)
	} else if v.storyIssues > 0 {
		col.Line(text.Tf("ui.story_problems", "Story has %d problems, see F1", v.storyIssues), 15, hpColor)
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
		rl.DrawCircleV(rl.Vector2{X: head.X + head.Width - 10, Y: head.Y + 13}, 8, rl.Fade(ui.Accent, float32(0.55+0.45*math.Sin(v.time*6))))
	}
	ui.Text(text.Tf("ui.level_short", "%s  L%d", c.Name, c.Level), head.X, head.Y, 26, nameCol)

	stage := col.Next(130)
	at := rl.Vector2{X: stage.X + stage.Width/2, Y: stage.Y + stage.Height/2}
	v.portraits[side] = at
	hit := v.sideHit[side]
	if img := foeImage(c.Foe); img.ID != 0 {
		sx := (rand.Float32()*2 - 1) * 8 * hit
		sy := (rand.Float32()*2 - 1) * 4 * hit
		drawContain(img, rl.Rectangle{X: at.X - 65 + sx, Y: stage.Y + sy, Width: 130, Height: 130})
	} else if c.Foe != nil {
		drawFoe(c.Foe.Shape, foeColor(c.Foe.Color), at, 52, v.time, hit, c.HP > 0)
	} else {
		cls := rpg.ClassByName(v.Hero.Class)
		drawFoe("hero", foeColor(cls.Color), at, 52, v.time, hit, c.HP > 0)
	}
	if hit > 0 {
		rl.DrawRectangleRec(rl.Rectangle{X: stage.X, Y: stage.Y, Width: stage.Width, Height: stage.Height}, rl.Fade(hpColor, 0.22*hit))
	}
	drawBlood(at, v.blood[side])

	v.hpBars[side] = col.Next(28)
	ui.Bar(v.hpBars[side], text.T("ui.hp", "HP"), c.HP, c.MaxHP, hpColor)
	if hit > 0 {
		rl.DrawRectangleRec(v.hpBars[side], rl.Fade(rl.White, 0.45*hit))
	}
	for i := range c.Mana {
		r := col.Next(26)
		if i == 0 {
			v.manaBars[side] = r
		}
		ui.Bar(r, rpg.ManaLabel(i), c.Mana[i], c.MaxMana[i], manaColors[i])
	}
	ui.Text(text.Tf("ui.gold", "Gold %d", c.Gold), col.X, col.Y+2, 16, goldColor)
	ui.Text(text.Tf("ui.xp_amount", "XP %d", c.XP), col.X+150, col.Y+2, 16, xpColor)
	col.Y += 24

	status := ""
	for _, e := range c.Effects {
		status += text.Tf("ui.effect", "%s(%d) ", rpg.EffectLabel(e.Kind), e.Turns)
	}
	if status != "" {
		col.Paragraph(status, 15, ui.Accent)
	}

	col.Heading(text.T("ui.spells", "Spells"))
	for i, s := range c.Spells {
		v.drawSpell(col, c, s, i, side)
	}
}

func (v *QuestView) drawSpell(col *ui.Column, c *rpg.Combatant, s *rpg.Spell, i, side int) {
	h := float32(32)
	if narrow() {
		h = 46
	}
	r := col.Next(h)
	const size = 18
	costW := manaCostWidth(s.Cost, size)
	gap := float32(8)
	if costW > 0 {
		costW += gap
	}
	btn := rl.Rectangle{X: r.X, Y: r.Y, Width: r.Width - costW, Height: r.Height}
	label := s.Label()
	if side == rpg.Player {
		label = text.Tf("ui.spell_button", "%d %s", i+1, s.Label())
	}
	if cd := c.Cooldowns[s.ID]; cd > 0 {
		label += text.Tf("ui.cooldown", " (%d)", cd)
	}
	enabled := side == rpg.Player && v.myTurn() && !v.TM.Busy() && c.CanCast(s)
	if ui.ButtonIf(btn, label, enabled) {
		v.cast(s)
	}
	if costW > gap {
		v.drawManaCosts(btn.X+btn.Width+gap, r.Y+7, size, s.Cost, &c.Mana)
	}
	col.Paragraph(s.About(), 13, ui.DimText)
}

// manaCostWidth is the width of the gem-and-number run for the nonzero costs.
func manaCostWidth(cost [rpg.NUM_MANA]int, size float32) float32 {
	var w float32
	n := 0
	for _, c := range cost {
		if c <= 0 {
			continue
		}
		if n > 0 {
			w += 8
		}
		w += size + 3 + ui.TextWidth(fmt.Sprint(c), size)
		n++
	}
	return w
}

// drawManaCosts draws each nonzero cost as a gem icon plus its number, left to right.
// have is the caster's mana. Nil means every number stays solid, as in the spellbook.
// A cost the caster cannot pay fades the number and leaves the icon solid.
func (v *QuestView) drawManaCosts(x, y, size float32, cost [rpg.NUM_MANA]int, have *[rpg.NUM_MANA]int) {
	for m, n := range cost {
		if n <= 0 {
			continue
		}
		v.drawManaIcon(m, x, y, size)
		num := fmt.Sprint(n)
		col := rl.ColorBrightness(manaColors[m], 0.25)
		if have != nil && have[m] < n {
			col = rl.Fade(col, 0.6)
		}
		ui.Text(num, x+size+3, y, size, col)
		x += size + 3 + ui.TextWidth(num, size) + 8
	}
}

// drawManaIcon draws the element's gem still. A missing texture becomes a diamond in that color.
func (v *QuestView) drawManaIcon(m int, x, y, size float32) {
	dst := rl.Rectangle{X: x, Y: y, Width: size, Height: size}
	var tex rl.Texture2D
	if v.DM != nil {
		if gt, ok := v.DM.GemTypes[manaGems[m]]; ok {
			tex = gt.Texture
		}
	}
	if tex.ID == 0 || tex.Width == 0 {
		rl.DrawPoly(rl.Vector2{X: x + size/2, Y: y + size/2}, 4, size*0.48, 0, manaColors[m])
		return
	}
	src := rl.Rectangle{Width: float32(tex.Width), Height: float32(tex.Height)}
	rl.DrawTexturePro(tex, src, dst, rl.Vector2{}, 0, rl.White)
}

// DrawRight fills the right panel: the foe in a duel, and context otherwise.
func (v *QuestView) DrawRight(r rl.Rectangle) {
	ui.Panel(r)
	col := ui.Column{X: r.X + 18, Y: r.Y + 16, W: r.Width - 36, Gap: 6}
	if narrow() {
		col.X, col.W = r.X+38, r.Width-84
		view := rl.Rectangle{X: r.X + 18, Y: r.Y + 30, Width: r.Width - 36, Height: r.Height - 46}
		col.Y = v.rightScroll.Begin(view, uiScale())
		defer func() { v.rightScroll.End(col.Y) }()
	}
	switch {
	case v.Hero == nil:
		col.Heading(text.T("ui.new_hero_heading", "A new hero"))
		col.Paragraph(rpg.Story.Blurb, 18, ui.TextColor)
	case v.InDuel():
		v.drawSide(&col, rpg.Enemy)
		col.Heading(text.T("ui.battle_log", "Battle log"))
		bottom := r.Y + r.Height - 24
		for i, line := range v.Duel.Log {
			if col.Y > bottom {
				break
			}
			c := ui.DimText
			if i == 0 {
				c = ui.TextColor
			}
			col.Paragraph(line.Text(), 15, c)
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

// drawBlood paints a stable splatter on a portrait. amount fades from 1 down to 0.
func drawBlood(at rl.Vector2, amount float32) {
	if amount <= 0.02 {
		return
	}
	drops := [...]struct {
		x, y, r, drip float32
	}{
		{-22, -8, 8, 22},
		{18, 4, 10, 30},
		{2, 16, 7, 14},
		{-30, 18, 5, 10},
		{28, -14, 4, 0},
		{-8, 28, 6, 24},
		{14, -24, 3.5, 0},
		{-2, 0, 9, 12},
		{34, 20, 4, 16},
		{-18, -26, 3, 0},
	}
	n := 3 + int(amount*float32(len(drops)-3))
	base := rl.Color{R: 132, G: 10, B: 18, A: 255}
	hot := rl.Color{R: 196, G: 36, B: 40, A: 255}
	for i := 0; i < n; i++ {
		d := drops[i]
		a := min(1, amount*1.25) * (0.6 + 0.13*float32(i%3))
		p := rl.Vector2{X: at.X + d.x, Y: at.Y + d.y}
		rl.DrawCircleV(p, d.r, rl.Fade(base, a))
		rl.DrawCircleV(rl.Vector2{X: p.X - d.r*0.28, Y: p.Y - d.r*0.28}, d.r*0.38, rl.Fade(hot, a*0.75))
		if d.drip > 0 {
			end := rl.Vector2{X: p.X + d.r*0.15, Y: p.Y + d.drip*amount}
			rl.DrawLineEx(p, end, max(1.6, d.r*0.42), rl.Fade(base, a))
			rl.DrawCircleV(end, d.r*0.42, rl.Fade(base, a))
		}
	}
}

func foeColor(c [3]uint8) rl.Color {
	return rl.Color{R: c[0], G: c[1], B: c[2], A: 255}
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
		for _, side := range []float32{-1, 1} {
			base := rl.Vector2{X: c.X + side*rad*0.55, Y: c.Y - rad*0.7}
			tip := rl.Vector2{X: c.X + side*rad*lean, Y: c.Y - rad*height}
			inner := rl.Vector2{X: c.X + side*rad*0.2, Y: c.Y - rad*0.9}
			rl.DrawTriangle(base, tip, inner, col)
			rl.DrawTriangle(base, inner, tip, col)
		}
	}
	switch shape {
	case "blob":
		rl.DrawCircleV(c, rad, col)
		rl.DrawRectangleRec(rl.Rectangle{X: c.X - rad, Y: c.Y + rad*0.5, Width: rad * 2, Height: rad * 0.5}, col)
	case "brute":
		rl.DrawRectangleRounded(rl.Rectangle{X: c.X - rad, Y: c.Y - rad*0.9, Width: rad * 2, Height: rad * 1.9}, 0.25, 6, col)
	case "ghost":
		alpha = 0.75
		rl.DrawCircleV(c, rad, rl.Fade(col, alpha))
		for i := float32(-1); i <= 1; i++ {
			rl.DrawCircleV(rl.Vector2{X: c.X + i*rad*0.62, Y: c.Y + rad*0.82}, rad*0.36, rl.Fade(col, alpha))
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
		rl.DrawRectangleRounded(rl.Rectangle{X: c.X - rad*0.9, Y: c.Y + rad*0.45, Width: rad * 1.8, Height: rad * 0.5}, 0.4, 6, col)
	}
	mouse := rl.GetMousePosition()
	for _, side := range []float32{-1, 1} {
		eye := rl.Vector2{X: c.X + side*rad*0.33, Y: c.Y - rad*0.12}
		rl.DrawCircleV(eye, rad*0.19, rl.White)
		dx, dy := mouse.X-eye.X, mouse.Y-eye.Y
		if d := float32(math.Hypot(float64(dx), float64(dy))); d > 0 {
			dx, dy = dx/d*rad*0.08, dy/d*rad*0.08
		}
		rl.DrawCircleV(rl.Vector2{X: eye.X + dx, Y: eye.Y + dy}, rad*0.09, rl.Black)
	}
	if shape != "hero" {
		mouthY := c.Y + rad*0.35
		for i := float32(-2); i < 2; i++ {
			x := c.X + i*rad*0.15
			a, b := rl.Vector2{X: x, Y: mouthY}, rl.Vector2{X: x + rad*0.15, Y: mouthY}
			tip := rl.Vector2{X: x + rad*0.075, Y: mouthY + rad*0.18}
			rl.DrawTriangle(a, tip, b, rl.White)
			rl.DrawTriangle(a, b, tip, rl.White)
		}
	}
}

func lerp(a, b rl.Color, t float32) rl.Color {
	l := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t) }
	return rl.Color{R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: a.A}
}

// thankOnce shows a thank-you once the story is beaten and its last scene is over.
func (v *QuestView) thankOnce() {
	h := v.Hero
	if h == nil || !h.Finished || h.Flags["thanked"] || v.scenes.Showing() || len(v.modals) > 0 {
		return
	}
	if h.Flags == nil {
		h.Flags = map[string]bool{}
	}
	h.Flags["thanked"] = true
	v.save()
	v.show(Modal{
		Title:   text.T("ui.thanks_title", "Thanks for playing!"),
		Body:    text.T("ui.thanks_body", "That's the end of this slice of the game. Your hero can keep exploring, and more is on the way."),
		Buttons: []ModalButton{{text.T("ui.keep_playing", "Keep playing"), nil}},
	})
}
