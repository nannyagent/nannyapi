package schedules

import (
	"strconv"
	"strings"
	"time"

	"github.com/nannyagent/nannyapi/internal/sbom"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// GrypeDBUpdateCronID is the cron job ID for grype DB updates
	GrypeDBUpdateCronID = "grype_db_update"

	// DefaultGrypeDBUpdateCron runs daily at 3 AM
	DefaultGrypeDBUpdateCron = "0 3 * * *"

	// Settings keys in sbom_settings collection
	SettingGrypeDBUpdateCron         = "grype_db_update_cron"
	SettingGrypeDBAutoUpdate         = "grype_db_auto_update"
	SettingDefaultMinSeverity        = "default_min_severity"
	SettingDefaultMinCVSS            = "default_min_cvss"
	SettingMaxVulnerabilitiesPerScan = "max_vulnerabilities_per_scan"
	SettingRetentionDays             = "retention_days"
)

// GrypeDBScheduler manages grype database update schedules
type GrypeDBScheduler struct {
	app     core.App
	scanner *sbom.Scanner
}

// NewGrypeDBScheduler creates a new grype database scheduler
func NewGrypeDBScheduler(app core.App, scanner *sbom.Scanner) *GrypeDBScheduler {
	return &GrypeDBScheduler{
		app:     app,
		scanner: scanner,
	}
}

// getSetting retrieves a setting from sbom_settings collection
func (s *GrypeDBScheduler) getSetting(key string) (string, error) {
	collection, err := s.app.FindCollectionByNameOrId("sbom_settings")
	if err != nil {
		return "", err
	}

	records, err := s.app.FindRecordsByFilter(collection, "key = {:key}", "", 1, 0, map[string]any{"key": key})
	if err != nil || len(records) == 0 {
		return "", err
	}

	return records[0].GetString("value"), nil
}

// getCronFromSettings retrieves cron expression from database settings
func (s *GrypeDBScheduler) getCronFromSettings() string {
	value, err := s.getSetting(SettingGrypeDBUpdateCron)
	if err != nil || value == "" {
		return DefaultGrypeDBUpdateCron
	}
	return value
}

// isAutoUpdateEnabled checks if auto-update is enabled from settings
func (s *GrypeDBScheduler) isAutoUpdateEnabled() bool {
	value, err := s.getSetting(SettingGrypeDBAutoUpdate)
	if err != nil || value == "" {
		return true // Default to enabled
	}
	return strings.ToLower(value) == "true" || value == "1"
}

// GetMinSeverity retrieves the default minimum severity filter from settings
func GetMinSeverity(app core.App) string {
	scheduler := &GrypeDBScheduler{app: app}
	value, err := scheduler.getSetting(SettingDefaultMinSeverity)
	if err != nil || value == "" {
		return "low" // Default: show all severities
	}
	return strings.ToLower(value)
}

// GetMinCVSS retrieves the default minimum CVSS score from settings
func GetMinCVSS(app core.App) float64 {
	scheduler := &GrypeDBScheduler{app: app}
	value, err := scheduler.getSetting(SettingDefaultMinCVSS)
	if err != nil || value == "" {
		return 0.0 // Default: no CVSS filter
	}
	cvss, _ := strconv.ParseFloat(value, 64)
	return cvss
}

// GetMaxVulnerabilitiesPerScan retrieves max vulnerabilities limit from settings
func GetMaxVulnerabilitiesPerScan(app core.App) int {
	scheduler := &GrypeDBScheduler{app: app}
	value, err := scheduler.getSetting(SettingMaxVulnerabilitiesPerScan)
	if err != nil || value == "" {
		return 10000 // Default: 10K limit
	}
	max, _ := strconv.Atoi(value)
	if max <= 0 {
		return 10000
	}
	return max
}

// GetRetentionDays retrieves the scan retention period from settings
func GetRetentionDays(app core.App) int {
	scheduler := &GrypeDBScheduler{app: app}
	value, err := scheduler.getSetting(SettingRetentionDays)
	if err != nil || value == "" {
		return 90 // Default: 90 days
	}
	days, _ := strconv.Atoi(value)
	if days <= 0 {
		return 90
	}
	return days
}

// RegisterGrypeDBScheduler registers the grype DB update cron job
// Settings are now read from the sbom_settings database collection
func RegisterGrypeDBScheduler(app core.App, scanner *sbom.Scanner) {
	if scanner == nil || !scanner.IsEnabled() {
		app.Logger().Info("Grype DB scheduler not registered - vulnerability scanning is disabled")
		return
	}

	scheduler := NewGrypeDBScheduler(app, scanner)

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Check if auto-update is enabled
		if !scheduler.isAutoUpdateEnabled() {
			app.Logger().Info("Grype DB auto-update is disabled via sbom_settings")
			return e.Next()
		}

		// Get cron expression from database settings
		cronExpr := scheduler.getCronFromSettings()

		// Register cron job for DB updates
		err := app.Cron().Add(GrypeDBUpdateCronID, cronExpr, scheduler.UpdateDB)
		if err != nil {
			app.Logger().Error("Failed to register grype DB update cron",
				"error", err,
				"cron_expr", cronExpr)
		} else {
			app.Logger().Info("Grype DB update cron registered (from database settings)",
				"cron_expr", cronExpr)
		}

		// Also run an initial check on startup (after a delay to let app fully start)
		go func() {
			time.Sleep(30 * time.Second)
			scheduler.CheckAndUpdate()
		}()

		return e.Next()
	})
}

// UpdateDB performs the grype database update
func (s *GrypeDBScheduler) UpdateDB() {
	s.app.Logger().Info("Starting scheduled grype database update")

	startTime := time.Now()
	if err := s.scanner.UpdateDB(); err != nil {
		s.app.Logger().Error("Scheduled grype DB update failed",
			"error", err,
			"duration", time.Since(startTime))
		return
	}

	s.app.Logger().Info("Scheduled grype DB update completed",
		"duration", time.Since(startTime))

	// Log the new status
	status, err := s.scanner.GetStatus()
	if err == nil {
		s.app.Logger().Info("Grype DB status after update",
			"version", status.GrypeVersion,
			"db_version", status.DBVersion,
			"db_built_at", status.DBBuiltAt)
	}
}

// CheckAndUpdate checks if the DB needs updating and updates if stale
func (s *GrypeDBScheduler) CheckAndUpdate() {
	status, err := s.scanner.GetStatus()
	if err != nil {
		s.app.Logger().Warn("Failed to get grype status during startup check", "error", err)
		return
	}

	// If DB is not valid, update it
	if status.DBVersion == "" {
		s.app.Logger().Info("Grype DB not found, downloading...")
		s.UpdateDB()
		return
	}

	// Parse built date and check if it's older than 24 hours
	// The built date format is typically like "2025-01-29T01:31:21Z"
	builtAt, err := time.Parse(time.RFC3339, status.DBBuiltAt)
	if err != nil {
		// Try alternative formats
		builtAt, err = time.Parse("2006-01-02T15:04:05Z", status.DBBuiltAt)
		if err != nil {
			s.app.Logger().Warn("Could not parse DB built date", "date", status.DBBuiltAt)
			return
		}
	}

	// If DB is older than 24 hours, update it
	if time.Since(builtAt) > 24*time.Hour {
		s.app.Logger().Info("Grype DB is stale, updating...",
			"db_age", time.Since(builtAt),
			"built_at", status.DBBuiltAt)
		s.UpdateDB()
	} else {
		s.app.Logger().Info("Grype DB is up to date",
			"db_age", time.Since(builtAt),
			"built_at", status.DBBuiltAt)
	}
}
