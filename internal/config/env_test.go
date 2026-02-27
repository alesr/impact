package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadScalewayFromEnv(t *testing.T) {
	t.Run("returns defaults when env is empty", func(t *testing.T) {
		t.Setenv("IMPACT_SCW_API_BASE_URL", "")
		t.Setenv("SCW_ACCESS_KEY", "")
		t.Setenv("SCW_SECRET_KEY", "")
		t.Setenv("SCW_ORGANIZATION_ID", "")

		cfg, err := LoadScalewayFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "https://api.scaleway.com", cfg.APIBaseURL)
		assert.Empty(t, cfg.AccessKey)
		assert.Empty(t, cfg.SecretKey)
		assert.Empty(t, cfg.OrganizationID)
	})

	t.Run("reads configured values from env", func(t *testing.T) {
		t.Setenv("IMPACT_SCW_API_BASE_URL", "https://example.invalid")
		t.Setenv("SCW_ACCESS_KEY", "SCWXXXXXXXXXXXXXXXXX")
		t.Setenv("SCW_SECRET_KEY", "secret")
		t.Setenv("SCW_ORGANIZATION_ID", "org-id")

		cfg, err := LoadScalewayFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "https://example.invalid", cfg.APIBaseURL)
		assert.Equal(t, "SCWXXXXXXXXXXXXXXXXX", cfg.AccessKey)
		assert.Equal(t, "secret", cfg.SecretKey)
		assert.Equal(t, "org-id", cfg.OrganizationID)
	})

	t.Run("returns error for invalid base url", func(t *testing.T) {
		t.Setenv("IMPACT_SCW_API_BASE_URL", "not-a-url")

		_, err := LoadScalewayFromEnv()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not validate IMPACT_SCW_API_BASE_URL")
	})

	t.Run("returns error for non-https base url", func(t *testing.T) {
		t.Setenv("IMPACT_SCW_API_BASE_URL", "http://api.scaleway.com")

		_, err := LoadScalewayFromEnv()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "https scheme is required")
	})
}
