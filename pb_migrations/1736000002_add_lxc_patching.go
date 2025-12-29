package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Add lxc_id to patch_operations
		patchOps, err := app.FindCollectionByNameOrId("patch_operations")
		if err != nil {
			return err
		}
		lxcCollection, err := app.FindCollectionByNameOrId("proxmox_lxc")
		if err != nil {
			return err
		}

		patchOps.Fields.Add(&core.RelationField{
			Name:          "lxc_id",
			Required:      false,
			CollectionId:  lxcCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		if err := app.Save(patchOps); err != nil {
			return err
		}

		// 2. Add lxc_id to package_exceptions
		pkgExceptions, err := app.FindCollectionByNameOrId("package_exceptions")
		if err != nil {
			return err
		}

		pkgExceptions.Fields.Add(&core.RelationField{
			Name:          "lxc_id",
			Required:      false,
			CollectionId:  lxcCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		// Update agent_id to be optional in package_exceptions if it was required
		// Actually, the user might want to set exceptions for an agent OR an LXC.
		// The original schema had agent_id as Required: true.
		// We should probably make it optional if lxc_id is present, but PocketBase validation is simple.
		// Let's keep agent_id required for now as the "host" agent is always involved,
		// but maybe for LXC exceptions, we link both agent_id (host) and lxc_id (container).
		// This allows filtering by agent to see all exceptions for that host and its containers.

		if err := app.Save(pkgExceptions); err != nil {
			return err
		}

		// 3. Create lxc_os_map collection
		lxcOsMap := core.NewBaseCollection("lxc_os_map")
		lxcOsMap.Fields.Add(&core.TextField{
			Name:     "ostype",
			Required: true,
		})
		lxcOsMap.Fields.Add(&core.TextField{
			Name:     "platform_family",
			Required: true,
		})
		// Add unique index on ostype? PocketBase doesn't have explicit unique index API in Go easily exposed here without raw SQL,
		// but we can just rely on app logic or pre-fill.

		// API Rules - Public read is fine for now, or restricted to auth
		lxcOsMap.ListRule = ptrString("@request.auth.id != ''")
		lxcOsMap.ViewRule = ptrString("@request.auth.id != ''")

		if err := app.Save(lxcOsMap); err != nil {
			return err
		}

		// 4. Seed lxc_os_map
		mappings := map[string]string{
			"alpine":    "alpine",
			"debian":    "debian",
			"ubuntu":    "debian",
			"centos":    "rhel",
			"fedora":    "rhel",
			"archlinux": "arch",
			"opensuse":  "suse",
		}

		for ostype, family := range mappings {
			rec := core.NewRecord(lxcOsMap)
			rec.Set("ostype", ostype)
			rec.Set("platform_family", family)
			if err := app.Save(rec); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// TO-DO: We don't support rollback of patch for now.
		// Revert logic if needed
		return nil
	})
}
