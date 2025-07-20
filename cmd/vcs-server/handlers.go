package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Telerallc/gamedev-vcs/database"
	"github.com/Telerallc/gamedev-vcs/internal/analytics"
	"github.com/Telerallc/gamedev-vcs/internal/auth"
	fileops "github.com/Telerallc/gamedev-vcs/internal/fileOps"
	"github.com/Telerallc/gamedev-vcs/internal/state"
	"github.com/Telerallc/gamedev-vcs/models"

	// "github.com/Telerallc/gamedev-vcs/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// File upload handler
func (s *Server) uploadFile(c *gin.Context) {
	// Get project ID from query parameter
	projectID := c.Query("project")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	fmt.Printf("DEBUG: Upload request for project: %s by user: %s\n", projectID, userID)

	// Check if user has write access to the project
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

	// Parse multipart form
	err = c.Request.ParseMultipartForm(100 << 20) // 100MB max
	if err != nil {
		fmt.Printf("DEBUG: Failed to parse form: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		fmt.Printf("DEBUG: No file provided: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}
	defer file.Close()

	filePath := c.PostForm("file_path")
	if filePath == "" {
		filePath = header.Filename
	}

	fmt.Printf("DEBUG: Uploading file: %s\n", filePath)

	userName := c.PostForm("user_name")
	if userName == "" {
		userName = c.GetString("user_name")
	}

	sessionID := c.PostForm("session_id")
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	branch := c.PostForm("branch")
	// if branch == "" {
	// 	branch = project.DefaultBranch
	// }

	commitHash := c.PostForm("commit_hash")
	commitMessage := c.PostForm("commit_message")

	// Create upload request
	uploadReq := &fileops.UploadRequest{
		ProjectID:     projectID,
		FilePath:      filePath,
		UserID:        userID,
		UserName:      userName,
		SessionID:     sessionID,
		Content:       file,
		CommitHash:    commitHash,
		CommitMessage: commitMessage,
		Metadata: map[string]string{
			"filename":     header.Filename,
			"content-type": header.Header.Get("Content-Type"),
			"uploaded_at":  time.Now().Format(time.RFC3339),
			"branch":       branch,
		},
	}

	fmt.Printf("DEBUG: About to call fileOps.UploadFile\n")

	// Check if fileOps is nil
	if s.fileOps == nil {
		fmt.Printf("DEBUG: fileOps is nil!\n")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "file operations not initialized"})
		return
	}

	// Upload file
	result, err := s.fileOps.UploadFile(uploadReq)
	if err != nil {
		fmt.Printf("DEBUG: Upload failed: %v\n", err)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("DEBUG: Upload successful: %s\n", result.ContentHash)

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"content_hash":       result.ContentHash,
		"size":               result.Size,
		"file_path":          result.FilePath,
		"analytics_recorded": result.AnalyticsRecorded,
		"asset_info":         result.AssetInfo,
		"dependencies":       result.Dependencies,
	})
}

// File download handler
func (s *Server) downloadFile(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content hash required"})
		return
	}

	userID := c.GetString("user_id")
	projectID := c.Query("project_id")

	downloadReq := &fileops.DownloadRequest{
		ProjectID:   projectID,
		ContentHash: hash,
		UserID:      userID,
		SessionID:   c.Query("session_id"),
	}

	result, err := s.fileOps.DownloadFile(downloadReq)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	defer result.Content.Close()

	// Set headers
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(result.Size, 10))
	c.Header("Content-Hash", result.ContentHash)

	// Stream content
	if _, err := io.Copy(c.Writer, result.Content); err != nil {
		// Log error but can't return JSON at this point
		fmt.Printf("Error streaming file: %v\n", err)
	}
}

// Chunked upload handler
func (s *Server) uploadChunk(c *gin.Context) {
	projectID := c.Query("project_id")
	filePath := c.Query("file_path")
	sessionID := c.Query("session_id")
	chunkIndexStr := c.Query("chunk_index")
	totalChunksStr := c.Query("total_chunks")
	userID := c.GetString("user_id")

	if projectID == "" || filePath == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameters"})
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk_index"})
		return
	}

	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid total_chunks"})
		return
	}

	// Read chunk data
	chunkData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read chunk data"})
		return
	}

	// Upload chunk
	chunkReq := &fileops.ChunkUploadRequest{
		ProjectID:   projectID,
		FilePath:    filePath,
		SessionID:   sessionID,
		ChunkIndex:  chunkIndex,
		TotalChunks: totalChunks,
		ChunkData:   chunkData,
		UserID:      userID,
	}

	err = s.fileOps.UploadChunk(chunkReq)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"chunk_index":  chunkIndex,
		"total_chunks": totalChunks,
		"session_id":   sessionID,
	})
}

// Finalize chunked upload
func (s *Server) finalizeUpload(c *gin.Context) {
	var req struct {
		ProjectID     string            `json:"project_id"`
		FilePath      string            `json:"file_path"`
		SessionID     string            `json:"session_id"`
		TotalChunks   int               `json:"total_chunks"`
		UserID        string            `json:"user_id"`
		UserName      string            `json:"user_name"`
		CommitHash    string            `json:"commit_hash"`
		CommitMessage string            `json:"commit_message"`
		Metadata      map[string]string `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.UserID == "" {
		req.UserID = c.GetString("user_id")
	}

	if req.Metadata == nil {
		req.Metadata = make(map[string]string)
	}
	req.Metadata["finalized_at"] = time.Now().Format(time.RFC3339)

	uploadReq := &fileops.UploadRequest{
		ProjectID:     req.ProjectID,
		FilePath:      req.FilePath,
		UserID:        req.UserID,
		UserName:      req.UserName,
		SessionID:     req.SessionID,
		CommitHash:    req.CommitHash,
		CommitMessage: req.CommitMessage,
		Metadata:      req.Metadata,
	}

	result, err := s.fileOps.FinalizeChunkedUpload(req.SessionID, req.TotalChunks, req.Metadata, uploadReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"content_hash":       result.ContentHash,
		"size":               result.Size,
		"file_path":          result.FilePath,
		"analytics_recorded": result.AnalyticsRecorded,
		"asset_info":         result.AssetInfo,
		"dependencies":       result.Dependencies,
	})
}

// File locking handlers

func (s *Server) lockFile(c *gin.Context) {
	projectID := c.Param("project")
	filePath := c.Param("file")

	// Remove leading slash from wildcard parameter
	filePath = strings.TrimPrefix(filePath, "/")

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Check project access
	// projectRepo := database.NewProjectRepository(s.db.DB)
	// project, err := projectRepo.GetProjectByID(projectID, userID)
	// if err != nil {
	// 	if err.Error() == "access denied" {
	// 		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	// 		return
	// 	}
	// 	c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
	// 	return
	// }

	// if !project.HasPermission(userID, "write") {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	// 	return
	// }

	var req struct {
		UserName  string `json:"user_name"`
		SessionID string `json:"session_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.UserName == "" {
		req.UserName = c.GetString("user_name")
	}

	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	// Check if file exists in database and update lock status
	// fileRepo := database.NewFileRepository(s.db.DB)
	// dbFile, err := fileRepo.GetFileByPath(projectID, filePath)
	// if err == nil {
	// 	// File exists in database, check if already locked
	// 	if dbFile.IsLocked && dbFile.LockedBy != nil && *dbFile.LockedBy != userID {
	// 		c.JSON(http.StatusConflict, gin.H{
	// 			"error":  fmt.Sprintf("File is already locked by user %s", *dbFile.LockedBy),
	// 			"locked": false,
	// 		})
	// 		return
	// 	}

	// 	// Update lock status in database
	// 	now := time.Now()
	// 	err = s.db.Model(&dbFile).Updates(map[string]interface{}{
	// 		"is_locked": true,
	// 		"locked_by": userID,
	// 		"locked_at": &now,
	// 	}).Error
	// 	if err != nil {
	// 		fmt.Printf("Failed to update file lock in database: %v\n", err)
	// 	}
	// }

	// Call existing lock functionality
	lockReq := &fileops.LockRequest{
		ProjectID: projectID,
		FilePath:  filePath,
		UserID:    userID,
		UserName:  req.UserName,
		SessionID: req.SessionID,
	}

	lock, err := s.fileOps.LockFile(lockReq)
	if err != nil {
		// Rollback database lock if Redis lock failed
		// if dbFile.ID != "" {
		// 	s.db.Model(&dbFile).Updates(map[string]interface{}{
		// 		"is_locked": false,
		// 		"locked_by": nil,
		// 		"locked_at": nil,
		// 	})
		// }

		c.JSON(http.StatusConflict, gin.H{
			"error":  err.Error(),
			"locked": false,
		})
		return
	}

	// Log the lock event
	// s.logFileEvent("file_locked", projectID, filePath, userID, req.UserName, map[string]interface{}{
	// 	"session_id": req.SessionID,
	// })

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"locked":    true,
		"lock_info": lock,
	})
}

func (s *Server) unlockFile(c *gin.Context) {
	projectID := c.Param("project")
	filePath := c.Param("file")

	// Remove leading slash from wildcard parameter
	filePath = strings.TrimPrefix(filePath, "/")

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Update database lock status
	// var dbFile models.File
	// err := s.db.Where("project_id = ? AND path = ?", projectID, filePath).First(&dbFile).Error
	// if err == nil {
	// 	// Verify user can unlock this file
	// 	if dbFile.IsLocked && dbFile.LockedBy != nil && *dbFile.LockedBy != userID {
	// 		// Check if user has admin permissions
	// 		projectRepo := database.NewProjectRepository(s.db.DB)
	// 		project, err := projectRepo.GetProjectByID(projectID, userID)
	// 		if err != nil || !project.HasPermission(userID, "admin") {
	// 			c.JSON(http.StatusForbidden, gin.H{"error": "cannot unlock file locked by another user"})
	// 			return
	// 		}
	// 	}

	// 	// Update lock status in database
	// 	err = s.db.Model(&dbFile).Updates(map[string]interface{}{
	// 		"is_locked": false,
	// 		"locked_by": nil,
	// 		"locked_at": nil,
	// 	}).Error
	// 	if err != nil {
	// 		fmt.Printf("Failed to update file unlock in database: %v\n", err)
	// 	}
	// }

	// // Call existing unlock functionality
	// err = s.fileOps.UnlockFile(projectID, filePath, userID)
	// if err != nil {
	// 	c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	// 	return
	// }

	// Log the unlock event
	// s.logFileEvent("file_unlocked", projectID, filePath, userID, c.GetString("user_name"), map[string]interface{}{})

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"unlocked": true,
		"project":  projectID,
		"file":     filePath,
	})
}

func (s *Server) listLocks(c *gin.Context) {
	projectID := c.Param("project")

	locks, err := s.fileOps.ListProjectLocks(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"project": projectID,
		"locks":   locks,
	})
}

// Analytics handlers

func (s *Server) getProductivityMetrics(c *gin.Context) {
	projectID := c.Param("project")
	daysStr := c.DefaultQuery("days", "7")

	days, err := strconv.Atoi(daysStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid days parameter"})
		return
	}

	metrics, err := s.analytics.GetTeamProductivity(projectID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"project": projectID,
		"period":  fmt.Sprintf("Last %d days", days),
		"metrics": metrics,
	})
}

func (s *Server) getActivityFeed(c *gin.Context) {
	projectID := c.Param("project")
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
		return
	}

	activities, err := s.analytics.GetActivityFeed(projectID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"project":    projectID,
		"activities": activities,
	})
}

func (s *Server) getDependencyGraph(c *gin.Context) {
	projectID := c.Param("project")
	assetPath := c.Query("asset_path")

	if assetPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "asset_path parameter required"})
		return
	}

	// Get dependencies
	dependencies, err := s.fileOps.GetAssetDependencies(assetPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get dependency impact (what depends on this asset)
	dependents, err := s.fileOps.GetDependencyImpact(assetPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"project":      projectID,
		"asset_path":   assetPath,
		"dependencies": dependencies,
		"dependents":   dependents,
	})
}

func (s *Server) getTeamInsights(c *gin.Context) {
	projectID := c.Param("project")
	daysStr := c.DefaultQuery("days", "30")

	days, err := strconv.Atoi(daysStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid days parameter"})
		return
	}

	insights, err := s.analytics.GetTeamInsights(projectID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"insights": insights,
	})
}

// Presence and collaboration handlers

func (s *Server) getProjectPresence(c *gin.Context) {
	projectID := c.Param("project")

	presence, err := s.fileOps.GetProjectPresence(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"project":  projectID,
		"presence": presence,
	})
}

func (s *Server) updatePresence(c *gin.Context) {
	projectID := c.Param("project")
	userID := c.GetString("user_id")

	var req struct {
		UserName    string `json:"user_name"`
		Status      string `json:"status"`
		CurrentFile string `json:"current_file"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.UserName == "" {
		req.UserName = userID
	}

	status := state.StatusOnline
	switch req.Status {
	case "editing":
		status = state.StatusEditing
	case "idle":
		status = state.StatusIdle
	case "offline":
		status = state.StatusOffline
	}

	err := s.stateManager.UpdatePresence(userID, req.UserName, projectID, status, req.CurrentFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"user_id":      userID,
		"project":      projectID,
		"status":       req.Status,
		"current_file": req.CurrentFile,
	})
}

// WebSocket handler for real-time collaboration
func (s *Server) websocketHandler(c *gin.Context) {
	projectID := c.Query("project_id")
	token := c.Query("token")

	if token != "" {
		// Validate JWT token
		claims, err := s.validateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user_id", claims.UserID)
	}
	// userID := c.GetString("user_id")

	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade connection"})
		return
	}
	defer conn.Close()

	// Create event channel
	eventChan := make(chan *state.CollaborationEvent, 100)

	// Start event subscription in a goroutine
	go func() {
		defer close(eventChan)
		err := s.stateManager.SubscribeToEvents(projectID, eventChan)
		if err != nil {
			fmt.Printf("Event subscription error: %v\n", err)
		}
	}()

	// Handle WebSocket communication
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return // Channel closed
			}

			// Send event to client
			if err := conn.WriteJSON(event); err != nil {
				fmt.Printf("WebSocket write error: %v\n", err)
				return
			}

		default:
			// Check for incoming messages from client
			conn.SetReadDeadline(time.Now().Add(time.Second))
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("WebSocket error: %v\n", err)
				}
				continue
			}

			// Handle client messages (ping, presence updates, etc.)
			var clientMsg map[string]interface{}
			if err := json.Unmarshal(message, &clientMsg); err == nil {
				if msgType, ok := clientMsg["type"].(string); ok && msgType == "ping" {
					// Respond to ping with pong
					conn.WriteJSON(map[string]string{"type": "pong"})
				}
			}
		}
	}
}

// Storage and system handlers

func (s *Server) getStorageStats(c *gin.Context) {
	stats := s.fileOps.GetStorageStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats":   stats,
	})
}

func (s *Server) validateAssetIntegrity(c *gin.Context) {
	projectID := c.Param("project")
	filePath := c.Query("file_path")

	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_path parameter required"})
		return
	}

	missingDeps, err := s.fileOps.ValidateAssetIntegrity(projectID, filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	isValid := len(missingDeps) == 0

	c.JSON(http.StatusOK, gin.H{
		"success":              true,
		"project":              projectID,
		"file_path":            filePath,
		"is_valid":             isValid,
		"missing_dependencies": missingDeps,
	})
}

func (s *Server) performCleanup(c *gin.Context) {
	cleanupType := c.Query("type")

	var err error
	var message string

	switch cleanupType {
	case "sessions":
		err = s.fileOps.CleanupExpiredSessions()
		message = "Cleaned up expired sessions and locks"
	case "storage":
		err = s.fileOps.CleanupStorage()
		message = "Cleaned up unused storage"
	case "all":
		err1 := s.fileOps.CleanupExpiredSessions()
		err2 := s.fileOps.CleanupStorage()
		if err1 != nil {
			err = err1
		} else if err2 != nil {
			err = err2
		}
		message = "Performed full cleanup"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cleanup type. Use: sessions, storage, or all"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// Commit recording handler
func (s *Server) recordCommit(c *gin.Context) {
	var req struct {
		Commit      analytics.Commit       `json:"commit"`
		FileChanges []analytics.FileChange `json:"file_changes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.fileOps.RecordCommit(&req.Commit, req.FileChanges)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"commit_hash":   req.Commit.Hash,
		"files_changed": len(req.FileChanges),
		"message":       "Commit recorded successfully",
	})
}

// Helper function to log file events
func (s *Server) logFileEvent(eventType, projectID, filePath, userID, userName string, metadata map[string]interface{}) {
	// Safe string conversion with nil check for session_id
	sessionID := ""
	if sid, ok := metadata["session_id"]; ok && sid != nil {
		if sidStr, ok := sid.(string); ok {
			sessionID = sidStr
		}
	}

	// Safe size conversion with nil check
	// var fileSize int64 = 0
	// if size, ok := metadata["size"]; ok && size != nil {
	// 	switch v := size.(type) {
	// 	case int64:
	// 		fileSize = v
	// 	case int:
	// 		fileSize = int64(v)
	// 	case float64:
	// 		fileSize = int64(v)
	// 	}
	// }

	// // Safe content hash conversion with nil check
	// contentHash := ""
	// if hash, ok := metadata["content_hash"]; ok && hash != nil {
	// 	if hashStr, ok := hash.(string); ok {
	// 		contentHash = hashStr
	// 	}
	// }

	// Create collaboration event for ClickHouse (with safe session_id)
	event := &analytics.CollaborationEvent{
		EventID:        fmt.Sprintf("%s_%d", eventType, time.Now().UnixNano()),
		EventType:      eventType,
		UserName:       userName,
		FilePath:       filePath,
		Project:        projectID,
		EventTime:      time.Now(),
		SessionID:      sessionID, // Now safe - won't be nil
		AdditionalData: make(map[string]string),
	}

	// Convert metadata to string map (with nil safety)
	for key, value := range metadata {
		if value == nil {
			event.AdditionalData[key] = ""
		} else if str, ok := value.(string); ok {
			event.AdditionalData[key] = str
		} else {
			event.AdditionalData[key] = fmt.Sprintf("%v", value)
		}
	}

	// Record in ClickHouse
	if err := s.analytics.RecordCollaborationEvent(event); err != nil {
		fmt.Printf("Failed to record file event in ClickHouse: %v\n", err)
	}

	// Create database file event for audit trail and database queries
	// dbEvent := models.FileEvent{
	// 	ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
	// 	ProjectID: projectID,
	// 	UserID:    userID,
	// 	EventType: eventType,
	// 	FilePath:  filePath,
	// 	Details: models.JSON{
	// 		"user_name":    userName,
	// 		"session_id":   sessionID,
	// 		"file_size":    fileSize,
	// 		"content_hash": contentHash,
	// 		"timestamp":    time.Now().Format(time.RFC3339),
	// 		"metadata":     metadata, // Store original metadata for reference
	// 	},
	// 	CreatedAt: time.Now(),
	// }

	// Store in database for audit trail
	// if err := s.db.Create(&dbEvent).Error; err != nil {
	// 	fmt.Printf("Failed to log file event to database: %v\n", err)
	// }

	// Debug logging (keep existing functionality)
	fmt.Printf("FILE_EVENT: %s - User: %s (%s), Project: %s, File: %s, Data: %+v\n",
		eventType, userName, userID, projectID, filePath, metadata)
}

func (s *Server) batchUploadFiles(c *gin.Context) {
	projectID := c.Query("project")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Parse batch upload request
	var req fileops.BatchUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	req.ProjectID = projectID
	req.UserID = userID

	// Process batch upload
	result, err := s.fileOps.ProcessObjectsBatch(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Store file metadata in database
	if err := s.storeFileMetadata(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to store file metadata: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"processed_objects":  result.ProcessedObjects,
		"skipped_objects":    result.SkippedObjects,
		"failed_objects":     result.FailedObjects,
		"total_size":         result.TotalSize,
		"duration_ms":        result.Duration.Milliseconds(),
		"analytics_recorded": result.AnalyticsRecorded,
	})
}

// storeFileMetadata stores file metadata in the database
func (s *Server) storeFileMetadata(req *fileops.BatchUploadRequest) error {
	fileRepo := database.NewFileRepository(s.db.DB)

	for filePath, contentHash := range req.FileMap {
		// Get object info for this file
		objectInfo, exists := req.Objects[contentHash]
		if !exists {
			continue // Skip if object info not found
		}

		// Create file record
		file := &models.File{
			ID:             fmt.Sprintf("file_%d", time.Now().UnixNano()),
			ProjectID:      req.ProjectID,
			Path:           filePath,
			ContentHash:    contentHash,
			Size:           objectInfo.Size,
			MimeType:       "application/octet-stream", // Default MIME type
			Branch:         "main",                     // Default branch
			LastModifiedBy: &req.UserID,
			LastModifiedAt: time.Now(),
		}

		// Store in database
		if err := fileRepo.CreateOrUpdateFile(file); err != nil {
			return fmt.Errorf("failed to store file %s: %w", filePath, err)
		}
	}

	return nil
}

// PHASE 1: New file existence check handler
func (s *Server) checkFileExists(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hash required"})
		return
	}
	// Check if file exists in storage
	exists := s.storage.Exists(hash)
	if !exists {
		c.Status(http.StatusInternalServerError)
		return
	}

	if exists {
		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusNotFound)
	}
}

// getProjectFiles retrieves all files for a project
func (s *Server) getProjectFiles(c *gin.Context) {
	projectID := c.Param("projectId")
	userID := c.GetString("user_id")

	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Check if user has access to the project
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

	// Get files for the project
	fileRepo := database.NewFileRepository(s.db.DB)
	files, err := fileRepo.GetProjectFiles(projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch project files",
		})
		return
	}

	// Format response
	fileList := make([]gin.H, len(files))
	for i, file := range files {
		fileList[i] = gin.H{
			"id":               file.ID,
			"path":             file.Path,
			"content_hash":     file.ContentHash,
			"size":             file.Size,
			"mime_type":        file.MimeType,
			"branch":           file.Branch,
			"is_locked":        file.IsLocked,
			"locked_by":        file.LockedBy,
			"locked_at":        file.LockedAt,
			"last_modified_by": file.LastModifiedBy,
			"last_modified_at": file.LastModifiedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"files":   fileList,
		"project": gin.H{
			"id":   project.ID,
			"name": project.Name,
		},
	})
}

func (s *Server) getProjectMembers(c *gin.Context) {
	projectID := c.Param("projectId")
	userID := c.GetString("user_id")
	userName := c.GetString("user_name")
	userEmail := c.GetString("user_email")

	// For now, return mock data
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"project_id": projectID,
		"members": []gin.H{
			{
				"id":        userID,
				"name":      userName,  // Real name from session
				"email":     userEmail, // Real email from session
				"role":      "owner",
				"status":    "online",
				"last_seen": time.Now().Format(time.RFC3339),
			},
		},
	})
}

// validateJWT validates a JWT token and returns the claims
func (s *Server) validateJWT(tokenString string) (*auth.JWTClaims, error) {
	// Get JWT secret from environment or use default
	jwtSecret := s.config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production"
	}

	authService := auth.NewAuthService(s.db.DB, jwtSecret)
	return authService.ValidateToken(tokenString)
}
