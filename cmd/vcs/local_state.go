package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type FileState struct {
	Path        string    `json:"path"`
	ContentHash string    `json:"content_hash"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Staged      bool      `json:"staged"`
	Added       bool      `json:"added"`
	Modified    bool      `json:"modified"`
	Deleted     bool      `json:"deleted"`
	LastUpdated time.Time `json:"last_updated"`
}

// LocalState represents the local VCS state
type LocalState struct {
	ProjectID     string               `json:"project_id"`
	CurrentBranch string               `json:"current_branch"`
	LocalCommits  []string             `json:"local_commits"`
	RemoteCommits []string             `json:"remote_commits"`
	StagedFiles   map[string]FileState `json:"staged_files"`
	LastSync      string               `json:"last_sync"`
	LocalRefs     map[string]string    `json:"local_refs"`
	Version       int                  `json:"version"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// LoadLocalState loads the local VCS state from .vcs/state.json
func LoadLocalState() (*LocalState, error) {
	statePath := ".vcs/state.json"

	// Check if state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		// Return new empty state if file doesn't exist
		return &LocalState{
			StagedFiles:   make(map[string]FileState),
			LocalCommits:  []string{},
			RemoteCommits: []string{},
			LocalRefs:     make(map[string]string),
			CurrentBranch: "main",
			Version:       1,
			UpdatedAt:     time.Now(),
		}, nil
	}

	// Read existing state file
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state LocalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Initialize maps if they're nil (for older state files)
	if state.StagedFiles == nil {
		state.StagedFiles = make(map[string]FileState)
	}
	if state.LocalRefs == nil {
		state.LocalRefs = make(map[string]string)
	}
	if state.LocalCommits == nil {
		state.LocalCommits = []string{}
	}
	if state.RemoteCommits == nil {
		state.RemoteCommits = []string{}
	}

	return &state, nil
}

// SaveLocalState saves the local VCS state to .vcs/state.json
func (ls *LocalState) SaveLocalState() error {
	// Ensure .vcs directory exists
	if err := os.MkdirAll(".vcs", 0755); err != nil {
		return fmt.Errorf("failed to create .vcs directory: %w", err)
	}

	// Update timestamp
	ls.UpdatedAt = time.Now()

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(ls, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first, then rename for atomic write
	statePath := ".vcs/state.json"
	tempPath := statePath + ".tmp"

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, statePath); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return fmt.Errorf("failed to finalize state file: %w", err)
	}

	return nil
}

func SaveLocalState(state *LocalState) error {
	return state.SaveLocalState()
}

// AddStagedFile adds a file to the staging area
func (ls *LocalState) AddStagedFile(filePath string) {
	if ls.StagedFiles == nil {
		ls.StagedFiles = make(map[string]FileState)
	}

	// Get current file info
	var fileState FileState
	if existing, exists := ls.StagedFiles[filePath]; exists {
		fileState = existing
	} else {
		fileState = FileState{
			Path: filePath,
		}
	}

	// Update file state
	fileState.Staged = true
	fileState.Added = true
	fileState.LastUpdated = time.Now()

	// Try to get file stats if file exists
	if info, err := os.Stat(filePath); err == nil {
		fileState.Size = info.Size()
		fileState.ModTime = info.ModTime()
	}

	ls.StagedFiles[filePath] = fileState
}

func (ls *LocalState) AddLocalCommit(commitID string) {
	// Add to beginning of slice (most recent first)
	ls.LocalCommits = append([]string{commitID}, ls.LocalCommits...)

	// Keep only last 100 commits to avoid bloat
	if len(ls.LocalCommits) > 100 {
		ls.LocalCommits = ls.LocalCommits[:100]
	}
}

// UpdateRemoteCommits updates the remote commit list
func (ls *LocalState) UpdateRemoteCommits(remoteCommits []string) {
	ls.RemoteCommits = remoteCommits
}

func (ls *LocalState) SetBranchHead(branch, commitID string) {
	if ls.LocalRefs == nil {
		ls.LocalRefs = make(map[string]string)
	}
	ls.LocalRefs["refs/heads/"+branch] = commitID

	// Update current branch if it matches
	if ls.CurrentBranch == branch {
		ls.LocalRefs["HEAD"] = commitID
	}
}

// GetBranchHead gets the HEAD commit for a branch
func (ls *LocalState) GetBranchHead(branch string) string {
	if ls.LocalRefs == nil {
		return ""
	}
	return ls.LocalRefs["refs/heads/"+branch]
}

// RemoveStagedFile removes a file from the staging area
func (ls *LocalState) RemoveStagedFile(filePath string) {
	delete(ls.StagedFiles, filePath)
}

// ClearStagedFiles clears all staged files
func (ls *LocalState) ClearStagedFiles() {
	ls.StagedFiles = make(map[string]FileState)
}

// GetStagedFiles returns all staged files
func (ls *LocalState) GetStagedFiles() []string {
	var files []string
	for filePath := range ls.StagedFiles {
		files = append(files, filePath)
	}
	return files
}

// IsFileStaged checks if a file is staged
func (ls *LocalState) IsFileStaged(filePath string) bool {
	_, exists := ls.StagedFiles[filePath]
	return exists
}

func (ls *LocalState) IsFileDirty(filePath string) bool {
	fileState, exists := ls.StagedFiles[filePath]
	if !exists {
		return false
	}
	return fileState.Modified || fileState.Added || fileState.Deleted
}

// GetCommitsSinceRemote returns local commits that are not in remote
func (ls *LocalState) GetCommitsSinceRemote() []string {
	remoteSet := make(map[string]bool)
	for _, commit := range ls.RemoteCommits {
		remoteSet[commit] = true
	}

	var localOnly []string
	for _, commit := range ls.LocalCommits {
		if !remoteSet[commit] {
			localOnly = append(localOnly, commit)
		}
	}

	return localOnly
}

// GetRemoteCommitsNotLocal returns remote commits that are not in local
func (ls *LocalState) GetRemoteCommitsNotLocal() []string {
	localSet := make(map[string]bool)
	for _, commit := range ls.LocalCommits {
		localSet[commit] = true
	}

	var remoteOnly []string
	for _, commit := range ls.RemoteCommits {
		if !localSet[commit] {
			remoteOnly = append(remoteOnly, commit)
		}
	}

	return remoteOnly
}

// SyncStatus returns the sync status between local and remote
func (ls *LocalState) SyncStatus() (ahead int, behind int, status string) {
	aheadCommits := ls.GetCommitsSinceRemote()
	behindCommits := ls.GetRemoteCommitsNotLocal()

	ahead = len(aheadCommits)
	behind = len(behindCommits)

	if ahead == 0 && behind == 0 {
		status = "up-to-date"
	} else if ahead > 0 && behind > 0 {
		status = "diverged"
	} else if ahead > 0 {
		status = "ahead"
	} else {
		status = "behind"
	}

	return ahead, behind, status
}

func (ls *LocalState) GetStats() map[string]interface{} {
	stagedCount := 0
	modifiedCount := 0
	addedCount := 0
	deletedCount := 0

	for _, fileState := range ls.StagedFiles {
		if fileState.Staged {
			stagedCount++
		}
		if fileState.Modified {
			modifiedCount++
		}
		if fileState.Added {
			addedCount++
		}
		if fileState.Deleted {
			deletedCount++
		}
	}

	return map[string]interface{}{
		"project_id":     ls.ProjectID,
		"current_branch": ls.CurrentBranch,
		"staged_files":   stagedCount,
		"modified_files": modifiedCount,
		"added_files":    addedCount,
		"deleted_files":  deletedCount,
		"local_commits":  len(ls.LocalCommits),
		"remote_commits": len(ls.RemoteCommits),
		"last_sync":      ls.LastSync,
		"version":        ls.Version,
	}
}
