package storage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ContentStore manages content-addressable storage with deduplication
type ContentStore struct {
	basePath     string
	tempPath     string
	statsCache   map[string]*FileStats
	statsMutex   sync.RWMutex
	statsFile    string
	indexFile    string
	lastSaveTime time.Time
}

// FileStats contains metadata about stored content
type FileStats struct {
	Hash         string    `json:"hash"`
	Size         int64     `json:"size"`
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
	RefCount     int       `json:"ref_count"`
	ContentType  string    `json:"content_type"`
}

// ChunkInfo represents a chunk of a larger file
type ChunkInfo struct {
	Index     int    `json:"index"`
	Hash      string `json:"hash"`
	Size      int64  `json:"size"`
	TotalSize int64  `json:"total_size"`
}

type PersistentStats struct {
	Version   string                 `json:"version"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Stats     map[string]*FileStats  `json:"stats"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NewContentStore creates a new content-addressable storage instance
func NewContentStore(basePath string) (*ContentStore, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	tempPath := filepath.Join(basePath, "tmp")
	if err := os.MkdirAll(tempPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create object directory structure
	objPath := filepath.Join(basePath, "objects")
	if err := os.MkdirAll(objPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create objects directory: %w", err)
	}

	// Create metadata directory
	metaPath := filepath.Join(basePath, "metadata")
	if err := os.MkdirAll(metaPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	cs := &ContentStore{
		basePath:   basePath,
		tempPath:   tempPath,
		statsCache: make(map[string]*FileStats),
		statsFile:  filepath.Join(metaPath, "file_stats.json"),
		indexFile:  filepath.Join(metaPath, "content_index.json"),
	}

	// Load existing file stats and index
	if err := cs.loadStats(); err != nil {
		// Log warning but don't fail - we can rebuild stats
		fmt.Printf("Warning: failed to load file stats: %v\n", err)
	}

	// Start background stats persistence
	go cs.backgroundStatsPersistence()

	return cs, nil
}

// Store saves content and returns its hash, enabling deduplication
func (cs *ContentStore) Store(reader io.Reader, metadata map[string]string) (*FileStats, error) {
	// Create temporary file for atomic operation
	tempFile, err := os.CreateTemp(cs.tempPath, "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Hash while writing to temp file
	hasher := sha256.New()
	multiWriter := io.MultiWriter(hasher, tempFile)

	size, err := io.Copy(multiWriter, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	if size == 0 {
		return nil, fmt.Errorf("cannot store empty content")
	}

	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	finalPath := cs.getContentPath(hash)

	fmt.Printf("DEBUG: Storing content with hash: %s, size: %d, path: %s\n", hash, size, finalPath)

	// Check if content already exists (deduplication)
	if _, err := os.Stat(finalPath); err == nil {
		fmt.Printf("DEBUG: Content already exists, updating stats\n")
		cs.updateStats(hash, size, metadata)
		stats := cs.getStats(hash)
		if stats == nil {
			return nil, fmt.Errorf("failed to get stats for existing content")
		}
		return stats, nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create content directory: %w", err)
	}

	// Atomic move from temp to final location
	if err := os.Rename(tempFile.Name(), finalPath); err != nil {
		return nil, fmt.Errorf("failed to move content to final location: %w", err)
	}

	// Create and cache stats
	stats := &FileStats{
		Hash:         hash,
		Size:         size,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		RefCount:     1,
		ContentType:  detectContentType(metadata),
	}

	fmt.Printf("DEBUG: Created stats: %+v\n", stats)

	cs.setStats(hash, stats)
	return stats, nil
}

// Get retrieves content by hash
func (cs *ContentStore) Get(hash string) (io.ReadCloser, *FileStats, error) {
	if !cs.isValidHash(hash) {
		return nil, nil, fmt.Errorf("invalid hash format: %s", hash)
	}

	contentPath := cs.getContentPath(hash)
	file, err := os.Open(contentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("content not found: %s", hash)
		}
		return nil, nil, fmt.Errorf("failed to open content: %w", err)
	}

	// Update access time
	stats := cs.getStats(hash)
	if stats != nil {
		stats.LastAccessed = time.Now()
		cs.setStats(hash, stats)
	}

	return file, stats, nil
}

// Exists checks if content with given hash exists
func (cs *ContentStore) Exists(hash string) bool {
	if !cs.isValidHash(hash) {
		return false
	}

	contentPath := cs.getContentPath(hash)
	_, err := os.Stat(contentPath)
	return err == nil
}

// ObjectExists is an alias for Exists for compatibility
func (cs *ContentStore) ObjectExists(hash string) bool {
	return cs.Exists(hash)
}

// GetStats returns metadata for stored content
func (cs *ContentStore) GetStats(hash string) *FileStats {
	return cs.getStats(hash)
}

// GetObjectInfo returns object information for compatibility with GitStyleObjectStore
func (cs *ContentStore) GetObjectInfo(hash string) (*ObjectInfo, error) {
	if !cs.isValidHash(hash) {
		return nil, fmt.Errorf("invalid hash format: %s", hash)
	}

	stats := cs.getStats(hash)
	if stats == nil {
		return nil, fmt.Errorf("object not found: %s", hash)
	}

	contentPath := cs.getContentPath(hash)
	fileInfo, err := os.Stat(contentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	return &ObjectInfo{
		Hash:           hash,
		Size:           stats.Size,
		CompressedSize: fileInfo.Size(),
		StoredAt:       stats.CreatedAt,
		ObjectPath:     contentPath,
	}, nil
}

// StoreChunk stores a chunk of a larger file
func (cs *ContentStore) StoreChunk(chunkData []byte, chunkIndex int, totalChunks int, sessionID string) (*ChunkInfo, error) {
	// Calculate chunk hash
	hasher := sha256.New()
	hasher.Write(chunkData)
	chunkHash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Store chunk in temporary location with session ID
	chunkPath := filepath.Join(cs.tempPath, "chunks", sessionID, fmt.Sprintf("chunk-%d", chunkIndex))
	if err := os.MkdirAll(filepath.Dir(chunkPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunk directory: %w", err)
	}

	if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write chunk: %w", err)
	}

	return &ChunkInfo{
		Index: chunkIndex,
		Hash:  chunkHash,
		Size:  int64(len(chunkData)),
	}, nil
}

// AssembleChunks combines chunks into final content
func (cs *ContentStore) AssembleChunks(sessionID string, totalChunks int, metadata map[string]string) (*FileStats, error) {
	chunksDir := filepath.Join(cs.tempPath, "chunks", sessionID)

	// Create temporary file for assembly
	tempFile, err := os.CreateTemp(cs.tempPath, "assembled-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create assembly temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(hasher, tempFile)

	var totalSize int64

	// Read and concatenate chunks in order
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(chunksDir, fmt.Sprintf("chunk-%d", i))
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk %d: %w", i, err)
		}

		if _, err := multiWriter.Write(chunkData); err != nil {
			return nil, fmt.Errorf("failed to write chunk %d to assembly: %w", i, err)
		}

		totalSize += int64(len(chunkData))
	}

	// Calculate final hash
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	finalPath := cs.getContentPath(hash)

	// Check for existing content
	if _, err := os.Stat(finalPath); err == nil {
		// Content already exists, clean up chunks
		os.RemoveAll(chunksDir)
		cs.updateStats(hash, totalSize, metadata)
		return cs.getStats(hash), nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create content directory: %w", err)
	}

	// Move assembled file to final location
	if err := os.Rename(tempFile.Name(), finalPath); err != nil {
		return nil, fmt.Errorf("failed to move assembled content: %w", err)
	}

	// Clean up chunks
	os.RemoveAll(chunksDir)

	// Create stats
	stats := &FileStats{
		Hash:         hash,
		Size:         totalSize,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		RefCount:     1,
		ContentType:  detectContentType(metadata),
	}

	cs.setStats(hash, stats)
	return stats, nil
}

// Delete removes content if no longer referenced
func (cs *ContentStore) Delete(hash string) error {
	stats := cs.getStats(hash)
	if stats == nil {
		return fmt.Errorf("content not found: %s", hash)
	}

	stats.RefCount--
	if stats.RefCount <= 0 {
		contentPath := cs.getContentPath(hash)
		if err := os.Remove(contentPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete content: %w", err)
		}
		cs.deleteStats(hash)
	} else {
		cs.setStats(hash, stats)
	}

	return nil
}

// ListContent returns all stored content hashes
func (cs *ContentStore) ListContent() []string {
	cs.statsMutex.RLock()
	defer cs.statsMutex.RUnlock()

	hashes := make([]string, 0, len(cs.statsCache))
	for hash := range cs.statsCache {
		hashes = append(hashes, hash)
	}
	return hashes
}

// Cleanup removes temporary files and unused content
func (cs *ContentStore) Cleanup() error {
	// Clean up old temporary files
	tempEntries, err := os.ReadDir(cs.tempPath)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-24 * time.Hour) // Remove temp files older than 24h

	for _, entry := range tempEntries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(cs.tempPath, entry.Name()))
		}
	}

	return nil
}

// Private helper methods

func (cs *ContentStore) getContentPath(hash string) string {
	// Split hash for filesystem efficiency: ab/cd/abcd1234...
	return filepath.Join(cs.basePath, "objects", hash[:2], hash[2:4], hash)
}

func (cs *ContentStore) isValidHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}

	for _, char := range hash {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}

	return true
}

func (cs *ContentStore) getStats(hash string) *FileStats {
	cs.statsMutex.RLock()
	defer cs.statsMutex.RUnlock()
	return cs.statsCache[hash]
}

func (cs *ContentStore) deleteStats(hash string) {
	cs.statsMutex.Lock()
	defer cs.statsMutex.Unlock()
	delete(cs.statsCache, hash)
}

func (cs *ContentStore) updateStats(hash string, size int64, metadata map[string]string) {
	stats := cs.getStats(hash)
	if stats != nil {
		stats.RefCount++
		stats.LastAccessed = time.Now()
		cs.setStats(hash, stats)
	}
}

func (cs *ContentStore) loadStats() error {
	cs.statsMutex.Lock()
	defer cs.statsMutex.Unlock()

	// Try to load from JSON file
	if data, err := os.ReadFile(cs.statsFile); err == nil {
		var persistent PersistentStats
		if err := json.Unmarshal(data, &persistent); err == nil {
			cs.statsCache = persistent.Stats
			if cs.statsCache == nil {
				cs.statsCache = make(map[string]*FileStats)
			}
			fmt.Printf("Loaded %d file stats from persistent storage\n", len(cs.statsCache))
			return nil
		}
	}

	// If loading fails, rebuild stats from filesystem
	return cs.rebuildStatsFromFilesystem()
}

func (cs *ContentStore) rebuildStatsFromFilesystem() error {
	fmt.Printf("Rebuilding stats from filesystem...\n")

	objectsPath := filepath.Join(cs.basePath, "objects")
	count := 0

	err := filepath.Walk(objectsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		// Extract hash from file path
		relPath, err := filepath.Rel(objectsPath, path)
		if err != nil {
			return nil
		}

		// Path should be like "ab/cd/abcd1234..."
		hash := strings.ReplaceAll(relPath, string(filepath.Separator), "")
		if len(hash) == 64 && cs.isValidHash(hash) {
			stats := &FileStats{
				Hash:         hash,
				Size:         info.Size(),
				CreatedAt:    info.ModTime(),
				LastAccessed: info.ModTime(),
				RefCount:     1,
				ContentType:  "application/octet-stream",
			}
			cs.statsCache[hash] = stats
			count++
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Rebuilt stats for %d files\n", count)

	// Save the rebuilt stats
	return cs.saveStatsImmediately()
}

func (cs *ContentStore) saveStats() error {
	cs.statsMutex.RLock()
	defer cs.statsMutex.RUnlock()

	persistent := PersistentStats{
		Version:   "1.0",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Stats:     cs.statsCache,
		Metadata: map[string]interface{}{
			"total_files": len(cs.statsCache),
			"base_path":   cs.basePath,
		},
	}

	data, err := json.MarshalIndent(persistent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	// Write to temporary file first, then atomic rename
	tempFile := cs.statsFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp stats file: %w", err)
	}

	if err := os.Rename(tempFile, cs.statsFile); err != nil {
		os.Remove(tempFile) // Clean up on failure
		return fmt.Errorf("failed to rename stats file: %w", err)
	}

	cs.lastSaveTime = time.Now()
	return nil
}

// saveStatsImmediately saves stats without acquiring locks (for internal use)
func (cs *ContentStore) saveStatsImmediately() error {
	persistent := PersistentStats{
		Version:   "1.0",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Stats:     cs.statsCache,
		Metadata: map[string]interface{}{
			"total_files": len(cs.statsCache),
			"base_path":   cs.basePath,
		},
	}

	data, err := json.MarshalIndent(persistent, "", "  ")
	if err != nil {
		return err
	}

	tempFile := cs.statsFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	return os.Rename(tempFile, cs.statsFile)
}

// backgroundStatsPersistence runs in background to periodically save stats
func (cs *ContentStore) backgroundStatsPersistence() {
	ticker := time.NewTicker(5 * time.Minute) // Save every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		if time.Since(cs.lastSaveTime) > 4*time.Minute {
			if err := cs.saveStats(); err != nil {
				fmt.Printf("Background stats save failed: %v\n", err)
			}
		}
	}
}

// Enhanced setStats to trigger background saves
func (cs *ContentStore) setStats(hash string, stats *FileStats) {
	cs.statsMutex.Lock()
	defer cs.statsMutex.Unlock()
	cs.statsCache[hash] = stats

	// Mark as needing save if enough time has passed
	if time.Since(cs.lastSaveTime) > time.Minute {
		go func() {
			time.Sleep(10 * time.Second) // Debounce rapid changes
			cs.saveStats()
		}()
	}
}

// GetStorageStats returns comprehensive storage statistics
func (cs *ContentStore) GetStorageStats() map[string]interface{} {
	cs.statsMutex.RLock()
	defer cs.statsMutex.RUnlock()

	var totalSize int64
	var totalFiles int
	var oldestFile, newestFile time.Time
	var totalRefCount int

	for _, stats := range cs.statsCache {
		totalSize += stats.Size
		totalFiles++
		totalRefCount += stats.RefCount

		if oldestFile.IsZero() || stats.CreatedAt.Before(oldestFile) {
			oldestFile = stats.CreatedAt
		}
		if newestFile.IsZero() || stats.CreatedAt.After(newestFile) {
			newestFile = stats.CreatedAt
		}
	}

	// Calculate directory sizes
	var objDirSize, tempDirSize int64
	if info, err := getDirSize(filepath.Join(cs.basePath, "objects")); err == nil {
		objDirSize = info
	}
	if info, err := getDirSize(cs.tempPath); err == nil {
		tempDirSize = info
	}

	return map[string]interface{}{
		"total_files":      totalFiles,
		"total_size":       totalSize,
		"total_size_str":   formatBytes(totalSize),
		"objects_dir_size": objDirSize,
		"temp_dir_size":    tempDirSize,
		"average_file_size": func() int64 {
			if totalFiles > 0 {
				return totalSize / int64(totalFiles)
			}
			return 0
		}(),
		"total_references": totalRefCount,
		"deduplication_ratio": func() float64 {
			if totalRefCount > 0 {
				return float64(totalRefCount) / float64(totalFiles)
			}
			return 1.0
		}(),
		"oldest_file":     oldestFile.Format(time.RFC3339),
		"newest_file":     newestFile.Format(time.RFC3339),
		"base_path":       cs.basePath,
		"last_stats_save": cs.lastSaveTime.Format(time.RFC3339),
	}
}

func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func detectContentType(metadata map[string]string) string {
	if contentType, exists := metadata["content-type"]; exists {
		return contentType
	}

	if filename, exists := metadata["filename"]; exists {
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".uasset", ".umap":
			return "application/x-unreal-asset"
		case ".fbx":
			return "application/x-fbx"
		case ".png":
			return "image/png"
		case ".jpg", ".jpeg":
			return "image/jpeg"
		case ".wav":
			return "audio/wav"
		case ".mp3":
			return "audio/mpeg"
		default:
			return "application/octet-stream"
		}
	}

	return "application/octet-stream"
}
