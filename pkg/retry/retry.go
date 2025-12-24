package retry

import (
	"context"
	"errors"
	"math"
	"time"
)

// Config holds retry configuration.
type Config struct {
	// MaxAttempts is the maximum number of retry attempts (0 = no retries).
	MaxAttempts int

	// InitialDelay is the delay before the first retry.
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases (exponential backoff).
	Multiplier float64

	// RetryIf determines if an error should trigger a retry (optional).
	RetryIf func(error) bool
}

// DefaultConfig returns a default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}
}

// Do executes the given function with retries on failure.
func Do(ctx context.Context, config Config, fn func() error) error {
	if config.MaxAttempts == 0 {
		return fn()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if config.RetryIf != nil && !config.RetryIf(err) {
			return err
		}

		// No more attempts left
		if attempt == config.MaxAttempts {
			break
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Calculate next delay (exponential backoff)
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return lastErr
}

// DoWithData executes a function that returns data with retries on failure.
func DoWithData[T any](ctx context.Context, config Config, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		data, err := fn()
		if err == nil {
			return data, nil
		}

		lastErr = err

		// Check if we should retry
		if config.RetryIf != nil && !config.RetryIf(err) {
			return result, err
		}

		// No more attempts left
		if attempt == config.MaxAttempts {
			break
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}

		// Calculate next delay (exponential backoff)
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return result, lastErr
}

// Backoff calculates the backoff delay for a given attempt.
func Backoff(attempt int, initialDelay, maxDelay time.Duration, multiplier float64) time.Duration {
	delay := float64(initialDelay) * math.Pow(multiplier, float64(attempt))
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}
	return time.Duration(delay)
}

// IsRetryable returns true if the error is retryable.
func IsRetryable(err error) bool {
	// Temporary errors are retryable
	var temporary interface{ Temporary() bool }
	if errors.As(err, &temporary) && temporary.Temporary() {
		return true
	}

	// Timeout errors are retryable
	var timeout interface{ Timeout() bool }
	if errors.As(err, &timeout) && timeout.Timeout() {
		return true
	}

	return false
}

// RetryableError wraps an error to mark it as retryable.
type RetryableError struct {
	Err error
}

func (e RetryableError) Error() string {
	return e.Err.Error()
}

func (e RetryableError) Unwrap() error {
	return e.Err
}

func (e RetryableError) Temporary() bool {
	return true
}

// Retryable wraps an error to mark it as retryable.
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return RetryableError{Err: err}
}
