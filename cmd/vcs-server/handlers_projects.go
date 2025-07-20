// handlers_projects.go - Add this as a new file or merge with existing handlers
package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Telerallc/gamedev-vcs/models"
	"github.com/gin-gonic/gin"
)

// Project management handlers

// Create a new project
func (s *Server) createProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ProjectResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Get user ID from auth middleware
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ProjectResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	// Generate project ID and slug
	projectID := fmt.Sprintf("proj_%d", time.Now().UnixNano())
	slug := generateSlug(req.Name)

	// Check if slug already exists for this user
	var existingProject models.Project
	err := s.db.Where("owner_id = ? AND slug = ?", userID, slug).First(&existingProject).Error
	if err == nil {
		c.JSON(http.StatusConflict, ProjectResponse{
			Success: false,
			Error:   fmt.Sprintf("Project with name '%s' already exists", req.Name),
		})
		return
	}

	// Create project
	project := models.Project{
		ID:            projectID,
		Name:          req.Name,
		Slug:          slug,
		Description:   req.Description,
		IsPrivate:     req.IsPrivate,
		DefaultBranch: "main",
		OwnerID:       &userID,
		Settings: models.JSON{
			"created_via": "vcs_cli",
			"vcs_version": "1.0",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ProjectResponse{
			Success: false,
			Error:   "Failed to create project",
		})
		return
	}

	c.JSON(http.StatusCreated, ProjectResponse{
		Success: true,
		Project: project,
	})
}

// updateProject updates project details
func (s *Server) updateProject(c *gin.Context) {
	projectID := c.Param("projectId")
	userID := c.GetString("user_id")

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ProjectResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	var project models.Project
	err := s.db.Where("(id = ? OR slug = ?) AND owner_id = ?", projectID, projectID, userID).First(&project).Error
	if err != nil {
		c.JSON(http.StatusNotFound, ProjectResponse{
			Success: false,
			Error:   "Project not found",
		})
		return
	}

	// Update fields
	project.Name = req.Name
	project.Description = req.Description
	project.IsPrivate = req.IsPrivate
	project.UpdatedAt = time.Now()

	if err := s.db.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ProjectResponse{
			Success: false,
			Error:   "Failed to update project",
		})
		return
	}

	c.JSON(http.StatusOK, ProjectResponse{
		Success: true,
		Project: project,
	})
}

// deleteProject deletes a project
func (s *Server) deleteProject(c *gin.Context) {
	projectID := c.Param("projectId")
	userID := c.GetString("user_id")

	var project models.Project
	err := s.db.Where("(id = ? OR slug = ?) AND owner_id = ?", projectID, projectID, userID).First(&project).Error
	if err != nil {
		c.JSON(http.StatusNotFound, ProjectResponse{
			Success: false,
			Error:   "Project not found",
		})
		return
	}

	if err := s.db.Delete(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ProjectResponse{
			Success: false,
			Error:   "Failed to delete project",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project deleted successfully",
	})
}

// Helper functions

// generateSlug creates a URL-friendly slug from project name
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove special characters, keep only alphanumeric and hyphens
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Remove multiple consecutive hyphens
	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}
