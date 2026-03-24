package ilink

import (
	"testing"
	"time"
)

func TestSessionGuard_Pause(t *testing.T) {
	guard := NewSessionGuard()

	if guard.IsPaused() {
		t.Error("new guard should not be paused")
	}

	guard.Pause()

	if !guard.IsPaused() {
		t.Error("guard should be paused after Pause()")
	}

	remaining := guard.RemainingPause()
	if remaining <= 0 {
		t.Error("remaining pause should be positive")
	}

	if remaining > SessionPauseDuration {
		t.Errorf("remaining pause %v > max %v", remaining, SessionPauseDuration)
	}
}

func TestSessionGuard_Reset(t *testing.T) {
	guard := NewSessionGuard()
	guard.Pause()

	if !guard.IsPaused() {
		t.Fatal("guard should be paused")
	}

	guard.Reset()

	if guard.IsPaused() {
		t.Error("guard should not be paused after Reset()")
	}

	if guard.RemainingPause() != 0 {
		t.Errorf("remaining pause should be 0 after Reset(), got %v", guard.RemainingPause())
	}
}

func TestSessionGuard_RemainingPause(t *testing.T) {
	guard := NewSessionGuard()

	// Not paused, should return 0
	if guard.RemainingPause() != 0 {
		t.Errorf("unpaused guard should have 0 remaining, got %v", guard.RemainingPause())
	}

	guard.Pause()

	// Paused, should return positive duration
	remaining := guard.RemainingPause()
	if remaining <= 0 {
		t.Errorf("paused guard should have positive remaining, got %v", remaining)
	}

	// Wait a bit and check that remaining decreases
	time.Sleep(100 * time.Millisecond)
	newRemaining := guard.RemainingPause()
	if newRemaining >= remaining {
		t.Errorf("remaining should decrease over time, was %v, now %v", remaining, newRemaining)
	}
}

func TestSessionGuard_Concurrent(t *testing.T) {
	guard := NewSessionGuard()

	done := make(chan bool)

	// Start multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				guard.Pause()
				_ = guard.IsPaused()
				_ = guard.RemainingPause()
				guard.Reset()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
