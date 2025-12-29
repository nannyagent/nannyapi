package hooks

import (
	"github.com/nannyagent/nannyapi/internal/proxmox"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterProxmoxHooks registers the Proxmox related hooks and routes
func RegisterProxmoxHooks(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Ingestion endpoints
		e.Router.POST("/api/proxmox/cluster", RequireAuthCollection("agents")(func(e *core.RequestEvent) error {
			return proxmox.HandleIngestCluster(app, e)
		}))

		e.Router.POST("/api/proxmox/node", RequireAuthCollection("agents")(func(e *core.RequestEvent) error {
			return proxmox.HandleIngestNode(app, e)
		}))

		e.Router.POST("/api/proxmox/lxc", RequireAuthCollection("agents")(func(e *core.RequestEvent) error {
			return proxmox.HandleIngestLXC(app, e)
		}))

		e.Router.POST("/api/proxmox/qemu", RequireAuthCollection("agents")(func(e *core.RequestEvent) error {
			return proxmox.HandleIngestQemu(app, e)
		}))

		// CRUD endpoints
		// Clusters
		e.Router.GET("/api/proxmox/clusters", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleListClusters(app, e)
		}))

		e.Router.GET("/api/proxmox/clusters/{id}", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleGetCluster(app, e)
		}))

		e.Router.DELETE("/api/proxmox/clusters/{id}", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleDeleteCluster(app, e)
		}))

		// Nodes
		e.Router.GET("/api/proxmox/nodes", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleListNodes(app, e)
		}))

		e.Router.GET("/api/proxmox/nodes/{id}", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleGetNode(app, e)
		}))

		e.Router.DELETE("/api/proxmox/nodes/{id}", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleDeleteNode(app, e)
		}))

		// LXC
		e.Router.GET("/api/proxmox/lxc", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleListLXC(app, e)
		}))

		e.Router.GET("/api/proxmox/lxc/{id}", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleGetLXC(app, e)
		}))

		e.Router.DELETE("/api/proxmox/lxc/{id}", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleDeleteLXC(app, e)
		}))

		// QEMU
		e.Router.GET("/api/proxmox/qemu", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleListQemu(app, e)
		}))

		e.Router.GET("/api/proxmox/qemu/{id}", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleGetQemu(app, e)
		}))

		e.Router.DELETE("/api/proxmox/qemu/{id}", RequireAuthCollection("users")(func(e *core.RequestEvent) error {
			return proxmox.HandleDeleteQemu(app, e)
		}))

		return e.Next()
	})
}
