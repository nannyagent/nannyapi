package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Check if already exists
		collection, _ := app.FindCollectionByNameOrId("reboot_schedules")
		if collection != nil {
			return nil
		}

		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}
		lxcCollection, err := app.FindCollectionByNameOrId("proxmox_lxc")
		if err != nil {
			return err
		}

		schedules := core.NewBaseCollection("reboot_schedules")

		schedules.Fields.Add(&core.RelationField{
			Name:          "user_id",
			Required:      true,
			CollectionId:  usersCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		schedules.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		schedules.Fields.Add(&core.RelationField{
			Name:          "lxc_id",
			Required:      false,
			CollectionId:  lxcCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		schedules.Fields.Add(&core.TextField{
			Name:     "cron_expression",
			Required: true,
		})

		// Reason for scheduled reboot
		schedules.Fields.Add(&core.TextField{
			Name:     "reason",
			Required: false,
		})

		schedules.Fields.Add(&core.DateField{
			Name:     "next_run_at",
			Required: false,
		})

		schedules.Fields.Add(&core.DateField{
			Name:     "last_run_at",
			Required: false,
		})

		schedules.Fields.Add(&core.BoolField{
			Name:     "is_active",
			Required: false,
		})

		// API Rules - Only users can schedule reboots
		userOnly := "@request.auth.id = user_id"
		schedules.ListRule = &userOnly
		schedules.ViewRule = &userOnly
		schedules.CreateRule = &userOnly
		schedules.UpdateRule = &userOnly
		schedules.DeleteRule = &userOnly

		return app.Save(schedules)
	}, func(app core.App) error {
		collection, _ := app.FindCollectionByNameOrId("reboot_schedules")
		if collection != nil {
			return app.Delete(collection)
		}
		return nil
	})
}
