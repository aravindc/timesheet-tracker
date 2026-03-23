package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// ---------------------------------------------------------------------------
// State helpers (HMAC-signed, stateless CSRF protection)
// ---------------------------------------------------------------------------

func generateOAuthState(provider string) (string, error) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	payload := provider + ":" + ts
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig, nil
}

func validateOAuthState(state, expectedProvider string) error {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid state format")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("invalid state encoding")
	}
	payload := string(payloadBytes)

	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(payload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expectedSig)) {
		return fmt.Errorf("state signature mismatch")
	}

	segs := strings.SplitN(payload, ":", 2)
	if len(segs) != 2 {
		return fmt.Errorf("invalid state payload")
	}
	if segs[0] != expectedProvider {
		return fmt.Errorf("state provider mismatch")
	}
	ts, err := strconv.ParseInt(segs[1], 10, 64)
	if err != nil || time.Now().Unix()-ts > 600 {
		return fmt.Errorf("state expired or invalid timestamp")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Provider config lookup
// ---------------------------------------------------------------------------

func oauthConfigForProvider(provider string) (*oauth2.Config, error) {
	switch provider {
	case "google":
		return googleOAuthConfig, nil
	case "facebook":
		return facebookOAuthConfig, nil
	}
	return nil, fmt.Errorf("unknown provider: %s", provider)
}

// ---------------------------------------------------------------------------
// OAuth handlers
// ---------------------------------------------------------------------------

func providerFromCtx(c *gin.Context) string {
	if v, ok := c.Get("provider"); ok {
		return v.(string)
	}
	return c.Param("provider")
}

func (s *Server) oauthLogin(c *gin.Context) {
	provider := providerFromCtx(c)
	cfg, err := oauthConfigForProvider(provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	state, err := generateOAuthState(provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, cfg.AuthCodeURL(state, oauth2.AccessTypeOnline))
}

func (s *Server) oauthCallback(c *gin.Context) {
	provider := providerFromCtx(c)
	redirectErr := func(msg string) {
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"?auth_error="+msg)
	}

	if err := validateOAuthState(c.Query("state"), provider); err != nil {
		redirectErr("invalid_state")
		return
	}

	cfg, err := oauthConfigForProvider(provider)
	if err != nil {
		redirectErr("unknown_provider")
		return
	}

	oauthToken, err := cfg.Exchange(c.Request.Context(), c.Query("code"))
	if err != nil {
		s.logger.Sugar().Errorf("OAuth token exchange failed (%s): %v", provider, err)
		redirectErr("token_exchange_failed")
		return
	}

	profile, err := fetchUserProfile(c.Request.Context(), provider, cfg, oauthToken)
	if err != nil {
		s.logger.Sugar().Errorf("fetchUserProfile failed (%s): %v", provider, err)
		redirectErr("profile_fetch_failed")
		return
	}

	user, err := s.upsertOAuthUser(profile, provider)
	if err != nil {
		s.logger.Sugar().Errorf("upsertOAuthUser failed (%s): %v", provider, err)
		redirectErr("upsert_failed")
		return
	}

	jwtToken, err := generateToken(user.ID, user.Username)
	if err != nil {
		redirectErr("token_generation_failed")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/#token="+jwtToken)
}

// ---------------------------------------------------------------------------
// Profile fetching
// ---------------------------------------------------------------------------

type oauthProfile struct {
	ProviderID string
	Email      string
	Name       string
}

func fetchUserProfile(ctx context.Context, provider string, cfg *oauth2.Config, token *oauth2.Token) (*oauthProfile, error) {
	client := cfg.Client(ctx, token)

	var url string
	switch provider {
	case "google":
		url = "https://www.googleapis.com/oauth2/v3/userinfo"
	case "facebook":
		url = "https://graph.facebook.com/me?fields=id,email,name"
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	p := &oauthProfile{}
	switch provider {
	case "google":
		p.ProviderID, _ = raw["sub"].(string)
		p.Email, _ = raw["email"].(string)
		name, _ := raw["name"].(string)
		p.Name = name
	case "facebook":
		p.ProviderID, _ = raw["id"].(string)
		p.Email, _ = raw["email"].(string)
		name, _ := raw["name"].(string)
		p.Name = name
	}

	if p.ProviderID == "" {
		return nil, fmt.Errorf("no provider ID in response")
	}
	return p, nil
}

// ---------------------------------------------------------------------------
// User upsert
// ---------------------------------------------------------------------------

var nonAlnum = regexp.MustCompile(`[^a-z0-9_]`)

func (s *Server) upsertOAuthUser(p *oauthProfile, provider string) (*User, error) {
	var user User

	err := s.db.QueryRow(
		`SELECT id, username FROM users WHERE provider = $1 AND provider_id = $2`,
		provider, p.ProviderID,
	).Scan(&user.ID, &user.Username)

	if err == nil {
		return &user, nil // existing user
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// New user — derive a unique username
	username, err := s.deriveUsername(p)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow(
		`INSERT INTO users (username, provider, provider_id, email)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, username`,
		username, provider, p.ProviderID, p.Email,
	).Scan(&user.ID, &user.Username)
	return &user, err
}

func (s *Server) deriveUsername(p *oauthProfile) (string, error) {
	base := p.Name
	if p.Email != "" {
		parts := strings.SplitN(p.Email, "@", 2)
		base = parts[0]
	}
	base = strings.ToLower(base)
	base = strings.ReplaceAll(base, " ", "_")
	base = nonAlnum.ReplaceAllString(base, "")
	if len(base) < 3 {
		base = "user"
	}

	// Check uniqueness, append suffix if needed
	candidate := base
	for i := 1; i <= 99; i++ {
		var exists bool
		if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`, candidate).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s%d", base, i)
	}
	return "", fmt.Errorf("could not derive unique username from '%s'", base)
}
