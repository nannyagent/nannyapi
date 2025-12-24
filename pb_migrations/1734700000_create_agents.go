package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func ptrFloat(f float64) *float64 {
	return &f
}

func init() {
	m.Register(func(app core.App) error {
		collection, _ := app.FindCollectionByNameOrId("agents")
		if collection != nil {
			return nil // Already exists
		}

		// Get users collection ID
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Create device_codes collection for anonymous agent registration
		deviceCodes := core.NewBaseCollection("device_codes")

		deviceCodes.Fields.Add(&core.TextField{
			Name:     "device_code",
			Required: true,
			Max:      255,
		})

		deviceCodes.Fields.Add(&core.TextField{
			Name:     "user_code",
			Required: true,
			Max:      8,
		})

		deviceCodes.Fields.Add(&core.RelationField{
			Name:          "user_id",
			Required:      false, // null until authorized
			CollectionId:  usersCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		deviceCodes.Fields.Add(&core.BoolField{
			Name:     "authorized",
			Required: false,
		})

		deviceCodes.Fields.Add(&core.BoolField{
			Name:     "consumed",
			Required: false,
		})

		deviceCodes.Fields.Add(&core.DateField{
			Name:     "expires_at",
			Required: true,
		})

		deviceCodes.Fields.Add(&core.TextField{
			Name:     "agent_id",
			Required: false,
			Max:      255,
		})

		if err := app.Save(deviceCodes); err != nil {
			return err
		}

		// Create agents collection of type auth
		agents := core.NewAuthCollection("agents")
		agents.Fields.Add(&core.EmailField{
			Name:     "email",
			Required: false,
			System:   true,
			Hidden:   true,
		})

		agents.Fields.Add(&core.PasswordField{
			Name:     "password",
			Required: false,
			System:   true,
			Hidden:   true,
		})

		agents.Fields.Add(&core.RelationField{
			Name:          "user_id",
			Required:      true,
			CollectionId:  usersCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		agents.Fields.Add(&core.RelationField{
			Name:          "device_code_id",
			Required:      true,
			CollectionId:  deviceCodes.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		agents.Fields.Add(&core.TextField{
			Name:     "hostname",
			Required: true,
			Max:      255,
		})

		agents.Fields.Add(&core.TextField{
			Name:     "platform_family",
			Required: true,
			Max:      100,
		})

		agents.Fields.Add(&core.TextField{
			Name:     "version",
			Required: true,
			Max:      50,
		})

		agents.Fields.Add(&core.TextField{
			Name:     "primary_ip",
			Required: false,
			Max:      45, // IPv6 max length
		})

		agents.Fields.Add(&core.JSONField{
			Name:     "all_ips",
			Required: false,
		})

		agents.Fields.Add(&core.DateField{
			Name:     "last_seen",
			Required: false,
		})

		agents.Fields.Add(&core.TextField{
			Name:     "kernel_version",
			Required: false,
			Max:      45,
		})

		agents.Fields.Add(&core.TextField{
			Name:     "refresh_token_hash",
			Required: false,
			Max:      255,
		})

		agents.Fields.Add(&core.DateField{
			Name:     "refresh_token_expires",
			Required: false,
		})

		if err := app.Save(agents); err != nil {
			return err
		}

		// Create agent_metrics collection - with individual fields for dashboard visibility
		agentMetrics := core.NewBaseCollection("agent_metrics")

		agentMetrics.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agents.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Individual metric fields for dashboard visibility
		agentMetrics.Fields.Add(&core.NumberField{
			Name: "cpu_percent",
			Min:  ptrFloat(0),
			Max:  ptrFloat(100),
		})

		agentMetrics.Fields.Add(&core.NumberField{
			Name: "memory_used_gb",
			Min:  ptrFloat(0),
		})

		agentMetrics.Fields.Add(&core.NumberField{
			Name: "memory_total_gb",
			Min:  ptrFloat(0),
		})

		agentMetrics.Fields.Add(&core.NumberField{
			Name: "disk_used_gb",
			Min:  ptrFloat(0),
		})

		agentMetrics.Fields.Add(&core.NumberField{
			Name: "disk_total_gb",
			Min:  ptrFloat(0),
		})

		agentMetrics.Fields.Add(&core.NumberField{
			Name: "network_in_gb",
			Min:  ptrFloat(0),
		})

		agentMetrics.Fields.Add(&core.NumberField{
			Name: "network_out_gb",
			Min:  ptrFloat(0),
		})

		agentMetrics.Fields.Add(&core.DateField{
			Name:     "recorded_at",
			Required: true,
		})

		if err := app.Save(agentMetrics); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete collections in reverse order
		collections := []string{"agent_metrics", "agents", "device_codes"}
		for _, name := range collections {
			collection, _ := app.FindCollectionByNameOrId(name)
			if collection != nil {
				if err := app.Delete(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
