package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
