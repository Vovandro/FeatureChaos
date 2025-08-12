package fc_sdk_go

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "gitlab.com/devpro_studio/FeatureChaos/src/controller/FeatureChaos"
)

// --- unit tests without server ---

func TestNew_Validation(t *testing.T) {
	if _, err := New(context.Background(), "localhost", "svc", Options{}); err == nil {
		t.Fatalf("expected error for bad address format")
	}
	if _, err := New(context.Background(), "127.0.0.1:5000", "", Options{}); err == nil {
		t.Fatalf("expected error for empty service name")
	}
}

func TestIsEnabled_GlobalAndKeysAndTrack(t *testing.T) {
	c := &Client{
		features: map[string]FeatureConfig{
			"f": {
				Name:       "f",
				AllPercent: 0,
				Keys: map[string]KeyConfig{
					"country": {AllPercent: 10, Items: map[string]int{"US": 100}},
				},
			},
		},
		statsCh:   make(chan string, 2),
		autoStats: true,
	}

	// global 0 => disabled
	if c.IsEnabled("f", "user1", map[string]string{}) {
		t.Fatalf("expected disabled by global=0")
	}
	// item override 100 => enabled
	if !c.IsEnabled("f", "user1", map[string]string{"country": "US"}) {
		t.Fatalf("expected enabled by key item=100")
	}
	// when enabled and autoStats, Track is enqueued
	select {
	case v := <-c.statsCh:
		if v != "f" {
			t.Fatalf("unexpected stat %s", v)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected stat sent")
	}
}

func TestGetSnapshot_DeepCopy(t *testing.T) {
	c := &Client{features: map[string]FeatureConfig{
		"a": {Name: "a", AllPercent: 1, Keys: map[string]KeyConfig{"k": {AllPercent: 2, Items: map[string]int{"x": 3}}}},
	}}
	snap := c.GetSnapshot()
	snap["a"].Keys["k"].Items["x"] = 99
	if c.features["a"].Keys["k"].Items["x"] == 99 {
		t.Fatalf("expected deep copy to protect original map")
	}
}

// --- integration with in-memory grpc server ---

type fakeSrv struct {
	pb.UnimplementedFeatureServiceServer
	mu    sync.Mutex
	stats []string
}

func (f *fakeSrv) Subscribe(req *pb.GetAllFeatureRequest, stream grpc.ServerStreamingServer[pb.GetFeatureResponse]) error {
	_ = req
	// send single update
	resp := &pb.GetFeatureResponse{
		Features: []*pb.FeatureItem{
			{Name: "f1", All: 20, Props: []*pb.PropsItem{{Name: "country", All: 0, Item: map[string]int32{"US": 100}}}},
		},
		Version: 5,
	}
	if err := stream.Send(resp); err != nil {
		return err
	}
	// end stream so client reconnect path is exercised (but not needed)
	return nil
}

func (f *fakeSrv) Stats(stream grpc.ClientStreamingServer[pb.SendStatsRequest, emptypb.Empty]) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return stream.SendAndClose(&emptypb.Empty{})
		}
		f.mu.Lock()
		f.stats = append(f.stats, req.FeatureName)
		f.mu.Unlock()
	}
}

func startTestServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := grpc.NewServer()
	srv := &fakeSrv{}
	pb.RegisterFeatureServiceServer(s, srv)
	go func() { _ = s.Serve(lis) }()
	return lis.Addr().String(), func() { s.GracefulStop() }
}

func TestRunSubscriberAndOnUpdate(t *testing.T) {
	addr, stop := startTestServer(t)
	defer stop()

	got := make(chan UpdateEvent, 1)
	c, err := New(context.Background(), addr, "svcA", Options{OnUpdate: func(ev UpdateEvent) { got <- ev }})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer c.Close()

	select {
	case ev := <-got:
		if ev.Version != 5 {
			t.Fatalf("expected version 5, got %d", ev.Version)
		}
		snap := c.GetSnapshot()
		if _, ok := snap["f1"]; !ok {
			t.Fatalf("expected f1 in snapshot")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for update")
	}
}

func TestRunStats_TrackSends(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := grpc.NewServer()
	srv := &fakeSrv{}
	pb.RegisterFeatureServiceServer(s, srv)
	go func() { _ = s.Serve(lis) }()
	defer s.GracefulStop()

	c, err := New(context.Background(), lis.Addr().String(), "svcB", Options{StatsFlushInterval: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer c.Close()

	c.Track("f1")
	c.Track("f2")
	// give some time to send
	time.Sleep(300 * time.Millisecond)

	srv.mu.Lock()
	got := append([]string(nil), srv.stats...)
	srv.mu.Unlock()
	if len(got) < 2 {
		t.Fatalf("expected at least 2 stats, got %v", got)
	}
}

func TestIsEnabled_Matrix(t *testing.T) {
	type tc struct {
		name      string
		cfg       FeatureConfig
		attrs     map[string]string
		autoStats bool
		expect    bool
	}
	cases := []tc{
		{
			name:      "global 0 -> disabled",
			cfg:       FeatureConfig{Name: "f", AllPercent: 0, Keys: nil},
			attrs:     map[string]string{},
			autoStats: true,
			expect:    false,
		},
		{
			name:      "global 100 -> enabled",
			cfg:       FeatureConfig{Name: "f", AllPercent: 100, Keys: nil},
			attrs:     map[string]string{"any": "x"},
			autoStats: true,
			expect:    true,
		},
		{
			name:      "key all 100 with matching attr -> enabled",
			cfg:       FeatureConfig{Name: "f", AllPercent: 0, Keys: map[string]KeyConfig{"country": {AllPercent: 100}}},
			attrs:     map[string]string{"country": "RU"},
			autoStats: true,
			expect:    true,
		},
		{
			name:      "key all 100 without attr -> global 0 -> disabled",
			cfg:       FeatureConfig{Name: "f", AllPercent: 0, Keys: map[string]KeyConfig{"country": {AllPercent: 100}}},
			attrs:     map[string]string{"lang": "en"},
			autoStats: true,
			expect:    false,
		},
		{
			name:      "key item 100 with matching value -> enabled",
			cfg:       FeatureConfig{Name: "f", AllPercent: 0, Keys: map[string]KeyConfig{"tier": {AllPercent: 0, Items: map[string]int{"pro": 100}}}},
			attrs:     map[string]string{"tier": "pro"},
			autoStats: true,
			expect:    true,
		},
		{
			name:      "key item 100 non-matching -> global 0 -> disabled",
			cfg:       FeatureConfig{Name: "f", AllPercent: 0, Keys: map[string]KeyConfig{"tier": {AllPercent: 0, Items: map[string]int{"pro": 100}}}},
			attrs:     map[string]string{"tier": "free"},
			autoStats: true,
			expect:    false,
		},
		{
			name: "multiple attrs one grants 100 -> enabled",
			cfg: FeatureConfig{Name: "f", AllPercent: 0, Keys: map[string]KeyConfig{
				"country": {AllPercent: 0, Items: map[string]int{"US": 100}},
				"device":  {AllPercent: 0, Items: map[string]int{"ios": 0}},
			}},
			attrs:     map[string]string{"country": "US", "device": "ios"},
			autoStats: true,
			expect:    true,
		},
		{
			name:      "enabled but autoStats=false -> no stat",
			cfg:       FeatureConfig{Name: "f", AllPercent: 100, Keys: nil},
			attrs:     map[string]string{},
			autoStats: false,
			expect:    true,
		},
	}

	for _, cse := range cases {
		t.Run(cse.name, func(t *testing.T) {
			c := &Client{
				features:  map[string]FeatureConfig{"f": cse.cfg},
				statsCh:   make(chan string, 4),
				autoStats: cse.autoStats,
			}
			got := c.IsEnabled("f", "seed", cse.attrs)
			if got != cse.expect {
				t.Fatalf("expected %v, got %v", cse.expect, got)
			}
			// verify stats behavior deterministically
			select {
			case v := <-c.statsCh:
				if !cse.expect || !cse.autoStats {
					t.Fatalf("unexpected stat %q when expect=%v autoStats=%v", v, cse.expect, cse.autoStats)
				}
			case <-time.After(50 * time.Millisecond):
				if cse.expect && cse.autoStats {
					t.Fatalf("expected stat but none received")
				}
			}
		})
	}
}
