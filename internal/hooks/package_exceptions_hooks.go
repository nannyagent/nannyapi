package hooks

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"
)

// RegisterPackageExceptionHooks registers hooks for package_exceptions collection
func RegisterPackageExceptionHooks(app core.App) {
	validateException := func(e *core.RecordEvent) error {
		agentID := e.Record.GetString("agent_id")
		lxcID := e.Record.GetString("lxc_id")

		if agentID == "" && lxcID == "" {
			return errors.New("either agent_id or lxc_id must be provided")
		}
		return e.Next()
	}

	app.OnRecordCreate("package_exceptions").BindFunc(validateException)
	app.OnRecordUpdate("package_exceptions").BindFunc(validateException)
}
