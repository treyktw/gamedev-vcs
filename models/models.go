// internal/models/models.go
package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// User represents a user in the system
type User struct {
	ID            string     `json:"id" gorm:"primaryKey"`
	Name          string     `json:"name"`
	Email         string     `json:"email" gorm:"uniqueIndex:idx_users_email"`
	EmailVerified *time.Time `json:"email_verified" gorm:"column:email_verified"`
	Image         string     `json:"image"`
	Username      string     `json:"username" gorm:"uniqueIndex:idx_users_username"`
	Bio           string     `json:"bio"`
	Location      string     `json:"location"`
	Website       string     `json:"website"`
	Company       string     `json:"company"`
	AvatarURL     string     `json:"avatar_url"`
	Settings      JSON       `json:"settings" gorm:"type:jsonb"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Relations
	OwnedProjects       []Project            `json:"owned_projects,omitempty" gorm:"foreignKey:OwnerID"`
	ProjectMembers      []ProjectMember      `json:"project_members,omitempty"`
	OrganizationMembers []OrganizationMember `json:"organization_members,omitempty"`
	TeamMembers         []TeamMember         `json:"team_members,omitempty"`
}

// Account represents OAuth account (NextAuth)
type Account struct {
	ID                string  `json:"id" gorm:"primaryKey"`
	UserID            string  `json:"user_id" gorm:"column:user_id"`
	Type              string  `json:"type"`
	Provider          string  `json:"provider"`
	ProviderAccountID string  `json:"provider_account_id" gorm:"column:provider_account_id"`
	RefreshToken      *string `json:"refresh_token"`
	AccessToken       *string `json:"access_token"`
	ExpiresAt         *int    `json:"expires_at"`
	TokenType         *string `json:"token_type"`
	Scope             *string `json:"scope"`
	IDToken           *string `json:"id_token"`
	SessionState      *string `json:"session_state"`

	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// Session represents user session (NextAuth)
type Session struct {
	SessionToken string    `json:"session_token" gorm:"primaryKey;column:session_token"`
	UserID       string    `json:"user_id" gorm:"column:user_id"`
	Expires      time.Time `json:"expires"`

	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// VerificationToken represents email verification (NextAuth)
type VerificationToken struct {
	Identifier string    `json:"identifier"`
	Token      string    `json:"token"`
	Expires    time.Time `json:"expires"`
}

// Organization represents a team/company organization
type Organization struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug" gorm:"uniqueIndex:idx_organizations_slug"`
	Description string    `json:"description"`
	AvatarURL   string    `json:"avatar_url"`
	Website     string    `json:"website"`
	Location    string    `json:"location"`
	Settings    JSON      `json:"settings" gorm:"type:jsonb"`
	OwnerID     string    `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Owner    User                 `json:"owner" gorm:"foreignKey:OwnerID"`
	Members  []OrganizationMember `json:"members,omitempty"`
	Projects []Project            `json:"projects,omitempty"`
	Teams    []Team               `json:"teams,omitempty"`
}

// OrganizationMember represents organization membership
type OrganizationMember struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	Role           string    `json:"role" gorm:"default:member"` // owner, admin, member
	JoinedAt       time.Time `json:"joined_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relations
	Organization Organization `json:"organization" gorm:"foreignKey:OrganizationID"`
	User         User         `json:"user" gorm:"foreignKey:UserID"`
}

// Team represents a team within an organization
type Team struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relations
	Organization Organization `json:"organization" gorm:"foreignKey:OrganizationID"`
	Members      []TeamMember `json:"members,omitempty"`
}

// TeamMember represents team membership
type TeamMember struct {
	TeamID   string    `json:"team_id" gorm:"primaryKey"`
	UserID   string    `json:"user_id" gorm:"primaryKey"`
	Role     string    `json:"role" gorm:"default:member"` // maintainer, member
	JoinedAt time.Time `json:"joined_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relations
	Team Team `json:"team" gorm:"foreignKey:TeamID"`
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// Project represents a VCS project/repository
type Project struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	IsPrivate      bool      `json:"is_private" gorm:"default:true"`
	DefaultBranch  string    `json:"default_branch" gorm:"default:main"`
	Settings       JSON      `json:"settings" gorm:"type:jsonb"`
	OwnerID        *string   `json:"owner_id"`
	OrganizationID *string   `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relations
	Owner        *User           `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
	Organization *Organization   `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
	Members      []ProjectMember `json:"members,omitempty"`
	Files        []File          `json:"files,omitempty"`
	Branches     []Branch        `json:"branches,omitempty"`

	// Computed fields
	Stats *ProjectStats `json:"stats,omitempty" gorm:"-"`
}

// ProjectMember represents project membership and permissions
type ProjectMember struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role" gorm:"default:member"` // owner, admin, write, read
	JoinedAt  time.Time `json:"joined_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relations
	Project Project `json:"project" gorm:"foreignKey:ProjectID"`
	User    User    `json:"user" gorm:"foreignKey:UserID"`
}

// Branch represents a git-like branch
type Branch struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	ProjectID   string    `json:"project_id"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	IsProtected bool      `json:"is_protected" gorm:"default:false"`
	LastCommit  string    `json:"last_commit"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Project Project `json:"project" gorm:"foreignKey:ProjectID"`
}

// File represents a file in the VCS
type File struct {
	ID             string     `json:"id" gorm:"primaryKey"`
	ProjectID      string     `json:"project_id"`
	Path           string     `json:"path"`
	ContentHash    string     `json:"content_hash"`
	Size           int64      `json:"size"`
	MimeType       string     `json:"mime_type"`
	Branch         string     `json:"branch" gorm:"default:main"`
	IsLocked       bool       `json:"is_locked" gorm:"default:false"`
	LockedBy       *string    `json:"locked_by"`
	LockedAt       *time.Time `json:"locked_at"`
	LastModifiedBy *string    `json:"last_modified_by"`
	LastModifiedAt time.Time  `json:"last_modified_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relations
	Project      Project `json:"project" gorm:"foreignKey:ProjectID"`
	LockUser     *User   `json:"lock_user,omitempty" gorm:"foreignKey:LockedBy"`
	ModifierUser *User   `json:"modifier_user,omitempty" gorm:"foreignKey:LastModifiedBy"`
}

// Commit represents a VCS commit
type Commit struct {
	ID        string    `json:"id" gorm:"primaryKey"` // SHA-1 hash
	ProjectID string    `json:"project_id"`
	AuthorID  string    `json:"author_id"`
	Message   string    `json:"message" gorm:"not null"`
	TreeHash  string    `json:"tree_hash"`                    // Hash of the file tree at this commit
	ParentIDs []string  `json:"parent_ids" gorm:"type:jsonb"` // Parent commit IDs
	CreatedAt time.Time `json:"created_at"`

	// Relations
	Project Project `json:"project" gorm:"foreignKey:ProjectID"`
	Author  User    `json:"author" gorm:"foreignKey:AuthorID"`
}

// CommitTree represents the file tree at a specific commit
type CommitTree struct {
	ID        string           `json:"id" gorm:"primaryKey"` // Tree hash
	ProjectID string           `json:"project_id"`
	CommitID  string           `json:"commit_id"`
	Files     []CommitTreeFile `json:"files" gorm:"type:jsonb"`
	CreatedAt time.Time        `json:"created_at"`

	// Relations
	Project Project `json:"project" gorm:"foreignKey:ProjectID"`
}

// CommitTreeFile represents a file entry in a commit tree
type CommitTreeFile struct {
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	Size        int64  `json:"size"`
	Mode        string `json:"mode"` // file permissions
	Type        string `json:"type"` // file, directory, symlink
}

// Ref represents a git-like reference (branch HEAD, tags, etc.)
type Ref struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"` // refs/heads/main, refs/tags/v1.0.0
	Type      string    `json:"type"` // branch, tag, remote
	CommitID  string    `json:"commit_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Project Project `json:"project" gorm:"foreignKey:ProjectID"`
	Commit  Commit  `json:"commit" gorm:"foreignKey:CommitID"`
}

// Tag represents a VCS tag/release
type Tag struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name" gorm:"not null"`
	Message     string    `json:"message"`
	CommitID    string    `json:"commit_id"`
	TaggerID    string    `json:"tagger_id"`
	IsAnnotated bool      `json:"is_annotated" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`

	// Relations
	Project Project `json:"project" gorm:"foreignKey:ProjectID"`
	Commit  Commit  `json:"commit" gorm:"foreignKey:CommitID"`
	Tagger  User    `json:"tagger" gorm:"foreignKey:TaggerID"`
}

// FileVersion represents a specific version of a file
type FileVersion struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	ProjectID   string    `json:"project_id"`
	Path        string    `json:"path"`
	ContentHash string    `json:"content_hash"`
	CommitID    string    `json:"commit_id"`
	Size        int64     `json:"size"`
	MimeType    string    `json:"mime_type"`
	CreatedAt   time.Time `json:"created_at"`

	// Relations
	Project Project `json:"project" gorm:"foreignKey:ProjectID"`
	Commit  Commit  `json:"commit" gorm:"foreignKey:CommitID"`
}

// ProjectStats represents computed project statistics
type ProjectStats struct {
	TotalFiles   int64  `json:"total_files"`
	TotalSize    int64  `json:"total_size"`
	TotalSizeStr string `json:"total_size_str"`
	Contributors int64  `json:"contributors"`
	LastActivity string `json:"last_activity"`
	ActiveLocks  int64  `json:"active_locks"`
}

// JSON type for JSONB fields
type JSON map[string]interface{}

// Scan implements the sql.Scanner interface for JSON
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return fmt.Errorf("cannot scan %T into JSON", value)
	}
}

// Value implements the driver.Valuer interface for JSON
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// FormatFileSize formats file size in human readable format
func FormatFileSize(bytes int64) string {
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

// GetProjectStats computes project statistics
func (p *Project) GetProjectStats(db interface{}) *ProjectStats {
	// This would be implemented with actual database queries
	// For now, return empty stats
	return &ProjectStats{
		TotalFiles:   0,
		TotalSize:    0,
		TotalSizeStr: "0 B",
		Contributors: 0,
		LastActivity: "Never",
		ActiveLocks:  0,
	}
}

// HasPermission checks if a user has the specified permission on a project
func (p *Project) HasPermission(userID string, permission string) bool {
	// This would be implemented with actual permission checking logic
	// For now, return true for basic access
	return true
}

type FileEvent struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	ProjectID string    `json:"project_id" gorm:"index"`
	UserID    string    `json:"user_id" gorm:"index"`
	EventType string    `json:"event_type"` // file_uploaded, file_downloaded, file_deleted, etc.
	FilePath  string    `json:"file_path"`
	Details   JSON      `json:"details" gorm:"type:jsonb"` // Additional event metadata
	CreatedAt time.Time `json:"created_at" gorm:"index"`

	// Relations
	Project Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	User    User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName methods for GORM
func (User) TableName() string               { return "users" }
func (Account) TableName() string            { return "accounts" }
func (Session) TableName() string            { return "sessions" }
func (VerificationToken) TableName() string  { return "verificationtokens" }
func (Organization) TableName() string       { return "organizations" }
func (OrganizationMember) TableName() string { return "organization_members" }
func (Team) TableName() string               { return "teams" }
func (TeamMember) TableName() string         { return "team_members" }
func (Project) TableName() string            { return "projects" }
func (ProjectMember) TableName() string      { return "project_members" }
func (Branch) TableName() string             { return "branches" }
func (File) TableName() string               { return "files" }
func (Commit) TableName() string             { return "commits" }
func (CommitTree) TableName() string         { return "commit_trees" }
func (Ref) TableName() string                { return "refs" }
func (Tag) TableName() string                { return "tags" }
func (FileVersion) TableName() string        { return "file_versions" }
