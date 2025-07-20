package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
)

// IndexEntry represents a file in the index (like Git's index)
type IndexEntry struct {
	Path      string    `json:"path"`
	Hash      string    `json:"hash"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
	Inode     uint64    `json:"inode"`
	Mode      uint32    `json:"mode"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Staged    bool      `json:"staged"`
}

// FileIndex manages a Git-style file index with stat optimization
type FileIndex struct {
	entries   map[string]*IndexEntry
	mu        sync.RWMutex
	indexPath string
	version   uint32
}

// IndexHeader represents the binary index file header
type IndexHeader struct {
	Signature  [4]byte // "GVCS" (GameDev VCS)
	Version    uint32
	NumEntries uint32
	Checksum   [32]byte // SHA256 of the index content
}

// NewFileIndex creates a new file index
func NewFileIndex(indexPath string) (*FileIndex, error) {
	index := &FileIndex{
		entries:   make(map[string]*IndexEntry),
		indexPath: indexPath,
		version:   1,
	}

	// Try to load existing index
	if err := index.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load existing index: %w", err)
	}

	return index, nil
}

// NeedsUpdate checks if a file needs to be updated using stat optimization
func (idx *FileIndex) NeedsUpdate(filePath string) (bool, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Get current file stats
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File was deleted, needs update to remove from index
			_, exists := idx.entries[filePath]
			return exists, nil
		}
		return false, err
	}

	// Check if we have this file in index
	entry, exists := idx.entries[filePath]
	if !exists {
		return true, nil // New file
	}

	// Git-style stat optimization: check size and mtime first
	currentSize := fileInfo.Size()
	currentModTime := fileInfo.ModTime()

	// If size or mtime changed, file needs update
	if currentSize != entry.Size || !currentModTime.Equal(entry.ModTime) {
		return true, nil
	}

	// Advanced check: inode and mode (Unix-like systems)
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		currentInode := stat.Ino
		currentMode := uint32(fileInfo.Mode())

		if currentInode != entry.Inode || currentMode != entry.Mode {
			return true, nil
		}
	}

	// File hasn't changed based on stat information
	return false, nil
}

// BatchNeedsUpdate checks multiple files for updates efficiently
func (idx *FileIndex) BatchNeedsUpdate(filePaths []string) (map[string]bool, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	results := make(map[string]bool, len(filePaths))

	for _, filePath := range filePaths {
		needsUpdate, err := idx.needsUpdateInternal(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to check %s: %w", filePath, err)
		}
		results[filePath] = needsUpdate
	}

	return results, nil
}

// needsUpdateInternal is the internal version without locking
func (idx *FileIndex) needsUpdateInternal(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			_, exists := idx.entries[filePath]
			return exists, nil
		}
		return false, err
	}

	entry, exists := idx.entries[filePath]
	if !exists {
		return true, nil
	}

	// Stat-based comparison
	currentSize := fileInfo.Size()
	currentModTime := fileInfo.ModTime()

	if currentSize != entry.Size || !currentModTime.Equal(entry.ModTime) {
		return true, nil
	}

	// Advanced stat check
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		currentInode := stat.Ino
		currentMode := uint32(fileInfo.Mode())

		if currentInode != entry.Inode || currentMode != entry.Mode {
			return true, nil
		}
	}

	return false, nil
}

// UpdateEntry updates or adds an entry to the index
func (idx *FileIndex) UpdateEntry(filePath, hash string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	var inode uint64
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		inode = stat.Ino
	}

	now := time.Now()
	entry := &IndexEntry{
		Path:      filePath,
		Hash:      hash,
		Size:      fileInfo.Size(),
		ModTime:   fileInfo.ModTime(),
		Inode:     inode,
		Mode:      uint32(fileInfo.Mode()),
		UpdatedAt: now,
		Staged:    true,
	}

	// Set CreatedAt for new entries
	if existing, exists := idx.entries[filePath]; exists {
		entry.CreatedAt = existing.CreatedAt
	} else {
		entry.CreatedAt = now
	}

	idx.entries[filePath] = entry
	return nil
}

// BatchUpdateEntries updates multiple entries efficiently
func (idx *FileIndex) BatchUpdateEntries(updates map[string]string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	now := time.Now()

	for filePath, hash := range updates {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to stat file %s: %w", filePath, err)
		}

		var inode uint64
		if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
			inode = stat.Ino
		}

		entry := &IndexEntry{
			Path:      filePath,
			Hash:      hash,
			Size:      fileInfo.Size(),
			ModTime:   fileInfo.ModTime(),
			Inode:     inode,
			Mode:      uint32(fileInfo.Mode()),
			UpdatedAt: now,
			Staged:    true,
		}

		// Preserve CreatedAt for existing entries
		if existing, exists := idx.entries[filePath]; exists {
			entry.CreatedAt = existing.CreatedAt
		} else {
			entry.CreatedAt = now
		}

		idx.entries[filePath] = entry
	}

	return nil
}

// RemoveEntry removes an entry from the index
func (idx *FileIndex) RemoveEntry(filePath string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	delete(idx.entries, filePath)
}

// GetEntry gets an entry by path
func (idx *FileIndex) GetEntry(filePath string) (*IndexEntry, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	entry, exists := idx.entries[filePath]
	return entry, exists
}

// GetAllEntries returns all entries
func (idx *FileIndex) GetAllEntries() map[string]*IndexEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*IndexEntry, len(idx.entries))
	for path, entry := range idx.entries {
		entryCopy := *entry
		result[path] = &entryCopy
	}
	return result
}

// GetStagedEntries returns only staged entries
func (idx *FileIndex) GetStagedEntries() map[string]*IndexEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make(map[string]*IndexEntry)
	for path, entry := range idx.entries {
		if entry.Staged {
			entryCopy := *entry
			result[path] = &entryCopy
		}
	}
	return result
}

// MarkStaged marks entries as staged
func (idx *FileIndex) MarkStaged(filePaths []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for _, path := range filePaths {
		if entry, exists := idx.entries[path]; exists {
			entry.Staged = true
			entry.UpdatedAt = time.Now()
		}
	}
}

// MarkUnstaged marks entries as unstaged
func (idx *FileIndex) MarkUnstaged(filePaths []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for _, path := range filePaths {
		if entry, exists := idx.entries[path]; exists {
			entry.Staged = false
			entry.UpdatedAt = time.Now()
		}
	}
}

// Save writes the index to disk in binary format (Git-style)
func (idx *FileIndex) Save() error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(idx.indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Create temporary file for atomic write
	tempPath := idx.indexPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp index file: %w", err)
	}
	defer func() {
		file.Close()
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	// Calculate content for checksum
	var contentBuf bytes.Buffer
	if err := idx.writeContent(&contentBuf); err != nil {
		return fmt.Errorf("failed to prepare content: %w", err)
	}

	// Calculate checksum
	hasher := sha256.New()
	hasher.Write(contentBuf.Bytes())
	checksum := hasher.Sum(nil)

	// Write header
	header := IndexHeader{
		Signature:  [4]byte{'G', 'V', 'C', 'S'},
		Version:    idx.version,
		NumEntries: uint32(len(idx.entries)),
	}
	copy(header.Checksum[:], checksum)

	if err := binary.Write(file, binary.BigEndian, header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write content
	if _, err := file.Write(contentBuf.Bytes()); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	file.Close()

	// Atomic rename
	if err := os.Rename(tempPath, idx.indexPath); err != nil {
		return fmt.Errorf("failed to finalize index: %w", err)
	}

	return nil
}

// writeContent writes the index entries in binary format
func (idx *FileIndex) writeContent(w io.Writer) error {
	// Sort entries by path for deterministic output
	paths := make([]string, 0, len(idx.entries))
	for path := range idx.entries {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		entry := idx.entries[path]

		// Write entry in binary format
		// Format: [path_len][path][hash][size][mod_time][inode][mode][created_at][updated_at][staged]

		pathBytes := []byte(entry.Path)
		hashBytes, _ := hex.DecodeString(entry.Hash)

		// Path length and path
		if err := binary.Write(w, binary.BigEndian, uint32(len(pathBytes))); err != nil {
			return err
		}
		if _, err := w.Write(pathBytes); err != nil {
			return err
		}

		// Hash (32 bytes for SHA256)
		if _, err := w.Write(hashBytes); err != nil {
			return err
		}

		// Size
		if err := binary.Write(w, binary.BigEndian, entry.Size); err != nil {
			return err
		}

		// ModTime (Unix timestamp + nanoseconds)
		if err := binary.Write(w, binary.BigEndian, entry.ModTime.Unix()); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, int32(entry.ModTime.Nanosecond())); err != nil {
			return err
		}

		// Inode and Mode
		if err := binary.Write(w, binary.BigEndian, entry.Inode); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, entry.Mode); err != nil {
			return err
		}

		// CreatedAt
		if err := binary.Write(w, binary.BigEndian, entry.CreatedAt.Unix()); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, int32(entry.CreatedAt.Nanosecond())); err != nil {
			return err
		}

		// UpdatedAt
		if err := binary.Write(w, binary.BigEndian, entry.UpdatedAt.Unix()); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, int32(entry.UpdatedAt.Nanosecond())); err != nil {
			return err
		}

		// Staged flag
		staged := uint8(0)
		if entry.Staged {
			staged = 1
		}
		if err := binary.Write(w, binary.BigEndian, staged); err != nil {
			return err
		}
	}

	return nil
}

// Load reads the index from disk
func (idx *FileIndex) Load() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	file, err := os.Open(idx.indexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read header
	var header IndexHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Verify signature
	expectedSig := [4]byte{'G', 'V', 'C', 'S'}
	if header.Signature != expectedSig {
		return fmt.Errorf("invalid index signature")
	}

	// Read content and verify checksum
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	hasher := sha256.New()
	hasher.Write(content)
	if !bytes.Equal(hasher.Sum(nil), header.Checksum[:]) {
		return fmt.Errorf("index checksum mismatch")
	}

	// Parse entries
	reader := bytes.NewReader(content)
	idx.entries = make(map[string]*IndexEntry, header.NumEntries)

	for i := uint32(0); i < header.NumEntries; i++ {
		entry, err := idx.readEntry(reader)
		if err != nil {
			return fmt.Errorf("failed to read entry %d: %w", i, err)
		}
		idx.entries[entry.Path] = entry
	}

	idx.version = header.Version
	return nil
}

// readEntry reads a single entry from binary format
func (idx *FileIndex) readEntry(r io.Reader) (*IndexEntry, error) {
	entry := &IndexEntry{}

	// Read path length and path
	var pathLen uint32
	if err := binary.Read(r, binary.BigEndian, &pathLen); err != nil {
		return nil, err
	}

	pathBytes := make([]byte, pathLen)
	if _, err := io.ReadFull(r, pathBytes); err != nil {
		return nil, err
	}
	entry.Path = string(pathBytes)

	// Read hash
	hashBytes := make([]byte, 32) // SHA256
	if _, err := io.ReadFull(r, hashBytes); err != nil {
		return nil, err
	}
	entry.Hash = hex.EncodeToString(hashBytes)

	// Read size
	if err := binary.Read(r, binary.BigEndian, &entry.Size); err != nil {
		return nil, err
	}

	// Read ModTime
	var modSec int64
	var modNsec int32
	if err := binary.Read(r, binary.BigEndian, &modSec); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &modNsec); err != nil {
		return nil, err
	}
	entry.ModTime = time.Unix(modSec, int64(modNsec))

	// Read inode and mode
	if err := binary.Read(r, binary.BigEndian, &entry.Inode); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &entry.Mode); err != nil {
		return nil, err
	}

	// Read CreatedAt
	var createdSec int64
	var createdNsec int32
	if err := binary.Read(r, binary.BigEndian, &createdSec); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &createdNsec); err != nil {
		return nil, err
	}
	entry.CreatedAt = time.Unix(createdSec, int64(createdNsec))

	// Read UpdatedAt
	var updatedSec int64
	var updatedNsec int32
	if err := binary.Read(r, binary.BigEndian, &updatedSec); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &updatedNsec); err != nil {
		return nil, err
	}
	entry.UpdatedAt = time.Unix(updatedSec, int64(updatedNsec))

	// Read staged flag
	var staged uint8
	if err := binary.Read(r, binary.BigEndian, &staged); err != nil {
		return nil, err
	}
	entry.Staged = staged == 1

	return entry, nil
}

// GetChangedFiles returns files that have changed since last index update
func (idx *FileIndex) GetChangedFiles(filePaths []string) ([]string, error) {
	needsUpdate, err := idx.BatchNeedsUpdate(filePaths)
	if err != nil {
		return nil, err
	}

	var changed []string
	for filePath, needs := range needsUpdate {
		if needs {
			changed = append(changed, filePath)
		}
	}

	return changed, nil
}

// Clean removes entries for files that no longer exist
func (idx *FileIndex) Clean() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	var toDelete []string

	for filePath := range idx.entries {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			toDelete = append(toDelete, filePath)
		}
	}

	for _, filePath := range toDelete {
		delete(idx.entries, filePath)
	}

	return nil
}

// GetStats returns index statistics
func (idx *FileIndex) GetStats() map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var totalSize int64
	var stagedCount int

	for _, entry := range idx.entries {
		totalSize += entry.Size
		if entry.Staged {
			stagedCount++
		}
	}

	return map[string]interface{}{
		"total_entries":  len(idx.entries),
		"staged_entries": stagedCount,
		"total_size":     totalSize,
		"index_path":     idx.indexPath,
		"version":        idx.version,
	}
}

// CompareWithFileSystem compares index with actual filesystem state
func (idx *FileIndex) CompareWithFileSystem(filePaths []string) (map[string]string, error) {
	results := make(map[string]string)

	for _, filePath := range filePaths {
		needsUpdate, err := idx.NeedsUpdate(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				results[filePath] = "deleted"
				continue
			}
			return nil, fmt.Errorf("failed to check %s: %w", filePath, err)
		}

		_, exists := idx.GetEntry(filePath)
		if !exists {
			results[filePath] = "new"
		} else if needsUpdate {
			results[filePath] = "modified"
		} else {
			results[filePath] = "unchanged"
		}
	}

	return results, nil
}
