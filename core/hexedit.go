package core

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"
	"jewel-game/utils"

	"jewel-game/rl"
)

// hexEditor is the paint mode on the world map. The brush is one cell of a sheet.
// A drag moves a place or a bend in a road.
type hexEditor struct {
	on     bool
	sheet  int
	pick   hexRef
	primed bool

	painting bool
	drag     dragKind
	dragFrom string
	dragTo   string
	dragVia  int
}

type dragKind int

const (
	dragNone dragKind = iota
	dragNode
	dragBend
)

const hexSwatch = 30

func (v *QuestView) updateHexEdit() {
	if !v.hex.on || v.Screen != ScreenWorld || v.HasModal() {
		return
	}
	if !rl.IsMouseButtonDown(rl.MouseLeftButton) {
		v.hex.painting = false
		v.hex.drag = dragNone
	}
	panel := centerPanelRect()
	m := panelMouse()
	if m.Y < panel.Y+58 || !rl.CheckCollisionPointRec(m, panel) {
		return
	}
	if rl.IsMouseButtonPressed(rl.MouseRightButton) {
		if from, to, index, ok := bendAt(panel, m); ok {
			currentMap().deleteBend(from, to, index)
		}
	}
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		switch {
		case v.grabNode(panel, m):
		case v.grabBend(panel, m):
		case v.splitRoad(panel, m):
		}
	}
	if v.hex.drag != dragNone && rl.IsMouseButtonDown(rl.MouseLeftButton) {
		v.dragMap(panel, m)
	}
}

func (v *QuestView) saveHexMap() {
	if err := writeHexMap(hexMapFile, currentMap()); err != nil {
		ui.AddMessage("%s", text.Tf("log.map_save", "Could not save the map: %v", err))
		return
	}
	if !mapSaveDeferred {
		ui.AddMessage("%s", text.T("log.map_saved", "Map saved"))
	}
}

func noteMapSave() {
	switch mapSaveNote {
	case "saved":
		ui.AddMessage("%s", text.T("log.map_saved", "Map saved"))
	case "fail":
		ui.AddMessage("%s", text.T("log.map_save_web", "Could not save the map. Use ./start.sh --dev so the browser can write assets/maps/world.yaml."))
	default:
		return
	}
	mapSaveNote = ""
}

func (v *QuestView) drawHexEditor(panel rl.Rectangle) {
	if !v.hex.on {
		btn := rl.Rectangle{X: panel.X + panel.Width - 210, Y: panel.Y + 18, Width: 150, Height: 34}
		if ui.Button(btn, text.T("ui.map_edit", "Edit map  [E]"), false) {
			v.hex.on = true
		}
		return
	}
	y := panel.Y + 18
	image := rl.Rectangle{X: panel.X + panel.Width - 470, Y: y, Width: 120, Height: 34}
	save := rl.Rectangle{X: panel.X + panel.Width - 342, Y: y, Width: 110, Height: 34}
	done := rl.Rectangle{X: panel.X + panel.Width - 224, Y: y, Width: 110, Height: 34}
	m := currentMap()
	if m.background == "" {
		if ui.Button(image, text.T("ui.map_image", "Image"), false) {
			v.chooseBackground()
		}
	} else if ui.Button(image, text.T("ui.map_no_image", "No image"), false) {
		m.background = ""
		m.ensureBackground()
	}
	if ui.Button(save, text.T("ui.map_save", "Save  [S]"), false) {
		v.saveHexMap()
	}
	if ui.Button(done, text.T("ui.map_done", "Done  [E]"), false) {
		v.hex.on = false
	}
	ui.TextOutlined(text.T("ui.map_edit_help", "Drag a place. Click a road to bend it."), panel.X+30, panel.Y+panel.Height-28, 16, ui.TextColor)
}

func (v *QuestView) paintHex(panel rl.Rectangle, m rl.Vector2) {
	view := layoutHex(panel, currentMap().cols, currentMap().rows)
	if c, r, ok := view.cellAt(m); ok {
		currentMap().set(c, r, v.hex.pick)
	}
}

func (v *QuestView) sampleHex(panel rl.Rectangle, m rl.Vector2) {
	view := layoutHex(panel, currentMap().cols, currentMap().rows)
	if c, r, ok := view.cellAt(m); ok {
		if ref, ok := currentMap().at(c, r); ok {
			v.hex.pick = ref
			v.hex.sheet = ref.sheet
		}
	}
}

func (v *QuestView) grabNode(panel rl.Rectangle, m rl.Vector2) bool {
	for i := range rpg.Locations {
		if dist2(placePoint(panel, rpg.Locations[i].ID), m) <= 18*18 {
			v.hex.drag = dragNode
			v.hex.dragFrom = rpg.Locations[i].ID
			return true
		}
	}
	return false
}

func (v *QuestView) grabBend(panel rl.Rectangle, m rl.Vector2) bool {
	from, to, index, ok := bendAt(panel, m)
	if !ok {
		return false
	}
	v.hex.drag = dragBend
	v.hex.dragFrom, v.hex.dragTo, v.hex.dragVia = from, to, index
	return true
}

func (v *QuestView) splitRoad(panel rl.Rectangle, m rl.Vector2) bool {
	best := float32(14 * 14)
	from, to := "", ""
	index := -1
	eachStoryRoad(func(a, b string) {
		pts := roadScreen(panel, a, b)
		for s := 0; s < len(pts)-1; s++ {
			if d := dist2ToSeg(m, pts[s], pts[s+1]); d < best {
				best, from, to, index = d, a, b, s
			}
		}
	})
	if index < 0 {
		return false
	}
	currentMap().insertBend(from, to, pointToNorm(mapField(panel), m), index)
	v.hex.drag = dragBend
	v.hex.dragFrom, v.hex.dragTo, v.hex.dragVia = from, to, index
	return true
}

func (v *QuestView) dragMap(panel rl.Rectangle, m rl.Vector2) {
	at := pointToNorm(mapField(panel), m)
	if v.hex.drag == dragNode {
		currentMap().setNode(v.hex.dragFrom, at)
		return
	}
	currentMap().moveBend(v.hex.dragFrom, v.hex.dragTo, v.hex.dragVia, at)
}

func bendAt(panel rl.Rectangle, m rl.Vector2) (from, to string, index int, ok bool) {
	best := float32(10 * 10)
	eachStoryRoad(func(a, b string) {
		pts := roadScreen(panel, a, b)
		for i := 1; i < len(pts)-1; i++ {
			if d := dist2(pts[i], m); d < best {
				best, from, to, index, ok = d, a, b, i-1, true
			}
		}
	})
	return from, to, index, ok
}

func eachStoryRoad(fn func(a, b string)) {
	for i := range rpg.Locations {
		l := &rpg.Locations[i]
		for _, id := range l.Roads {
			if l.ID < id {
				fn(l.ID, id)
			}
		}
	}
}

func (v *QuestView) chooseBackground() {
	src, ok := pickImageFile()
	if !ok {
		return
	}
	dst, err := copyMapImage(src)
	if err != nil {
		ui.AddMessage("%s", text.Tf("log.map_image", "Could not use that image: %v", err))
		return
	}
	m := currentMap()
	m.background = dst
	m.hideHexes = true
	m.bgLoaded = "\x00"
	m.ensureBackground()
	if m.bg.ID == 0 {
		ui.AddMessage("%s", text.T("log.map_image_bad", "That file is not a PNG or JPEG the map can show."))
	}
}

func pickImageFile() (string, bool) {
	if path, err := exec.LookPath("zenity"); err == nil {
		out, err := exec.Command(path, "--file-selection", "--title=Map background",
			"--file-filter=PNG or JPEG | *.png *.jpg *.jpeg").Output()
		if err != nil {
			return "", false
		}
		return strings.TrimSpace(string(out)), true
	}
	const fallback = "assets/maps/background.png"
	if _, err := os.Stat(fallback); err == nil {
		return fallback, true
	}
	ui.AddMessage("%s", text.T("log.map_image_missing", "No file chooser. Save a PNG as assets/maps/background.png, then press Image again."))
	return "", false
}

func copyMapImage(src string) (string, error) {
	ext := strings.ToLower(filepath.Ext(src))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		ext = ".png"
	}
	dst := "assets/maps/background" + ext
	if filepath.Clean(src) == filepath.Clean(dst) {
		return dst, nil
	}
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	if err := os.MkdirAll("assets/maps", 0o755); err != nil {
		return "", err
	}
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return dst, os.Rename(tmp, dst)
}

// sheetGrid is how many swatches the palette shows. Parchment is the 3 by 3 ink set.
func sheetGrid(sheet int) (cols, rows int) {
	if sheet == hexSheetParchment {
		return 3, 3
	}
	return 7, 7
}

func paletteRect(panel rl.Rectangle, sheet int) rl.Rectangle {
	cols, rows := sheetGrid(sheet)
	w := float32(cols*hexSwatch + 16)
	if w < 200 {
		w = 200
	}
	h := float32(rows*hexSwatch + 80)
	return rl.Rectangle{X: panel.X + panel.Width - w - 16, Y: panel.Y + panel.Height - h - 16, Width: w, Height: h}
}

func swatchBox(pal rl.Rectangle, sheet, c, r int) rl.Rectangle {
	cols, _ := sheetGrid(sheet)
	gridW := float32(cols * hexSwatch)
	x0 := pal.X + (pal.Width-gridW)/2
	return rl.Rectangle{
		X: x0 + float32(c*hexSwatch), Y: pal.Y + 38 + float32(r*hexSwatch),
		Width: hexSwatch - 2, Height: hexSwatch - 2,
	}
}

func swatchAt(pal rl.Rectangle, sheet int, m rl.Vector2) (hexRef, bool) {
	cols, rows := sheetGrid(sheet)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if rl.CheckCollisionPointRec(m, swatchBox(pal, sheet, c, r)) {
				return hexRef{sheet, c, r}, true
			}
		}
	}
	return hexRef{}, false
}

func overPalette(panel rl.Rectangle, m rl.Vector2, sheet int) bool {
	return rl.CheckCollisionPointRec(m, paletteRect(panel, sheet))
}

// panelMouse is the cursor in panel space. Call it from Update, when the mouse scale is 1.
func panelMouse() rl.Vector2 {
	m := rl.GetMousePosition()
	s := uiScale()
	return rl.Vector2{X: m.X / s, Y: m.Y / s}
}

// centerPanelRect matches GameController.centerPanel.
func centerPanelRect() rl.Rectangle {
	s := uiScale()
	w := float32(rl.GetScreenWidth()) / s
	h := float32(rl.GetScreenHeight()) / s
	left := float32(MARGIN)
	side := float32(utils.SIDE_PANEL_WIDTH)
	return rl.Rectangle{X: left + side + 16, Y: left, Width: w - 2*left - 2*side - 32, Height: h - 2*left}
}
