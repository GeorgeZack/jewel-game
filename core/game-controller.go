package core

import (
	"image/color"
	"jewel-game/entities"
	"jewel-game/ui"
	"jewel-game/utils"
	"log"

	"github.com/fsnotify/fsnotify"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type GameController struct {
	AvailalbleGemTypes	[]*entities.GemType
	DataManager			*DataManager
	ReloadChannel		chan struct{}
	TileManager			TileManager
	TileStates			[]TileState
	Watcher				*fsnotify.Watcher
	WatcherErrors		chan error
	WatcherEvents		chan fsnotify.Event
	bkg					rl.Texture2D
}

func NewGameController() *GameController {
	reload := make(chan struct{}, 100)
	dm := NewDataManager(reload)
	tm := NewTileManager(utils.GRID_LENGTH, utils.GRID_LENGTH, 500, 500, &dm)
	tm.PopulateRandomGems();
	states := InitTileStates()
	bkg := rl.LoadTexture("_res/bkg.jpg")

	gc := &GameController{
		AvailalbleGemTypes:	dm.GetAllGemTypes(),
		DataManager:		&dm,
		ReloadChannel:		reload,
		TileManager:		tm,
		TileStates:			states,
		WatcherErrors:		make(chan error, 1),
		WatcherEvents:		make(chan fsnotify.Event, 10),
		bkg:				bkg,
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create file watcher: %v", err)
	}
	gc.Watcher = watcher

	watchDir := "assets"
	if err := gc.Watcher.Add(watchDir); err != nil {
		log.Fatalf("Failed to add directory %s to watcher: %v", watchDir, err)
	}

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

func (c *GameController) Unload() {
	rl.UnloadTexture(c.bkg)
	rl.UnloadFont(c.DataManager.Consts.DebugFont)
	for _, g := range c.AvailalbleGemTypes {
		defer rl.UnloadTexture(g.Texture)
	}

	if c.Watcher != nil {
		c.Watcher.Close()
		close(c.WatcherEvents)
		close(c.WatcherErrors)
	}
	ui.AddMessage("GameController resources unloaded.")
}

func (c *GameController) Update() {
	select {
	case event, ok := <-c.WatcherEvents:
		if !ok {
			ui.AddMessage("Watcher events channel closed.")
			c.Watcher = nil
			return
		}
		if event.Name == "assets/gemtypes.yaml" && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
			ui.AddMessage("Detected change in %s, reloading gem types...", event.Name)
			c.DataManager.LoadGemTypes("assets/gemtypes.yaml")
			c.TileManager.PopulateRandomGems()
		}
	case err, ok := <-c.WatcherErrors:
		if !ok {
			ui.AddMessage("Watcher errors channel closed.")
			return
		}
		ui.AddMessage("File watcher error: %v", err)
	default:
	}
	c.TileManager.Update()
}

func (c *GameController) Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.RayWhite)
	rl.DrawTexture(c.bkg, 0, -1200, color.RGBA{ 255, 255, 255, 105 })
	c.TileManager.Draw()
	ui.Draw(c.DataManager.Consts.DebugFont)
	rl.EndDrawing()
}

func InitTileStates() []TileState {
	blank := NewTileState("_blank", color.RGBA{ 55, 55, 55, 128 })
	grass := NewTileState("Grass", color.RGBA{ 0, 102, 0, 255 })
	forest := NewTileState("Forest", color.RGBA{ 0, 51, 0, 255 })
	water := NewTileState("Water", color.RGBA{ 0, 0, 185, 255 })
	return []TileState{ blank, grass, forest, water }
}

