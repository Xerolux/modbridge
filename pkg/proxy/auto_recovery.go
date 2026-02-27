package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// RecoveryManager manages automatic recovery from failures
type RecoveryManager struct {
	mu              sync.RWMutex
	recoveryTasks   map[string]*RecoveryTask
	maxConcurrent   int
	currentTasks    int
	config          RecoveryConfig
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         bool
	onRecovery      func(string) error
}

// RecoveryTask represents a recovery operation
type RecoveryTask struct {
	ID          string
	Target      string
	Priority    int
	StartedAt   time.Time
	CompletedAt time.Time
	Status      RecoveryStatus
	Error       error
	Attempts    int
	MaxAttempts int
}

// RecoveryStatus represents the status of a recovery task
type RecoveryStatus string

const (
	StatusPending   RecoveryStatus = "pending"
	StatusRunning   RecoveryStatus = "running"
	StatusCompleted RecoveryStatus = "completed"
	StatusFailed    RecoveryStatus = "failed"
	StatusCancelled RecoveryStatus = "cancelled"
)

// RecoveryConfig holds configuration for recovery manager
type RecoveryConfig struct {
	MaxConcurrent   int           // Maximum concurrent recovery tasks (default: 5)
	RetryInterval   time.Duration // Time between retries (default: 30s)
	MaxAttempts     int           // Maximum recovery attempts (default: 3)
	TaskTimeout     time.Duration // Timeout for each recovery task (default: 60s)
}

// DefaultRecoveryConfig returns sensible defaults
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		MaxConcurrent: 5,
		RetryInterval: 30 * time.Second,
		MaxAttempts:   3,
		TaskTimeout:   60 * time.Second,
	}
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(config RecoveryConfig, onRecovery func(string) error) *RecoveryManager {
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 5
	}
	if config.RetryInterval <= 0 {
		config.RetryInterval = 30 * time.Second
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.TaskTimeout <= 0 {
		config.TaskTimeout = 60 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	rm := &RecoveryManager{
		recoveryTasks: make(map[string]*RecoveryTask),
		maxConcurrent: config.MaxConcurrent,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		running:       true,
		onRecovery:    onRecovery,
	}

	// Start task processor
	rm.wg.Add(1)
	go rm.taskProcessor()

	return rm
}

// AddTask adds a recovery task
func (rm *RecoveryManager) AddTask(target string, priority int) (string, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	taskID := fmt.Sprintf("recovery_%s_%d", target, time.Now().UnixNano())

	task := &RecoveryTask{
		ID:          taskID,
		Target:      target,
		Priority:    priority,
		StartedAt:   time.Now(),
		Status:      StatusPending,
		MaxAttempts: rm.config.MaxAttempts,
	}

	rm.recoveryTasks[taskID] = task

	return taskID, nil
}

// taskProcessor processes recovery tasks
func (rm *RecoveryManager) taskProcessor() {
	defer rm.wg.Done()

	for {
		select {
		case <-rm.ctx.Done():
			return
		default:
		}

		task := rm.getNextTask()
		if task == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		rm.wg.Add(1)
		go rm.executeTask(task)
	}
}

// getNextTask returns the next task to execute
func (rm *RecoveryManager) getNextTask() *RecoveryTask {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Check if we can start a new task
	if rm.currentTasks >= rm.maxConcurrent {
		return nil
	}

	// Find highest priority pending task
	var selected *RecoveryTask
	highestPriority := -1

	for _, task := range rm.recoveryTasks {
		if task.Status == StatusPending && task.Priority > highestPriority {
			selected = task
			highestPriority = task.Priority
		}
	}

	if selected != nil {
		selected.Status = StatusRunning
		rm.currentTasks++
	}

	return selected
}

// executeTask executes a recovery task
func (rm *RecoveryManager) executeTask(task *RecoveryTask) {
	defer rm.wg.Done()
	defer func() {
		rm.mu.Lock()
		rm.currentTasks--
		rm.mu.Unlock()
	}()

	for task.Attempts < task.MaxAttempts {
		select {
		case <-rm.ctx.Done():
			task.Status = StatusCancelled
			return
		default:
		}

		task.Attempts++

		// Create timeout context
		ctx, cancel := context.WithTimeout(rm.ctx, rm.config.TaskTimeout)

		// Execute recovery
		err := rm.attemptRecovery(ctx, task)

		cancel()

		if err == nil {
			task.Status = StatusCompleted
			task.CompletedAt = time.Now()
			return
		}

		task.Error = err

		// Wait before retry
		if task.Attempts < task.MaxAttempts {
			select {
			case <-rm.ctx.Done():
				task.Status = StatusCancelled
				return
			case <-time.After(rm.config.RetryInterval):
				// Continue to next attempt
			}
		}
	}

	task.Status = StatusFailed
	task.CompletedAt = time.Now()
}

// attemptRecovery attempts to recover a target
func (rm *RecoveryManager) attemptRecovery(ctx context.Context, task *RecoveryTask) error {
	// Try to connect to target
	conn, err := net.DialTimeout("tcp", task.Target, 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	// If we have a custom recovery function, use it
	if rm.onRecovery != nil {
		if err := rm.onRecovery(task.Target); err != nil {
			return fmt.Errorf("recovery function failed: %w", err)
		}
	}

	return nil
}

// GetTask returns a recovery task
func (rm *RecoveryManager) GetTask(taskID string) (*RecoveryTask, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	task, exists := rm.recoveryTasks[taskID]
	return task, exists
}

// GetStats returns recovery statistics
func (rm *RecoveryManager) GetStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_tasks":     len(rm.recoveryTasks),
		"current_tasks":   rm.currentTasks,
		"max_concurrent":  rm.maxConcurrent,
		"running":         rm.running,
	}

	pending := 0
	running := 0
	completed := 0
	failed := 0
	cancelled := 0

	for _, task := range rm.recoveryTasks {
		switch task.Status {
		case StatusPending:
			pending++
		case StatusRunning:
			running++
		case StatusCompleted:
			completed++
		case StatusFailed:
			failed++
		case StatusCancelled:
			cancelled++
		}
	}

	stats["pending"] = pending
	stats["running"] = running
	stats["completed"] = completed
	stats["failed"] = failed
	stats["cancelled"] = cancelled
	stats["success_rate"] = float64(completed) / float64(len(rm.recoveryTasks)) * 100

	return stats
}

// Stop stops the recovery manager
func (rm *RecoveryManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return
	}

	rm.running = false
	rm.cancel()
	rm.wg.Wait()
}

// CancelTask cancels a recovery task
func (rm *RecoveryManager) CancelTask(taskID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	task, exists := rm.recoveryTasks[taskID]
	if !exists {
		return fmt.Errorf("task not found")
	}

	if task.Status == StatusPending || task.Status == StatusRunning {
		task.Status = StatusCancelled
		task.CompletedAt = time.Now()
		return nil
	}

	return fmt.Errorf("cannot cancel task in status: %s", task.Status)
}

// ClearCompleted clears completed tasks
func (rm *RecoveryManager) ClearCompleted() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for id, task := range rm.recoveryTasks {
		if task.Status == StatusCompleted || task.Status == StatusFailed {
			delete(rm.recoveryTasks, id)
		}
	}
}
