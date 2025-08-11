package featurechaos

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "gitlab.com/devpro_studio/featurechaos-sdk/pb"
)

// FeatureConfig represents one feature configuration cached in SDK
type FeatureConfig struct {
	Name       string
	AllPercent int
	Keys       map[string]KeyConfig // key name -> config
}

// KeyConfig represents percent configuration for a specific key
type KeyConfig struct {
	AllPercent int
	Items      map[string]int // param value -> percent
}

// UpdateEvent is emitted when a feature changes
type UpdateEvent struct {
	Version  int64
	Features []FeatureConfig
}

// Options configures the Client
type Options struct {
	// If true, call to IsEnabled will enqueue a stat event when result is true
	AutoSendStats bool
	// Optional callback fired when updates are received
	OnUpdate func(UpdateEvent)
	// Dial timeout for gRPC connections
	DialTimeout time.Duration
	// Initial last known version; usually 0
	InitialVersion int64
}

// Client is a FeatureChaos SDK instance
type Client struct {
	mu          sync.RWMutex
	conn        *grpc.ClientConn
	client      pb.FeatureServiceClient
	serviceName string
	lastVersion int64

	// in-memory cache: feature name -> config
	features map[string]FeatureConfig

	// stats pipeline
	statsCh    chan string
	autoStats  bool
	onUpdate   func(UpdateEvent)
	cancelRoot context.CancelFunc
	wg         sync.WaitGroup
}

// New creates and starts a Client. Address must be host:port of the gRPC server.
func New(ctx context.Context, address string, serviceName string, opts Options) (*Client, error) {
	if serviceName == "" {
		return nil, errors.New("serviceName is required")
	}
	if _, _, err := net.SplitHostPort(address); err != nil {
		return nil, errors.New("address must be in host:port format")
	}
	dialTimeout := opts.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}

	dctx, cancel := context.WithTimeout(ctx, dialTimeout)
	conn, err := grpc.DialContext(
		dctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	cancel()
	if err != nil {
		return nil, err
	}

	c := &Client{
		conn:        conn,
		client:      pb.NewFeatureServiceClient(conn),
		serviceName: serviceName,
		lastVersion: opts.InitialVersion,
		features:    make(map[string]FeatureConfig),
		statsCh:     make(chan string, 1024),
		autoStats:   opts.AutoSendStats,
		onUpdate:    opts.OnUpdate,
	}

	rootCtx, cancelRoot := context.WithCancel(context.Background())
	c.cancelRoot = cancelRoot

	// start subscriber and stats workers
	c.wg.Add(2)
	go func() {
		defer c.wg.Done()
		c.runSubscriber(rootCtx)
	}()
	go func() {
		defer c.wg.Done()
		c.runStats(rootCtx)
	}()

	return c, nil
}

// Close stops background workers and closes the connection
func (c *Client) Close() error {
	if c.cancelRoot != nil {
		c.cancelRoot()
	}
	c.wg.Wait()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsEnabled evaluates a feature for a given seed and attributes.
// seed must be a stable identifier for the subject (e.g., userID) to ensure stickiness.
func (c *Client) IsEnabled(featureName string, seed string, attrs map[string]string) bool {
	cfg, ok := c.getFeature(featureName)
	if !ok {
		return false
	}

	// Priority: exact value match -> key-level percent -> feature-level percent
	percent := -1
	if attrs != nil {
		// Single pass over attrs: stop on first exact match, otherwise remember first key-level percent
		keyLevel := -1
		for key, val := range attrs {
			if keyCfg, ok := cfg.Keys[key]; ok {
				if p, ok2 := keyCfg.Items[val]; ok2 {
					percent = p
					break
				}
				if keyLevel < 0 {
					keyLevel = keyCfg.AllPercent
				}
			}
		}
		if percent < 0 && keyLevel >= 0 {
			percent = keyLevel
		}
	}
	if percent < 0 {
		percent = cfg.AllPercent
	}
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}

	enabled := fastBucketHit(featureName, seed, percent)
	if enabled && c.autoStats {
		c.Track(featureName)
	}
	return enabled
}

// Track enqueues a usage event for given feature
func (c *Client) Track(featureName string) {
	select {
	case c.statsCh <- featureName:
	default:
		// drop if buffer full
	}
}

// GetSnapshot returns a copy of current features map
func (c *Client) GetSnapshot() map[string]FeatureConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]FeatureConfig, len(c.features))
	for k, v := range c.features {
		// deep copy maps to keep isolation
		keysCopy := make(map[string]KeyConfig, len(v.Keys))
		for kk, kv := range v.Keys {
			itemsCopy := make(map[string]int, len(kv.Items))
			for ik, iv := range kv.Items {
				itemsCopy[ik] = iv
			}
			keysCopy[kk] = KeyConfig{AllPercent: kv.AllPercent, Items: itemsCopy}
		}
		out[k] = FeatureConfig{Name: v.Name, AllPercent: v.AllPercent, Keys: keysCopy}
	}
	return out
}

// internal helpers

func (c *Client) getFeature(name string) (FeatureConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.features[name]
	return v, ok
}

func (c *Client) runSubscriber(ctx context.Context) {
	// persistent reconnect loop
	op := backoff.NewExponentialBackOff()
	op.InitialInterval = 500 * time.Millisecond
	op.MaxInterval = 10 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		stream, err := c.client.Subscribe(ctx, &pb.GetAllFeatureRequest{
			ServiceName: c.serviceName,
			LastVersion: c.lastVersion,
		})
		if err != nil {
			sleep := op.NextBackOff()
			if sleep == backoff.Stop {
				sleep = op.MaxInterval
			}
			time.Sleep(sleep)
			continue
		}
		// reset backoff on successful connect
		op.Reset()

		for {
			resp, err := stream.Recv()
			if err != nil {
				// reconnect on any error
				break
			}
			if resp == nil {
				continue
			}
			c.applyUpdate(resp)
		}
		// loop and reconnect
	}
}

func (c *Client) applyUpdate(resp *pb.GetFeatureResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, f := range resp.GetFeatures() {
		cfg := FeatureConfig{
			Name:       f.GetName(),
			AllPercent: int(f.GetAll()),
			Keys:       make(map[string]KeyConfig, len(f.GetProps())),
		}
		for _, p := range f.GetProps() {
			items := make(map[string]int, len(p.GetItem()))
			for k, v := range p.GetItem() {
				items[k] = int(v)
			}
			cfg.Keys[p.GetName()] = KeyConfig{AllPercent: int(p.GetAll()), Items: items}
		}
		c.features[cfg.Name] = cfg
	}
	if resp.GetVersion() > c.lastVersion {
		c.lastVersion = resp.GetVersion()
	}
	if c.onUpdate != nil {
		// prepare shallow copy for callback
		changed := make([]FeatureConfig, 0, len(resp.GetFeatures()))
		for _, f := range resp.GetFeatures() {
			if v, ok := c.features[f.GetName()]; ok {
				changed = append(changed, v)
			}
		}
		go c.onUpdate(UpdateEvent{Version: c.lastVersion, Features: changed})
	}
}

func (c *Client) runStats(ctx context.Context) {
	// persistent stream with reconnect and drain of stats channel
	op := backoff.NewExponentialBackOff()
	op.InitialInterval = 500 * time.Millisecond
	op.MaxInterval = 10 * time.Second
	var stream grpc.ClientStreamingClient[pb.SendStatsRequest, emptypb.Empty]

	connect := func() bool {
		s, err := c.client.Stats(ctx)
		if err != nil {
			return false
		}
		stream = s
		op.Reset()
		return true
	}

	for {
		if stream == nil {
			if !connect() {
				sleep := op.NextBackOff()
				if sleep == backoff.Stop {
					sleep = op.MaxInterval
				}
				time.Sleep(sleep)
				continue
			}
		}
		select {
		case <-ctx.Done():
			if stream != nil {
				_, _ = stream.CloseAndRecv()
			}
			return
		case feat := <-c.statsCh:
			if stream == nil {
				continue
			}
			_ = stream.Send(&pb.SendStatsRequest{ServiceName: c.serviceName, FeatureName: feat})
		}
	}
}

// percentageHit returns true if hash(seed+featureName) bucket is below percent [0..100]
func percentageHit(featureName string, seed string, percent int) bool { // deprecated: kept for backwards compatibility
	return fastBucketHit(featureName, seed, percent)
}

// fastBucketHit computes a bucket [0..99] using FNV-1a without allocations.
func fastBucketHit(featureName string, seed string, percent int) bool {
	// clamp percent
	if percent <= 0 {
		return false
	}
	if percent >= 100 {
		return true
	}
	const (
		offset64 = 1469598103934665603
		prime64  = 1099511628211
	)
	var hash uint64 = offset64
	for i := 0; i < len(featureName); i++ {
		hash ^= uint64(featureName[i])
		hash *= prime64
	}
	// '::'
	hash ^= uint64(':')
	hash *= prime64
	hash ^= uint64(':')
	hash *= prime64
	for i := 0; i < len(seed); i++ {
		hash ^= uint64(seed[i])
		hash *= prime64
	}
	bucket := int(hash % 100)
	return bucket < percent
}

func clampPercent(p int) int {
	switch {
	case p < 0:
		return 0
	case p > 100:
		return 100
	default:
		return p
	}
}
