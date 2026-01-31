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

		// --- Create sbom_scans collection ---
		// Stores metadata about SBOM uploads and scan results
		if !collectionExists("sbom_scans") {
			sbomScansCollection := core.NewBaseCollection("sbom_scans")
			sbomScansCollection.Fields.Add(
				&core.TextField{
					Name:     "agent_id",
					Required: true,
				},
				&core.TextField{
					Name:     "user_id",
					Required: true,
				},
				&core.TextField{
					Name:     "scan_type",
					Required: true,
				}, // "host", "container"
				&core.TextField{
					Name:     "source_name",
					Required: false,
				}, // hostname or container name
				&core.TextField{
					Name:     "source_type",
					Required: false,
				}, // "filesystem", "podman", "docker", etc.
				&core.TextField{
					Name:     "status",
					Required: true,
				}, // "pending", "scanning", "completed", "failed"
				&core.NumberField{
					Name:     "total_packages",
					Required: false,
				},
				&core.NumberField{
					Name:     "critical_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "high_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "medium_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "low_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "negligible_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "unknown_count",
					Required: false,
				},
				&core.TextField{
					Name:     "error_message",
					Required: false,
				},
				&core.DateField{
					Name:     "scanned_at",
					Required: false,
				},
				&core.TextField{
					Name:     "grype_version",
					Required: false,
				},
				&core.TextField{
					Name:     "db_version",
					Required: false,
				},
			)

			// Set API rules
			sbomScansCollection.CreateRule = ptrString("@request.auth.id != ''")
			sbomScansCollection.ListRule = ptrString("@request.auth.id = user_id || @request.auth.id = agent_id")
			sbomScansCollection.ViewRule = ptrString("@request.auth.id = user_id || @request.auth.id = agent_id")
			sbomScansCollection.UpdateRule = ptrString("@request.auth.id = user_id || @request.auth.id = agent_id")
			sbomScansCollection.DeleteRule = ptrString("@request.auth.id = user_id")

			// Add indexes
			sbomScansCollection.Indexes = append(sbomScansCollection.Indexes,
				"CREATE INDEX idx_sbom_scans_agent ON sbom_scans(agent_id)",
				"CREATE INDEX idx_sbom_scans_user ON sbom_scans(user_id)",
				"CREATE INDEX idx_sbom_scans_status ON sbom_scans(status)",
			)

			if err := app.Save(sbomScansCollection); err != nil {
				return err
			}
		}

		// --- Create vulnerabilities collection ---
		// Stores individual vulnerability findings (linked to scans)
		if !collectionExists("vulnerabilities") {
			vulnCollection := core.NewBaseCollection("vulnerabilities")
			vulnCollection.Fields.Add(
				&core.TextField{
					Name:     "scan_id",
					Required: true,
				},
				&core.TextField{
					Name:     "agent_id",
					Required: true,
				},
				&core.TextField{
					Name:     "vulnerability_id",
					Required: true,
				}, // CVE-XXXX-XXXX or GHSA-XXXX
				&core.TextField{
					Name:     "severity",
					Required: true,
				}, // "critical", "high", "medium", "low", "negligible", "unknown"
				&core.TextField{
					Name:     "package_name",
					Required: true,
				},
				&core.TextField{
					Name:     "package_version",
					Required: true,
				},
				&core.TextField{
					Name:     "package_type",
					Required: false,
				}, // "rpm", "deb", "go-module", "npm", etc.
				&core.TextField{
					Name:     "fix_state",
					Required: false,
				}, // "fixed", "not-fixed", "wont-fix", "unknown"
				&core.TextField{
					Name:     "fix_versions",
					Required: false,
				}, // JSON array as string
				&core.TextField{
					Name:     "description",
					Required: false,
				},
				&core.TextField{
					Name:     "data_source",
					Required: false,
				},
				&core.TextField{
					Name:     "related_cves",
					Required: false,
				}, // JSON array as string
				&core.NumberField{
					Name:     "cvss_score",
					Required: false,
				},
				&core.TextField{
					Name:     "cvss_vector",
					Required: false,
				},
				&core.NumberField{
					Name:     "epss_score",
					Required: false,
				},
				&core.NumberField{
					Name:     "epss_percentile",
					Required: false,
				},
				&core.NumberField{
					Name:     "risk_score",
					Required: false,
				},
				&core.TextField{
					Name:     "artifact_locations",
					Required: false,
				}, // JSON array as string
			)

			// Set API rules
			vulnCollection.CreateRule = ptrString("@request.auth.id != ''")
			vulnCollection.ListRule = ptrString("@request.auth.id != ''")
			vulnCollection.ViewRule = ptrString("@request.auth.id != ''")
			vulnCollection.UpdateRule = nil // No direct updates
			vulnCollection.DeleteRule = nil // Deleted via scan deletion

			// Add indexes for efficient querying
			vulnCollection.Indexes = append(vulnCollection.Indexes,
				"CREATE INDEX idx_vuln_agent ON vulnerabilities(agent_id)",
				"CREATE INDEX idx_vuln_scan ON vulnerabilities(scan_id)",
				"CREATE INDEX idx_vuln_severity ON vulnerabilities(severity)",
				"CREATE INDEX idx_vuln_id ON vulnerabilities(vulnerability_id)",
			)

			if err := app.Save(vulnCollection); err != nil {
				return err
			}
		}

		// --- Create vulnerability_summary collection ---
		// Stores aggregated vulnerability data per agent for quick dashboard display
		if !collectionExists("vulnerability_summary") {
			summaryCollection := core.NewBaseCollection("vulnerability_summary")
			summaryCollection.Fields.Add(
				&core.TextField{
					Name:     "agent_id",
					Required: true,
				},
				&core.TextField{
					Name:     "user_id",
					Required: true,
				},
				&core.NumberField{
					Name:     "total_vulnerabilities",
					Required: false,
				},
				&core.NumberField{
					Name:     "critical_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "high_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "medium_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "low_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "fixable_count",
					Required: false,
				},
				&core.NumberField{
					Name:     "total_scans",
					Required: false,
				},
				&core.DateField{
					Name:     "last_scan_at",
					Required: false,
				},
				&core.TextField{
					Name:     "last_scan_id",
					Required: false,
				},
			)

			// Set API rules
			summaryCollection.CreateRule = ptrString("@request.auth.id != ''")
			summaryCollection.ListRule = ptrString("@request.auth.id = user_id || @request.auth.id = agent_id")
			summaryCollection.ViewRule = ptrString("@request.auth.id = user_id || @request.auth.id = agent_id")
			summaryCollection.UpdateRule = ptrString("@request.auth.id = user_id || @request.auth.id = agent_id")
			summaryCollection.DeleteRule = ptrString("@request.auth.id = user_id")

			// Add unique index on agent_id
			summaryCollection.Indexes = append(summaryCollection.Indexes,
				"CREATE UNIQUE INDEX idx_summary_agent ON vulnerability_summary(agent_id)",
			)

			if err := app.Save(summaryCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback - delete collections
		collections := []string{"vulnerability_summary", "vulnerabilities", "sbom_scans"}
		for _, name := range collections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err == nil {
				if err := app.Delete(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
