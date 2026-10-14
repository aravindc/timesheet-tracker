package main

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// dummyPasswordHash is compared against when no usable password hash exists,
// so a login for an unknown username takes as long as one with a wrong password.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser"), bcrypt.DefaultCost)

func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if ok, retryAfter := s.loginLimiter.allow(c.ClientIP(), req.Username); !ok {
		s.logger.Sugar().Warnw("login rate limited", "ip", c.ClientIP(), "retry_after", retryAfter)
		c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many login attempts. Please try again later."})
		return
	}

	var user User
	query := "SELECT id, username, password_hash, created_at FROM users WHERE username = $1 AND provider = 'local'"

	err := s.db.QueryRow(query, req.Username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	hash := dummyPasswordHash
	if err == nil && user.PasswordHash.Valid {
		hash = []byte(user.PasswordHash.String)
	}
	// Always run bcrypt and return the same message, so responses don't reveal
	// whether a username exists.
	if bcrypt.CompareHashAndPassword(hash, []byte(req.Password)) != nil || err != nil || !user.PasswordHash.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	s.loginLimiter.succeeded(req.Username)

	token, err := generateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:    token,
		Username: user.Username,
		UserID:   user.ID,
	})
}

func (s *Server) verify(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)
	username, _ := c.Get("username")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
		"valid":    true,
	})
}

func generateToken(userID int, username string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
