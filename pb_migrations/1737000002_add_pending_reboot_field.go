package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add pending_reboot_id field to agents collection
		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}

		// Check if field already exists
		if agentsCollection.Fields.GetByName("pending_reboot_id") != nil {
			return nil
		}

		// Get reboot_operations collection for relation
		rebootOpsCollection, err := app.FindCollectionByNameOrId("reboot_operations")
		if err != nil {
			return err
		}

		// Add relation to pending reboot operation
		agentsCollection.Fields.Add(&core.RelationField{
			Name:          "pending_reboot_id",
			Required:      false,
			CollectionId:  rebootOpsCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		return app.Save(agentsCollection)
	}, func(app core.App) error {
		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return nil // Collection doesn't exist
		}

		field := agentsCollection.Fields.GetByName("pending_reboot_id")
		if field != nil {
			agentsCollection.Fields.RemoveByName("pending_reboot_id")
			return app.Save(agentsCollection)
		}
		return nil
	})
}
