package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getProjects(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)

	rows, err := s.db.Query(`
		SELECT id, name, description, created_at
		FROM projects
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt); err != nil {
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
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)

	var project Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.db.QueryRow(`
		INSERT INTO projects (name, description, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, project.Name, project.Description, userID).Scan(&project.ID, &project.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func (s *Server) updateProject(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)
	id := c.Param("id")

	var project Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.db.Exec(`
		UPDATE projects
		SET name = $1, description = $2
		WHERE id = $3 AND user_id = $4
	`, project.Name, project.Description, id, userID)
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
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)
	id := c.Param("id")

	result, err := s.db.Exec("DELETE FROM projects WHERE id = $1 AND user_id = $2", id, userID)
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

func (s *Server) handleStats(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userID, _ := userIDRaw.(int)

	rows, err := s.db.Query(`
		SELECT
			project_id,
			project_name,
			COUNT(*) as entry_count,
			SUM(total_hours) as total_hours
		FROM work_days
		WHERE end_time IS NOT NULL AND total_hours IS NOT NULL AND user_id = $1
		GROUP BY project_id, project_name
		ORDER BY total_hours DESC
	`, userID)
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
