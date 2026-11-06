//go:build dev

package core

import (
	"os"
	"time"
)

// devTools turns on the map editor and hot reload. Build with -tags dev (./start.sh does).
const devTools = true

// devTour steps through the screens on a timer when JEWEL_TOUR is set, for screenshots.
var (
	tourStep  = -1
	tourStart = time.Now()
)

func (v *QuestView) devTour() {
	if os.Getenv("JEWEL_TOUR") == "" || v.Hero == nil {
		return
	}
	n := int(time.Since(tourStart).Seconds() / 3)
	if n == tourStep {
		return
	}
	tourStep = n
	screens := []Screen{ScreenWorld, ScreenHero, ScreenSpells, ScreenQuests, ScreenShop, ScreenCitadel}
	if n < len(screens) {
		v.Screen = screens[n]
	} else if n == len(screens) {
		v.startDuel("goblin", 2, v.Hero.Location)
	}
}
