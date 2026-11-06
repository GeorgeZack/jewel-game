//go:build !dev

package core

// devTools is off in the public build: no map editor, no hot reload.
const devTools = false

func (v *QuestView) devTour() {}
