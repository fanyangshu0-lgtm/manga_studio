package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type SessionManager struct {
	key []byte
	ttl time.Duration
}

type sessionPayload struct {
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expiresAt"`
}

func NewSessionManager(secret string, ttl time.Duration) *SessionManager {
	key := sha256.Sum256([]byte(secret))
	return &SessionManager{key: key[:], ttl: ttl}
}

func (m *SessionManager) Issue(username string, now time.Time) (string, error) {
	payload, err := json.Marshal(sessionPayload{Username: username, ExpiresAt: now.Add(m.ttl).Unix()})
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, m.key)
	_, _ = mac.Write([]byte(encoded))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + signature, nil
}

func (m *SessionManager) Verify(token string, now time.Time) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid session")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("invalid session")
	}
	mac := hmac.New(sha256.New, m.key)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", errors.New("invalid session")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("invalid session")
	}
	var payload sessionPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil || payload.Username == "" {
		return "", errors.New("invalid session")
	}
	if now.Unix() >= payload.ExpiresAt {
		return "", errors.New("session expired")
	}
	return payload.Username, nil
}
