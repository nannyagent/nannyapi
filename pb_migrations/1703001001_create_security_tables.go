package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create password_change_history collection
		historyCollection := core.NewBaseCollection("password_change_history")

		historyCollection.Fields.Add(&core.TextField{
			Name:     "user_id",
			Required: true,
		})

		historyCollection.Fields.Add(&core.TextField{
			Name:     "password_hash",
			Required: true,
		})

		historyCollection.Fields.Add(&core.TextField{
			Name:     "ip_address",
			Required: false,
		})

		historyCollection.Fields.Add(&core.TextField{
			Name:     "user_agent",
			Required: false,
		})

		historyCollection.Fields.Add(&core.BoolField{
			Name:     "changed_by_agent",
			Required: false,
		})

		// Add explicit timestamp field since base collections don't auto-create one
		historyCollection.Fields.Add(&core.DateField{
			Name:     "created",
			Required: false,
		})

		// API rules - only admins can access
		historyCollection.ListRule = nil
		historyCollection.ViewRule = nil
		historyCollection.CreateRule = nil
		historyCollection.UpdateRule = nil
		historyCollection.DeleteRule = nil

		if err := app.Save(historyCollection); err != nil {
			return err
		}

		// Create password_change_attempts collection
		attemptsCollection := core.NewBaseCollection("password_change_attempts")

		attemptsCollection.Fields.Add(&core.TextField{
			Name:     "user_id",
			Required: true,
		})

		attemptsCollection.Fields.Add(&core.TextField{
			Name:     "ip_address",
			Required: false,
		})

		attemptsCollection.Fields.Add(&core.BoolField{
			Name:     "success",
			Required: true,
		})

		// API rules - only admins can access
		attemptsCollection.ListRule = nil
		attemptsCollection.ViewRule = nil
		attemptsCollection.CreateRule = nil
		attemptsCollection.UpdateRule = nil
		attemptsCollection.DeleteRule = nil

		if err := app.Save(attemptsCollection); err != nil {
			return err
		}

		// Create failed_auth_attempts collection for tracking failed logins
		failedAuthCollection := core.NewBaseCollection("failed_auth_attempts")

		failedAuthCollection.Fields.Add(&core.TextField{
			Name:     "user_id",
			Required: true,
		})

		failedAuthCollection.Fields.Add(&core.TextField{
			Name:     "ip_address",
			Required: false,
		})

		// API rules - only admins can access
		failedAuthCollection.ListRule = nil
		failedAuthCollection.ViewRule = nil
		failedAuthCollection.CreateRule = nil
		failedAuthCollection.UpdateRule = nil
		failedAuthCollection.DeleteRule = nil

		if err := app.Save(failedAuthCollection); err != nil {
			return err
		}

		// Create account_lockout collection
		lockoutCollection := core.NewBaseCollection("account_lockout")

		lockoutCollection.Fields.Add(&core.TextField{
			Name:     "user_id",
			Required: true,
		})

		lockoutCollection.Fields.Add(&core.DateField{
			Name:     "locked_until",
			Required: true,
		})

		lockoutCollection.Fields.Add(&core.TextField{
			Name:     "reason",
			Required: false,
		})

		lockoutCollection.Fields.Add(&core.TextField{
			Name:     "ip_address",
			Required: false,
		})

		// API rules - only admins can access
		lockoutCollection.ListRule = nil
		lockoutCollection.ViewRule = nil
		lockoutCollection.CreateRule = nil
		lockoutCollection.UpdateRule = nil
		lockoutCollection.DeleteRule = nil

		if err := app.Save(lockoutCollection); err != nil {
			return err
		}

		// Create system_config collection for security settings
		configCollection := core.NewBaseCollection("system_config")

		configCollection.Fields.Add(&core.TextField{
			Name:     "key",
			Required: true,
		})

		configCollection.Fields.Add(&core.TextField{
			Name:     "value",
			Required: true,
		})

		configCollection.Fields.Add(&core.TextField{
			Name:     "description",
			Required: false,
		})

		// API rules - only admins can access
		configCollection.ListRule = nil
		configCollection.ViewRule = nil
		configCollection.CreateRule = nil
		configCollection.UpdateRule = nil
		configCollection.DeleteRule = nil

		if err := app.Save(configCollection); err != nil {
			return err
		}

		// Insert default config values
		defaults := []struct {
			key         string
			value       string
			description string
		}{
			{"security.password_change_limit_per_24h", "5", "Maximum password changes allowed per 24 hours"},
			{"security.password_history_window_hours", "24", "Window in hours to check password reuse"},
			{"security.account_lockout_duration_hours", "24", "Duration to lock account after too many failures"},
			{"security.failed_login_attempts_limit", "10", "Maximum failed login attempts before lockout"},
		}

		for _, def := range defaults {
			record := core.NewRecord(configCollection)
			record.Set("key", def.key)
			record.Set("value", def.value)
			record.Set("description", def.description)
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete all collections
		collections := []string{"password_change_history", "password_change_attempts", "failed_auth_attempts", "account_lockout", "system_config"}
		for _, name := range collections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}
			if err := app.Delete(collection); err != nil {
				return err
			}
		}
		return nil
	})
}
