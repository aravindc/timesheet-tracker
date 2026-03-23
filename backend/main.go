package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var jwtSecret = []byte("")

func main() {
	logger, err := InitLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	dbURL := os.Getenv("SUPABASE_DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Fatal("Failed to connect to Supabase", zap.Error(err))
	}

	jwtSecretStr := os.Getenv("JWT_SECRET")
	if jwtSecretStr == "" {
		logger.Fatal("JWT_SECRET is not set")
	}
	jwtSecret = []byte(jwtSecretStr)

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
