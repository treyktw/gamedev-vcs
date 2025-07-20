package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Telerallc/gamedev-vcs/internal/auth"
	"github.com/Telerallc/gamedev-vcs/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// QuickSignupRequest for simple user creation
type QuickSignupRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Password string `json:"password" binding:"required,min=6"`
}

// quickSignup handles quick user registration
func (s *Server) quickSignup(c *gin.Context) {
	var req QuickSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Check if username or email already exists
	var existingUser models.User
	err := s.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "Username or email already exists",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to process password",
		})
		return
	}

	// Create user
	user := models.User{
		ID:       fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Username: req.Username,
		Email:    req.Email,
		Name:     req.Name,
		Settings: models.JSON{
			"password_hash": string(hashedPassword),
			"created_via":   "quick_signup",
			"auth_method":   "password",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create user account",
		})
		return
	}

	// Generate JWT tokens
	jwtSecret := s.config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production"
	}

	// Create auth service instance
	authService := auth.NewAuthService(s.db.DB, jwtSecret)
	token, refreshToken, expiresAt, err := authService.GenerateTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to generate authentication tokens",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"success":       true,
		"token":         token,
		"refresh_token": refreshToken,
		"expires_at":    expiresAt,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"name":     user.Name,
		},
	})
}

// login handles user authentication
func (s *Server) login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get JWT secret from environment or use default
	jwtSecret := s.config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production"
	}

	authService := auth.NewAuthService(s.db.DB, jwtSecret)

	loginResp, err := authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Invalid credentials",
		})
		return
	}

	c.JSON(http.StatusOK, loginResp)
}

// register handles user registration
func (s *Server) register(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get JWT secret from environment or use default
	jwtSecret := s.config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production"
	}

	authService := auth.NewAuthService(s.db.DB, jwtSecret)

	loginResp, err := authService.Register(req)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		}

		c.JSON(statusCode, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, loginResp)
}

// refreshToken handles JWT token refresh
func (s *Server) refreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get JWT secret from environment or use default
	jwtSecret := s.config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production"
	}

	authService := auth.NewAuthService(s.db.DB, jwtSecret)

	loginResp, err := authService.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Invalid refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, loginResp)
}

// authMiddleware validates JWT tokens
func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Fall back to X-User-ID header for backward compatibility
			userID := c.GetHeader("X-User-ID")
			userName := c.GetHeader("X-User-Name")

			if userID != "" {
				c.Set("user_id", userID)
				if userName != "" {
					c.Set("user_name", userName)
				} else {
					c.Set("user_name", userID)
				}
				c.Next()
				return
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token from Bearer header
		tokenString := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Get JWT secret from environment or use default
		jwtSecret := s.config.JWTSecret
		if jwtSecret == "" {
			jwtSecret = "default-secret-key-change-in-production"
		}

		authService := auth.NewAuthService(s.db.DB, jwtSecret)

		// Validate token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_name", claims.Username)
		c.Set("user_email", claims.Email)

		c.Next()
	}
}

// googleLogin initiates Google OAuth flow
func (s *Server) googleLogin(c *gin.Context) {
	// Get callback URL from query parameter (for CLI local server)
	callbackURL := c.Query("callback_url")

	// Debug logging
	if callbackURL != "" {
		log.Printf("CLI requested custom callback URL: %s", callbackURL)
	} else {
		log.Printf("No custom callback URL provided, using default")
	}

	// Get JWT secret from environment or use default
	jwtSecret := s.config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production"
	}

	authService := auth.NewAuthService(s.db.DB, jwtSecret)

	// Create OAuth config with custom callback URL if provided
	var oauthConfig *auth.GoogleOAuthConfig
	if callbackURL != "" {
		oauthConfig = authService.SetupOAuth(callbackURL) // Pass the callback URL
	} else {
		oauthConfig = authService.SetupOAuth() // Use default
	}

	authURL, state, err := authService.HandleGoogleLogin(oauthConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initiate OAuth flow",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"auth_url": authURL,
		"state":    state,
	})
}

// getCurrentUser returns current user information
func (s *Server) getCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var user models.User
	err := s.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"name":     user.Name,
		},
	})
}
