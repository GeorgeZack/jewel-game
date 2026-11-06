package core

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"time"

	"jewel-game/rl"
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"

	"gopkg.in/yaml.v3"
)

// The four downloaded sheets are 7x7 pointy-top hexes in 144px cells.
// Every tile shares a footprint: the bottom tip is at (71, 135) in the cell,
// centers sit 94px apart and 73px down, and odd rows shift half a step.
const (
	hexSheetOver = iota
	hexSheetTown
	hexSheetLava
	hexSheetOcean
	hexSheetParchment

	hexPitchX = 94
	hexPitchY = 73
	hexFootX  = 71
	hexFootY  = 135
	hexCell   = 144
	hexPadX   = 8

	hexMapCols = 16
	hexMapRows = 14
	hexMapFile = "assets/maps/world.yaml"
)

var hexSheetPaths = []string{
	"_res/hex/5783dcb917a9fa769d33d1d58aace82b.png",
	"_res/hex/adb91e209d915bdd49724fdae3b427f1.png",
	"_res/hex/56fdd71a7c1d3e1de461c4a4b72b5114.png",
	"_res/hex/856c5bb98246f0456b9c81fd9932a52f.png",
	"_res/hex/parchment.png",
}

var hexSheetLabel = []struct{ key, fallback string }{
	{"ui.hex_overworld", "Overworld"},
	{"ui.hex_towns", "Towns"},
	{"ui.hex_lava", "Lava"},
	{"ui.hex_sea", "Sea"},
	{"ui.hex_parchment", "Parchment"},
}

var hexSheets []rl.Texture2D

type hexRef struct {
	sheet int
	c, r  int
}

type hexBiome struct {
	ground []hexRef
	mark   hexRef
}

func hx(sheet, c, r int) hexRef { return hexRef{sheet, c, r} }

// Ground tiles vary inside a region. The mark sits on the location's own hex.
var hexBiomes = map[string]hexBiome{
	"emberfield":  {ground: []hexRef{hx(hexSheetTown, 0, 0), hx(hexSheetOver, 0, 3)}, mark: hx(hexSheetTown, 2, 0)},
	"mossgate":    {ground: []hexRef{hx(hexSheetOver, 5, 2), hx(hexSheetOver, 2, 3), hx(hexSheetOver, 0, 3)}, mark: hx(hexSheetOver, 5, 2)},
	"stonebridge": {ground: []hexRef{hx(hexSheetTown, 0, 0), hx(hexSheetOver, 4, 2), hx(hexSheetOver, 3, 6)}, mark: hx(hexSheetOver, 3, 5)},
	"greyfen":     {ground: []hexRef{hx(hexSheetOver, 4, 3), hx(hexSheetOver, 4, 2), hx(hexSheetOver, 3, 6)}, mark: hx(hexSheetOver, 4, 3)},
	"barrow":      {ground: []hexRef{hx(hexSheetOver, 6, 4), hx(hexSheetOver, 4, 3), hx(hexSheetOver, 2, 0)}, mark: hx(hexSheetOver, 6, 4)},
	"highwatch":   {ground: []hexRef{hx(hexSheetTown, 0, 0), hx(hexSheetOver, 6, 4)}, mark: hx(hexSheetTown, 5, 2)},
	"redtusk":     {ground: []hexRef{hx(hexSheetOver, 2, 0), hx(hexSheetOver, 3, 1), hx(hexSheetOver, 0, 4)}, mark: hx(hexSheetOver, 4, 4)},
	"ironpeak":    {ground: []hexRef{hx(hexSheetOver, 6, 4), hx(hexSheetOver, 5, 3), hx(hexSheetLava, 0, 0)}, mark: hx(hexSheetOver, 6, 4)},
	"silverlake":  {ground: []hexRef{hx(hexSheetOver, 4, 6), hx(hexSheetOver, 3, 6), hx(hexSheetTown, 0, 0)}, mark: hx(hexSheetTown, 4, 0)},
	"sunken":      {ground: []hexRef{hx(hexSheetOcean, 5, 0), hx(hexSheetOcean, 2, 2), hx(hexSheetOcean, 1, 1)}, mark: hx(hexSheetOcean, 0, 1)},
	"ashen":       {ground: []hexRef{hx(hexSheetLava, 0, 0), hx(hexSheetLava, 2, 4), hx(hexSheetLava, 5, 2)}, mark: hx(hexSheetLava, 4, 0)},
	"hollow":      {ground: []hexRef{hx(hexSheetLava, 0, 0), hx(hexSheetOver, 4, 3)}, mark: hx(hexSheetTown, 4, 2)},
}

// norm is a point on the map, 0..1 across the map field.
type norm struct{ x, y float32 }

// mapRoad bends the straight link between two locations. Via runs from the
// alphabetically first id to the second.
type mapRoad struct {
	a, b string
	via  []norm
}

// worldMap is the painted hex field, plus where places and roads sit.
type worldMap struct {
	cols, rows int
	tiles      []hexRef
	nodes      map[string]norm
	roads      []mapRoad
	background string
	hideHexes  bool
	bg         rl.Texture2D
	bgLoaded   string
	bgSrc      rl.Rectangle // the part of the file that is the picture, skipping empty margins
	bgAspect   float32
}

func (m *worldMap) index(c, r int) (int, bool) {
	if m == nil || c < 0 || r < 0 || c >= m.cols || r >= m.rows {
		return 0, false
	}
	return r*m.cols + c, true
}

func (m *worldMap) at(c, r int) (hexRef, bool) {
	i, ok := m.index(c, r)
	if !ok {
		return hexRef{}, false
	}
	return m.tiles[i], true
}

func (m *worldMap) set(c, r int, ref hexRef) {
	if i, ok := m.index(c, r); ok {
		m.tiles[i] = ref
	}
}

type hexFile struct {
	Cols       int                   `yaml:"cols"`
	Rows       int                   `yaml:"rows"`
	Tiles      [][3]int              `yaml:"tiles"`
	Nodes      map[string][2]float32 `yaml:"nodes,omitempty"`
	Roads      []roadFile            `yaml:"roads,omitempty"`
	Background string                `yaml:"background,omitempty"`
	HideHexes  bool                  `yaml:"hide_hexes,omitempty"`
}

type roadFile struct {
	From string       `yaml:"from"`
	To   string       `yaml:"to"`
	Via  [][2]float32 `yaml:"via,omitempty"`
}

var (
	theMap    *worldMap
	mapLoaded bool
	mapMod    time.Time
	mapErr    string
	// mapSaveDeferred is set in the browser, where the save finishes after the server answers.
	mapSaveDeferred bool
	mapSaveNote     string
)

func currentMap() *worldMap {
	info, statErr := os.Stat(hexMapFile)
	if mapLoaded && (statErr != nil || info.ModTime().Equal(mapMod)) {
		return theMap
	}
	mapLoaded = true
	if statErr == nil {
		mapMod = info.ModTime()
	}
	m, err := readHexMap(hexMapFile)
	if err != nil {
		if err.Error() != mapErr {
			mapErr = err.Error()
			ui.AddMessage("%s", text.Tf("log.map_read", "Could not read assets/maps/world.yaml: %v", err))
		}
		m = seedMap()
	} else {
		mapErr = ""
	}
	useMapPicture(m)
	if theMap != nil && theMap != m && theMap.bg.ID != 0 {
		rl.UnloadTexture(theMap.bg)
	}
	theMap = m
	return theMap
}

// useMapPicture uses assets/maps/background.png when the yaml names no picture.
func useMapPicture(m *worldMap) {
	if m == nil || m.background != "" {
		return
	}
	for _, path := range []string{"assets/maps/background.png", "assets/maps/background.jpg", "assets/maps/background.jpeg"} {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			m.background = path
			return
		}
	}
}

func readHexMap(path string) (*worldMap, error) {
	if b, ok := liveMapBytes(); ok {
		return parseHexMap(b)
	}
	b, err := rl.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &worldMap{cols: hexMapCols, rows: hexMapRows, nodes: map[string]norm{}}, nil
		}
		return nil, err
	}
	return parseHexMap(b)
}

func parseHexMap(b []byte) (*worldMap, error) {
	var f hexFile
	if len(bytes.TrimSpace(b)) > 0 {
		if err := yaml.Unmarshal(b, &f); err != nil {
			return nil, err
		}
	}
	// An empty tile list is a layout-only map: picture, places, and bends. Hexes are not drawn.
	full := f.Cols >= 2 && f.Rows >= 2 && len(f.Tiles) == f.Cols*f.Rows
	if !full && len(f.Tiles) > 0 {
		return nil, fmt.Errorf("map is %d x %d with %d tiles", f.Cols, f.Rows, len(f.Tiles))
	}
	m := &worldMap{cols: hexMapCols, rows: hexMapRows, nodes: map[string]norm{}, background: f.Background, hideHexes: f.HideHexes}
	if full {
		m.cols, m.rows = f.Cols, f.Rows
		m.tiles = make([]hexRef, len(f.Tiles))
		for i, t := range f.Tiles {
			if t[0] < 0 || t[0] >= len(hexSheetPaths) || t[1] < 0 || t[1] > 6 || t[2] < 0 || t[2] > 6 {
				return nil, fmt.Errorf("tile %d out of range", i)
			}
			m.tiles[i] = hexRef{t[0], t[1], t[2]}
		}
	}
	m.nodes = map[string]norm{}
	for id, p := range f.Nodes {
		m.nodes[id] = norm{p[0], p[1]}
	}
	for _, road := range f.Roads {
		if road.From == "" || road.To == "" || road.From == road.To {
			continue
		}
		via := make([]norm, len(road.Via))
		for i, p := range road.Via {
			via[i] = norm{p[0], p[1]}
		}
		a, b := road.From, road.To
		if a > b {
			a, b = b, a
			for i, j := 0, len(via)-1; i < j; i, j = i+1, j-1 {
				via[i], via[j] = via[j], via[i]
			}
		}
		m.roads = append(m.roads, mapRoad{a: a, b: b, via: via})
	}
	return m, nil
}

func encodeHexMap(m *worldMap) ([]byte, error) {
	f := hexFile{
		Cols: m.cols, Rows: m.rows, Tiles: make([][3]int, len(m.tiles)),
		Background: m.background, HideHexes: m.hideHexes,
		Nodes: map[string][2]float32{},
	}
	for i, t := range m.tiles {
		f.Tiles[i] = [3]int{t.sheet, t.c, t.r}
	}
	for id, p := range m.nodes {
		f.Nodes[id] = [2]float32{p.x, p.y}
	}
	for _, road := range m.roads {
		if len(road.via) == 0 {
			continue
		}
		rf := roadFile{From: road.a, To: road.b, Via: make([][2]float32, len(road.via))}
		for i, p := range road.via {
			rf.Via[i] = [2]float32{p.x, p.y}
		}
		f.Roads = append(f.Roads, rf)
	}
	body, err := yaml.Marshal(f)
	if err != nil {
		return nil, err
	}
	head := "# World map. nodes are [x, y] from 0 to 1 across the picture.\n# roads.via bends the path from `from` to `to`.\n"
	return append([]byte(head), body...), nil
}

func seedMap() *worldMap {
	m := &worldMap{cols: hexMapCols, rows: hexMapRows, tiles: make([]hexRef, hexMapCols*hexMapRows), nodes: map[string]norm{}}
	for i := range rpg.Locations {
		l := &rpg.Locations[i]
		m.nodes[l.ID] = norm{l.X, l.Y}
	}
	marks := map[[2]int]hexRef{}
	for i := range rpg.Locations {
		l := &rpg.Locations[i]
		bio, ok := hexBiomes[l.ID]
		if !ok {
			continue
		}
		c, r := locationCell(m, l)
		marks[[2]int{c, r}] = bio.mark
	}
	for r := 0; r < m.rows; r++ {
		for c := 0; c < m.cols; c++ {
			if mark, ok := marks[[2]int{c, r}]; ok {
				m.set(c, r, mark)
				continue
			}
			nx, ny := cellNorm(m, c, r)
			id := nearestLocation(nx, ny)
			bio, ok := hexBiomes[id]
			if !ok || len(bio.ground) == 0 {
				m.set(c, r, hx(hexSheetTown, 0, 0))
				continue
			}
			m.set(c, r, bio.ground[hexHash(c, r)%len(bio.ground)])
		}
	}
	return m
}

func loadHexSheets() {
	if hexSheets != nil {
		return
	}
	hexSheets = make([]rl.Texture2D, len(hexSheetPaths))
	for i, path := range hexSheetPaths {
		hexSheets[i] = rl.LoadTexture(path)
		if hexSheets[i].ID != 0 {
			rl.SetTextureFilter(hexSheets[i], rl.FilterPoint)
		}
	}
}

func hexHash(c, r int) int {
	n := uint32(c*374761393 ^ r*668265263)
	n = (n ^ (n >> 13)) * 1274126177
	return int(n & 0x7fffffff)
}

// cellNorm is the cell's place in 0..1, matching location X/Y.
// Odd rows are shifted half a step, and the last odd cell lands on 1.
func cellNorm(m *worldMap, c, r int) (float32, float32) {
	spanX := float32(m.cols) - 0.5
	spanY := float32(m.rows - 1)
	if spanY < 1 {
		spanY = 1
	}
	off := float32(0)
	if r%2 != 0 {
		off = 0.5
	}
	return (float32(c) + off) / spanX, float32(r) / spanY
}

func nearestLocation(nx, ny float32) string {
	best := float32(1e9)
	id := "emberfield"
	for i := range rpg.Locations {
		l := &rpg.Locations[i]
		dx := (nx - l.X) * hexPitchX
		dy := (ny - l.Y) * hexPitchY
		if d := dx*dx + dy*dy; d < best {
			best, id = d, l.ID
		}
	}
	return id
}

// locationCell is the hex a location stands on. It does not depend on the window size.
func locationCell(m *worldMap, l *rpg.Location) (int, int) {
	best := float32(1e9)
	col, row := 0, 0
	for r := 0; r < m.rows; r++ {
		for c := 0; c < m.cols; c++ {
			nx, ny := cellNorm(m, c, r)
			dx := (nx - l.X) * hexPitchX
			dy := (ny - l.Y) * hexPitchY
			if d := dx*dx + dy*dy; d < best {
				best = d
				col, row = c, r
			}
		}
	}
	return col, row
}

// hexView places the grid inside the same rectangle map points use.
type hexView struct {
	origin       rl.Vector2
	scale        float32
	stepX, stepY float32
	cols, rows   int
}

func layoutHex(panel rl.Rectangle, cols, rows int) hexView {
	contentW := panel.Width - 100
	contentH := panel.Height - 120
	if contentW < 40 {
		contentW = 40
	}
	if contentH < 40 {
		contentH = 40
	}
	spanX := float32(cols) - 0.5
	spanY := float32(rows - 1)
	if spanY < 1 {
		spanY = 1
	}
	// Leave room for the hex body past the first and last center.
	sx := contentW / ((spanX + 1) * hexPitchX)
	sy := contentH / (float32(hexFootY-36) + spanY*hexPitchY)
	scale := min(sx, sy)
	if scale > 1.15 {
		scale = 1.15
	}
	stepX, stepY := hexPitchX*scale, hexPitchY*scale
	gridW := spanX * stepX
	gridH := spanY * stepY
	top := float32(hexFootY-36) * scale
	return hexView{
		origin: rl.Vector2{
			X: panel.X + 50 + (contentW-(gridW+stepX))/2 + stepX/2,
			Y: panel.Y + 60 + (contentH-(gridH+top))/2 + top,
		},
		scale: scale, stepX: stepX, stepY: stepY, cols: cols, rows: rows,
	}
}

func (v hexView) anchor(c, r int) rl.Vector2 {
	off := float32(0)
	if r%2 != 0 {
		off = v.stepX / 2
	}
	return rl.Vector2{X: v.origin.X + float32(c)*v.stepX + off, Y: v.origin.Y + float32(r)*v.stepY}
}

func (v hexView) cellAt(m rl.Vector2) (int, int, bool) {
	best := float32(1e9)
	col, row := 0, 0
	for r := 0; r < v.rows; r++ {
		for c := 0; c < v.cols; c++ {
			at := v.anchor(c, r)
			dx, dy := at.X-m.X, at.Y-m.Y
			if d := dx*dx + dy*dy; d < best {
				best = d
				col, row = c, r
			}
		}
	}
	if best > v.stepX*v.stepX*0.55 {
		return 0, 0, false
	}
	return col, row, true
}

// mapArea is the space inside the panel where the map is allowed to sit.
func mapArea(panel rl.Rectangle) rl.Rectangle {
	return rl.Rectangle{X: panel.X + 50, Y: panel.Y + 60, Width: panel.Width - 100, Height: panel.Height - 120}
}

// mapField is the picture's rectangle. Places use 0..1 across this, so they stay
// on the art when the window changes size. The picture keeps its own shape.
func mapField(panel rl.Rectangle) rl.Rectangle {
	area := mapArea(panel)
	m := currentMap()
	if m == nil || m.bgAspect <= 0 {
		return area
	}
	return fitRect(area, m.bgAspect)
}

// fitRect fits a picture of the given width/height into area without stretching it.
func fitRect(area rl.Rectangle, aspect float32) rl.Rectangle {
	if aspect <= 0 || area.Width < 1 || area.Height < 1 {
		return area
	}
	w, h := area.Width, area.Width/aspect
	if h > area.Height {
		h = area.Height
		w = h * aspect
	}
	return rl.Rectangle{
		X: area.X + (area.Width-w)/2, Y: area.Y + (area.Height-h)/2,
		Width: w, Height: h,
	}
}

func nodeNorm(id string) norm {
	if n, ok := currentMap().nodes[id]; ok {
		return n
	}
	if l := rpg.LocationByID(id); l != nil {
		return norm{l.X, l.Y}
	}
	return norm{}
}

func (m *worldMap) setNode(id string, n norm) {
	if m.nodes == nil {
		m.nodes = map[string]norm{}
	}
	n.x = max(0, min(1, n.x))
	n.y = max(0, min(1, n.y))
	m.nodes[id] = n
}

func normToPoint(field rl.Rectangle, n norm) rl.Vector2 {
	return rl.Vector2{X: field.X + n.x*field.Width, Y: field.Y + n.y*field.Height}
}

func pointToNorm(field rl.Rectangle, p rl.Vector2) norm {
	return norm{x: (p.X - field.X) / field.Width, y: (p.Y - field.Y) / field.Height}
}

func placePoint(panel rl.Rectangle, id string) rl.Vector2 {
	return normToPoint(mapField(panel), nodeNorm(id))
}

// roadBetween returns the bend points ordered from `from` to `to`.
func (m *worldMap) roadBetween(from, to string) []norm {
	a, b := from, to
	flip := false
	if a > b {
		a, b = b, a
		flip = true
	}
	var via []norm
	for i := range m.roads {
		if m.roads[i].a == a && m.roads[i].b == b {
			via = m.roads[i].via
			break
		}
	}
	if flip {
		out := make([]norm, len(via))
		for i := range via {
			out[i] = via[len(via)-1-i]
		}
		return out
	}
	return via
}

func (m *worldMap) bend(from, to string) *mapRoad {
	a, b := from, to
	if a > b {
		a, b = b, a
	}
	for i := range m.roads {
		if m.roads[i].a == a && m.roads[i].b == b {
			return &m.roads[i]
		}
	}
	m.roads = append(m.roads, mapRoad{a: a, b: b})
	return &m.roads[len(m.roads)-1]
}

// insertBend adds a point on the road from `from` to `to`. index is its place in that direction.
func (m *worldMap) insertBend(from, to string, at norm, index int) {
	road := m.bend(from, to)
	if from > to {
		index = len(road.via) - index
	}
	if index < 0 {
		index = 0
	}
	if index > len(road.via) {
		index = len(road.via)
	}
	road.via = append(road.via, norm{})
	copy(road.via[index+1:], road.via[index:])
	road.via[index] = at
}

func (m *worldMap) moveBend(from, to string, index int, at norm) {
	road := m.bend(from, to)
	if from > to {
		index = len(road.via) - 1 - index
	}
	if index < 0 || index >= len(road.via) {
		return
	}
	road.via[index] = at
}

func (m *worldMap) deleteBend(from, to string, index int) {
	road := m.bend(from, to)
	if from > to {
		index = len(road.via) - 1 - index
	}
	if index < 0 || index >= len(road.via) {
		return
	}
	road.via = append(road.via[:index], road.via[index+1:]...)
}

func roadScreen(panel rl.Rectangle, from, to string) []rl.Vector2 {
	field := mapField(panel)
	via := currentMap().roadBetween(from, to)
	pts := make([]rl.Vector2, 0, len(via)+2)
	pts = append(pts, normToPoint(field, nodeNorm(from)))
	for _, n := range via {
		pts = append(pts, normToPoint(field, n))
	}
	pts = append(pts, normToPoint(field, nodeNorm(to)))
	return pts
}

func pointAlong(pts []rl.Vector2, t float32) rl.Vector2 {
	if len(pts) == 0 {
		return rl.Vector2{}
	}
	if len(pts) == 1 || t <= 0 {
		return pts[0]
	}
	if t >= 1 {
		return pts[len(pts)-1]
	}
	total := float32(0)
	seg := make([]float32, len(pts)-1)
	for i := 1; i < len(pts); i++ {
		dx, dy := pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y
		seg[i-1] = float32(math.Hypot(float64(dx), float64(dy)))
		total += seg[i-1]
	}
	if total < 1 {
		return pts[0]
	}
	want := total * t
	for i, d := range seg {
		if want <= d || i == len(seg)-1 {
			f := float32(0)
			if d > 0 {
				f = want / d
			}
			return rl.Vector2{X: pts[i].X + (pts[i+1].X-pts[i].X)*f, Y: pts[i].Y + (pts[i+1].Y-pts[i].Y)*f}
		}
		want -= d
	}
	return pts[len(pts)-1]
}

func (m *worldMap) ensureBackground() {
	if m.background == m.bgLoaded {
		return
	}
	if m.bg.ID != 0 {
		rl.UnloadTexture(m.bg)
		m.bg = rl.Texture2D{}
	}
	m.bgLoaded = m.background
	m.bgSrc = rl.Rectangle{}
	m.bgAspect = 0
	if m.background == "" {
		return
	}
	raw, err := rl.ReadFile(m.background)
	if err != nil {
		return
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return
	}
	m.bgSrc = pictureBounds(img)
	if m.bgSrc.Height > 0 {
		m.bgAspect = m.bgSrc.Width / m.bgSrc.Height
	}
	m.bg = rl.NewTexture(img)
	if m.bg.ID != 0 {
		rl.SetTextureFilter(m.bg, rl.FilterBilinear)
	}
}

// pictureBounds drops empty margins, such as the black band beside a hex map.
func pictureBounds(img image.Image) rl.Rectangle {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a>>8 < 16 || (r>>8 < 12 && g>>8 < 12 && bl>>8 < 12) {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		return rl.Rectangle{Width: float32(b.Dx()), Height: float32(b.Dy())}
	}
	return rl.Rectangle{X: float32(minX - b.Min.X), Y: float32(minY - b.Min.Y), Width: float32(maxX - minX + 1), Height: float32(maxY - minY + 1)}
}

func dist2(a, b rl.Vector2) float32 {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}

func dist2ToSeg(p, a, b rl.Vector2) float32 {
	abx, aby := b.X-a.X, b.Y-a.Y
	len2 := abx*abx + aby*aby
	if len2 < 1 {
		return dist2(p, a)
	}
	t := ((p.X-a.X)*abx + (p.Y-a.Y)*aby) / len2
	t = max(0, min(1, t))
	return dist2(p, rl.Vector2{X: a.X + abx*t, Y: a.Y + aby*t})
}

func drawMapBackground(panel rl.Rectangle) {
	m := currentMap()
	m.ensureBackground()
	if m.bg.ID == 0 {
		return
	}
	field := mapField(panel)
	src := m.bgSrc
	if src.Width == 0 || src.Height == 0 {
		src = rl.Rectangle{Width: float32(m.bg.Width), Height: float32(m.bg.Height)}
	}
	rl.DrawTexturePro(m.bg, src, field, rl.Vector2{}, 0, rl.White)
}

func (v *QuestView) drawHexField(panel rl.Rectangle) {
	m := currentMap()
	if m == nil || m.hideHexes {
		return
	}
	loadHexSheets()
	if len(hexSheets) == 0 || hexSheets[0].ID == 0 {
		return
	}
	view := layoutHex(panel, m.cols, m.rows)
	field := rl.Rectangle{X: panel.X + 18, Y: panel.Y + 58, Width: panel.Width - 36, Height: panel.Height - 74}
	s := uiScale()
	rl.BeginScissorMode(int32(field.X*s), int32(field.Y*s), int32(field.Width*s), int32(field.Height*s))
	for r := 0; r < m.rows; r++ {
		for c := 0; c < m.cols; c++ {
			ref, ok := m.at(c, r)
			if !ok {
				continue
			}
			drawHexTile(ref, view.anchor(c, r), view.scale, rl.White)
		}
	}
	if v.hex.on {
		// Draw runs with the mouse already scaled into panel space.
		m := rl.GetMousePosition()
		if c, r, ok := view.cellAt(m); ok && !overPalette(panel, m, v.hex.sheet) {
			at := view.anchor(c, r)
			rl.DrawRing(at, 16*view.scale, 20*view.scale, 0, 360, 24, rl.Fade(rl.White, 0.85))
		}
	}
	rl.EndScissorMode()
}

func drawHexTile(ref hexRef, at rl.Vector2, scale float32, col rl.Color) {
	if ref.sheet < 0 || ref.sheet >= len(hexSheets) || hexSheets[ref.sheet].ID == 0 {
		return
	}
	rl.DrawTexturePro(hexSheets[ref.sheet],
		rl.Rectangle{X: float32(hexPadX + ref.c*hexCell), Y: float32(ref.r * hexCell), Width: hexCell, Height: hexCell},
		rl.Rectangle{
			X: at.X - hexFootX*scale, Y: at.Y - hexFootY*scale,
			Width: hexCell * scale, Height: hexCell * scale,
		},
		rl.Vector2{}, 0, col)
}
