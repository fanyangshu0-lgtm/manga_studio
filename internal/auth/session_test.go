package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionRoundTripAndTamperRejection(t *testing.T) {
	manager := NewSessionManager("01234567890123456789012345678901", 12*time.Hour)
	token, err := manager.Issue("admin", time.Unix(1000, 0))
	require.NoError(t, err)
	username, err := manager.Verify(token, time.Unix(1001, 0))
	require.NoError(t, err)
	assert.Equal(t, "admin", username)
	_, err = manager.Verify(token+"x", time.Unix(1001, 0))
	assert.Error(t, err)
	_, err = manager.Verify(token, time.Unix(1000, 0).Add(13*time.Hour))
	assert.Error(t, err)
}
