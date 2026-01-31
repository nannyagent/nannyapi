package sbom

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// MaxSBOMSize is the maximum allowed SBOM archive size (50MB)
	MaxSBOMSize = 50 * 1024 * 1024

	// MaxExtractedSize is the maximum total extracted size (100MB)
	MaxExtractedSize = 100 * 1024 * 1024

	// SBOMTempDir is the temporary directory for SBOM processing
	SBOMTempDir = "/tmp/nannyapi_sbom"

	// SBOMStorageDir is the subdirectory in pb_data/storage for SBOM archives
	SBOMStorageDir = "sbom_archives"

	// AllowedSBOMExtensions are valid SBOM file extensions
	AllowedSBOMExtensions = ".json,.xml,.spdx,.cdx"

	// DefaultGrypeDBCacheDir is the default grype database cache directory
	DefaultGrypeDBCacheDir = "/var/cache/grype/db"

	// DefaultScansPerAgent is the default retention limit per agent
	DefaultScansPerAgent = 10
)

var (
	// ErrSBOMDisabled is returned when vulnerability scanning is disabled
	ErrSBOMDisabled = errors.New("vulnerability scanning is not enabled")

	// ErrInvalidArchive is returned for invalid archive formats
	ErrInvalidArchive = errors.New("invalid archive format")

	// ErrArchiveTooLarge is returned when archive exceeds size limit
	ErrArchiveTooLarge = errors.New("archive exceeds maximum size limit")

	// ErrNoSBOMFound is returned when no SBOM file is found in archive
	ErrNoSBOMFound = errors.New("no valid SBOM file found in archive")

	// ErrGrypeNotInstalled is returned when grype is not available
	ErrGrypeNotInstalled = errors.New("grype is not installed or not in PATH")

	// ErrGrypeDBInvalid is returned when grype database is invalid
	ErrGrypeDBInvalid = errors.New("grype vulnerability database is invalid or not available")
)

// Scanner handles SBOM scanning operations
type Scanner struct {
	app        core.App
	enabled    bool
	grypePath  string
	dbCacheDir string
	mu         sync.RWMutex
}

// Config holds scanner configuration
type Config struct {
	Enabled    bool
	GrypePath  string
	DBCacheDir string
}

// NewScanner creates a new SBOM scanner
func NewScanner(app core.App, cfg Config) (*Scanner, error) {
	s := &Scanner{
		app:        app,
		enabled:    cfg.Enabled,
		dbCacheDir: cfg.DBCacheDir,
	}

	if !cfg.Enabled {
		return s, nil
	}

	// Find grype binary
	grypePath := cfg.GrypePath
	if grypePath == "" {
		var err error
		grypePath, err = exec.LookPath("grype")
		if err != nil {
			return nil, ErrGrypeNotInstalled
		}
	}
	s.grypePath = grypePath

	// Set default DB cache dir
	if s.dbCacheDir == "" {
		s.dbCacheDir = DefaultGrypeDBCacheDir
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(SBOMTempDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return s, nil
}

// IsEnabled returns whether vulnerability scanning is enabled
func (s *Scanner) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// GrypePath returns the path to the grype binary
func (s *Scanner) GrypePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.grypePath
}

// GetStatus returns the scanner status
func (s *Scanner) GetStatus() (*types.SBOMStatusResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.enabled {
		return &types.SBOMStatusResponse{
			Enabled: false,
			Message: "Vulnerability scanning is not enabled. Set --enable-vuln-scan flag to activate.",
		}, nil
	}

	// Get grype version
	version, err := s.getGrypeVersion()
	if err != nil {
		return &types.SBOMStatusResponse{
			Enabled: true,
			Message: "Grype is enabled but version check failed: " + err.Error(),
		}, nil
	}

	// Get DB status
	dbStatus, err := s.getDBStatus()
	if err != nil {
		return &types.SBOMStatusResponse{
			Enabled:      true,
			GrypeVersion: version,
			Message:      "Grype database status check failed: " + err.Error(),
		}, nil
	}

	return &types.SBOMStatusResponse{
		Enabled:      true,
		GrypeVersion: version,
		DBVersion:    dbStatus.SchemaVersion,
		DBBuiltAt:    dbStatus.Built,
		Message:      "Vulnerability scanning is active",
	}, nil
}

// DBStatus holds grype database status info
type DBStatus struct {
	SchemaVersion string
	Built         string
	Valid         bool
}

func (s *Scanner) getGrypeVersion() (string, error) {
	cmd := exec.Command(s.grypePath, "version", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var result struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		// Fallback to plain text
		cmd = exec.Command(s.grypePath, "version")
		output, _ = cmd.Output()
		return strings.TrimSpace(string(output)), nil
	}
	return result.Version, nil
}

func (s *Scanner) getDBStatus() (*DBStatus, error) {
	cmd := exec.Command(s.grypePath, "db", "status", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result struct {
		SchemaVersion string `json:"schemaVersion"`
		Built         string `json:"built"`
		Valid         bool   `json:"valid"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &DBStatus{
		SchemaVersion: result.SchemaVersion,
		Built:         result.Built,
		Valid:         result.Valid,
	}, nil
}

// UpdateDB updates the grype vulnerability database
func (s *Scanner) UpdateDB() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		return ErrSBOMDisabled
	}

	cmd := exec.Command(s.grypePath, "db", "update")
	cmd.Env = append(os.Environ(), fmt.Sprintf("GRYPE_DB_CACHE_DIR=%s", s.dbCacheDir))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update grype database: %w: %s", err, string(output))
	}

	s.app.Logger().Info("Grype database updated successfully")
	return nil
}

// ProcessSBOMArchive extracts and scans an SBOM archive
func (s *Scanner) ProcessSBOMArchive(agentID, userID string, archive io.Reader, archiveSize int64, req types.SBOMUploadRequest) (*types.SBOMScanResult, error) {
	if !s.enabled {
		return nil, ErrSBOMDisabled
	}

	// Check archive size
	if archiveSize > MaxSBOMSize {
		return nil, ErrArchiveTooLarge
	}

	// Create a unique temp directory for this scan
	scanDir := filepath.Join(SBOMTempDir, fmt.Sprintf("%s_%d", agentID, time.Now().UnixNano()))
	if err := os.MkdirAll(scanDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create scan directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(scanDir) }() // Clean up after processing

	// Read archive into buffer (for detecting format)
	buf := &bytes.Buffer{}
	limitedReader := io.LimitReader(archive, MaxSBOMSize+1)
	n, err := io.Copy(buf, limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive: %w", err)
	}
	if n > MaxSBOMSize {
		return nil, ErrArchiveTooLarge
	}

	archiveBytes := buf.Bytes()

	// Detect and extract archive
	sbomPath, err := s.extractSBOM(archiveBytes, scanDir)
	if err != nil {
		return nil, err
	}

	// Create scan record in database
	scanRecord, err := s.createScanRecord(agentID, userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create scan record: %w", err)
	}

	// Run grype scan
	grypeOutput, err := s.runGrypeScan(sbomPath)
	if err != nil {
		// Update scan record with error
		s.updateScanWithError(scanRecord, err.Error())
		return nil, err
	}

	// Store grype output as archive and update scan record
	result, err := s.storeAndFinalizeScan(scanRecord, grypeOutput)
	if err != nil {
		s.updateScanWithError(scanRecord, err.Error())
		return nil, err
	}

	// Update vulnerability summary (without storing individual vulns)
	s.updateVulnerabilitySummary(agentID, userID, result)

	// Enforce retention limit for this agent
	s.enforceRetentionLimit(agentID)

	return result, nil
}

// extractSBOM extracts SBOM file from archive with security validations
func (s *Scanner) extractSBOM(archiveBytes []byte, destDir string) (string, error) {
	reader := bytes.NewReader(archiveBytes)

	// Try to detect archive type by magic bytes
	header := make([]byte, 512)
	if _, err := reader.Read(header); err != nil {
		return "", fmt.Errorf("failed to read archive header: %w", err)
	}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek: %w", err)
	}

	var sbomPath string
	var err error

	// Check for gzip magic bytes
	if header[0] == 0x1f && header[1] == 0x8b {
		sbomPath, err = s.extractTarGz(reader, destDir)
	} else if header[0] == 0x50 && header[1] == 0x4b { // ZIP magic bytes
		sbomPath, err = s.extractZip(archiveBytes, destDir)
	} else if header[0] == '{' || (header[0] == 0xef && header[1] == 0xbb && header[2] == 0xbf && header[3] == '{') {
		// Plain JSON (possibly with BOM)
		sbomPath = filepath.Join(destDir, "sbom.json")
		if writeErr := os.WriteFile(sbomPath, archiveBytes, 0600); writeErr != nil {
			return "", writeErr
		}
	} else {
		return "", ErrInvalidArchive
	}

	if err != nil {
		return "", err
	}

	// Validate the extracted file is a valid SBOM
	if err := s.validateSBOM(sbomPath); err != nil {
		return "", err
	}

	return sbomPath, nil
}

func (s *Scanner) extractTarGz(reader io.Reader, destDir string) (string, error) {
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)
	var extractedSize int64
	var sbomPath string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Security check: prevent path traversal
		cleanName := filepath.Clean(header.Name)
		if strings.Contains(cleanName, "..") {
			continue // Skip files with path traversal attempts
		}

		// Only extract files with allowed extensions
		ext := strings.ToLower(filepath.Ext(cleanName))
		if !strings.Contains(AllowedSBOMExtensions, ext) {
			continue
		}

		targetPath := filepath.Join(destDir, filepath.Base(cleanName))

		// Check extracted size limit
		extractedSize += header.Size
		if extractedSize > MaxExtractedSize {
			return "", ErrArchiveTooLarge
		}

		// Create file with restricted permissions
		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return "", fmt.Errorf("failed to create file: %w", err)
		}

		// Copy with size limit
		_, copyErr := io.CopyN(outFile, tarReader, header.Size)
		_ = outFile.Close()
		if copyErr != nil && copyErr != io.EOF {
			return "", fmt.Errorf("failed to extract file: %w", copyErr)
		}

		// Keep track of JSON files as potential SBOM
		if ext == ".json" && sbomPath == "" {
			sbomPath = targetPath
		}
	}

	if sbomPath == "" {
		return "", ErrNoSBOMFound
	}

	return sbomPath, nil
}

func (s *Scanner) extractZip(archiveBytes []byte, destDir string) (string, error) {
	reader := bytes.NewReader(archiveBytes)
	zipReader, err := zip.NewReader(reader, int64(len(archiveBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to create zip reader: %w", err)
	}

	var extractedSize int64
	var sbomPath string

	for _, file := range zipReader.File {
		// Security check: prevent path traversal
		cleanName := filepath.Clean(file.Name)
		if strings.Contains(cleanName, "..") {
			continue
		}

		// Only extract files with allowed extensions
		ext := strings.ToLower(filepath.Ext(cleanName))
		if !strings.Contains(AllowedSBOMExtensions, ext) {
			continue
		}

		// Check extracted size limit
		extractedSize += int64(file.UncompressedSize64)
		if extractedSize > MaxExtractedSize {
			return "", ErrArchiveTooLarge
		}

		targetPath := filepath.Join(destDir, filepath.Base(cleanName))

		// Extract file
		rc, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("failed to open zip entry: %w", err)
		}

		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			_ = rc.Close()
			return "", fmt.Errorf("failed to create file: %w", err)
		}

		_, copyErr := io.CopyN(outFile, rc, int64(file.UncompressedSize64))
		_ = outFile.Close()
		_ = rc.Close()
		if copyErr != nil && copyErr != io.EOF {
			return "", fmt.Errorf("failed to extract file: %w", copyErr)
		}

		if ext == ".json" && sbomPath == "" {
			sbomPath = targetPath
		}
	}

	if sbomPath == "" {
		return "", ErrNoSBOMFound
	}

	return sbomPath, nil
}

func (s *Scanner) validateSBOM(sbomPath string) error {
	// Read file and check it's valid JSON with expected SBOM structure
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		return fmt.Errorf("failed to read SBOM file: %w", err)
	}

	// Basic JSON validation
	var rawJSON map[string]interface{}
	if err := json.Unmarshal(data, &rawJSON); err != nil {
		return fmt.Errorf("invalid JSON in SBOM file: %w", err)
	}

	// Check for known SBOM formats (Syft, CycloneDX, SPDX)
	if _, ok := rawJSON["artifacts"]; ok {
		return nil // Syft format
	}
	if _, ok := rawJSON["bomFormat"]; ok {
		return nil // CycloneDX format
	}
	if _, ok := rawJSON["spdxVersion"]; ok {
		return nil // SPDX format
	}
	if _, ok := rawJSON["packages"]; ok {
		return nil // Could be SPDX or other format
	}

	return fmt.Errorf("unrecognized SBOM format")
}

func (s *Scanner) createScanRecord(agentID, userID string, req types.SBOMUploadRequest) (*core.Record, error) {
	collection, err := s.app.FindCollectionByNameOrId("sbom_scans")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	record.Set("agent_id", agentID)
	record.Set("user_id", userID)
	record.Set("scan_type", string(req.ScanType))
	record.Set("source_name", req.SourceName)
	record.Set("source_type", req.SourceType)
	record.Set("status", string(types.SBOMScanStatusScanning))

	if err := s.app.Save(record); err != nil {
		return nil, err
	}

	return record, nil
}

func (s *Scanner) updateScanWithError(record *core.Record, errMsg string) {
	record.Set("status", string(types.SBOMScanStatusFailed))
	record.Set("error_message", errMsg)
	if err := s.app.Save(record); err != nil {
		s.app.Logger().Error("Failed to update scan record with error", "error", err)
	}
}

func (s *Scanner) runGrypeScan(sbomPath string) (*types.GrypeOutput, error) {
	// Compute checksum of SBOM for logging
	data, _ := os.ReadFile(sbomPath)
	checksum := sha256.Sum256(data)
	s.app.Logger().Info("Running grype scan",
		"sbom_checksum", hex.EncodeToString(checksum[:8]),
		"sbom_size", len(data))

	cmd := exec.Command(s.grypePath, fmt.Sprintf("sbom:%s", sbomPath), "-o", "json")
	cmd.Env = append(os.Environ(), fmt.Sprintf("GRYPE_DB_CACHE_DIR=%s", s.dbCacheDir))

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Grype returns non-zero exit code when vulnerabilities found
			// This is expected behavior, so check if we got valid output
			if len(output) == 0 {
				return nil, fmt.Errorf("grype scan failed: %w: %s", err, string(exitErr.Stderr))
			}
			// Continue with parsing if we have output
		} else {
			return nil, fmt.Errorf("failed to execute grype: %w", err)
		}
	}

	var grypeOutput types.GrypeOutput
	if err := json.Unmarshal(output, &grypeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse grype output: %w", err)
	}

	return &grypeOutput, nil
}

// getStorageDir returns the storage directory for SBOM archives
func (s *Scanner) getStorageDir() string {
	dataDir := s.app.DataDir()
	return filepath.Join(dataDir, "storage", SBOMStorageDir)
}

// storeAndFinalizeScan stores grype output as archive and updates the scan record
func (s *Scanner) storeAndFinalizeScan(scanRecord *core.Record, grypeOutput *types.GrypeOutput) (*types.SBOMScanResult, error) {
	agentID := scanRecord.GetString("agent_id")
	scanID := scanRecord.Id

	// Count severities
	counts := map[types.VulnerabilitySeverity]int{
		types.SeverityCritical:   0,
		types.SeverityHigh:       0,
		types.SeverityMedium:     0,
		types.SeverityLow:        0,
		types.SeverityNegligible: 0,
		types.SeverityUnknown:    0,
	}

	// Count fixable vulnerabilities
	fixableCount := 0
	for _, match := range grypeOutput.Matches {
		severity := types.VulnerabilitySeverity(strings.ToLower(match.Vulnerability.Severity))
		if severity == "" {
			severity = types.SeverityUnknown
		}
		counts[severity]++
		if match.Vulnerability.Fix.State == "fixed" && len(match.Vulnerability.Fix.Versions) > 0 {
			fixableCount++
		}
	}

	// Create storage directory for this agent
	agentStorageDir := filepath.Join(s.getStorageDir(), agentID)
	if err := os.MkdirAll(agentStorageDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Marshal grype output to JSON
	grypeJSON, err := json.Marshal(grypeOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal grype output: %w", err)
	}

	// Create tar.gz archive with grype output
	archivePath := filepath.Join(agentStorageDir, fmt.Sprintf("%s.tar.gz", scanID))
	if err := s.createGrypeArchive(archivePath, grypeJSON, scanID); err != nil {
		return nil, fmt.Errorf("failed to create grype archive: %w", err)
	}

	// Update scan record with archive path and counts
	now := time.Now()
	scanRecord.Set("status", string(types.SBOMScanStatusCompleted))
	scanRecord.Set("archive_path", archivePath)
	scanRecord.Set("total_packages", len(grypeOutput.Matches))
	scanRecord.Set("critical_count", counts[types.SeverityCritical])
	scanRecord.Set("high_count", counts[types.SeverityHigh])
	scanRecord.Set("medium_count", counts[types.SeverityMedium])
	scanRecord.Set("low_count", counts[types.SeverityLow])
	scanRecord.Set("negligible_count", counts[types.SeverityNegligible])
	scanRecord.Set("unknown_count", counts[types.SeverityUnknown])
	scanRecord.Set("scanned_at", now)
	scanRecord.Set("grype_version", grypeOutput.Descriptor.Version)
	if grypeOutput.Descriptor.DB.Status.SchemaVersion != "" {
		scanRecord.Set("db_version", grypeOutput.Descriptor.DB.Status.SchemaVersion)
	}

	if err := s.app.Save(scanRecord); err != nil {
		// Clean up archive on failure
		_ = os.Remove(archivePath)
		return nil, fmt.Errorf("failed to update scan record: %w", err)
	}

	s.app.Logger().Info("SBOM scan completed and archived",
		"scan_id", scanID,
		"agent_id", agentID,
		"archive_path", archivePath,
		"vulnerabilities", len(grypeOutput.Matches))

	return &types.SBOMScanResult{
		ScanID:          scanID,
		AgentID:         agentID,
		ScanType:        types.SBOMScanType(scanRecord.GetString("scan_type")),
		SourceName:      scanRecord.GetString("source_name"),
		Status:          types.SBOMScanStatusCompleted,
		TotalPackages:   len(grypeOutput.Matches),
		CriticalCount:   counts[types.SeverityCritical],
		HighCount:       counts[types.SeverityHigh],
		MediumCount:     counts[types.SeverityMedium],
		LowCount:        counts[types.SeverityLow],
		NegligibleCount: counts[types.SeverityNegligible],
		UnknownCount:    counts[types.SeverityUnknown],
		FixableCount:    fixableCount,
		ScannedAt:       &now,
		GrypeVersion:    grypeOutput.Descriptor.Version,
		DBVersion:       grypeOutput.Descriptor.DB.Status.SchemaVersion,
	}, nil
}

// createGrypeArchive creates a tar.gz archive containing the grype output
func (s *Scanner) createGrypeArchive(archivePath string, grypeJSON []byte, scanID string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	gzWriter := gzip.NewWriter(file)
	defer func() { _ = gzWriter.Close() }()

	tarWriter := tar.NewWriter(gzWriter)
	defer func() { _ = tarWriter.Close() }()

	// Add grype output JSON to archive
	header := &tar.Header{
		Name:    fmt.Sprintf("%s-grype-output.json", scanID),
		Size:    int64(len(grypeJSON)),
		Mode:    0600,
		ModTime: time.Now(),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tarWriter.Write(grypeJSON); err != nil {
		return err
	}

	return nil
}

// getScansPerAgent returns the configured retention limit from sbom_settings
func (s *Scanner) getScansPerAgent() int {
	record, err := s.app.FindFirstRecordByData("sbom_settings", "key", "scans_per_agent")
	if err != nil {
		return DefaultScansPerAgent
	}
	value := record.GetString("value")
	if value == "" {
		return DefaultScansPerAgent
	}
	var limit int
	if _, err := fmt.Sscanf(value, "%d", &limit); err != nil || limit <= 0 {
		return DefaultScansPerAgent
	}
	return limit
}

// enforceRetentionLimit deletes old scans exceeding the per-agent limit
func (s *Scanner) enforceRetentionLimit(agentID string) {
	limit := s.getScansPerAgent()

	collection, err := s.app.FindCollectionByNameOrId("sbom_scans")
	if err != nil {
		s.app.Logger().Error("Failed to find sbom_scans collection for retention", "error", err)
		return
	}

	// Get all completed scans for this agent, ordered by scanned_at descending
	records, err := s.app.FindRecordsByFilter(
		collection,
		"agent_id = {:agentId} && status = 'completed'",
		"-scanned_at",
		0, // No limit, get all
		0,
		map[string]any{"agentId": agentID},
	)
	if err != nil {
		s.app.Logger().Error("Failed to query scans for retention", "error", err, "agent_id", agentID)
		return
	}

	// Delete scans beyond the limit
	if len(records) > limit {
		for i := limit; i < len(records); i++ {
			record := records[i]
			archivePath := record.GetString("archive_path")

			// Delete archive file
			if archivePath != "" {
				if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
					s.app.Logger().Warn("Failed to delete scan archive",
						"path", archivePath,
						"error", err)
				}
			}

			// Delete scan record
			if err := s.app.Delete(record); err != nil {
				s.app.Logger().Error("Failed to delete old scan record",
					"scan_id", record.Id,
					"error", err)
			} else {
				s.app.Logger().Info("Deleted old scan due to retention policy",
					"scan_id", record.Id,
					"agent_id", agentID)
			}
		}
	}
}

// LoadVulnerabilitiesFromArchive loads vulnerabilities from a stored archive on-demand
func (s *Scanner) LoadVulnerabilitiesFromArchive(archivePath string) (*types.GrypeOutput, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() { _ = file.Close() }()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Look for JSON file
		if strings.HasSuffix(header.Name, ".json") {
			content, err := io.ReadAll(io.LimitReader(tarReader, MaxExtractedSize))
			if err != nil {
				return nil, fmt.Errorf("failed to read archive content: %w", err)
			}

			var grypeOutput types.GrypeOutput
			if err := json.Unmarshal(content, &grypeOutput); err != nil {
				return nil, fmt.Errorf("failed to parse grype output: %w", err)
			}
			return &grypeOutput, nil
		}
	}

	return nil, ErrNoSBOMFound
}

// GetSyftExcludePatterns returns the syft exclude patterns for an agent
func (s *Scanner) GetSyftExcludePatterns(agentID string) ([]string, error) {
	// First try to get agent-specific patterns
	agentRecord, err := s.app.FindRecordById("agents", agentID)
	if err == nil {
		patterns := agentRecord.Get("syft_exclude_patterns")
		if patterns != nil {
			// Parse JSON array
			var excludePatterns []string
			switch v := patterns.(type) {
			case []string:
				if len(v) > 0 {
					return v, nil
				}
			case []interface{}:
				if len(v) > 0 {
					for _, p := range v {
						if s, ok := p.(string); ok {
							excludePatterns = append(excludePatterns, s)
						}
					}
					if len(excludePatterns) > 0 {
						return excludePatterns, nil
					}
				}
			case string:
				if v != "" && v != "null" && v != "[]" {
					if err := json.Unmarshal([]byte(v), &excludePatterns); err == nil && len(excludePatterns) > 0 {
						return excludePatterns, nil
					}
				}
			}
		}
	}

	// Fall back to default patterns from settings
	settingRecord, err := s.app.FindFirstRecordByData("sbom_settings", "key", "default_syft_exclude_patterns")
	if err != nil {
		// Return hard-coded defaults if no setting found
		return []string{
			"**/proc/**",
			"**/sys/**",
			"**/dev/**",
			"**/run/**",
			"**/tmp/**",
			"**/var/cache/**",
			"**/var/log/**",
			"**/home/*/.cache/**",
		}, nil
	}

	var defaultPatterns []string
	if err := json.Unmarshal([]byte(settingRecord.GetString("value")), &defaultPatterns); err != nil {
		return nil, fmt.Errorf("failed to parse default exclude patterns: %w", err)
	}

	return defaultPatterns, nil
}

func (s *Scanner) updateVulnerabilitySummary(agentID, userID string, result *types.SBOMScanResult) {
	collection, err := s.app.FindCollectionByNameOrId("vulnerability_summary")
	if err != nil {
		s.app.Logger().Error("Failed to find vulnerability_summary collection", "error", err)
		return
	}

	// Try to find existing summary for this agent
	records, err := s.app.FindRecordsByFilter(collection, "agent_id = {:agentId}", "", 1, 0, map[string]any{"agentId": agentID})

	var summaryRecord *core.Record
	if err == nil && len(records) > 0 {
		summaryRecord = records[0]
		// Update existing record
		summaryRecord.Set("total_scans", summaryRecord.GetInt("total_scans")+1)
	} else {
		// Create new record
		summaryRecord = core.NewRecord(collection)
		summaryRecord.Set("agent_id", agentID)
		summaryRecord.Set("user_id", userID)
		summaryRecord.Set("total_scans", 1)
	}

	summaryRecord.Set("total_vulnerabilities", result.CriticalCount+result.HighCount+result.MediumCount+result.LowCount+result.NegligibleCount+result.UnknownCount)
	summaryRecord.Set("critical_count", result.CriticalCount)
	summaryRecord.Set("high_count", result.HighCount)
	summaryRecord.Set("medium_count", result.MediumCount)
	summaryRecord.Set("low_count", result.LowCount)
	summaryRecord.Set("fixable_count", result.FixableCount)
	summaryRecord.Set("last_scan_at", result.ScannedAt)
	summaryRecord.Set("last_scan_id", result.ScanID)

	if err := s.app.Save(summaryRecord); err != nil {
		s.app.Logger().Error("Failed to save vulnerability summary", "error", err)
	}
}
