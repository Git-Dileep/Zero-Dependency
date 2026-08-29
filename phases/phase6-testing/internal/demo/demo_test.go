package demo

import (
	"testing"
)

func TestDemosRunWithoutPanic(t *testing.T) {
	// Simple test to ensure the demo scripts execute without panicking.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Demo panicked: %v", r)
		}
	}()

	// We redirect or ignore stdout if needed in a real test suite, 
	// but here we just ensure no crashes occur.
	RunStealKeyDemo()
	RunCompromiseDemo()
	RunTwoPanelDemo()
	RunDestroyDemo()
}
