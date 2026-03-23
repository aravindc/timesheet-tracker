package main

import (
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID           int            `json:"id"`
	Username     string         `json:"username" binding:"required"`
	PasswordHash sql.NullString `json:"-"`
	Provider     string         `json:"provider"`
	ProviderID   sql.NullString `json:"-"`
	Email        sql.NullString `json:"email,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
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
