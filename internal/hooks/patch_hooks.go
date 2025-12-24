package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/nannyagent/nannyapi/internal/patches"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// RegisterPatchHooks registers all patch management endpoints and hooks
func RegisterPatchHooks(app core.App) {
	// Hook to calculate SHA256 checksum for scripts
	calculateScriptChecksum := func(e *core.RecordEvent) error {
		// Check if "file" field has a new file
		val := e.Record.Get("file")
		f, ok := val.(*filesystem.File)
		if !ok {
			// Not a new file upload, or it's just a filename string
			return e.Next()
		}

		// Open the uploaded file
		reader, err := f.Reader.Open()
		if err != nil {
			return err
		}
		defer reader.Close()

		// Calculate SHA256
		hasher := sha256.New()
		if _, err := io.Copy(hasher, reader); err != nil {
			return err
		}

		checksum := hex.EncodeToString(hasher.Sum(nil))
		e.Record.Set("sha256", checksum)

		return e.Next()
	}

	app.OnRecordCreate("scripts").BindFunc(calculateScriptChecksum)
	app.OnRecordUpdate("scripts").BindFunc(calculateScriptChecksum)

	// Hook to populate script_url on patch operation creation
	app.OnRecordCreate("patch_operations").BindFunc(func(e *core.RecordEvent) error {
		scriptID := e.Record.GetString("script_id")
		if scriptID == "" {
			return e.Next()
		}

		// Always populating ensures consistency.
		scriptsCollection, err := app.FindCollectionByNameOrId("scripts")
		if err != nil {
			return err
		}

		scriptRecord, err := app.FindRecordById(scriptsCollection, scriptID)
		if err != nil {
			return err
		}

		// Construct URL: /api/files/<collection>/<id>/<filename>
		scriptURL := fmt.Sprintf("/api/files/%s/%s/%s", scriptsCollection.Id, scriptRecord.Id, scriptRecord.GetString("file"))
		e.Record.Set("script_url", scriptURL)

		return e.Next()
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Helper to wrap handler with auth middleware
		withAuth := func(handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
			return LoadAuthContext(app)(RequireAuth()(handler))
		}

		// Single endpoint for all patch operations
		// POST /api/patches - Create patch operation
		// GET /api/patches - List/get patch operations
		// PATCH /api/patches - Update patch operation
		e.Router.POST("/api/patches", withAuth(func(c *core.RequestEvent) error {
			return patches.HandlePatchOperations(app, c)
		}))

		e.Router.GET("/api/patches", withAuth(func(c *core.RequestEvent) error {
			return patches.HandlePatchOperations(app, c)
		}))

		e.Router.PATCH("/api/patches", withAuth(func(c *core.RequestEvent) error {
			return patches.HandlePatchOperations(app, c)
		}))

		// Agent endpoints
		e.Router.POST("/api/patches/{id}/result", withAuth(func(c *core.RequestEvent) error {
			return patches.HandlePatchResult(app, c)
		}))

		e.Router.GET("/api/scripts/{id}/validate", withAuth(func(c *core.RequestEvent) error {
			return patches.HandleValidateScript(app, c)
		}))

		return e.Next()
	})
}
