package sbom

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Handlers provides HTTP handlers for SBOM operations
type Handlers struct {
	app     core.App
	scanner *Scanner
}

// NewHandlers creates new SBOM handlers
func NewHandlers(app core.App, scanner *Scanner) *Handlers {
	return &Handlers{
		app:     app,
		scanner: scanner,
	}
}

// HandleSBOMStatus returns the SBOM scanning status
func (h *Handlers) HandleSBOMStatus(e *core.RequestEvent) error {
	status, err := h.scanner.GetStatus()
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get scanner status: " + err.Error(),
		})
	}

	return e.JSON(http.StatusOK, status)
}

// HandleSBOMUpload handles SBOM archive uploads
func (h *Handlers) HandleSBOMUpload(e *core.RequestEvent) error {
	if !h.scanner.IsEnabled() {
		return e.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Vulnerability scanning is not enabled. Contact administrator to enable --enable-vuln-scan flag.",
		})
	}

	// Get authenticated user/agent
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Determine agent ID and user ID based on authentication type
	var agentID, userID string

	// Check if this is an agent making the request
	if authRecord.Collection().Name == "agents" {
		agentID = authRecord.Id
		userID = authRecord.GetString("user_id")
	} else {
		// User is making request - agent_id should be in the request
		userID = authRecord.Id
		agentID = e.Request.Header.Get("X-Agent-ID")
		if agentID == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "X-Agent-ID header required when uploading as user",
			})
		}
		// Verify user owns the agent
		agentRecord, err := h.app.FindRecordById("agents", agentID)
		if err != nil || agentRecord.GetString("user_id") != userID {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Agent not found or not owned by user",
			})
		}
	}

	// Parse multipart form (max 50MB)
	if err := e.Request.ParseMultipartForm(MaxSBOMSize); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to parse form: " + err.Error(),
		})
	}

	// Get SBOM archive file
	file, header, err := e.Request.FormFile("sbom_archive")
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing sbom_archive file",
		})
	}
	defer func() { _ = file.Close() }()

	// Build request from form data
	req := types.SBOMUploadRequest{
		ScanType:   types.ScanTypeImage,
		SourceName: e.Request.FormValue("source_name"),
		SourceType: e.Request.FormValue("source_type"),
	}

	if scanType := e.Request.FormValue("scan_type"); scanType != "" {
		req.ScanType = types.SBOMScanType(scanType)
	}

	// Default source name to filename if not provided
	if req.SourceName == "" {
		req.SourceName = header.Filename
	}

	// Process the SBOM archive
	result, err := h.scanner.ProcessSBOMArchive(agentID, userID, file, header.Size, req)
	if err != nil {
		h.app.Logger().Error("SBOM processing failed",
			"agent_id", agentID,
			"error", err)

		statusCode := http.StatusInternalServerError
		switch err {
		case ErrArchiveTooLarge:
			statusCode = http.StatusRequestEntityTooLarge
		case ErrInvalidArchive, ErrNoSBOMFound:
			statusCode = http.StatusBadRequest
		}

		return e.JSON(statusCode, map[string]string{
			"error": err.Error(),
		})
	}

	return e.JSON(http.StatusOK, types.SBOMUploadResponse{
		ScanID:     result.ScanID,
		Status:     string(result.Status),
		Message:    "SBOM scanned successfully",
		VulnCounts: result.GetVulnerabilityCounts(),
	})
}

// HandleListScans lists SBOM scans for an agent
func (h *Handlers) HandleListScans(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Parse query parameters
	agentID := e.Request.URL.Query().Get("agent_id")
	status := e.Request.URL.Query().Get("status")
	limitStr := e.Request.URL.Query().Get("limit")
	offsetStr := e.Request.URL.Query().Get("offset")

	limit := 50
	offset := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build filter
	filter := ""
	filterParams := make(map[string]any)

	// For regular users, filter by their user_id
	if authRecord.Collection().Name != "agents" {
		filter = "user_id = {:userId}"
		filterParams["userId"] = authRecord.Id

		if agentID != "" {
			filter += " && agent_id = {:agentId}"
			filterParams["agentId"] = agentID
		}
	} else {
		// Agent can only see their own scans
		filter = "agent_id = {:agentId}"
		filterParams["agentId"] = authRecord.Id
	}

	if status != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "status = {:status}"
		filterParams["status"] = status
	}

	// Query scans
	collection, err := h.app.FindCollectionByNameOrId("sbom_scans")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to find collection",
		})
	}

	records, err := h.app.FindRecordsByFilter(collection, filter, "-scanned_at,-id", limit, offset, filterParams)
	if err != nil {
		h.app.Logger().Error("Failed to query scans",
			"error", err,
			"filter", filter,
			"params", filterParams)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to query scans: " + err.Error(),
		})
	}

	// Convert to response format
	scans := make([]types.SBOMScanResult, 0, len(records))
	for _, record := range records {
		scan := types.SBOMScanResult{
			ScanID:          record.Id,
			AgentID:         record.GetString("agent_id"),
			ScanType:        types.SBOMScanType(record.GetString("scan_type")),
			SourceName:      record.GetString("source_name"),
			SourceType:      record.GetString("source_type"),
			Status:          types.SBOMScanStatus(record.GetString("status")),
			TotalPackages:   record.GetInt("total_packages"),
			CriticalCount:   record.GetInt("critical_count"),
			HighCount:       record.GetInt("high_count"),
			MediumCount:     record.GetInt("medium_count"),
			LowCount:        record.GetInt("low_count"),
			NegligibleCount: record.GetInt("negligible_count"),
			UnknownCount:    record.GetInt("unknown_count"),
			GrypeVersion:    record.GetString("grype_version"),
			DBVersion:       record.GetString("db_version"),
			ErrorMessage:    record.GetString("error_message"),
		}

		if scannedAt := record.GetDateTime("scanned_at"); !scannedAt.IsZero() {
			t := scannedAt.Time()
			scan.ScannedAt = &t
		}

		scans = append(scans, scan)
	}

	// Get total count by querying all matching records
	allRecords, _ := h.app.FindRecordsByFilter(collection, filter, "", 0, 0, filterParams)
	total := len(allRecords)

	return e.JSON(http.StatusOK, types.ListScansResponse{
		Scans:  scans,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// HandleGetScan gets details of a specific scan
func (h *Handlers) HandleGetScan(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	scanID := e.Request.PathValue("id")
	if scanID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Scan ID required",
		})
	}

	// Get scan record
	record, err := h.app.FindRecordById("sbom_scans", scanID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Scan not found",
		})
	}

	// Verify ownership
	if authRecord.Collection().Name == "agents" {
		if record.GetString("agent_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	} else {
		if record.GetString("user_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	}

	scan := types.SBOMScanResult{
		ScanID:          record.Id,
		AgentID:         record.GetString("agent_id"),
		ScanType:        types.SBOMScanType(record.GetString("scan_type")),
		SourceName:      record.GetString("source_name"),
		SourceType:      record.GetString("source_type"),
		Status:          types.SBOMScanStatus(record.GetString("status")),
		TotalPackages:   record.GetInt("total_packages"),
		CriticalCount:   record.GetInt("critical_count"),
		HighCount:       record.GetInt("high_count"),
		MediumCount:     record.GetInt("medium_count"),
		LowCount:        record.GetInt("low_count"),
		NegligibleCount: record.GetInt("negligible_count"),
		UnknownCount:    record.GetInt("unknown_count"),
		GrypeVersion:    record.GetString("grype_version"),
		DBVersion:       record.GetString("db_version"),
		ErrorMessage:    record.GetString("error_message"),
	}

	if scannedAt := record.GetDateTime("scanned_at"); !scannedAt.IsZero() {
		t := scannedAt.Time()
		scan.ScannedAt = &t
	}

	return e.JSON(http.StatusOK, scan)
}

// HandleGetVulnerabilities gets vulnerabilities for a scan
// Supports advanced filtering:
//   - severity: single severity (e.g., "critical") for backward compatibility
//   - severities: comma-separated severities (e.g., "critical,high")
//   - min_cvss: minimum CVSS score (0.0-10.0)
//   - fixable: "true" for only fixable vulnerabilities (deprecated, use fix_state)
//   - fix_state: "fixed", "not-fixed", "wont-fix", or "unknown"
func (h *Handlers) HandleGetVulnerabilities(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	scanID := e.Request.PathValue("id")
	if scanID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Scan ID required",
		})
	}

	// Parse query parameters
	severity := e.Request.URL.Query().Get("severity")
	severities := e.Request.URL.Query().Get("severities")
	minCVSSStr := e.Request.URL.Query().Get("min_cvss")
	fixable := e.Request.URL.Query().Get("fixable")
	fixState := e.Request.URL.Query().Get("fix_state")
	limitStr := e.Request.URL.Query().Get("limit")
	offsetStr := e.Request.URL.Query().Get("offset")

	limit := 100
	offset := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse min CVSS score
	var minCVSS float64
	if minCVSSStr != "" {
		if cvss, err := strconv.ParseFloat(minCVSSStr, 64); err == nil && cvss >= 0 && cvss <= 10 {
			minCVSS = cvss
		}
	}

	// Verify scan access
	scanRecord, err := h.app.FindRecordById("sbom_scans", scanID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Scan not found",
		})
	}

	// Verify ownership
	if authRecord.Collection().Name == "agents" {
		if scanRecord.GetString("agent_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	} else {
		if scanRecord.GetString("user_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	}

	// Get archive path from scan record
	archivePath := scanRecord.GetString("archive_path")
	if archivePath == "" {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Vulnerability data not available for this scan",
		})
	}

	// Load vulnerabilities from archive on-demand
	grypeOutput, err := h.scanner.LoadVulnerabilitiesFromArchive(archivePath)
	if err != nil {
		h.app.Logger().Error("Failed to load vulnerabilities from archive",
			"scan_id", scanID,
			"archive_path", archivePath,
			"error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load vulnerability data",
		})
	}

	// Build severity filter set
	severityFilter := make(map[string]bool)
	if severities != "" {
		for _, sev := range strings.Split(severities, ",") {
			sev = strings.TrimSpace(strings.ToLower(sev))
			if sev != "" {
				severityFilter[sev] = true
			}
		}
	} else if severity != "" {
		severityFilter[strings.ToLower(severity)] = true
	}

	// Filter and convert grype matches to vulnerabilities
	allVulns := make([]types.Vulnerability, 0, len(grypeOutput.Matches))
	for _, match := range grypeOutput.Matches {
		sev := strings.ToLower(match.Vulnerability.Severity)

		// Apply severity filter
		if len(severityFilter) > 0 && !severityFilter[sev] {
			continue
		}

		// Get CVSS score
		var cvssScore float64
		var cvssVector string
		if len(match.Vulnerability.CVSS) > 0 {
			cvssScore = match.Vulnerability.CVSS[0].Metrics.BaseScore
			cvssVector = match.Vulnerability.CVSS[0].Vector
		} else if len(match.RelatedVulnerabilities) > 0 && len(match.RelatedVulnerabilities[0].CVSS) > 0 {
			cvssScore = match.RelatedVulnerabilities[0].CVSS[0].Metrics.BaseScore
			cvssVector = match.RelatedVulnerabilities[0].CVSS[0].Vector
		}

		// Apply CVSS filter
		if minCVSS > 0 && cvssScore < minCVSS {
			continue
		}

		// Apply fix state filter
		matchFixState := strings.ToLower(match.Vulnerability.Fix.State)
		if fixState != "" && matchFixState != strings.ToLower(fixState) {
			continue
		}
		if fixable == "true" && matchFixState != "fixed" {
			continue
		}

		// Get EPSS score
		var epssScore, epssPercentile float64
		if len(match.Vulnerability.EPSS) > 0 {
			epssScore = match.Vulnerability.EPSS[0].EPSS
			epssPercentile = match.Vulnerability.EPSS[0].Percentile
		}

		// Get description
		description := match.Vulnerability.Description
		if description == "" && len(match.RelatedVulnerabilities) > 0 {
			description = match.RelatedVulnerabilities[0].Description
		}

		// Collect locations
		locations := make([]string, 0, len(match.Artifact.Locations))
		for _, loc := range match.Artifact.Locations {
			locations = append(locations, loc.Path)
		}

		// Collect related CVEs
		relatedCVEs := make([]string, 0)
		for _, related := range match.RelatedVulnerabilities {
			if strings.HasPrefix(related.ID, "CVE-") {
				relatedCVEs = append(relatedCVEs, related.ID)
			}
		}

		vuln := types.Vulnerability{
			VulnerabilityID:   match.Vulnerability.ID,
			Severity:          types.VulnerabilitySeverity(sev),
			PackageName:       match.Artifact.Name,
			PackageVersion:    match.Artifact.Version,
			PackageType:       match.Artifact.Type,
			FixState:          types.FixState(matchFixState),
			FixVersions:       match.Vulnerability.Fix.Versions,
			Description:       description,
			DataSource:        match.Vulnerability.DataSource,
			RelatedCVEs:       relatedCVEs,
			CVSSScore:         cvssScore,
			CVSSVector:        cvssVector,
			EPSSScore:         epssScore,
			EPSSPercentile:    epssPercentile,
			RiskScore:         match.Vulnerability.Risk,
			ArtifactLocations: locations,
		}

		allVulns = append(allVulns, vuln)
	}

	// Sort by CVSS score descending (simple bubble sort for now)
	for i := 0; i < len(allVulns)-1; i++ {
		for j := i + 1; j < len(allVulns); j++ {
			if allVulns[j].CVSSScore > allVulns[i].CVSSScore {
				allVulns[i], allVulns[j] = allVulns[j], allVulns[i]
			}
		}
	}

	// Apply pagination
	total := len(allVulns)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	paginatedVulns := allVulns[start:end]

	return e.JSON(http.StatusOK, types.GetVulnerabilitiesResponse{
		Vulnerabilities: paginatedVulns,
		Total:           total,
		Limit:           limit,
		Offset:          offset,
	})
}

// HandleGetAgentSummary gets vulnerability summary for an agent
func (h *Handlers) HandleGetAgentSummary(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	agentID := e.Request.PathValue("agentId")
	if agentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Agent ID required",
		})
	}

	// Verify agent ownership
	agentRecord, err := h.app.FindRecordById("agents", agentID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Agent not found",
		})
	}

	if authRecord.Collection().Name == "agents" {
		if agentID != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	} else {
		if agentRecord.GetString("user_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	}

	// Get summary record
	collection, err := h.app.FindCollectionByNameOrId("vulnerability_summary")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to find collection",
		})
	}

	records, err := h.app.FindRecordsByFilter(collection, "agent_id = {:agentId}", "", 1, 0, map[string]any{"agentId": agentID})
	if err != nil || len(records) == 0 {
		// No summary yet
		return e.JSON(http.StatusOK, types.VulnerabilitySummary{
			AgentID: agentID,
		})
	}

	record := records[0]
	summary := types.VulnerabilitySummary{
		ID:                   record.Id,
		AgentID:              agentID,
		TotalVulnerabilities: record.GetInt("total_vulnerabilities"),
		CriticalCount:        record.GetInt("critical_count"),
		HighCount:            record.GetInt("high_count"),
		MediumCount:          record.GetInt("medium_count"),
		LowCount:             record.GetInt("low_count"),
		FixableCount:         record.GetInt("fixable_count"),
		TotalScans:           record.GetInt("total_scans"),
		LastScanID:           record.GetString("last_scan_id"),
	}

	if lastScanAt := record.GetDateTime("last_scan_at"); !lastScanAt.IsZero() {
		t := lastScanAt.Time()
		summary.LastScanAt = &t
	}

	return e.JSON(http.StatusOK, summary)
}

// HandleUpdateDB triggers a grype database update
func (h *Handlers) HandleUpdateDB(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil || authRecord.Collection().Name == "agents" {
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	if !h.scanner.IsEnabled() {
		return e.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Vulnerability scanning is not enabled",
		})
	}

	// Run DB update in background
	go func() {
		if err := h.scanner.UpdateDB(); err != nil {
			h.app.Logger().Error("Failed to update grype database", "error", err)
		}
	}()

	return e.JSON(http.StatusAccepted, map[string]string{
		"message": "Database update started",
	})
}

// HandleAgentVulnerabilities lists all vulnerabilities for an agent across all scans
// Supports advanced filtering:
//   - severity: single severity (e.g., "critical") for backward compatibility
//   - severities: comma-separated severities (e.g., "critical,high")
//   - min_cvss: minimum CVSS score (0.0-10.0)
//   - fix_state: "fixed", "not-fixed", "wont-fix", or "unknown"
func (h *Handlers) HandleAgentVulnerabilities(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	agentID := e.Request.PathValue("agentId")
	if agentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Agent ID required",
		})
	}

	// Parse query parameters
	severity := e.Request.URL.Query().Get("severity")
	severities := e.Request.URL.Query().Get("severities")
	minCVSSStr := e.Request.URL.Query().Get("min_cvss")
	fixState := e.Request.URL.Query().Get("fix_state")
	limitStr := e.Request.URL.Query().Get("limit")
	offsetStr := e.Request.URL.Query().Get("offset")

	limit := 100
	offset := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse min CVSS score
	var minCVSS float64
	if minCVSSStr != "" {
		if cvss, err := strconv.ParseFloat(minCVSSStr, 64); err == nil && cvss >= 0 && cvss <= 10 {
			minCVSS = cvss
		}
	}

	// Verify agent ownership
	agentRecord, err := h.app.FindRecordById("agents", agentID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Agent not found",
		})
	}

	if authRecord.Collection().Name == "agents" {
		if agentID != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	} else {
		if agentRecord.GetString("user_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	}

	// Build filter
	filter := "agent_id = {:agentId}"
	filterParams := map[string]any{"agentId": agentID}

	// Handle severity filtering
	if severities != "" {
		// Multiple severities (comma-separated)
		sevList := strings.Split(severities, ",")
		sevFilters := make([]string, 0, len(sevList))
		for i, sev := range sevList {
			sev = strings.TrimSpace(strings.ToLower(sev))
			if sev != "" {
				paramName := "severity" + strconv.Itoa(i)
				sevFilters = append(sevFilters, "severity = {:"+paramName+"}")
				filterParams[paramName] = sev
			}
		}
		if len(sevFilters) > 0 {
			filter += " && (" + strings.Join(sevFilters, " || ") + ")"
		}
	} else if severity != "" {
		// Single severity (backward compatibility)
		filter += " && severity = {:severity}"
		filterParams["severity"] = strings.ToLower(severity)
	}

	// Handle CVSS score filtering
	if minCVSS > 0 {
		filter += " && cvss_score >= {:minCvss}"
		filterParams["minCvss"] = minCVSS
	}

	// Handle fix state filtering
	if fixState != "" {
		filter += " && fix_state = {:fixState}"
		filterParams["fixState"] = strings.ToLower(fixState)
	}

	// Query vulnerabilities
	collection, err := h.app.FindCollectionByNameOrId("vulnerabilities")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to find collection",
		})
	}

	records, err := h.app.FindRecordsByFilter(collection, filter, "-cvss_score,-epss_score", limit, offset, filterParams)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to query vulnerabilities",
		})
	}

	// Convert to response format
	vulns := make([]types.Vulnerability, 0, len(records))
	for _, record := range records {
		vuln := types.Vulnerability{
			ID:              record.Id,
			VulnerabilityID: record.GetString("vulnerability_id"),
			Severity:        types.VulnerabilitySeverity(record.GetString("severity")),
			PackageName:     record.GetString("package_name"),
			PackageVersion:  record.GetString("package_version"),
			PackageType:     record.GetString("package_type"),
			FixState:        types.FixState(record.GetString("fix_state")),
			Description:     record.GetString("description"),
			CVSSScore:       record.GetFloat("cvss_score"),
		}

		vulns = append(vulns, vuln)
	}

	// Get total count by querying all matching records
	allAgentVulnRecords, _ := h.app.FindRecordsByFilter(collection, filter, "", 0, 0, filterParams)
	total := len(allAgentVulnRecords)

	return e.JSON(http.StatusOK, types.GetVulnerabilitiesResponse{
		Vulnerabilities: vulns,
		Total:           total,
		Limit:           limit,
		Offset:          offset,
	})
}

// HandleAcknowledgeScan handles agent acknowledging a scan request
func (h *Handlers) HandleAcknowledgeScan(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	scanID := e.Request.PathValue("id")
	if scanID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Scan ID required",
		})
	}

	// Get scan record
	record, err := h.app.FindRecordById("sbom_scans", scanID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Scan not found",
		})
	}

	// Verify ownership - agent can only acknowledge their own scans
	if authRecord.Collection().Name == "agents" {
		if record.GetString("agent_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	} else {
		if record.GetString("user_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	}

	// Update status to acknowledged/scanning
	record.Set("status", "scanning")
	if err := h.app.Save(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to acknowledge scan",
		})
	}

	return e.JSON(http.StatusOK, map[string]any{
		"success": true,
		"scan_id": scanID,
		"status":  "scanning",
	})
}

// HandleUpdateScanStatus handles agent updating scan status
func (h *Handlers) HandleUpdateScanStatus(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	scanID := e.Request.PathValue("id")
	if scanID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Scan ID required",
		})
	}

	// Parse request body
	var req struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message,omitempty"`
	}
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate status
	validStatuses := map[string]bool{
		"pending": true, "scanning": true, "completed": true, "failed": true,
	}
	if !validStatuses[req.Status] {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid status. Must be: pending, scanning, completed, or failed",
		})
	}

	// Get scan record
	record, err := h.app.FindRecordById("sbom_scans", scanID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Scan not found",
		})
	}

	// Verify ownership
	if authRecord.Collection().Name == "agents" {
		if record.GetString("agent_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	} else {
		if record.GetString("user_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Access denied",
			})
		}
	}

	// Update status
	record.Set("status", req.Status)
	if req.ErrorMessage != "" {
		record.Set("error_message", req.ErrorMessage)
	}
	if req.Status == "failed" || req.Status == "completed" {
		record.Set("scanned_at", time.Now().UTC())
	}

	if err := h.app.Save(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update scan status",
		})
	}

	return e.JSON(http.StatusOK, map[string]any{
		"success": true,
		"scan_id": scanID,
		"status":  req.Status,
	})
}

// HandleRequestScan creates a new scan request for an agent (user-initiated)
func (h *Handlers) HandleRequestScan(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Only users can request scans (not agents)
	if authRecord.Collection().Name == "agents" {
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "Only users can request scans",
		})
	}

	// Parse request
	var req struct {
		AgentID    string `json:"agent_id"`
		ScanType   string `json:"scan_type"`
		SourceName string `json:"source_name,omitempty"`
		SourceType string `json:"source_type,omitempty"`
	}
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.AgentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "agent_id is required",
		})
	}

	// Verify user owns the agent
	agentRecord, err := h.app.FindRecordById("agents", req.AgentID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Agent not found",
		})
	}
	if agentRecord.GetString("user_id") != authRecord.Id {
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "Agent not owned by user",
		})
	}

	// Default scan type
	if req.ScanType == "" {
		req.ScanType = "host"
	}

	// Create scan record with pending status
	collection, err := h.app.FindCollectionByNameOrId("sbom_scans")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to find collection",
		})
	}

	record := core.NewRecord(collection)
	record.Set("agent_id", req.AgentID)
	record.Set("user_id", authRecord.Id)
	record.Set("scan_type", req.ScanType)
	record.Set("source_name", req.SourceName)
	record.Set("source_type", req.SourceType)
	record.Set("status", "pending")

	if err := h.app.Save(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create scan request",
		})
	}

	return e.JSON(http.StatusCreated, map[string]any{
		"scan_id":     record.Id,
		"agent_id":    req.AgentID,
		"scan_type":   req.ScanType,
		"status":      "pending",
		"source_name": req.SourceName,
		"source_type": req.SourceType,
	})
}

// HandleGetPendingScans returns pending scans for an agent
func (h *Handlers) HandleGetPendingScans(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Only agents can get their pending scans via this endpoint
	if authRecord.Collection().Name != "agents" {
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "This endpoint is for agents only",
		})
	}

	// Query pending scans for this agent
	collection, err := h.app.FindCollectionByNameOrId("sbom_scans")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to find collection",
		})
	}

	filter := "agent_id = {:agentId} && status = 'pending'"
	records, err := h.app.FindRecordsByFilter(collection, filter, "-id", 10, 0, map[string]any{
		"agentId": authRecord.Id,
	})
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to query scans",
		})
	}

	scans := make([]map[string]any, 0, len(records))
	for _, record := range records {
		scans = append(scans, map[string]any{
			"scan_id":     record.Id,
			"scan_type":   record.GetString("scan_type"),
			"source_name": record.GetString("source_name"),
			"source_type": record.GetString("source_type"),
			"status":      record.GetString("status"),
		})
	}

	return e.JSON(http.StatusOK, map[string]any{
		"scans": scans,
		"total": len(scans),
	})
}

// HandleGetSyftConfig returns the syft configuration for an agent
// This includes exclusion patterns that the agent should use when generating SBOMs
func (h *Handlers) HandleGetSyftConfig(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	var agentID string
	if authRecord.Collection().Name == "agents" {
		agentID = authRecord.Id
	} else {
		// User requesting config for a specific agent
		agentID = e.Request.PathValue("agentId")
		if agentID == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "Agent ID required",
			})
		}
		// Verify user owns the agent
		agentRecord, err := h.app.FindRecordById("agents", agentID)
		if err != nil || agentRecord.GetString("user_id") != authRecord.Id {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "Agent not found or not owned by user",
			})
		}
	}

	// Get syft exclude patterns
	patterns, err := h.scanner.GetSyftExcludePatterns(agentID)
	if err != nil {
		h.app.Logger().Error("Failed to get syft exclude patterns",
			"agent_id", agentID,
			"error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get configuration",
		})
	}

	return e.JSON(http.StatusOK, types.SyftConfigResponse{
		ExcludePatterns: patterns,
	})
}

// HandleUpdateSyftConfig updates the syft configuration for an agent
func (h *Handlers) HandleUpdateSyftConfig(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Only users can update agent config (not agents themselves)
	if authRecord.Collection().Name == "agents" {
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "Agents cannot update their own configuration",
		})
	}

	agentID := e.Request.PathValue("agentId")
	if agentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Agent ID required",
		})
	}

	// Verify user owns the agent
	agentRecord, err := h.app.FindRecordById("agents", agentID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Agent not found",
		})
	}
	if agentRecord.GetString("user_id") != authRecord.Id {
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	// Parse request body
	var req types.UpdateSyftConfigRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Update agent's syft_exclude_patterns field
	agentRecord.Set("syft_exclude_patterns", req.ExcludePatterns)
	if err := h.app.Save(agentRecord); err != nil {
		h.app.Logger().Error("Failed to update agent syft config",
			"agent_id", agentID,
			"error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update configuration",
		})
	}

	return e.JSON(http.StatusOK, map[string]any{
		"success":          true,
		"agent_id":         agentID,
		"exclude_patterns": req.ExcludePatterns,
	})
}

// RegisterRoutes registers SBOM-related routes
// This should be called from within an OnServe handler
func RegisterRoutes(app core.App, scanner *Scanner, se *core.ServeEvent) {
	handlers := NewHandlers(app, scanner)

	// Status endpoint (public)
	se.Router.GET("/api/sbom/status", handlers.HandleSBOMStatus)

	// Protected endpoints
	group := se.Router.Group("/api/sbom")
	group.Bind(apis.RequireAuth())

	// Upload and scan management
	group.POST("/upload", handlers.HandleSBOMUpload)
	group.POST("/request", handlers.HandleRequestScan)    // User requests scan
	group.GET("/pending", handlers.HandleGetPendingScans) // Agent gets pending scans
	group.GET("/scans", handlers.HandleListScans)
	group.GET("/scans/{id}", handlers.HandleGetScan)
	group.POST("/scans/{id}/acknowledge", handlers.HandleAcknowledgeScan) // Agent acknowledges scan
	group.PATCH("/scans/{id}/status", handlers.HandleUpdateScanStatus)    // Agent updates status
	group.GET("/scans/{id}/vulnerabilities", handlers.HandleGetVulnerabilities)

	// Agent summaries and configuration
	group.GET("/agents/{agentId}/summary", handlers.HandleGetAgentSummary)
	group.GET("/agents/{agentId}/vulnerabilities", handlers.HandleAgentVulnerabilities)
	group.GET("/agents/{agentId}/syft-config", handlers.HandleGetSyftConfig)    // Get syft config
	group.PUT("/agents/{agentId}/syft-config", handlers.HandleUpdateSyftConfig) // Update syft config

	// Agent self-config endpoint (agents get their own config)
	group.GET("/config/syft", handlers.HandleGetSyftConfig)

	// Admin endpoints
	group.POST("/db/update", handlers.HandleUpdateDB)
}
