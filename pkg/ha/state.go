package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// StateManager manages shared state using etcd.
type StateManager struct {
	client *clientv3.Client
	prefix string
}

// NewStateManager creates a new state manager.
func NewStateManager(endpoints []string, prefix string) (*StateManager, error) {
	if prefix == "" {
		prefix = "/modbus-proxy/state"
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &StateManager{
		client: client,
		prefix: prefix,
	}, nil
}

// Put stores a value in the shared state.
func (sm *StateManager) Put(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	fullKey := sm.prefix + "/" + key
	_, err = sm.client.Put(ctx, fullKey, string(data))
	if err != nil {
		return fmt.Errorf("failed to put value: %w", err)
	}

	return nil
}

// Get retrieves a value from the shared state.
func (sm *StateManager) Get(ctx context.Context, key string, value interface{}) error {
	fullKey := sm.prefix + "/" + key
	resp, err := sm.client.Get(ctx, fullKey)
	if err != nil {
		return fmt.Errorf("failed to get value: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("key not found: %s", key)
	}

	if err := json.Unmarshal(resp.Kvs[0].Value, value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Delete deletes a value from the shared state.
func (sm *StateManager) Delete(ctx context.Context, key string) error {
	fullKey := sm.prefix + "/" + key
	_, err := sm.client.Delete(ctx, fullKey)
	if err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}

	return nil
}

// List lists all keys with the given prefix.
func (sm *StateManager) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := sm.prefix + "/" + prefix
	resp, err := sm.client.Get(ctx, fullPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	keys := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		keys = append(keys, string(kv.Key))
	}

	return keys, nil
}

// Watch watches for changes to a key.
func (sm *StateManager) Watch(ctx context.Context, key string) clientv3.WatchChan {
	fullKey := sm.prefix + "/" + key
	return sm.client.Watch(ctx, fullKey)
}

// Close closes the state manager.
func (sm *StateManager) Close() error {
	return sm.client.Close()
}
