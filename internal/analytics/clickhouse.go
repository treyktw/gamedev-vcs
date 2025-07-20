package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// AnalyticsClient handles ClickHouse operations for team analytics
type AnalyticsClient struct {
	conn driver.Conn
	ctx  context.Context
}

// Commit represents a VCS commit for analytics
type Commit struct {
	Hash         string            `json:"hash"`
	Author       string            `json:"author"`
	CommitTime   time.Time         `json:"commit_time"`
	Branch       string            `json:"branch"`
	Message      string            `json:"message"`
	Project      string            `json:"project"`
	ParentHashes []string          `json:"parent_hashes"`
	TotalFiles   uint32            `json:"total_files"`
	TotalSize    uint64            `json:"total_size"`
	Metadata     map[string]string `json:"metadata"`
}

// FileChange represents a file modification in a commit
type FileChange struct {
	CommitHash       string    `json:"commit_hash"`
	FilePath         string    `json:"file_path"`
	ChangeType       string    `json:"change_type"` // added, modified, deleted, moved
	ContentHash      string    `json:"content_hash"`
	FileSizeBytes    uint64    `json:"file_size_bytes"`
	Author           string    `json:"author"`
	CommitTime       time.Time `json:"commit_time"`
	AssetType        string    `json:"asset_type"`
	AssetClass       string    `json:"asset_class"`
	UE5PackagePath   string    `json:"ue5_package_path"`
	IsBlueprint      bool      `json:"is_blueprint"`
	BlueprintType    string    `json:"blueprint_type"`
	SyncDurationMS   uint32    `json:"sync_duration_ms"`
	CompressionRatio float32   `json:"compression_ratio"`
}

// AssetDependency represents dependency relationships between assets
type AssetDependency struct {
	AssetPath        string    `json:"asset_path"`
	DependsOnPath    string    `json:"depends_on_path"`
	DependencyType   string    `json:"dependency_type"` // HardReference, SoftReference, SearchableReference
	DiscoveredTime   time.Time `json:"discovered_time"`
	CommitHash       string    `json:"commit_hash"`
	IsCircular       bool      `json:"is_circular"`
	DependencyWeight float32   `json:"dependency_weight"`
}

// CollaborationEvent represents real-time collaboration analytics
type CollaborationEvent struct {
	EventID        string            `json:"event_id"`
	EventType      string            `json:"event_type"`
	UserName       string            `json:"user_name"`
	FilePath       string            `json:"file_path"`
	Project        string            `json:"project"`
	EventTime      time.Time         `json:"event_time"`
	SessionID      string            `json:"session_id"`
	AdditionalData map[string]string `json:"additional_data"`
}

// ProductivityMetrics represents team productivity data
type ProductivityMetrics struct {
	Date               time.Time `json:"date"`
	Author             string    `json:"author"`
	Project            string    `json:"project"`
	CommitsCount       uint32    `json:"commits_count"`
	FilesChanged       uint32    `json:"files_changed"`
	LinesOfCodeAdded   uint32    `json:"lines_of_code_added"`
	LinesOfCodeRemoved uint32    `json:"lines_of_code_removed"`
	AssetsCreated      uint32    `json:"assets_created"`
	AssetsModified     uint32    `json:"assets_modified"`
	TimeActiveMinutes  uint32    `json:"time_active_minutes"`
	ConflictsResolved  uint32    `json:"conflicts_resolved"`
	BuildsTriggered    uint32    `json:"builds_triggered"`
	TestsRun           uint32    `json:"tests_run"`
}

// TeamInsights represents aggregated team analytics
type TeamInsights struct {
	Project              string             `json:"project"`
	Period               string             `json:"period"`
	TotalCommits         uint64             `json:"total_commits"`
	TotalFilesChanged    uint64             `json:"total_files_changed"`
	TotalSizeProcessed   uint64             `json:"total_size_processed"`
	ActiveDevelopers     uint32             `json:"active_developers"`
	TopContributors      []ContributorStats `json:"top_contributors"`
	HottestAssets        []AssetActivity    `json:"hottest_assets"`
	DependencyComplexity DependencyStats    `json:"dependency_complexity"`
	CollaborationScore   float64            `json:"collaboration_score"`
}

// ContributorStats represents individual contributor metrics
type ContributorStats struct {
	Author       string  `json:"author"`
	Commits      uint64  `json:"commits"`
	FilesChanged uint64  `json:"files_changed"`
	Velocity     float64 `json:"velocity"`
}

// AssetActivity represents asset modification frequency
type AssetActivity struct {
	FilePath        string    `json:"file_path"`
	ChangeFrequency uint64    `json:"change_frequency"`
	Contributors    uint32    `json:"unique_contributors"`
	LastModified    time.Time `json:"last_modified"`
}

// DependencyStats represents dependency complexity metrics
type DependencyStats struct {
	TotalDependencies    uint64  `json:"total_dependencies"`
	CircularDependencies uint64  `json:"circular_dependencies"`
	MaxDepth             uint32  `json:"max_depth"`
	AverageConnections   float64 `json:"average_connections"`
}

// NewAnalyticsClient creates a new ClickHouse analytics client
func NewAnalyticsClient(dsn string) (*AnalyticsClient, error) {
	// Connect using the vcs_user that's configured in the container
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{dsn},
		Auth: clickhouse.Auth{
			Database: "", // Don't specify database initially
			Username: "vcs_user",
			Password: "dev_password",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:     time.Duration(10) * time.Second,
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Duration(10) * time.Minute,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	ctx := context.Background()
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	// Create database if it doesn't exist
	if err := conn.Exec(ctx, "CREATE DATABASE IF NOT EXISTS vcs_analytics"); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Switch to using the vcs_analytics database
	if err := conn.Exec(ctx, "USE vcs_analytics"); err != nil {
		return nil, fmt.Errorf("failed to use vcs_analytics database: %w", err)
	}

	client := &AnalyticsClient{
		conn: conn,
		ctx:  ctx,
	}

	// Initialize schema
	if err := client.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return client, nil
}

// Commit Operations

// RecordCommit stores a commit in the analytics database
func (ac *AnalyticsClient) RecordCommit(commit *Commit) error {
	query := `
		INSERT INTO commits (
			commit_hash, author, commit_time, branch, message, project, 
			parent_hashes, total_files, total_size_bytes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	return ac.conn.Exec(ac.ctx, query,
		commit.Hash,
		commit.Author,
		commit.CommitTime,
		commit.Branch,
		commit.Message,
		commit.Project,
		commit.ParentHashes,
		commit.TotalFiles,
		commit.TotalSize,
	)
}

// RecordFileChanges stores file changes for a commit
func (ac *AnalyticsClient) RecordFileChanges(changes []FileChange) error {
	if len(changes) == 0 {
		return nil
	}

	batch, err := ac.conn.PrepareBatch(ac.ctx, `
		INSERT INTO file_changes (
			commit_hash, file_path, change_type, content_hash, file_size_bytes,
			author, commit_time, asset_type, asset_class, ue5_package_path,
			is_blueprint, blueprint_type, sync_duration_ms, compression_ratio
		)`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, change := range changes {
		if err := batch.Append(
			change.CommitHash,
			change.FilePath,
			change.ChangeType,
			change.ContentHash,
			change.FileSizeBytes,
			change.Author,
			change.CommitTime,
			change.AssetType,
			change.AssetClass,
			change.UE5PackagePath,
			change.IsBlueprint,
			change.BlueprintType,
			change.SyncDurationMS,
			change.CompressionRatio,
		); err != nil {
			return fmt.Errorf("failed to append change: %w", err)
		}
	}

	return batch.Send()
}

// Asset Dependency Operations

// RecordAssetDependencies stores asset dependency relationships
func (ac *AnalyticsClient) RecordAssetDependencies(dependencies []AssetDependency) error {
	if len(dependencies) == 0 {
		return nil
	}

	batch, err := ac.conn.PrepareBatch(ac.ctx, `
		INSERT INTO asset_dependencies (
			asset_path, depends_on_path, dependency_type, discovered_time,
			commit_hash, is_circular, dependency_weight
		)`)
	if err != nil {
		return fmt.Errorf("failed to prepare dependency batch: %w", err)
	}

	for _, dep := range dependencies {
		if err := batch.Append(
			dep.AssetPath,
			dep.DependsOnPath,
			dep.DependencyType,
			dep.DiscoveredTime,
			dep.CommitHash,
			dep.IsCircular,
			dep.DependencyWeight,
		); err != nil {
			return fmt.Errorf("failed to append dependency: %w", err)
		}
	}

	return batch.Send()
}

// GetAssetDependencies retrieves dependencies for an asset
func (ac *AnalyticsClient) GetAssetDependencies(assetPath string) ([]AssetDependency, error) {
	query := `
		SELECT asset_path, depends_on_path, dependency_type, discovered_time,
			   commit_hash, is_circular, dependency_weight
		FROM vcs_analytics.asset_dependencies
		WHERE asset_path = ?
		ORDER BY discovered_time DESC`

	rows, err := ac.conn.Query(ac.ctx, query, assetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependencies: %w", err)
	}
	defer rows.Close()

	var dependencies []AssetDependency
	for rows.Next() {
		var dep AssetDependency
		if err := rows.Scan(
			&dep.AssetPath,
			&dep.DependsOnPath,
			&dep.DependencyType,
			&dep.DiscoveredTime,
			&dep.CommitHash,
			&dep.IsCircular,
			&dep.DependencyWeight,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dependency: %w", err)
		}
		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}

// GetDependencyImpact finds all assets that depend on a given asset
func (ac *AnalyticsClient) GetDependencyImpact(assetPath string) ([]AssetDependency, error) {
	query := `
		SELECT asset_path, depends_on_path, dependency_type, discovered_time,
			   commit_hash, is_circular, dependency_weight
		FROM asset_dependencies
		WHERE depends_on_path = ?
		ORDER BY dependency_weight DESC`

	rows, err := ac.conn.Query(ac.ctx, query, assetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependency impact: %w", err)
	}
	defer rows.Close()

	var dependencies []AssetDependency
	for rows.Next() {
		var dep AssetDependency
		if err := rows.Scan(
			&dep.AssetPath,
			&dep.DependsOnPath,
			&dep.DependencyType,
			&dep.DiscoveredTime,
			&dep.CommitHash,
			&dep.IsCircular,
			&dep.DependencyWeight,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dependency impact: %w", err)
		}
		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}

// Team Analytics

// GetTeamProductivity retrieves productivity metrics for a team
func (ac *AnalyticsClient) GetTeamProductivity(project string, days int) ([]ProductivityMetrics, error) {
	query := `
		SELECT date, author, project, commits_count, files_changed,
			   lines_of_code_added, lines_of_code_removed, assets_created,
			   assets_modified, time_active_minutes, conflicts_resolved
		FROM vcs_analytics.productivity_metrics
		WHERE project = ? AND date >= today() - INTERVAL ? DAY
		ORDER BY date DESC, commits_count DESC`

	rows, err := ac.conn.Query(ac.ctx, query, project, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query productivity: %w", err)
	}
	defer rows.Close()

	var metrics []ProductivityMetrics
	for rows.Next() {
		var pm ProductivityMetrics
		if err := rows.Scan(
			&pm.Date,
			&pm.Author,
			&pm.Project,
			&pm.CommitsCount,
			&pm.FilesChanged,
			&pm.LinesOfCodeAdded,
			&pm.LinesOfCodeRemoved,
			&pm.AssetsCreated,
			&pm.AssetsModified,
			&pm.TimeActiveMinutes,
			&pm.ConflictsResolved,
		); err != nil {
			return nil, fmt.Errorf("failed to scan productivity: %w", err)
		}
		metrics = append(metrics, pm)
	}

	return metrics, nil
}

// GetTeamInsights provides comprehensive team analytics
func (ac *AnalyticsClient) GetTeamInsights(project string, days int) (*TeamInsights, error) {
	insights := &TeamInsights{
		Project: project,
		Period:  fmt.Sprintf("Last %d days", days),
	}

	// Get overall stats
	statsQuery := `
		SELECT 
			count() as total_commits,
			sum(total_files) as total_files_changed,
			sum(total_size_bytes) as total_size_processed,
			uniq(author) as active_developers
		FROM commits 
		WHERE project = ? AND commit_time >= now() - INTERVAL ? DAY`

	row := ac.conn.QueryRow(ac.ctx, statsQuery, project, days)
	if err := row.Scan(
		&insights.TotalCommits,
		&insights.TotalFilesChanged,
		&insights.TotalSizeProcessed,
		&insights.ActiveDevelopers,
	); err != nil {
		return nil, fmt.Errorf("failed to get team stats: %w", err)
	}

	// Get top contributors
	contributorsQuery := `
		SELECT 
			author,
			count() as commits,
			sum(total_files) as files_changed,
			commits / ? as velocity
		FROM commits
		WHERE project = ? AND commit_time >= now() - INTERVAL ? DAY
		GROUP BY author
		ORDER BY commits DESC
		LIMIT 10`

	rows, err := ac.conn.Query(ac.ctx, contributorsQuery, float64(days), project, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query contributors: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var contributor ContributorStats
		if err := rows.Scan(
			&contributor.Author,
			&contributor.Commits,
			&contributor.FilesChanged,
			&contributor.Velocity,
		); err != nil {
			continue
		}
		insights.TopContributors = append(insights.TopContributors, contributor)
	}

	// Get hottest assets
	hotAssetsQuery := `
		SELECT 
			file_path,
			count() as change_frequency,
			uniq(author) as unique_contributors,
			max(commit_time) as last_modified
		FROM file_changes
		WHERE commit_time >= now() - INTERVAL ? DAY
		GROUP BY file_path
		HAVING change_frequency > 1
		ORDER BY change_frequency DESC
		LIMIT 20`

	assetRows, err := ac.conn.Query(ac.ctx, hotAssetsQuery, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query hot assets: %w", err)
	}
	defer assetRows.Close()

	for assetRows.Next() {
		var asset AssetActivity
		if err := assetRows.Scan(
			&asset.FilePath,
			&asset.ChangeFrequency,
			&asset.Contributors,
			&asset.LastModified,
		); err != nil {
			continue
		}
		insights.HottestAssets = append(insights.HottestAssets, asset)
	}

	// Get dependency complexity
	depQuery := `
		SELECT 
			count() as total_dependencies,
			sum(is_circular) as circular_dependencies,
			avg(dependency_weight) as average_connections
		FROM asset_dependencies`

	depRow := ac.conn.QueryRow(ac.ctx, depQuery)
	if err := depRow.Scan(
		&insights.DependencyComplexity.TotalDependencies,
		&insights.DependencyComplexity.CircularDependencies,
		&insights.DependencyComplexity.AverageConnections,
	); err != nil {
		// Non-fatal error, continue without dependency stats
	}

	// Calculate collaboration score (simplified metric)
	if insights.ActiveDevelopers > 0 {
		insights.CollaborationScore = float64(insights.TotalCommits) / float64(insights.ActiveDevelopers)
	}

	return insights, nil
}

// Collaboration Events

// RecordCollaborationEvent stores a real-time collaboration event
func (ac *AnalyticsClient) RecordCollaborationEvent(event *CollaborationEvent) error {
	query := `
		INSERT INTO collaboration_events (
			event_id, event_type, user_name, file_path, project,
			event_time, session_id, additional_data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	return ac.conn.Exec(ac.ctx, query,
		event.EventID,
		event.EventType,
		event.UserName,
		event.FilePath,
		event.Project,
		event.EventTime,
		event.SessionID,
		event.AdditionalData,
	)
}

// GetActivityFeed retrieves recent activity for a project
func (ac *AnalyticsClient) GetActivityFeed(project string, limit int) ([]CollaborationEvent, error) {
	query := `
		SELECT event_id, event_type, user_name, file_path, project,
			   event_time, session_id, additional_data
		FROM vcs_analytics.collaboration_events
		WHERE project = ?
		ORDER BY event_time DESC
		LIMIT ?`

	rows, err := ac.conn.Query(ac.ctx, query, project, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query activity feed: %w", err)
	}
	defer rows.Close()

	var events []CollaborationEvent
	for rows.Next() {
		var event CollaborationEvent
		if err := rows.Scan(
			&event.EventID,
			&event.EventType,
			&event.UserName,
			&event.FilePath,
			&event.Project,
			&event.EventTime,
			&event.SessionID,
			&event.AdditionalData,
		); err != nil {
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// Schema Management

func (ac *AnalyticsClient) initializeSchema() error {
	tables := []string{
		createCommitsTable,
		createFileChangesTable,
		createAssetDependenciesTable,
		createCollaborationEventsTable,
		createProductivityMetricsTable,
	}

	for _, table := range tables {
		if err := ac.conn.Exec(ac.ctx, table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// Close closes the ClickHouse connection
func (ac *AnalyticsClient) Close() error {
	return ac.conn.Close()
}

// Table creation statements
const (
	createCommitsTable = `
		CREATE TABLE IF NOT EXISTS commits (
			commit_hash String,
			author String,
			commit_time DateTime64(3),
			branch String,
			message String,
			project String,
			parent_hashes Array(String),
			total_files UInt32,
			total_size_bytes UInt64
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(commit_time)
		ORDER BY (project, commit_time, author)`

	createFileChangesTable = `
		CREATE TABLE IF NOT EXISTS file_changes (
			commit_hash String,
			file_path String,
			change_type Enum('added', 'modified', 'deleted', 'moved'),
			content_hash String,
			file_size_bytes UInt64,
			author String,
			commit_time DateTime64(3),
			asset_type String,
			asset_class String,
			ue5_package_path String,
			is_blueprint Bool,
			blueprint_type String,
			sync_duration_ms UInt32,
			compression_ratio Float32
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(commit_time)
		ORDER BY (file_path, commit_time)`

	createAssetDependenciesTable = `
		CREATE TABLE IF NOT EXISTS asset_dependencies (
			asset_path String,
			depends_on_path String,
			dependency_type Enum('HardReference', 'SoftReference', 'SearchableReference'),
			discovered_time DateTime64(3),
			commit_hash String,
			is_circular Bool,
			dependency_weight Float32
		) ENGINE = ReplacingMergeTree()
		ORDER BY (asset_path, depends_on_path)`

	createCollaborationEventsTable = `
		CREATE TABLE IF NOT EXISTS collaboration_events (
			event_id String,
			event_type Enum('file_locked', 'file_unlocked', 'user_joined', 'user_left', 'conflict_detected'),
			user_name String,
			file_path String,
			project String,
			event_time DateTime64(3),
			session_id String,
			additional_data Map(String, String)
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(event_time)
		ORDER BY (project, event_time)`

	createProductivityMetricsTable = `
		CREATE TABLE IF NOT EXISTS productivity_metrics (
			date Date,
			author String,
			project String,
			commits_count UInt32,
			files_changed UInt32,
			lines_of_code_added UInt32,
			lines_of_code_removed UInt32,
			assets_created UInt32,
			assets_modified UInt32,
			time_active_minutes UInt32,
			conflicts_resolved UInt32
		) ENGINE = SummingMergeTree()
		PARTITION BY toYYYYMM(date)
		ORDER BY (project, date, author)`
)
