package fileops

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/Telerallc/gamedev-vcs/internal/analytics"
	"github.com/Telerallc/gamedev-vcs/internal/analyzer"
	"github.com/Telerallc/gamedev-vcs/internal/state"
	"github.com/Telerallc/gamedev-vcs/internal/storage"
)

type FilePathIndex struct {
	ProjectID string                    `json:"project_id"`
	Files     map[string]FileIndexEntry `json:"files"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

type FileIndexEntry struct {
	ContentHash  string    `json:"content_hash"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	CommitID     string    `json:"commit_id"`
	Branch       string    `json:"branch"`
}

type FileOperations struct {
	storage      *storage.ContentStore
	stateManager *state.StateManager
	analytics    *analytics.AnalyticsClient
	analyzer     *analyzer.UE5AssetAnalyzer

	// PHASE 1.5: Git-style components
	objectStore *storage.GitStyleObjectStore
	fileIndex   *storage.FileIndex
	commitStore *storage.GitStyleCommitStore // NEW
}

// UploadRequest represents a file upload request
type UploadRequest struct {
	ProjectID     string            `json:"project_id"`
	FilePath      string            `json:"file_path"`
	UserID        string            `json:"user_id"`
	UserName      string            `json:"user_name"`
	SessionID     string            `json:"session_id"`
	Content       io.Reader         `json:"-"`
	Metadata      map[string]string `json:"metadata"`
	CommitHash    string            `json:"commit_hash,omitempty"`
	CommitMessage string            `json:"commit_message,omitempty"`
}

// UploadResult represents the result of a file upload
type UploadResult struct {
	ContentHash       string                     `json:"content_hash"`
	Size              int64                      `json:"size"`
	FilePath          string                     `json:"file_path"`
	AssetInfo         *analyzer.AssetInfo        `json:"asset_info,omitempty"`
	Dependencies      []analyzer.AssetDependency `json:"dependencies,omitempty"`
	AnalyticsRecorded bool                       `json:"analytics_recorded"`
	LockStatus        *state.FileLock            `json:"lock_status,omitempty"`
}

// DownloadRequest represents a file download request
type DownloadRequest struct {
	ProjectID   string `json:"project_id"`
	FilePath    string `json:"file_path,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
	UserID      string `json:"user_id"`
	SessionID   string `json:"session_id"`
}

// DownloadResult represents the result of a file download
type DownloadResult struct {
	Content     io.ReadCloser      `json:"-"`
	ContentHash string             `json:"content_hash"`
	Size        int64              `json:"size"`
	Metadata    *storage.FileStats `json:"metadata"`
}

// ChunkUploadRequest represents a chunked upload request
type ChunkUploadRequest struct {
	ProjectID   string `json:"project_id"`
	FilePath    string `json:"file_path"`
	SessionID   string `json:"session_id"`
	ChunkIndex  int    `json:"chunk_index"`
	TotalChunks int    `json:"total_chunks"`
	ChunkData   []byte `json:"-"`
	UserID      string `json:"user_id"`
}

// LockRequest represents a file locking request
type LockRequest struct {
	ProjectID string `json:"project_id"`
	FilePath  string `json:"file_path"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	SessionID string `json:"session_id"`
}

type BatchUploadRequest struct {
	ProjectID     string                         `json:"project_id"`
	Objects       map[string]*storage.ObjectInfo `json:"objects"`
	FileMap       map[string]string              `json:"file_map"`
	UserID        string                         `json:"user_id"`
	UserName      string                         `json:"user_name"`
	SessionID     string                         `json:"session_id"`
	CommitHash    string                         `json:"commit_hash,omitempty"`
	CommitMessage string                         `json:"commit_message,omitempty"`
}

type BatchUploadResult struct {
	ProcessedObjects  int                            `json:"processed_objects"`
	ProcessedFiles    int                            `json:"processed_files"`
	SkippedObjects    int                            `json:"skipped_objects"`
	FailedObjects     int                            `json:"failed_objects"`
	TotalSize         int64                          `json:"total_size"`
	ObjectResults     map[string]*ObjectUploadResult `json:"object_results"`
	AnalyticsRecorded bool                           `json:"analytics_recorded"`
	Duration          time.Duration                  `json:"duration"`
}

type ObjectUploadResult struct {
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	Success    bool   `json:"success"`
	Error      error  `json:"error,omitempty"`
	Skipped    bool   `json:"skipped"`
	SkipReason string `json:"skip_reason,omitempty"`
}

// NewFileOperations creates a new file operations coordinator
func NewFileOperations(storage *storage.ContentStore, stateManager *state.StateManager, analytics *analytics.AnalyticsClient, objectStore *storage.GitStyleObjectStore, fileIndex *storage.FileIndex) *FileOperations {
	fo := &FileOperations{
		storage:      storage,
		stateManager: stateManager,
		analytics:    analytics,
		analyzer:     analyzer.NewUE5AssetAnalyzer(),
	}

	// Initialize Git-style components
	fo.objectStore = objectStore

	fo.fileIndex = fileIndex

	return fo
}

func (fo *FileOperations) SetCommitStore(commitStore *storage.GitStyleCommitStore) {
	fo.commitStore = commitStore
}

// NEW: Create Git-style commit
func (fo *FileOperations) CreateCommit(projectID, message, author, authorID, branch string, parents []string) (*storage.CommitResult, error) {
	if fo.commitStore == nil {
		return nil, fmt.Errorf("commit store not initialized")
	}

	commit := &storage.CommitObject{
		Author:    author,
		AuthorID:  authorID,
		Committer: author,
		Message:   message,
		ProjectID: projectID,
		Branch:    branch,
		Parents:   parents,
		Metadata:  make(map[string]string),
	}

	return fo.commitStore.CreateCommit(commit)
}

// UploadFile handles complete file upload with locking, analysis, and analytics
func (fo *FileOperations) UploadFile(req *UploadRequest) (*UploadResult, error) {
	// Check if file is locked by another user
	currentLock, err := fo.stateManager.GetFileLock(req.ProjectID, req.FilePath)
	if err == nil && currentLock.UserID != req.UserID {
		return nil, fmt.Errorf("file is locked by %s", currentLock.UserName)
	}

	// Read content for analysis
	content, err := io.ReadAll(req.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Store content in content-addressable storage
	fmt.Printf("DEBUG: About to store content for file: %s\n", req.FilePath)
	fileStats, err := fo.storage.Store(strings.NewReader(string(content)), req.Metadata)
	if err != nil {
		fmt.Printf("DEBUG: Storage error: %v\n", err)
		return nil, fmt.Errorf("failed to store content: %w", err)
	}

	if fileStats == nil {
		fmt.Printf("DEBUG: Storage returned nil file stats!\n")
		return nil, fmt.Errorf("storage returned nil file stats")
	}

	fmt.Printf("DEBUG: Storage successful, hash: %s, size: %d\n", fileStats.Hash, fileStats.Size)

	result := &UploadResult{
		ContentHash: fileStats.Hash,
		Size:        fileStats.Size,
		FilePath:    req.FilePath,
	}

	// Analyze asset if it's a UE5 file
	if fo.isUE5Asset(req.FilePath) {
		assetInfo, err := fo.analyzer.AnalyzeAsset(req.FilePath, content)
		if err == nil {
			result.AssetInfo = assetInfo
			result.Dependencies = assetInfo.Dependencies

			// Record asset dependencies in analytics
			if len(assetInfo.Dependencies) > 0 {
				analyticsDeps := fo.convertToAnalyticsDependencies(assetInfo.Dependencies, req.CommitHash)
				if err := fo.analytics.RecordAssetDependencies(analyticsDeps); err != nil {
					// Log error but don't fail the upload
					fmt.Printf("Failed to record dependencies: %v\n", err)
				}
			}
		}
	}

	// Record file change in analytics
	if req.CommitHash != "" {
		fileChange := &analytics.FileChange{
			CommitHash:     req.CommitHash,
			FilePath:       req.FilePath,
			ChangeType:     fo.determineChangeType(req.FilePath),
			ContentHash:    fileStats.Hash,
			FileSizeBytes:  uint64(fileStats.Size),
			Author:         req.UserName,
			CommitTime:     time.Now(),
			AssetType:      string(fo.getAssetType(req.FilePath)),
			SyncDurationMS: 0, // TODO: Track actual sync duration
		}

		if result.AssetInfo != nil {
			fileChange.AssetClass = result.AssetInfo.AssetClass
			fileChange.IsBlueprint = result.AssetInfo.IsBlueprint
			fileChange.BlueprintType = result.AssetInfo.BlueprintType
			fileChange.UE5PackagePath = result.AssetInfo.PackageName
		}

		if err := fo.analytics.RecordFileChanges([]analytics.FileChange{*fileChange}); err != nil {
			fmt.Printf("Failed to record file change: %v\n", err)
		} else {
			result.AnalyticsRecorded = true
		}
	}

	// Update user presence
	fo.stateManager.UpdatePresence(req.UserID, req.UserName, req.ProjectID, state.StatusEditing, req.FilePath)

	// Publish file modification event
	fo.stateManager.PublishEvent(&state.CollaborationEvent{
		EventID:   fmt.Sprintf("mod_%d", time.Now().UnixNano()),
		Type:      state.EventFileModified,
		UserID:    req.UserID,
		UserName:  req.UserName,
		ProjectID: req.ProjectID,
		FilePath:  req.FilePath,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"content_hash": fileStats.Hash,
			"file_size":    fileStats.Size,
		},
	})

	// Update file path index
	// if err := fo.updateFilePathIndex(req.ProjectID, req.FilePath, fileStats.Hash, fileStats.Size, req.CommitHash); err != nil {
	// 	fmt.Printf("Warning: failed to update file path index: %v\n", err)
	// }

	return result, nil
}

// DownloadFile handles file download with access tracking
func (fo *FileOperations) DownloadFile(req *DownloadRequest) (*DownloadResult, error) {
	var contentHash string

	if req.ContentHash != "" {
		contentHash = req.ContentHash
	} else if req.FilePath != "" {
		// Resolve file path to content hash
		// hash, err := fo.resolveFilePath(req.ProjectID, req.FilePath)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to resolve file path: %w", err)
		// }
		// contentHash = hash
	} else {
		return nil, fmt.Errorf("either content_hash or file_path must be provided")
	}

	// Get content from storage
	content, fileStats, err := fo.storage.Get(contentHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	// Update user presence if downloading a specific file
	if req.FilePath != "" {
		fo.stateManager.UpdatePresence(req.UserID, "", req.ProjectID, state.StatusOnline, req.FilePath)
	}

	return &DownloadResult{
		Content:     content,
		ContentHash: contentHash,
		Size:        fileStats.Size,
		Metadata:    fileStats,
	}, nil
}

// UploadChunk handles chunked file uploads for large assets
func (fo *FileOperations) UploadChunk(req *ChunkUploadRequest) error {
	// Check file lock
	currentLock, err := fo.stateManager.GetFileLock(req.ProjectID, req.FilePath)
	if err == nil && currentLock.UserID != req.UserID {
		return fmt.Errorf("file is locked by %s", currentLock.UserName)
	}

	// Store chunk
	_, err = fo.storage.StoreChunk(req.ChunkData, req.ChunkIndex, req.TotalChunks, req.SessionID)
	if err != nil {
		return fmt.Errorf("failed to store chunk: %w", err)
	}

	// Update user presence to show upload progress
	fo.stateManager.UpdatePresence(req.UserID, "", req.ProjectID, state.StatusEditing,
		fmt.Sprintf("%s (uploading %d/%d)", req.FilePath, req.ChunkIndex+1, req.TotalChunks))

	return nil
}

// FinalizeChunkedUpload assembles chunks into final file
func (fo *FileOperations) FinalizeChunkedUpload(sessionID string, totalChunks int, metadata map[string]string, req *UploadRequest) (*UploadResult, error) {
	// Assemble chunks
	fileStats, err := fo.storage.AssembleChunks(sessionID, totalChunks, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble chunks: %w", err)
	}

	// Get assembled content for analysis
	content, _, err := fo.storage.Get(fileStats.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get assembled content: %w", err)
	}
	defer content.Close()

	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read assembled content: %w", err)
	}

	result := &UploadResult{
		ContentHash: fileStats.Hash,
		Size:        fileStats.Size,
		FilePath:    req.FilePath,
	}

	// Analyze if UE5 asset
	if fo.isUE5Asset(req.FilePath) {
		assetInfo, err := fo.analyzer.AnalyzeAsset(req.FilePath, contentBytes)
		if err == nil {
			result.AssetInfo = assetInfo
			result.Dependencies = assetInfo.Dependencies
		}
	}

	// Record analytics
	if req.CommitHash != "" {
		fileChange := &analytics.FileChange{
			CommitHash:    req.CommitHash,
			FilePath:      req.FilePath,
			ChangeType:    fo.determineChangeType(req.FilePath),
			ContentHash:   fileStats.Hash,
			FileSizeBytes: uint64(fileStats.Size),
			Author:        req.UserName,
			CommitTime:    time.Now(),
			AssetType:     string(fo.getAssetType(req.FilePath)),
		}

		fo.analytics.RecordFileChanges([]analytics.FileChange{*fileChange})
		result.AnalyticsRecorded = true
	}

	// Update presence
	fo.stateManager.UpdatePresence(req.UserID, req.UserName, req.ProjectID, state.StatusOnline, "")

	return result, nil
}

// LockFile acquires an exclusive lock on a file
func (fo *FileOperations) LockFile(req *LockRequest) (*state.FileLock, error) {
	lock, err := fo.stateManager.LockFile(req.ProjectID, req.FilePath, req.UserID, req.UserName, req.SessionID)
	if err != nil {
		return nil, err
	}

	// Update user presence
	fo.stateManager.UpdatePresence(req.UserID, req.UserName, req.ProjectID, state.StatusEditing, req.FilePath)

	return lock, nil
}

// UnlockFile releases a file lock
func (fo *FileOperations) UnlockFile(projectID, filePath, userID string) error {
	err := fo.stateManager.UnlockFile(projectID, filePath, userID)
	if err != nil {
		return err
	}

	// Update presence
	fo.stateManager.UpdatePresence(userID, "", projectID, state.StatusOnline, "")

	return nil
}

// GetFileLock retrieves current lock information for a file
func (fo *FileOperations) GetFileLock(projectID, filePath string) (*state.FileLock, error) {
	return fo.stateManager.GetFileLock(projectID, filePath)
}

// ListProjectLocks returns all active locks for a project
func (fo *FileOperations) ListProjectLocks(projectID string) ([]state.FileLock, error) {
	return fo.stateManager.ListProjectLocks(projectID)
}

// GetProjectPresence returns active users in a project
func (fo *FileOperations) GetProjectPresence(projectID string) ([]state.UserPresence, error) {
	return fo.stateManager.GetProjectPresence(projectID)
}

// ValidateAssetIntegrity checks if all dependencies are available
func (fo *FileOperations) ValidateAssetIntegrity(projectID, filePath string) ([]string, error) {
	// Get asset content hash (simplified - would need path to hash mapping)
	// For now, return empty slice
	return []string{}, nil
}

// GetAssetDependencies retrieves dependency information for an asset
func (fo *FileOperations) GetAssetDependencies(assetPath string) ([]analytics.AssetDependency, error) {
	return fo.analytics.GetAssetDependencies(assetPath)
}

// GetDependencyImpact finds what assets depend on the given asset
func (fo *FileOperations) GetDependencyImpact(assetPath string) ([]analytics.AssetDependency, error) {
	return fo.analytics.GetDependencyImpact(assetPath)
}

// RecordCommit records a commit with all associated file changes
func (fo *FileOperations) RecordCommit(commit *analytics.Commit, fileChanges []analytics.FileChange) error {
	// Record commit
	if err := fo.analytics.RecordCommit(commit); err != nil {
		return fmt.Errorf("failed to record commit: %w", err)
	}

	// Record file changes
	if len(fileChanges) > 0 {
		if err := fo.analytics.RecordFileChanges(fileChanges); err != nil {
			return fmt.Errorf("failed to record file changes: %w", err)
		}
	}

	// Publish commit event
	fo.stateManager.PublishEvent(&state.CollaborationEvent{
		EventID:   fmt.Sprintf("commit_%s", commit.Hash),
		Type:      state.EventCommitCreated,
		UserID:    "", // Would need to be passed in
		UserName:  commit.Author,
		ProjectID: commit.Project,
		Timestamp: commit.CommitTime,
		Data: map[string]interface{}{
			"commit_hash":    commit.Hash,
			"commit_message": commit.Message,
			"files_changed":  len(fileChanges),
		},
	})

	return nil
}

// Helper methods

func (fo *FileOperations) isUE5Asset(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	ue5Extensions := []string{".uasset", ".umap", ".uexp", ".ubulk"}

	for _, ue5Ext := range ue5Extensions {
		if ext == ue5Ext {
			return true
		}
	}

	return false
}

func (fo *FileOperations) getAssetType(filePath string) analyzer.AssetType {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".umap":
		return analyzer.AssetTypeLevel
	case ".uasset":
		return analyzer.AssetTypeUnknown // Would need content analysis
	case ".fbx":
		return analyzer.AssetTypeStaticMesh
	case ".png", ".jpg", ".jpeg", ".tga":
		return analyzer.AssetTypeTexture2D
	case ".wav", ".mp3":
		return analyzer.AssetTypeSound
	default:
		return analyzer.AssetTypeUnknown
	}
}

func (fo *FileOperations) determineChangeType(filePath string) string {
	// Simplified - in a real implementation, this would compare with previous version
	return "modified"
}

func (fo *FileOperations) convertToAnalyticsDependencies(deps []analyzer.AssetDependency, commitHash string) []analytics.AssetDependency {
	var analyticsDeps []analytics.AssetDependency

	for _, dep := range deps {
		analyticsDep := analytics.AssetDependency{
			AssetPath:        dep.SourceAsset,
			DependsOnPath:    dep.TargetAsset,
			DependencyType:   string(dep.DependencyType),
			DiscoveredTime:   time.Now(),
			CommitHash:       commitHash,
			IsCircular:       dep.IsCircular,
			DependencyWeight: float32(dep.Weight),
		}
		analyticsDeps = append(analyticsDeps, analyticsDep)
	}

	return analyticsDeps
}

// Cleanup operations

// CleanupExpiredSessions removes expired user sessions and locks
func (fo *FileOperations) CleanupExpiredSessions() error {
	return fo.stateManager.CleanupExpiredLocks()
}

// CleanupStorage removes unused content from storage
func (fo *FileOperations) CleanupStorage() error {
	return fo.storage.Cleanup()
}

func (fo *FileOperations) GetStorageStats() map[string]interface{} {
	stats := fo.storage.GetStorageStats()

	// Add Git-style storage stats if available
	if fo.objectStore != nil {
		if gitStats, err := fo.objectStore.GetStats(); err == nil {
			stats["git_style_objects"] = gitStats["total_objects"]
			stats["git_style_compression"] = gitStats["compression_ratio"]
			stats["git_style_space_saved"] = gitStats["space_saved"]
		}
	}

	if fo.fileIndex != nil {
		indexStats := fo.fileIndex.GetStats()
		stats["index_entries"] = indexStats["total_entries"]
		stats["staged_entries"] = indexStats["staged_entries"]
	}

	return stats
}

// func (fo *FileOperations) getFilePathIndex(projectID string) (*FilePathIndex, error) {
// 	// Try to load from storage first
// 	indexPath := filepath.Join("indexes", projectID, "file_index.json")

// 	if data, err := os.ReadFile(indexPath); err == nil {
// 		var index FilePathIndex
// 		if err := json.Unmarshal(data, &index); err == nil {
// 			return &index, nil
// 		}
// 	}

// 	// Build index from database if not found
// 	return fo.buildFilePathIndex(projectID)
// }

// func (fo *FileOperations) buildFilePathIndex(projectID string) (*FilePathIndex, error) {
// 	// This would query your database for the current file state
// 	// For now, we'll create a basic implementation

// 	index := &FilePathIndex{
// 		ProjectID: projectID,
// 		Files:     make(map[string]FileIndexEntry),
// 		UpdatedAt: time.Now(),
// 	}

// 	// Query database for current files (you'd implement this with your DB)
// 	// For now, return empty index
// 	return index, nil
// }

// func (fo *FileOperations) updateFilePathIndex(projectID, filePath, contentHash string, size int64, commitID string) error {
// 	index, err := fo.getFilePathIndex(projectID)
// 	if err != nil {
// 		index = &FilePathIndex{
// 			ProjectID: projectID,
// 			Files:     make(map[string]FileIndexEntry),
// 		}
// 	}

// 	index.Files[filePath] = FileIndexEntry{
// 		ContentHash:  contentHash,
// 		Size:         size,
// 		LastModified: time.Now(),
// 		CommitID:     commitID,
// 		Branch:       "main", // You'd determine this from context
// 	}
// 	index.UpdatedAt = time.Now()

// 	return fo.saveFilePathIndex(index)
// }

// func (fo *FileOperations) saveFilePathIndex(index *FilePathIndex) error {
// 	indexDir := filepath.Join("indexes", index.ProjectID)
// 	if err := os.MkdirAll(indexDir, 0755); err != nil {
// 		return err
// 	}

// 	data, err := json.MarshalIndent(index, "", "  ")
// 	if err != nil {
// 		return err
// 	}

// 	indexPath := filepath.Join(indexDir, "file_index.json")
// 	return os.WriteFile(indexPath, data, 0644)
// }

// func (fo *FileOperations) resolveFilePath(projectID, filePath string) (string, error) {
// 	index, err := fo.getFilePathIndex(projectID)
// 	if err != nil {
// 		return "", err
// 	}

// 	entry, exists := index.Files[filePath]
// 	if !exists {
// 		return "", fmt.Errorf("file not found: %s", filePath)
// 	}

// 	return entry.ContentHash, nil
// }

// New
func (fo *FileOperations) ProcessObjectsBatch(req *BatchUploadRequest) (*BatchUploadResult, error) {
	start := time.Now()

	result := &BatchUploadResult{
		ObjectResults: make(map[string]*ObjectUploadResult),
	}

	// Process each object using existing storage
	for hash, objectInfo := range req.Objects {
		objResult := &ObjectUploadResult{
			Hash: hash,
			Size: objectInfo.Size,
		}

		// Check if object exists in main storage
		exists := fo.storage.Exists(hash)
		if exists {
			objResult.Success = true
			objResult.Skipped = true
			objResult.SkipReason = "object already exists"
			result.SkippedObjects++
		} else {
			objResult.Success = true
			result.ProcessedObjects++
			result.TotalSize += objectInfo.Size
		}

		result.ObjectResults[hash] = objResult
	}

	// Store file metadata in database
	if err := fo.storeFileMetadata(req); err != nil {
		return nil, fmt.Errorf("failed to store file metadata: %w", err)
	}

	// Record batch analytics (single event for entire batch)
	if result.ProcessedObjects > 0 {
		fo.recordBatchAnalytics(req, result)
		result.AnalyticsRecorded = true
	}

	result.ProcessedFiles = len(req.FileMap)
	result.Duration = time.Since(start)

	return result, nil
}

// storeFileMetadata stores file metadata in the database
func (fo *FileOperations) storeFileMetadata(req *BatchUploadRequest) error {
	// We need to access the database through the server
	// For now, we'll skip database storage and just log the files
	// The actual database storage should be handled by the server handler

	fmt.Printf("📝 Would store %d files in database for project %s\n", len(req.FileMap), req.ProjectID)
	for filePath, contentHash := range req.FileMap {
		// Get object info for this file
		objectInfo, exists := req.Objects[contentHash]
		if !exists {
			continue // Skip if object info not found
		}

		fmt.Printf("  - %s (hash: %s, size: %d)\n", filePath, contentHash, objectInfo.Size)
	}

	return nil
}

// recordBatchAnalytics records analytics for entire batch (not per file)
func (fo *FileOperations) recordBatchAnalytics(req *BatchUploadRequest, result *BatchUploadResult) {
	if fo.analytics == nil {
		return
	}

	// Create a generic batch event using existing event structure
	fo.stateManager.PublishEvent(&state.CollaborationEvent{
		EventID:   fmt.Sprintf("batch_%s", req.SessionID),
		Type:      state.EventFileModified, // Reuse existing event type
		UserID:    req.UserID,
		UserName:  req.UserName,
		ProjectID: req.ProjectID,
		FilePath:  fmt.Sprintf("batch_%d_files", len(req.FileMap)), // Aggregate path
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"total_objects":     len(req.Objects),
			"processed_objects": result.ProcessedObjects,
			"skipped_objects":   result.SkippedObjects,
			"failed_objects":    result.FailedObjects,
			"total_files":       len(req.FileMap),
			"total_size":        result.TotalSize,
			"duration_ms":       result.Duration.Milliseconds(),
			"batch_id":          req.SessionID,
		},
	})
}
