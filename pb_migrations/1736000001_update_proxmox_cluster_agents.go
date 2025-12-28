package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("proxmox_cluster")
		if err != nil {
			return err
		}

		// Add 'agents' multi-relation field
		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}

		collection.Fields.Add(&core.RelationField{
			Name:          "agents",
			Required:      false,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: false,
			MaxSelect:     100, // Unlimited
		})

		// Migrate existing agent_id to agents
		// We need to save the collection first to create the field
		if err := app.Save(collection); err != nil {
			return err
		}

		// Now migrate data
		records, err := app.FindRecordsByFilter("proxmox_cluster", "agent_id != ''", "", 0, 0, nil)
		if err != nil {
			return err
		}

		for _, record := range records {
			agentID := record.GetString("agent_id")
			if agentID != "" {
				record.Set("agents", []string{agentID})
				if err := app.Save(record); err != nil {
					return err
				}
			}
		}

		// Remove old agent_id field
		collection.Fields.RemoveByName("agent_id")

		// Update API rules to use 'agents' instead of 'agent_id'
		// Old: user_id = @request.auth.id || agent_id = @request.auth.id
		// New: user_id = @request.auth.id || agents ?= @request.auth.id
		rule := "user_id = @request.auth.id || agents ?= @request.auth.id"
		collection.ListRule = &rule
		collection.ViewRule = &rule
		collection.CreateRule = &rule
		collection.UpdateRule = &rule

		return app.Save(collection)

	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("proxmox_cluster")
		if err != nil {
			return err
		}

		// Revert: Add agent_id back
		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}

		collection.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      false,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		if err := app.Save(collection); err != nil {
			return err
		}

		// Migrate data back (take first agent)
		records, err := app.FindRecordsByFilter("proxmox_cluster", "agents != ''", "", 0, 0, nil)
		if err != nil {
			return err
		}

		for _, record := range records {
			agents := record.GetStringSlice("agents")
			if len(agents) > 0 {
				record.Set("agent_id", agents[0])
				if err := app.Save(record); err != nil {
					return err
				}
			}
		}

		// Remove agents field
		collection.Fields.RemoveByName("agents")

		// Revert rules
		rule := "user_id = @request.auth.id || agent_id = @request.auth.id"
		collection.ListRule = &rule
		collection.ViewRule = &rule
		collection.CreateRule = &rule
		collection.UpdateRule = &rule

		return app.Save(collection)
	})
}
