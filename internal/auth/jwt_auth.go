package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Telerallc/gamedev-vcs/models"
)

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// AuthService handles authentication operations
type AuthService struct {
	db        *gorm.DB
	jwtSecret []byte
}

// NewAuthService creates a new authentication service
func NewAuthService(db *gorm.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: []byte(jwtSecret),
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success      bool     `json:"success"`
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresAt    int64    `json:"expires_at"`
	User         UserInfo `json:"user"`
}

// UserInfo represents basic user information
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

// Register creates a new user account
func (as *AuthService) Register(req RegisterRequest) (*LoginResponse, error) {
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
			"created_via":   "registration",
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

// Login authenticates a user and returns JWT tokens
func (as *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	// Find user by username or email
	var user models.User
	err := as.db.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Verify password
	if !as.verifyPassword(user, req.Password) {
		return nil, fmt.Errorf("invalid credentials")
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

// RefreshToken generates a new access token from a refresh token
func (as *AuthService) RefreshToken(refreshToken string) (*LoginResponse, error) {
	// Parse and validate refresh token
	token, err := jwt.ParseWithClaims(refreshToken, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return as.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	// Check if token is for refresh (should have longer expiry)
	if claims.RegisteredClaims.Subject != "refresh" {
		return nil, fmt.Errorf("not a refresh token")
	}

	// Get user from database
	var user models.User
	err = as.db.Where("id = ?", claims.UserID).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Generate new tokens
	newToken, newRefreshToken, expiresAt, err := as.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &LoginResponse{
		Success:      true,
		Token:        newToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Name:     user.Name,
		},
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (as *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return as.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check if token is an access token (not refresh)
	if claims.RegisteredClaims.Subject == "refresh" {
		return nil, fmt.Errorf("refresh token used where access token expected")
	}

	return claims, nil
}

// generateTokens generates both access and refresh tokens
func (as *AuthService) generateTokens(user models.User) (string, string, int64, error) {
	now := time.Now()
	accessExpiry := now.Add(24 * time.Hour)
	refreshExpiry := now.Add(30 * 24 * time.Hour) // 30 days

	// Access token claims
	accessClaims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "access",
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "gamedev-vcs",
		},
	}

	// Refresh token claims
	refreshClaims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "refresh",
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "gamedev-vcs",
		},
	}

	// Generate access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(as.jwtSecret)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(as.jwtSecret)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessTokenString, refreshTokenString, accessExpiry.Unix(), nil
}

// verifyPassword checks if the provided password matches the user's stored password
func (as *AuthService) verifyPassword(user models.User, password string) bool {
	if user.Settings == nil {
		return false
	}

	passwordHash, exists := user.Settings["password_hash"]
	if !exists {
		return false
	}

	hashStr, ok := passwordHash.(string)
	if !ok {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashStr), []byte(password))
	return err == nil
}

// GetUserByID retrieves a user by ID
func (as *AuthService) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	err := as.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// If you don't have generateTokens as a public method, add this:
func (as *AuthService) GenerateTokens(user models.User) (string, string, int64, error) {
	return as.generateTokens(user)
}
