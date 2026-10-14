package main

import (
	"strings"
	"sync"
	"time"
)

// attemptLimiter tracks login attempts per key (client IP or username) and
// locks a key out with exponential backoff once it exceeds maxAttempts within
// window. Attempts are counted before the password is checked, so concurrent
// requests cannot slip past the limit. State is in memory, which suits the
// single backend instance this app runs as.
type attemptLimiter struct {
	mu          sync.Mutex
	entries     map[string]*attemptEntry
	maxAttempts int
	window      time.Duration
	baseLockout time.Duration
	maxLockout  time.Duration
	now         func() time.Time
}

type attemptEntry struct {
	attempts    int
	lastAttempt time.Time
	lockedUntil time.Time
}

func newAttemptLimiter(maxAttempts int, window, baseLockout, maxLockout time.Duration) *attemptLimiter {
	return &attemptLimiter{
		entries:     make(map[string]*attemptEntry),
		maxAttempts: maxAttempts,
		window:      window,
		baseLockout: baseLockout,
		maxLockout:  maxLockout,
		now:         time.Now,
	}
}

// attempt records an attempt for key. It returns false and the remaining
// lockout when the key is locked out, in which case nothing is recorded.
func (l *attemptLimiter) attempt(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	e, ok := l.entries[key]
	if !ok {
		e = &attemptEntry{}
		l.entries[key] = e
	}
	if now.Before(e.lockedUntil) {
		return false, e.lockedUntil.Sub(now)
	}
	if now.Sub(e.lastAttempt) > l.window {
		e.attempts = 0
	}

	e.attempts++
	e.lastAttempt = now
	if e.attempts >= l.maxAttempts {
		// Double the lockout for every attempt past the limit, up to maxLockout.
		lockout := l.maxLockout
		if shift := e.attempts - l.maxAttempts; shift < 32 {
			if d := l.baseLockout << shift; d > 0 && d < l.maxLockout {
				lockout = d
			}
		}
		e.lockedUntil = now.Add(lockout)
	}
	return true, 0
}

// reset forgets all attempts for key, e.g. after a successful login.
func (l *attemptLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// prune drops entries whose lockout has expired and whose attempts have aged
// out of the window, so keys chosen by an attacker cannot grow memory forever.
func (l *attemptLimiter) prune() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for key, e := range l.entries {
		if !now.Before(e.lockedUntil) && now.Sub(e.lastAttempt) > l.window {
			delete(l.entries, key)
		}
	}
}

// loginLimiter combines a per-IP limit, which slows one client guessing many
// accounts, with a per-username limit, which slows many clients guessing one
// account.
type loginLimiter struct {
	byIP       *attemptLimiter
	byUsername *attemptLimiter
}

// The window is longer than the maximum lockout so that a key which keeps
// failing stays at the maximum lockout instead of starting over. Once maxed
// out, an attacker gets roughly one guess per username every 30 minutes.
func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		byIP:       newAttemptLimiter(20, 2*time.Hour, time.Minute, time.Hour),
		byUsername: newAttemptLimiter(5, time.Hour, time.Minute, 30*time.Minute),
	}
}

// allow records a login attempt and reports whether it may proceed. The
// username limit is only consumed when the IP limit allows the attempt.
func (l *loginLimiter) allow(ip, username string) (bool, time.Duration) {
	if ok, retry := l.byIP.attempt(ip); !ok {
		return false, retry
	}
	return l.byUsername.attempt(normalizeUsername(username))
}

// succeeded clears the username's attempts. The IP count is left alone so an
// attacker cannot reset it by logging in to their own account between guesses.
func (l *loginLimiter) succeeded(username string) {
	l.byUsername.reset(normalizeUsername(username))
}

// startPruning removes stale entries every interval until stop is closed.
func (l *loginLimiter) startPruning(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				l.byIP.prune()
				l.byUsername.prune()
			case <-stop:
				return
			}
		}
	}()
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}
