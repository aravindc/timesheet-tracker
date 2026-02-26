package main

import (
	"database/sql"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("")

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username" binding:"required"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	UserID   int    `json:"user_id"`
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type WorkDay struct {
	ID          int        `json:"id"`
	Date        string     `json:"date"` // Store as string in YYYY-MM-DD format
	ProjectID   *int       `json:"project_id"`
	ProjectName string     `json:"project_name"`
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	BreakHours  float64    `json:"break_hours"`
	TotalHours  *float64   `json:"total_hours"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProjectStat struct {
	ProjectID   int     `json:"project_id"`
	ProjectName string  `json:"project_name"`
	EntryCount  int     `json:"entry_count"`
	TotalHours  float64 `json:"total_hours"`
}

type HourlyRate struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	Rate          float64   `json:"rate"`
	IsCurrent     bool      `json:"is_current"`
	EffectiveFrom time.Time `json:"effective_from"`
	CreatedAt     time.Time `json:"created_at"`
}

type HourlyRateRequest struct {
	Rate          float64    `json:"rate" binding:"required,gt=0"`
	EffectiveFrom *time.Time `json:"effective_from"`
}

type TaxBand struct {
	Name   string  `json:"name"`
	Rate   float64 `json:"rate"`
	Amount float64 `json:"amount"`
}

type PayslipResponse struct {
	Month             string    `json:"month"`
	TotalHours        float64   `json:"total_hours"`
	TotalMinutes      int       `json:"total_minutes"`
	HourlyRate        float64   `json:"hourly_rate"`
	GrossPay          float64   `json:"gross_pay"`
	IncomeTax         float64   `json:"income_tax"`
	NationalInsurance float64   `json:"national_insurance"`
	NetPay            float64   `json:"net_pay"`
	TaxBreakdown      []TaxBand `json:"tax_breakdown"`
	NIBreakdown       []TaxBand `json:"ni_breakdown"`
}

type Server struct {
	db     *sql.DB
	router *gin.Engine
	logger *zap.Logger
}

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
		logger.Fatal("Failed to connect to Supabase", zap.Error(err))
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
	s.router.Use(ginzap.Ginzap(s.logger, time.RFC3339, true))
	s.router.Use(ginzap.RecoveryWithZap(s.logger, true))

	// Configure CORS - more permissive for development
	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8089", "https://timesheet.pravitha.in"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	allowedIPs := getWhitelistedIPs()
	// allowedCIDRs := getWhitelistedCIDRs()
	trustedProxies := getTrustedProxies()
	if len(trustedProxies) > 0 {
		if err := s.router.SetTrustedProxies(trustedProxies); err != nil {
			log.Printf("Warning: Failed to set trusted proxies: %v", err)
		}
		log.Printf("Configured trusted proxies: %v", trustedProxies)
	} else {
		// Trust all proxies (useful for Docker but less secure)
		// Remove this in production and specify exact proxy IPs
		log.Println("Warning: No trusted proxies configured, trusting all. Set TRUSTED_PROXIES in production!")
		// s.router.SetTrustedProxies([]string{"0.0.0.0/0", "::/0"})
	}

	// API routes
	api := s.router.Group("/api")
	{
		// Authentication routes (public)
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

		// Protected routes
		protected := api.Group("")
		protected.Use(s.authMiddleware())
		{
			// Project routes
			protected.GET("/projects", s.getProjects)
			protected.POST("/projects", s.createProject)
			protected.PUT("/projects/:id", s.updateProject)
			protected.DELETE("/projects/:id", s.deleteProject)
			protected.GET("/stats", s.handleStats)

			// Work days routes
			protected.GET("/workdays", s.getWorkDays)
			protected.GET("/workdays/:year/:month", s.getWorkDaysByMonth)
			protected.POST("/workdays", s.createWorkDay)
			protected.PUT("/workdays/:id", s.updateWorkDay)
			protected.DELETE("/workdays/:id", s.deleteWorkDay)

			// Hourly rates routes
			protected.GET("/hourly-rates", s.getHourlyRates)
			protected.POST("/hourly-rates", s.createHourlyRate)
			protected.PUT("/hourly-rates/:id", s.updateHourlyRate)
			protected.DELETE("/hourly-rates/:id", s.deleteHourlyRate)

			// Payslip route
			protected.GET("/payslip/:month", s.handlePayslip)
		}
	}
}

func (s *Server) liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Authentication handlers
func (s *Server) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", req.Username).Scan(&exists)
	s.logger.Sugar().Error(err)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	query := `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id, username, created_at
	`

	var user User
	err = s.db.QueryRow(query, req.Username, string(hashedPassword)).
		Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate JWT token
	token, err := generateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, LoginResponse{
		Token:    token,
		Username: user.Username,
		UserID:   user.ID,
	})
}

func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from database
	var user User
	query := "SELECT id, username, password_hash, created_at FROM users WHERE username = $1"

	err := s.db.QueryRow(query, req.Username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
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
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
		"valid":    true,
	})
}

// JWT helper functions
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

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func (s *Server) handleStats(c *gin.Context) {
	// Get stats from work_days grouped by project
	query := `
		SELECT 
			project_id,
			project_name,
			COUNT(*) as entry_count,
			SUM(total_hours) as total_hours
		FROM work_days
		WHERE end_time IS NOT NULL AND total_hours IS NOT NULL
		GROUP BY project_id, project_name
		ORDER BY total_hours DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var stats []ProjectStat
	for rows.Next() {
		var s ProjectStat
		if err := rows.Scan(&s.ProjectID, &s.ProjectName, &s.EntryCount, &s.TotalHours); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		stats = append(stats, s)
	}

	if stats == nil {
		stats = []ProjectStat{}
	}

	c.JSON(http.StatusOK, stats)
}

// Project handlers
func (s *Server) getProjects(c *gin.Context) {
	query := `
		SELECT id, name, description, created_at
		FROM projects
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		projects = append(projects, p)
	}

	if projects == nil {
		projects = []Project{}
	}

	c.JSON(http.StatusOK, projects)
}

func (s *Server) createProject(c *gin.Context) {
	var project Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
			INSERT INTO projects (name, description)
			VALUES ($1, $2)
			RETURNING id, created_at
	`

	err := s.db.QueryRow(query, project.Name, project.Description).
		Scan(&project.ID, &project.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func (s *Server) updateProject(c *gin.Context) {
	id := c.Param("id")

	var project Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
			UPDATE projects
			SET name = $1, description = $2
			WHERE id = $3
	`

	result, err := s.db.Exec(query, project.Name, project.Description, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	c.JSON(http.StatusOK, project)
}

func (s *Server) deleteProject(c *gin.Context) {
	id := c.Param("id")

	result, err := s.db.Exec("DELETE FROM projects WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Work days handlers
func (s *Server) getWorkDays(c *gin.Context) {
	query := `
		SELECT id, date, project_id, project_name, start_time, end_time, break_hours, total_hours, created_at, updated_at
		FROM work_days
		ORDER BY date DESC
		LIMIT 100
	`

	rows, err := s.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var workDays []WorkDay
	for rows.Next() {
		var wd WorkDay
		err := rows.Scan(&wd.ID, &wd.Date, &wd.ProjectID, &wd.ProjectName, &wd.StartTime, &wd.EndTime, &wd.BreakHours, &wd.TotalHours, &wd.CreatedAt, &wd.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		workDays = append(workDays, wd)
	}

	if workDays == nil {
		workDays = []WorkDay{}
	}

	c.JSON(http.StatusOK, workDays)
}

func (s *Server) getWorkDaysByMonth(c *gin.Context) {
	year := c.Param("year")
	month := c.Param("month")
	projectID := c.Query("project_id")

	var query string
	var args []interface{}

	if projectID != "" {
		query = `
			SELECT id, date, project_id, project_name, start_time, end_time, break_hours, total_hours, created_at, updated_at
			FROM work_days
			WHERE EXTRACT(YEAR FROM date) = $1 AND EXTRACT(MONTH FROM date) = $2 AND project_id = $3
			ORDER BY date ASC
		`
		args = []interface{}{year, month, projectID}
	} else {
		query = `
			SELECT id, date, project_id, project_name, start_time, end_time, break_hours, total_hours, created_at, updated_at
			FROM work_days
			WHERE EXTRACT(YEAR FROM date) = $1 AND EXTRACT(MONTH FROM date) = $2
			ORDER BY date ASC
		`
		args = []interface{}{year, month}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var workDays []WorkDay
	for rows.Next() {
		var wd WorkDay
		err := rows.Scan(&wd.ID, &wd.Date, &wd.ProjectID, &wd.ProjectName, &wd.StartTime, &wd.EndTime, &wd.BreakHours, &wd.TotalHours, &wd.CreatedAt, &wd.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		workDays = append(workDays, wd)
	}

	if workDays == nil {
		workDays = []WorkDay{}
	}

	c.JSON(http.StatusOK, workDays)
}

func (s *Server) createWorkDay(c *gin.Context) {
	var wd WorkDay
	if err := c.ShouldBindJSON(&wd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure we have a date
	if wd.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}

	var totalHours *float64
	if wd.StartTime != nil && wd.EndTime != nil {
		hours := wd.EndTime.Sub(*wd.StartTime).Hours() - wd.BreakHours
		totalHours = &hours
	}

	query := `
			INSERT INTO work_days
			(date, project_id, project_name, start_time, end_time, break_hours, total_hours)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(query, wd.Date, wd.ProjectID, wd.ProjectName, wd.StartTime, wd.EndTime, wd.BreakHours, totalHours).
		Scan(&wd.ID, &wd.CreatedAt, &wd.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	wd.TotalHours = totalHours
	c.JSON(http.StatusCreated, wd)
}

func (s *Server) updateWorkDay(c *gin.Context) {
	id := c.Param("id")

	var wd WorkDay
	if err := c.ShouldBindJSON(&wd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var totalHours *float64
	if wd.StartTime != nil && wd.EndTime != nil {
		hours := wd.EndTime.Sub(*wd.StartTime).Hours() - wd.BreakHours
		totalHours = &hours
	}

	query := `
		UPDATE work_days
		SET date = $1, project_id = $2, project_name = $3, start_time = $4, end_time = $5, break_hours = $6, total_hours = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8
	`

	result, err := s.db.Exec(query, wd.Date, wd.ProjectID, wd.ProjectName, wd.StartTime, wd.EndTime, wd.BreakHours, totalHours, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "work day not found"})
		return
	}

	wd.TotalHours = totalHours
	c.JSON(http.StatusOK, wd)
}

func (s *Server) deleteWorkDay(c *gin.Context) {
	id := c.Param("id")

	result, err := s.db.Exec("DELETE FROM work_days WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "work day not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Get whitelisted IPs from environment
func getWhitelistedIPs() []string {
	ipsStr := os.Getenv("WHITELIST_IPS")
	if ipsStr == "" {
		log.Println("Warning: No WHITELIST_IPS configured, defaulting to localhost only")
		return []string{"127.0.0.1", "::1"}
	}

	// Split by comma and trim spaces
	ips := strings.Split(ipsStr, ",")
	var cleanIPs []string
	for _, ip := range ips {
		cleanIP := strings.TrimSpace(ip)
		if cleanIP != "" {
			cleanIPs = append(cleanIPs, cleanIP)
		}
	}

	log.Printf("Loaded whitelisted IPs: %v", cleanIPs)
	return cleanIPs
}

func getWhitelistedCIDRs() []string {
	cidrsStr := os.Getenv("WHITELIST_CIDRS")
	if cidrsStr == "" {
		return []string{}
	}

	cidrs := strings.Split(cidrsStr, ",")
	var cleanCIDRs []string
	for _, cidr := range cidrs {
		cleanCIDR := strings.TrimSpace(cidr)
		if cleanCIDR != "" {
			cleanCIDRs = append(cleanCIDRs, cleanCIDR)
		}
	}

	log.Printf("Loaded whitelisted CIDRs: %v", cleanCIDRs)
	return cleanCIDRs
}

// Get trusted proxy IPs from environment
func getTrustedProxies() []string {
	proxiesStr := os.Getenv("TRUSTED_PROXIES")
	if proxiesStr == "" {
		// Default common proxy IPs
		return []string{"127.0.0.1", "::1"}
	}

	proxies := strings.Split(proxiesStr, ",")
	var cleanProxies []string
	for _, proxy := range proxies {
		cleanProxy := strings.TrimSpace(proxy)
		if cleanProxy != "" {
			cleanProxies = append(cleanProxies, cleanProxy)
		}
	}

	return cleanProxies
}

// IP Whitelist Middleware
func IPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		allowed := false
		for _, ip := range allowedIPs {
			if clientIP == ip {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: IP not whitelisted", "ClientIP": clientIP, "AllowedIPs": allowedIPs})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IP Range Whitelist (CIDR)
func IPRangeWhitelistMiddleware(allowedCIDRs []string) gin.HandlerFunc {
	var ipNets []*net.IPNet
	for _, cidr := range allowedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			ipNets = append(ipNets, ipNet)
		}
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		ip := net.ParseIP(clientIP)

		if ip == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid IP address"})
			c.Abort()
			return
		}

		allowed := false
		for _, ipNet := range ipNets {
			if ipNet.Contains(ip) {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: IP not in allowed range"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Payslip handler

func (s *Server) handlePayslip(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monthParam := c.Param("month") // e.g. "Jan-2026"

	// Parse month param
	t, err := time.Parse("Jan-2006", monthParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month format, use MMM-YYYY (e.g. Jan-2026)"})
		return
	}
	year := t.Year()
	month := int(t.Month())

	// Sum total_hours and break_hours for the month
	var totalHoursRaw, totalBreakHours float64
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(total_hours), 0), COALESCE(SUM(break_hours), 0)
		FROM work_days
		WHERE EXTRACT(YEAR FROM date) = $1 AND EXTRACT(MONTH FROM date) = $2
		  AND total_hours IS NOT NULL
	`, year, month).Scan(&totalHoursRaw, &totalBreakHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	netHours := totalHoursRaw - totalBreakHours
	if netHours < 0 {
		netHours = 0
	}
	totalMinutes := int(math.Round(netHours * 60))

	// Get current hourly rate for user
	var hourlyRate float64
	err = s.db.QueryRow(`
		SELECT rate FROM hourly_rates
		WHERE user_id = $1 AND is_current = TRUE
		LIMIT 1
	`, userID).Scan(&hourlyRate)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no current hourly rate set"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	grossPay := netHours * hourlyRate

	// UK Income Tax & NI 2024/25 — applied directly on monthly gross
	incomeTax, taxBreakdown := calcUKIncomeTax(grossPay)
	ni, niBreakdown := calcUKNationalInsurance(grossPay)

	netPay := grossPay - incomeTax - ni

	c.JSON(http.StatusOK, PayslipResponse{
		Month:             monthParam,
		TotalHours:        math.Round(netHours*100) / 100,
		TotalMinutes:      totalMinutes,
		HourlyRate:        hourlyRate,
		GrossPay:          math.Round(grossPay*100) / 100,
		IncomeTax:         math.Round(incomeTax*100) / 100,
		NationalInsurance: math.Round(ni*100) / 100,
		NetPay:            math.Round(netPay*100) / 100,
		TaxBreakdown:      taxBreakdown,
		NIBreakdown:       niBreakdown,
	})
}

// calcUKIncomeTax returns monthly income tax and breakdown for a given monthly gross (2024/25 bands).
// Thresholds are annual values divided by 12.
func calcUKIncomeTax(monthlyGross float64) (float64, []TaxBand) {
	type band struct {
		name  string
		lower float64 // monthly
		upper float64 // monthly
		rate  float64
	}
	bands := []band{
		{"Personal Allowance", 0, 12570.0 / 12, 0.0},
		{"Basic Rate (20%)", 12570.0 / 12, 50270.0 / 12, 0.20},
		{"Higher Rate (40%)", 50270.0 / 12, 125140.0 / 12, 0.40},
		{"Additional Rate (45%)", 125140.0 / 12, math.MaxFloat64, 0.45},
	}

	total := 0.0
	breakdown := []TaxBand{}
	for _, b := range bands {
		if monthlyGross <= b.lower {
			break
		}
		taxable := math.Min(monthlyGross, b.upper) - b.lower
		amount := taxable * b.rate
		total += amount
		if b.rate > 0 {
			breakdown = append(breakdown, TaxBand{Name: b.name, Rate: b.rate * 100, Amount: amount})
		}
	}
	return total, breakdown
}

// calcUKNationalInsurance returns monthly NI (employee Class 1) and breakdown (2024/25 bands).
// Thresholds are annual values divided by 12.
func calcUKNationalInsurance(monthlyGross float64) (float64, []TaxBand) {
	type band struct {
		name  string
		lower float64 // monthly
		upper float64 // monthly
		rate  float64
	}
	bands := []band{
		{"Below Primary Threshold", 0, 12570.0 / 12, 0.0},
		{"NI Standard Rate (8%)", 12570.0 / 12, 50270.0 / 12, 0.08},
		{"NI Upper Rate (2%)", 50270.0 / 12, math.MaxFloat64, 0.02},
	}

	total := 0.0
	breakdown := []TaxBand{}
	for _, b := range bands {
		if monthlyGross <= b.lower {
			break
		}
		taxable := math.Min(monthlyGross, b.upper) - b.lower
		amount := taxable * b.rate
		total += amount
		if b.rate > 0 {
			breakdown = append(breakdown, TaxBand{Name: b.name, Rate: b.rate * 100, Amount: amount})
		}
	}
	return total, breakdown
}

// Hourly rates handlers

func (s *Server) getHourlyRates(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := s.db.Query(`
		SELECT id, user_id, rate, is_current, effective_from, created_at
		FROM hourly_rates
		WHERE user_id = $1
		ORDER BY effective_from DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var rates []HourlyRate
	for rows.Next() {
		var r HourlyRate
		if err := rows.Scan(&r.ID, &r.UserID, &r.Rate, &r.IsCurrent, &r.EffectiveFrom, &r.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rates = append(rates, r)
	}

	if rates == nil {
		rates = []HourlyRate{}
	}

	c.JSON(http.StatusOK, rates)
}

func (s *Server) createHourlyRate(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req HourlyRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	effectiveFrom := time.Now()
	if req.EffectiveFrom != nil {
		effectiveFrom = *req.EffectiveFrom
	}

	tx, err := s.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE hourly_rates SET is_current = FALSE WHERE user_id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var r HourlyRate
	err = tx.QueryRow(`
		INSERT INTO hourly_rates (user_id, rate, is_current, effective_from)
		VALUES ($1, $2, TRUE, $3)
		RETURNING id, user_id, rate, is_current, effective_from, created_at
	`, userID, req.Rate, effectiveFrom).Scan(&r.ID, &r.UserID, &r.Rate, &r.IsCurrent, &r.EffectiveFrom, &r.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, r)
}

func (s *Server) updateHourlyRate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	var req HourlyRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	effectiveFrom := time.Now()
	if req.EffectiveFrom != nil {
		effectiveFrom = *req.EffectiveFrom
	}

	result, err := s.db.Exec(`
		UPDATE hourly_rates
		SET rate = $1, effective_from = $2
		WHERE id = $3 AND user_id = $4
	`, req.Rate, effectiveFrom, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "hourly rate not found"})
		return
	}

	var r HourlyRate
	err = s.db.QueryRow(`
		SELECT id, user_id, rate, is_current, effective_from, created_at
		FROM hourly_rates WHERE id = $1
	`, id).Scan(&r.ID, &r.UserID, &r.Rate, &r.IsCurrent, &r.EffectiveFrom, &r.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, r)
}

func (s *Server) deleteHourlyRate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	tx, err := s.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var wasCurrent bool
	err = tx.QueryRow(`
		DELETE FROM hourly_rates WHERE id = $1 AND user_id = $2 RETURNING is_current
	`, id, userID).Scan(&wasCurrent)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "hourly rate not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if wasCurrent {
		_, err = tx.Exec(`
			UPDATE hourly_rates SET is_current = TRUE
			WHERE id = (
				SELECT id FROM hourly_rates
				WHERE user_id = $1
				ORDER BY effective_from DESC
				LIMIT 1
			)
		`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func LoggerWithIP() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s | %3d | %13v | %15s | %-7s %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	})
}
