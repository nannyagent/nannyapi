package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Check if already exists
		collection, _ := app.FindCollectionByNameOrId("reboot_operations")
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

		rebootOps := core.NewBaseCollection("reboot_operations")

		// User who initiated the reboot (required)
		rebootOps.Fields.Add(&core.RelationField{
			Name:          "user_id",
			Required:      true,
			CollectionId:  usersCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		// Target agent (required)
		rebootOps.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Optional: LXC container to reboot (if reboot is for LXC, not host)
		rebootOps.Fields.Add(&core.RelationField{
			Name:          "lxc_id",
			Required:      false,
			CollectionId:  lxcCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		// VMID of the LXC container (copied from proxmox_lxc for convenience)
		rebootOps.Fields.Add(&core.TextField{
			Name:     "vmid",
			Required: false,
		})

		// Status: pending, sent, rebooting, completed, failed, timeout
		rebootOps.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"pending", "sent", "rebooting", "completed", "failed", "timeout"},
		})

		// Reason for reboot (optional - user can provide context)
		rebootOps.Fields.Add(&core.TextField{
			Name:     "reason",
			Required: false,
		})

		// When the reboot was requested
		rebootOps.Fields.Add(&core.DateField{
			Name:     "requested_at",
			Required: true,
		})

		// When the agent acknowledged the reboot command
		rebootOps.Fields.Add(&core.DateField{
			Name:     "acknowledged_at",
			Required: false,
		})

		// When the agent came back online after reboot
		rebootOps.Fields.Add(&core.DateField{
			Name:     "completed_at",
			Required: false,
		})

		// Error message if failed
		rebootOps.Fields.Add(&core.TextField{
			Name:     "error_message",
			Required: false,
		})

		// Timeout duration in seconds (default 300 = 5 minutes)
		rebootOps.Fields.Add(&core.NumberField{
			Name:     "timeout_seconds",
			Required: false,
			Min:      nil,
			Max:      nil,
		})

		// API Rules - Users and agents can view reboot operations
		// Users: can create, update, delete their own
		// Agents: can view (for realtime) and update (acknowledge)
		userOrAgent := "user_id = @request.auth.id || agent_id = @request.auth.id"
		userOnly := "@request.auth.id = user_id"
		rebootOps.ListRule = &userOrAgent
		rebootOps.ViewRule = &userOrAgent
		rebootOps.CreateRule = &userOnly
		rebootOps.UpdateRule = &userOrAgent // Agent needs to update for acknowledge
		rebootOps.DeleteRule = &userOnly

		return app.Save(rebootOps)
	}, func(app core.App) error {
		collection, _ := app.FindCollectionByNameOrId("reboot_operations")
		if collection != nil {
			return app.Delete(collection)
		}
		return nil
	})
}
