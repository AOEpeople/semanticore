package internal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTryCreateSignKey(t *testing.T) {
	// Setup test data
	testPrivateKeyFilePath := "test/private-key.asc"
	keyData, err := os.ReadFile(testPrivateKeyFilePath)
	assert.NoError(t, err, "Failed to read test key file")
	validKeyContent := string(keyData)

	t.Run("valid file path", func(t *testing.T) {
		entity, err := TryCreateSignKey(&testPrivateKeyFilePath)
		assert.NoError(t, err)
		assert.NotNil(t, entity, "Expected entity, got nil")
	})

	t.Run("valid environment variable", func(t *testing.T) {
		t.Setenv("SEMANTICORE_SIGN_KEY", validKeyContent)

		emptyPath := ""
		entity, err := TryCreateSignKey(&emptyPath)

		assert.NoError(t, err)
		assert.NotNil(t, entity, "Expected entity, got nil")
	})

	t.Run("both file and env var set", func(t *testing.T) {
		t.Setenv("SEMANTICORE_SIGN_KEY", validKeyContent)

		filePath := "dummy.asc"
		_, err := TryCreateSignKey(&filePath)

		assert.Error(t, err, "Expected error when both file and env var are set")
	})

	t.Run("no signing key provided", func(t *testing.T) {
		t.Setenv("SEMANTICORE_SIGN_KEY", "")
		emptyPath := ""
		entity, err := TryCreateSignKey(&emptyPath)

		assert.ErrorIs(t, err, ErrNoSigningKeyFound)
		assert.Nil(t, entity, "Expected nil entity when no key is provided")
	})

	t.Run("signing key file path nil", func(t *testing.T) {
		entity, err := TryCreateSignKey(nil)

		assert.ErrorIs(t, err, ErrNoSigningKeyFound)
		assert.Nil(t, entity, "Expected nil entity when no key is provided")
	})
}
