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
	n := int((time.Since(tourStart).Seconds() - 3) / 3)
	if n < 0 {
		return
	}
	if n == tourStep {
		return
	}
	tourStep = n
	screens := []Screen{ScreenWorld, ScreenHero, ScreenSpells, ScreenQuests, ScreenShop, ScreenCitadel}
	if n < len(screens) {
		v.Screen = screens[n]
	} else if n == len(screens) {
		foe := os.Getenv("JEWEL_TOUR_FOE")
		if foe == "" {
			foe = "goblin"
		}
		v.startDuel(foe, v.Hero.Level, v.Hero.Location)
	}
}

// devTitle leaves the title screen after a moment on a tour, so the tour can show the rest.
func (c *GameController) devTitle() {
	if os.Getenv("JEWEL_TOUR") != "" && time.Since(tourStart) > 2500*time.Millisecond {
		c.title.on = false
	}
}

// devAutoplay lets the AI play the hero's side of duels too (JEWEL_AUTOPLAY), for screenshots.
func devAutoplay() bool { return os.Getenv("JEWEL_AUTOPLAY") != "" }

// devHideEditor hides the map editor's button while taking screenshots.
func devHideEditor() bool { return os.Getenv("JEWEL_SHOTS") != "" }
