package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Telerallc/gamedev-vcs/database"
	"github.com/Telerallc/gamedev-vcs/internal/version"
	"github.com/Telerallc/gamedev-vcs/models"
	"github.com/gin-gonic/gin"
)

// CreateCommitRequest represents the request payload for creating a commit
type CreateCommitRequest struct {
	Message       string   `json:"message" binding:"required"`
	Branch        string   `json:"branch"`
	FilePaths     []string `json:"file_paths"`
	ParentCommits []string `json:"parent_commits"`
}

func (s *Server) createCommit(c *gin.Context) {
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

	var req CreateCommitRequest
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

	// Default branch if not specified
	if req.Branch == "" {
		req.Branch = project.DefaultBranch
	}

	// FIXED: Work with object store and file paths instead of database files
	var files []models.File

	if len(req.FilePaths) == 0 {
		// If no specific files provided, try to get files from the working directory
		// This is a fallback - normally CLI should provide file paths
		fileRepo := database.NewFileRepository(s.db.DB)
		dbFiles, err := fileRepo.GetProjectFiles(projectID, userID)
		if err == nil && len(dbFiles) > 0 {
			files = dbFiles
		} else {
			// No files in database and no file paths provided
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "no files to commit - please specify file_paths in request",
			})
			return
		}
	} else {
		// Create file objects from the provided file paths
		for _, filePath := range req.FilePaths {
			// First check if file exists in database
			fileRepo := database.NewFileRepository(s.db.DB)
			dbFile, err := fileRepo.GetFileByPath(projectID, filePath)

			if err == nil {
				// File exists in database, use it
				files = append(files, *dbFile)
			} else {
				// File not in database, create from object store/filesystem
				file, err := s.createFileFromPath(projectID, filePath, userID)
				if err != nil {
					// Log warning but continue with other files
					fmt.Printf("Warning: could not process file %s: %v\n", filePath, err)
					continue
				}
				if file != nil {
					files = append(files, *file)
				}
			}
		}
	}

	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "no valid files found to commit",
			"detail": "files may not exist or may not be uploaded to the server",
		})
		return
	}

	// Create commit service
	commitService := version.NewCommitService(s.db.DB)

	// Create the commit
	commit, err := commitService.CreateCommit(projectID, userID, req.Message, files, req.ParentCommits)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update branch HEAD
	err = commitService.UpdateBranchHead(projectID, req.Branch, commit.ID)
	if err != nil {
		fmt.Printf("Warning: failed to update branch HEAD: %v\n", err)
	}

	// Log the commit event
	s.logCommitEvent("commit_created", projectID, commit.ID, userID, c.GetString("user_name"), map[string]interface{}{
		"message":    req.Message,
		"branch":     req.Branch,
		"file_count": len(files),
	})

	c.JSON(http.StatusCreated, gin.H{
		"success":         true,
		"commit":          commit,
		"files_committed": len(files),
	})
}

// getCommitHistory retrieves commit history for a project/branch
func (s *Server) getCommitHistory(c *gin.Context) {
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

	// Get query parameters
	branch := c.DefaultQuery("branch", project.DefaultBranch)
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	// Create commit service and get history
	commitService := version.NewCommitService(s.db.DB)
	commits, err := commitService.GetCommitHistory(projectID, branch, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"commits": commits,
		"branch":  branch,
		"total":   len(commits),
	})
}

// getCommit retrieves a specific commit by ID
func (s *Server) getCommit(c *gin.Context) {
	projectID := c.Param("project")
	commitID := c.Param("commit")

	if projectID == "" || commitID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID and commit ID required"})
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

	// Get commit
	commitService := version.NewCommitService(s.db.DB)
	commit, err := commitService.GetCommitByID(commitID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "commit not found"})
		return
	}

	// Verify commit belongs to the project
	if commit.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "commit not found in project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"commit":  commit,
	})
}

// getFileHistory retrieves the history of a specific file
func (s *Server) getFileHistory(c *gin.Context) {
	projectID := c.Param("project")
	filePath := c.Param("file")

	if projectID == "" || filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID and file path required"})
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

	// Get query parameters
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	// Get file history
	commitService := version.NewCommitService(s.db.DB)
	fileVersions, err := commitService.GetFileHistory(projectID, filePath, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"versions": fileVersions,
		"file":     filePath,
		"total":    len(fileVersions),
	})
}

// diffCommits compares two commits
func (s *Server) diffCommits(c *gin.Context) {
	projectID := c.Param("project")
	fromCommit := c.Query("from")
	toCommit := c.Query("to")

	if projectID == "" || fromCommit == "" || toCommit == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "project ID, from commit, and to commit required",
		})
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

	// Get diff
	commitService := version.NewCommitService(s.db.DB)
	diff, err := commitService.DiffCommits(fromCommit, toCommit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"diff":    diff,
	})
}

// Helper function to log commit events
func (s *Server) logCommitEvent(eventType, projectID, commitID, userID, userName string, metadata map[string]interface{}) {
	// Add standard metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["commit_id"] = commitID

	// Use existing logging infrastructure
	s.logFileEvent(eventType, projectID, "", userID, userName, metadata)
}

// Helper function to create a File model from a file path
func (s *Server) createFileFromPath(projectID, filePath, userID string) (*models.File, error) {
	// Try to find the file in the object store first
	contentHash, err := s.findFileInObjectStore(filePath)
	if err != nil {
		// If not in object store, try to read from filesystem
		return s.createFileFromFilesystem(projectID, filePath, userID)
	}

	// Get file info from object store
	objectInfo, err := s.storage.GetObjectInfo(contentHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info for %s: %w", filePath, err)
	}

	// Create file model
	file := &models.File{
		ID:             fmt.Sprintf("file_%d", time.Now().UnixNano()),
		ProjectID:      projectID,
		Path:           filePath,
		ContentHash:    contentHash,
		Size:           objectInfo.Size,
		MimeType:       s.detectMimeType(filePath),
		Branch:         "main",
		IsLocked:       false,
		LastModifiedBy: &userID,
		LastModifiedAt: time.Now(),
	}

	return file, nil
}

// Helper function to find file in object store by checking uploaded objects
func (s *Server) findFileInObjectStore(filePath string) (string, error) {
	// This would check your object store for files with this path
	// For now, we'll calculate the hash from the file if it exists

	// Try to read the file and calculate hash
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("file not found: %s", filePath)
	}

	// Calculate SHA-256 hash
	hasher := sha256.New()
	hasher.Write(fileData)
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Check if this hash exists in object store
	if s.storage.ObjectExists(hash) {
		return hash, nil
	}

	return "", fmt.Errorf("file not found in object store")
}

// Helper function to create file from filesystem
func (s *Server) createFileFromFilesystem(projectID, filePath, userID string) (*models.File, error) {
	// Check if file exists in filesystem
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Read file and calculate hash
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Calculate content hash
	hasher := sha256.New()
	hasher.Write(fileData)
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	// Create file model
	file := &models.File{
		ID:             fmt.Sprintf("file_%d", time.Now().UnixNano()),
		ProjectID:      projectID,
		Path:           filePath,
		ContentHash:    contentHash,
		Size:           fileInfo.Size(),
		MimeType:       s.detectMimeType(filePath),
		Branch:         "main",
		IsLocked:       false,
		LastModifiedBy: &userID,
		LastModifiedAt: fileInfo.ModTime(),
	}

	return file, nil
}

// Helper function to detect MIME type
func (s *Server) detectMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".txt", ".md":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".html":
		return "text/html"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
