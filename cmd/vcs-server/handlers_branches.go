package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Telerallc/gamedev-vcs/database"
	"github.com/Telerallc/gamedev-vcs/internal/version"
	"github.com/Telerallc/gamedev-vcs/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateBranchRequest represents a request to create a new branch
type CreateBranchRequest struct {
	Name        string `json:"name" binding:"required"`
	FromCommit  string `json:"from_commit"`
	FromBranch  string `json:"from_branch"`
	IsProtected bool   `json:"is_protected"`
}

// listBranches returns all branches for a project
func (s *Server) listBranches(c *gin.Context) {
	projectID := c.Param("project")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Verify user has read access to the project
	projectRepo := database.NewProjectRepository(s.db.DB)
	project, err := projectRepo.GetProjectByID(projectID, userID)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	if !project.HasPermission(userID, "read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	// Get all branches for the project
	var branches []models.Branch
	err = s.db.Where("project_id = ?", projectID).Order("is_default DESC, name ASC").Find(&branches).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get branches"})
		return
	}

	// Enhance branches with commit information
	commitService := version.NewCommitService(s.db.DB)
	var enhancedBranches []map[string]interface{}

	for _, branch := range branches {
		branchData := map[string]interface{}{
			"id":           branch.ID,
			"name":         branch.Name,
			"is_default":   branch.IsDefault,
			"is_protected": branch.IsProtected,
			"last_commit":  branch.LastCommit,
			"created_at":   branch.CreatedAt,
			"updated_at":   branch.UpdatedAt,
		}

		// Get commit count and last commit info
		if branch.LastCommit != "" {
			commits, err := commitService.GetCommitHistory(projectID, branch.Name, 1)
			if err == nil && len(commits) > 0 {
				branchData["last_commit_info"] = map[string]interface{}{
					"id":         commits[0].ID,
					"message":    commits[0].Message,
					"author_id":  commits[0].AuthorID,
					"created_at": commits[0].CreatedAt,
				}
			}

			// Get commit count
			allCommits, err := commitService.GetCommitHistory(projectID, branch.Name, 1000)
			if err == nil {
				branchData["commit_count"] = len(allCommits)
			}
		}

		enhancedBranches = append(enhancedBranches, branchData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"branches": enhancedBranches,
		"total":    len(branches),
	})
}

// createBranch creates a new branch
func (s *Server) createBranch(c *gin.Context) {
	projectID := c.Param("project")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user has write access to the project
	projectRepo := database.NewProjectRepository(s.db.DB)
	project, err := projectRepo.GetProjectByID(projectID, userID)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	if !project.HasPermission(userID, "write") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	// Check if branch name already exists
	var existingBranch models.Branch
	err = s.db.Where("project_id = ? AND name = ?", projectID, req.Name).First(&existingBranch).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "branch already exists"})
		return
	}

	// Determine source commit
	var sourceCommitID string
	if req.FromCommit != "" {
		sourceCommitID = req.FromCommit
	} else if req.FromBranch != "" {
		// Get the HEAD of the source branch
		var sourceBranch models.Branch
		err = s.db.Where("project_id = ? AND name = ?", projectID, req.FromBranch).First(&sourceBranch).Error
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source branch not found"})
			return
		}
		sourceCommitID = sourceBranch.LastCommit
	} else {
		// Use default branch HEAD
		var defaultBranch models.Branch
		err = s.db.Where("project_id = ? AND is_default = ?", projectID, true).First(&defaultBranch).Error
		if err == nil {
			sourceCommitID = defaultBranch.LastCommit
		}
	}

	// Create the branch
	newBranch := models.Branch{
		ID:          fmt.Sprintf("branch_%d", time.Now().UnixNano()),
		Name:        req.Name,
		ProjectID:   projectID,
		IsDefault:   false,
		IsProtected: req.IsProtected,
		LastCommit:  sourceCommitID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.Create(&newBranch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create branch"})
		return
	}

	// Create ref for the branch
	if sourceCommitID != "" {
		ref := models.Ref{
			ID:        fmt.Sprintf("refs/heads/%s:%s", req.Name, projectID),
			ProjectID: projectID,
			Name:      fmt.Sprintf("refs/heads/%s", req.Name),
			Type:      "branch",
			CommitID:  sourceCommitID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		s.db.Create(&ref)
	}

	// Log the branch creation
	s.logFileEvent("branch_created", projectID, req.Name, userID, c.GetString("user_name"), map[string]interface{}{
		"branch_name":   req.Name,
		"source_commit": sourceCommitID,
		"source_branch": req.FromBranch,
		"is_protected":  req.IsProtected,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"branch":  newBranch,
	})
}

// deleteBranch deletes a branch
func (s *Server) deleteBranch(c *gin.Context) {
	projectID := c.Param("project")
	branchName := c.Param("branch")

	if projectID == "" || branchName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID and branch name required"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Verify user has admin access to the project
	projectRepo := database.NewProjectRepository(s.db.DB)
	project, err := projectRepo.GetProjectByID(projectID, userID)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	if !project.HasPermission(userID, "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	// Get the branch
	var branch models.Branch
	err = s.db.Where("project_id = ? AND name = ?", projectID, branchName).First(&branch).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "branch not found"})
		return
	}

	// Don't allow deletion of default branch
	if branch.IsDefault {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete default branch"})
		return
	}

	// Don't allow deletion of protected branch unless forced
	force := c.Query("force") == "true"
	if branch.IsProtected && !force {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "branch is protected, use force=true to delete",
		})
		return
	}

	// Delete the branch and its ref
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Delete the branch
		if err := tx.Delete(&branch).Error; err != nil {
			return err
		}

		// Delete the ref
		return tx.Where("project_id = ? AND name = ?", projectID, fmt.Sprintf("refs/heads/%s", branchName)).Delete(&models.Ref{}).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete branch"})
		return
	}

	// Log the branch deletion
	s.logFileEvent("branch_deleted", projectID, branchName, userID, c.GetString("user_name"), map[string]interface{}{
		"branch_name":   branchName,
		"was_protected": branch.IsProtected,
		"forced":        force,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Branch '%s' deleted successfully", branchName),
	})
}

// switchBranch switches the default branch or updates branch protection
func (s *Server) updateBranch(c *gin.Context) {
	projectID := c.Param("project")
	branchName := c.Param("branch")

	if projectID == "" || branchName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID and branch name required"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var updates struct {
		IsDefault   *bool `json:"is_default"`
		IsProtected *bool `json:"is_protected"`
	}

	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user has admin access to the project
	projectRepo := database.NewProjectRepository(s.db.DB)
	project, err := projectRepo.GetProjectByID(projectID, userID)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	if !project.HasPermission(userID, "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	// Get the branch
	var branch models.Branch
	err = s.db.Where("project_id = ? AND name = ?", projectID, branchName).First(&branch).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "branch not found"})
		return
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// If setting as default, unset other default branches
		if updates.IsDefault != nil && *updates.IsDefault {
			err := tx.Model(&models.Branch{}).
				Where("project_id = ? AND is_default = ?", projectID, true).
				Update("is_default", false).Error
			if err != nil {
				return err
			}

			// Update project default branch
			err = tx.Model(&models.Project{}).
				Where("id = ?", projectID).
				Update("default_branch", branchName).Error
			if err != nil {
				return err
			}
		}

		// Update the branch
		updateData := make(map[string]interface{})
		if updates.IsDefault != nil {
			updateData["is_default"] = *updates.IsDefault
		}
		if updates.IsProtected != nil {
			updateData["is_protected"] = *updates.IsProtected
		}
		updateData["updated_at"] = time.Now()

		return tx.Model(&branch).Updates(updateData).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update branch"})
		return
	}

	// Log the branch update
	s.logFileEvent("branch_updated", projectID, branchName, userID, c.GetString("user_name"), map[string]interface{}{
		"branch_name":  branchName,
		"is_default":   updates.IsDefault,
		"is_protected": updates.IsProtected,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Branch '%s' updated successfully", branchName),
	})
}
