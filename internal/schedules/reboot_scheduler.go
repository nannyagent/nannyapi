package schedules

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/robfig/cron/v3"
)

// RegisterRebootScheduler registers the reboot scheduler
func RegisterRebootScheduler(app core.App) {
	// Load and register all active reboot schedules on app start
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		records, err := app.FindRecordsByFilter("reboot_schedules", "is_active = true", "", 0, 0, nil)
		if err != nil {
			app.Logger().Error("Failed to load reboot schedules", "error", err)
			return e.Next()
		}

		for _, record := range records {
			registerRebootCronJob(app, record)
		}

		return e.Next()
	})

	// Hook to manage cron jobs on create/update/delete
	manageRebootCron := func(e *core.RecordEvent) error {
		// Calculate next_run_at for display purposes
		if err := updateRebootNextRun(e.Record); err != nil {
			// If cron expression is invalid, fail the save
			return err
		}

		if e.Record.GetBool("is_active") {
			registerRebootCronJob(app, e.Record)
		} else {
			app.Cron().Remove(e.Record.Id)
		}

		return e.Next()
	}

	app.OnRecordCreate("reboot_schedules").BindFunc(manageRebootCron)
	app.OnRecordUpdate("reboot_schedules").BindFunc(manageRebootCron)

	app.OnRecordDelete("reboot_schedules").BindFunc(func(e *core.RecordEvent) error {
		app.Cron().Remove(e.Record.Id)
		return e.Next()
	})
}

func registerRebootCronJob(app core.App, record *core.Record) {
	cronExpr := record.GetString("cron_expression")
	scheduleID := record.Id

	// PocketBase app.Cron().Add(id, spec, cmd) replaces if exists.
	// No prefix needed - schedule IDs are unique across collections
	err := app.Cron().Add(scheduleID, cronExpr, func() {
		ExecuteRebootSchedule(app, scheduleID)
	})

	if err != nil {
		app.Logger().Error("Failed to register reboot cron job", "schedule_id", scheduleID, "error", err)
	}
}

func updateRebootNextRun(record *core.Record) error {
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

// ExecuteRebootSchedule creates a reboot operation from a schedule
// Note: This creates the record directly, bypassing the pending_reboot check that
// applies to user-initiated reboots. Scheduled reboots should proceed even if
// a previous scheduled reboot timed out (the hook will clear pending_reboot_id).
func ExecuteRebootSchedule(app core.App, scheduleID string) {
	// Fetch fresh record to ensure it's still valid
	schedule, err := app.FindRecordById("reboot_schedules", scheduleID)
	if err != nil {
		app.Logger().Error("Failed to fetch reboot schedule for execution", "schedule_id", scheduleID, "error", err)
		// If record is gone, remove cron
		app.Cron().Remove(scheduleID)
		return
	}

	if !schedule.GetBool("is_active") {
		app.Cron().Remove(scheduleID)
		return
	}

	// Find reboot_operations collection
	collection, err := app.FindCollectionByNameOrId("reboot_operations")
	if err != nil {
		app.Logger().Error("Failed to find reboot_operations collection", "error", err)
		return
	}

	agentID := schedule.GetString("agent_id")
	lxcID := schedule.GetString("lxc_id")
	userID := schedule.GetString("user_id")

	// Get vmid from lxc if present (like patches do)
	var vmid string
	if lxcID != "" {
		lxcRecord, err := app.FindRecordById("proxmox_lxc", lxcID)
		if err == nil {
			vmid = lxcRecord.GetString("vmid")
		}
	}

	// Create reboot operation directly (like patches scheduler)
	record := core.NewRecord(collection)
	record.Set("user_id", userID)
	record.Set("agent_id", agentID)
	record.Set("lxc_id", lxcID)
	record.Set("vmid", vmid)
	record.Set("status", "pending") // Will be set to "sent" by OnRecordCreate hook
	record.Set("reason", schedule.GetString("reason"))
	record.Set("requested_at", time.Now().UTC())
	record.Set("timeout_seconds", 300)

	if err := app.Save(record); err != nil {
		app.Logger().Error("Failed to create reboot operation from schedule", "schedule_id", scheduleID, "error", err)
		return
	}

	// Update last_run_at and next_run_at
	schedule.Set("last_run_at", time.Now().UTC())
	if err := updateRebootNextRun(schedule); err != nil {
		app.Logger().Error("Failed to update reboot next_run_at", "schedule_id", scheduleID, "error", err)
	}

	if err := app.Save(schedule); err != nil {
		app.Logger().Error("Failed to update reboot schedule stats", "schedule_id", scheduleID, "error", err)
	}
}
