package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		metricsCollection, err := app.FindCollectionByNameOrId("agent_metrics")
		if err != nil || metricsCollection == nil {
			return nil // Collection doesn't exist yet
		}

		// Helper to check if field exists
		fieldExists := func(fieldName string) bool {
			for _, field := range metricsCollection.Fields {
				if field.GetName() == fieldName {
					return true
				}
			}
			return false
		}

		// Add CPU cores field
		if !fieldExists("cpu_cores") {
			metricsCollection.Fields.Add(&core.NumberField{
				Name: "cpu_cores",
				Min:  ptrFloat(0),
			})
		}

		// Add Memory percent field
		if !fieldExists("memory_percent") {
			metricsCollection.Fields.Add(&core.NumberField{
				Name: "memory_percent",
				Min:  ptrFloat(0),
				Max:  ptrFloat(100),
			})
		}

		// Add Disk usage percent field
		if !fieldExists("disk_usage_percent") {
			metricsCollection.Fields.Add(&core.NumberField{
				Name: "disk_usage_percent",
				Min:  ptrFloat(0),
				Max:  ptrFloat(100),
			})
		}

		// Add Load Average fields (1min, 5min, 15min)
		if !fieldExists("load_avg_1min") {
			metricsCollection.Fields.Add(&core.NumberField{
				Name: "load_avg_1min",
				Min:  ptrFloat(0),
			})
		}

		if !fieldExists("load_avg_5min") {
			metricsCollection.Fields.Add(&core.NumberField{
				Name: "load_avg_5min",
				Min:  ptrFloat(0),
			})
		}

		if !fieldExists("load_avg_15min") {
			metricsCollection.Fields.Add(&core.NumberField{
				Name: "load_avg_15min",
				Min:  ptrFloat(0),
			})
		}

		// Add Filesystems JSON field (store as text)
		if !fieldExists("filesystems") {
			metricsCollection.Fields.Add(&core.TextField{
				Name: "filesystems",
			})
		}

		// Set API rules - only investigation owner and their agents
		metricsCollection.ListRule = ptrString("agent_id.user_id = @request.auth.id || agent_id = @request.auth.id ")
		metricsCollection.ViewRule = ptrString("agent_id.user_id = @request.auth.id || agent_id = @request.auth.id ")
		metricsCollection.CreateRule = ptrString("agent_id.user_id = @request.auth.id || agent_id = @request.auth.id ")
		metricsCollection.UpdateRule = ptrString("agent_id.user_id = @request.auth.id || agent_id = @request.auth.id ")
		metricsCollection.DeleteRule = ptrString("agent_id.user_id = @request.auth.id")

		return app.Save(metricsCollection)
	}, func(app core.App) error {
		// Rollback: no need to actively remove - the migration is idempotent
		// Fields will just not be added if they already exist
		return nil
	})
}
