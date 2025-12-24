package backup

import (
	"context"
	"log"
	"sync"
	"time"
)

// SchedulerConfig holds backup scheduler configuration.
type SchedulerConfig struct {
	// Interval is the time between backups.
	Interval time.Duration

	// RetentionCount is the number of backups to keep (0 = keep all).
	RetentionCount int

	// RetentionDays is the number of days to keep backups (0 = keep all).
	RetentionDays int

	// OnBackupComplete is called when a backup completes successfully.
	OnBackupComplete func(filename string)

	// OnBackupError is called when a backup fails.
	OnBackupError func(error)
}

// Scheduler manages automated configuration backups.
type Scheduler struct {
	manager *Manager
	config  SchedulerConfig

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc

	// Data providers
	configProvider  func() interface{}
	usersProvider   func() interface{}
	devicesProvider func() interface{}
}

// NewScheduler creates a new backup scheduler.
func NewScheduler(manager *Manager, config SchedulerConfig) *Scheduler {
	if config.Interval == 0 {
		config.Interval = 24 * time.Hour // Default: daily backups
	}

	return &Scheduler{
		manager: manager,
		config:  config,
	}
}

// SetProviders sets the data providers for backups.
func (s *Scheduler) SetProviders(config, users, devices func() interface{}) {
	s.configProvider = config
	s.usersProvider = users
	s.devicesProvider = devices
}

// Start starts the backup scheduler.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true

	go s.run(ctx)

	log.Printf("[Backup] Scheduler started (interval: %v)", s.config.Interval)
}

// Stop stops the backup scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	if s.cancel != nil {
		s.cancel()
	}

	s.running = false
	log.Println("[Backup] Scheduler stopped")
}

// run executes the backup schedule.
func (s *Scheduler) run(ctx context.Context) {
	// Create initial backup
	s.performBackup()

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.performBackup()
		}
	}
}

// performBackup creates a backup and manages retention.
func (s *Scheduler) performBackup() {
	// Get current data
	var config, users, devices interface{}

	if s.configProvider != nil {
		config = s.configProvider()
	}
	if s.usersProvider != nil {
		users = s.usersProvider()
	}
	if s.devicesProvider != nil {
		devices = s.devicesProvider()
	}

	// Create backup
	description := "Automated backup at " + time.Now().Format(time.RFC3339)
	filename, err := s.manager.Create(config, users, devices, description)

	if err != nil {
		log.Printf("[Backup] Failed to create backup: %v", err)
		if s.config.OnBackupError != nil {
			s.config.OnBackupError(err)
		}
		return
	}

	log.Printf("[Backup] Created backup: %s", filename)

	if s.config.OnBackupComplete != nil {
		s.config.OnBackupComplete(filename)
	}

	// Clean up old backups
	s.cleanup()
}

// cleanup removes old backups based on retention policy.
func (s *Scheduler) cleanup() {
	backups, err := s.manager.List()
	if err != nil {
		log.Printf("[Backup] Failed to list backups for cleanup: %v", err)
		return
	}

	if len(backups) == 0 {
		return
	}

	// Remove backups based on retention count
	if s.config.RetentionCount > 0 && len(backups) > s.config.RetentionCount {
		// Sort by creation time (oldest first)
		// Backups are already sorted by filename which includes timestamp
		toDelete := len(backups) - s.config.RetentionCount

		for i := 0; i < toDelete; i++ {
			if err := s.manager.Delete(backups[i].Filename); err != nil {
				log.Printf("[Backup] Failed to delete old backup %s: %v", backups[i].Filename, err)
			} else {
				log.Printf("[Backup] Deleted old backup: %s", backups[i].Filename)
			}
		}
	}

	// Remove backups older than retention days
	if s.config.RetentionDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)

		for _, backup := range backups {
			if backup.Created.Before(cutoff) {
				if err := s.manager.Delete(backup.Filename); err != nil {
					log.Printf("[Backup] Failed to delete expired backup %s: %v", backup.Filename, err)
				} else {
					log.Printf("[Backup] Deleted expired backup: %s", backup.Filename)
				}
			}
		}
	}
}

// TriggerBackup manually triggers a backup.
func (s *Scheduler) TriggerBackup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current data
	var config, users, devices interface{}

	if s.configProvider != nil {
		config = s.configProvider()
	}
	if s.usersProvider != nil {
		users = s.usersProvider()
	}
	if s.devicesProvider != nil {
		devices = s.devicesProvider()
	}

	// Create backup
	description := "Manual backup at " + time.Now().Format(time.RFC3339)
	filename, err := s.manager.Create(config, users, devices, description)

	if err != nil {
		return err
	}

	log.Printf("[Backup] Created manual backup: %s", filename)

	if s.config.OnBackupComplete != nil {
		s.config.OnBackupComplete(filename)
	}

	return nil
}

// IsRunning returns true if the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
