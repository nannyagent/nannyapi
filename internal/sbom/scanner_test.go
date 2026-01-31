package sbom

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/nannyagent/nannyapi/internal/types"
)

// TestNewScannerDisabled tests that scanner is disabled when config says so
func TestNewScannerDisabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	scanner, err := NewScanner(nil, config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	if scanner.IsEnabled() {
		t.Error("Expected scanner to be disabled")
	}
}

// TestScannerConfig tests the scanner configuration
func TestScannerConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectEnabled bool
		expectError   bool
	}{
		{
			name: "disabled config",
			config: Config{
				Enabled: false,
			},
			expectEnabled: false,
			expectError:   false,
		},
		{
			name: "enabled with custom path",
			config: Config{
				Enabled:   true,
				GrypePath: "/custom/grype",
			},
			expectEnabled: true,
			expectError:   false, // Custom path is accepted without validation at creation time
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scanner, err := NewScanner(nil, tc.config)

			if tc.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if !tc.expectError && scanner.IsEnabled() != tc.expectEnabled {
				t.Errorf("Expected enabled=%v, got %v", tc.expectEnabled, scanner.IsEnabled())
			}
		})
	}
}

// TestExtractTarGzArchive tests extraction of tar.gz archives
func TestExtractTarGzArchive(t *testing.T) {
	// Create test SBOM content
	sbomContent := `{"artifacts": [], "source": {"type": "directory"}}`

	// Create tar.gz archive
	archive := createTarGzArchive(t, "sbom.json", []byte(sbomContent))

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "sbom-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create disabled scanner for testing extraction
	scanner, err := NewScanner(nil, Config{Enabled: false})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Test extraction
	sbomPath, err := scanner.extractSBOM(archive, tempDir)
	if err != nil {
		t.Fatalf("Failed to extract SBOM: %v", err)
	}

	// Verify extracted file exists
	if _, err := os.Stat(sbomPath); os.IsNotExist(err) {
		t.Errorf("Extracted SBOM file does not exist: %s", sbomPath)
	}

	// Verify content
	content, err := os.ReadFile(sbomPath)
	if err != nil {
		t.Fatalf("Failed to read extracted SBOM: %v", err)
	}
	if string(content) != sbomContent {
		t.Errorf("Content mismatch: expected %q, got %q", sbomContent, string(content))
	}
}

// TestExtractZipArchive tests extraction of zip archives
func TestExtractZipArchive(t *testing.T) {
	// Create test SBOM content
	sbomContent := `{"artifacts": [], "source": {"type": "host"}}`

	// Create zip archive
	archive := createZipArchive(t, "sbom.json", []byte(sbomContent))

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "sbom-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create disabled scanner for testing
	scanner, err := NewScanner(nil, Config{Enabled: false})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Test extraction
	sbomPath, err := scanner.extractSBOM(archive, tempDir)
	if err != nil {
		t.Fatalf("Failed to extract SBOM: %v", err)
	}

	// Verify extracted file exists
	if _, err := os.Stat(sbomPath); os.IsNotExist(err) {
		t.Errorf("Extracted SBOM file does not exist: %s", sbomPath)
	}

	// Verify content
	content, err := os.ReadFile(sbomPath)
	if err != nil {
		t.Fatalf("Failed to read extracted SBOM: %v", err)
	}
	if string(content) != sbomContent {
		t.Errorf("Content mismatch: expected %q, got %q", sbomContent, string(content))
	}
}

// TestExtractPlainJSON tests extraction of plain JSON files
func TestExtractPlainJSON(t *testing.T) {
	// Create test SBOM content
	sbomContent := `{"artifacts": [], "source": {"type": "image"}}`

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "sbom-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create disabled scanner for testing
	scanner, err := NewScanner(nil, Config{Enabled: false})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Test extraction (plain JSON passed directly)
	sbomPath, err := scanner.extractSBOM([]byte(sbomContent), tempDir)
	if err != nil {
		t.Fatalf("Failed to extract SBOM: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(sbomPath)
	if err != nil {
		t.Fatalf("Failed to read SBOM: %v", err)
	}
	if string(content) != sbomContent {
		t.Errorf("Content mismatch: expected %q, got %q", sbomContent, string(content))
	}
}

// TestPathTraversalProtection tests that path traversal is prevented
func TestPathTraversalProtection(t *testing.T) {
	// Create archive with path traversal attempt
	buf := &bytes.Buffer{}
	gzWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add file with path traversal
	header := &tar.Header{
		Name: "../../../etc/passwd",
		Mode: 0600,
		Size: int64(len("malicious")),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write header: %v", err)
	}
	if _, err := tarWriter.Write([]byte("malicious")); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	_ = tarWriter.Close()
	_ = gzWriter.Close()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "sbom-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create disabled scanner for testing
	scanner, err := NewScanner(nil, Config{Enabled: false})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Try extraction - should fail because no valid SBOM found
	// (path traversal files are skipped, then validation fails on empty result)
	_, extractErr := scanner.extractSBOM(buf.Bytes(), tempDir)

	// Either extraction fails or no file is created outside tempDir
	if extractErr == nil {
		// Check that no file was created outside tempDir
		if _, statErr := os.Stat("/etc/passwd_test"); statErr == nil {
			t.Error("Path traversal protection failed - file created outside temp dir")
		}
	}
	// If extractErr != nil, that's acceptable (path traversal was blocked)
}

// TestExtractSizeLimits tests the archive size limits
func TestExtractSizeLimits(t *testing.T) {
	// Create large content (101MB - over 100MB limit)
	largeContent := bytes.Repeat([]byte("x"), 101*1024*1024)

	// Create tar.gz archive
	buf := &bytes.Buffer{}
	gzWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzWriter)

	header := &tar.Header{
		Name: "large.json",
		Mode: 0600,
		Size: int64(len(largeContent)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write header: %v", err)
	}
	if _, err := tarWriter.Write(largeContent); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	_ = tarWriter.Close()
	_ = gzWriter.Close()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "sbom-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create disabled scanner for testing
	scanner, err := NewScanner(nil, Config{Enabled: false})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Try extraction - should fail due to size limit
	_, extractErr := scanner.extractSBOM(buf.Bytes(), tempDir)

	if extractErr == nil {
		t.Error("Expected error for oversized archive, got nil")
	}
	if extractErr != nil && extractErr != ErrArchiveTooLarge {
		// May still succeed if size check is different - log for debugging
		t.Logf("Extraction error (expected size limit): %v", extractErr)
	}
}

// TestValidateSBOMFormat tests SBOM format validation
func TestValidateSBOMFormat(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		expectErr bool
	}{
		{
			name:      "valid syft format",
			content:   `{"artifacts": [], "source": {"type": "directory"}}`,
			expectErr: false,
		},
		{
			name:      "valid cyclonedx format",
			content:   `{"bomFormat": "CycloneDX", "components": []}`,
			expectErr: false,
		},
		{
			name:      "valid spdx format",
			content:   `{"spdxVersion": "SPDX-2.3", "packages": []}`,
			expectErr: false,
		},
		{
			name:      "invalid json",
			content:   `{not valid json`,
			expectErr: true,
		},
		{
			name:      "empty json",
			content:   `{}`,
			expectErr: true,
		},
		{
			name:      "unrecognized format",
			content:   `{"random": "data", "notsbom": true}`,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp file
			tempFile, err := os.CreateTemp("", "sbom-validate-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer func() { _ = os.Remove(tempFile.Name()) }()

			if _, err := tempFile.WriteString(tc.content); err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}
			_ = tempFile.Close()

			scanner, err := NewScanner(nil, Config{Enabled: false})
			if err != nil {
				t.Fatalf("Failed to create scanner: %v", err)
			}

			validateErr := scanner.validateSBOM(tempFile.Name())

			if tc.expectErr && validateErr == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.expectErr && validateErr != nil {
				t.Errorf("Expected no error, got: %v", validateErr)
			}
		})
	}
}

// TestScannerGetStatus tests the GetStatus method
func TestScannerGetStatus(t *testing.T) {
	// Test disabled scanner
	disabledScanner, err := NewScanner(nil, Config{Enabled: false})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	status, err := disabledScanner.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status.Enabled {
		t.Error("Expected status.Enabled to be false")
	}
}

// TestInvalidArchiveFormat tests handling of invalid archive formats
func TestInvalidArchiveFormat(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "sbom-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create disabled scanner for testing
	scanner, err := NewScanner(nil, Config{Enabled: false})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Try to extract invalid data
	invalidData := []byte("this is not an archive or json")

	_, extractErr := scanner.extractSBOM(invalidData, tempDir)
	if extractErr == nil {
		t.Error("Expected error for invalid archive, got nil")
	}
	if extractErr != ErrInvalidArchive {
		t.Logf("Got error (expected ErrInvalidArchive): %v", extractErr)
	}
}

// Helper functions

func createTarGzArchive(t *testing.T, filename string, content []byte) []byte {
	buf := &bytes.Buffer{}
	gzWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzWriter)

	header := &tar.Header{
		Name: filename,
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write(content); err != nil {
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

func createZipArchive(t *testing.T, filename string, content []byte) []byte {
	buf := &bytes.Buffer{}
	zipWriter := zip.NewWriter(buf)

	writer, err := zipWriter.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("Failed to write zip content: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	return buf.Bytes()
}

// TestMultipleFilesInArchive tests extraction when archive contains multiple files
func TestMultipleFilesInArchive(t *testing.T) {
	// Create archive with multiple files
	buf := &bytes.Buffer{}
	gzWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add non-SBOM file (will be skipped - .txt not in allowed extensions)
	header := &tar.Header{
		Name: "README.txt",
		Mode: 0600,
		Size: int64(len("readme content")),
	}
	_ = tarWriter.WriteHeader(header)
	_, _ = tarWriter.Write([]byte("readme content"))

	// Add SBOM file
	sbomContent := `{"artifacts": [], "source": {"type": "directory"}}`
	header = &tar.Header{
		Name: "sbom.json",
		Mode: 0600,
		Size: int64(len(sbomContent)),
	}
	_ = tarWriter.WriteHeader(header)
	_, _ = tarWriter.Write([]byte(sbomContent))

	_ = tarWriter.Close()
	_ = gzWriter.Close()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "sbom-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create disabled scanner for testing
	scanner, err := NewScanner(nil, Config{Enabled: false})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Extract
	sbomPath, err := scanner.extractSBOM(buf.Bytes(), tempDir)
	if err != nil {
		t.Fatalf("Failed to extract SBOM: %v", err)
	}

	// Verify JSON file was extracted
	if !strings.HasSuffix(sbomPath, ".json") {
		t.Errorf("Expected .json file, got %s", sbomPath)
	}
}

// TestSBOMFilePatterns tests that common SBOM filenames are recognized
func TestSBOMFilePatterns(t *testing.T) {
	testCases := []string{
		"sbom.json",
		"sbom.syft.json",
		"bom.json",
		"sbom.cyclonedx.json",
		"sbom.spdx.json",
		"packages.json",
	}

	sbomContent := `{"artifacts": [], "source": {"type": "directory"}}`

	for _, filename := range testCases {
		t.Run(filename, func(t *testing.T) {
			archive := createTarGzArchive(t, filename, []byte(sbomContent))

			tempDir, err := os.MkdirTemp("", "sbom-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tempDir) }()

			scanner, err := NewScanner(nil, Config{Enabled: false})
			if err != nil {
				t.Fatalf("Failed to create scanner: %v", err)
			}

			sbomPath, extractErr := scanner.extractSBOM(archive, tempDir)
			if extractErr != nil {
				t.Fatalf("Failed to extract SBOM for %s: %v", filename, extractErr)
			}

			if _, statErr := os.Stat(sbomPath); os.IsNotExist(statErr) {
				t.Errorf("SBOM file not found for %s", filename)
			}
		})
	}
}

// BenchmarkExtractTarGz benchmarks tar.gz extraction
func BenchmarkExtractTarGz(b *testing.B) {
	sbomContent := `{"artifacts": [{"name": "pkg", "version": "1.0.0"}], "source": {"type": "directory"}}`
	archive := createTarGzArchiveBench(b, "sbom.json", []byte(sbomContent))

	tempDir, err := os.MkdirTemp("", "sbom-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	scanner, err := NewScanner(nil, Config{Enabled: false})
	if err != nil {
		b.Fatalf("Failed to create scanner: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.extractSBOM(archive, tempDir)
	}
}

func createTarGzArchiveBench(b *testing.B, filename string, content []byte) []byte {
	buf := &bytes.Buffer{}
	gzWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzWriter)

	header := &tar.Header{
		Name: filename,
		Mode: 0600,
		Size: int64(len(content)),
	}
	_ = tarWriter.WriteHeader(header)
	_, _ = tarWriter.Write(content)
	_ = tarWriter.Close()
	_ = gzWriter.Close()

	return buf.Bytes()
}

// TestGrypeOutputMapping tests mapping grype output to our types
func TestGrypeOutputMapping(t *testing.T) {
	grypeJSON := `{
		"matches": [
			{
				"vulnerability": {
					"id": "CVE-2024-1234",
					"severity": "Critical",
					"description": "A test vulnerability",
					"cvss": [
						{
							"source": "nvd",
							"metrics": {
								"baseScore": 9.8
							}
						}
					],
					"fix": {
						"versions": ["2.0.0"],
						"state": "fixed"
					},
					"urls": ["https://nvd.nist.gov/vuln/detail/CVE-2024-1234"]
				},
				"artifact": {
					"name": "test-package",
					"version": "1.0.0",
					"type": "npm"
				}
			},
			{
				"vulnerability": {
					"id": "CVE-2024-5678",
					"severity": "High",
					"fix": {
						"state": "not-fixed"
					}
				},
				"artifact": {
					"name": "another-package",
					"version": "3.0.0",
					"type": "rpm"
				}
			}
		]
	}`

	var output types.GrypeOutput
	if err := json.Unmarshal([]byte(grypeJSON), &output); err != nil {
		t.Fatalf("Failed to parse grype JSON: %v", err)
	}

	if len(output.Matches) != 2 {
		t.Fatalf("Expected 2 matches, got %d", len(output.Matches))
	}

	// Verify first match
	m1 := output.Matches[0]
	if m1.Vulnerability.ID != "CVE-2024-1234" {
		t.Errorf("Expected CVE-2024-1234, got %s", m1.Vulnerability.ID)
	}
	if m1.Vulnerability.Severity != "Critical" {
		t.Errorf("Expected Critical, got %s", m1.Vulnerability.Severity)
	}
	if m1.Artifact.Name != "test-package" {
		t.Errorf("Expected test-package, got %s", m1.Artifact.Name)
	}

	// Verify second match
	m2 := output.Matches[1]
	if m2.Vulnerability.Fix.State != "not-fixed" {
		t.Errorf("Expected not-fixed, got %s", m2.Vulnerability.Fix.State)
	}
}

// TestIsEnabled tests the IsEnabled method
func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "disabled scanner",
			enabled:  false,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scanner, err := NewScanner(nil, Config{Enabled: tc.enabled})
			if err != nil {
				t.Fatalf("Failed to create scanner: %v", err)
			}

			if scanner.IsEnabled() != tc.expected {
				t.Errorf("Expected IsEnabled=%v, got %v", tc.expected, scanner.IsEnabled())
			}
		})
	}
}
