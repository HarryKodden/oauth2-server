package auth

import (
	"time"

	"github.com/ory/fosite"
)

// UserSession represents a user session for OAuth2 flows and implements fosite.Session
type UserSession struct {
	UserID    string                 `json:"user_id"`
	Username  string                 `json:"username"`
	Subject   string                 `json:"subject"`
	Extra     map[string]interface{} `json:"extra"`
	ExpiresAt map[string]time.Time   `json:"expires_at"`
}

// GetSubject returns the subject (user ID) for the session - required by fosite.Session
func (s *UserSession) GetSubject() string {
	if s.Subject != "" {
		return s.Subject
	}
	return s.UserID
}

// GetUsername returns the username for the session
func (s *UserSession) GetUsername() string {
	return s.Username
}

// GetExpiresAt returns the expiration time for a specific token type - required by fosite.Session
func (s *UserSession) GetExpiresAt(tokenType fosite.TokenType) time.Time {
	if s.ExpiresAt == nil {
		return time.Time{}
	}
	return s.ExpiresAt[string(tokenType)]
}

// SetExpiresAt sets the expiration time for a specific token type - required by fosite.Session
func (s *UserSession) SetExpiresAt(tokenType fosite.TokenType, exp time.Time) {
	if s.ExpiresAt == nil {
		s.ExpiresAt = make(map[string]time.Time)
	}
	s.ExpiresAt[string(tokenType)] = exp
}

// Clone creates a copy of the session - required by fosite.Session
// This must return fosite.Session to satisfy the interface
func (s *UserSession) Clone() fosite.Session {
	clone := &UserSession{
		UserID:   s.UserID,
		Username: s.Username,
		Subject:  s.Subject,
	}

	if s.Extra != nil {
		clone.Extra = make(map[string]interface{})
		for k, v := range s.Extra {
			clone.Extra[k] = v
		}
	}

	if s.ExpiresAt != nil {
		clone.ExpiresAt = make(map[string]time.Time)
		for k, v := range s.ExpiresAt {
			clone.ExpiresAt[k] = v
		}
	}

	return clone
}

// GetExtra returns extra information stored in the session - sometimes required by fosite
func (s *UserSession) GetExtra(key string) interface{} {
	if s.Extra == nil {
		return nil
	}
	return s.Extra[key]
}

// SetExtra stores extra information in the session - sometimes required by fosite
func (s *UserSession) SetExtra(key string, value interface{}) {
	if s.Extra == nil {
		s.Extra = make(map[string]interface{})
	}
	s.Extra[key] = value
}
