package main

import (
	"database/sql"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handlePayslip(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monthParam := c.Param("month") // e.g. "Jan-2026"

	t, err := time.Parse("Jan-2006", monthParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month format, use MMM-YYYY (e.g. Jan-2026)"})
		return
	}
	year := t.Year()
	month := int(t.Month())

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
func calcUKIncomeTax(monthlyGross float64) (float64, []TaxBand) {
	type band struct {
		name  string
		lower float64
		upper float64
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
func calcUKNationalInsurance(monthlyGross float64) (float64, []TaxBand) {
	type band struct {
		name  string
		lower float64
		upper float64
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
