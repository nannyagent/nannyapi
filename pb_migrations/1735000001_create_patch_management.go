package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Check if already exists
		collection, _ := app.FindCollectionByNameOrId("patch_operations")
		if collection != nil {
			return nil
		}

		// Get required collections
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}

		// Create patch_operations collection
		patchOps := core.NewBaseCollection("patch_operations")

		// User who initiated patch
		patchOps.Fields.Add(&core.RelationField{
			Name:          "user_id",
			Required:      true,
			CollectionId:  usersCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Agent to patch
		patchOps.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Patch mode: dry-run or apply
		patchOps.Fields.Add(&core.TextField{
			Name:     "mode",
			Required: true,
			Max:      50,
		})

		// Operation status: pending, running, completed, failed, rolled_back
		patchOps.Fields.Add(&core.TextField{
			Name:     "status",
			Required: true,
			Max:      50,
		})

		// Reference to script in storage (URL or path)
		patchOps.Fields.Add(&core.TextField{
			Name:     "script_url",
			Required: true,
			Max:      1000,
		})

		// Path in storage where output was saved (null until completed)
		patchOps.Fields.Add(&core.TextField{
			Name:     "output_path",
			Required: false,
			Max:      1000,
		})

		// Error message if operation failed
		patchOps.Fields.Add(&core.TextField{
			Name:     "error_msg",
			Required: false,
			Max:      2000,
		})

		// When patch execution started
		patchOps.Fields.Add(&core.DateField{
			Name:     "started_at",
			Required: false,
		})

		// When patch execution completed
		patchOps.Fields.Add(&core.DateField{
			Name:     "completed_at",
			Required: false,
		})

		// Set API rules
		patchOps.ListRule = ptrString("user_id = @request.auth.id")
		patchOps.ViewRule = ptrString("user_id = @request.auth.id")
		patchOps.CreateRule = ptrString("user_id = @request.auth.id")
		patchOps.UpdateRule = ptrString("user_id = @request.auth.id")
		patchOps.DeleteRule = ptrString("user_id = @request.auth.id")

		if err := app.Save(patchOps); err != nil {
			return err
		}

		// Create package_updates collection to track packages affected by patches
		packageUpdates := core.NewBaseCollection("package_updates")

		// Reference to patch operation
		packageUpdates.Fields.Add(&core.RelationField{
			Name:          "patch_op_id",
			Required:      true,
			CollectionId:  patchOps.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Package name
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "package_name",
			Required: true,
			Max:      255,
		})

		// Current installed version
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "current_ver",
			Required: false,
			Max:      100,
		})

		// Target version (empty for removal)
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "target_ver",
			Required: false,
			Max:      100,
		})

		// Update type: install, update, remove
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "update_type",
			Required: true,
			Max:      50,
		})

		// Status: pending, applied, failed
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "status",
			Required: true,
			Max:      50,
		})

		// Dry-run simulation results
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "dry_run_results",
			Required: false,
			Max:      2000,
		})

		// Set API rules - only through patch operation owner
		packageUpdates.ListRule = ptrString("patch_op_id.user_id = @request.auth.id")
		packageUpdates.ViewRule = ptrString("patch_op_id.user_id = @request.auth.id")
		packageUpdates.CreateRule = ptrString("") // Backend only
		packageUpdates.UpdateRule = ptrString("") // Backend only
		packageUpdates.DeleteRule = ptrString("") // Backend only

		if err := app.Save(packageUpdates); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete collections in reverse order
		collections := []string{"package_updates", "patch_operations"}
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
