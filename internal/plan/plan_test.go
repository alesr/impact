package plan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBytes(t *testing.T) {
	t.Parallel()

	t.Run("parses resource changes and inherited defaults", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{
			"format_version": "1.2",
			"terraform_version": "1.6.0",
			"variables": {
				"region": {"value": "fr-par"},
				"zone": {"value": "fr-par-1"}
			},
			"resource_changes": [
				{
					"address": "scaleway_instance_server.web",
					"type": "scaleway_instance_server",
					"name": "web",
					"change": {
						"actions": ["create"],
						"before": null,
						"after": {"zone": "fr-par-2"}
					}
				}
			]
		}`)

		changes, err := ParseBytes(data)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		assert.Equal(t, "scaleway_instance_server", changes[0].Type)
		assert.Equal(t, "fr-par-2", changes[0].After["zone"])
		assert.Equal(t, "fr-par", changes[0].Region)
		assert.Equal(t, "fr-par-1", changes[0].Zone)
	})

	t.Run("applies plan-level defaults when resource attributes are nil", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{
			"format_version": "1.2",
			"terraform_version": "1.6.0",
			"variables": {
				"region": {"value": "nl-ams"},
				"zone": {"value": "nl-ams-1"}
			},
			"resource_changes": [
				{
					"address": "scaleway_lb.edge",
					"type": "scaleway_lb",
					"name": "edge",
					"change": {
						"actions": ["create"],
						"before": null,
						"after": {"type": "LB-S", "zone": null}
					}
				}
			]
		}`)

		changes, err := ParseBytes(data)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		assert.Equal(t, "nl-ams", changes[0].Region)
		assert.Equal(t, "nl-ams-1", changes[0].Zone)
	})

	t.Run("returns error when payload exceeds max size", func(t *testing.T) {
		t.Parallel()

		data := make([]byte, maxPlanFileBytes+1)
		_, err := ParseBytes(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payload too large")
	})
}

func TestParseFile(t *testing.T) {
	t.Parallel()

	t.Run("returns error when file exceeds max size", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "too-large.json")
		require.NoError(t, os.WriteFile(path, []byte("{}"), 0o600))
		require.NoError(t, os.Truncate(path, maxPlanFileBytes+1))

		_, err := ParseFile(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file too large")
	})
}
