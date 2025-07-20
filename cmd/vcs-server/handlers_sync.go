package main

import (
	"net/http"
	"strings"

	"github.com/Telerallc/gamedev-vcs/database"
	"github.com/Telerallc/gamedev-vcs/internal/version"
	"github.com/Telerallc/gamedev-vcs/models"
	"github.com/gin-gonic/gin"
)

// PushRequest represents a push request from the client
type PushRequest struct {
	Branch        string   `json:"branch"`
	LocalCommits  []string `json:"local_commits"`
	RemoteCommits []string `json:"remote_commits"`
	Files         []string `json:"files,omitempty"` // Optional: push specific files only
}

// PushResponse represents the server response to a push
type PushResponse struct {
	Success       bool     `json:"success"`
	Updated       bool     `json:"updated"`
	NewCommits    []string `json:"new_commits,omitempty"`
	Conflicts     []string `json:"conflicts,omitempty"`
	RequiredPull  bool     `json:"required_pull"`
	RemoteCommits []string `json:"remote_commits,omitempty"`
}

// PullRequest represents a pull request from the client
type PullRequest struct {
	Branch        string   `json:"branch"`
	LocalCommits  []string `json:"local_commits"`
	RemoteCommits []string `json:"remote_commits"`
}

// PullResponse represents the server response to a pull
type PullResponse struct {
	Success    bool                 `json:"success"`
	Updated    bool                 `json:"updated"`
	NewCommits []models.Commit      `json:"new_commits,omitempty"`
	Files      []models.FileVersion `json:"files,omitempty"`
	HeadCommit string               `json:"head_commit"`
	Conflicts  []string             `json:"conflicts,omitempty"`
}

// pushChanges handles pushing local changes to the server
func (s *Server) pushChanges(c *gin.Context) {
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

	var req PushRequest
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

	commitService := version.NewCommitService(s.db.DB)

	// Get current remote commits for the branch
	remoteCommits, err := commitService.GetCommitHistory(projectID, req.Branch, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get remote commits"})
		return
	}

	// Check for conflicts - if there are remote commits not in local commits
	var conflictCommits []string
	remoteCommitIDs := make(map[string]bool)
	for _, commit := range remoteCommits {
		remoteCommitIDs[commit.ID] = true
	}

	localCommitIDs := make(map[string]bool)
	for _, commitID := range req.LocalCommits {
		localCommitIDs[commitID] = true
	}

	// Find commits that exist remotely but not locally
	for _, commit := range remoteCommits {
		if !localCommitIDs[commit.ID] {
			conflictCommits = append(conflictCommits, commit.ID)
		}
	}

	// If there are conflicts, require pull first
	if len(conflictCommits) > 0 {
		c.JSON(http.StatusConflict, PushResponse{
			Success:       false,
			RequiredPull:  true,
			RemoteCommits: conflictCommits,
			Conflicts:     conflictCommits,
		})
		return
	}

	// Find new commits to push (commits in local but not in remote)
	var newCommits []string
	for _, commitID := range req.LocalCommits {
		if !remoteCommitIDs[commitID] {
			newCommits = append(newCommits, commitID)
		}
	}

	if len(newCommits) == 0 {
		c.JSON(http.StatusOK, PushResponse{
			Success: true,
			Updated: false,
		})
		return
	}

	// Update the branch HEAD to the latest local commit
	if len(req.LocalCommits) > 0 {
		latestCommit := req.LocalCommits[0] // Assuming first commit is the latest
		err = commitService.UpdateBranchHead(projectID, req.Branch, latestCommit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update branch"})
			return
		}
	}

	// Log the push event
	s.logFileEvent("branch_pushed", projectID, req.Branch, userID, c.GetString("user_name"), map[string]interface{}{
		"branch":      req.Branch,
		"new_commits": len(newCommits),
	})

	c.JSON(http.StatusOK, PushResponse{
		Success:    true,
		Updated:    true,
		NewCommits: newCommits,
	})
}

// pullChanges handles pulling remote changes to the client
func (s *Server) pullChanges(c *gin.Context) {
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

	var req PullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Default branch if not specified
	if req.Branch == "" {
		req.Branch = project.DefaultBranch
	}

	commitService := version.NewCommitService(s.db.DB)

	// Get remote commits for the branch
	remoteCommits, err := commitService.GetCommitHistory(projectID, req.Branch, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get remote commits"})
		return
	}

	// Find new commits (remote commits not in local)
	localCommitIDs := make(map[string]bool)
	for _, commitID := range req.LocalCommits {
		localCommitIDs[commitID] = true
	}

	var newCommits []models.Commit
	var headCommit string

	if len(remoteCommits) > 0 {
		headCommit = remoteCommits[0].ID
	}

	for _, commit := range remoteCommits {
		if !localCommitIDs[commit.ID] {
			newCommits = append(newCommits, commit)
		}
	}

	if len(newCommits) == 0 {
		c.JSON(http.StatusOK, PullResponse{
			Success:    true,
			Updated:    false,
			HeadCommit: headCommit,
		})
		return
	}

	// Get files for the new commits
	var files []models.FileVersion
	for _, commit := range newCommits {
		commitWithTree, err := commitService.GetCommitByID(commit.ID)
		if err != nil {
			continue
		}

		// Get commit tree
		var tree models.CommitTree
		if err := s.db.Where("id = ?", commitWithTree.TreeHash).First(&tree).Error; err == nil {
			// Get file versions for this commit
			for _, treeFile := range tree.Files {
				fileVersion, err := commitService.GetFileAtCommit(commit.ID, treeFile.Path)
				if err != nil {
					continue
				}
				files = append(files, *fileVersion)
			}
		}
	}

	// Log the pull event
	s.logFileEvent("branch_pulled", projectID, req.Branch, userID, c.GetString("user_name"), map[string]interface{}{
		"branch":      req.Branch,
		"new_commits": len(newCommits),
	})

	c.JSON(http.StatusOK, PullResponse{
		Success:    true,
		Updated:    true,
		NewCommits: newCommits,
		Files:      files,
		HeadCommit: headCommit,
	})
}

// getBranchInfo returns information about a specific branch
func (s *Server) getBranchInfo(c *gin.Context) {
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

	// Get branch information
	var branch models.Branch
	err = s.db.Where("project_id = ? AND name = ?", projectID, branchName).First(&branch).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "branch not found"})
		return
	}

	// Get recent commits
	commitService := version.NewCommitService(s.db.DB)
	commits, err := commitService.GetCommitHistory(projectID, branchName, 10)
	if err != nil {
		commits = []models.Commit{} // Empty if error
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"branch":      branch,
		"commits":     commits,
		"head_commit": branch.LastCommit,
	})
}

// syncStatus returns the sync status between local and remote
func (s *Server) syncStatus(c *gin.Context) {
	projectID := c.Param("project")
	branch := c.DefaultQuery("branch", "main")
	localCommitsParam := c.Query("local_commits")

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
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	if !project.HasPermission(userID, "read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	// Parse local commits
	var localCommits []string
	if localCommitsParam != "" {
		localCommits = strings.Split(localCommitsParam, ",")
	}

	// Get remote commits
	commitService := version.NewCommitService(s.db.DB)
	remoteCommits, err := commitService.GetCommitHistory(projectID, branch, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get remote commits"})
		return
	}

	// Analyze sync status
	localCommitIDs := make(map[string]bool)
	for _, commitID := range localCommits {
		localCommitIDs[commitID] = true
	}

	var behind []string // Remote commits not in local
	var ahead []string  // Local commits not in remote

	remoteCommitIDs := make(map[string]bool)
	for _, commit := range remoteCommits {
		remoteCommitIDs[commit.ID] = true
		if !localCommitIDs[commit.ID] {
			behind = append(behind, commit.ID)
		}
	}

	for _, commitID := range localCommits {
		if !remoteCommitIDs[commitID] {
			ahead = append(ahead, commitID)
		}
	}

	status := "up-to-date"
	if len(behind) > 0 && len(ahead) > 0 {
		status = "diverged"
	} else if len(behind) > 0 {
		status = "behind"
	} else if len(ahead) > 0 {
		status = "ahead"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"status":         status,
		"ahead":          len(ahead),
		"behind":         len(behind),
		"ahead_commits":  ahead,
		"behind_commits": behind,
		"branch":         branch,
		"remote_head": func() string {
			if len(remoteCommits) > 0 {
				return remoteCommits[0].ID
			}
			return ""
		}(),
	})
}
