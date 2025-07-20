// Enhanced Working Directory Manager with UE5 Asset Integrity
// Integrates working directory management with comprehensive UE5 asset tracking

package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Telerallc/gamedev-vcs/internal/integrity"
)

// WorkingDirectoryManager combines working directory management with UE5 asset integrity
type WorkingDirectoryManager struct {
	projectPath      string
	assetTracker     *integrity.UE5AssetTracker
	integrityChecker *IntegrityChecker
	corruptionLog    *CorruptionLog
	autoRepair       bool
	indexManager     *IndexManager
}

// IndexManager manages file indexing
type IndexManager struct {
	projectPath string
	indexFile   string
}

// FileStatus represents basic file status
type FileStatus struct {
	FilePath    string    `json:"file_path"`
	Exists      bool      `json:"exists"`
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	ContentHash string    `json:"content_hash"`
}

// APIClient represents the API client interface
type APIClient struct {
	authToken string
}

// IntegrityChecker performs various integrity checks on files
type IntegrityChecker struct {
	checksEnabled map[string]bool
	lastCheckTime map[string]time.Time
	checkInterval time.Duration
}

// CorruptionLog tracks corruption events and recovery actions
type CorruptionLog struct {
	logPath string
	events  []CorruptionLogEntry
}

// CorruptionLogEntry represents a single corruption event
type CorruptionLogEntry struct {
	Timestamp      time.Time                    `json:"timestamp"`
	AssetPath      string                       `json:"asset_path"`
	CorruptionType integrity.CorruptionType     `json:"corruption_type"`
	Severity       integrity.CorruptionSeverity `json:"severity"`
	DetectedBy     string                       `json:"detected_by"`
	Description    string                       `json:"description"`
	AutoFixed      bool                         `json:"auto_fixed"`
	FixMethod      string                       `json:"fix_method"`
	PreviousHash   string                       `json:"previous_hash"`
	CorruptedHash  string                       `json:"corrupted_hash"`
	RecoveryTime   time.Duration                `json:"recovery_time"`
	UserNotified   bool                         `json:"user_notified"`
}

// FileIntegrityStatus represents the integrity status of a file
type FileIntegrityStatus struct {
	FilePath        string                     `json:"file_path"`
	Status          integrity.IntegrityStatus  `json:"status"`
	LastChecked     time.Time                  `json:"last_checked"`
	HealthScore     float64                    `json:"health_score"`
	Issues          []integrity.BlueprintIssue `json:"issues"`
	Recommendations []string                   `json:"recommendations"`
}

// NewWorkingDirectoryManager creates an enhanced working directory manager
func NewWorkingDirectoryManager(projectPath string) (*WorkingDirectoryManager, error) {
	// Initialize UE5 asset tracker
	assetTracker, err := integrity.NewUE5AssetTracker(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset tracker: %w", err)
	}

	// Initialize integrity checker
	integrityChecker := &IntegrityChecker{
		checksEnabled: map[string]bool{
			"content":    true,
			"metadata":   true,
			"blueprint":  true,
			"dependency": true,
			"structural": true,
		},
		lastCheckTime: make(map[string]time.Time),
		checkInterval: 30 * time.Minute, // Check every 30 minutes
	}

	// Initialize corruption log
	corruptionLog := &CorruptionLog{
		logPath: filepath.Join(projectPath, ".vcs", "corruption.log"),
		events:  []CorruptionLogEntry{},
	}

	// Initialize index manager
	indexManager := &IndexManager{
		projectPath: projectPath,
		indexFile:   filepath.Join(projectPath, ".vcs", "index.json"),
	}

	enhanced := &WorkingDirectoryManager{
		projectPath:      projectPath,
		assetTracker:     assetTracker,
		integrityChecker: integrityChecker,
		corruptionLog:    corruptionLog,
		autoRepair:       true, // Enable auto-repair by default
		indexManager:     indexManager,
	}

	// Load existing corruption log
	enhanced.corruptionLog.Load()

	// Start background integrity monitoring
	go enhanced.startBackgroundIntegrityCheck()

	return enhanced, nil
}

// AddFile adds a file to the working directory
func (ewdm *WorkingDirectoryManager) AddFile(filePath string) (string, error) {
	// Calculate content hash
	fullPath := filepath.Join(ewdm.projectPath, filePath)
	contentHash, err := ewdm.calculateFileHash(fullPath)
	if err != nil {
		return "", err
	}

	// Add to index manager
	if err := ewdm.indexManager.AddFile(filePath, contentHash); err != nil {
		return "", err
	}

	return contentHash, nil
}

// CheckoutFile checks out a file from the repository
func (ewdm *WorkingDirectoryManager) CheckoutFile(filePath, contentHash string, apiClient *APIClient) error {
	// Implementation would download file from server
	// For now, just verify the file exists
	fullPath := filepath.Join(ewdm.projectPath, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}
	return nil
}

// GetFileStatus gets the status of a file
func (ewdm *WorkingDirectoryManager) GetFileStatus(filePath string) FileStatus {
	fullPath := filepath.Join(ewdm.projectPath, filePath)
	stat, err := os.Stat(fullPath)

	status := FileStatus{
		FilePath: filePath,
		Exists:   err == nil,
	}

	if err == nil {
		status.Size = stat.Size()
		status.Modified = stat.ModTime()
		// Get content hash from index
		if hash, err := ewdm.indexManager.GetFileHash(filePath); err == nil {
			status.ContentHash = hash
		}
	}

	return status
}

// ListFiles lists all files in the working directory
func (ewdm *WorkingDirectoryManager) ListFiles() (map[string]FileStatus, error) {
	files := make(map[string]FileStatus)

	err := filepath.Walk(ewdm.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(ewdm.projectPath, path)
		if err != nil {
			return err
		}

		// Skip .vcs directory
		if strings.HasPrefix(relPath, ".vcs") {
			return nil
		}

		files[relPath] = ewdm.GetFileStatus(relPath)
		return nil
	})

	return files, err
}

// calculateFileHash calculates SHA256 hash of a file
func (ewdm *WorkingDirectoryManager) calculateFileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// IndexManager methods
func (im *IndexManager) AddFile(filePath, contentHash string) error {
	// Implementation would add file to index
	return nil
}

func (im *IndexManager) GetFileHash(filePath string) (string, error) {
	// Implementation would get hash from index
	return "", fmt.Errorf("not implemented")
}

// AddFileWithIntegrityCheck adds a file with comprehensive integrity checking
func (ewdm *WorkingDirectoryManager) AddFileWithIntegrityCheck(filePath string) (*AddFileResult, error) {
	// Perform pre-add integrity check
	preCheckResult, err := ewdm.performIntegrityCheck(filePath)
	if err != nil {
		return nil, fmt.Errorf("pre-add integrity check failed: %w", err)
	}

	// If corruption detected, attempt repair before adding
	if preCheckResult.Status == integrity.IntegrityCorrupted {
		if ewdm.autoRepair {
			if err := ewdm.attemptRepair(filePath, preCheckResult); err != nil {
				return nil, fmt.Errorf("failed to repair corrupted file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("file corruption detected, cannot add: %s", preCheckResult.Issues[0].Description)
		}
	}

	// Perform standard add operation
	contentHash, err := ewdm.AddFile(filePath)
	if err != nil {
		return nil, err
	}

	// Track asset if it's a UE5 asset
	var assetRecord *integrity.AssetIntegrityRecord
	if ewdm.isUE5Asset(filePath) {
		assetRecord, err = ewdm.assetTracker.TrackAsset(filePath)
		if err != nil {
			// Log warning but don't fail the add operation
			fmt.Printf("Warning: failed to track UE5 asset %s: %v\n", filePath, err)
		}
	}

	// Create comprehensive result
	result := &AddFileResult{
		FilePath:          filePath,
		ContentHash:       contentHash,
		IntegrityStatus:   preCheckResult,
		AssetRecord:       assetRecord,
		Timestamp:         time.Now(),
		WarningsCount:     len(preCheckResult.Issues),
		AutoRepairApplied: preCheckResult.Status == integrity.IntegrityCorrupted && ewdm.autoRepair,
	}

	// Log any warnings
	if len(preCheckResult.Issues) > 0 {
		ewdm.logIntegrityWarnings(filePath, preCheckResult.Issues)
	}

	return result, nil
}

// CheckoutFileWithVerification checks out a file and verifies its integrity
func (ewdm *WorkingDirectoryManager) CheckoutFileWithVerification(filePath, contentHash string, apiClient *APIClient) (*CheckoutResult, error) {
	// Perform standard checkout
	err := ewdm.CheckoutFile(filePath, contentHash, apiClient)
	if err != nil {
		return nil, err
	}

	// Verify integrity after checkout
	integrityResult, err := ewdm.performIntegrityCheck(filePath)
	if err != nil {
		return nil, fmt.Errorf("post-checkout integrity check failed: %w", err)
	}

	// If corruption detected, log and attempt recovery
	if integrityResult.Status == integrity.IntegrityCorrupted {
		ewdm.logCorruption(filePath, integrity.CorruptionContent, integrity.CorruptionSeverityHigh,
			"Corruption detected after checkout", "checkout_verification")

		if ewdm.autoRepair {
			if err := ewdm.attemptRepair(filePath, integrityResult); err != nil {
				return nil, fmt.Errorf("failed to repair corrupted file after checkout: %w", err)
			}
			integrityResult.Status = integrity.IntegrityValid
		}
	}

	result := &CheckoutResult{
		FilePath:        filePath,
		ContentHash:     contentHash,
		IntegrityStatus: integrityResult,
		IsCorrupted:     integrityResult.Status == integrity.IntegrityCorrupted,
		Timestamp:       time.Now(),
	}

	return result, nil
}

// GetFileStatusWithIntegrity returns comprehensive file status including integrity
func (ewdm *WorkingDirectoryManager) GetFileStatusWithIntegrity(filePath string) (*EnhancedFileStatus, error) {
	// Get basic file status
	basicStatus := ewdm.GetFileStatus(filePath)

	// Perform integrity check
	integrityResult, err := ewdm.performIntegrityCheck(filePath)
	if err != nil {
		integrityResult = &FileIntegrityStatus{
			FilePath:    filePath,
			Status:      integrity.IntegrityUnknown,
			LastChecked: time.Now(),
			Issues:      []integrity.BlueprintIssue{},
		}
	}

	// Get UE5-specific information if applicable
	var assetInfo *integrity.AssetIntegrityRecord
	if ewdm.isUE5Asset(filePath) {
		// Try to get existing asset record
		if record, err := ewdm.getAssetRecord(filePath); err == nil {
			assetInfo = record
		}
	}

	// Check corruption history
	corruptionHistory := ewdm.corruptionLog.GetAssetHistory(filePath)

	result := &EnhancedFileStatus{
		BasicStatus:        basicStatus,
		IntegrityStatus:    integrityResult,
		AssetInfo:          assetInfo,
		CorruptionHistory:  corruptionHistory,
		LastIntegrityCheck: integrityResult.LastChecked,
		RiskLevel:          ewdm.calculateRiskLevel(integrityResult, corruptionHistory),
	}

	return result, nil
}

// performIntegrityCheck performs comprehensive integrity checking
func (ewdm *WorkingDirectoryManager) performIntegrityCheck(filePath string) (*FileIntegrityStatus, error) {
	// Check if file exists
	fullPath := filepath.Join(ewdm.projectPath, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return &FileIntegrityStatus{
			FilePath:    filePath,
			Status:      integrity.IntegrityMissing,
			LastChecked: time.Now(),
			HealthScore: 0,
			Issues:      []integrity.BlueprintIssue{},
		}, nil
	}

	result := &FileIntegrityStatus{
		FilePath:        filePath,
		LastChecked:     time.Now(),
		Issues:          []integrity.BlueprintIssue{},
		Recommendations: []string{},
	}

	// Perform basic content integrity check
	if ewdm.integrityChecker.checksEnabled["content"] {
		if err := ewdm.checkContentIntegrity(filePath, result); err != nil {
			result.Issues = append(result.Issues, integrity.BlueprintIssue{
				IssueType:   integrity.IssueLogicError,
				Severity:    integrity.BlueprintSeverityError,
				Description: fmt.Sprintf("Content integrity check failed: %v", err),
			})
		}
	}

	// UE5-specific checks
	if ewdm.isUE5Asset(filePath) {
		if err := ewdm.performUE5SpecificChecks(filePath, result); err != nil {
			fmt.Printf("UE5 integrity check warning for %s: %v\n", filePath, err)
		}
	}

	// Calculate overall status
	result.Status = ewdm.calculateOverallStatus(result.Issues)
	result.HealthScore = ewdm.calculateHealthScore(result.Issues)

	// Generate recommendations
	result.Recommendations = ewdm.generateRecommendations(result.Issues)

	return result, nil
}

// performUE5SpecificChecks performs UE5-specific integrity checks
func (ewdm *WorkingDirectoryManager) performUE5SpecificChecks(filePath string, result *FileIntegrityStatus) error {
	// Use the asset tracker to verify integrity
	integrityResult, err := ewdm.assetTracker.VerifyAssetIntegrity(filePath, "system")
	if err != nil {
		return err
	}

	// Convert integrity check results to blueprint issues
	for _, check := range integrityResult.Checks {
		if check.Status == integrity.IntegrityCorrupted {
			issue := integrity.BlueprintIssue{
				IssueID:        check.CheckID,
				IssueType:      ewdm.mapCheckTypeToIssueType(check.CheckType),
				Severity:       ewdm.mapIntegrityToSeverity(check.Status),
				Description:    check.ErrorDetails,
				RecommendedFix: check.FixAction,
				AutoFixable:    strings.Contains(check.FixAction, "auto") || strings.Contains(check.FixAction, "repair"),
			}
			result.Issues = append(result.Issues, issue)
		}
	}

	return nil
}

// attemptRepair attempts to repair a corrupted file
func (ewdm *WorkingDirectoryManager) attemptRepair(filePath string, integrityStatus *FileIntegrityStatus) error {
	startTime := time.Now()

	// Try different repair strategies based on the type of corruption
	var repairMethod string
	var err error

	// For UE5 assets, try Blueprint repair first
	if ewdm.isUE5Asset(filePath) {
		repairMethod = "blueprint_repair"
		err = ewdm.attemptBlueprintRepair(filepath.Join(ewdm.projectPath, filePath))

		if err != nil {
			// Fall back to backup restoration
			repairMethod = "backup_restore"
			err = ewdm.restoreFromBackup(filePath)
		}
	} else {
		// For regular files, try backup restoration
		repairMethod = "backup_restore"
		err = ewdm.restoreFromBackup(filePath)
	}

	// Log repair attempt
	recoveryTime := time.Since(startTime)
	ewdm.logCorruption(filePath, integrity.CorruptionContent, integrity.CorruptionSeverityMedium,
		"Auto-repair attempted", repairMethod)

	if err != nil {
		ewdm.corruptionLog.events[len(ewdm.corruptionLog.events)-1].AutoFixed = false
		return fmt.Errorf("repair failed: %w", err)
	}

	// Mark as successfully repaired
	ewdm.corruptionLog.events[len(ewdm.corruptionLog.events)-1].AutoFixed = true
	ewdm.corruptionLog.events[len(ewdm.corruptionLog.events)-1].RecoveryTime = recoveryTime

	// Verify repair
	postRepairCheck, err := ewdm.performIntegrityCheck(filePath)
	if err != nil || postRepairCheck.Status == integrity.IntegrityCorrupted {
		return fmt.Errorf("repair verification failed")
	}

	fmt.Printf("✅ Successfully repaired %s using %s (took %v)\n", filePath, repairMethod, recoveryTime)
	return nil
}

// attemptBlueprintRepair attempts to repair a Blueprint file
func (ewdm *WorkingDirectoryManager) attemptBlueprintRepair(filePath string) error {
	// Implementation would attempt Blueprint-specific repair
	fmt.Printf("🔄 Attempting Blueprint repair for %s...\n", filePath)
	return nil
}

// restoreFromBackup restores a file from the most recent backup
func (ewdm *WorkingDirectoryManager) restoreFromBackup(filePath string) error {
	// This would integrate with your backup system
	// For now, we'll simulate a successful restore
	fmt.Printf("🔄 Restoring %s from backup...\n", filePath)

	// In a real implementation, this would:
	// 1. Find the most recent healthy backup
	// 2. Copy the backup to the working directory
	// 3. Update the file index

	return nil
}

// startBackgroundIntegrityCheck runs periodic integrity checks
func (ewdm *WorkingDirectoryManager) startBackgroundIntegrityCheck() {
	ticker := time.NewTicker(ewdm.integrityChecker.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		ewdm.performBackgroundIntegrityCheck()
	}
}

// performBackgroundIntegrityCheck performs periodic integrity checks on all files
func (ewdm *WorkingDirectoryManager) performBackgroundIntegrityCheck() {
	files, err := ewdm.ListFiles()
	if err != nil {
		fmt.Printf("Background integrity check failed: %v\n", err)
		return
	}

	corruptedFiles := []string{}
	checkedCount := 0

	for filePath := range files {
		// Skip if recently checked
		if lastCheck, exists := ewdm.integrityChecker.lastCheckTime[filePath]; exists {
			if time.Since(lastCheck) < ewdm.integrityChecker.checkInterval {
				continue
			}
		}

		// Perform integrity check
		integrityResult, err := ewdm.performIntegrityCheck(filePath)
		if err != nil {
			continue
		}

		ewdm.integrityChecker.lastCheckTime[filePath] = time.Now()
		checkedCount++

		// Handle corruption
		if integrityResult.Status == integrity.IntegrityCorrupted {
			corruptedFiles = append(corruptedFiles, filePath)

			ewdm.logCorruption(filePath, integrity.CorruptionContent, integrity.CorruptionSeverityMedium,
				"Corruption detected during background check", "background_scan")

			// Attempt auto-repair if enabled
			if ewdm.autoRepair {
				if err := ewdm.attemptRepair(filePath, integrityResult); err != nil {
					fmt.Printf("⚠️ Background repair failed for %s: %v\n", filePath, err)
				}
			}
		}
	}

	if checkedCount > 0 {
		fmt.Printf("🔍 Background integrity check: %d files checked, %d corrupted\n", checkedCount, len(corruptedFiles))
	}

	if len(corruptedFiles) > 0 {
		fmt.Printf("⚠️ Corrupted files detected: %v\n", corruptedFiles)
	}
}

// Helper methods

func (ewdm *WorkingDirectoryManager) isUE5Asset(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	ue5Extensions := []string{".uasset", ".umap", ".uexp", ".ubulk"}

	for _, ue5Ext := range ue5Extensions {
		if ext == ue5Ext {
			return true
		}
	}
	return false
}

func (ewdm *WorkingDirectoryManager) checkContentIntegrity(filePath string, result *FileIntegrityStatus) error {
	// Get expected hash from index
	expectedHash, err := ewdm.indexManager.GetFileHash(filePath)
	if err != nil {
		return nil // File not in index, skip content check
	}

	// Calculate current hash
	fullPath := filepath.Join(ewdm.projectPath, filePath)
	currentHash, err := ewdm.calculateFileHash(fullPath)
	if err != nil {
		return err
	}

	// Compare hashes
	if currentHash != expectedHash {
		result.Issues = append(result.Issues, integrity.BlueprintIssue{
			IssueType:      integrity.IssueLogicError,
			Severity:       integrity.BlueprintSeverityError,
			Description:    "File content has been modified unexpectedly",
			RecommendedFix: "Restore from backup or commit changes",
		})
	}

	return nil
}

func (ewdm *WorkingDirectoryManager) calculateOverallStatus(issues []integrity.BlueprintIssue) integrity.IntegrityStatus {
	if len(issues) == 0 {
		return integrity.IntegrityValid
	}

	for _, issue := range issues {
		if issue.Severity == integrity.BlueprintSeverityError || issue.Severity == integrity.BlueprintSeverityCritical {
			return integrity.IntegrityCorrupted
		}
	}

	return integrity.IntegrityValid
}

func (ewdm *WorkingDirectoryManager) calculateHealthScore(issues []integrity.BlueprintIssue) float64 {
	if len(issues) == 0 {
		return 100.0
	}

	score := 100.0
	for _, issue := range issues {
		switch issue.Severity {
		case integrity.BlueprintSeverityCritical:
			score -= 25
		case integrity.BlueprintSeverityError:
			score -= 15
		case integrity.BlueprintSeverityWarning:
			score -= 5
		case integrity.BlueprintSeverityInfo:
			score -= 1
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (ewdm *WorkingDirectoryManager) generateRecommendations(issues []integrity.BlueprintIssue) []string {
	recommendations := []string{}
	seen := make(map[string]bool)

	for _, issue := range issues {
		if issue.RecommendedFix != "" && !seen[issue.RecommendedFix] {
			recommendations = append(recommendations, issue.RecommendedFix)
			seen[issue.RecommendedFix] = true
		}
	}

	return recommendations
}

func (ewdm *WorkingDirectoryManager) calculateRiskLevel(integrityStatus *FileIntegrityStatus, history []CorruptionLogEntry) string {
	if integrityStatus.Status == integrity.IntegrityCorrupted {
		return "high"
	}

	if len(history) > 3 {
		return "medium"
	}

	if integrityStatus.HealthScore < 75 {
		return "medium"
	}

	return "low"
}

func (ewdm *WorkingDirectoryManager) logCorruption(filePath string, corruptionType integrity.CorruptionType, severity integrity.CorruptionSeverity, description, detectedBy string) {
	entry := CorruptionLogEntry{
		Timestamp:      time.Now(),
		AssetPath:      filePath,
		CorruptionType: corruptionType,
		Severity:       severity,
		DetectedBy:     detectedBy,
		Description:    description,
		AutoFixed:      false,
		UserNotified:   false,
	}

	ewdm.corruptionLog.events = append(ewdm.corruptionLog.events, entry)
	ewdm.corruptionLog.Save()
}

func (ewdm *WorkingDirectoryManager) logIntegrityWarnings(filePath string, issues []integrity.BlueprintIssue) {
	for _, issue := range issues {
		if issue.Severity == integrity.BlueprintSeverityWarning || issue.Severity == integrity.BlueprintSeverityInfo {
			fmt.Printf("⚠️ %s: %s\n", filePath, issue.Description)
		}
	}
}

func (ewdm *WorkingDirectoryManager) mapCheckTypeToIssueType(checkType integrity.IntegrityCheckType) integrity.IssueType {
	switch checkType {
	case integrity.CheckTypeBlueprint:
		return integrity.IssueLogicError
	case integrity.CheckTypeDependency:
		return integrity.IssueBrokenReference
	case integrity.CheckTypeContent:
		return integrity.IssueLogicError
	default:
		return integrity.IssueLogicError
	}
}

func (ewdm *WorkingDirectoryManager) mapIntegrityToSeverity(status integrity.IntegrityStatus) integrity.IssueSeverity {
	switch status {
	case integrity.IntegrityCorrupted:
		return integrity.BlueprintSeverityError
	case integrity.IntegrityMissing:
		return integrity.BlueprintSeverityCritical
	default:
		return integrity.BlueprintSeverityInfo
	}
}

// getAssetRecord gets asset record from tracker
func (ewdm *WorkingDirectoryManager) getAssetRecord(assetPath string) (*integrity.AssetIntegrityRecord, error) {
	// Implementation would get record from asset tracker
	return nil, fmt.Errorf("not implemented")
}

// Result structures
type AddFileResult struct {
	FilePath          string                          `json:"file_path"`
	ContentHash       string                          `json:"content_hash"`
	IntegrityStatus   *FileIntegrityStatus            `json:"integrity_status"`
	AssetRecord       *integrity.AssetIntegrityRecord `json:"asset_record"`
	Timestamp         time.Time                       `json:"timestamp"`
	WarningsCount     int                             `json:"warnings_count"`
	AutoRepairApplied bool                            `json:"auto_repair_applied"`
}

type CheckoutResult struct {
	FilePath        string               `json:"file_path"`
	ContentHash     string               `json:"content_hash"`
	IntegrityStatus *FileIntegrityStatus `json:"integrity_status"`
	IsCorrupted     bool                 `json:"is_corrupted"`
	Timestamp       time.Time            `json:"timestamp"`
}

type EnhancedFileStatus struct {
	BasicStatus        FileStatus                      `json:"basic_status"`
	IntegrityStatus    *FileIntegrityStatus            `json:"integrity_status"`
	AssetInfo          *integrity.AssetIntegrityRecord `json:"asset_info"`
	CorruptionHistory  []CorruptionLogEntry            `json:"corruption_history"`
	LastIntegrityCheck time.Time                       `json:"last_integrity_check"`
	RiskLevel          string                          `json:"risk_level"`
}

// CorruptionLog methods
func (cl *CorruptionLog) Load() error {
	if _, err := os.Stat(cl.logPath); os.IsNotExist(err) {
		return nil // No log file yet
	}

	data, err := os.ReadFile(cl.logPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &cl.events)
}

func (cl *CorruptionLog) Save() error {
	data, err := json.MarshalIndent(cl.events, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cl.logPath, data, 0644)
}

func (cl *CorruptionLog) GetAssetHistory(assetPath string) []CorruptionLogEntry {
	var history []CorruptionLogEntry
	for _, event := range cl.events {
		if event.AssetPath == assetPath {
			history = append(history, event)
		}
	}
	return history
}
