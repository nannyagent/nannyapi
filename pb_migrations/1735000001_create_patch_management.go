package pb_migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func init() {
	m.Register(func(app core.App) error {
		// Check if already exists
		collection, _ := app.FindCollectionByNameOrId("patch_operations")
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

		// Create scripts collection first so we can reference it
		scripts := core.NewBaseCollection("scripts")
		scripts.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})
		scripts.Fields.Add(&core.TextField{
			Name:     "description",
			Required: false,
		})
		scripts.Fields.Add(&core.TextField{
			Name:     "platform_family",
			Required: true,
		})
		scripts.Fields.Add(&core.TextField{
			Name:     "os_version",
			Required: false,
		})
		scripts.Fields.Add(&core.FileField{
			Name:      "file",
			Required:  true,
			MaxSelect: 1,
			MaxSize:   1024 * 1024 * 10, // 10MB
		})
		scripts.Fields.Add(&core.TextField{
			Name:     "sha256",
			Required: true,
		})

		// Set API rules for scripts (public read for agents)
		scripts.ListRule = ptrString("@request.auth.id != ''") // Authenticated users can list
		scripts.ViewRule = ptrString("@request.auth.id != ''") // Authenticated users can view/download

		if err := app.Save(scripts); err != nil {
			return err
		}

		// Create patch_operations collection
		patchOps := core.NewBaseCollection("patch_operations")

		// User who initiated patch
		patchOps.Fields.Add(&core.RelationField{
			Name:          "user_id",
			Required:      true,
			CollectionId:  usersCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Agent to patch
		patchOps.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Patch mode: dry-run or apply
		patchOps.Fields.Add(&core.TextField{
			Name:     "mode",
			Required: true,
			Max:      50,
		})

		// Operation status: pending, running, completed, failed, rolled_back
		patchOps.Fields.Add(&core.TextField{
			Name:     "status",
			Required: true,
			Max:      50,
		})

		// Reference to script in scripts collection
		patchOps.Fields.Add(&core.RelationField{
			Name:          "script_id",
			Required:      true,
			CollectionId:  scripts.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		// Legacy script URL (optional now)
		patchOps.Fields.Add(&core.TextField{
			Name:     "script_url",
			Required: false,
			Max:      1000,
		})

		// Path in storage where output was saved (null until completed)
		patchOps.Fields.Add(&core.TextField{
			Name:     "output_path",
			Required: false,
			Max:      1000,
		})

		// Error message if operation failed
		patchOps.Fields.Add(&core.TextField{
			Name:     "error_msg",
			Required: false,
			Max:      2000,
		})

		// When patch execution started
		patchOps.Fields.Add(&core.DateField{
			Name:     "started_at",
			Required: false,
		})

		// When patch execution completed
		patchOps.Fields.Add(&core.DateField{
			Name:     "completed_at",
			Required: false,
		})

		// File fields for stdout/stderr
		patchOps.Fields.Add(&core.FileField{
			Name:      "stdout_file",
			Required:  false,
			MaxSelect: 1,
			MaxSize:   1024 * 1024 * 10, // 10MB
		})
		patchOps.Fields.Add(&core.FileField{
			Name:      "stderr_file",
			Required:  false,
			MaxSelect: 1,
			MaxSize:   1024 * 1024 * 10, // 10MB
		})
		patchOps.Fields.Add(&core.NumberField{
			Name:     "exit_code",
			Required: false,
		})

		// Set API rules
		patchOps.ListRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id")
		patchOps.ViewRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id")
		patchOps.CreateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id")
		patchOps.UpdateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id")
		patchOps.DeleteRule = ptrString("user_id = @request.auth.id")

		if err := app.Save(patchOps); err != nil {
			return err
		}

		// Populate scripts from patch_scripts directory
		scriptDir := "patch_scripts"
		if _, err := os.Stat(scriptDir); err == nil {
			err = filepath.Walk(scriptDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}

				// Expected path: patch_scripts/<platform_family>/<script_name>
				relPath, err := filepath.Rel(scriptDir, path)
				if err != nil {
					return err
				}
				parts := strings.Split(relPath, string(os.PathSeparator))
				if len(parts) != 2 {
					return nil // Skip files not in platform subdirectories
				}
				platformFamily := parts[0]
				scriptName := parts[1]

				// Calculate SHA256
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()

				h := sha256.New()
				if _, err := io.Copy(h, f); err != nil {
					return err
				}
				hash := hex.EncodeToString(h.Sum(nil))

				// Re-open file for upload
				f.Seek(0, 0)
				content, err := io.ReadAll(f)
				if err != nil {
					return err
				}
				file, err := filesystem.NewFileFromBytes(content, scriptName)
				if err != nil {
					return err
				}

				record := core.NewRecord(scripts)
				record.Set("name", scriptName)
				record.Set("platform_family", platformFamily)
				record.Set("file", file)
				record.Set("sha256", hash)
				record.Set("description", "Auto-imported patch script")

				return app.Save(record)
			})
			if err != nil {
				return err
			}
		}

		// Update agent_metrics collection with distro info
		metrics, err := app.FindCollectionByNameOrId("agent_metrics")
		if err == nil {
			metrics.Fields.Add(&core.TextField{
				Name:     "distro_type",
				Required: false,
			})
			metrics.Fields.Add(&core.TextField{
				Name:     "distro_version",
				Required: false,
			})
			app.Save(metrics)
		}

		// Create package_updates collection to track packages affected by patches
		packageUpdates := core.NewBaseCollection("package_updates")

		// Reference to patch operation
		packageUpdates.Fields.Add(&core.RelationField{
			Name:          "patch_op_id",
			Required:      true,
			CollectionId:  patchOps.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Package name
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "package_name",
			Required: true,
			Max:      255,
		})

		// Current installed version
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "current_ver",
			Required: false,
			Max:      100,
		})

		// Target version (empty for removal)
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "target_ver",
			Required: false,
			Max:      100,
		})

		// Update type: install, update, remove
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "update_type",
			Required: true,
			Max:      50,
		})

		// Status: pending, applied, failed
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "status",
			Required: true,
			Max:      50,
		})

		// Dry-run simulation results
		packageUpdates.Fields.Add(&core.TextField{
			Name:     "dry_run_results",
			Required: false,
			Max:      2000,
		})

		// Set API rules - only through patch operation owner
		packageUpdates.ListRule = ptrString("patch_op_id.user_id = @request.auth.id")
		packageUpdates.ViewRule = ptrString("patch_op_id.user_id = @request.auth.id")
		packageUpdates.CreateRule = ptrString("") // Backend only
		packageUpdates.UpdateRule = ptrString("") // Backend only
		packageUpdates.DeleteRule = ptrString("") // Backend only

		if err := app.Save(packageUpdates); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete collections in reverse order
		collections := []string{"package_updates", "scripts", "patch_operations"}
		for _, name := range collections {
			collection, _ := app.FindCollectionByNameOrId(name)
			if collection != nil {
				if err := app.Delete(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
