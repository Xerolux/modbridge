package ha

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// LeaderElector manages leader election using etcd.
type LeaderElector struct {
	client   *clientv3.Client
	session  *concurrency.Session
	election *concurrency.Election
	nodeID   string
	prefix   string

	isLeader bool
	cancel   context.CancelFunc

	// Callbacks
	onBecomeLeader  func()
	onLoseLeadership func()
}

// Config holds leader election configuration.
type Config struct {
	// Endpoints is the list of etcd endpoints.
	Endpoints []string

	// NodeID is the unique identifier for this node.
	NodeID string

	// ElectionPrefix is the etcd key prefix for elections.
	ElectionPrefix string

	// SessionTTL is the session time-to-live in seconds.
	SessionTTL int

	// OnBecomeLeader is called when this node becomes the leader.
	OnBecomeLeader func()

	// OnLoseLeadership is called when this node loses leadership.
	OnLoseLeadership func()
}

// NewLeaderElector creates a new leader elector.
func NewLeaderElector(config Config) (*LeaderElector, error) {
	if len(config.Endpoints) == 0 {
		return nil, errors.New("no etcd endpoints provided")
	}

	if config.NodeID == "" {
		return nil, errors.New("node ID is required")
	}

	if config.ElectionPrefix == "" {
		config.ElectionPrefix = "/modbus-proxy/election"
	}

	if config.SessionTTL == 0 {
		config.SessionTTL = 10
	}

	// Create etcd client
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &LeaderElector{
		client:           client,
		nodeID:           config.NodeID,
		prefix:           config.ElectionPrefix,
		onBecomeLeader:   config.OnBecomeLeader,
		onLoseLeadership: config.OnLoseLeadership,
	}, nil
}

// Start starts the leader election process.
func (le *LeaderElector) Start(ctx context.Context) error {
	// Create a session
	session, err := concurrency.NewSession(le.client, concurrency.WithTTL(10))
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	le.session = session

	// Create an election
	le.election = concurrency.NewElection(session, le.prefix)

	// Start campaign in a goroutine
	campaignCtx, cancel := context.WithCancel(ctx)
	le.cancel = cancel

	go le.campaign(campaignCtx)

	return nil
}

// campaign runs the election campaign.
func (le *LeaderElector) campaign(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Campaign to become leader
		log.Printf("[HA] Node %s campaigning for leadership...", le.nodeID)
		if err := le.election.Campaign(ctx, le.nodeID); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("[HA] Campaign failed: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// We are now the leader
		le.isLeader = true
		log.Printf("[HA] Node %s became the leader", le.nodeID)

		if le.onBecomeLeader != nil {
			le.onBecomeLeader()
		}

		// Observe leadership (blocks until we lose it)
		select {
		case <-ctx.Done():
			return
		case <-le.session.Done():
			// Session expired, we lost leadership
			le.isLeader = false
			log.Printf("[HA] Node %s lost leadership (session expired)", le.nodeID)

			if le.onLoseLeadership != nil {
				le.onLoseLeadership()
			}

			// Create a new session and try again
			session, err := concurrency.NewSession(le.client, concurrency.WithTTL(10))
			if err != nil {
				log.Printf("[HA] Failed to create new session: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			le.session = session
			le.election = concurrency.NewElection(session, le.prefix)
		}
	}
}

// IsLeader returns true if this node is the current leader.
func (le *LeaderElector) IsLeader() bool {
	return le.isLeader
}

// GetLeader returns the current leader's node ID.
func (le *LeaderElector) GetLeader(ctx context.Context) (string, error) {
	resp, err := le.election.Leader(ctx)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", errors.New("no leader elected")
	}

	return string(resp.Kvs[0].Value), nil
}

// Resign resigns from leadership if this node is the leader.
func (le *LeaderElector) Resign(ctx context.Context) error {
	if !le.isLeader {
		return nil
	}

	if err := le.election.Resign(ctx); err != nil {
		return err
	}

	le.isLeader = false
	return nil
}

// Stop stops the leader election process.
func (le *LeaderElector) Stop() error {
	if le.cancel != nil {
		le.cancel()
	}

	if le.session != nil {
		if err := le.session.Close(); err != nil {
			log.Printf("[HA] Error closing session: %v", err)
		}
	}

	if le.client != nil {
		if err := le.client.Close(); err != nil {
			log.Printf("[HA] Error closing etcd client: %v", err)
		}
	}

	return nil
}
