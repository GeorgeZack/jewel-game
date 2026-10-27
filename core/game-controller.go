package core

import (
	"fmt"
	"image/color"
	"jewel-game/board"
	"jewel-game/ui"
	"jewel-game/utils"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Mode int

const (
	ModeClassic Mode = iota
	ModeSandbox
	ModeBattle
	MODE_COUNT
)

type GameController struct {
	DataManager		*DataManager
	Mode			Mode
	Classic			*TileManager
	Sandbox			*TileManager
	Battle			*BattleView
	Quit			bool
	Watcher			*fsnotify.Watcher
	WatcherErrors	chan error
	WatcherEvents	chan fsnotify.Event
	bkg				rl.Texture2D
}

func NewGameController() *GameController {
	dm := NewDataManager()
	ui.Font = dm.Consts.UIFont

	sandbox := NewTileManager(Presets()[1].Settings, dm)
	sandbox.UndoEnabled = true

	gc := &GameController{
		DataManager:	dm,
		Classic:		NewTileManager(board.DefaultSettings(), dm),
		Sandbox:		sandbox,
		Battle:			NewBattleView(dm),
		WatcherErrors:	make(chan error, 1),
		WatcherEvents:	make(chan fsnotify.Event, 10),
		bkg:			rl.LoadTexture("_res/bkg.jpg"),
	}
	if gc.bkg.ID == 0 {
		ui.AddMessage("Background missing or not an image (git lfs pull?), using a gradient")
	}

	// Watch for gem type edits; events are forwarded here and handled in Update so
	// textures are only ever loaded on the main (OpenGL) thread.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		ui.AddMessage("Hot reload disabled, failed to create file watcher: %v", err)
		return gc
	}
	watchDir := filepath.Dir(utils.GEM_TYPES_FILE)
	if err := watcher.Add(watchDir); err != nil {
		ui.AddMessage("Hot reload disabled, failed to watch %s: %v", watchDir, err)
		watcher.Close()
		return gc
	}
	gc.Watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return // Channel closed
				}
				gc.WatcherEvents <- event // Forward event to GameController's channel
			case err, ok := <-watcher.Errors:
				if !ok {
					return // Channel closed
				}
				gc.WatcherErrors <- err // Forward error to GameController's channel
			}
		}
	}()

	return gc
}

func (c *GameController) Active() *TileManager {
	switch c.Mode {
	case ModeSandbox:
		return c.Sandbox
	case ModeBattle:
		return c.Battle.TM
	}
	return c.Classic
}

func (c *GameController) Unload() {
	if c.bkg.ID != 0 {
		rl.UnloadTexture(c.bkg)
	}
	c.DataManager.Unload()
	if c.Watcher != nil {
		// closing the watcher ends the forwarding goroutine
		c.Watcher.Close()
	}
	ui.AddMessage("GameController resources unloaded.")
}

func (c *GameController) Update() {
	select {
	case event := <-c.WatcherEvents:
		if filepath.Clean(event.Name) == filepath.Clean(utils.GEM_TYPES_FILE) && event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
			ui.AddMessage("Detected change in %s, reloading gem types...", event.Name)
			// boards keep their state; gems pick up the new looks on the next frame
			c.DataManager.LoadGemTypes(utils.GEM_TYPES_FILE)
		}
	case err := <-c.WatcherErrors:
		ui.AddMessage("File watcher error: %v", err)
	default:
	}

	c.handleKeys()
	if c.Mode == ModeBattle {
		c.Battle.Update()
	}
	tm := c.Active()
	tm.Layout(c.boardArea())
	tm.Update()
}

func (c *GameController) handleKeys() {
	tm := c.Active()
	switch {
	case rl.IsKeyPressed(rl.KeyF1):
		ui.ShowDebug = !ui.ShowDebug
	case rl.IsKeyPressed(rl.KeyTab):
		c.Mode = (c.Mode + 1) % MODE_COUNT
	case rl.IsKeyPressed(rl.KeyH):
		tm.ShowHint()
	case rl.IsKeyPressed(rl.KeySpace):
		tm.ReleaseLocks()
	case rl.IsKeyPressed(rl.KeyEnter):
		tm.Resolve()
	case rl.IsKeyPressed(rl.KeyR):
		// in battle a new board would be a free escape; R only restarts after defeat
		if c.Mode != ModeBattle {
			tm.Reset(tm.Board.Settings)
		}
	case rl.IsKeyPressed(rl.KeyZ), rl.IsKeyPressed(rl.KeyBackspace):
		tm.Undo()
	case rl.IsKeyPressed(rl.KeyEscape):
		if tm.Brush != nil {
			tm.Brush = nil
		} else if !tm.ClearSelection() {
			c.Quit = true
		}
	}
}

const MARGIN = 20
const DESIGN_HEIGHT = 1058	// side panels are laid out for this height and scaled down in smaller windows

// uiScale shrinks the side panels so they always fit the window height.
func uiScale() float32 {
	return max(0.55, min(1, float32(rl.GetScreenHeight())/DESIGN_HEIGHT))
}

// leftPanel and rightPanel are in panel space, which is screen space divided by uiScale.
func (c *GameController) leftPanel() rl.Rectangle {
	h := float32(rl.GetScreenHeight()) / uiScale()
	return rl.Rectangle{ X: MARGIN, Y: MARGIN, Width: utils.SIDE_PANEL_WIDTH, Height: h - 2*MARGIN }
}

func (c *GameController) rightPanel() rl.Rectangle {
	w, h := float32(rl.GetScreenWidth())/uiScale(), float32(rl.GetScreenHeight())/uiScale()
	return rl.Rectangle{ X: w - MARGIN - utils.SIDE_PANEL_WIDTH, Y: MARGIN, Width: utils.SIDE_PANEL_WIDTH, Height: h - 2*MARGIN }
}

// boardArea is in screen space.
func (c *GameController) boardArea() rl.Rectangle {
	s := uiScale()
	l, r := c.leftPanel(), c.rightPanel()
	x := (l.X+l.Width)*s + 60
	return rl.Rectangle{ X: x, Y: 70, Width: r.X*s - 60 - x, Height: float32(rl.GetScreenHeight()) - 140 }
}

func (c *GameController) Draw() {
	rl.BeginDrawing()
	c.drawBackground()
	tm := c.Active()
	tm.Draw()
	if c.Mode == ModeBattle {
		c.Battle.DrawScreenFlash()
	}

	// draw panels scaled, and scale the mouse to match so the widgets' hit tests line up
	s := uiScale()
	rl.BeginMode2D(rl.Camera2D{ Zoom: s })
	rl.SetMouseScale(1/s, 1/s)
	c.drawLeftPanel(tm)
	switch c.Mode {
	case ModeSandbox:
		DrawSandboxPanel(tm, c.DataManager, c.rightPanel())
	case ModeBattle:
		c.Battle.DrawEnemy(c.rightPanel())
		c.Battle.DrawPanelEffects()
	default:
		c.drawGuide(c.rightPanel())
	}
	rl.SetMouseScale(1, 1)
	rl.EndMode2D()

	if c.Mode == ModeBattle {
		c.Battle.DrawOverlay()
	}

	ui.Draw(c.DataManager.Consts.DebugFont)
	rl.EndDrawing()
}

func (c *GameController) drawBackground() {
	w, h := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	rl.DrawRectangleGradientV(0, 0, w, h, color.RGBA{ 38, 22, 70, 255 }, color.RGBA{ 8, 10, 28, 255 })
	if c.bkg.ID == 0 {
		return
	}
	// cover the window, keeping the photo's aspect ratio
	scale := max(float32(w)/float32(c.bkg.Width), float32(h)/float32(c.bkg.Height))
	dw, dh := float32(c.bkg.Width)*scale, float32(c.bkg.Height)*scale
	src := rl.Rectangle{ Width: float32(c.bkg.Width), Height: float32(c.bkg.Height) }
	dst := rl.Rectangle{ X: (float32(w) - dw) / 2, Y: (float32(h) - dh) / 2, Width: dw, Height: dh }
	rl.DrawTexturePro(c.bkg, src, dst, rl.Vector2{}, 0, color.RGBA{ 255, 255, 255, 105 })
}

func (c *GameController) drawLeftPanel(tm *TileManager) {
	r := c.leftPanel()
	ui.Panel(r)
	col := ui.Column{ X: r.X + 18, Y: r.Y + 16, W: r.Width - 36, Gap: 6 }
	title := col.Next(44)
	ui.Text("JEWEL GAME", title.X, title.Y, 40, ui.Accent)

	tabs := ui.Row(col.Next(36), int(MODE_COUNT), 8)
	for i, name := range []string{ "Classic", "Sandbox", "Battle" } {
		if ui.Button(tabs[i], name, c.Mode == Mode(i)) {
			c.Mode = Mode(i)
		}
	}

	if c.Mode == ModeBattle {
		c.Battle.DrawPlayer(&col)
		col.Heading("Actions")
		if ui.Button(col.Next(32), "Release locks  [Space]", false) {
			tm.ReleaseLocks()
		}
		if ui.Button(col.Next(32), "Hint  [H]", false) {
			tm.ShowHint()
		}
		col.Heading("Controls")
		for _, line := range []string{
			"Drag or click  swap (uses a turn)",
			"Right click    lock / unlock a gem",
			"1 2 3          cast skills",
			"Tab            switch mode",
		} {
			col.Line(line, 15, ui.DimText)
		}
		return
	}

	b := tm.Board
	col.Heading("Score")
	score := col.Next(50)
	ui.Text(fmt.Sprint(b.Score), score.X, score.Y, 48, ui.TextColor)
	col.Line(fmt.Sprintf("Moves      %d", b.Moves), 18, ui.TextColor)
	col.Line(fmt.Sprintf("Chain      %d  (best %d)", b.Chain, b.BestChain), 18, ui.TextColor)
	col.Line(fmt.Sprintf("Cleared    %d", b.Cleared), 18, ui.TextColor)
	locks := fmt.Sprintf("Locks      %d / %s", b.LockCount(), lockLabel(b.Settings.MaxLocks))
	col.Line(locks, 18, ui.TextColor)
	if b.Settings.Gravity != board.Down || b.Settings.ChaosGravity {
		col.Line(fmt.Sprintf("Gravity    %s", b.Settings.Gravity), 18, ui.Accent)
	}

	col.Heading("Actions")
	if ui.Button(col.Next(32), "Release locks  [Space]", false) {
		tm.ReleaseLocks()
	}
	if ui.Button(col.Next(32), "Hint  [H]", false) {
		tm.ShowHint()
	}
	if ui.Button(col.Next(32), "New board  [R]", false) {
		tm.Reset(b.Settings)
	}
	if c.Mode == ModeSandbox {
		if ui.Button(col.Next(32), fmt.Sprintf("Undo  [Z]  (%d)", len(tm.undo)), false) {
			tm.Undo()
		}
		if ui.Button(col.Next(32), "Clear matches  [Enter]", false) {
			tm.Resolve()
		}
	}

	col.Heading("Controls")
	for _, line := range []string{
		"Left drag   swap gems",
		"Left click  select, click to swap",
		"Right click lock / unlock a gem",
		"Tab         switch mode",
		"F1          debug log    Esc  quit",
	} {
		col.Line(line, 15, ui.DimText)
	}
}

func (c *GameController) drawGuide(r rl.Rectangle) {
	ui.Panel(r)
	col := ui.Column{ X: r.X + 18, Y: r.Y + 16, W: r.Width - 36, Gap: 4 }
	para := func(s string) {
		col.Paragraph(s, 16, ui.TextColor)
		col.Y += 6
	}
	col.Heading("Specials")
	para("4 in a row makes a Bolt. It clears its whole row or column.")
	para("An L or T shape makes a Bomb. It blasts the 3x3 around it.")
	para("5 in a row makes a Prism. Swap it with any gem to clear that color.")
	col.Heading("Combos")
	para("Swap two specials into each other:")
	para("Bolt + Bolt: a full cross.")
	para("Bolt + Bomb: a cross three lines wide.")
	para("Bomb + Bomb: a 5x5 blast.")
	para("Prism + Bolt/Bomb: every gem of that color turns into one and goes off.")
	para("Prism + Prism: clears the board.")
	col.Heading("Locks")
	para("Right-click a gem to lock it. Locked gems hang in place: they don't fall, match, or break, and gems fall past them.")
	para("Build setups around them, then press Space to release every lock at once. Several specials made in one step earn a Synchro bonus.")
	col.Y += 8
	col.Paragraph("Tab opens the Sandbox, where every rule is up for grabs.", 16, ui.Accent)
}
