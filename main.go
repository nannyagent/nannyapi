package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/nannyagent/nannyapi/internal/hooks"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
)

func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: os.Getenv("PB_AUTOMIGRATE") == "true",
	})

	// Register auth hooks
	app.OnRecordCreate("users").BindFunc(hooks.OnUserCreate(app))
	app.OnRecordUpdate("users").BindFunc(hooks.OnUserUpdate(app))
	app.OnRecordAuthWithPasswordRequest("users").BindFunc(hooks.OnAuthWithPasswordRequest(app))

	// Register agent hooks
	hooks.RegisterAgentHooks(app)

	// Register investigation management hooks
	hooks.RegisterInvestigationHooks(app)

	// Register patch management hooks
	hooks.RegisterPatchHooks(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
