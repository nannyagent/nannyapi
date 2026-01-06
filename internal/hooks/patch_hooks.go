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
		defer func() {
			if cerr := reader.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

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

	// Hook to populate script_id, script_url and exclusions on patch operation creation
	app.OnRecordCreate("patch_operations").BindFunc(func(e *core.RecordEvent) error {
		scriptID := e.Record.GetString("script_id")
		agentID := e.Record.GetString("agent_id")
		lxcID := e.Record.GetString("lxc_id")

		// 1. Populate script_id if missing
		if scriptID == "" {
			var err error
			scriptID, err = resolveScriptID(app, agentID, lxcID)
			if err != nil {
				return err
			}
			if scriptID != "" {
				e.Record.Set("script_id", scriptID)
			}
		}

		// 2. Populate script_url
		if scriptID != "" {
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
		}

		// 3. Populate exclusions
		if agentID != "" {
			exceptions, err := app.FindRecordsByFilter("package_exceptions",
				"agent_id = {:agentID} && is_active = true && (expires_at = '' || expires_at >= @now)",
				"", 0, 0, map[string]interface{}{
					"agentID": agentID,
				})
			if err != nil {
				return err
			}

			var excludedPackages []string
			for _, ex := range exceptions {
				excludedPackages = append(excludedPackages, ex.GetString("package_name"))
			}

			if len(excludedPackages) > 0 {
				e.Record.Set("exclusions", excludedPackages)
			}
		}

		// 4. Populate vmid if lxc_id is present
		if lxcID != "" {
			lxcRecord, err := app.FindRecordById("proxmox_lxc", lxcID)
			if err != nil {
				return err
			}
			vmid := lxcRecord.GetInt("vmid")
			e.Record.Set("vmid", fmt.Sprintf("%d", vmid))
		}

		return e.Next()
	})

	// Hook to enforce unique patch schedule per agent/lxc
	validateUniqueSchedule := func(e *core.RecordEvent) error {
		agentID := e.Record.GetString("agent_id")
		lxcID := e.Record.GetString("lxc_id")

		// Uniqueness scope: agent_id + lxc_id
		// If lxc_id is empty, it scopes to agent_id (host)
		// We exclude the current record ID to allow updates
		filter := ""
		params := map[string]interface{}{
			"agent": agentID,
			"id":    e.Record.Id,
		}

		if lxcID == "" {
			// Check for host schedule (lxc_id is empty or null)
			filter = "agent_id = {:agent} && lxc_id = '' && id != {:id}"
		} else {
			// Check for specific LXC schedule
			filter = "agent_id = {:agent} && lxc_id = {:lxc} && id != {:id}"
			params["lxc"] = lxcID
		}

		// Check if any *other* record exists
		records, err := app.FindRecordsByFilter("patch_schedules", filter, "", 1, 0, params)
		if err != nil {
			return err
		}

		if len(records) > 0 {
			if lxcID != "" {
				return fmt.Errorf("a patch schedule already exists for this LXC container (agent: %s, lxc: %s)", agentID, lxcID)
			}
			return fmt.Errorf("a patch schedule already exists for this agent (%s)", agentID)
		}
		return e.Next()
	}

	app.OnRecordCreate("patch_schedules").BindFunc(validateUniqueSchedule)
	app.OnRecordUpdate("patch_schedules").BindFunc(validateUniqueSchedule)

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

func resolveScriptID(app core.App, agentID, lxcID string) (string, error) {
	var platformFamily string

	if lxcID != "" {
		// LXC Logic
		lxc, err := app.FindRecordById("proxmox_lxc", lxcID)
		if err != nil {
			return "", err
		}
		ostype := lxc.GetString("ostype")

		// Map ostype to platform_family
		mapping, err := app.FindRecordsByFilter("lxc_os_map", "ostype = {:os}", "", 1, 0, map[string]interface{}{"os": ostype})
		if err != nil {
			return "", err
		}
		if len(mapping) == 0 {
			return "", fmt.Errorf("no OS mapping found for ostype: %s", ostype)
		}
		platformFamily = mapping[0].GetString("platform_family")

	} else if agentID != "" {
		// Agent Logic
		agent, err := app.FindRecordById("agents", agentID)
		if err != nil {
			return "", err
		}
		platformFamily = agent.GetString("platform_family")
	}

	if platformFamily != "" {
		records, err := app.FindRecordsByFilter("scripts", "platform_family = {:platform}", "", 1, 0, map[string]interface{}{
			"platform": platformFamily,
		})
		if err != nil {
			return "", err
		}
		if len(records) == 0 {
			return "", fmt.Errorf("no script found for platform family: %s", platformFamily)
		}

		return records[0].Id, nil
	}

	return "", nil
}
