package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		patchOps, err := app.FindCollectionByNameOrId("patch_operations")
		if err != nil {
			return err
		}

		// Add vmid field (Text field to store the Proxmox VMID)
		// actually vmid translates to LXC ID or QEMU ID in Proxmox context
		patchOps.Fields.Add(&core.TextField{
			Name:     "vmid",
			Required: false,
		})

		return app.Save(patchOps)
	}, func(app core.App) error {
		patchOps, err := app.FindCollectionByNameOrId("patch_operations")
		if err != nil {
			return err
		}

		patchOps.Fields.RemoveByName("vmid")

		return app.Save(patchOps)
	})
}
