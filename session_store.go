package main

import (
	"sync"
	"time"
)

// DeviceAuthorization represents a device authorization request
type DeviceAuthorization struct {
	DeviceCode string
	UserCode   string
	ClientID   string
	Scopes     []string
	ExpiresAt  time.Time
	Authorized bool
	UserID     string
}

// MemorySessionStore provides in-memory storage for sessions and device authorizations
type MemorySessionStore struct {
	deviceAuthorizations map[string]*DeviceAuthorization
	userCodeMap         map[string]string // maps user_code to device_code
	mutex               sync.RWMutex
}

// NewMemorySessionStore creates a new in-memory session store
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		deviceAuthorizations: make(map[string]*DeviceAuthorization),
		userCodeMap:         make(map[string]string),
	}
}

// StoreDeviceAuthorization stores a device authorization
func (s *MemorySessionStore) StoreDeviceAuthorization(deviceCode string, auth *DeviceAuthorization) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.deviceAuthorizations[deviceCode] = auth
	s.userCodeMap[auth.UserCode] = deviceCode
}

// GetDeviceAuthorization retrieves a device authorization by device code
func (s *MemorySessionStore) GetDeviceAuthorization(deviceCode string) *DeviceAuthorization {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return s.deviceAuthorizations[deviceCode]
}

// GetDeviceAuthorizationByUserCode retrieves a device authorization by user code
func (s *MemorySessionStore) GetDeviceAuthorizationByUserCode(userCode string) *DeviceAuthorization {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	if deviceCode, exists := s.userCodeMap[userCode]; exists {
		return s.deviceAuthorizations[deviceCode]
	}
	return nil
}

// DeleteDeviceAuthorization removes a device authorization
func (s *MemorySessionStore) DeleteDeviceAuthorization(deviceCode string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if auth := s.deviceAuthorizations[deviceCode]; auth != nil {
		delete(s.userCodeMap, auth.UserCode)
		delete(s.deviceAuthorizations, deviceCode)
	}
}

// CleanupExpired removes expired device authorizations
func (s *MemorySessionStore) CleanupExpired() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	now := time.Now()
	for deviceCode, auth := range s.deviceAuthorizations {
		if now.After(auth.ExpiresAt) {
			delete(s.userCodeMap, auth.UserCode)
			delete(s.deviceAuthorizations, deviceCode)
		}
	}
}
