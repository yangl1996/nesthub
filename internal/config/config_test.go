package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var cfg = Config{
	HubName:           "my",
	SDMProjectID:      "guitar",
	GCPProjectID:      "gently",
	OAuthClientID:     "weaps",
	OAuthClientSecret: "hey",
	OAuthTokenPath:    "jude",
	ServiceAccountKey: "here",
	PairingCode:       "comes",
	StoragePath:       "the",
	SetupRedirectUri:  "sun",
}

func TestValidateRequiredFields(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		tempConfig := cfg
		assert.NoError(t, tempConfig.validateRequiredFields())
	})

	t.Run("missing hub name", func(t *testing.T) {
		t.Parallel()
		tempConfig := cfg
		tempConfig.HubName = ""
		assert.Error(t, tempConfig.validateRequiredFields())
	})

	t.Run("missing other required fields", func(t *testing.T) {
		t.Parallel()
		tempConfig := Config{
			HubName: "some_name",
		}
		assert.Error(t, tempConfig.validateRequiredFields())
	})
}

func TestPopulateOptionalFields(t *testing.T) {
	t.Parallel()

	t.Run("no changes", func(t *testing.T) {
		t.Parallel()
		tempConfig := cfg
		tempConfig.populateOptionalFields()
		assert.Equal(t, "comes", tempConfig.PairingCode)
		assert.Equal(t, "the", tempConfig.StoragePath)
		assert.Equal(t, "sun", tempConfig.SetupRedirectUri)
	})

	t.Run("changes", func(t *testing.T) {
		t.Parallel()
		tempConfig := cfg
		tempConfig.PairingCode = ""
		tempConfig.StoragePath = ""
		tempConfig.SetupRedirectUri = ""
		tempConfig.populateOptionalFields()
		assert.Equal(t, "", tempConfig.PairingCode)
		assert.Equal(t, "", tempConfig.StoragePath)
		assert.Equal(t, "http://localhost:7979", tempConfig.SetupRedirectUri)
	})
}
