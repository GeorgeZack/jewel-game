//go:build dev

package core

import (
	"fmt"
	"jewel-game/rpg"
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

// devReturned is set once JEWEL_RETURN has made its return.
var devReturned bool

// devReturn makes one return as the game starts when JEWEL_RETURN is set, then starts the next run
// as the class it names (Knight if it isn't a class), so the entities' scene plays. JEWEL_META=n
// first banks n meta currency, to try buying in the Grimoire.
func (v *QuestView) devReturn() {
	if devReturned || os.Getenv("JEWEL_RETURN") == "" || v.Hero == nil || time.Since(tourStart) < 2*time.Second {
		return
	}
	devReturned = true
	var meta int
	fmt.Sscan(os.Getenv("JEWEL_META"), &meta)
	v.Profile.Meta += meta
	v.doReturn()
	v.modals = nil
	v.startRun(rpg.ClassByName(os.Getenv("JEWEL_RETURN")).Name)
}

func (v *QuestView) devTour() {
	v.devReturn()
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
	screens := []Screen{ScreenWorld, ScreenHero, ScreenSpells, ScreenQuests, ScreenShop, ScreenAcademy, ScreenOrrery, ScreenGrimoire}
	if n < len(screens) {
		v.Screen = screens[n]
		if v.Screen == ScreenOrrery {
			// JEWEL_ORRERY_ZOOM=n shows region n zoomed, as the phone's region buttons do
			fmt.Sscan(os.Getenv("JEWEL_ORRERY_ZOOM"), &v.orreryZoom)
			// JEWEL_ORRERY_NODE picks a node to look at; otherwise buy the first node the hero can,
			// to show buying works in the running game
			v.orrerySel = os.Getenv("JEWEL_ORRERY_NODE")
			for i := range rpg.Orrery {
				if v.orrerySel != "" {
					break
				}
				if v.Hero.BuyNode(rpg.Orrery[i].ID) {
					v.orrerySel = rpg.Orrery[i].ID
					v.save()
					break
				}
			}
		}
	} else if n == len(screens) {
		foe := os.Getenv("JEWEL_TOUR_FOE")
		if foe == "" {
			foe = "goblin"
		}
		v.startDuel(foe, v.Hero.Level, v.Hero.Location, "")
	} else if n == len(screens)+2 && devAutoplay() && v.InDuel() {
		// close a rule card's box, so the screenshots go on to the duel itself
		v.modals = nil
	}
}

// devTitle leaves the title screen after a moment on a tour or a dev puzzle, so they can show the rest.
func (c *GameController) devTitle() {
	if (os.Getenv("JEWEL_TOUR") != "" || devPuzzle() != "") && time.Since(tourStart) > 2500*time.Millisecond {
		c.title.on = false
	}
}

// devAutoplay lets the AI play the hero's side of duels too (JEWEL_AUTOPLAY), for screenshots.
func devAutoplay() bool { return os.Getenv("JEWEL_AUTOPLAY") != "" }

// devPuzzle opens the puzzle board named by JEWEL_PUZZLE once a hero is loaded, to try one out.
func devPuzzle() string { return os.Getenv("JEWEL_PUZZLE") }

// devPuzzleSolution is the stored solution of the route named by JEWEL_PUZZLE_ROUTE, which the
// puzzle screen then plays by itself.
func devPuzzleSolution(p *rpg.Puzzle) []string {
	for _, r := range p.Routes {
		if r.ID == os.Getenv("JEWEL_PUZZLE_ROUTE") {
			return r.Solution
		}
	}
	return nil
}

// devCard plays every duel on the rule card named by JEWEL_CARD, to try a card out.
func devCard() string { return os.Getenv("JEWEL_CARD") }

// devHideEditor hides the map editor's button while taking screenshots.
func devHideEditor() bool { return os.Getenv("JEWEL_SHOTS") != "" }
