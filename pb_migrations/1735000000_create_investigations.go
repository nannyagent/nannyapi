package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Check if already exists
		collection, _ := app.FindCollectionByNameOrId("investigations")
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

		// Create investigations collection
		investigations := core.NewBaseCollection("investigations")

		// User who initiated the investigation
		investigations.Fields.Add(&core.RelationField{
			Name:          "user_id",
			Required:      true,
			CollectionId:  usersCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Agent being investigated
		investigations.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// TensorZero episode reference - used to fetch inferences from ClickHouse
		investigations.Fields.Add(&core.TextField{
			Name:     "episode_id",
			Required: false,
			Max:      255,
		})

		// User's initial issue description
		investigations.Fields.Add(&core.TextField{
			Name:     "user_prompt",
			Required: true,
			Max:      2000,
		})

		// Investigation priority
		investigations.Fields.Add(&core.TextField{
			Name:     "priority",
			Required: false,
			Max:      50,
		})

		// Investigation status: pending, in_progress, completed, failed
		investigations.Fields.Add(&core.TextField{
			Name:     "status",
			Required: true,
			Max:      50,
		})

		// AI-generated resolution plan from TensorZero
		investigations.Fields.Add(&core.TextField{
			Name:     "resolution_plan",
			Required: false,
			Max:      5000,
		})

		// When investigation was initiated
		investigations.Fields.Add(&core.DateField{
			Name:     "initiated_at",
			Required: true,
		})

		// When investigation was completed (null if ongoing)
		investigations.Fields.Add(&core.DateField{
			Name:     "completed_at",
			Required: false,
		})

		// Additional metadata as JSON
		investigations.Fields.Add(&core.JSONField{
			Name:     "metadata",
			Required: false,
		})

		// Set API rules - only investigation owner and their agents
		investigations.ListRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		investigations.ViewRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		investigations.CreateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		investigations.UpdateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		investigations.DeleteRule = ptrString("user_id = @request.auth.id")

		if err := app.Save(investigations); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete investigations collection
		collection, _ := app.FindCollectionByNameOrId("investigations")
		if collection != nil {
			if err := app.Delete(collection); err != nil {
				return err
			}
		}
		return nil
	})
}
