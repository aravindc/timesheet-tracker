package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeClock struct{ t time.Time }

func (f *fakeClock) now() time.Time          { return f.t }
func (f *fakeClock) advance(d time.Duration) { f.t = f.t.Add(d) }

func newTestLimiter(maxAttempts int, window, base, max time.Duration) (*attemptLimiter, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	l := newAttemptLimiter(maxAttempts, window, base, max)
	l.now = clock.now
	return l, clock
}

func TestAttemptLimiterLocksOutAfterMaxAttempts(t *testing.T) {
	l, clock := newTestLimiter(3, time.Hour, time.Minute, 10*time.Minute)

	for i := 1; i <= 3; i++ {
		if ok, _ := l.attempt("k"); !ok {
			t.Fatalf("attempt %d should be allowed", i)
		}
	}
	ok, retry := l.attempt("k")
	if ok || retry != time.Minute {
		t.Fatalf("4th attempt: ok=%v retry=%v, want locked for 1m", ok, retry)
	}

	clock.advance(30 * time.Second)
	if ok, retry := l.attempt("k"); ok || retry != 30*time.Second {
		t.Fatalf("during lockout: ok=%v retry=%v, want locked for 30s", ok, retry)
	}

	if ok, _ := l.attempt("other"); !ok {
		t.Fatal("other keys must not be affected")
	}
}

func TestAttemptLimiterBackoffDoublesAndCaps(t *testing.T) {
	l, clock := newTestLimiter(1, 24*time.Hour, time.Minute, 5*time.Minute)

	want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 5 * time.Minute, 5 * time.Minute}
	for i, w := range want {
		if ok, _ := l.attempt("k"); !ok {
			t.Fatalf("attempt %d should be allowed after lockout expired", i+1)
		}
		if _, retry := l.attempt("k"); retry != w {
			t.Fatalf("lockout after attempt %d = %v, want %v", i+1, retry, w)
		}
		clock.advance(w)
	}
}

func TestAttemptLimiterWindowExpiryAndReset(t *testing.T) {
	l, clock := newTestLimiter(3, time.Hour, time.Minute, 10*time.Minute)

	l.attempt("k")
	l.attempt("k")
	clock.advance(time.Hour + time.Second)
	for i := 1; i <= 2; i++ {
		if ok, _ := l.attempt("k"); !ok {
			t.Fatalf("attempt %d after window expiry should be allowed", i)
		}
	}

	l.reset("k")
	for i := 1; i <= 3; i++ {
		if ok, _ := l.attempt("k"); !ok {
			t.Fatalf("attempt %d after reset should be allowed", i)
		}
	}
}

func TestAttemptLimiterConcurrentAttemptsRespectLimit(t *testing.T) {
	l, _ := newTestLimiter(5, time.Hour, time.Minute, 10*time.Minute)

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := l.attempt("k"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != 5 {
		t.Fatalf("allowed %d concurrent attempts, want 5", allowed)
	}
}

func TestAttemptLimiterPrune(t *testing.T) {
	l, clock := newTestLimiter(1, time.Hour, time.Minute, time.Minute)

	l.attempt("old")
	clock.advance(2 * time.Hour)
	l.attempt("fresh")
	l.prune()

	if _, ok := l.entries["old"]; ok {
		t.Error("expired entry should be pruned")
	}
	if _, ok := l.entries["fresh"]; !ok {
		t.Error("active entry should be kept")
	}
}

func TestLoginLimiterUsernameIsCaseInsensitiveAndIPNotResetOnSuccess(t *testing.T) {
	ll := newLoginLimiter()

	for i := 0; i < 5; i++ {
		ll.allow("10.0.0.1", "Alice")
	}
	if ok, _ := ll.allow("10.0.0.2", " alice "); ok {
		t.Fatal("username limit should apply across IPs and letter case")
	}

	ll.succeeded("ALICE")
	if ok, _ := ll.allow("10.0.0.3", "alice"); !ok {
		t.Fatal("successful login should clear the username limit")
	}

	ll.byIP.entries["10.0.0.1"].attempts = ll.byIP.maxAttempts - 1
	ll.allow("10.0.0.1", "bob")
	ll.succeeded("bob")
	if ok, _ := ll.allow("10.0.0.1", "carol"); ok {
		t.Fatal("a successful login must not reset the IP limit")
	}
}

func TestLoginReturns429WhenRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, _ := newTestServer()
	s.loginLimiter = newLoginLimiter()
	for i := 0; i < 5; i++ {
		s.loginLimiter.allow("192.0.2.1", "alice")
	}

	r := gin.New()
	r.POST("/api/auth/login", s.login)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"guess"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After = %q, want \"60\"", got)
	}
}
