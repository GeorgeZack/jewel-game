package rl

import (
	"errors"
	"image"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Key = ebiten.Key
type MouseButton = ebiten.MouseButton

const (
	KeyF1           = ebiten.KeyF1
	KeyTab          = ebiten.KeyTab
	KeyH            = ebiten.KeyH
	KeyR            = ebiten.KeyR
	KeyZ            = ebiten.KeyZ
	KeySpace        = ebiten.KeySpace
	KeyEnter        = ebiten.KeyEnter
	KeyBackspace    = ebiten.KeyBackspace
	KeyEscape       = ebiten.KeyEscape
	KeyOne          = ebiten.KeyDigit1
	KeyTwo          = ebiten.KeyDigit2
	KeyThree        = ebiten.KeyDigit3
	KeyFour         = ebiten.KeyDigit4
	KeyFive         = ebiten.KeyDigit5
	KeySix          = ebiten.KeyDigit6
	KeyE            = ebiten.KeyE
	KeyS            = ebiten.KeyS
	KeyLeftBracket  = ebiten.KeyLeftBracket
	KeyRightBracket = ebiten.KeyRightBracket

	MouseLeftButton  = ebiten.MouseButtonLeft
	MouseRightButton = ebiten.MouseButtonRight
)

// Frame state. The game updates and draws once per displayed frame, inside Ebitengine's Draw,
// so animation runs at the display rate. Input edges are gathered every tick and handed to the
// next frame, so a press is seen exactly once whether ticks outnumber frames or the reverse.
var (
	screen     *ebiten.Image // the current draw target, possibly a scissored sub-image
	fullScreen *ebiten.Image
	screenW    = 1280
	screenH    = 720
	dpi        = 1.0 // physical pixels per logical pixel
	zoom       = float32(1)
	camOffset  Vector2
	mouseScale = Vector2{X: 1, Y: 1}
	frameTime  float32
	lastFrame  time.Time

	pendingKeys    = map[Key]bool{}
	pendingPress   = map[MouseButton]bool{}
	pendingRelease = map[MouseButton]bool{}
	frameKeys      = map[Key]bool{}
	framePress     = map[MouseButton]bool{}
	frameRelease   = map[MouseButton]bool{}
)

// ErrQuit stops Run cleanly when the update function asks to.
var errQuit = errors.New("quit")

type game struct {
	update func() bool // returns false to quit
	draw   func()
	quit   bool
}

func (g *game) Update() error {
	if g.quit {
		return ebiten.Termination
	}
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		pendingKeys[k] = true
	}
	for _, b := range []MouseButton{MouseLeftButton, MouseRightButton} {
		if inpututil.IsMouseButtonJustPressed(b) {
			pendingPress[b] = true
		}
		if inpututil.IsMouseButtonJustReleased(b) {
			pendingRelease[b] = true
		}
	}
	return nil
}

func (g *game) Draw(s *ebiten.Image) {
	now := time.Now()
	if !lastFrame.IsZero() {
		frameTime = float32(min(now.Sub(lastFrame).Seconds(), 0.1))
	} else {
		frameTime = 1.0 / 60
	}
	lastFrame = now

	frameKeys, pendingKeys = pendingKeys, frameKeys
	framePress, pendingPress = pendingPress, framePress
	frameRelease, pendingRelease = pendingRelease, frameRelease
	clear(pendingKeys)
	clear(pendingPress)
	clear(pendingRelease)

	screen, fullScreen = s, s
	if g.quit {
		return
	}
	if !g.update() {
		g.quit = true
		return
	}
	g.draw()
	flush()
	trimTextCaches()
	zoom, camOffset, mouseScale = 1, Vector2{}, Vector2{X: 1, Y: 1}
}

func (g *game) Layout(w, h int) (int, int) {
	screenW, screenH = w, h
	dpi = ebiten.Monitor().DeviceScaleFactor()
	return int(math.Ceil(float64(w) * dpi)), int(math.Ceil(float64(h) * dpi))
}

// Run opens the window and calls update then draw once per frame until update returns false
// or the window is closed.
func Run(width, height int, title string, update func() bool, draw func()) error {
	ebiten.SetWindowSize(width, height)
	SetWindowTitle(title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(false)
	err := ebiten.RunGameWithOptions(&game{update: update, draw: draw}, &ebiten.RunGameOptions{ScreenTransparent: false})
	if errors.Is(err, ebiten.Termination) || errors.Is(err, errQuit) {
		return nil
	}
	return err
}

func SetWindowTitle(title string) { ebiten.SetWindowTitle(title) }

func GetScreenWidth() int   { return screenW }
func GetScreenHeight() int  { return screenH }
func GetFrameTime() float32 { return frameTime }

// BeginDrawing and EndDrawing exist for raylib-style call sites; Ebitengine clears and presents.
func BeginDrawing() {}
func EndDrawing()   {}

func BeginMode2D(c Camera2D) {
	zoom = c.Zoom
	if zoom == 0 {
		zoom = 1
	}
	camOffset = Vector2{X: c.Offset.X - c.Target.X*zoom, Y: c.Offset.Y - c.Target.Y*zoom}
}

func EndMode2D() {
	zoom, camOffset = 1, Vector2{}
}

// BeginScissorMode clips drawing to a rectangle given in screen space.
func BeginScissorMode(x, y, w, h int32) {
	d := float32(dpi)
	r := image.Rect(int(float32(x)*d), int(float32(y)*d), int(math.Ceil(float64(float32(x+w)*d))), int(math.Ceil(float64(float32(y+h)*d))))
	flush()
	screen = fullScreen.SubImage(r).(*ebiten.Image)
}

func EndScissorMode() {
	flush()
	screen = fullScreen
}

func IsKeyPressed(k Key) bool                  { return frameKeys[k] }
func IsMouseButtonPressed(b MouseButton) bool  { return framePress[b] }
func IsMouseButtonReleased(b MouseButton) bool { return frameRelease[b] }
func IsMouseButtonDown(b MouseButton) bool     { return ebiten.IsMouseButtonPressed(b) }

// Wheel is the vertical scroll this frame. Positive is away from the user.
func Wheel() float32 {
	_, y := ebiten.Wheel()
	return float32(y)
}

func SetMouseScale(x, y float32) {
	mouseScale = Vector2{X: x, Y: y}
}

// GetMousePosition is in screen space, scaled by SetMouseScale.
func GetMousePosition() Vector2 {
	x, y := ebiten.CursorPosition()
	return Vector2{X: float32(float64(x)/dpi) * mouseScale.X, Y: float32(float64(y)/dpi) * mouseScale.Y}
}

// transform maps a point from the current drawing space to physical pixels.
func transform(x, y float32) (float32, float32) {
	d := float32(dpi)
	return (x*zoom + camOffset.X) * d, (y*zoom + camOffset.Y) * d
}

// scale is how many physical pixels one unit of the current drawing space covers.
func scale() float32 {
	return zoom * float32(dpi)
}

// GetFPS is the measured frame rate.
func GetFPS() float64 { return ebiten.ActualFPS() }
