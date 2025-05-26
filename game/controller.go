package game

import (
	"image/color"
	"jewel-game/data"
	"jewel-game/debug"
	"jewel-game/entities"
	"jewel-game/tiles"
	"jewel-game/ui"
	"jewel-game/utils"
	"log"

	"github.com/fsnotify/fsnotify"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type GameController struct {
	AvailalbleGemTypes	[]*entities.GemType
	DataManager			*data.DataManager
	ReloadChannel		chan struct{}
	TileManager			tiles.TileManager
	TileStates			[]tiles.TileState
	bkg					rl.Texture2D
	watcherEvents		chan fsnotify.Event
	watcherErrors		chan error
	watcher				*fsnotify.Watcher
}

func NewGameController() *GameController {
	reload := make(chan struct{}, 100)
	dm := data.NewDataManager(reload)
	tm := tiles.NewTileManager(utils.GRID_LENGTH, utils.GRID_LENGTH, &dm)
	tm.PopulateRandomGems();
	states := InitTileStates()
	bkg := rl.LoadTexture("_res/bkg.jpg")

	gc := &GameController{
		AvailalbleGemTypes:	dm.GetAllGemTypes(),
		DataManager:		&dm,
		ReloadChannel:		reload,
		TileManager:		tm,
		TileStates:			states,
		bkg:				bkg,
		watcherEvents:		make(chan fsnotify.Event, 10),
		watcherErrors:		make(chan error, 1),
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create file watcher: %v", err)
	}
	gc.watcher = watcher

	watchDir := "assets"
	if err := gc.watcher.Add(watchDir); err != nil {
		log.Fatalf("Failed to add directory %s to watcher: %v", watchDir, err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return // Channel closed
				}
				gc.watcherEvents <- event // Forward event to GameController's channel
			case err, ok := <-watcher.Errors:
				if !ok {
					return // Channel closed
				}
				gc.watcherErrors <- err // Forward error to GameController's channel
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

	if c.watcher != nil {
		c.watcher.Close()
		close(c.watcherEvents)
		close(c.watcherErrors)
	}
	debug.AddMessage("GameController resources unloaded.")
}

func (c *GameController) Update() {
	select {
	case event, ok := <-c.watcherEvents:
		if !ok {
			debug.AddMessage("Watcher events channel closed.")
			c.watcher = nil
			return
		}
		if event.Name == "assets/gemtypes.yaml" && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
			debug.AddMessage("Detected change in %s, reloading gem types...", event.Name)
			c.DataManager.LoadGemTypes("assets/gemtypes.yaml")
			c.TileManager.PopulateRandomGems()
		}
	case err, ok := <-c.watcherErrors:
		if !ok {
			debug.AddMessage("Watcher errors channel closed.")
			return
		}
		debug.AddMessage("File watcher error: %v", err)
	default:
	}
	c.TileManager.Update()
}

func (c *GameController) Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.RayWhite)
	rl.DrawTexture(c.bkg, 0, -1200, color.RGBA{ 255, 255, 255, 105 })
	c.TileManager.Draw()
	ui.Draw(c.DataManager)
	rl.EndDrawing()
}

func InitTileStates() []tiles.TileState {
	blank := tiles.NewTileState("_blank", color.RGBA{ 55, 55, 55, 128 })
	grass := tiles.NewTileState("Grass", color.RGBA{ 0, 102, 0, 255 })
	forest := tiles.NewTileState("Forest", color.RGBA{ 0, 51, 0, 255 })
	water := tiles.NewTileState("Water", color.RGBA{ 0, 0, 185, 255 })
	return []tiles.TileState{ blank, grass, forest, water }
}

