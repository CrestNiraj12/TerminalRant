package app

import "context"

// Composer captures rant content from the user.
// Implemented by infrastructure (e.g. EditorComposer spawning $EDITOR).
// The inline TUI composer does NOT implement this — it lives entirely
// in the Bubble Tea layer as a model.
type Composer interface {
	Compose(ctx context.Context) (string, error)
}
