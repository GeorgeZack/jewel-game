package core

import (
	"jewel-game/ui"
	"jewel-game/entities"
	"jewel-game/utils"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	rl "github.com/gen2brain/raylib-go/raylib"
	"gopkg.in/yaml.v3"
)

type Consts struct {
	DebugFont	rl.Font
}

type DataManager struct {
	Animating				bool
	// AnimationQueue			[]*types.Cell
	AnimationTimeout		int
	// CellCur					*types.Cell
	// CellEnd					*types.Cell
	// CellHover				*types.Cell
	// CellPrev				*types.Cell
	// CellStart				*types.Cell
	Consts					*Consts
	gemTypes				map[string]*entities.GemType
	IsMouseButtonPressed	bool
	MoveStarted				bool
	ReloadChannel			chan struct{}
	watcher					*fsnotify.Watcher
}

func NewDataManager(reload chan struct{}) DataManager {
	consts := &Consts{
		DebugFont: rl.LoadFontEx("_res/NotoMono-Regular.ttf", 16, nil),
	}
	dm := DataManager{
		Animating:				false,
		// AnimationQueue:			make([]*types.Cell, 0),
		AnimationTimeout:		0,
		// CellCur:				nil,
		// CellEnd:				nil,
		// CellHover:				nil,
		// CellPrev:				nil,
		// CellStart:				nil,
		Consts:					consts,
		gemTypes: 				make(map[string]*entities.GemType),
		IsMouseButtonPressed:	false,
		MoveStarted: 			false,
		ReloadChannel: 			reload,
	}
	dm.LoadGemTypes("assets/gemtypes.yaml")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	dm.watcher = watcher

	watchDir := "assets"
	if err := dm.watcher.Add(watchDir); err != nil {
		log.Fatal(err)
	}

	go dm.watchForChanges()

	return dm
}

func (dm *DataManager) GetAllGemTypes() []*entities.GemType {
	types := make([]*entities.GemType, 0, len(dm.gemTypes))
	for _, gt := range dm.gemTypes {
		types = append(types, gt)
	}
	return types
}

func (dm *DataManager) LoadGemTypes(fileName string) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		ui.AddMessage("Error reading gem types file %s: %v", fileName, err)
		return
	}

	var tempGemTypes []struct {
		Name		string	`yaml:"name"`
		ColorHex	string	`yaml:"color"`
		TexturePath	string	`yaml:"texturePath"`
	}
	if err := yaml.Unmarshal(data, &tempGemTypes); err != nil {
		ui.AddMessage("Error unmarshaling gem types: %v", err)
		return
	}

	newGemTypes := make(map[string]*entities.GemType)
	for name, oldGem := range dm.gemTypes {
		ui.AddMessage("Old Gem: %+v", oldGem.Color)
		found := false
		for _, newDef := range tempGemTypes {
			if newDef.Name == name && filepath.Clean(newDef.TexturePath) == filepath.Clean(oldGem.TexturePath) && oldGem.Texture.ID != 0 {
				ui.AddMessage("Found difference in %s type", name)
				found = true
				break
			}
		}
		if !found && oldGem.Texture.ID != 0 {
			rl.UnloadTexture(oldGem.Texture)
			ui.AddMessage("Unloaded old texture for gem: %s", name)
		}
	}

	for _, temp := range tempGemTypes {
		r, g, b, a := utils.ParseHexColor(temp.ColorHex)
		color := rl.Color{ R: r, G: g, B: b, A: a }
		var texture rl.Texture2D
		var texturePath string
		if oldGem, ok := dm.gemTypes[temp.Name]; ok && filepath.Clean(temp.TexturePath) == filepath.Clean(oldGem.TexturePath) && oldGem.Texture.ID != 0 {
			texture = oldGem.Texture
			texturePath = oldGem.TexturePath
		} else {
			texture = rl.LoadTexture(temp.TexturePath)
			texturePath = temp.TexturePath
			if texture.ID == 0 {
				ui.AddMessage("Warning: Failed to load texture for gem %s at %s", temp.Name, temp.TexturePath)
			} else {
				ui.AddMessage("Loaded new texture for gem: %s from %s", temp.Name, temp.TexturePath)
			}
		}

		newGemTypes[temp.Name] = &entities.GemType{
			Color: color,
			Name: temp.Name,
			Rotation: 0,
			Scale: 1,
			Texture: texture,
			TexturePath: texturePath,
		}
		ui.AddMessage("New Gem: %+v", newGemTypes[temp.Name].Color)
	}

	dm.gemTypes = newGemTypes
	ui.AddMessage("%+v", dm.gemTypes)
	ui.AddMessage("Gem types reloaded successfully.")
}

func (dm *DataManager) watchForChanges() {
	for {
		select {
		case event, ok := <-dm.watcher.Events:
			if !ok {
				return
			}
			// Filter for relevant events (Write, Create, Remove, Rename) on the specific file
			// Note: Some editors write to a temp file and then rename/move, so watching the directory is often safer.
			if event.Name == "assets/gemtypes.yaml" && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				ui.AddMessage("Detected change in %s, reloading...", event.Name)
				dm.LoadGemTypes(event.Name)
				// Here, you would also trigger an update in the GameController/TileManager
				// e.g., send a channel message, or call a callback.
				dm.ReloadChannel <- struct{}{}
			}
		case err, ok := <-dm.watcher.Errors:
			if !ok {
				return
			}
			ui.AddMessage("Watcher error: %v", err)
		}
	}
}

