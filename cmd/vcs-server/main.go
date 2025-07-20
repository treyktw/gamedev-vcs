// main.go - Updated to include database and new routes
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Telerallc/gamedev-vcs/database"
	"github.com/Telerallc/gamedev-vcs/internal/analytics"
	"github.com/Telerallc/gamedev-vcs/internal/auth"
	fileops "github.com/Telerallc/gamedev-vcs/internal/fileOps"
	"github.com/Telerallc/gamedev-vcs/internal/state"
	"github.com/Telerallc/gamedev-vcs/internal/storage"
	"github.com/Telerallc/gamedev-vcs/models"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	StoragePath    string
	RedisURL       string
	ClickHouseURL  string
	ClickHouseDB   string
	ClickHouseUser string
	ClickHousePass string
	JWTSecret      string
	Environment    string
	MaxUploadSize  int64
	ChunkSize      int64
	DatabaseURL    string
}

type Server struct {
	config       *Config
	router       *gin.Engine
	storage      *storage.ContentStore
	stateManager *state.StateManager
	analytics    *analytics.AnalyticsClient
	fileOps      *fileops.FileOperations
	db           *database.DB
}

// Project creation request
type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"is_private"`
}

// Project response
type ProjectResponse struct {
	Success bool           `json:"success"`
	Project models.Project `json:"project,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// List projects response
type ListProjectsResponse struct {
	Success  bool             `json:"success"`
	Projects []models.Project `json:"projects,omitempty"`
	Error    string           `json:"error,omitempty"`
}

func (s *Server) setupProjectRoutes(v1 *gin.RouterGroup) {
	projects := v1.Group("/projects")
	projects.Use(s.AuthMiddleware())
	{
		projects.POST("", s.createProject)                       // POST /api/v1/projects
		projects.GET("", s.listUserProjects)                     // GET /api/v1/projects
		projects.GET("/:projectId", s.getProject)                // GET /api/v1/projects/:projectId
		projects.PUT("/:projectId", s.updateProject)             // PUT /api/v1/projects/:projectId
		projects.DELETE("/:projectId", s.deleteProject)          // DELETE /api/v1/projects/:projectId
		projects.GET("/:projectId/exists", s.checkProjectExists) // GET /api/v1/projects/:projectId/exists
		projects.GET("/:projectId/files", s.getProjectFiles)     // GET /api/v1/projects/:projectId/files
		projects.GET("/:projectId/members", s.getProjectMembers) // GET /api/v1/projects/:projectId/members
	}
}

// listUserProjects lists all projects for the authenticated user
func (s *Server) listUserProjects(c *gin.Context) {
	userID := c.GetString("user_id")

	var projects []models.Project
	err := s.db.Where("owner_id = ?", userID).Find(&projects).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, ListProjectsResponse{
			Success: false,
			Error:   "Failed to fetch projects",
		})
		return
	}

	c.JSON(http.StatusOK, ListProjectsResponse{
		Success:  true,
		Projects: projects,
	})
}

// getProject gets a specific project
func (s *Server) getProject(c *gin.Context) {
	projectID := c.Param("projectId")
	userID := c.GetString("user_id")

	var project models.Project
	err := s.db.Where("(id = ? OR slug = ?) AND owner_id = ?", projectID, projectID, userID).First(&project).Error
	if err != nil {
		c.JSON(http.StatusNotFound, ProjectResponse{
			Success: false,
			Error:   "Project not found",
		})
		return
	}

	c.JSON(http.StatusOK, ProjectResponse{
		Success: true,
		Project: project,
	})
}

// checkProjectExists checks if a project exists
func (s *Server) checkProjectExists(c *gin.Context) {
	projectID := c.Param("projectId")
	userID := c.GetString("user_id")

	var project models.Project
	err := s.db.Where("(id = ? OR slug = ?) AND owner_id = ?", projectID, projectID, userID).First(&project).Error

	exists := err == nil
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"exists":  exists,
		"project": func() interface{} {
			if exists {
				return project
			}
			return nil
		}(),
	})
}

func main() {
	// Load configuration
	config := loadConfig()

	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Create server instance
	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.cleanup()

	log.Println("✅ July Updates Version Running!")

	// Setup routes
	server.setupRoutes()

	// Start server
	server.start()
}

func loadConfig() *Config {
	config := &Config{
		Port:           getEnv("PORT", "8080"),
		StoragePath:    getEnv("STORAGE_PATH", "./storage"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		ClickHouseURL:  getEnv("CLICKHOUSE_URL", "localhost:9000"),
		ClickHouseDB:   getEnv("CLICKHOUSE_DATABASE", "vcs_analytics"),
		ClickHouseUser: getEnv("CLICKHOUSE_USERNAME", "vcs_user"),
		ClickHousePass: getEnv("CLICKHOUSE_PASSWORD", "dev_password"),
		JWTSecret:      getEnv("JWT_SECRET", "your-secret-key"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		MaxUploadSize:  parseInt64(getEnv("MAX_UPLOAD_SIZE", "5368709120")), // 5GB
		ChunkSize:      parseInt64(getEnv("CHUNK_SIZE", "1048576")),         // 1MB
		DatabaseURL:    getEnv("DATABASE_URL", "postgresql://neondb_owner:npg_o7VpBxu2Fmnj@ep-super-darkness-a8fi3dz0-pooler.eastus2.azure.neon.tech/neondb?sslmode=require"),
	}

	log.Printf("Configuration loaded:")
	log.Printf("  Port: %s", config.Port)
	log.Printf("  Storage Path: %s", config.StoragePath)
	log.Printf("  Redis URL: %s", config.RedisURL)
	log.Printf("  ClickHouse URL: %s", config.ClickHouseURL)
	log.Printf("  Database URL: %s", maskDatabaseURL(config.DatabaseURL))
	log.Printf("  Environment: %s", config.Environment)

	return config
}

func NewServer(config *Config) (*Server, error) {
	// Set Gin mode based on environment
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Set DATABASE_URL environment variable for database connection
	os.Setenv("DATABASE_URL", config.DatabaseURL)

	// Initialize database
	log.Printf("Connecting to database...")
	db, err := database.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// // Run migrations

	// Seed database in development
	if config.Environment == "development" {
		log.Printf("Seeding database...")
		if err := db.Seed(); err != nil {
			log.Printf("Warning: failed to seed database: %v", err)
		}
	}

	// Initialize storage
	log.Printf("Initializing content store at: %s", config.StoragePath)
	contentStore, err := storage.NewContentStore(config.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize state manager (Redis)
	log.Printf("Connecting to Redis at: %s", config.RedisURL)
	stateManager, err := state.NewStateManager(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Initialize analytics client (ClickHouse)
	log.Printf("Connecting to ClickHouse at: %s", config.ClickHouseURL)
	analyticsClient, err := analytics.NewAnalyticsClient(config.ClickHouseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize analytics client: %w", err)
	}

	// Initialize Git-style object store
	objectStore, err := storage.NewGitStyleObjectStore(".vcs/objects")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize object store: %w", err)
	}

	// Initialize file index
	fileIndex, err := storage.NewFileIndex(".vcs/index")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize file index: %w", err)
	}

	// Initialize file operations coordinator
	commitStore := storage.NewGitStyleCommitStore(".vcs", objectStore, fileIndex)
	fileOps := fileops.NewFileOperations(contentStore, stateManager, analyticsClient, objectStore, fileIndex)
	fileOps.SetCommitStore(commitStore)

	// Create Gin router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestSizeLimitMiddleware(config.MaxUploadSize))

	server := &Server{
		config:       config,
		router:       router,
		storage:      contentStore,
		stateManager: stateManager,
		analytics:    analyticsClient,
		fileOps:      fileOps,
		db:           db,
	}

	log.Printf("✅ Server initialized successfully")
	return server, nil
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.health)
	s.router.GET("/ready", s.ready)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Setup project routes
		s.setupProjectRoutes(v1)

		// Authentication (placeholder for now)
		// Authentication routes
		auth := v1.Group("/auth")
		{
			// Existing routes
			auth.POST("/login", s.login)
			auth.POST("/register", s.register)
			auth.POST("/refresh", s.refreshToken)

			// NEW ROUTES - Add these!
			auth.POST("/signup", s.quickSignup) // Quick signup

			// Google OAuth routes
			auth.GET("/google", s.googleLogin)
			auth.GET("/google/callback", s.googleCallback)

			// Protected routes
			protected := auth.Group("")
			protected.Use(s.AuthMiddleware())
			{
				protected.GET("/me", s.getCurrentUser) // Current user info
			}
		}

		// Files (protected routes)
		files := v1.Group("/files")
		files.Use(s.AuthMiddleware())
		{
			files.POST("/upload", s.uploadFile)
			files.POST("/batch-upload", s.batchUploadFiles) // NEW: Add this line
			files.GET("/:hash", s.downloadFile)
			files.POST("/upload-chunk", s.uploadChunk)
			files.POST("/finalize-upload", s.finalizeUpload)
			files.HEAD("/exists/:hash", s.checkFileExists) // NEW: Add this line too
		}

		// Locks (protected routes)
		locks := v1.Group("/locks")
		locks.Use(s.AuthMiddleware())
		{
			locks.POST("/:project/*file", s.lockFile)
			locks.DELETE("/:project/*file", s.unlockFile)
			locks.GET("/:project", s.listLocks)
		}

		// Presence and collaboration
		collaboration := v1.Group("/collaboration")
		collaboration.Use(s.AuthMiddleware())
		{
			collaboration.GET("/:project/presence", s.getProjectPresence)
			collaboration.POST("/:project/presence", s.updatePresence)
			collaboration.GET("/ws", s.websocketHandler)
		}

		// Commits and version control (protected routes)
		commits := v1.Group("/commits")
		commits.Use(s.AuthMiddleware())
		{
			commits.POST("/:project", s.createCommit)              // Create new commit
			commits.GET("/:project", s.getCommitHistory)           // Get commit history
			commits.GET("/:project/:commit", s.getCommit)          // Get specific commit
			commits.GET("/:project/diff", s.diffCommits)           // Compare commits
			commits.GET("/:project/files/*file", s.getFileHistory) // Get file history
		}

		// Branches (protected routes)
		branches := v1.Group("/branches")
		branches.Use(s.AuthMiddleware())
		{
			branches.GET("/:project", s.listBranches)            // List branches
			branches.POST("/:project", s.createBranch)           // Create branch
			branches.DELETE("/:project/:branch", s.deleteBranch) // Delete branch
			branches.PATCH("/:project/:branch", s.updateBranch)  // Update branch
		}

		// Sync operations (protected routes)
		sync := v1.Group("/sync")
		sync.Use(s.AuthMiddleware())
		{
			sync.POST("/:project/push", s.pushChanges)              // Push changes to server
			sync.POST("/:project/pull", s.pullChanges)              // Pull changes from server
			sync.GET("/:project/status", s.syncStatus)              // Get sync status
			sync.GET("/:project/branches/:branch", s.getBranchInfo) // Get branch info
		}

		// Analytics
		analytics := v1.Group("/analytics")
		analytics.Use(s.AuthMiddleware())
		{
			analytics.GET("/productivity/:project", s.getProductivityMetrics)
			analytics.GET("/activity/:project", s.getActivityFeed)
			analytics.GET("/dependencies/:project", s.getDependencyGraph)
			analytics.GET("/insights/:project", s.getTeamInsights)
			analytics.POST("/commits", s.recordCommit)
		}

		// Asset management
		assets := v1.Group("/assets")
		assets.Use(s.AuthMiddleware())
		{
			assets.GET("/:project/validate", s.validateAssetIntegrity)
			assets.GET("/:project/dependencies", s.getDependencyGraph)
		}

		// System management
		system := v1.Group("/system")
		system.Use(s.AuthMiddleware())
		{
			system.GET("/storage/stats", s.getStorageStats)
			system.POST("/cleanup", s.performCleanup)
		}

		// Health check
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}
}

func (s *Server) start() {
	// Start background cleanup routine
	go s.startBackgroundTasks()

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      s.router,
		ReadTimeout:  300 * time.Second, // 5 minutes for large uploads
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 VCS Server starting on port %s", s.config.Port)
		log.Printf("📊 Health check: http://localhost:%s/health", s.config.Port)
		log.Printf("📡 API endpoint: http://localhost:%s/api/v1", s.config.Port)
		log.Printf("🔌 WebSocket: ws://localhost:%s/api/v1/collaboration/ws", s.config.Port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("❌ Server forced to shutdown:", err)
	}

	log.Println("✅ Server exited")
}

func (s *Server) startBackgroundTasks() {
	ticker := time.NewTicker(5 * time.Minute) // Run every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		// Cleanup expired sessions and locks
		if err := s.fileOps.CleanupExpiredSessions(); err != nil {
			log.Printf("❌ Session cleanup error: %v", err)
		}

		// Cleanup storage
		if err := s.fileOps.CleanupStorage(); err != nil {
			log.Printf("❌ Storage cleanup error: %v", err)
		}

		log.Printf("🧹 Background cleanup completed")
	}
}

func (s *Server) cleanup() {
	log.Println("🧹 Cleaning up resources...")

	if s.analytics != nil {
		s.analytics.Close()
	}

	if s.stateManager != nil {
		s.stateManager.Close()
	}

	log.Println("✅ Cleanup completed")
}

// Middleware functions (existing ones)
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-User-ID, X-User-Name")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func requestSizeLimitMiddleware(maxSize int64) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("Request body too large. Maximum size: %d bytes", maxSize),
			})
			c.Abort()
			return
		}
		c.Next()
	})
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt64(s string) int64 {
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		return val
	}
	return 0
}

func maskDatabaseURL(url string) string {
	// Simple masking for logs - hide password
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) >= 2 {
			return "*****@" + parts[len(parts)-1]
		}
	}
	return url
}

// Handler methods
func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) ready(c *gin.Context) {
	// Check if all services are ready
	checks := map[string]bool{
		"storage":   s.storage != nil,
		"state":     s.stateManager != nil,
		"analytics": s.analytics != nil,
		"database":  s.db != nil,
	}

	allReady := true
	for _, ready := range checks {
		if !ready {
			allReady = false
			break
		}
	}

	if allReady {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"checks": checks,
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"checks": checks,
		})
	}
}

// // googleLogin initiates Google OAuth flow
// func (s *Server) googleLogin(c *gin.Context) {
// 	// Get JWT secret from environment or use default
// 	jwtSecret := s.config.JWTSecret
// 	if jwtSecret == "" {
// 		jwtSecret = "default-secret-key-change-in-production"
// 	}

// 	authService := auth.NewAuthService(s.db.DB, jwtSecret)
// 	oauthConfig := authService.SetupOAuth()

// 	authURL, state, err := authService.HandleGoogleLogin(oauthConfig)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to initiate OAuth flow",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success":  true,
// 		"auth_url": authURL,
// 		"state":    state,
// 	})
// }

// googleCallback handles Google OAuth callback
func (s *Server) googleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Missing code or state parameter",
		})
		return
	}

	// Get JWT secret from environment or use default
	jwtSecret := s.config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production"
	}

	authService := auth.NewAuthService(s.db.DB, jwtSecret)

	// IMPORTANT: The callback needs to use the SAME OAuth config that was used
	// to generate the original auth URL. Since this is coming from CLI,
	// we need to create the config with the CLI callback URL

	// Check if this request is coming from CLI by looking at the User-Agent or
	// by reconstructing the OAuth config. For now, let's assume CLI if we get here
	// and try to determine the correct callback URL from the request context

	// For CLI requests, always use the CLI callback URL
	oauthConfig := authService.SetupOAuth("http://localhost:8081/callback")

	loginResp, err := authService.HandleGoogleCallback(code, state, oauthConfig)
	if err != nil {
		log.Printf("OAuth callback error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "OAuth authentication failed",
		})
		return
	}

	c.JSON(http.StatusOK, loginResp)
}
