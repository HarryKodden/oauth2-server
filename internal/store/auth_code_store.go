package store

import (
	"errors"
	"sync"
	"time"
)

// AuthCode represents an authorization code
type AuthCode struct {
	Code        string    `json:"code"`
	ClientID    string    `json:"client_id"`
	UserID      string    `json:"user_id"`
	RedirectURI string    `json:"redirect_uri"`
	Scopes      []string  `json:"scopes"`
	ExpiresAt   time.Time `json:"expires_at"`
	Used        bool      `json:"used"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuthCodeStore manages authorization codes
type AuthCodeStore struct {
	codes map[string]*AuthCode
	mutex sync.RWMutex
}

// NewAuthCodeStore creates a new authorization code store
func NewAuthCodeStore() *AuthCodeStore {
	return &AuthCodeStore{
		codes: make(map[string]*AuthCode),
	}
}

// StoreAuthCode stores an authorization code
func (s *AuthCodeStore) StoreAuthCode(code *AuthCode) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.codes[code.Code] = code
	return nil
}

// GetAuthCode retrieves an authorization code
func (s *AuthCodeStore) GetAuthCode(code string) (*AuthCode, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	authCode, exists := s.codes[code]
	if !exists {
		return nil, errors.New("authorization code not found")
	}

	return authCode, nil
}

// UseAuthCode marks an authorization code as used
func (s *AuthCodeStore) UseAuthCode(code string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	authCode, exists := s.codes[code]
	if !exists {
		return errors.New("authorization code not found")
	}

	if authCode.Used {
		return errors.New("authorization code already used")
	}

	if time.Now().After(authCode.ExpiresAt) {
		return errors.New("authorization code expired")
	}

	authCode.Used = true
	return nil
}

// DeleteAuthCode removes an authorization code
func (s *AuthCodeStore) DeleteAuthCode(code string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.codes, code)
	return nil
}

// CleanupExpiredCodes removes expired authorization codes
func (s *AuthCodeStore) CleanupExpiredCodes() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	var expiredCodes []string

	for code, authCode := range s.codes {
		if now.After(authCode.ExpiresAt) {
			expiredCodes = append(expiredCodes, code)
		}
	}

	for _, code := range expiredCodes {
		delete(s.codes, code)
	}

	return len(expiredCodes)
}
