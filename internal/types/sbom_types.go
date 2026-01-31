package types

import (
	"encoding/json"
	"time"
)

// SBOMScanType represents the type of SBOM scan
type SBOMScanType string

const (
	ScanTypeHost      SBOMScanType = "host"
	ScanTypeContainer SBOMScanType = "container"
	ScanTypeImage     SBOMScanType = "image"
	ScanTypeDirectory SBOMScanType = "directory"
)

// SBOMScanStatus represents the status of an SBOM scan
type SBOMScanStatus string

const (
	SBOMScanStatusPending   SBOMScanStatus = "pending"
	SBOMScanStatusScanning  SBOMScanStatus = "scanning"
	SBOMScanStatusCompleted SBOMScanStatus = "completed"
	SBOMScanStatusFailed    SBOMScanStatus = "failed"
)

// VulnerabilitySeverity represents vulnerability severity levels
type VulnerabilitySeverity string

const (
	SeverityCritical   VulnerabilitySeverity = "critical"
	SeverityHigh       VulnerabilitySeverity = "high"
	SeverityMedium     VulnerabilitySeverity = "medium"
	SeverityLow        VulnerabilitySeverity = "low"
	SeverityNegligible VulnerabilitySeverity = "negligible"
	SeverityUnknown    VulnerabilitySeverity = "unknown"
)

// FixState represents the fix availability state
type FixState string

const (
	FixStateFixed    FixState = "fixed"
	FixStateNotFixed FixState = "not-fixed"
	FixStateWontFix  FixState = "wont-fix"
	FixStateUnknown  FixState = "unknown"
)

// SBOMUploadRequest is sent by agents when uploading SBOM archives
type SBOMUploadRequest struct {
	ScanType   SBOMScanType `json:"scan_type"`             // "host" or "container"
	SourceName string       `json:"source_name"`           // hostname or container name
	SourceType string       `json:"source_type,omitempty"` // "filesystem", "podman", etc.
}

// SBOMUploadResponse is returned after SBOM upload
type SBOMUploadResponse struct {
	Success    bool           `json:"success"`
	ScanID     string         `json:"scan_id"`
	Status     string         `json:"status"`
	Message    string         `json:"message"`
	VulnCounts map[string]int `json:"vuln_counts,omitempty"`
}

// SyftConfigResponse contains syft configuration for an agent
type SyftConfigResponse struct {
	ExcludePatterns []string `json:"exclude_patterns"`
}

// UpdateSyftConfigRequest is used to update agent's syft configuration
type UpdateSyftConfigRequest struct {
	ExcludePatterns []string `json:"exclude_patterns"`
}

// SBOMScanResult represents a complete scan result
type SBOMScanResult struct {
	ScanID          string          `json:"scan_id"`
	AgentID         string          `json:"agent_id"`
	ScanType        SBOMScanType    `json:"scan_type"`
	SourceName      string          `json:"source_name"`
	SourceType      string          `json:"source_type,omitempty"`
	Status          SBOMScanStatus  `json:"status"`
	TotalPackages   int             `json:"total_packages"`
	CriticalCount   int             `json:"critical_count"`
	HighCount       int             `json:"high_count"`
	MediumCount     int             `json:"medium_count"`
	LowCount        int             `json:"low_count"`
	NegligibleCount int             `json:"negligible_count"`
	UnknownCount    int             `json:"unknown_count"`
	FixableCount    int             `json:"fixable_count"`
	ArchivePath     string          `json:"archive_path,omitempty"`
	ScannedAt       *time.Time      `json:"scanned_at,omitempty"`
	GrypeVersion    string          `json:"grype_version,omitempty"`
	DBVersion       string          `json:"db_version,omitempty"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
}

// GetVulnerabilityCounts returns a map of severity to count
func (r *SBOMScanResult) GetVulnerabilityCounts() map[string]int {
	return map[string]int{
		"critical":   r.CriticalCount,
		"high":       r.HighCount,
		"medium":     r.MediumCount,
		"low":        r.LowCount,
		"negligible": r.NegligibleCount,
		"unknown":    r.UnknownCount,
		"total":      r.CriticalCount + r.HighCount + r.MediumCount + r.LowCount + r.NegligibleCount + r.UnknownCount,
	}
}

// Vulnerability represents a single vulnerability finding
type Vulnerability struct {
	ID                string                `json:"id"`
	VulnerabilityID   string                `json:"vulnerability_id"`
	Severity          VulnerabilitySeverity `json:"severity"`
	PackageName       string                `json:"package_name"`
	PackageVersion    string                `json:"package_version"`
	PackageType       string                `json:"package_type,omitempty"`
	FixState          FixState              `json:"fix_state,omitempty"`
	FixVersions       []string              `json:"fix_versions,omitempty"`
	Description       string                `json:"description,omitempty"`
	DataSource        string                `json:"data_source,omitempty"`
	RelatedCVEs       []string              `json:"related_cves,omitempty"`
	CVSSScore         float64               `json:"cvss_score,omitempty"`
	CVSSVector        string                `json:"cvss_vector,omitempty"`
	EPSSScore         float64               `json:"epss_score,omitempty"`
	EPSSPercentile    float64               `json:"epss_percentile,omitempty"`
	RiskScore         float64               `json:"risk_score,omitempty"`
	ArtifactLocations []string              `json:"artifact_locations,omitempty"`
	IsKEV             bool                  `json:"is_kev,omitempty"`
}

// VulnerabilitySummary represents aggregated vulnerability data per agent
type VulnerabilitySummary struct {
	ID                   string     `json:"id,omitempty"`
	AgentID              string     `json:"agent_id"`
	TotalScans           int        `json:"total_scans"`
	TotalVulnerabilities int        `json:"total_vulnerabilities"`
	CriticalCount        int        `json:"critical_count"`
	HighCount            int        `json:"high_count"`
	MediumCount          int        `json:"medium_count"`
	LowCount             int        `json:"low_count"`
	FixableCount         int        `json:"fixable_count"`
	LastScanAt           *time.Time `json:"last_scan_at,omitempty"`
	LastScanID           string     `json:"last_scan_id,omitempty"`
}

// ListScansRequest is used to list SBOM scans
type ListScansRequest struct {
	AgentID string `json:"agent_id,omitempty"`
	Page    int    `json:"page,omitempty"`
	PerPage int    `json:"per_page,omitempty"`
}

// ListScansResponse contains paginated scan results
type ListScansResponse struct {
	Scans  []SBOMScanResult `json:"scans"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// GetVulnerabilitiesRequest is used to get vulnerabilities
type GetVulnerabilitiesRequest struct {
	ScanID   string                `json:"scan_id,omitempty"`
	AgentID  string                `json:"agent_id,omitempty"`
	Severity VulnerabilitySeverity `json:"severity,omitempty"`
	Page     int                   `json:"page,omitempty"`
	PerPage  int                   `json:"per_page,omitempty"`
}

// GetVulnerabilitiesResponse contains paginated vulnerability results
type GetVulnerabilitiesResponse struct {
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Total           int             `json:"total"`
	Limit           int             `json:"limit"`
	Offset          int             `json:"offset"`
}

// SBOMStatusResponse indicates if vulnerability management is enabled
type SBOMStatusResponse struct {
	Enabled      bool   `json:"enabled"`
	GrypeVersion string `json:"grype_version,omitempty"`
	DBVersion    string `json:"db_version,omitempty"`
	DBBuiltAt    string `json:"db_built_at,omitempty"`
	Message      string `json:"message,omitempty"`
}

// GrypeMatch represents a vulnerability match from grype output
type GrypeMatch struct {
	Vulnerability          GrypeVulnerability `json:"vulnerability"`
	RelatedVulnerabilities []GrypeRelatedVuln `json:"relatedVulnerabilities"`
	MatchDetails           []GrypeMatchDetail `json:"matchDetails"`
	Artifact               GrypeArtifact      `json:"artifact"`
}

// GrypeVulnerability is the vulnerability info from grype
type GrypeVulnerability struct {
	ID          string          `json:"id"`
	DataSource  string          `json:"dataSource"`
	Namespace   string          `json:"namespace"`
	Severity    string          `json:"severity"`
	URLs        []string        `json:"urls"`
	Description string          `json:"description"`
	CVSS        []GrypeCVSS     `json:"cvss"`
	EPSS        []GrypeEPSS     `json:"epss"`
	CWEs        []GrypeCWE      `json:"cwes"`
	Fix         GrypeFix        `json:"fix"`
	Advisories  []GrypeAdvisory `json:"advisories"`
	Risk        float64         `json:"risk"`
}

// GrypeAdvisory contains advisory info from grype
type GrypeAdvisory struct {
	ID   string `json:"id"`
	Link string `json:"link"`
}

// GrypeRelatedVuln contains related vulnerability info (like CVE for GHSA)
type GrypeRelatedVuln struct {
	ID          string      `json:"id"`
	DataSource  string      `json:"dataSource"`
	Namespace   string      `json:"namespace"`
	Severity    string      `json:"severity"`
	URLs        []string    `json:"urls"`
	Description string      `json:"description"`
	CVSS        []GrypeCVSS `json:"cvss"`
	EPSS        []GrypeEPSS `json:"epss"`
	CWEs        []GrypeCWE  `json:"cwes"`
}

// GrypeCVSS contains CVSS scoring info
type GrypeCVSS struct {
	Source  string `json:"source"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Vector  string `json:"vector"`
	Metrics struct {
		BaseScore           float64 `json:"baseScore"`
		ExploitabilityScore float64 `json:"exploitabilityScore"`
		ImpactScore         float64 `json:"impactScore"`
	} `json:"metrics"`
}

// GrypeEPSS contains EPSS scoring info
type GrypeEPSS struct {
	CVE        string  `json:"cve"`
	EPSS       float64 `json:"epss"`
	Percentile float64 `json:"percentile"`
	Date       string  `json:"date"`
}

// GrypeCWE contains CWE info
type GrypeCWE struct {
	CVE    string `json:"cve"`
	CWE    string `json:"cwe"`
	Source string `json:"source"`
	Type   string `json:"type"`
}

// GrypeFix contains fix availability info
type GrypeFix struct {
	Versions  []string `json:"versions"`
	State     string   `json:"state"`
	Available []struct {
		Version string `json:"version"`
		Date    string `json:"date"`
	} `json:"available"`
}

// GrypeMatchDetail contains match details
type GrypeMatchDetail struct {
	Type       string `json:"type"`
	Matcher    string `json:"matcher"`
	SearchedBy struct {
		Language  string `json:"language"`
		Namespace string `json:"namespace"`
		Package   struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"package"`
	} `json:"searchedBy"`
	Found struct {
		VulnerabilityID   string `json:"vulnerabilityID"`
		VersionConstraint string `json:"versionConstraint"`
	} `json:"found"`
	Fix struct {
		SuggestedVersion string `json:"suggestedVersion"`
	} `json:"fix"`
}

// GrypeArtifact contains package/artifact info
type GrypeArtifact struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Type      string `json:"type"`
	Locations []struct {
		Path        string            `json:"path"`
		AccessPath  string            `json:"accessPath"`
		Annotations map[string]string `json:"annotations"`
	} `json:"locations"`
	Language string   `json:"language"`
	Licenses []string `json:"licenses"`
	CPEs     []string `json:"cpes"`
	PURL     string   `json:"purl"`
}

// GrypeOutput is the full grype JSON output structure
type GrypeOutput struct {
	Matches []GrypeMatch `json:"matches"`
	Source  struct {
		Type   string          `json:"type"`
		Target json.RawMessage `json:"target"` // Can be string (for dir/file) or object (for images)
	} `json:"source"`
	Distro struct {
		Name    string   `json:"name"`
		Version string   `json:"version"`
		IDLike  []string `json:"idLike"`
	} `json:"distro"`
	Descriptor struct {
		Name          string `json:"name"`
		Version       string `json:"version"`
		Configuration struct {
			DB struct {
				Status struct {
					SchemaVersion string `json:"schemaVersion"`
					From          string `json:"from"`
					Built         string `json:"built"`
					Path          string `json:"path"`
					Valid         bool   `json:"valid"`
				} `json:"status"`
			} `json:"db"`
		} `json:"configuration"`
		DB struct {
			Status struct {
				SchemaVersion string `json:"schemaVersion"`
				From          string `json:"from"`
				Built         string `json:"built"`
				Path          string `json:"path"`
				Valid         bool   `json:"valid"`
			} `json:"status"`
		} `json:"db"`
		Timestamp string `json:"timestamp"`
	} `json:"descriptor"`
}
