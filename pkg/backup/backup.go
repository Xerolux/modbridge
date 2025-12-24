package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Backup represents a configuration backup.
type Backup struct {
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Config      map[string]interface{} `json:"config"`
	Users       []interface{}          `json:"users,omitempty"`
	Devices     []interface{}          `json:"devices,omitempty"`
	Description string                 `json:"description"`
}

// Manager handles configuration backups and restores.
type Manager struct {
	backupDir string
}

// NewManager creates a new backup manager.
func NewManager(backupDir string) (*Manager, error) {
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, err
	}

	return &Manager{
		backupDir: backupDir,
	}, nil
}

// Create creates a new backup.
func (m *Manager) Create(config, users, devices interface{}, description string) (string, error) {
	backup := Backup{
		Timestamp:   time.Now(),
		Version:     "1.0.0",
		Description: description,
	}

	// Convert config to map
	configData, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := json.Unmarshal(configData, &backup.Config); err != nil {
		return "", fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Convert users if provided
	if users != nil {
		usersData, err := json.Marshal(users)
		if err != nil {
			return "", fmt.Errorf("failed to marshal users: %w", err)
		}
		if err := json.Unmarshal(usersData, &backup.Users); err != nil {
			return "", fmt.Errorf("failed to unmarshal users: %w", err)
		}
	}

	// Convert devices if provided
	if devices != nil {
		devicesData, err := json.Marshal(devices)
		if err != nil {
			return "", fmt.Errorf("failed to marshal devices: %w", err)
		}
		if err := json.Unmarshal(devicesData, &backup.Devices); err != nil {
			return "", fmt.Errorf("failed to unmarshal devices: %w", err)
		}
	}

	// Generate backup filename
	filename := fmt.Sprintf("backup_%s.tar.gz", backup.Timestamp.Format("20060102_150405"))
	filepath := filepath.Join(m.backupDir, filename)

	// Create backup archive
	if err := m.createArchive(filepath, &backup); err != nil {
		return "", fmt.Errorf("failed to create archive: %w", err)
	}

	return filename, nil
}

// Restore restores a configuration from a backup.
func (m *Manager) Restore(filename string) (*Backup, error) {
	filepath := filepath.Join(m.backupDir, filename)

	// Extract and parse backup
	backup, err := m.extractArchive(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	return backup, nil
}

// List lists all available backups.
func (m *Manager) List() ([]BackupInfo, error) {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return nil, err
	}

	backups := make([]BackupInfo, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".gz" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Filename: entry.Name(),
			Size:     info.Size(),
			Created:  info.ModTime(),
		})
	}

	return backups, nil
}

// Delete deletes a backup file.
func (m *Manager) Delete(filename string) error {
	filepath := filepath.Join(m.backupDir, filename)
	return os.Remove(filepath)
}

// Export exports a backup file for download.
func (m *Manager) Export(filename string) (string, error) {
	filepath := filepath.Join(m.backupDir, filename)

	// Verify file exists
	if _, err := os.Stat(filepath); err != nil {
		return "", fmt.Errorf("backup file not found: %w", err)
	}

	return filepath, nil
}

// Import imports a backup file from upload.
func (m *Manager) Import(sourceFile string, filename string) error {
	destPath := filepath.Join(m.backupDir, filename)

	// Copy file to backup directory
	source, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// createArchive creates a tar.gz archive with the backup data.
func (m *Manager) createArchive(filepath string, backup *Backup) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Marshal backup to JSON
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return err
	}

	// Create tar header
	header := &tar.Header{
		Name:    "backup.json",
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: backup.Timestamp,
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tarWriter.Write(data); err != nil {
		return err
	}

	return nil
}

// extractArchive extracts a backup from a tar.gz archive.
func (m *Manager) extractArchive(filepath string) (*Backup, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Read first file in archive
	_, err = tarReader.Next()
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(tarReader)
	if err != nil {
		return nil, err
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, err
	}

	return &backup, nil
}

// BackupInfo contains metadata about a backup file.
type BackupInfo struct {
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	Created  time.Time `json:"created"`
}
