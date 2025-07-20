package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// StateManager handles real-time state using Redis
type StateManager struct {
	client *redis.Client
	ctx    context.Context
}

// FileLock represents a file lock with metadata
type FileLock struct {
	FilePath  string    `json:"file_path"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	ProjectID string    `json:"project_id"`
	LockedAt  time.Time `json:"locked_at"`
	ExpiresAt time.Time `json:"expires_at"`
	LockType  LockType  `json:"lock_type"`
	SessionID string    `json:"session_id"`
}

// UserPresence represents a user's current activity
type UserPresence struct {
	UserID       string            `json:"user_id"`
	UserName     string            `json:"user_name"`
	ProjectID    string            `json:"project_id"`
	LastSeen     time.Time         `json:"last_seen"`
	Status       PresenceStatus    `json:"status"`
	CurrentFile  string            `json:"current_file,omitempty"`
	ActivityData map[string]string `json:"activity_data,omitempty"`
}

// CollaborationEvent represents real-time collaboration events
type CollaborationEvent struct {
	EventID   string                 `json:"event_id"`
	Type      EventType              `json:"type"`
	UserID    string                 `json:"user_id"`
	UserName  string                 `json:"user_name"`
	ProjectID string                 `json:"project_id"`
	FilePath  string                 `json:"file_path,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// LockType defines the type of file lock
type LockType string

const (
	LockTypeExclusive LockType = "exclusive"
	LockTypeShared    LockType = "shared"
)

// PresenceStatus defines user presence states
type PresenceStatus string

const (
	StatusOnline  PresenceStatus = "online"
	StatusEditing PresenceStatus = "editing"
	StatusIdle    PresenceStatus = "idle"
	StatusOffline PresenceStatus = "offline"
)

// EventType defines collaboration event types
type EventType string

const (
	EventFileLocked       EventType = "file_locked"
	EventFileUnlocked     EventType = "file_unlocked"
	EventFileModified     EventType = "file_modified"
	EventUserJoined       EventType = "user_joined"
	EventUserLeft         EventType = "user_left"
	EventUserIdle         EventType = "user_idle"
	EventConflictDetected EventType = "conflict_detected"
	EventCommitCreated    EventType = "commit_created"
)

// Redis key patterns
const (
	keyLockPrefix        = "lock:"
	keyPresencePrefix    = "presence:"
	keyProjectPrefix     = "project:"
	keySessionPrefix     = "session:"
	keyEventStreamPrefix = "events:"
	lockTTL              = 1 * time.Hour   // 1 hour lock expiration (was 24 hours)
	presenceTTL          = 5 * time.Minute // Presence heartbeat TTL
)

// NewStateManager creates a new Redis-based state manager
func NewStateManager(redisURL string) (*StateManager, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &StateManager{
		client: client,
		ctx:    ctx,
	}, nil
}

// File Locking Operations

// LockFile attempts to acquire an exclusive lock on a file
func (sm *StateManager) LockFile(projectID, filePath, userID, userName, sessionID string) (*FileLock, error) {
	lockKey := fmt.Sprintf("%s%s:%s", keyLockPrefix, projectID, filePath)

	lock := &FileLock{
		FilePath:  filePath,
		UserID:    userID,
		UserName:  userName,
		ProjectID: projectID,
		LockedAt:  time.Now(),
		ExpiresAt: time.Now().Add(lockTTL),
		LockType:  LockTypeExclusive,
		SessionID: sessionID,
	}

	lockData, err := json.Marshal(lock)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal lock data: %w", err)
	}

	// Atomic lock operation with expiration
	success, err := sm.client.SetNX(sm.ctx, lockKey, lockData, lockTTL).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !success {
		// Lock already exists, get current owner
		existingLock, err := sm.GetFileLock(projectID, filePath)
		if err != nil {
			return nil, fmt.Errorf("file is locked but couldn't get lock info: %w", err)
		}
		return nil, fmt.Errorf("file already locked by %s since %s", existingLock.UserName, existingLock.LockedAt.Format(time.RFC3339))
	}

	// Publish lock event
	sm.publishEvent(&CollaborationEvent{
		EventID:   generateEventID(),
		Type:      EventFileLocked,
		UserID:    userID,
		UserName:  userName,
		ProjectID: projectID,
		FilePath:  filePath,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"lock_type":  string(LockTypeExclusive),
			"session_id": sessionID,
		},
	})

	return lock, nil
}

// UnlockFile releases a file lock
func (sm *StateManager) UnlockFile(projectID, filePath, userID string) error {
	lockKey := fmt.Sprintf("%s%s:%s", keyLockPrefix, projectID, filePath)

	// Get current lock to verify ownership
	currentLock, err := sm.GetFileLock(projectID, filePath)
	if err != nil {
		// If lock doesn't exist, consider it already unlocked (success)
		if err.Error() == "file is not locked" {
			return nil
		}
		return fmt.Errorf("failed to get current lock: %w", err)
	}

	if currentLock.UserID != userID {
		return fmt.Errorf("cannot unlock file: owned by %s", currentLock.UserName)
	}

	// Delete the lock
	result, err := sm.client.Del(sm.ctx, lockKey).Result()
	if err != nil {
		return fmt.Errorf("failed to delete lock: %w", err)
	}

	if result == 0 {
		// Lock was already deleted, consider it success
		return nil
	}

	// Publish unlock event
	sm.publishEvent(&CollaborationEvent{
		EventID:   generateEventID(),
		Type:      EventFileUnlocked,
		UserID:    userID,
		UserName:  currentLock.UserName,
		ProjectID: projectID,
		FilePath:  filePath,
		Timestamp: time.Now(),
	})

	return nil
}

// GetFileLock retrieves information about a file lock
func (sm *StateManager) GetFileLock(projectID, filePath string) (*FileLock, error) {
	lockKey := fmt.Sprintf("%s%s:%s", keyLockPrefix, projectID, filePath)

	lockData, err := sm.client.Get(sm.ctx, lockKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("file is not locked")
		}
		return nil, fmt.Errorf("failed to get lock data: %w", err)
	}

	var lock FileLock
	if err := json.Unmarshal([]byte(lockData), &lock); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lock data: %w", err)
	}

	return &lock, nil
}

// ListProjectLocks returns all active locks for a project
func (sm *StateManager) ListProjectLocks(projectID string) ([]FileLock, error) {
	pattern := fmt.Sprintf("%s%s:*", keyLockPrefix, projectID)
	keys, err := sm.client.Keys(sm.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get lock keys: %w", err)
	}

	var locks []FileLock
	for _, key := range keys {
		lockData, err := sm.client.Get(sm.ctx, key).Result()
		if err != nil {
			continue // Skip invalid locks
		}

		var lock FileLock
		if err := json.Unmarshal([]byte(lockData), &lock); err != nil {
			continue // Skip invalid locks
		}

		locks = append(locks, lock)
	}

	return locks, nil
}

// User Presence Operations

// UpdatePresence updates user presence information
func (sm *StateManager) UpdatePresence(userID, userName, projectID string, status PresenceStatus, currentFile string) error {
	presenceKey := fmt.Sprintf("%s%s:%s", keyPresencePrefix, projectID, userID)

	presence := &UserPresence{
		UserID:      userID,
		UserName:    userName,
		ProjectID:   projectID,
		LastSeen:    time.Now(),
		Status:      status,
		CurrentFile: currentFile,
	}

	presenceData, err := json.Marshal(presence)
	if err != nil {
		return fmt.Errorf("failed to marshal presence data: %w", err)
	}

	// Set presence with TTL
	if err := sm.client.SetEx(sm.ctx, presenceKey, presenceData, presenceTTL).Err(); err != nil {
		return fmt.Errorf("failed to update presence: %w", err)
	}

	// Publish presence event
	eventType := EventUserJoined
	if status == StatusOffline {
		eventType = EventUserLeft
	} else if status == StatusIdle {
		eventType = EventUserIdle
	}

	sm.publishEvent(&CollaborationEvent{
		EventID:   generateEventID(),
		Type:      eventType,
		UserID:    userID,
		UserName:  userName,
		ProjectID: projectID,
		FilePath:  currentFile,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status": string(status),
		},
	})

	return nil
}

// GetProjectPresence returns all active users in a project
func (sm *StateManager) GetProjectPresence(projectID string) ([]UserPresence, error) {
	pattern := fmt.Sprintf("%s%s:*", keyPresencePrefix, projectID)
	keys, err := sm.client.Keys(sm.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get presence keys: %w", err)
	}

	var users []UserPresence
	for _, key := range keys {
		presenceData, err := sm.client.Get(sm.ctx, key).Result()
		if err != nil {
			continue // Skip expired presence
		}

		var presence UserPresence
		if err := json.Unmarshal([]byte(presenceData), &presence); err != nil {
			continue // Skip invalid presence
		}

		users = append(users, presence)
	}

	return users, nil
}

// Real-time Events

// PublishEvent publishes a collaboration event to Redis streams
func (sm *StateManager) PublishEvent(event *CollaborationEvent) error {
	return sm.publishEvent(event)
}

// SubscribeToEvents subscribes to collaboration events for a project
func (sm *StateManager) SubscribeToEvents(projectID string, eventChan chan<- *CollaborationEvent) error {
	streamKey := fmt.Sprintf("%s%s", keyEventStreamPrefix, projectID)

	// Start reading from the latest events
	lastID := "$"

	for {
		// Read new events from stream
		streams, err := sm.client.XRead(sm.ctx, &redis.XReadArgs{
			Streams: []string{streamKey, lastID},
			Block:   time.Second,
			Count:   10,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue // No new events
			}
			return fmt.Errorf("failed to read events: %w", err)
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				lastID = message.ID

				// Parse event data
				eventData, exists := message.Values["data"]
				if !exists {
					continue
				}

				var event CollaborationEvent
				if err := json.Unmarshal([]byte(eventData.(string)), &event); err != nil {
					continue // Skip invalid events
				}

				// Send to channel (non-blocking)
				select {
				case eventChan <- &event:
				default:
					// Channel full, skip this event
				}
			}
		}
	}
}

// Session Management

// CreateSession creates a new user session
func (sm *StateManager) CreateSession(userID, projectID string, metadata map[string]string) (string, error) {
	sessionID := generateSessionID()
	sessionKey := fmt.Sprintf("%s%s", keySessionPrefix, sessionID)

	sessionData := map[string]interface{}{
		"user_id":    userID,
		"project_id": projectID,
		"created_at": time.Now(),
		"metadata":   metadata,
	}

	sessionJSON, err := json.Marshal(sessionData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Set session with TTL
	if err := sm.client.SetEx(sm.ctx, sessionKey, sessionJSON, 24*time.Hour).Err(); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionID, nil
}

// ValidateSession checks if a session is valid
func (sm *StateManager) ValidateSession(sessionID string) (map[string]interface{}, error) {
	sessionKey := fmt.Sprintf("%s%s", keySessionPrefix, sessionID)

	sessionData, err := sm.client.Get(sm.ctx, sessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session map[string]interface{}
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return session, nil
}

// Cleanup Operations

// CleanupExpiredLocks removes expired file locks
func (sm *StateManager) CleanupExpiredLocks() error {
	pattern := fmt.Sprintf("%s*", keyLockPrefix)
	keys, err := sm.client.Keys(sm.ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get lock keys: %w", err)
	}

	now := time.Now()
	for _, key := range keys {
		lockData, err := sm.client.Get(sm.ctx, key).Result()
		if err != nil {
			continue
		}

		var lock FileLock
		if err := json.Unmarshal([]byte(lockData), &lock); err != nil {
			continue
		}

		if now.After(lock.ExpiresAt) {
			sm.client.Del(sm.ctx, key)
		}
	}

	return nil
}

// Private helper methods

func (sm *StateManager) publishEvent(event *CollaborationEvent) error {
	streamKey := fmt.Sprintf("%s%s", keyEventStreamPrefix, event.ProjectID)

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = sm.client.XAdd(sm.ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"data": string(eventData),
		},
		MaxLen: 1000, // Keep last 1000 events
		Approx: true,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// Close closes the Redis connection
func (sm *StateManager) Close() error {
	return sm.client.Close()
}
