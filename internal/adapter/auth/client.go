package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrUnavailable  = errors.New("auth service unavailable")
)

type User struct {
	Subject     string   `json:"sub"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"sessionId"`
	Type        string   `json:"type"`
	ExpiresAt   int64    `json:"exp"`
}

func (user User) HasPermission(permission string) bool {
	for _, current := range user.Permissions {
		if current == permission {
			return true
		}
	}
	return false
}

type cacheEntry struct {
	user      User
	expiresAt time.Time
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	cacheTTL   time.Duration
	mutex      sync.RWMutex
	cache      map[string]cacheEntry
}

func NewClient(baseURL string, cacheTTL time.Duration) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		cacheTTL:   cacheTTL,
		cache:      make(map[string]cacheEntry),
	}
}

func (client *Client) Validate(ctx context.Context, token string) (User, error) {
	key := tokenHash(token)
	if user, found := client.cached(key); found {
		return user, nil
	}
	body, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		return User{}, err
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		client.baseURL+"/api/v1/auth/validate-token",
		bytes.NewReader(body),
	)
	if err != nil {
		return User{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return User{}, ErrUnauthorized
	}
	if response.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("%w: status %d", ErrUnavailable, response.StatusCode)
	}
	var user User
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		return User{}, fmt.Errorf("%w: decode response: %v", ErrUnavailable, err)
	}
	if user.Type != "access" && user.Type != "service" {
		return User{}, ErrUnauthorized
	}
	client.store(key, user)
	return user, nil
}

func (client *Client) cached(key string) (User, bool) {
	client.mutex.RLock()
	entry, found := client.cache[key]
	client.mutex.RUnlock()
	if !found {
		return User{}, false
	}
	if time.Now().After(entry.expiresAt) {
		client.mutex.Lock()
		delete(client.cache, key)
		client.mutex.Unlock()
		return User{}, false
	}
	return entry.user, true
}

func (client *Client) store(key string, user User) {
	expiresAt := time.Now().Add(client.cacheTTL)
	if user.ExpiresAt > 0 {
		tokenExpiry := time.Unix(user.ExpiresAt, 0)
		if tokenExpiry.Before(expiresAt) {
			expiresAt = tokenExpiry
		}
	}
	if expiresAt.Before(time.Now()) {
		return
	}
	client.mutex.Lock()
	client.cache[key] = cacheEntry{user: user, expiresAt: expiresAt}
	client.mutex.Unlock()
}

func tokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
