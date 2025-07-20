// internal/database/database.go
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Telerallc/gamedev-vcs/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB holds the database connection
type DB struct {
	*gorm.DB
}

// Connect initializes the database connection
func Connect() (*DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	// Configure GORM logger
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &DB{db}, nil
}

// ConnectDrizzle initializes the database connection for Drizzle compatibility
func ConnectDrizzle() (*DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	// Configure GORM logger
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &DB{db}, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	// Create team_members table with composite primary key manually
	teamMembersTable := `
		CREATE TABLE IF NOT EXISTS team_members (
			team_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT DEFAULT 'member',
			joined_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (team_id, user_id),
			FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`

	if err := db.Exec(teamMembersTable).Error; err != nil {
		return fmt.Errorf("failed to create team_members table: %w", err)
	}

	err := db.AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.Session{},
		&models.VerificationToken{},
		&models.Organization{},
		&models.OrganizationMember{},
		&models.Team{},
		&models.Project{},
		&models.ProjectMember{},
		&models.Branch{},
		&models.File{},
		&models.Commit{},
		&models.CommitTree{},
		&models.Ref{},
		&models.Tag{},
		&models.FileVersion{},
		&models.FileEvent{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

// MigrateDrizzle runs Drizzle-compatible migrations
func (db *DB) MigrateDrizzle() error {
	// Create enums first
	enums := []string{
		`CREATE TYPE IF NOT EXISTS organization_role AS ENUM ('OWNER', 'ADMIN', 'MEMBER')`,
		`CREATE TYPE IF NOT EXISTS team_role AS ENUM ('MAINTAINER', 'MEMBER')`,
		`CREATE TYPE IF NOT EXISTS project_role AS ENUM ('ADMIN', 'WRITE', 'MEMBER', 'READ')`,
	}

	for _, enum := range enums {
		if err := db.Exec(enum).Error; err != nil {
			return fmt.Errorf("failed to create enum: %w", err)
		}
	}

	// Create team_members table with composite primary key manually
	teamMembersTable := `
		CREATE TABLE IF NOT EXISTS team_members (
			team_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role team_role DEFAULT 'MEMBER',
			joined_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (team_id, user_id),
			FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`

	if err := db.Exec(teamMembersTable).Error; err != nil {
		return fmt.Errorf("failed to create team_members table: %w", err)
	}

	// Run GORM migrations for all other tables
	err := db.AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.Session{},
		&models.VerificationToken{},
		&models.Organization{},
		&models.OrganizationMember{},
		&models.Team{},
		&models.Project{},
		&models.ProjectMember{},
		&models.Branch{},
		&models.File{},
		&models.Commit{},
		&models.CommitTree{},
		&models.Ref{},
		&models.Tag{},
		&models.FileVersion{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("✅ Drizzle migrations completed successfully")
	return nil
}

// Seed creates initial data for development
func (db *DB) Seed() error {
	// Check if we already have data
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount > 0 {
		log.Println("Database already seeded, skipping...")
		return nil
	}

	// Create default users
	users := []models.User{
		{
			ID:       "user_1",
			Name:     "Alice Johnson",
			Email:    "alice@example.com",
			Username: "alice",
			Bio:      "Lead Game Developer",
		},
		{
			ID:       "user_2",
			Name:     "Bob Smith",
			Email:    "bob@example.com",
			Username: "bob",
			Bio:      "3D Artist",
		},
		{
			ID:       "user_3",
			Name:     "Charlie Brown",
			Email:    "charlie@example.com",
			Username: "charlie",
			Bio:      "Level Designer",
		},
	}

	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			log.Printf("Failed to create user %s: %v", user.Username, err)
		}
	}

	// Create default organization
	org := models.Organization{
		ID:          "org_1",
		Name:        "Awesome Game Studio",
		Slug:        "awesome-game-studio",
		Description: "Independent game development studio",
		OwnerID:     "user_1",
	}

	if err := db.Create(&org).Error; err != nil {
		log.Printf("Failed to create organization: %v", err)
	}

	// Create default projects
	projects := []models.Project{
		{
			ID:             "proj_1",
			Name:           "Awesome Game",
			Slug:           "awesome-game",
			Description:    "Main UE5 game project with character systems",
			IsPrivate:      true,
			DefaultBranch:  "main",
			OwnerID:        &users[0].ID,
			OrganizationID: &org.ID,
		},
		{
			ID:            "proj_2",
			Name:          "Character Assets",
			Slug:          "character-assets",
			Description:   "3D character models and animations",
			IsPrivate:     true,
			DefaultBranch: "main",
			OwnerID:       &users[1].ID,
		},
	}

	for _, project := range projects {
		if err := db.Create(&project).Error; err != nil {
			log.Printf("Failed to create project %s: %v", project.Name, err)
		}
	}

	// Create project members
	members := []models.ProjectMember{
		{
			ID:        "member_1",
			ProjectID: "proj_1",
			UserID:    "user_2",
			Role:      "write",
		},
		{
			ID:        "member_2",
			ProjectID: "proj_1",
			UserID:    "user_3",
			Role:      "write",
		},
	}

	for _, member := range members {
		if err := db.Create(&member).Error; err != nil {
			log.Printf("Failed to create project member: %v", err)
		}
	}

	// Create default branches
	branches := []models.Branch{
		{
			ID:        "branch_1",
			Name:      "main",
			ProjectID: "proj_1",
			IsDefault: true,
		},
		{
			ID:        "branch_2",
			Name:      "main",
			ProjectID: "proj_2",
			IsDefault: true,
		},
	}

	for _, branch := range branches {
		if err := db.Create(&branch).Error; err != nil {
			log.Printf("Failed to create branch: %v", err)
		}
	}

	log.Println("Database seeding completed successfully")
	return nil
}

// ProjectRepository provides project-related database operations
type ProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// GetUserProjects retrieves all projects accessible to a user
func (r *ProjectRepository) GetUserProjects(userID string) ([]models.Project, error) {
	var projects []models.Project

	err := r.db.
		Preload("Owner").
		Preload("Organization").
		Preload("Members").
		Preload("Files").
		Where("owner_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?)", userID, userID).
		Find(&projects).Error

	if err != nil {
		return nil, err
	}

	// Calculate stats for each project
	for i := range projects {
		projects[i].Stats = projects[i].GetProjectStats(r.db)
	}

	return projects, nil
}

// GetProjectByID retrieves a project by ID with permission check
func (r *ProjectRepository) GetProjectByID(projectID, userID string) (*models.Project, error) {
	var project models.Project

	err := r.db.
		Preload("Owner").
		Preload("Organization").
		Preload("Members.User").
		Preload("Files").
		Preload("Branches").
		Where("id = ?", projectID).
		First(&project).Error

	if err != nil {
		return nil, err
	}

	// Check permissions
	if !project.HasPermission(userID, "read") {
		return nil, fmt.Errorf("access denied")
	}

	project.Stats = project.GetProjectStats(r.db)
	return &project, nil
}

// CreateProject creates a new project
func (r *ProjectRepository) CreateProject(project *models.Project) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create the project
		if err := tx.Create(project).Error; err != nil {
			return err
		}

		// Create default branch
		defaultBranch := models.Branch{
			ID:        fmt.Sprintf("branch_%d", time.Now().UnixNano()),
			Name:      project.DefaultBranch,
			ProjectID: project.ID,
			IsDefault: true,
		}

		if err := tx.Create(&defaultBranch).Error; err != nil {
			return err
		}

		return nil
	})
}

// UpdateProject updates an existing project
func (r *ProjectRepository) UpdateProject(projectID string, updates map[string]interface{}, userID string) (*models.Project, error) {
	var project models.Project

	// Check if user has permission to update
	if err := r.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return nil, err
	}

	if !project.HasPermission(userID, "admin") {
		return nil, fmt.Errorf("insufficient permissions")
	}

	// Update the project
	if err := r.db.Model(&project).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Return updated project
	return r.GetProjectByID(projectID, userID)
}

// DeleteProject deletes a project
func (r *ProjectRepository) DeleteProject(projectID, userID string) error {
	var project models.Project

	// Check if user has permission to delete
	if err := r.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return err
	}

	if !project.HasPermission(userID, "admin") {
		return fmt.Errorf("insufficient permissions")
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete related records
		if err := tx.Where("project_id = ?", projectID).Delete(&models.File{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&models.Branch{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&models.ProjectMember{}).Error; err != nil {
			return err
		}

		// Delete the project
		return tx.Delete(&project).Error
	})
}

// FileRepository provides file-related database operations
type FileRepository struct {
	db *gorm.DB
}

// NewFileRepository creates a new file repository
func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{db: db}
}

// GetProjectFiles retrieves all files for a project
func (r *FileRepository) GetProjectFiles(projectID, userID string) ([]models.File, error) {
	// First check if user has access to the project
	var project models.Project
	if err := r.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return nil, err
	}

	if !project.HasPermission(userID, "read") {
		return nil, fmt.Errorf("access denied")
	}

	var files []models.File
	err := r.db.
		Preload("LockUser").
		Preload("ModifierUser").
		Where("project_id = ?", projectID).
		Order("path").
		Find(&files).Error

	return files, err
}

// CreateOrUpdateFile creates or updates a file record
func (r *FileRepository) CreateOrUpdateFile(file *models.File) error {
	// Check if file exists
	var existing models.File
	err := r.db.Where("project_id = ? AND path = ? AND branch = ?",
		file.ProjectID, file.Path, file.Branch).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create new file
		return r.db.Create(file).Error
	} else if err != nil {
		return err
	} else {
		// Update existing file
		return r.db.Model(&existing).Updates(map[string]interface{}{
			"content_hash":     file.ContentHash,
			"size":             file.Size,
			"mime_type":        file.MimeType,
			"last_modified_by": file.LastModifiedBy,
			"last_modified_at": time.Now(),
		}).Error
	}
}

// GetFileByPath retrieves a file by project ID and path
func (r *FileRepository) GetFileByPath(projectID, filePath string) (*models.File, error) {
	var file models.File
	err := r.db.Where("project_id = ? AND path = ?", projectID, filePath).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}
