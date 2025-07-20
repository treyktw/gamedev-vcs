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
	"sort"
	"strings"
	"time"
)

// GitStyleCommitStore manages Git-style commits with tree objects
type GitStyleCommitStore struct {
	basePath    string
	objectStore *GitStyleObjectStore
	fileIndex   *FileIndex
}

// TreeEntry represents a file in a tree object
type TreeEntry struct {
	Mode string `json:"mode"` // File mode (e.g., "100644")
	Name string `json:"name"` // File name
	Hash string `json:"hash"` // Content hash
	Size int64  `json:"size"` // File size
}

// TreeObject represents a Git-style tree (directory listing)
type TreeObject struct {
	Entries   []TreeEntry `json:"entries"`
	CreatedAt time.Time   `json:"created_at"`
}

// CommitObject represents a Git-style commit
type CommitObject struct {
	Tree      string            `json:"tree"`       // Hash of tree object
	Parents   []string          `json:"parents"`    // Parent commit hashes
	Author    string            `json:"author"`     // Author name
	AuthorID  string            `json:"author_id"`  // Author ID
	Committer string            `json:"committer"`  // Committer name
	Message   string            `json:"message"`    // Commit message
	Timestamp time.Time         `json:"timestamp"`  // Commit timestamp
	ProjectID string            `json:"project_id"` // Project ID
	Branch    string            `json:"branch"`     // Target branch
	Metadata  map[string]string `json:"metadata"`   // Additional metadata
}

// CommitResult represents the result of creating a commit
type CommitResult struct {
	CommitHash string    `json:"commit_hash"`
	TreeHash   string    `json:"tree_hash"`
	FilesCount int       `json:"files_count"`
	TreeSize   int64     `json:"tree_size"`
	Parents    []string  `json:"parents"`
	Message    string    `json:"message"`
	Author     string    `json:"author"`
	Timestamp  time.Time `json:"timestamp"`
	Branch     string    `json:"branch"`
}

// NewGitStyleCommitStore creates a new commit store
func NewGitStyleCommitStore(basePath string, objectStore *GitStyleObjectStore, fileIndex *FileIndex) *GitStyleCommitStore {
	return &GitStyleCommitStore{
		basePath:    basePath,
		objectStore: objectStore,
		fileIndex:   fileIndex,
	}
}

// CreateCommit creates a Git-style commit from current index state
func (cs *GitStyleCommitStore) CreateCommit(commit *CommitObject) (*CommitResult, error) {
	// Get staged files from index
	stagedEntries := cs.fileIndex.GetStagedEntries()
	if len(stagedEntries) == 0 {
		return nil, fmt.Errorf("no files staged for commit")
	}

	// Create tree object from staged files
	treeObject := &TreeObject{
		Entries:   make([]TreeEntry, 0, len(stagedEntries)),
		CreatedAt: time.Now(),
	}

	// Convert index entries to tree entries
	for path, entry := range stagedEntries {
		treeEntry := TreeEntry{
			Mode: "100644", // Regular file mode
			Name: path,
			Hash: entry.Hash,
			Size: entry.Size,
		}
		treeObject.Entries = append(treeObject.Entries, treeEntry)
	}

	// Sort entries by name (Git requirement)
	sort.Slice(treeObject.Entries, func(i, j int) bool {
		return treeObject.Entries[i].Name < treeObject.Entries[j].Name
	})

	// Store tree object
	treeHash, err := cs.storeTreeObject(treeObject)
	if err != nil {
		return nil, fmt.Errorf("failed to store tree object: %w", err)
	}

	// Set tree hash in commit
	commit.Tree = treeHash
	commit.Timestamp = time.Now()

	// Store commit object
	commitHash, err := cs.storeCommitObject(commit)
	if err != nil {
		return nil, fmt.Errorf("failed to store commit object: %w", err)
	}

	// Update branch reference
	if err := cs.updateBranchRef(commit.Branch, commitHash); err != nil {
		return nil, fmt.Errorf("failed to update branch ref: %w", err)
	}

	// Mark files as committed (unstage them)
	cs.fileIndex.MarkUnstaged(getFilePathsFromEntries(stagedEntries))

	// Save updated index
	if err := cs.fileIndex.Save(); err != nil {
		fmt.Printf("Warning: failed to save index after commit: %v\n", err)
	}

	return &CommitResult{
		CommitHash: commitHash,
		TreeHash:   treeHash,
		FilesCount: len(treeObject.Entries),
		TreeSize:   cs.calculateTreeSize(treeObject),
		Parents:    commit.Parents,
		Message:    commit.Message,
		Author:     commit.Author,
		Timestamp:  commit.Timestamp,
		Branch:     commit.Branch,
	}, nil
}

// storeTreeObject stores a tree object and returns its hash
func (cs *GitStyleCommitStore) storeTreeObject(tree *TreeObject) (string, error) {
	// Serialize tree object in Git-style binary format
	var buf bytes.Buffer

	// Write tree header
	header := fmt.Sprintf("tree %d\x00", len(tree.Entries))
	buf.WriteString(header)

	// Write sorted entries in binary format
	for _, entry := range tree.Entries {
		// Format: mode<space>name<null>hash_bytes
		fmt.Fprintf(&buf, "%s %s\x00", entry.Mode, entry.Name)

		// Write hash as binary (20 bytes for SHA-1, 32 for SHA-256)
		hashBytes, err := hex.DecodeString(entry.Hash)
		if err != nil {
			return "", fmt.Errorf("invalid hash for %s: %w", entry.Name, err)
		}
		buf.Write(hashBytes)
	}

	// Calculate tree hash
	hasher := sha256.New()
	hasher.Write(buf.Bytes())
	treeHash := hex.EncodeToString(hasher.Sum(nil))

	// Store as compressed object
	objectPath := cs.getObjectPath(treeHash)
	if err := os.MkdirAll(filepath.Dir(objectPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create object directory: %w", err)
	}

	// Compress and store
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	writer.Write(buf.Bytes())
	writer.Close()

	// Atomic write
	tempPath := objectPath + ".tmp"
	if err := os.WriteFile(tempPath, compressed.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write tree object: %w", err)
	}

	if err := os.Rename(tempPath, objectPath); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to finalize tree object: %w", err)
	}

	return treeHash, nil
}

// storeCommitObject stores a commit object and returns its hash
func (cs *GitStyleCommitStore) storeCommitObject(commit *CommitObject) (string, error) {
	// Serialize commit object in Git-style format
	var buf bytes.Buffer

	// Git-style commit format
	fmt.Fprintf(&buf, "tree %s\n", commit.Tree)

	for _, parent := range commit.Parents {
		fmt.Fprintf(&buf, "parent %s\n", parent)
	}

	fmt.Fprintf(&buf, "author %s <%s> %d +0000\n",
		commit.Author, commit.AuthorID, commit.Timestamp.Unix())
	fmt.Fprintf(&buf, "committer %s <%s> %d +0000\n",
		commit.Committer, commit.AuthorID, commit.Timestamp.Unix())

	// Add project metadata
	fmt.Fprintf(&buf, "project %s\n", commit.ProjectID)
	fmt.Fprintf(&buf, "branch %s\n", commit.Branch)

	// Add custom metadata
	for key, value := range commit.Metadata {
		fmt.Fprintf(&buf, "%s %s\n", key, value)
	}

	fmt.Fprintf(&buf, "\n%s\n", commit.Message)

	// Calculate commit hash
	commitData := buf.Bytes()
	fullObject := fmt.Sprintf("commit %d\x00%s", len(commitData), string(commitData))

	hasher := sha256.New()
	hasher.Write([]byte(fullObject))
	commitHash := hex.EncodeToString(hasher.Sum(nil))

	// Store as compressed object
	objectPath := cs.getObjectPath(commitHash)
	if err := os.MkdirAll(filepath.Dir(objectPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create object directory: %w", err)
	}

	// Compress and store
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	writer.Write([]byte(fullObject))
	writer.Close()

	// Atomic write
	tempPath := objectPath + ".tmp"
	if err := os.WriteFile(tempPath, compressed.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write commit object: %w", err)
	}

	if err := os.Rename(tempPath, objectPath); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to finalize commit object: %w", err)
	}

	return commitHash, nil
}

// updateBranchRef updates the branch reference to point to new commit
func (cs *GitStyleCommitStore) updateBranchRef(branch, commitHash string) error {
	refsDir := filepath.Join(cs.basePath, "refs", "heads")
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		return fmt.Errorf("failed to create refs directory: %w", err)
	}

	refPath := filepath.Join(refsDir, branch)
	tempPath := refPath + ".tmp"

	// Write commit hash to branch ref
	if err := os.WriteFile(tempPath, []byte(commitHash+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write branch ref: %w", err)
	}

	// Atomic update
	if err := os.Rename(tempPath, refPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to update branch ref: %w", err)
	}

	// Update HEAD if this is the current branch
	headPath := filepath.Join(cs.basePath, "HEAD")
	currentBranch, err := cs.getCurrentBranch()
	if err == nil && currentBranch == branch {
		headTempPath := headPath + ".tmp"
		if err := os.WriteFile(headTempPath, []byte(commitHash+"\n"), 0644); err == nil {
			os.Rename(headTempPath, headPath)
		}
	}

	return nil
}

// GetCommit retrieves a commit object by hash
func (cs *GitStyleCommitStore) GetCommit(commitHash string) (*CommitObject, error) {
	objectPath := cs.getObjectPath(commitHash)

	// Read and decompress object
	compressedData, err := os.ReadFile(objectPath)
	if err != nil {
		return nil, fmt.Errorf("commit not found: %s", commitHash)
	}

	reader := bytes.NewReader(compressedData)
	decompressor, err := zlib.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress commit: %w", err)
	}
	defer decompressor.Close()

	data, err := io.ReadAll(decompressor)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit: %w", err)
	}

	// Parse Git-style commit format
	content := string(data)
	if !strings.HasPrefix(content, "commit ") {
		return nil, fmt.Errorf("invalid commit object format")
	}

	// Extract commit data after header
	nullIndex := strings.Index(content, "\x00")
	if nullIndex == -1 {
		return nil, fmt.Errorf("invalid commit object: missing null separator")
	}

	commitData := content[nullIndex+1:]
	return cs.parseCommitData(commitData)
}

// GetTree retrieves a tree object by hash
func (cs *GitStyleCommitStore) GetTree(treeHash string) (*TreeObject, error) {
	objectPath := cs.getObjectPath(treeHash)

	// Read and decompress object
	compressedData, err := os.ReadFile(objectPath)
	if err != nil {
		return nil, fmt.Errorf("tree not found: %s", treeHash)
	}

	reader := bytes.NewReader(compressedData)
	decompressor, err := zlib.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress tree: %w", err)
	}
	defer decompressor.Close()

	data, err := io.ReadAll(decompressor)
	if err != nil {
		return nil, fmt.Errorf("failed to read tree: %w", err)
	}

	// Parse tree format
	return cs.parseTreeData(data)
}

// GetBranchHead gets the current commit hash for a branch
func (cs *GitStyleCommitStore) GetBranchHead(branch string) (string, error) {
	refPath := filepath.Join(cs.basePath, "refs", "heads", branch)
	data, err := os.ReadFile(refPath)
	if err != nil {
		return "", fmt.Errorf("branch not found: %s", branch)
	}
	return strings.TrimSpace(string(data)), nil
}

// ListCommits returns commit history for a branch
func (cs *GitStyleCommitStore) ListCommits(branch string, limit int) ([]*CommitObject, error) {
	commitHash, err := cs.GetBranchHead(branch)
	if err != nil {
		return nil, err
	}

	var commits []*CommitObject
	visited := make(map[string]bool)

	// Walk commit history
	for len(commits) < limit && commitHash != "" && !visited[commitHash] {
		visited[commitHash] = true

		commit, err := cs.GetCommit(commitHash)
		if err != nil {
			break
		}

		commits = append(commits, commit)

		// Move to first parent
		if len(commit.Parents) > 0 {
			commitHash = commit.Parents[0]
		} else {
			break
		}
	}

	return commits, nil
}

// Helper methods

func (cs *GitStyleCommitStore) getObjectPath(hash string) string {
	if len(hash) < 3 {
		return filepath.Join(cs.basePath, "objects", hash)
	}
	return filepath.Join(cs.basePath, "objects", hash[:2], hash[2:])
}

func (cs *GitStyleCommitStore) getCurrentBranch() (string, error) {
	headPath := filepath.Join(cs.basePath, "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/"), nil
	}

	return "main", nil // Default branch
}

func (cs *GitStyleCommitStore) calculateTreeSize(tree *TreeObject) int64 {
	var total int64
	for _, entry := range tree.Entries {
		total += entry.Size
	}
	return total
}

func (cs *GitStyleCommitStore) parseCommitData(data string) (*CommitObject, error) {
	lines := strings.Split(data, "\n")
	commit := &CommitObject{
		Metadata: make(map[string]string),
	}

	messageStart := -1
	for i, line := range lines {
		if line == "" {
			messageStart = i + 1
			break
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "tree":
			commit.Tree = value
		case "parent":
			commit.Parents = append(commit.Parents, value)
		case "author":
			// Parse: "Name <email> timestamp timezone"
			commit.Author = strings.Split(value, " <")[0]
		case "committer":
			commit.Committer = strings.Split(value, " <")[0]
		case "project":
			commit.ProjectID = value
		case "branch":
			commit.Branch = value
		default:
			commit.Metadata[key] = value
		}
	}

	if messageStart > 0 && messageStart < len(lines) {
		commit.Message = strings.Join(lines[messageStart:], "\n")
	}

	return commit, nil
}

func (cs *GitStyleCommitStore) parseTreeData(data []byte) (*TreeObject, error) {
	// Parse binary tree format
	tree := &TreeObject{
		Entries:   []TreeEntry{},
		CreatedAt: time.Now(),
	}

	// Skip tree header
	content := data
	if bytes.HasPrefix(content, []byte("tree ")) {
		nullIndex := bytes.IndexByte(content, 0)
		if nullIndex > 0 {
			content = content[nullIndex+1:]
		}
	}

	// Parse entries
	offset := 0
	for offset < len(content) {
		// Find space separator
		spaceIndex := bytes.IndexByte(content[offset:], ' ')
		if spaceIndex == -1 {
			break
		}

		mode := string(content[offset : offset+spaceIndex])
		offset += spaceIndex + 1

		// Find null separator
		nullIndex := bytes.IndexByte(content[offset:], 0)
		if nullIndex == -1 {
			break
		}

		name := string(content[offset : offset+nullIndex])
		offset += nullIndex + 1

		// Read hash (32 bytes for SHA-256)
		if offset+32 > len(content) {
			break
		}

		hash := hex.EncodeToString(content[offset : offset+32])
		offset += 32

		tree.Entries = append(tree.Entries, TreeEntry{
			Mode: mode,
			Name: name,
			Hash: hash,
			Size: 0, // Size not stored in tree object
		})
	}

	return tree, nil
}

func getFilePathsFromEntries(entries map[string]*IndexEntry) []string {
	paths := make([]string, 0, len(entries))
	for path := range entries {
		paths = append(paths, path)
	}
	return paths
}
