package tests

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"testing"

	"github.com/nannyagent/nannyapi/internal/types"
)

// TestSBOMTypesVulnerabilityCounts tests the GetVulnerabilityCounts method
func TestSBOMTypesVulnerabilityCounts(t *testing.T) {
	result := &types.SBOMScanResult{
		CriticalCount:   5,
		HighCount:       10,
		MediumCount:     20,
		LowCount:        30,
		NegligibleCount: 5,
		UnknownCount:    2,
	}

	counts := result.GetVulnerabilityCounts()

	if counts["critical"] != 5 {
		t.Errorf("Expected critical=5, got %d", counts["critical"])
	}
	if counts["high"] != 10 {
		t.Errorf("Expected high=10, got %d", counts["high"])
	}
	if counts["total"] != 72 {
		t.Errorf("Expected total=72, got %d", counts["total"])
	}
}

// TestSBOMScanTypes tests the scan type constants
func TestSBOMScanTypes(t *testing.T) {
	tests := []struct {
		scanType types.SBOMScanType
		expected string
	}{
		{types.ScanTypeHost, "host"},
		{types.ScanTypeContainer, "container"},
		{types.ScanTypeImage, "image"},
		{types.ScanTypeDirectory, "directory"},
	}

	for _, tc := range tests {
		if string(tc.scanType) != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.scanType)
		}
	}
}

// TestSBOMSeverityTypes tests the severity type constants
func TestSBOMSeverityTypes(t *testing.T) {
	tests := []struct {
		severity types.VulnerabilitySeverity
		expected string
	}{
		{types.SeverityCritical, "critical"},
		{types.SeverityHigh, "high"},
		{types.SeverityMedium, "medium"},
		{types.SeverityLow, "low"},
		{types.SeverityNegligible, "negligible"},
		{types.SeverityUnknown, "unknown"},
	}

	for _, tc := range tests {
		if string(tc.severity) != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.severity)
		}
	}
}

// TestSBOMFixStates tests the fix state constants
func TestSBOMFixStates(t *testing.T) {
	tests := []struct {
		state    types.FixState
		expected string
	}{
		{types.FixStateFixed, "fixed"},
		{types.FixStateNotFixed, "not-fixed"},
		{types.FixStateWontFix, "wont-fix"},
		{types.FixStateUnknown, "unknown"},
	}

	for _, tc := range tests {
		if string(tc.state) != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.state)
		}
	}
}

// TestSBOMScanStatuses tests the scan status constants
func TestSBOMScanStatuses(t *testing.T) {
	tests := []struct {
		status   types.SBOMScanStatus
		expected string
	}{
		{types.SBOMScanStatusPending, "pending"},
		{types.SBOMScanStatusScanning, "scanning"},
		{types.SBOMScanStatusCompleted, "completed"},
		{types.SBOMScanStatusFailed, "failed"},
	}

	for _, tc := range tests {
		if string(tc.status) != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.status)
		}
	}
}

// createTestSBOM creates a minimal test SBOM in Syft format
func createTestSBOM() []byte {
	sbom := map[string]interface{}{
		"artifacts": []map[string]interface{}{
			{
				"id":      "pkg:rpm/centos/openssl@1.1.1k-12.el8",
				"name":    "openssl",
				"version": "1.1.1k-12.el8",
				"type":    "rpm",
			},
		},
		"source": map[string]interface{}{
			"type":   "directory",
			"target": "/",
		},
		"distro": map[string]interface{}{
			"name":    "centos",
			"version": "8",
		},
	}

	data, _ := json.Marshal(sbom)
	return data
}

// createTestArchive creates a gzipped tar archive containing the SBOM
func createTestArchive(t *testing.T, sbomData []byte) []byte {
	buf := &bytes.Buffer{}
	gzWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add SBOM file to archive
	header := &tar.Header{
		Name: "sbom.json",
		Mode: 0600,
		Size: int64(len(sbomData)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write(sbomData); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	return buf.Bytes()
}

// TestGrypeOutputParsing tests parsing of grype JSON output
func TestGrypeOutputParsing(t *testing.T) {
	// Sample grype output (simplified)
	grypeJSON := `{
		"matches": [
			{
				"vulnerability": {
					"id": "CVE-2024-12345",
					"dataSource": "https://nvd.nist.gov/vuln/detail/CVE-2024-12345",
					"severity": "High",
					"cvss": [
						{
							"source": "nvd",
							"type": "CVSS_V3",
							"version": "3.1",
							"vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
							"metrics": {
								"baseScore": 9.8
							}
						}
					],
					"fix": {
						"versions": ["1.1.1n"],
						"state": "fixed"
					}
				},
				"artifact": {
					"name": "openssl",
					"version": "1.1.1k",
					"type": "rpm"
				}
			}
		],
		"descriptor": {
			"name": "grype",
			"version": "0.106.0",
			"db": {
				"status": {
					"schemaVersion": "6.1.4"
				}
			}
		}
	}`

	var output types.GrypeOutput
	if err := json.Unmarshal([]byte(grypeJSON), &output); err != nil {
		t.Fatalf("Failed to parse grype output: %v", err)
	}

	// Verify matches
	if len(output.Matches) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(output.Matches))
	}

	match := output.Matches[0]

	// Verify vulnerability info
	if match.Vulnerability.ID != "CVE-2024-12345" {
		t.Errorf("Expected CVE-2024-12345, got %s", match.Vulnerability.ID)
	}
	if match.Vulnerability.Severity != "High" {
		t.Errorf("Expected High severity, got %s", match.Vulnerability.Severity)
	}

	// Verify CVSS
	if len(match.Vulnerability.CVSS) != 1 {
		t.Fatalf("Expected 1 CVSS entry, got %d", len(match.Vulnerability.CVSS))
	}
	if match.Vulnerability.CVSS[0].Metrics.BaseScore != 9.8 {
		t.Errorf("Expected CVSS 9.8, got %f", match.Vulnerability.CVSS[0].Metrics.BaseScore)
	}

	// Verify fix info
	if match.Vulnerability.Fix.State != "fixed" {
		t.Errorf("Expected fixed state, got %s", match.Vulnerability.Fix.State)
	}
	if len(match.Vulnerability.Fix.Versions) != 1 || match.Vulnerability.Fix.Versions[0] != "1.1.1n" {
		t.Errorf("Unexpected fix versions: %v", match.Vulnerability.Fix.Versions)
	}

	// Verify artifact
	if match.Artifact.Name != "openssl" {
		t.Errorf("Expected openssl, got %s", match.Artifact.Name)
	}

	// Verify descriptor
	if output.Descriptor.Version != "0.106.0" {
		t.Errorf("Expected grype 0.106.0, got %s", output.Descriptor.Version)
	}
}

// TestVulnerabilityResponseSerialization tests JSON serialization of vulnerability responses
func TestVulnerabilityResponseSerialization(t *testing.T) {
	resp := types.GetVulnerabilitiesResponse{
		Vulnerabilities: []types.Vulnerability{
			{
				ID:              "vuln123",
				VulnerabilityID: "CVE-2024-12345",
				Severity:        types.SeverityCritical,
				PackageName:     "openssl",
				PackageVersion:  "1.1.1k",
				FixState:        types.FixStateFixed,
				FixVersions:     []string{"1.1.1n", "3.0.7"},
				CVSSScore:       9.8,
			},
		},
		Total:  1,
		Limit:  100,
		Offset: 0,
	}

	// Serialize
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to serialize response: %v", err)
	}

	// Deserialize
	var parsed types.GetVulnerabilitiesResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to deserialize response: %v", err)
	}

	// Verify
	if len(parsed.Vulnerabilities) != 1 {
		t.Fatalf("Expected 1 vulnerability, got %d", len(parsed.Vulnerabilities))
	}
	vuln := parsed.Vulnerabilities[0]
	if vuln.VulnerabilityID != "CVE-2024-12345" {
		t.Errorf("Expected CVE-2024-12345, got %s", vuln.VulnerabilityID)
	}
	if vuln.Severity != types.SeverityCritical {
		t.Errorf("Expected critical severity, got %s", vuln.Severity)
	}
	if len(vuln.FixVersions) != 2 {
		t.Errorf("Expected 2 fix versions, got %d", len(vuln.FixVersions))
	}
}

// TestSBOMUploadResponseSerialization tests JSON serialization of upload responses
func TestSBOMUploadResponseSerialization(t *testing.T) {
	resp := types.SBOMUploadResponse{
		ScanID:  "scan123",
		Status:  "completed",
		Message: "SBOM scanned successfully",
		VulnCounts: map[string]int{
			"critical": 5,
			"high":     10,
			"medium":   20,
			"total":    35,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var parsed types.SBOMUploadResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	if parsed.ScanID != "scan123" {
		t.Errorf("Expected scan123, got %s", parsed.ScanID)
	}
	if parsed.VulnCounts["critical"] != 5 {
		t.Errorf("Expected critical=5, got %d", parsed.VulnCounts["critical"])
	}
}

// TestArchiveCreation tests that we can create valid tar.gz archives
func TestArchiveCreation(t *testing.T) {
	sbomData := createTestSBOM()
	archive := createTestArchive(t, sbomData)

	// Verify it's a valid gzip
	gzReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer func() { _ = gzReader.Close() }()

	// Verify it's a valid tar
	tarReader := tar.NewReader(gzReader)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	if header.Name != "sbom.json" {
		t.Errorf("Expected filename sbom.json, got %s", header.Name)
	}

	// Read content
	content, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatalf("Failed to read tar content: %v", err)
	}

	if !bytes.Equal(content, sbomData) {
		t.Error("Archive content doesn't match original")
	}
}
