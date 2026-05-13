package ui

import "testing"

func TestQuantityEntryCallsOnFocusLost(t *testing.T) {
	got := ""
	entry := NewQuantityEntry(func(value string) {
		got = value
	})
	entry.SetText("1,5")

	entry.FocusLost()

	if got != "1,5" {
		t.Fatalf("OnFocusLost value = %q, want 1,5", got)
	}
}
