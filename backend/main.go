package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
)

var (
	jwtSecret           = []byte("")
	googleOAuthConfig   *oauth2.Config
	facebookOAuthConfig *oauth2.Config
	frontendURL         string
)

func main() {
	logger, err := InitLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	dbURL := os.Getenv("SUPABASE_DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	jwtSecretStr := os.Getenv("JWT_SECRET")
	if jwtSecretStr == "" {
		logger.Fatal("JWT_SECRET is not set")
	}
	jwtSecret = []byte(jwtSecretStr)

	redirectBase := os.Getenv("OAUTH_REDIRECT_BASE")
	frontendURL = os.Getenv("FRONTEND_URL")

	googleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  redirectBase + "/api/auth/google/callback",
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}

	facebookOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("FACEBOOK_CLIENT_ID"),
		ClientSecret: os.Getenv("FACEBOOK_CLIENT_SECRET"),
		RedirectURL:  redirectBase + "/api/auth/facebook/callback",
		Scopes:       []string{"email", "public_profile"},
		Endpoint:     facebook.Endpoint,
	}

	server := &Server{db: db, logger: logger}

	if err := server.initDB(); err != nil {
		logger.Fatal("Failed to init DB", zap.Error(err))
	}

	server.setupRouter()

	fmt.Println("Server starting on :8080")
	if err := server.router.Run(":8080"); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}
