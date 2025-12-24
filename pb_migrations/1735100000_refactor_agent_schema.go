package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Helper to check if field exists
		fieldExists := func(collection *core.Collection, fieldName string) bool {
			return collection.Fields.GetByName(fieldName) != nil
		}

		// --- Update agents collection ---
		agents, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}

		// Add new fields
		if !fieldExists(agents, "os_info") {
			agents.Fields.Add(&core.TextField{
				Name:     "os_info",
				Required: false,
			})
		}
		if !fieldExists(agents, "os_version") {
			agents.Fields.Add(&core.TextField{
				Name:     "os_version",
				Required: false,
			})
		}
		if !fieldExists(agents, "device_user_code") {
			agents.Fields.Add(&core.TextField{
				Name:     "device_user_code",
				Required: false,
			})
		}
		if !fieldExists(agents, "os_type") {
			agents.Fields.Add(&core.TextField{
				Name:     "os_type",
				Required: false,
			})
		}
		if !fieldExists(agents, "arch") {
			agents.Fields.Add(&core.TextField{
				Name:     "arch",
				Required: false,
			})
		}

		if err := app.Save(agents); err != nil {
			return err
		}

		// --- Update agent_metrics collection ---
		metrics, err := app.FindCollectionByNameOrId("agent_metrics")
		if err != nil {
			return err
		}

		// network_in_gb and network_out_gb should already exist
		if !fieldExists(metrics, "network_in_gb") {
			metrics.Fields.Add(&core.NumberField{
				Name: "network_in_gb",
				Min:  ptrFloat(0),
			})
		}
		if !fieldExists(metrics, "network_out_gb") {
			metrics.Fields.Add(&core.NumberField{
				Name: "network_out_gb",
				Min:  ptrFloat(0),
			})
		}

		if err := app.Save(metrics); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback logic (omitted for brevity in this context, but ideally should reverse changes)
		return nil
	})
}
