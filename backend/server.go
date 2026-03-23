package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	db     *sql.DB
	router *gin.Engine
	logger *zap.Logger
}

func (s *Server) initDB() error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR NOT NULL UNIQUE,
			password_hash VARCHAR NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS projects (
			id SERIAL PRIMARY KEY,
			name VARCHAR NOT NULL UNIQUE,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS work_days (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL,
			project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL,
			project_name VARCHAR,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			break_hours DOUBLE PRECISION DEFAULT 0,
			total_hours DOUBLE PRECISION,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(date, project_id)
		);

		CREATE TABLE IF NOT EXISTS hourly_rates (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rate DOUBLE PRECISION NOT NULL,
			is_current BOOLEAN NOT NULL DEFAULT FALSE,
			effective_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *Server) setupRouter() {
	s.router = gin.Default()
	for _, mw := range zapMiddlewares(s) {
		s.router.Use(mw)
	}

	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8089", "https://timesheet.pravitha.in"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	allowedIPs := getWhitelistedIPs()
	trustedProxies := getTrustedProxies()
	if len(trustedProxies) > 0 {
		if err := s.router.SetTrustedProxies(trustedProxies); err != nil {
			log.Printf("Warning: Failed to set trusted proxies: %v", err)
		}
		log.Printf("Configured trusted proxies: %v", trustedProxies)
	} else {
		log.Println("Warning: No trusted proxies configured, trusting all. Set TRUSTED_PROXIES in production!")
	}

	api := s.router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", IPWhitelistMiddleware(allowedIPs), s.register)
			auth.POST("/login", s.login)
			auth.GET("/verify", s.authMiddleware(), s.verify)
		}

		health := api.Group("/healthz")
		{
			health.GET("/live", s.liveness)
		}

		protected := api.Group("")
		protected.Use(s.authMiddleware())
		{
			protected.GET("/projects", s.getProjects)
			protected.POST("/projects", s.createProject)
			protected.PUT("/projects/:id", s.updateProject)
			protected.DELETE("/projects/:id", s.deleteProject)
			protected.GET("/stats", s.handleStats)

			protected.GET("/workdays", s.getWorkDays)
			protected.GET("/workdays/:year/:month", s.getWorkDaysByMonth)
			protected.POST("/workdays", s.createWorkDay)
			protected.PUT("/workdays/:id", s.updateWorkDay)
			protected.DELETE("/workdays/:id", s.deleteWorkDay)

			protected.GET("/hourly-rates", s.getHourlyRates)
			protected.POST("/hourly-rates", s.createHourlyRate)
			protected.PUT("/hourly-rates/:id", s.updateHourlyRate)
			protected.DELETE("/hourly-rates/:id", s.deleteHourlyRate)

			protected.GET("/payslip/:month", s.handlePayslip)
		}
	}
}

func (s *Server) liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
