package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretCodecDecryptsAndRejectsWrongKey(t *testing.T) {
	encoded, err := newSecretCodec("first-secret").Encrypt("sk-private")
	require.NoError(t, err)
	value, err := newSecretCodec("first-secret").Decrypt(encoded)
	require.NoError(t, err)
	assert.Equal(t, "sk-private", value)
	_, err = newSecretCodec("second-secret").Decrypt(encoded)
	assert.Error(t, err)
}
