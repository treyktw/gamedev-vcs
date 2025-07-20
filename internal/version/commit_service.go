package version

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Telerallc/gamedev-vcs/models"
	"gorm.io/gorm"
)

// CommitService handles version control operations
type CommitService struct {
	db *gorm.DB
}

// NewCommitService creates a new commit service
func NewCommitService(db *gorm.DB) *CommitService {
	return &CommitService{db: db}
}

// CreateCommit creates a new commit with the given files
func (cs *CommitService) CreateCommit(projectID, authorID, message string, files []models.File, parentCommitIDs []string) (*models.Commit, error) {
	// Create commit tree from files
	treeFiles := make([]models.CommitTreeFile, len(files))
	for i, file := range files {
		treeFiles[i] = models.CommitTreeFile{
			Path:        file.Path,
			ContentHash: file.ContentHash,
			Size:        file.Size,
			Mode:        "100644", // Regular file
			Type:        "file",
		}
	}

	// Calculate tree hash
	treeHash := cs.calculateTreeHash(treeFiles)
	
	// Calculate commit hash
	commitHash := cs.calculateCommitHash(projectID, authorID, message, treeHash, parentCommitIDs, time.Now())

	var createdCommit *models.Commit
	err := cs.db.Transaction(func(tx *gorm.DB) error {
		// Create commit tree
		commitTree := &models.CommitTree{
			ID:        treeHash,
			ProjectID: projectID,
			CommitID:  commitHash,
			Files:     treeFiles,
			CreatedAt: time.Now(),
		}

		if err := tx.Create(commitTree).Error; err != nil {
			return fmt.Errorf("failed to create commit tree: %w", err)
		}

		// Create commit
		commit := &models.Commit{
			ID:        commitHash,
			ProjectID: projectID,
			AuthorID:  authorID,
			Message:   message,
			TreeHash:  treeHash,
			ParentIDs: parentCommitIDs,
			CreatedAt: time.Now(),
		}

		if err := tx.Create(commit).Error; err != nil {
			return fmt.Errorf("failed to create commit: %w", err)
		}

		// Create file versions for this commit
		for _, file := range files {
			fileVersion := &models.FileVersion{
				ID:          fmt.Sprintf("%s:%s", commitHash, file.Path),
				ProjectID:   projectID,
				Path:        file.Path,
				ContentHash: file.ContentHash,
				CommitID:    commitHash,
				Size:        file.Size,
				MimeType:    file.MimeType,
				CreatedAt:   time.Now(),
			}

			if err := tx.Create(fileVersion).Error; err != nil {
				return fmt.Errorf("failed to create file version: %w", err)
			}
		}

		createdCommit = commit
		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdCommit, nil
}

// GetCommitHistory retrieves commit history for a project/branch
func (cs *CommitService) GetCommitHistory(projectID string, branchName string, limit int) ([]models.Commit, error) {
	// Get the branch to find the HEAD commit
	var branch models.Branch
	if err := cs.db.Where("project_id = ? AND name = ?", projectID, branchName).First(&branch).Error; err != nil {
		return nil, fmt.Errorf("branch not found: %w", err)
	}

	if branch.LastCommit == "" {
		return []models.Commit{}, nil
	}

	// Traverse commit history starting from HEAD
	var commits []models.Commit
	visitedCommits := make(map[string]bool)
	commitQueue := []string{branch.LastCommit}

	for len(commitQueue) > 0 && len(commits) < limit {
		commitID := commitQueue[0]
		commitQueue = commitQueue[1:]

		if visitedCommits[commitID] {
			continue
		}
		visitedCommits[commitID] = true

		var commit models.Commit
		if err := cs.db.Preload("Author").Where("id = ?", commitID).First(&commit).Error; err != nil {
			continue // Skip missing commits
		}

		commits = append(commits, commit)

		// Add parent commits to queue
		commitQueue = append(commitQueue, commit.ParentIDs...)
	}

	return commits, nil
}

// GetCommitByID retrieves a specific commit with its tree
func (cs *CommitService) GetCommitByID(commitID string) (*models.Commit, error) {
	var commit models.Commit
	if err := cs.db.Preload("Author").Where("id = ?", commitID).First(&commit).Error; err != nil {
		return nil, fmt.Errorf("commit not found: %w", err)
	}

	// Get commit tree separately
	var tree models.CommitTree
	if err := cs.db.Where("id = ?", commit.TreeHash).First(&tree).Error; err != nil {
		// Tree not found - this is okay for now
		tree = models.CommitTree{Files: []models.CommitTreeFile{}}
	}
	return &commit, nil
}

// GetFileAtCommit retrieves a specific file at a given commit
func (cs *CommitService) GetFileAtCommit(commitID, filePath string) (*models.FileVersion, error) {
	var fileVersion models.FileVersion
	if err := cs.db.Where("commit_id = ? AND path = ?", commitID, filePath).First(&fileVersion).Error; err != nil {
		return nil, fmt.Errorf("file not found at commit: %w", err)
	}

	return &fileVersion, nil
}

// GetFileHistory retrieves the history of a specific file
func (cs *CommitService) GetFileHistory(projectID, filePath string, limit int) ([]models.FileVersion, error) {
	var fileVersions []models.FileVersion
	
	err := cs.db.
		Preload("Commit.Author").
		Where("project_id = ? AND path = ?", projectID, filePath).
		Order("created_at DESC").
		Limit(limit).
		Find(&fileVersions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get file history: %w", err)
	}

	return fileVersions, nil
}

// UpdateBranchHead updates the HEAD commit of a branch
func (cs *CommitService) UpdateBranchHead(projectID, branchName, commitID string) error {
	// Verify commit exists
	var commit models.Commit
	if err := cs.db.Where("id = ? AND project_id = ?", commitID, projectID).First(&commit).Error; err != nil {
		return fmt.Errorf("commit not found: %w", err)
	}

	// Update branch
	result := cs.db.Model(&models.Branch{}).
		Where("project_id = ? AND name = ?", projectID, branchName).
		Update("last_commit", commitID)

	if result.Error != nil {
		return fmt.Errorf("failed to update branch: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("branch not found")
	}

	// Update or create ref
	ref := &models.Ref{
		ID:        fmt.Sprintf("refs/heads/%s:%s", branchName, projectID),
		ProjectID: projectID,
		Name:      fmt.Sprintf("refs/heads/%s", branchName),
		Type:      "branch",
		CommitID:  commitID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return cs.db.Save(ref).Error
}

// DiffCommits compares two commits and returns the differences
func (cs *CommitService) DiffCommits(fromCommitID, toCommitID string) (*CommitDiff, error) {
	fromCommit, err := cs.GetCommitByID(fromCommitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from commit: %w", err)
	}

	toCommit, err := cs.GetCommitByID(toCommitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get to commit: %w", err)
	}

	// Get commit trees
	var fromTree, toTree models.CommitTree
	cs.db.Where("id = ?", fromCommit.TreeHash).First(&fromTree)
	cs.db.Where("id = ?", toCommit.TreeHash).First(&toTree)

	// Create file maps for comparison
	fromFiles := make(map[string]models.CommitTreeFile)
	for _, file := range fromTree.Files {
		fromFiles[file.Path] = file
	}

	toFiles := make(map[string]models.CommitTreeFile)
	for _, file := range toTree.Files {
		toFiles[file.Path] = file
	}

	diff := &CommitDiff{
		FromCommit: *fromCommit,
		ToCommit:   *toCommit,
		Changes:    []FileDiff{},
	}

	// Find all unique file paths
	allPaths := make(map[string]bool)
	for path := range fromFiles {
		allPaths[path] = true
	}
	for path := range toFiles {
		allPaths[path] = true
	}

	// Compare files
	for path := range allPaths {
		fromFile, fromExists := fromFiles[path]
		toFile, toExists := toFiles[path]

		var changeType string
		switch {
		case !fromExists && toExists:
			changeType = "added"
		case fromExists && !toExists:
			changeType = "deleted"
		case fromFile.ContentHash != toFile.ContentHash:
			changeType = "modified"
		default:
			continue // No change
		}

		diff.Changes = append(diff.Changes, FileDiff{
			Path:       path,
			ChangeType: changeType,
			FromFile:   fromFile,
			ToFile:     toFile,
		})
	}

	return diff, nil
}

// Helper functions

func (cs *CommitService) calculateTreeHash(files []models.CommitTreeFile) string {
	// Sort files by path for consistent hashing
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Create tree content string
	var treeContent strings.Builder
	for _, file := range files {
		treeContent.WriteString(fmt.Sprintf("%s %s %s %d\n", 
			file.Mode, file.Type, file.Path, file.Size))
		treeContent.WriteString(file.ContentHash)
		treeContent.WriteString("\n")
	}

	hash := sha1.Sum([]byte(treeContent.String()))
	return hex.EncodeToString(hash[:])
}

func (cs *CommitService) calculateCommitHash(projectID, authorID, message, treeHash string, parentIDs []string, timestamp time.Time) string {
	// Sort parent IDs for consistent hashing
	sort.Strings(parentIDs)

	var commitContent strings.Builder
	commitContent.WriteString(fmt.Sprintf("tree %s\n", treeHash))
	
	for _, parentID := range parentIDs {
		commitContent.WriteString(fmt.Sprintf("parent %s\n", parentID))
	}
	
	commitContent.WriteString(fmt.Sprintf("author %s %d\n", authorID, timestamp.Unix()))
	commitContent.WriteString(fmt.Sprintf("project %s\n", projectID))
	commitContent.WriteString("\n")
	commitContent.WriteString(message)

	hash := sha1.Sum([]byte(commitContent.String()))
	return hex.EncodeToString(hash[:])
}

// CommitDiff represents the differences between two commits
type CommitDiff struct {
	FromCommit models.Commit `json:"from_commit"`
	ToCommit   models.Commit `json:"to_commit"`
	Changes    []FileDiff    `json:"changes"`
}

// FileDiff represents a file change between commits
type FileDiff struct {
	Path       string                  `json:"path"`
	ChangeType string                  `json:"change_type"` // added, deleted, modified
	FromFile   models.CommitTreeFile   `json:"from_file"`
	ToFile     models.CommitTreeFile   `json:"to_file"`
}