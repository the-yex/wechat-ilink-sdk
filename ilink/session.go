package ilink

import (
	"sync"
	"time"
)

const (
	// SessionPauseDuration is how long to pause after session expiry.
	SessionPauseDuration = 60 * time.Minute
)

// SessionGuard protects against session expiry by pausing API calls.
// When errcode -14 is received, the client should pause making requests
// for a cooldown period to avoid request storms.
type SessionGuard struct {
	mu         sync.RWMutex
	pauseUntil time.Time
}

// NewSessionGuard creates a new session guard.
func NewSessionGuard() *SessionGuard {
	return &SessionGuard{}
}

// Pause initiates a cooldown period after session expiry.
func (g *SessionGuard) Pause() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pauseUntil = time.Now().Add(SessionPauseDuration)
}

// IsPaused returns true if the session is currently paused.
func (g *SessionGuard) IsPaused() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.pauseUntil.IsZero() {
		return false
	}

	if time.Now().After(g.pauseUntil) {
		return false
	}

	return true
}

// RemainingPause returns the remaining pause duration.
// Returns 0 if not paused.
func (g *SessionGuard) RemainingPause() time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.pauseUntil.IsZero() {
		return 0
	}

	remaining := time.Until(g.pauseUntil)
	if remaining < 0 {
		return 0
	}

	return remaining
}

// Reset clears the pause state.
func (g *SessionGuard) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pauseUntil = time.Time{}
}

// PauseUntil returns the time until which the session is paused.
// Returns zero time if not paused.
func (g *SessionGuard) PauseUntil() time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.pauseUntil
}