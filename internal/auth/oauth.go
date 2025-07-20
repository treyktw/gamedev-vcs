// Google OAuth Integration for VCS

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"

	"github.com/Telerallc/gamedev-vcs/models"
	"github.com/gin-gonic/gin"
)

// GoogleOAuthConfig holds OAuth configuration
type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	oauth2Config *oauth2.Config
}

// GoogleUserInfo represents user info from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

// OAuthLoginResponse represents OAuth login response
type OAuthLoginResponse struct {
	Success      bool     `json:"success"`
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresAt    int64    `json:"expires_at"`
	User         UserInfo `json:"user"`
	IsNewUser    bool     `json:"is_new_user"`
	AuthMethod   string   `json:"auth_method"`
}

// QuickSignupRequest for simple user creation
type QuickSignupRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Password string `json:"password" binding:"required,min=6"`
}

// // NewGoogleOAuthConfig creates OAuth configuration
// func NewGoogleOAuthConfig() *GoogleOAuthConfig {
// 	clientID := os.Getenv("GOOGLE_CLIENT_ID")
// 	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
// 	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

// 	log.Printf("Environment check:")
// 	log.Printf("  GOOGLE_CLIENT_ID: %s", clientID)
// 	log.Printf("  GOOGLE_CLIENT_SECRET: %s", func() string {
// 		if clientSecret == "" {
// 			return "EMPTY/NOT_SET"
// 		}
// 		return clientSecret[:10] + "..." // Show first 10 chars
// 	}())
// 	log.Printf("  GOOGLE_REDIRECT_URL: %s", redirectURL)

// 	// Default values for development
// 	if clientID == "" {
// 		clientID = "337458724787-rdh1frb7e5qp6c9a6pu205ls87m7kcp1.apps.googleusercontent.com"
// 	}
// 	if redirectURL == "" {
// 		redirectURL = "http://localhost:8080/api/v1/auth/google/callback"
// 	}

// 	if clientSecret == "" {
// 		clientSecret = "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"
// 	}

// 	config := &GoogleOAuthConfig{
// 		ClientID:     clientID,
// 		ClientSecret: clientSecret,
// 		RedirectURL:  redirectURL,
// 	}

// 	config.oauth2Config = &oauth2.Config{
// 		ClientID:     clientID,
// 		ClientSecret: clientSecret,
// 		RedirectURL:  redirectURL,
// 		Scopes: []string{
// 			"https://www.googleapis.com/auth/userinfo.email",
// 			"https://www.googleapis.com/auth/userinfo.profile",
// 		},
// 		Endpoint: google.Endpoint,
// 	}

// 	return config
// }

// NewGoogleOAuthConfigWithCallback creates OAuth configuration with custom redirect URL
func NewGoogleOAuthConfigWithCallback(callbackURL string) *GoogleOAuthConfig {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	// Use hardcoded values if env vars are empty
	if clientID == "" {
		clientID = "337458724787-rdh1frb7e5qp6c9a6pu205ls87m7kcp1.apps.googleusercontent.com"
	}
	if clientSecret == "" {
		clientSecret = "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"
	}

	log.Printf("Using custom callback URL: %s", callbackURL)

	config := &GoogleOAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  callbackURL, // Use the custom callback URL
	}

	config.oauth2Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  callbackURL, // Use the custom callback URL
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return config
}

// Update this existing function
func (as *AuthService) SetupOAuth(callbackURL ...string) *GoogleOAuthConfig {
	var config *GoogleOAuthConfig

	if len(callbackURL) > 0 && callbackURL[0] != "" {
		// Use custom callback URL (for CLI)
		config = NewGoogleOAuthConfigWithCallback(callbackURL[0])
		log.Printf("OAuth config with custom callback - ClientID: %s..., RedirectURL: %s",
			config.ClientID[:20], config.RedirectURL)
	}
	// else {
	// 	// Use default config (for web)
	// 	config = NewGoogleOAuthConfig()
	// 	log.Printf("OAuth config with default callback - ClientID: %s..., RedirectURL: %s",
	// 		config.ClientID[:20], config.RedirectURL)
	// }

	return config
}

// QuickSignup creates a user account quickly
func (as *AuthService) QuickSignup(req QuickSignupRequest) (*LoginResponse, error) {
	// Check if username or email already exists
	var existingUser models.User
	err := as.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error
	if err == nil {
		return nil, fmt.Errorf("username or email already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
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

	if err := as.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	token, refreshToken, expiresAt, err := as.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &LoginResponse{
		Success:      true,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Name:     user.Name,
		},
	}, nil
}

// HandleGoogleLogin initiates Google OAuth flow
func (as *AuthService) HandleGoogleLogin(oauthConfig *GoogleOAuthConfig) (string, string, error) {
	// Generate state parameter for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state temporarily (in production, use Redis or database)
	// For now, we'll return it and validate in callback

	authURL := oauthConfig.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return authURL, state, nil
}

// HandleGoogleCallback processes Google OAuth callback
func (as *AuthService) HandleGoogleCallback(code, state string, oauthConfig *GoogleOAuthConfig) (*OAuthLoginResponse, error) {
	log.Printf("Starting OAuth callback with code: %s", code[:10]+"...") // Log partial code for security

	// Exchange code for token
	ctx := context.Background()
	token, err := oauthConfig.oauth2Config.Exchange(ctx, code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	log.Printf("Token exchange successful")

	// Get user info from Google
	userInfo, err := as.getGoogleUserInfo(token.AccessToken)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	log.Printf("Got user info for: %s", userInfo.Email)

	// Find or create user
	user, isNewUser, err := as.findOrCreateOAuthUser(userInfo)
	if err != nil {
		log.Printf("Failed to find/create user: %v", err)
		return nil, fmt.Errorf("failed to find/create user: %w", err)
	}
	log.Printf("User found/created: %s", user.Username)

	user.Username = as.generateUsernameFromEmail(user.Email)

	// Generate JWT tokens
	jwtToken, refreshToken, expiresAt, err := as.generateTokens(*user)
	if err != nil {
		log.Printf("Failed to generate tokens: %v", err)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}
	log.Printf("Tokens generated successfully")

	return &OAuthLoginResponse{
		Success:      true,
		Token:        jwtToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Name:     user.Name,
		},
		IsNewUser:  isNewUser,
		AuthMethod: "google_oauth",
	}, nil
}

// getGoogleUserInfo fetches user information from Google
func (as *AuthService) getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &userInfo, nil
}

func (as *AuthService) findOrCreateOAuthUser(googleUser *GoogleUserInfo) (*models.User, bool, error) {
	// Try to find existing user by email
	var user models.User
	err := as.db.Where("email = ?", googleUser.Email).First(&user).Error

	if err == nil {
		// User exists, update OAuth info
		// Initialize Settings map if it's nil
		if user.Settings == nil {
			user.Settings = make(models.JSON)
		}

		user.Settings["google_id"] = googleUser.ID
		user.Settings["auth_method"] = "google_oauth"
		user.Settings["last_oauth_login"] = time.Now().Format(time.RFC3339)
		user.AvatarURL = googleUser.Picture
		user.UpdatedAt = time.Now()

		as.db.Save(&user)
		return &user, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, fmt.Errorf("database error: %w", err)
	}

	// Create new user
	user = models.User{
		ID:        fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Username:  as.generateUsernameFromEmail(googleUser.Email),
		Email:     googleUser.Email,
		Name:      googleUser.Name,
		AvatarURL: googleUser.Picture,
		Settings: models.JSON{
			"google_id":         googleUser.ID,
			"auth_method":       "google_oauth",
			"created_via":       "google_oauth",
			"verified_email":    googleUser.VerifiedEmail,
			"given_name":        googleUser.GivenName,
			"family_name":       googleUser.FamilyName,
			"first_oauth_login": time.Now().Format(time.RFC3339),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Ensure username is unique
	user.Username = as.ensureUniqueUsername(user.Username)

	if err := as.db.Create(&user).Error; err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, true, nil
}

// generateUsernameFromEmail creates a username from email
func (as *AuthService) generateUsernameFromEmail(email string) string {
	// Take part before @
	parts := strings.Split(email, "@")
	if len(parts) == 0 {
		return "user"
	}

	username := parts[0]
	// Clean up username (remove dots, etc.)
	username = strings.ReplaceAll(username, ".", "")
	username = strings.ReplaceAll(username, "+", "")

	return strings.ToLower(username)
}

// ensureUniqueUsername ensures the username is unique
func (as *AuthService) ensureUniqueUsername(baseUsername string) string {
	username := baseUsername
	counter := 1

	for {
		var existingUser models.User
		err := as.db.Where("username = ?", username).First(&existingUser).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return username
		}

		counter++
		username = fmt.Sprintf("%s%d", baseUsername, counter)

		// Prevent infinite loop
		if counter > 1000 {
			username = fmt.Sprintf("%s_%d", baseUsername, time.Now().UnixNano())
			break
		}
	}

	return username
}

// generateRandomState generates a random state for CSRF protection
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Standalone handler functions that can be used by any server implementation

// HandleGoogleLoginHandler initiates Google OAuth flow
func HandleGoogleLoginHandler(authService *AuthService, c *gin.Context) {
	oauthConfig := authService.SetupOAuth()

	authURL, state, err := authService.HandleGoogleLogin(oauthConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initiate Google login",
		})
		return
	}

	// Store state in session/cookie for validation
	c.SetCookie("oauth_state", state, 600, "/", "", false, true) // 10 minutes

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"auth_url": authURL,
		"state":    state,
	})
}

// HandleGoogleCallbackHandler handles Google OAuth callback
func HandleGoogleCallbackHandler(authService *AuthService, c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	// Validate state parameter
	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState != state {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid state parameter",
		})
		return
	}

	// Clear the state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	oauthConfig := authService.SetupOAuth()

	response, err := authService.HandleGoogleCallback(code, state, oauthConfig)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "OAuth authentication failed",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleQuickSignupHandler handles quick user registration
func HandleQuickSignupHandler(authService *AuthService, c *gin.Context) {
	var req QuickSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	response, err := authService.QuickSignup(req)
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

	c.JSON(http.StatusCreated, response)
}

// HandleGetCurrentUserHandler returns current user info
func HandleGetCurrentUserHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	userName := c.GetString("user_name")
	userEmail := c.GetString("user_email")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":       userID,
			"username": userName,
			"email":    userEmail,
			"name":     c.GetString("user_name"),
		},
	})
}

// Example route setup function (to be used in your server)
func SetupAuthRoutes(authService *AuthService, v1 *gin.RouterGroup) {
	auth := v1.Group("/auth")
	{
		// OAuth routes
		auth.GET("/google", func(c *gin.Context) {
			HandleGoogleLoginHandler(authService, c)
		})
		auth.GET("/google/callback", func(c *gin.Context) {
			HandleGoogleCallbackHandler(authService, c)
		})
		auth.POST("/signup", func(c *gin.Context) {
			HandleQuickSignupHandler(authService, c)
		})

		// Protected routes (require auth middleware)
		protected := auth.Group("")
		// protected.Use(authMiddleware) // Add your auth middleware here
		{
			protected.GET("/me", HandleGetCurrentUserHandler)
		}
	}
}
