package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Helper to check if collection exists
		collectionExists := func(name string) bool {
			_, err := app.FindCollectionByNameOrId(name)
			return err == nil
		}

		// --- Create sbom_settings collection ---
		// Stores global SBOM/vulnerability scanning settings configurable by superusers
		if !collectionExists("sbom_settings") {
			settingsCollection := core.NewBaseCollection("sbom_settings")
			settingsCollection.Fields.Add(
				&core.TextField{
					Name:     "key",
					Required: true,
				},
				&core.TextField{
					Name:     "value",
					Required: false,
				},
				&core.TextField{
					Name:     "description",
					Required: false,
				},
				&core.TextField{
					Name:     "value_type",
					Required: false,
				}, // "string", "bool", "number", "cron"
				&core.TextField{
					Name:     "updated_by",
					Required: false,
				}, // user_id who last updated
			)

			// Only superusers can manage settings
			settingsCollection.ListRule = ptrString("@request.auth.role = 'superuser'")
			settingsCollection.ViewRule = ptrString("@request.auth.role = 'superuser'")
			settingsCollection.CreateRule = ptrString("@request.auth.role = 'superuser'")
			settingsCollection.UpdateRule = ptrString("@request.auth.role = 'superuser'")
			settingsCollection.DeleteRule = ptrString("@request.auth.role = 'superuser'")

			// Add unique index on key
			settingsCollection.Indexes = append(settingsCollection.Indexes,
				"CREATE UNIQUE INDEX idx_sbom_settings_key ON sbom_settings(key)",
			)

			if err := app.Save(settingsCollection); err != nil {
				return err
			}

			// Insert default settings
			defaults := []struct {
				Key         string
				Value       string
				Description string
				ValueType   string
			}{
				{
					Key:         "grype_db_update_cron",
					Value:       "0 3 * * *",
					Description: "Cron expression for automatic grype database updates (default: daily at 3 AM)",
					ValueType:   "cron",
				},
				{
					Key:         "grype_db_auto_update",
					Value:       "true",
					Description: "Enable automatic grype database updates",
					ValueType:   "bool",
				},
				{
					Key:         "default_min_severity",
					Value:       "low",
					Description: "Default minimum severity to report (critical, high, medium, low, negligible)",
					ValueType:   "string",
				},
				{
					Key:         "default_min_cvss",
					Value:       "0",
					Description: "Default minimum CVSS score to report (0-10)",
					ValueType:   "number",
				},
				{
					Key:         "max_vulnerabilities_per_scan",
					Value:       "10000",
					Description: "Maximum number of vulnerabilities to store per scan",
					ValueType:   "number",
				},
				{
					Key:         "retention_days",
					Value:       "90",
					Description: "Number of days to retain scan data",
					ValueType:   "number",
				},
			}

			for _, d := range defaults {
				record := core.NewRecord(settingsCollection)
				record.Set("key", d.Key)
				record.Set("value", d.Value)
				record.Set("description", d.Description)
				record.Set("value_type", d.ValueType)
				if err := app.Save(record); err != nil {
					app.Logger().Warn("Failed to create default sbom_setting",
						"key", d.Key,
						"error", err)
				}
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete collection
		collection, err := app.FindCollectionByNameOrId("sbom_settings")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
