package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// --- Add archive_path field to sbom_scans collection ---
		// This stores the path to the grype output archive instead of storing
		// individual vulnerabilities in the database
		scansCollection, err := app.FindCollectionByNameOrId("sbom_scans")
		if err != nil {
			return err
		}

		// Check if field already exists
		if scansCollection.Fields.GetByName("archive_path") == nil {
			scansCollection.Fields.Add(&core.TextField{
				Name:     "archive_path",
				Required: false,
			})

			if err := app.Save(scansCollection); err != nil {
				return err
			}
		}

		// --- Add syft_exclude_patterns field to agents collection ---
		// Stores per-agent syft exclusion patterns
		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			// Agents collection might not exist yet, skip this part
			app.Logger().Warn("Agents collection not found, skipping syft_exclude_patterns field")
		} else {
			if agentsCollection.Fields.GetByName("syft_exclude_patterns") == nil {
				agentsCollection.Fields.Add(&core.JSONField{
					Name:     "syft_exclude_patterns",
					Required: false,
				})

				if err := app.Save(agentsCollection); err != nil {
					return err
				}
			}
		}

		// --- Add scans_per_agent setting to sbom_settings ---
		settingsCollection, err := app.FindCollectionByNameOrId("sbom_settings")
		if err != nil {
			app.Logger().Warn("sbom_settings collection not found, skipping scans_per_agent")
			return nil
		}

		// Check if scans_per_agent setting already exists
		existingRecord, _ := app.FindFirstRecordByData("sbom_settings", "key", "scans_per_agent")
		if existingRecord == nil {
			record := core.NewRecord(settingsCollection)
			record.Set("key", "scans_per_agent")
			record.Set("value", "10")
			record.Set("description", "Maximum number of SBOM scan results to retain per agent. Older scans are automatically deleted.")
			record.Set("value_type", "number")
			if err := app.Save(record); err != nil {
				app.Logger().Warn("Failed to create scans_per_agent setting", "error", err)
			}
		}

		// Add default_syft_exclude_patterns setting
		existingPatterns, _ := app.FindFirstRecordByData("sbom_settings", "key", "default_syft_exclude_patterns")
		if existingPatterns == nil {
			record := core.NewRecord(settingsCollection)
			record.Set("key", "default_syft_exclude_patterns")
			record.Set("value", `["**/proc/**","**/sys/**","**/dev/**","**/run/**","**/tmp/**","**/var/cache/**","**/var/log/**","**/home/*/.cache/**"]`)
			record.Set("description", "Default syft exclusion patterns for host filesystem scans (JSON array). These are used when agent has no custom patterns configured.")
			record.Set("value_type", "json")
			if err := app.Save(record); err != nil {
				app.Logger().Warn("Failed to create default_syft_exclude_patterns setting", "error", err)
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove added fields and settings
		scansCollection, err := app.FindCollectionByNameOrId("sbom_scans")
		if err == nil {
			if field := scansCollection.Fields.GetByName("archive_path"); field != nil {
				scansCollection.Fields.RemoveByName("archive_path")
				_ = app.Save(scansCollection)
			}
		}

		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err == nil {
			if field := agentsCollection.Fields.GetByName("syft_exclude_patterns"); field != nil {
				agentsCollection.Fields.RemoveByName("syft_exclude_patterns")
				_ = app.Save(agentsCollection)
			}
		}

		// Remove settings
		if record, _ := app.FindFirstRecordByData("sbom_settings", "key", "scans_per_agent"); record != nil {
			_ = app.Delete(record)
		}
		if record, _ := app.FindFirstRecordByData("sbom_settings", "key", "default_syft_exclude_patterns"); record != nil {
			_ = app.Delete(record)
		}

		return nil
	})
}
