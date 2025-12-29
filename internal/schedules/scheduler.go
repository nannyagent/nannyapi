package schedules

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/robfig/cron/v3"
)

// RegisterScheduler registers the patch scheduler
func RegisterScheduler(app core.App) {
	// Load and register all active schedules on app start
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		records, err := app.FindRecordsByFilter("patch_schedules", "is_active = true", "", 0, 0, nil)
		if err != nil {
			app.Logger().Error("Failed to load patch schedules", "error", err)
			return e.Next()
		}

		for _, record := range records {
			registerCronJob(app, record)
		}

		return e.Next()
	})

	// Hook to manage cron jobs on create/update/delete
	manageCron := func(e *core.RecordEvent) error {
		// Calculate next_run_at for display purposes
		if err := updateNextRun(e.Record); err != nil {
			// If cron expression is invalid, don't register but don't fail the save?
			// Or fail the save? Better to fail the save if invalid.
			return err
		}

		if e.Record.GetBool("is_active") {
			registerCronJob(app, e.Record)
		} else {
			app.Cron().Remove(e.Record.Id)
		}

		return e.Next()
	}

	app.OnRecordCreate("patch_schedules").BindFunc(manageCron)
	app.OnRecordUpdate("patch_schedules").BindFunc(manageCron)

	app.OnRecordDelete("patch_schedules").BindFunc(func(e *core.RecordEvent) error {
		app.Cron().Remove(e.Record.Id)
		return e.Next()
	})
}

func registerCronJob(app core.App, record *core.Record) {
	cronExpr := record.GetString("cron_expression")
	scheduleID := record.Id

	// PocketBase app.Cron().Add(id, spec, cmd) replaces if exists.
	err := app.Cron().Add(scheduleID, cronExpr, func() {
		ExecuteSchedule(app, scheduleID)
	})

	if err != nil {
		app.Logger().Error("Failed to register cron job", "schedule_id", scheduleID, "error", err)
	}
}

func updateNextRun(record *core.Record) error {
	cronExpr := record.GetString("cron_expression")
	if cronExpr == "" {
		return nil
	}

	// Use standard cron parser
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return err
	}

	nextRun := schedule.Next(time.Now().UTC())
	record.Set("next_run_at", nextRun)
	return nil
}

func ExecuteSchedule(app core.App, scheduleID string) {
	// Fetch fresh record to ensure it's still valid
	schedule, err := app.FindRecordById("patch_schedules", scheduleID)
	if err != nil {
		app.Logger().Error("Failed to fetch schedule for execution", "schedule_id", scheduleID, "error", err)
		// If record is gone, remove cron
		app.Cron().Remove(scheduleID)
		return
	}

	if !schedule.GetBool("is_active") {
		app.Cron().Remove(scheduleID)
		return
	}

	// Create patch operation
	patchOps, err := app.FindCollectionByNameOrId("patch_operations")
	if err != nil {
		app.Logger().Error("Failed to find patch_operations collection", "error", err)
		return
	}

	op := core.NewRecord(patchOps)
	op.Set("user_id", schedule.GetString("user_id"))
	op.Set("agent_id", schedule.GetString("agent_id"))
	op.Set("lxc_id", schedule.GetString("lxc_id"))
	op.Set("mode", "apply")
	op.Set("status", "pending")
	op.Set("started_at", time.Now().UTC())

	// Save operation
	if err := app.Save(op); err != nil {
		app.Logger().Error("Failed to create patch operation", "schedule_id", scheduleID, "error", err)
		return
	}

	// Update last_run_at and next_run_at
	schedule.Set("last_run_at", time.Now().UTC())
	updateNextRun(schedule) // Update next_run_at for display

	if err := app.Save(schedule); err != nil {
		app.Logger().Error("Failed to update schedule stats", "schedule_id", scheduleID, "error", err)
	}
}
