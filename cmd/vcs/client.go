package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	fileops "github.com/Telerallc/gamedev-vcs/internal/fileOps"
	"github.com/Telerallc/gamedev-vcs/internal/storage"
	"github.com/gorilla/websocket"
)

// APIClient handles communication with the VCS server
type APIClient struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
	sessionID  string

	// Phase 1: Git-style components
	objectStore *storage.GitStyleObjectStore
	fileIndex   *storage.FileIndex
}

// FileUploadResponse represents the server response for file uploads
type FileUploadResponse struct {
	Success           bool                     `json:"success"`
	ContentHash       string                   `json:"content_hash"`
	Size              int64                    `json:"size"`
	FilePath          string                   `json:"file_path"`
	AnalyticsRecorded bool                     `json:"analytics_recorded"`
	AssetInfo         map[string]interface{}   `json:"asset_info,omitempty"`
	Dependencies      []map[string]interface{} `json:"dependencies,omitempty"`
}

type BatchUploadResult struct {
	TotalFiles     int                            `json:"total_files"`
	ProcessedFiles int                            `json:"processed_files"`
	SkippedFiles   int                            `json:"skipped_files"`
	FailedFiles    int                            `json:"failed_files"`
	Results        []FileUploadResult             `json:"results"`
	Duration       time.Duration                  `json:"duration"`
	ObjectsStored  map[string]*storage.ObjectInfo `json:"objects_stored"`
}

// LockResponse represents the server response for file locking
type LockResponse struct {
	Success  bool                   `json:"success"`
	Locked   bool                   `json:"locked"`
	LockInfo map[string]interface{} `json:"lock_info,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// PresenceInfo represents user presence information
type PresenceInfo struct {
	UserID      string    `json:"user_id"`
	UserName    string    `json:"user_name"`
	ProjectID   string    `json:"project_id"`
	LastSeen    time.Time `json:"last_seen"`
	Status      string    `json:"status"`
	CurrentFile string    `json:"current_file,omitempty"`
}

// TeamStatus represents team activity status
type TeamStatus struct {
	Success  bool           `json:"success"`
	Project  string         `json:"project"`
	Presence []PresenceInfo `json:"presence"`
	Locks    []LockInfo     `json:"locks,omitempty"`
}

// LockInfo represents file lock information
type LockInfo struct {
	FilePath  string    `json:"file_path"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	LockedAt  time.Time `json:"locked_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type FileCache struct {
	mu       sync.RWMutex
	hashes   map[string]string    // filepath -> hash
	sizes    map[string]int64     // filepath -> size
	modTimes map[string]time.Time // filepath -> modification time
}

// FileUploadResult represents result of parallel upload
type FileUploadResult struct {
	FilePath    string
	Success     bool
	Error       error
	ContentHash string
	Size        int64
	Duration    time.Duration
	Skipped     bool // true if file was already uploaded
}

func NewFileCache() *FileCache {
	return &FileCache{
		hashes:   make(map[string]string),
		sizes:    make(map[string]int64),
		modTimes: make(map[string]time.Time),
	}
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) (*APIClient, error) {
	client := &APIClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Initialize Git-style object store
	objectStore, err := storage.NewGitStyleObjectStore(".vcs/objects")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize object store: %w", err)
	}
	client.objectStore = objectStore

	// Initialize file index
	fileIndex, err := storage.NewFileIndex(".vcs/index")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize file index: %w", err)
	}
	client.fileIndex = fileIndex

	return client, nil
}

// SetAuth sets the authentication token
func (c *APIClient) SetAuth(token, sessionID string) {
	c.authToken = token
	c.sessionID = sessionID
}

func (c *APIClient) Login(username, password string) error {
	loginData := map[string]string{
		"username": username,
		"password": password,
	}

	resp, err := c.makeRequest("POST", "/api/v1/auth/login", loginData)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}

	var loginResp struct {
		Success   bool   `json:"success"`
		Token     string `json:"token"`
		SessionID string `json:"session_id"`
		ExpiresAt int64  `json:"expires_at"`
		User      struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			Name     string `json:"name"`
		} `json:"user"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(resp, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	if !loginResp.Success {
		return fmt.Errorf("login failed: %s", loginResp.Error)
	}

	c.SetAuth(loginResp.Token, loginResp.SessionID)

	if verbose {
		fmt.Printf("Logged in as: %s (%s)\n", loginResp.User.Name, loginResp.User.Username)
	}

	return nil
}

// UploadFile uploads a file to the server
func (c *APIClient) UploadFile(projectID, filePath string, content io.Reader) (*FileUploadResponse, error) {
	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, content); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add metadata
	writer.WriteField("file_path", filePath)
	writer.WriteField("user_name", "CLI User")
	writer.WriteField("session_id", c.sessionID)

	writer.Close()

	// Create request with project ID as query parameter
	url := fmt.Sprintf("%s/api/v1/files/upload?project=%s", c.baseURL, projectID)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var uploadResp FileUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &uploadResp, nil
}

// DownloadFile downloads a file from the server
func (c *APIClient) DownloadFile(contentHash, projectID string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/api/v1/files/%s?project_id=%s", c.baseURL, contentHash, projectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// LockFile locks a file for exclusive editing
func (c *APIClient) LockFile(projectID, filePath string) (*LockResponse, error) {
	lockData := map[string]string{
		"user_name":  "CLI User",
		"session_id": c.sessionID,
	}

	url := fmt.Sprintf("/api/v1/locks/%s/%s", projectID, filePath)

	resp, err := c.makeRequest("POST", url, lockData)
	if err != nil {
		return nil, fmt.Errorf("lock request failed: %w", err)
	}

	var lockResp LockResponse
	if err := json.Unmarshal(resp, &lockResp); err != nil {
		return nil, fmt.Errorf("failed to parse lock response: %w", err)
	}

	return &lockResp, nil
}

// UnlockFile unlocks a file
func (c *APIClient) UnlockFile(projectID, filePath string) error {
	url := fmt.Sprintf("/api/v1/locks/%s/%s", projectID, filePath)

	_, err := c.makeRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("unlock request failed: %w", err)
	}

	return nil
}

// GetTeamStatus gets current team activity and file locks
func (c *APIClient) GetTeamStatus(projectID string) (*TeamStatus, error) {
	// Get presence
	presenceResp, err := c.makeRequest("GET", fmt.Sprintf("/api/v1/collaboration/%s/presence", projectID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get presence: %w", err)
	}

	// Get locks
	locksResp, err := c.makeRequest("GET", fmt.Sprintf("/api/v1/locks/%s", projectID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get locks: %w", err)
	}

	var presence struct {
		Success  bool           `json:"success"`
		Presence []PresenceInfo `json:"presence"`
	}

	var locks struct {
		Success bool       `json:"success"`
		Locks   []LockInfo `json:"locks"`
	}

	if err := json.Unmarshal(presenceResp, &presence); err != nil {
		return nil, fmt.Errorf("failed to parse presence response: %w", err)
	}

	if err := json.Unmarshal(locksResp, &locks); err != nil {
		return nil, fmt.Errorf("failed to parse locks response: %w", err)
	}

	return &TeamStatus{
		Success:  true,
		Project:  projectID,
		Presence: presence.Presence,
		Locks:    locks.Locks,
	}, nil
}

// UpdatePresence updates user presence information
func (c *APIClient) UpdatePresence(projectID, status, currentFile string) error {
	presenceData := map[string]string{
		"user_name":    "CLI User",
		"status":       status,
		"current_file": currentFile,
	}

	_, err := c.makeRequest("POST", fmt.Sprintf("/api/v1/collaboration/%s/presence", projectID), presenceData)
	if err != nil {
		return fmt.Errorf("presence update failed: %w", err)
	}

	return nil
}

// GetProductivityMetrics gets team productivity analytics
func (c *APIClient) GetProductivityMetrics(projectID string, days int) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/analytics/productivity/%s?days=%d", projectID, days)
	return c.makeRequest("GET", url, nil)
}

// GetActivityFeed gets recent team activity
func (c *APIClient) GetActivityFeed(projectID string, limit int) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/analytics/activity/%s?limit=%d", projectID, limit)
	return c.makeRequest("GET", url, nil)
}

// GetDependencyGraph gets asset dependency information
func (c *APIClient) GetDependencyGraph(projectID, assetPath string) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/analytics/dependencies/%s?asset_path=%s", projectID, assetPath)
	return c.makeRequest("GET", url, nil)
}

// SubscribeToEvents creates a WebSocket connection for real-time events
func (c *APIClient) SubscribeToEvents(projectID string, eventHandler func(map[string]interface{})) error {
	wsURL := strings.Replace(c.baseURL, "http", "ws", 1) + "/api/v1/collaboration/ws?project_id=" + projectID

	headers := http.Header{}
	if c.authToken != "" {
		headers.Set("Authorization", "Bearer "+c.authToken)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	// Send periodic pings
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
				return
			}
		}
	}()

	// Read messages
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return fmt.Errorf("WebSocket error: %w", err)
			}
			break
		}

		// Handle the event
		if eventHandler != nil {
			eventHandler(message)
		}
	}

	return nil
}

// CreateCommit creates a new commit on the server
func (c *APIClient) CreateCommit(projectID, message, branch string, filePaths []string, parentCommits []string) ([]byte, error) {
	commitData := map[string]interface{}{
		"message":        message,
		"branch":         branch,
		"file_paths":     filePaths,
		"parent_commits": parentCommits,
	}

	url := fmt.Sprintf("/api/v1/commits/%s", projectID)
	return c.makeRequest("POST", url, commitData)
}

// GetCommitHistory retrieves commit history for a project/branch
func (c *APIClient) GetCommitHistory(projectID, branch string, limit int) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/commits/%s?branch=%s&limit=%d", projectID, branch, limit)
	return c.makeRequest("GET", url, nil)
}

// GetCommit retrieves a specific commit by ID
func (c *APIClient) GetCommit(projectID, commitID string) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/commits/%s/%s", projectID, commitID)
	return c.makeRequest("GET", url, nil)
}

// GetFileHistory retrieves the history of a specific file
func (c *APIClient) GetFileHistory(projectID, filePath string, limit int) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/commits/%s/files/%s?limit=%d", projectID, filePath, limit)
	return c.makeRequest("GET", url, nil)
}

// DiffCommits compares two commits
func (c *APIClient) DiffCommits(projectID, fromCommit, toCommit string) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/commits/%s/diff?from=%s&to=%s", projectID, fromCommit, toCommit)
	return c.makeRequest("GET", url, nil)
}

// PushChanges pushes local changes to the server
func (c *APIClient) PushChanges(projectID, branch string, localCommits, remoteCommits []string, files []string) ([]byte, error) {
	pushData := map[string]interface{}{
		"branch":         branch,
		"local_commits":  localCommits,
		"remote_commits": remoteCommits,
		"files":          files,
	}

	url := fmt.Sprintf("/api/v1/sync/%s/push", projectID)
	return c.makeRequest("POST", url, pushData)
}

// PullChanges pulls remote changes from the server
func (c *APIClient) PullChanges(projectID, branch string, localCommits, remoteCommits []string) ([]byte, error) {
	pullData := map[string]interface{}{
		"branch":         branch,
		"local_commits":  localCommits,
		"remote_commits": remoteCommits,
	}

	url := fmt.Sprintf("/api/v1/sync/%s/pull", projectID)
	return c.makeRequest("POST", url, pullData)
}

// GetSyncStatus gets the sync status between local and remote
func (c *APIClient) GetSyncStatus(projectID, branch string, localCommits []string) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/sync/%s/status?branch=%s", projectID, branch)

	if len(localCommits) > 0 {
		url += "&local_commits=" + strings.Join(localCommits, ",")
	}

	return c.makeRequest("GET", url, nil)
}

// GetBranchInfo gets information about a specific branch
func (c *APIClient) GetBranchInfo(projectID, branch string) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/sync/%s/branches/%s", projectID, branch)
	return c.makeRequest("GET", url, nil)
}

// ListBranches lists all branches for a project
func (c *APIClient) ListBranches(projectID string) ([]byte, error) {
	url := fmt.Sprintf("/api/v1/branches/%s", projectID)
	return c.makeRequest("GET", url, nil)
}

// CreateBranch creates a new branch
func (c *APIClient) CreateBranch(projectID, name, fromCommit, fromBranch string, isProtected bool) ([]byte, error) {
	branchData := map[string]interface{}{
		"name":         name,
		"from_commit":  fromCommit,
		"from_branch":  fromBranch,
		"is_protected": isProtected,
	}

	url := fmt.Sprintf("/api/v1/branches/%s", projectID)
	return c.makeRequest("POST", url, branchData)
}

// DeleteBranch deletes a branch
func (c *APIClient) DeleteBranch(projectID, branchName string, force bool) error {
	url := fmt.Sprintf("/api/v1/branches/%s/%s", projectID, branchName)
	if force {
		url += "?force=true"
	}

	_, err := c.makeRequest("DELETE", url, nil)
	return err
}

// UpdateBranch updates branch settings (default, protection)
func (c *APIClient) UpdateBranch(projectID, branchName string, isDefault, isProtected *bool) ([]byte, error) {
	updateData := make(map[string]interface{})
	if isDefault != nil {
		updateData["is_default"] = *isDefault
	}
	if isProtected != nil {
		updateData["is_protected"] = *isProtected
	}

	url := fmt.Sprintf("/api/v1/branches/%s/%s", projectID, branchName)
	return c.makeRequest("PATCH", url, updateData)
}

// RecordCommit records a commit with the server (legacy analytics method)
func (c *APIClient) RecordCommit(commit map[string]interface{}, fileChanges []map[string]interface{}) error {
	commitData := map[string]interface{}{
		"commit":       commit,
		"file_changes": fileChanges,
	}

	_, err := c.makeRequest("POST", "/api/v1/analytics/commits", commitData)
	if err != nil {
		return fmt.Errorf("commit recording failed: %w", err)
	}

	return nil
}

// GetStorageStats gets storage utilization statistics
func (c *APIClient) GetStorageStats() ([]byte, error) {
	return c.makeRequest("GET", "/api/v1/system/storage/stats", nil)
}

// PerformCleanup triggers server-side cleanup operations
func (c *APIClient) PerformCleanup(cleanupType string) error {
	url := fmt.Sprintf("/api/v1/system/cleanup?type=%s", cleanupType)
	_, err := c.makeRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("cleanup request failed: %w", err)
	}

	return nil
}

// Helper method to make HTTP requests
func (c *APIClient) makeRequest(method, endpoint string, data interface{}) ([]byte, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// Utility functions for file operations

// IsUE5Asset checks if a file is a UE5 asset
func IsUE5Asset(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	ue5Extensions := []string{".uasset", ".umap", ".uexp", ".ubulk"}

	for _, ue5Ext := range ue5Extensions {
		if ext == ue5Ext {
			return true
		}
	}

	return false
}

// IsBinaryAsset checks if a file should be treated as binary
func IsBinaryAsset(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	binaryExtensions := []string{
		".uasset", ".umap", ".uexp", ".ubulk", // UE5
		".fbx", ".obj", ".dae", // 3D models
		".png", ".jpg", ".jpeg", ".tga", ".bmp", ".gif", // Images
		".wav", ".mp3", ".ogg", ".flac", // Audio
		".mp4", ".avi", ".mov", // Video
		".exe", ".dll", ".so", ".dylib", // Executables
		".zip", ".rar", ".7z", // Archives
	}

	for _, binExt := range binaryExtensions {
		if ext == binExt {
			return true
		}
	}

	return false
}

// GetFileSize returns the size of a file
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ShouldUseChunkedUpload determines if a file should be uploaded in chunks
func ShouldUseChunkedUpload(filePath string, threshold int64) bool {
	size, err := GetFileSize(filePath)
	if err != nil {
		return false
	}
	return size > threshold
}

func (fc *FileCache) LoadFromFile(cacheFile string) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		// Cache doesn't exist yet, initialize empty maps
		fc.hashes = make(map[string]string)
		fc.sizes = make(map[string]int64)
		fc.modTimes = make(map[string]time.Time)
		return nil
	}

	// Read cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	// Parse JSON
	var cache struct {
		Hashes   map[string]string    `json:"hashes"`
		Sizes    map[string]int64     `json:"sizes"`
		ModTimes map[string]time.Time `json:"mod_times"`
	}

	if err := json.Unmarshal(data, &cache); err != nil {
		// If cache is corrupted, start fresh
		fc.hashes = make(map[string]string)
		fc.sizes = make(map[string]int64)
		fc.modTimes = make(map[string]time.Time)
		return nil
	}

	// Initialize maps if nil
	if cache.Hashes == nil {
		cache.Hashes = make(map[string]string)
	}
	if cache.Sizes == nil {
		cache.Sizes = make(map[string]int64)
	}
	if cache.ModTimes == nil {
		cache.ModTimes = make(map[string]time.Time)
	}

	fc.hashes = cache.Hashes
	fc.sizes = cache.Sizes
	fc.modTimes = cache.ModTimes

	return nil
}

func (fc *FileCache) SaveToFile(cacheFile string) error {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	// Ensure .vcs directory exists
	dir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Prepare cache data
	cache := struct {
		Hashes   map[string]string    `json:"hashes"`
		Sizes    map[string]int64     `json:"sizes"`
		ModTimes map[string]time.Time `json:"mod_times"`
		SavedAt  time.Time            `json:"saved_at"`
	}{
		Hashes:   fc.hashes,
		Sizes:    fc.sizes,
		ModTimes: fc.modTimes,
		SavedAt:  time.Now(),
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	// Write to temporary file first, then rename (atomic operation)
	tempFile := cacheFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary cache file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, cacheFile); err != nil {
		os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to finalize cache file: %w", err)
	}

	return nil
}

// CalculateFileHash calculates SHA256 hash of file
func CalculateFileHash(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hasher := sha256.New()
	written, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, err
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash, written, nil
}

// ParallelUploadFiles uploads multiple files concurrently
func (c *APIClient) ParallelUploadFiles(projectID string, filePaths []string, maxConcurrency int) ([]FileUploadResult, error) {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}

	// Load cache
	cache := NewFileCache()
	cacheFile := ".vcs/file_cache.json"
	cache.LoadFromFile(cacheFile)

	// Channels for work distribution
	workChan := make(chan string, len(filePaths))
	resultChan := make(chan FileUploadResult, len(filePaths))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range workChan {
				result := c.uploadSingleFileOptimized(projectID, filePath, cache)
				resultChan <- result
			}
		}()
	}

	// Send work
	for _, filePath := range filePaths {
		workChan <- filePath
	}
	close(workChan)

	// Wait and collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []FileUploadResult
	for result := range resultChan {
		results = append(results, result)
	}

	// Save cache
	cache.SaveToFile(cacheFile)
	return results, nil
}

func (c *APIClient) uploadSingleFileOptimized(projectID, filePath string, cache *FileCache) FileUploadResult {
	start := time.Now()

	result := FileUploadResult{
		FilePath: filePath,
		Duration: time.Since(start),
	}

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		result.Error = fmt.Errorf("file not found: %s", filePath)
		return result
	}

	// Check if file changed since last cache
	fileChanged, err := cache.IsFileChanged(filePath)
	if err != nil {
		// If we can't determine, assume changed
		fileChanged = true
	}

	var hash string
	var size int64

	if !fileChanged {
		// Use cached hash
		if cachedHash, exists := cache.GetHash(filePath); exists {
			hash = cachedHash
			size = cache.sizes[filePath]

			// Quick check if server has this file
			if hasFile, err := c.CheckServerHasFile(hash); err == nil && hasFile {
				result.Success = true
				result.Skipped = true
				result.ContentHash = hash
				result.Size = size
				result.Duration = time.Since(start)
				return result
			}
		}
	}

	// File changed or not in cache, calculate new hash
	if hash == "" {
		hash, size, err = CalculateFileHash(filePath)
		if err != nil {
			result.Error = fmt.Errorf("failed to calculate hash: %w", err)
			return result
		}

		// Update cache
		cache.SetHash(filePath, hash, size, fileInfo.ModTime())

		// Check if server already has this content
		if hasFile, err := c.CheckServerHasFile(hash); err == nil && hasFile {
			result.Success = true
			result.Skipped = true
			result.ContentHash = hash
			result.Size = size
			result.Duration = time.Since(start)
			return result
		}
	}

	// Need to upload file
	file, err := os.Open(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to open file: %w", err)
		return result
	}
	defer file.Close()

	// Upload with pre-calculated hash
	uploadResp, err := c.UploadFile(projectID, filePath, file)
	if err != nil {
		result.Error = fmt.Errorf("upload failed: %w", err)
		return result
	}

	result.Success = uploadResp.Success
	result.ContentHash = uploadResp.ContentHash
	result.Size = uploadResp.Size
	result.Duration = time.Since(start)
	return result
}

func (fc *FileCache) GetHash(filePath string) (string, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	hash, exists := fc.hashes[filePath]
	return hash, exists
}

// SetHash sets hash, size and modification time for a file
func (fc *FileCache) SetHash(filePath, hash string, size int64, modTime time.Time) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.hashes[filePath] = hash
	fc.sizes[filePath] = size
	fc.modTimes[filePath] = modTime
}

// IsFileChanged checks if file has changed since last cache
func (fc *FileCache) IsFileChanged(filePath string) (bool, error) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	// Get current file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return true, err // File doesn't exist or error, consider changed
	}

	// Check if we have cached info
	cachedModTime, hasModTime := fc.modTimes[filePath]
	cachedSize, hasSize := fc.sizes[filePath]

	if !hasModTime || !hasSize {
		return true, nil // No cached info, consider changed
	}

	// Compare modification time and size
	currentModTime := fileInfo.ModTime()
	currentSize := fileInfo.Size()

	// File changed if either mod time or size changed
	return !currentModTime.Equal(cachedModTime) || currentSize != cachedSize, nil
}

// Clean removes entries for files that no longer exist
func (fc *FileCache) Clean() error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	var toDelete []string

	for filePath := range fc.hashes {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			toDelete = append(toDelete, filePath)
		}
	}

	// Remove non-existent files from cache
	for _, filePath := range toDelete {
		delete(fc.hashes, filePath)
		delete(fc.sizes, filePath)
		delete(fc.modTimes, filePath)
	}

	return nil
}

// GetStats returns cache statistics
func (fc *FileCache) GetStats() map[string]int {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return map[string]int{
		"total_files": len(fc.hashes),
		"hash_count":  len(fc.hashes),
		"size_count":  len(fc.sizes),
		"time_count":  len(fc.modTimes),
	}
}

func (c *APIClient) ProcessFilesBatchGitStyle(projectID string, filePaths []string) (*BatchUploadResult, error) {
	return c.processFilesBatchGitStyleInternal(projectID, filePaths, false)
}

func (c *APIClient) ProcessFilesBatchGitStyleForCommit(projectID string, filePaths []string) (*BatchUploadResult, error) {
	return c.processFilesBatchGitStyleInternal(projectID, filePaths, true)
}

func (c *APIClient) processFilesBatchGitStyleInternal(projectID string, filePaths []string, forceProcess bool) (*BatchUploadResult, error) {
	start := time.Now()

	result := &BatchUploadResult{
		TotalFiles:    len(filePaths),
		Results:       make([]FileUploadResult, 0, len(filePaths)),
		ObjectsStored: make(map[string]*storage.ObjectInfo),
	}

	var changedFiles []string
	var err error

	if forceProcess {
		fmt.Printf("🔄 Processing %d files for commit (forced)...\n", len(filePaths))
		// For commit, process all files regardless of stat optimization
		changedFiles = filePaths
	} else {
		fmt.Printf("🔍 Phase 1: Checking %d files for changes using stat optimization...\n", len(filePaths))

		// STEP 1: Batch stat-based change detection
		changedFiles, err = c.fileIndex.GetChangedFiles(filePaths)
		if err != nil {
			return nil, fmt.Errorf("failed to detect changes: %w", err)
		}

		skippedCount := len(filePaths) - len(changedFiles)
		if skippedCount > 0 {
			fmt.Printf("⏭️  Skipped %d unchanged files (stat optimization)\n", skippedCount)
		}

		if len(changedFiles) == 0 {
			fmt.Printf("✅ All files are up to date!\n")
			result.SkippedFiles = len(filePaths)
			result.Duration = time.Since(start)
			return result, nil
		}

		fmt.Printf("📝 Processing %d changed files...\n", len(changedFiles))
	}

	// STEP 2: Calculate hashes and store objects locally
	fileToHash := make(map[string]string)
	hashToFile := make(map[string][]string) // Multiple files can have same content

	for _, filePath := range changedFiles {
		fileResult := FileUploadResult{
			FilePath: filePath,
		}

		// Calculate hash and store in object store
		file, err := os.Open(filePath)
		if err != nil {
			fileResult.Error = fmt.Errorf("failed to open file: %w", err)
			result.Results = append(result.Results, fileResult)
			result.FailedFiles++
			continue
		}

		objectInfo, err := c.objectStore.Store(file, map[string]string{
			"source_path": filePath,
			"stored_at":   time.Now().Format(time.RFC3339),
		})
		file.Close()

		if err != nil {
			fileResult.Error = fmt.Errorf("failed to store object: %w", err)
			result.Results = append(result.Results, fileResult)
			result.FailedFiles++
			continue
		}

		fileToHash[filePath] = objectInfo.Hash
		hashToFile[objectInfo.Hash] = append(hashToFile[objectInfo.Hash], filePath)
		result.ObjectsStored[objectInfo.Hash] = objectInfo

		fileResult.Success = true
		fileResult.ContentHash = objectInfo.Hash
		fileResult.Size = objectInfo.Size
		result.Results = append(result.Results, fileResult)
	}

	// STEP 3: Check which objects already exist on server (batch)
	hashes := make([]string, 0, len(result.ObjectsStored))
	for hash := range result.ObjectsStored {
		hashes = append(hashes, hash)
	}

	existingObjects, err := c.batchCheckObjectsExist(hashes)
	if err != nil {
		fmt.Printf("⚠️  Failed to check existing objects: %v\n", err)
		// Continue anyway - will upload all objects
		existingObjects = make(map[string]bool)
	}

	// STEP 4: Upload only new objects (batch)
	newObjects := make(map[string]*storage.ObjectInfo)
	for hash, objectInfo := range result.ObjectsStored {
		if !existingObjects[hash] {
			newObjects[hash] = objectInfo
		}
	}

	if len(newObjects) > 0 {
		fmt.Printf("📤 Uploading %d new objects...\n", len(newObjects))
		err := c.batchUploadObjects(projectID, newObjects)
		if err != nil {
			return nil, fmt.Errorf("failed to upload objects: %w", err)
		}
	} else {
		fmt.Printf("✅ All objects already exist on server\n 🔄 Skipping upload\n")
	}

	// STEP 5: Update file index with new hashes
	indexUpdates := make(map[string]string)
	for filePath, hash := range fileToHash {
		indexUpdates[filePath] = hash
	}

	if err := c.fileIndex.BatchUpdateEntries(indexUpdates); err != nil {
		fmt.Printf("⚠️  Failed to update file index: %v\n", err)
	}

	// STEP 6: Save index to disk
	if err := c.fileIndex.Save(); err != nil {
		fmt.Printf("⚠️  Failed to save file index: %v\n", err)
	}

	// Update results
	result.ProcessedFiles = len(changedFiles)
	result.SkippedFiles = len(filePaths) - len(changedFiles)
	result.Duration = time.Since(start)

	fmt.Printf("✅ Batch processing completed: %d processed, %d skipped in %v\n",
		result.ProcessedFiles, result.SkippedFiles, result.Duration)

	return result, nil
}

func (c *APIClient) batchCheckObjectsExist(hashes []string) (map[string]bool, error) {
	if len(hashes) == 0 {
		return make(map[string]bool), nil
	}

	// For now, check each hash individually (Phase 2 will optimize this)
	results := make(map[string]bool, len(hashes))

	for _, hash := range hashes {
		exists, err := c.CheckServerHasFile(hash)
		if err != nil {
			// Assume doesn't exist on error
			results[hash] = false
		} else {
			results[hash] = exists
		}
	}

	return results, nil
}

func (c *APIClient) batchUploadObjects(projectID string, objects map[string]*storage.ObjectInfo) error {
	// Create batch upload request
	req := fileops.BatchUploadRequest{
		ProjectID: projectID,
		Objects:   objects,
		FileMap:   make(map[string]string), // We'll populate this from the file index
		UserID:    "cli-user",              // TODO: Get from auth
		UserName:  "CLI User",
		SessionID: c.sessionID,
	}

	// Get file paths for each hash from the file index
	stagedEntries := c.fileIndex.GetStagedEntries()
	for hash := range objects {
		// Find files that have this hash
		for filePath, entry := range stagedEntries {
			if entry.Hash == hash {
				req.FileMap[filePath] = hash
			}
		}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal batch request: %w", err)
	}

	// Make batch upload request
	url := fmt.Sprintf("%s/api/v1/files/batch-upload?project=%s", c.baseURL, projectID)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create batch upload request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("batch upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("batch upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// CheckServerHasFile checks if server has an object by hash
func (c *APIClient) CheckServerHasFile(contentHash string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/files/exists/%s", c.baseURL, contentHash)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err // Assume we need to upload on error
	}
	defer resp.Body.Close()

	// Server returns 200 if exists, 404 if not
	return resp.StatusCode == 200, nil
}

func (c *APIClient) GetIndexStats() map[string]interface{} {
	if c.fileIndex == nil {
		return map[string]interface{}{"error": "index not initialized"}
	}
	return c.fileIndex.GetStats()
}

// GetObjectStoreStats returns statistics about the local object store
func (c *APIClient) GetObjectStoreStats() (map[string]interface{}, error) {
	if c.objectStore == nil {
		return nil, fmt.Errorf("object store not initialized")
	}
	return c.objectStore.GetStats()
}

// CleanupLocalStorage cleans up unused local objects and index entries
func (c *APIClient) CleanupLocalStorage() error {
	if c.fileIndex != nil {
		if err := c.fileIndex.Clean(); err != nil {
			return fmt.Errorf("failed to clean index: %w", err)
		}
	}

	if c.objectStore != nil {
		if err := c.objectStore.Cleanup(); err != nil {
			return fmt.Errorf("failed to clean object store: %w", err)
		}
	}

	return nil
}
