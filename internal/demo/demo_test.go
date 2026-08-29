package demo

import (
	"testing"
)

// TestRunStealKey verifies the steal-key demo runs without errors.
func TestRunStealKey(t *testing.T) {
	if err := RunStealKey(); err != nil {
		t.Fatalf("RunStealKey failed: %v", err)
	}
}

// TestRunCompromise verifies the compromise demo runs without errors.
func TestRunCompromise(t *testing.T) {
	if err := RunCompromise(); err != nil {
		t.Fatalf("RunCompromise failed: %v", err)
	}
}

// TestRunTwoPanel verifies the two-panel demo runs without errors.
func TestRunTwoPanel(t *testing.T) {
	if err := RunTwoPanel(); err != nil {
		t.Fatalf("RunTwoPanel failed: %v", err)
	}
}

// TestRunDestroy verifies the destroy demo runs without errors.
func TestRunDestroy(t *testing.T) {
	if err := RunDestroy(); err != nil {
		t.Fatalf("RunDestroy failed: %v", err)
	}
}

// TestRunDemo_All verifies the "all" demo mode runs every demo without errors.
func TestRunDemo_All(t *testing.T) {
	if err := RunDemo("all"); err != nil {
		t.Fatalf("RunDemo(all) failed: %v", err)
	}
}

// TestRunDemo_InvalidMode verifies that an invalid mode returns an error.
func TestRunDemo_InvalidMode(t *testing.T) {
	if err := RunDemo("nonexistent"); err == nil {
		t.Fatal("Expected error for unknown demo mode")
	}
}
