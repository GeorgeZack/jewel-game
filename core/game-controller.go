package core

import (
	"image/color"
	"jewel-game/text"
	"jewel-game/ui"
	"jewel-game/utils"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"jewel-game/rl"
)

type GameController struct {
	DataManager   *DataManager
	Quest         *QuestView
	Quit          bool
	Watcher       *fsnotify.Watcher
	WatcherErrors chan error
	WatcherEvents chan fsnotify.Event
	bkg           rl.Texture2D
}

func NewGameController() *GameController {
	loadLocale()
	dm := NewDataManager()
	ui.Font = dm.Consts.UIFont

	gc := &GameController{
		DataManager:   dm,
		Quest:         NewQuestView(dm),
		WatcherErrors: make(chan error, 1),
		WatcherEvents: make(chan fsnotify.Event, 10),
		bkg:           rl.LoadTexture("_res/bkg.jpg"),
	}
	if gc.bkg.ID == 0 {
		ui.AddMessage("%s", text.T("log.background", "Background missing or not an image (git lfs pull?), using a gradient"))
	}

	// Watch for gem type edits; events are forwarded here and handled in Update so
	// textures are only ever loaded on the main (OpenGL) thread.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		ui.AddMessage("%s", text.Tf("log.watch_create", "Hot reload disabled, failed to create file watcher: %v", err))
		return gc
	}
	watchDir := filepath.Dir(utils.GEM_TYPES_FILE)
	if err := watcher.Add(watchDir); err != nil {
		ui.AddMessage("%s", text.Tf("log.watch_dir", "Hot reload disabled, failed to watch %s: %v", watchDir, err))
		watcher.Close()
		return gc
	}
	gc.Watcher = watcher
	gc.watchExtra()

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

func (c *GameController) Active() *TileManager { return c.Quest.TM }

func (c *GameController) Unload() {
	if c.bkg.ID != 0 {
		rl.UnloadTexture(c.bkg)
	}
	c.DataManager.Unload()
	if c.Watcher != nil {
		// closing the watcher ends the forwarding goroutine
		c.Watcher.Close()
	}
	ui.AddMessage("%s", text.T("log.unloaded", "GameController resources unloaded."))
}

// watchExtra watches the language folder and every story folder. fsnotify is not recursive.
func (c *GameController) watchExtra() {
	if c.Watcher == nil {
		return
	}
	dirs := append([]string{langDir}, storyDirs(defaultStoryDir)...)
	if storyRoot != defaultStoryDir {
		dirs = append(dirs, storyDirs(storyRoot)...)
	}
	for _, dir := range dirs {
		if err := c.Watcher.Add(dir); err != nil {
			ui.AddMessage("%s", text.Tf("log.watch_skip", "Not watching %s: %v", dir, err))
		}
	}
}

func (c *GameController) Update() {
	select {
	case event := <-c.WatcherEvents:
		if filepath.Clean(event.Name) == filepath.Clean(utils.GEM_TYPES_FILE) && event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
			ui.AddMessage("%s", text.Tf("log.watch_gems", "Detected change in %s, reloading gem types...", event.Name))
			// boards keep their state; gems pick up the new looks on the next frame
			c.DataManager.LoadGemTypes(utils.GEM_TYPES_FILE)
		}
		if isLocaleFile(event.Name) && event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
			ui.AddMessage("%s", text.Tf("log.watch_lang", "Detected change in %s, reloading language...", event.Name))
			loadLocale()
			c.watchExtra()
			c.Quest.ReloadStory()
		} else if isStoryFile(event.Name) && event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
			ui.AddMessage("%s", text.Tf("log.watch_story", "Detected change in %s, reloading the story...", event.Name))
			c.Quest.ReloadStory()
		}
	case err := <-c.WatcherErrors:
		ui.AddMessage("%s", text.Tf("log.watch_err", "File watcher error: %v", err))
	default:
	}

	c.handleKeys()
	c.Quest.Update()
	if !c.Quest.InDuel() {
		return
	}
	tm := c.Active()
	tm.Layout(c.boardArea())
	tm.Update()
}

func (c *GameController) handleKeys() {
	tm := c.Active()
	// a dialog or scene takes the keys, apart from the debug log
	busy := c.Quest.HasModal()
	if !busy && c.Quest.hex.on {
		switch {
		case rl.IsKeyPressed(rl.KeyF1):
			ui.ShowDebug = !ui.ShowDebug
		case rl.IsKeyPressed(rl.KeyE), rl.IsKeyPressed(rl.KeyEscape):
			c.Quest.hex.on = false
		case rl.IsKeyPressed(rl.KeyS):
			c.Quest.saveHexMap()
		}
		return
	}
	switch {
	case busy && !rl.IsKeyPressed(rl.KeyF1):
	case rl.IsKeyPressed(rl.KeyF1):
		ui.ShowDebug = !ui.ShowDebug
	case rl.IsKeyPressed(rl.KeyH):
		tm.ShowHint()
	case rl.IsKeyPressed(rl.KeySpace):
		tm.ReleaseLocks()
	case rl.IsKeyPressed(rl.KeyE):
		if c.Quest.Screen == ScreenWorld && !c.Quest.HasModal() {
			c.Quest.hex.on = true
		}
	case rl.IsKeyPressed(rl.KeyEscape):
		if !tm.ClearSelection() {
			c.Quit = true
		}
	}
}

const MARGIN = 20
const DESIGN_HEIGHT = 1058 // side panels are laid out for this height and scaled down in smaller windows

// uiScale shrinks the side panels so they always fit the window height.
func uiScale() float32 {
	return max(0.55, min(1, float32(rl.GetScreenHeight())/DESIGN_HEIGHT))
}

// leftPanel and rightPanel are in panel space, which is screen space divided by uiScale.
func (c *GameController) leftPanel() rl.Rectangle {
	h := float32(rl.GetScreenHeight()) / uiScale()
	return rl.Rectangle{X: MARGIN, Y: MARGIN, Width: utils.SIDE_PANEL_WIDTH, Height: h - 2*MARGIN}
}

func (c *GameController) rightPanel() rl.Rectangle {
	w, h := float32(rl.GetScreenWidth())/uiScale(), float32(rl.GetScreenHeight())/uiScale()
	return rl.Rectangle{X: w - MARGIN - utils.SIDE_PANEL_WIDTH, Y: MARGIN, Width: utils.SIDE_PANEL_WIDTH, Height: h - 2*MARGIN}
}

// centerPanel is the space between the side panels, in panel space.
func (c *GameController) centerPanel() rl.Rectangle { return centerPanelRect() }

// boardArea is in screen space.
func (c *GameController) boardArea() rl.Rectangle {
	s := uiScale()
	l, r := c.leftPanel(), c.rightPanel()
	x := (l.X+l.Width)*s + 60
	return rl.Rectangle{X: x, Y: 70, Width: r.X*s - 60 - x, Height: float32(rl.GetScreenHeight()) - 140}
}

func (c *GameController) Draw() {
	rl.BeginDrawing()
	ui.NewFrame()
	c.drawBackground()
	tm := c.Active()
	if c.Quest.InDuel() {
		tm.Draw()
	}
	c.Quest.DrawScreenFlash()

	// draw panels scaled, and scale the mouse to match so the widgets' hit tests line up
	s := uiScale()
	rl.BeginMode2D(rl.Camera2D{Zoom: s})
	rl.SetMouseScale(1/s, 1/s)
	ui.Blocked = c.Quest.HasModal()
	c.drawLeftPanel()
	c.Quest.DrawCenter(c.centerPanel())
	c.Quest.DrawRight(c.rightPanel())
	c.Quest.DrawEffects()
	ui.Blocked = false
	c.Quest.DrawModal(c.centerPanel(), rl.Rectangle{Width: float32(rl.GetScreenWidth()) / s, Height: float32(rl.GetScreenHeight()) / s})
	rl.SetMouseScale(1, 1)
	rl.EndMode2D()

	ui.Draw(c.DataManager.Consts.DebugFont)
	rl.EndDrawing()
}

func (c *GameController) drawBackground() {
	w, h := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	rl.DrawRectangleGradientV(0, 0, w, h, color.RGBA{38, 22, 70, 255}, color.RGBA{8, 10, 28, 255})
	if c.bkg.ID == 0 {
		return
	}
	// cover the window, keeping the photo's aspect ratio
	scale := max(float32(w)/float32(c.bkg.Width), float32(h)/float32(c.bkg.Height))
	dw, dh := float32(c.bkg.Width)*scale, float32(c.bkg.Height)*scale
	src := rl.Rectangle{Width: float32(c.bkg.Width), Height: float32(c.bkg.Height)}
	dst := rl.Rectangle{X: (float32(w) - dw) / 2, Y: (float32(h) - dh) / 2, Width: dw, Height: dh}
	rl.DrawTexturePro(c.bkg, src, dst, rl.Vector2{}, 0, color.RGBA{255, 255, 255, 105})
}

func (c *GameController) drawLeftPanel() {
	r := c.leftPanel()
	ui.Panel(r)
	col := ui.Column{X: r.X + 18, Y: r.Y + 16, W: r.Width - 36, Gap: 6}
	title := col.Next(44)
	ui.Text(text.T("ui.title", "JEWEL GAME"), title.X, title.Y, 40, ui.Accent)
	c.Quest.DrawLeft(&col, r)
}

// isStoryFile is true for story YAML and art, skipping editor temp files.
func isStoryFile(name string) bool {
	rel, err := filepath.Rel(storyRoot, name)
	if err != nil || strings.HasPrefix(rel, "..") {
		return false
	}
	base := filepath.Base(name)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".yaml", ".yml", ".png", ".jpg", ".jpeg":
		return true
	}
	return false
}
