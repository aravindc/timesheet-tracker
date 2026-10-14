package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func setupOAuthTest(t *testing.T) *gin.Engine {
	t.Helper()
	jwtSecret = []byte("test-secret")
	frontendURL = "https://app.example"

	// Token endpoint that always refuses, so a callback that passes state
	// validation ends in token_exchange_failed without any network access.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
	}))
	t.Cleanup(tokenSrv.Close)
	googleOAuthConfig = &oauth2.Config{
		ClientID:    "id",
		RedirectURL: "https://app.example/api/auth/google/callback",
		Endpoint:    oauth2.Endpoint{AuthURL: "https://accounts.example/auth", TokenURL: tokenSrv.URL},
	}

	s, _ := newTestServer()
	r := gin.New()
	r.GET("/api/auth/google/login", func(c *gin.Context) { c.Set("provider", "google"); s.oauthLogin(c) })
	r.GET("/api/auth/google/callback", func(c *gin.Context) { c.Set("provider", "google"); s.oauthCallback(c) })
	return r
}

// startLogin runs the login step and returns the state sent to the provider
// and the state cookie set in the browser.
func startLogin(t *testing.T, r *gin.Engine) (string, *http.Cookie) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/auth/google/login", nil))

	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("bad redirect: %v", err)
	}
	state := loc.Query().Get("state")
	var cookie *http.Cookie
	for _, ck := range w.Result().Cookies() {
		if ck.Name == oauthStateCookie {
			cookie = ck
		}
	}
	if state == "" || cookie == nil {
		t.Fatalf("login must set state (%q) and cookie (%v)", state, cookie)
	}
	return state, cookie
}

func callback(r *gin.Engine, state string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?code=c&state="+url.QueryEscape(state), nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOAuthLoginSetsHardenedStateCookie(t *testing.T) {
	r := setupOAuthTest(t)
	oauthCookieSecure = true
	t.Cleanup(func() { oauthCookieSecure = false })

	_, ck := startLogin(t, r)
	if !ck.HttpOnly || !ck.Secure || ck.SameSite != http.SameSiteLaxMode || ck.Path != "/api/auth" || ck.MaxAge != 600 {
		t.Errorf("unexpected cookie attributes: %+v", ck)
	}
}

func TestOAuthCallbackRequiresMatchingStateCookie(t *testing.T) {
	r := setupOAuthTest(t)
	state, cookie := startLogin(t, r)
	_, otherCookie := startLogin(t, r)

	tests := []struct {
		name    string
		state   string
		cookie  *http.Cookie
		wantErr string
	}{
		// The login CSRF case: an attacker's valid state replayed in a
		// browser that never started the flow.
		{"no cookie", state, nil, "invalid_state"},
		{"cookie from another login", state, otherCookie, "invalid_state"},
		{"tampered state", state + "x", cookie, "invalid_state"},
		{"matching cookie", state, cookie, "token_exchange_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := callback(r, tt.state, tt.cookie)
			if w.Code != http.StatusTemporaryRedirect {
				t.Fatalf("status = %d, want redirect", w.Code)
			}
			if loc := w.Header().Get("Location"); !strings.HasSuffix(loc, "?auth_error="+tt.wantErr) {
				t.Errorf("Location = %q, want auth_error=%s", loc, tt.wantErr)
			}
			cleared := false
			for _, ck := range w.Result().Cookies() {
				if ck.Name == oauthStateCookie && ck.MaxAge < 0 {
					cleared = true
				}
			}
			if !cleared {
				t.Error("callback must clear the state cookie")
			}
		})
	}
}

func TestValidateOAuthState(t *testing.T) {
	jwtSecret = []byte("test-secret")
	state, nonce, err := generateOAuthState("google")
	if err != nil {
		t.Fatal(err)
	}

	if err := validateOAuthState(state, "google", nonce); err != nil {
		t.Errorf("valid state rejected: %v", err)
	}
	if err := validateOAuthState(state, "facebook", nonce); err == nil {
		t.Error("state for another provider must be rejected")
	}
	if err := validateOAuthState(state, "google", ""); err == nil {
		t.Error("state without a cookie nonce must be rejected")
	}

	// Expired: sign a payload with an old timestamp and the right nonce.
	old := strconv.FormatInt(time.Now().Add(-oauthStateTTL-time.Minute).Unix(), 10)
	payload := "google:" + old + ":" + nonce
	expired := encodeState(payload)
	if err := validateOAuthState(expired, "google", nonce); err == nil {
		t.Error("expired state must be rejected")
	}

	// A state signed under a different JWT_SECRET must not validate.
	jwtSecret = []byte("other-secret")
	if err := validateOAuthState(state, "google", nonce); err == nil {
		t.Error("state signed with a different secret must be rejected")
	}
}

func encodeState(payload string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + signOAuthState(payload)
}
