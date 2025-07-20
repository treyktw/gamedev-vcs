package storage

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GitStyleObjectStore implements Git-like content-addressable storage
type GitStyleObjectStore struct {
	basePath string
	mu       sync.RWMutex
}

// ObjectInfo contains metadata about a stored object
type ObjectInfo struct {
	Hash           string    `json:"hash"`
	Size           int64     `json:"size"`
	CompressedSize int64     `json:"compressed_size"`
	StoredAt       time.Time `json:"stored_at"`
	ObjectPath     string    `json:"object_path"`
}

// NewGitStyleObjectStore creates a new Git-style object store
func NewGitStyleObjectStore(basePath string) (*GitStyleObjectStore, error) {
	store := &GitStyleObjectStore{
		basePath: basePath,
	}

	// Create objects directory structure
	objectsDir := filepath.Join(basePath, "objects")
	if err := os.MkdirAll(objectsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create objects directory: %w", err)
	}

	return store, nil
}

// Store stores content and returns object info
func (s *GitStyleObjectStore) Store(content io.Reader, metadata map[string]string) (*ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read all content to calculate hash and compress
	data, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Calculate SHA256 hash
	hasher := sha256.New()
	hasher.Write(data)
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Check if object already exists
	if exists, err := s.exists(hash); err != nil {
		return nil, fmt.Errorf("failed to check object existence: %w", err)
	} else if exists {
		// Object already exists, return existing info
		return s.getObjectInfo(hash)
	}

	// Create object path like Git: objects/aa/bbcc...
	objectPath := s.getObjectPath(hash)
	objectDir := filepath.Dir(objectPath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(objectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create object directory: %w", err)
	}

	// Compress content using zlib (like Git)
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)

	// Write Git-style object header: "blob <size>\0<content>"
	header := fmt.Sprintf("blob %d\x00", len(data))
	if _, err := writer.Write([]byte(header)); err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to write object header: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to write compressed content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close compressor: %w", err)
	}

	// Write to temporary file first, then atomic rename
	tempPath := objectPath + ".tmp"
	if err := os.WriteFile(tempPath, compressed.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("failed to write temporary object: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, objectPath); err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to finalize object: %w", err)
	}

	return &ObjectInfo{
		Hash:           hash,
		Size:           int64(len(data)),
		CompressedSize: int64(compressed.Len()),
		StoredAt:       time.Now(),
		ObjectPath:     objectPath,
	}, nil
}

// Get retrieves content by hash
func (s *GitStyleObjectStore) Get(hash string) (io.ReadCloser, *ObjectInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	objectPath := s.getObjectPath(hash)

	// Check if object exists
	if _, err := os.Stat(objectPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("object not found: %s", hash)
	}

	// Read compressed data
	compressedData, err := os.ReadFile(objectPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read object: %w", err)
	}

	// Decompress
	reader := bytes.NewReader(compressedData)
	decompressor, err := zlib.NewReader(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create decompressor: %w", err)
	}

	// Read and validate Git-style header
	decompressed, err := io.ReadAll(decompressor)
	decompressor.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decompress object: %w", err)
	}

	// Parse Git-style header: "blob <size>\0<content>"
	nullIndex := bytes.IndexByte(decompressed, 0)
	if nullIndex == -1 {
		return nil, nil, fmt.Errorf("invalid object format: missing null separator")
	}

	header := string(decompressed[:nullIndex])
	content := decompressed[nullIndex+1:]

	// Validate header format
	if !strings.HasPrefix(header, "blob ") {
		return nil, nil, fmt.Errorf("invalid object format: expected blob header")
	}

	// Get object info
	info := &ObjectInfo{
		Hash:           hash,
		Size:           int64(len(content)),
		CompressedSize: int64(len(compressedData)),
		ObjectPath:     objectPath,
	}

	// Return content as ReadCloser
	contentReader := io.NopCloser(bytes.NewReader(content))
	return contentReader, info, nil
}

// Exists checks if an object exists by hash
func (s *GitStyleObjectStore) Exists(hash string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exists(hash)
}

// exists is the internal version without locking
func (s *GitStyleObjectStore) exists(hash string) (bool, error) {
	objectPath := s.getObjectPath(hash)
	_, err := os.Stat(objectPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// getObjectPath returns Git-style object path: objects/aa/bbcc...
func (s *GitStyleObjectStore) getObjectPath(hash string) string {
	if len(hash) < 3 {
		// Fallback for short hashes
		return filepath.Join(s.basePath, "objects", hash)
	}

	// Git-style: first 2 chars as directory, rest as filename
	dir := hash[:2]
	filename := hash[2:]
	return filepath.Join(s.basePath, "objects", dir, filename)
}

// getObjectInfo gets metadata about an existing object
func (s *GitStyleObjectStore) getObjectInfo(hash string) (*ObjectInfo, error) {
	objectPath := s.getObjectPath(hash)

	stat, err := os.Stat(objectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	// Read compressed data to get original size
	compressedData, err := os.ReadFile(objectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read object for size calculation: %w", err)
	}

	// Quick decompress to get original size
	reader := bytes.NewReader(compressedData)
	decompressor, err := zlib.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create decompressor: %w", err)
	}

	decompressed, err := io.ReadAll(decompressor)
	decompressor.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to decompress for size: %w", err)
	}

	// Parse header to get content size
	nullIndex := bytes.IndexByte(decompressed, 0)
	if nullIndex == -1 {
		return nil, fmt.Errorf("invalid object format")
	}

	content := decompressed[nullIndex+1:]

	return &ObjectInfo{
		Hash:           hash,
		Size:           int64(len(content)),
		CompressedSize: stat.Size(),
		StoredAt:       stat.ModTime(),
		ObjectPath:     objectPath,
	}, nil
}

// BatchExists checks existence of multiple objects
func (s *GitStyleObjectStore) BatchExists(hashes []string) (map[string]bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make(map[string]bool, len(hashes))

	for _, hash := range hashes {
		exists, err := s.exists(hash)
		if err != nil {
			return nil, fmt.Errorf("failed to check existence of %s: %w", hash, err)
		}
		results[hash] = exists
	}

	return results, nil
}

// Cleanup removes unused objects (can be extended for garbage collection)
func (s *GitStyleObjectStore) Cleanup() error {
	// Placeholder for future garbage collection
	// Could implement mark-and-sweep based on index references
	return nil
}

// GetStats returns storage statistics
func (s *GitStyleObjectStore) GetStats() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	objectsDir := filepath.Join(s.basePath, "objects")

	var totalObjects int64
	var totalSize int64
	var totalCompressedSize int64

	err := filepath.Walk(objectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && !strings.HasSuffix(path, ".tmp") {
			totalObjects++
			totalCompressedSize += info.Size()

			// Try to get original size by reading object
			relPath, _ := filepath.Rel(objectsDir, path)
			hash := strings.ReplaceAll(relPath, string(filepath.Separator), "")

			if objInfo, err := s.getObjectInfo(hash); err == nil {
				totalSize += objInfo.Size
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate stats: %w", err)
	}

	compressionRatio := float64(0)
	if totalSize > 0 {
		compressionRatio = float64(totalCompressedSize) / float64(totalSize)
	}

	return map[string]interface{}{
		"total_objects":     totalObjects,
		"total_size":        totalSize,
		"total_compressed":  totalCompressedSize,
		"compression_ratio": compressionRatio,
		"space_saved":       totalSize - totalCompressedSize,
		"objects_directory": objectsDir,
	}, nil
}
