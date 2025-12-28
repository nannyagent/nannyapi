package proxmox

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Helper for pagination
func getPagination(e *core.RequestEvent) (int, int) {
	page := 1
	perPage := 20
	if p := e.Request.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if pp := e.Request.URL.Query().Get("perPage"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 {
			perPage = v
		}
	}
	return page, perPage
}

// Helper for list response
type ListResponse struct {
	Page       int            `json:"page"`
	PerPage    int            `json:"perPage"`
	TotalItems int64          `json:"totalItems"`
	TotalPages int            `json:"totalPages"`
	Items      []*core.Record `json:"items"`
}

func createListResponse(app core.App, collection string, pbFilter string, sqlFilter dbx.Expression, sort string, page, perPage int) (*ListResponse, error) {
	totalItems, err := app.CountRecords(collection, sqlFilter)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(perPage)))
	offset := (page - 1) * perPage

	records, err := app.FindRecordsByFilter(collection, pbFilter, sort, perPage, offset)
	if err != nil {
		return nil, err
	}

	return &ListResponse{
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Items:      records,
	}, nil
}

// HandleIngestCluster handles the ingestion of a Proxmox cluster
func HandleIngestCluster(app core.App, e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil || authRecord.Collection().Name != "agents" {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var req types.ProxmoxCluster
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Upsert Cluster
	record, err := app.FindFirstRecordByFilter("proxmox_cluster", "px_cluster_id = {:id}", dbx.Params{"id": req.ClusterID})
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if record == nil {
		collection, _ := app.FindCollectionByNameOrId("proxmox_cluster")
		record = core.NewRecord(collection)
	}

	record.Set("cluster_name", req.ClusterName)
	record.Set("nodes", req.Nodes)
	record.Set("quorate", req.Quorate)
	record.Set("version", req.Version)
	record.Set("px_cluster_id", req.ClusterID)

	// Set ownership (append to agents list)
	currentAgents := record.GetStringSlice("agents")

	newAgents := make([]string, 0, len(currentAgents)+1)
	newAgents = append(newAgents, currentAgents...)

	found := false
	for _, id := range newAgents {
		if id == authRecord.Id {
			found = true
			break
		}
	}
	if !found {
		newAgents = append(newAgents, authRecord.Id)
		record.Set("agents", newAgents)
		log.Printf("[DEBUG] Appended agent %s to cluster %s", authRecord.Id, record.Id)
	}

	record.Set("user_id", authRecord.GetString("user_id"))
	record.Set("recorded_at", time.Now())

	if err := app.Save(record); err != nil {
		return err
	}

	// Backfill cluster_id for nodes/lxc/qemu that might have been ingested before the cluster
	// This handles the case where Node/LXC/QEMU are reported before the Cluster
	backfillCollections := []string{"proxmox_nodes", "proxmox_lxc", "proxmox_qemu"}
	for _, colName := range backfillCollections {
		// Find records for this agent that don't have a cluster_id yet
		orphans, err := app.FindRecordsByFilter(colName, "agent_id = {:agent} && cluster_id = ''", "", 0, 0, dbx.Params{"agent": authRecord.Id})
		if err == nil && len(orphans) > 0 {
			for _, orphan := range orphans {
				orphan.Set("cluster_id", record.Id)
				if err := app.Save(orphan); err != nil {
					log.Printf("[ERROR] Failed to backfill cluster_id for %s %s: %v", colName, orphan.Id, err)
				} else {
					log.Printf("[DEBUG] Backfilled cluster_id for %s %s", colName, orphan.Id)
				}
			}
		}
	}

	return e.JSON(http.StatusOK, map[string]string{"message": "Cluster ingested successfully", "id": record.Id})
}

// HandleIngestNode handles the ingestion of a Proxmox node
func HandleIngestNode(app core.App, e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil || authRecord.Collection().Name != "agents" {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var req types.ProxmoxNode
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Upsert Node
	record, err := app.FindFirstRecordByFilter("proxmox_nodes", "name = {:name} && agent_id = {:agent}", dbx.Params{"name": req.Name, "agent": authRecord.Id})
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if record == nil {
		collection, _ := app.FindCollectionByNameOrId("proxmox_nodes")
		record = core.NewRecord(collection)
	}

	// Upsert Cluster link if needed
	// assumes cluster must already exist && agent is part of it
	clusterRec, err := app.FindFirstRecordByFilter("proxmox_cluster", "agents ~ {:agent}", dbx.Params{"agent": authRecord.Id})
	if err == nil {
		record.Set("cluster_id", clusterRec.Id)
	} else {
		log.Printf("[DEBUG] Node Ingest: Cluster not found for agent %s. Error: %v", authRecord.Id, err)
	}

	record.Set("ip", req.IP)
	record.Set("level", req.Level)
	record.Set("local", req.Local)
	record.Set("name", req.Name)
	record.Set("pve_version", req.PVEVersion)
	record.Set("px_node_id", req.NodeID)
	record.Set("online", req.Online)
	record.Set("recorded_at", time.Now())

	// Set ownership
	record.Set("agent_id", authRecord.Id)
	record.Set("user_id", authRecord.GetString("user_id"))

	if err := app.Save(record); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]string{"message": "Node ingested successfully", "id": record.Id})
}

// HandleIngestLXC handles the ingestion of a Proxmox LXC container
func HandleIngestLXC(app core.App, e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil || authRecord.Collection().Name != "agents" {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var req types.ProxmoxLXC
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Find Node DB ID
	nodeRec, err := app.FindFirstRecordByFilter("proxmox_nodes", "name = {:name} && agent_id = {:agent}", dbx.Params{"name": req.Node, "agent": authRecord.Id})
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Node not found: " + req.Node})
	}

	// Upsert LXC
	record, err := app.FindFirstRecordByFilter("proxmox_lxc", "vmid = {:vmid} && node_id = {:node}", dbx.Params{"vmid": req.VMID, "node": nodeRec.Id})
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if record == nil {
		collection, _ := app.FindCollectionByNameOrId("proxmox_lxc")
		record = core.NewRecord(collection)
	}

	record.Set("node_id", nodeRec.Id)
	// Upsert Cluster link if needed
	// assumes cluster must already exist && agent is part of it
	clusterRec, err := app.FindFirstRecordByFilter("proxmox_cluster", "agents ~ {:agent}", dbx.Params{"agent": authRecord.Id})
	if err == nil {
		record.Set("cluster_id", clusterRec.Id)
	} else {
		log.Printf("[DEBUG] LXC Ingest: Cluster not found for agent %s. Error: %v", authRecord.Id, err)
	}

	record.Set("name", req.Name)
	record.Set("lxc_id", req.LXCID)
	record.Set("status", req.Status)
	record.Set("ostype", req.OSType)
	record.Set("uptime", req.Uptime)
	record.Set("vmid", req.VMID)
	record.Set("recorded_at", time.Now())

	// Set ownership
	record.Set("agent_id", authRecord.Id)
	record.Set("user_id", authRecord.GetString("user_id"))

	if err := app.Save(record); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]string{"message": "LXC ingested successfully", "id": record.Id})
}

// HandleIngestQemu handles the ingestion of a Proxmox QEMU VM
func HandleIngestQemu(app core.App, e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil || authRecord.Collection().Name != "agents" {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var req types.ProxmoxQemu
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Find Node DB ID
	nodeRec, err := app.FindFirstRecordByFilter("proxmox_nodes", "name = {:name} && agent_id = {:agent}", dbx.Params{"name": req.Node, "agent": authRecord.Id})
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "Node not found: " + req.Node})
	}

	// Upsert Qemu
	record, err := app.FindFirstRecordByFilter("proxmox_qemu", "vmid = {:vmid} && node_id = {:node}", dbx.Params{"vmid": req.VMID, "node": nodeRec.Id})
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if record == nil {
		collection, _ := app.FindCollectionByNameOrId("proxmox_qemu")
		record = core.NewRecord(collection)
	}

	record.Set("node_id", nodeRec.Id)
	// Upsert Cluster link if needed
	// assumes cluster must already exist && agent is part of it
	clusterRec, err := app.FindFirstRecordByFilter("proxmox_cluster", "agents ~ {:agent}", dbx.Params{"agent": authRecord.Id})
	if err == nil {
		record.Set("cluster_id", clusterRec.Id)
	} else {
		log.Printf("[DEBUG] QEMU Ingest: Cluster not found for agent %s. Error: %v", authRecord.Id, err)
	}

	record.Set("name", req.Name)
	record.Set("qemu_id", req.QemuID)
	record.Set("status", req.Status)
	record.Set("ostype", req.OSType)
	record.Set("uptime", req.Uptime)
	record.Set("vmid", req.VMID)
	record.Set("vmgenid", req.VMGenID)
	record.Set("kvm", req.KVM)
	record.Set("boot", req.Boot)
	record.Set("host_cpu", req.HostCPU)
	record.Set("recorded_at", time.Now())

	// Set ownership
	record.Set("agent_id", authRecord.Id)
	record.Set("user_id", authRecord.GetString("user_id"))

	if err := app.Save(record); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]string{"message": "QEMU VM ingested successfully", "id": record.Id})
}

// Generic CRUD handlers

func HandleListClusters(app core.App, e *core.RequestEvent) error {
	page, perPage := getPagination(e)
	resp, err := createListResponse(app, "proxmox_cluster", "1=1", dbx.NewExp("1=1"), "", page, perPage)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, resp)
}

func HandleGetCluster(app core.App, e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := app.FindRecordById("proxmox_cluster", id)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, record)
}

func HandleDeleteCluster(app core.App, e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := app.FindRecordById("proxmox_cluster", id)
	if err != nil {
		return err
	}
	// Check ownership if user is authenticated
	if e.Auth != nil && e.Auth.Collection().Name == "users" {
		if record.GetString("user_id") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
		}
	}
	if err := app.Delete(record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]string{"message": "Cluster deleted successfully"})
}

func HandleListNodes(app core.App, e *core.RequestEvent) error {
	pbFilter := "1=1"
	sqlFilter := dbx.NewExp("1=1")
	if clusterID := e.Request.URL.Query().Get("cluster_id"); clusterID != "" {
		pbFilter += fmt.Sprintf(" && cluster_id = '%s'", clusterID)
		sqlFilter = dbx.And(sqlFilter, dbx.HashExp{"cluster_id": clusterID})
	}
	page, perPage := getPagination(e)
	resp, err := createListResponse(app, "proxmox_nodes", pbFilter, sqlFilter, "", page, perPage)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, resp)
}

func HandleGetNode(app core.App, e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := app.FindRecordById("proxmox_nodes", id)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, record)
}

func HandleDeleteNode(app core.App, e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := app.FindRecordById("proxmox_nodes", id)
	if err != nil {
		return err
	}
	if e.Auth != nil && e.Auth.Collection().Name == "users" {
		if record.GetString("user_id") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
		}
	}
	if err := app.Delete(record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]string{"message": "Node deleted successfully"})
}

func HandleListLXC(app core.App, e *core.RequestEvent) error {
	pbFilter := "1=1"
	sqlFilter := dbx.NewExp("1=1")
	if nodeID := e.Request.URL.Query().Get("node_id"); nodeID != "" {
		pbFilter += fmt.Sprintf(" && node_id = '%s'", nodeID)
		sqlFilter = dbx.And(sqlFilter, dbx.HashExp{"node_id": nodeID})
	}
	if clusterID := e.Request.URL.Query().Get("cluster_id"); clusterID != "" {
		pbFilter += fmt.Sprintf(" && cluster_id = '%s'", clusterID)
		sqlFilter = dbx.And(sqlFilter, dbx.HashExp{"cluster_id": clusterID})
	}
	page, perPage := getPagination(e)
	resp, err := createListResponse(app, "proxmox_lxc", pbFilter, sqlFilter, "", page, perPage)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, resp)
}

func HandleGetLXC(app core.App, e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := app.FindRecordById("proxmox_lxc", id)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, record)
}

func HandleDeleteLXC(app core.App, e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := app.FindRecordById("proxmox_lxc", id)
	if err != nil {
		return err
	}
	if e.Auth != nil && e.Auth.Collection().Name == "users" {
		if record.GetString("user_id") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
		}
	}
	if err := app.Delete(record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]string{"message": "LXC deleted successfully"})
}

func HandleListQemu(app core.App, e *core.RequestEvent) error {
	pbFilter := "1=1"
	sqlFilter := dbx.NewExp("1=1")
	if nodeID := e.Request.URL.Query().Get("node_id"); nodeID != "" {
		pbFilter += fmt.Sprintf(" && node_id = '%s'", nodeID)
		sqlFilter = dbx.And(sqlFilter, dbx.HashExp{"node_id": nodeID})
	}
	if clusterID := e.Request.URL.Query().Get("cluster_id"); clusterID != "" {
		pbFilter += fmt.Sprintf(" && cluster_id = '%s'", clusterID)
		sqlFilter = dbx.And(sqlFilter, dbx.HashExp{"cluster_id": clusterID})
	}
	page, perPage := getPagination(e)
	resp, err := createListResponse(app, "proxmox_qemu", pbFilter, sqlFilter, "", page, perPage)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, resp)
}

func HandleGetQemu(app core.App, e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := app.FindRecordById("proxmox_qemu", id)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, record)
}

func HandleDeleteQemu(app core.App, e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := app.FindRecordById("proxmox_qemu", id)
	if err != nil {
		return err
	}
	if e.Auth != nil && e.Auth.Collection().Name == "users" {
		if record.GetString("user_id") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
		}
	}
	if err := app.Delete(record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]string{"message": "QEMU VM deleted successfully"})
}
