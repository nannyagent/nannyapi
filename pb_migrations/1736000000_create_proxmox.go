package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create proxmox_cluster collection
		proxmoxClusterCollection := core.NewBaseCollection("proxmox_cluster")

		proxmoxClusterCollection.Fields.Add(&core.TextField{
			Name:     "cluster_name",
			Required: true,
		})

		proxmoxClusterCollection.Fields.Add(&core.NumberField{
			Name:     "nodes",
			Required: true,
		})

		proxmoxClusterCollection.Fields.Add(&core.NumberField{
			Name:     "quorate",
			Required: true,
		})

		proxmoxClusterCollection.Fields.Add(&core.NumberField{
			Name:     "version",
			Required: true,
		})

		proxmoxClusterCollection.Fields.Add(&core.TextField{
			Name:     "px_cluster_id",
			Required: true,
		})

		// API Rules
		proxmoxClusterCollection.ListRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxClusterCollection.ViewRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxClusterCollection.CreateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxClusterCollection.UpdateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxClusterCollection.DeleteRule = ptrString("user_id = @request.auth.id")

		if err := app.Save(proxmoxClusterCollection); err != nil {
			return err
		}

		// create proxmox_node collections now
		agentsCollection, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}

		proxmoxNodeCollection := core.NewBaseCollection("proxmox_nodes")

		proxmoxNodeCollection.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		proxmoxNodeCollection.Fields.Add(&core.RelationField{
			Name:          "cluster_id",
			Required:      false,
			CollectionId:  proxmoxClusterCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		proxmoxNodeCollection.Fields.Add(&core.TextField{
			Name:     "ip",
			Required: true,
		})

		proxmoxNodeCollection.Fields.Add(&core.TextField{
			Name:     "level",
			Required: false,
		})

		proxmoxNodeCollection.Fields.Add(&core.NumberField{
			Name:     "local",
			Required: false,
		})

		proxmoxNodeCollection.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})

		proxmoxNodeCollection.Fields.Add(&core.TextField{
			Name:     "pve_version",
			Required: true,
		})

		proxmoxNodeCollection.Fields.Add(&core.NumberField{
			Name:     "px_node_id",
			Required: false,
		})

		proxmoxNodeCollection.Fields.Add(&core.NumberField{
			Name:     "online",
			Required: false,
		})

		// API Rules
		proxmoxNodeCollection.ListRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxNodeCollection.ViewRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxNodeCollection.CreateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxNodeCollection.UpdateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxNodeCollection.DeleteRule = ptrString("user_id = @request.auth.id")

		if err := app.Save(proxmoxNodeCollection); err != nil {
			return err
		}

		// create proxmox_lxc collections now
		proxmoxLxcCollection := core.NewBaseCollection("proxmox_lxc")

		proxmoxLxcCollection.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		proxmoxLxcCollection.Fields.Add(&core.RelationField{
			Name:          "cluster_id",
			Required:      false,
			CollectionId:  proxmoxClusterCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		proxmoxLxcCollection.Fields.Add(&core.RelationField{
			Name:          "node_id",
			Required:      false,
			CollectionId:  proxmoxNodeCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		proxmoxLxcCollection.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})

		proxmoxLxcCollection.Fields.Add(&core.TextField{
			Name:     "lxc_id",
			Required: true,
		})

		proxmoxLxcCollection.Fields.Add(&core.TextField{
			Name:     "status",
			Required: true,
		})

		proxmoxLxcCollection.Fields.Add(&core.TextField{
			Name:     "ostype",
			Required: true,
		})

		proxmoxLxcCollection.Fields.Add(&core.TextField{
			Name:     "uptime",
			Required: false,
		})

		proxmoxLxcCollection.Fields.Add(&core.NumberField{
			Name:     "vmid",
			Required: false,
		})

		// we will add metrics like cpu, mem, disk & network later
		// API Rules
		proxmoxLxcCollection.ListRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxLxcCollection.ViewRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxLxcCollection.CreateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxLxcCollection.UpdateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxLxcCollection.DeleteRule = ptrString("user_id = @request.auth.id")

		if err := app.Save(proxmoxLxcCollection); err != nil {
			return err
		}

		// create proxmox_qemu collections now
		proxmoxQemuCollection := core.NewBaseCollection("proxmox_qemu")

		proxmoxQemuCollection.Fields.Add(&core.RelationField{
			Name:          "agent_id",
			Required:      true,
			CollectionId:  agentsCollection.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		proxmoxQemuCollection.Fields.Add(&core.RelationField{
			Name:          "cluster_id",
			Required:      false,
			CollectionId:  proxmoxClusterCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		proxmoxQemuCollection.Fields.Add(&core.RelationField{
			Name:          "node_id",
			Required:      false,
			CollectionId:  proxmoxNodeCollection.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})

		proxmoxQemuCollection.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})

		proxmoxQemuCollection.Fields.Add(&core.TextField{
			Name:     "qemu_id",
			Required: true,
		})

		proxmoxQemuCollection.Fields.Add(&core.TextField{
			Name:     "status",
			Required: true,
		})

		proxmoxQemuCollection.Fields.Add(&core.TextField{
			Name:     "ostype",
			Required: true,
		})

		proxmoxQemuCollection.Fields.Add(&core.NumberField{
			Name:     "kvm",
			Required: true,
		})

		proxmoxQemuCollection.Fields.Add(&core.NumberField{
			Name:     "host_cpu",
			Required: true,
		})

		proxmoxQemuCollection.Fields.Add(&core.TextField{
			Name:     "boot",
			Required: true,
		})

		proxmoxQemuCollection.Fields.Add(&core.TextField{
			Name:     "uptime",
			Required: false,
		})

		proxmoxQemuCollection.Fields.Add(&core.NumberField{
			Name:     "vmid",
			Required: false,
		})

		proxmoxQemuCollection.Fields.Add(&core.TextField{
			Name:     "vmgenid",
			Required: false,
		})

		// we will add metrics like cpu, mem, disk & network later

		// API Rules
		proxmoxQemuCollection.ListRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxQemuCollection.ViewRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxQemuCollection.CreateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxQemuCollection.UpdateRule = ptrString("user_id = @request.auth.id || agent_id = @request.auth.id ")
		proxmoxQemuCollection.DeleteRule = ptrString("user_id = @request.auth.id")

		if err := app.Save(proxmoxQemuCollection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		//Rollback: delete collections in reverse order
		collections := []string{"proxmox_qemu", "proxmox_lxc", "proxmox_nodes", "proxmox_cluster"}
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
