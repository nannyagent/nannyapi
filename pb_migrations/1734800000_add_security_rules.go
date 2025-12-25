package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	// Migration to add API rules preventing agents from modifying users/other agents
	m.Register(func(app core.App) error {
		// Get collections - skip if they don't exist yet
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil // Skip if users collection doesn't exist
		}

		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return nil // Skip if agents collection doesn't exist
		}

		// Users collection: Prevent API updates to sensitive fields
		// Only allow users to update their own non-sensitive data
		usersCollection.UpdateRule = ptrString("@request.auth.id = id")

		if err := app.Save(usersCollection); err != nil {
			return err
		}

		// Agents collection: Only allow users to manage their own agents
		// Agents cannot modify other agents or create new agents via API
		agentsCollection.ListRule = ptrString("user_id = @request.auth.id")
		agentsCollection.ViewRule = ptrString("user_id = @request.auth.id")
		agentsCollection.UpdateRule = ptrString("user_id = @request.auth.id")
		agentsCollection.DeleteRule = ptrString("user_id = @request.auth.id")
		agentsCollection.CreateRule = nil // Only via device auth flow

		if err := app.Save(agentsCollection); err != nil {
			return err
		}

		// Agent Metrics collection: Allow reading metrics for own agents or any authenticated user
		metricsCollection, err := app.FindCollectionByNameOrId("agent_metrics")
		if err == nil {
			// Allow authenticated users to view metrics
			metricsCollection.ListRule = ptrString("@request.auth.id != ''")
			metricsCollection.ViewRule = ptrString("@request.auth.id != ''")
			metricsCollection.CreateRule = ptrString("") // Handled via agent auth in backend
			metricsCollection.UpdateRule = nil           // No updates allowed
			metricsCollection.DeleteRule = nil           // No deletes allowed

			if err := app.Save(metricsCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove API rules
		usersCollection, _ := app.FindCollectionByNameOrId("users")
		if usersCollection != nil {
			usersCollection.UpdateRule = nil
			err := app.Save(usersCollection)
			if err != nil {
				return err
			}
		}

		agentsCollection, _ := app.FindCollectionByNameOrId("agents")
		if agentsCollection != nil {
			agentsCollection.ListRule = nil
			agentsCollection.ViewRule = nil
			agentsCollection.UpdateRule = nil
			agentsCollection.DeleteRule = nil
			agentsCollection.CreateRule = nil
			err := app.Save(agentsCollection)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func ptrString(s string) *string {
	return &s
}
