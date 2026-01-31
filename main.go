package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"

	"github.com/nannyagent/nannyapi/internal/hooks"
	"github.com/nannyagent/nannyapi/internal/mfa"
	"github.com/nannyagent/nannyapi/internal/sbom"
	"github.com/nannyagent/nannyapi/internal/schedules"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
)

// Version is set at build time via -ldflags
var Version = "dev"

// CLI flags
var (
	enableVulnScan  bool
	grypePath       string
	grypeDBCacheDir string
)

func main() {
	// Check for --version flag before initializing PocketBase
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Printf("nannyapi version %s\n", Version)
			return
		}
	}

	app := pocketbase.New()

	// Add vulnerability scanning flags to serve command
	app.RootCmd.PersistentFlags().BoolVar(&enableVulnScan, "enable-vuln-scan", false, "Enable SBOM vulnerability scanning (requires grype)")
	app.RootCmd.PersistentFlags().StringVar(&grypePath, "grype-path", "", "Path to grype binary (default: auto-detect from PATH)")
	app.RootCmd.PersistentFlags().StringVar(&grypeDBCacheDir, "grype-db-cache-dir", "/var/cache/grype/db", "Directory for grype database cache")

	// Add version command
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("nannyapi version %s\n", Version)
		},
	})

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: os.Getenv("PB_AUTOMIGRATE") == "true",
	})

	// Register auth hooks
	app.OnRecordCreate("users").BindFunc(hooks.OnUserCreate(app))
	app.OnRecordUpdate("users").BindFunc(hooks.OnUserUpdate(app))
	app.OnRecordAuthWithPasswordRequest("users").BindFunc(hooks.OnAuthWithPasswordRequest(app))

	// Register MFA hooks
	mfaHooks := hooks.NewMFAHooks(app)
	app.OnRecordAuthRequest("users").BindFunc(mfaHooks.OnAuthSuccess)

	// Register MFA routes
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		mfaHandler := mfa.NewHandler(app)
		mfaHandler.RegisterRoutes(e)
		return e.Next()
	})

	// Register agent hooks
	hooks.RegisterAgentHooks(app)

	// Register investigation and patch management hooks
	hooks.RegisterInvestigationHooks(app)
	hooks.RegisterPatchHooks(app)
	hooks.RegisterPackageExceptionHooks(app)

	// Register reboot hooks
	hooks.RegisterRebootHooks(app)

	// Register proxmox hooks
	hooks.RegisterProxmoxHooks(app)

	// Register schedulers for patch and reboot schedules
	schedules.RegisterScheduler(app)
	schedules.RegisterRebootScheduler(app)

	// Initialize SBOM vulnerability scanning if enabled
	var sbomScanner *sbom.Scanner
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if enableVulnScan {
			var err error
			sbomScanner, err = sbom.NewScanner(app, sbom.Config{
				Enabled:    true,
				GrypePath:  grypePath,
				DBCacheDir: grypeDBCacheDir,
			})
			if err != nil {
				app.Logger().Error("Failed to initialize SBOM scanner",
					"error", err,
					"grype_path", grypePath)
				// Don't fail startup, just disable the feature
				sbomScanner, _ = sbom.NewScanner(app, sbom.Config{Enabled: false})
			} else {
				app.Logger().Info("SBOM vulnerability scanning enabled",
					"grype_path", sbomScanner.GrypePath(),
					"db_cache_dir", grypeDBCacheDir)
			}

			// Register SBOM routes
			sbom.RegisterRoutes(app, sbomScanner, e)

			// Register grype DB update scheduler (settings read from sbom_settings collection)
			schedules.RegisterGrypeDBScheduler(app, sbomScanner)
		} else {
			app.Logger().Info("SBOM vulnerability scanning is disabled. Use --enable-vuln-scan to enable.")
			// Register routes with disabled scanner for status endpoint
			sbomScanner, _ = sbom.NewScanner(app, sbom.Config{Enabled: false})
			sbom.RegisterRoutes(app, sbomScanner, e)
		}
		return e.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
