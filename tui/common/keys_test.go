package common

import "testing"

func TestDefaultKeyMap_HasCriticalBindings(t *testing.T) {
	km := DefaultKeyMap()
	if len(km.ToggleHints.Keys()) == 0 || km.ToggleHints.Keys()[0] != "?" {
		t.Fatalf("expected ? key binding for hints")
	}
	if len(km.ForceQuit.Keys()) == 0 || km.ForceQuit.Keys()[0] != "ctrl+c" {
		t.Fatalf("expected ctrl+c force quit binding")
	}
	if len(km.LoadMore.Keys()) != 0 {
		t.Fatalf("legacy load more should be hidden/disabled")
	}
}
