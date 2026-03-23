package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getWorkDays(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)

	rows, err := s.db.Query(`
		SELECT id, date, project_id, project_name, start_time, end_time, break_hours, total_hours, created_at, updated_at
		FROM work_days
		WHERE user_id = $1
		ORDER BY date DESC
		LIMIT 100
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var workDays []WorkDay
	for rows.Next() {
		var wd WorkDay
		if err := rows.Scan(&wd.ID, &wd.Date, &wd.ProjectID, &wd.ProjectName, &wd.StartTime, &wd.EndTime, &wd.BreakHours, &wd.TotalHours, &wd.CreatedAt, &wd.UpdatedAt); err != nil {
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
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)
	year := c.Param("year")
	month := c.Param("month")
	projectID := c.Query("project_id")

	var query string
	var args []interface{}

	if projectID != "" {
		query = `
			SELECT id, date, project_id, project_name, start_time, end_time, break_hours, total_hours, created_at, updated_at
			FROM work_days
			WHERE user_id = $1 AND EXTRACT(YEAR FROM date) = $2 AND EXTRACT(MONTH FROM date) = $3 AND project_id = $4
			ORDER BY date ASC
		`
		args = []interface{}{userID, year, month, projectID}
	} else {
		query = `
			SELECT id, date, project_id, project_name, start_time, end_time, break_hours, total_hours, created_at, updated_at
			FROM work_days
			WHERE user_id = $1 AND EXTRACT(YEAR FROM date) = $2 AND EXTRACT(MONTH FROM date) = $3
			ORDER BY date ASC
		`
		args = []interface{}{userID, year, month}
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
		if err := rows.Scan(&wd.ID, &wd.Date, &wd.ProjectID, &wd.ProjectName, &wd.StartTime, &wd.EndTime, &wd.BreakHours, &wd.TotalHours, &wd.CreatedAt, &wd.UpdatedAt); err != nil {
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
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)

	var wd WorkDay
	if err := c.ShouldBindJSON(&wd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if wd.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}

	var totalHours *float64
	if wd.StartTime != nil && wd.EndTime != nil {
		hours := wd.EndTime.Sub(*wd.StartTime).Hours() - wd.BreakHours
		totalHours = &hours
	}

	err := s.db.QueryRow(`
		INSERT INTO work_days
		(user_id, date, project_id, project_name, start_time, end_time, break_hours, total_hours)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, userID, wd.Date, wd.ProjectID, wd.ProjectName, wd.StartTime, wd.EndTime, wd.BreakHours, totalHours).
		Scan(&wd.ID, &wd.CreatedAt, &wd.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	wd.TotalHours = totalHours
	c.JSON(http.StatusCreated, wd)
}

func (s *Server) updateWorkDay(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)
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

	result, err := s.db.Exec(`
		UPDATE work_days
		SET date = $1, project_id = $2, project_name = $3, start_time = $4, end_time = $5,
		    break_hours = $6, total_hours = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND user_id = $9
	`, wd.Date, wd.ProjectID, wd.ProjectName, wd.StartTime, wd.EndTime, wd.BreakHours, totalHours, id, userID)
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
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)
	id := c.Param("id")

	result, err := s.db.Exec("DELETE FROM work_days WHERE id = $1 AND user_id = $2", id, userID)
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
