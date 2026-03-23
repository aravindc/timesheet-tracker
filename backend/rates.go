package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) getHourlyRates(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)

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
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)

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
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)
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
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)
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
