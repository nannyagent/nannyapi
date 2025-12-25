package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Create package_exceptions collection
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}

		pkgExceptions := core.NewBaseCollection("package_exceptions")

		pkgExceptions.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		pkgExceptions.Fields.Add(&core.TextField{
			Name:     "package_name",
			Required: true,
		})

		pkgExceptions.Fields.Add(&core.TextField{
			Name:     "reason",
			Required: false,
		})

		pkgExceptions.Fields.Add(&core.RelationField{
			Name:          "user_id",
			Required:      true,
			CollectionId:  usersCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		pkgExceptions.Fields.Add(&core.DateField{
			Name:     "expires_at",
			Required: false,
		})

		pkgExceptions.Fields.Add(&core.BoolField{
			Name:     "is_active",
			Required: false,
		})

		// API Rules
		pkgExceptions.ListRule = ptrString("user_id = @request.auth.id")
		pkgExceptions.ViewRule = ptrString("user_id = @request.auth.id")
		pkgExceptions.CreateRule = ptrString("user_id = @request.auth.id")
		pkgExceptions.UpdateRule = ptrString("user_id = @request.auth.id")
		pkgExceptions.DeleteRule = ptrString("user_id = @request.auth.id")

		if err := app.Save(pkgExceptions); err != nil {
			return err
		}

		// 2. Add exclusions field to patch_operations
		patchOps, err := app.FindCollectionByNameOrId("patch_operations")
		if err != nil {
			return err
		}

		patchOps.Fields.Add(&core.JSONField{
			Name:     "exclusions",
			Required: false,
		})

		// 3. Make script_id optional so hooks can populate it
		scriptIdField := patchOps.Fields.GetByName("script_id")
		if val, ok := scriptIdField.(*core.RelationField); ok {
			val.Required = false
		}

		return app.Save(patchOps)
	}, func(app core.App) error {
		// Revert operations
		// 1. Remove exclusions field
		patchOps, err := app.FindCollectionByNameOrId("patch_operations")
		if err != nil {
			return err
		}
		patchOps.Fields.RemoveByName("exclusions")

		// Revert script_id to required
		scriptIdField := patchOps.Fields.GetByName("script_id")
		if val, ok := scriptIdField.(*core.RelationField); ok {
			val.Required = true
		}

		if err := app.Save(patchOps); err != nil {
			return err
		}

		// 2. Delete package_exceptions collection
		pkgExceptions, err := app.FindCollectionByNameOrId("package_exceptions")
		if err != nil {
			return err
		}
		return app.Delete(pkgExceptions)
	})
}
