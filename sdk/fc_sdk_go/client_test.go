package fc_sdk_go

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	pb "gitlab.com/devpro_studio/FeatureChaos/sdk/fc_sdk_go/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

// helper to build a minimal client suitable for unit testing without gRPC
func newTestClient() *Client {
	return &Client{
		features: make(map[string]FeatureConfig),
		statsCh:  make(chan string, 16),
	}
}

func TestIsEnabled_AllPercentClamp(t *testing.T) {
	c := newTestClient()
	c.features["feat"] = FeatureConfig{Name: "feat", AllPercent: -5, Keys: map[string]KeyConfig{}}

	if got := c.IsEnabled("feat", "seed", nil); got {
		t.Fatalf("expected disabled for negative percent, got %v", got)
	}

	c.features["feat"] = FeatureConfig{Name: "feat", AllPercent: 1000, Keys: map[string]KeyConfig{}}
	if got := c.IsEnabled("feat", "seed", nil); !got {
		t.Fatalf("expected enabled for percent > 100 (clamped), got %v", got)
	}
}

func TestIsEnabled_PriorityAndKeyOverrides(t *testing.T) {
	c := newTestClient()
	c.features["search"] = FeatureConfig{
		Name:       "search",
		AllPercent: 0,
		Keys: map[string]KeyConfig{
			"country": { // key-level 100%
				AllPercent: 100,
				Items: map[string]int{
					"US": 0, // exact value override should win over key-level 100
				},
			},
		},
	}

	if enabled := c.IsEnabled("search", "user-1", map[string]string{"country": "US"}); enabled {
		t.Fatalf("expected exact match to force disable (0%%), got enabled")
	}

	if enabled := c.IsEnabled("search", "user-2", map[string]string{"country": "DE"}); !enabled {
		t.Fatalf("expected key-level percent (100%%) to enable, got disabled")
	}

	if enabled := c.IsEnabled("search", "user-3", nil); enabled {
		t.Fatalf("expected feature-level percent (0%%) to disable, got enabled")
	}
}

func TestIsEnabled_AutoStatsTracksOnEnable(t *testing.T) {
	c := newTestClient()
	c.autoStats = true
	c.features["billing"] = FeatureConfig{Name: "billing", AllPercent: 100, Keys: map[string]KeyConfig{}}

	if enabled := c.IsEnabled("billing", "seed", nil); !enabled {
		t.Fatalf("expected enabled for 100%% feature")
	}

	select {
	case feat := <-c.statsCh:
		if feat != "billing" {
			t.Fatalf("expected stats for 'billing', got %q", feat)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected stats event to be queued")
	}
}

func TestGetSnapshot_DeepCopyIsolation(t *testing.T) {
	c := newTestClient()
	c.features["f"] = FeatureConfig{
		Name:       "f",
		AllPercent: 10,
		Keys: map[string]KeyConfig{
			"k": {AllPercent: 20, Items: map[string]int{"US": 30}},
		},
	}

	snap := c.GetSnapshot()
	// mutate snapshot
	snap["f"] = FeatureConfig{Name: "f", AllPercent: 99, Keys: map[string]KeyConfig{"k": {AllPercent: 0, Items: map[string]int{"US": 0}}}}

	if orig := c.features["f"].AllPercent; orig != 10 {
		t.Fatalf("expected original AllPercent to remain 10, got %d", orig)
	}

	// mutate nested map
	k := snap["f"].Keys["k"]
	k.Items["US"] = 99
	if orig := c.features["f"].Keys["k"].Items["US"]; orig != 30 {
		t.Fatalf("expected original nested item to remain 30, got %d", orig)
	}
}

func TestApplyUpdate_UpdatesCacheVersionAndCallback(t *testing.T) {
	c := newTestClient()

	// capture callback
	ch := make(chan UpdateEvent, 1)
	c.onUpdate = func(ev UpdateEvent) { ch <- ev }

	// build proto response
	resp := &pb.GetFeatureResponse{
		Version: 2,
		Features: []*pb.FeatureItem{
			{
				Name: "A",
				All:  30,
				Props: []*pb.PropsItem{
					{Name: "country", All: 40, Item: map[string]int32{"US": 70}},
				},
			},
			{Name: "B", All: 90},
		},
	}

	c.applyUpdate(resp)

	// verify cache
	a, ok := c.features["A"]
	if !ok {
		t.Fatalf("expected feature A to be present")
	}
	if a.AllPercent != 30 {
		t.Fatalf("feature A AllPercent = %d, want 30", a.AllPercent)
	}
	if got := a.Keys["country"].AllPercent; got != 40 {
		t.Fatalf("feature A/country AllPercent = %d, want 40", got)
	}
	if got := a.Keys["country"].Items["US"]; got != 70 {
		t.Fatalf("feature A/country[US] = %d, want 70", got)
	}

	if c.lastVersion != 2 {
		t.Fatalf("lastVersion = %d, want 2", c.lastVersion)
	}

	select {
	case ev := <-ch:
		if ev.Version != 2 {
			t.Fatalf("callback version = %d, want 2", ev.Version)
		}
		if len(ev.Features) != 2 {
			t.Fatalf("callback features len = %d, want 2", len(ev.Features))
		}
		// quick existence checks
		names := map[string]bool{}
		for _, f := range ev.Features {
			names[f.Name] = true
		}
		if !names["A"] || !names["B"] {
			t.Fatalf("callback features missing A or B: %#v", names)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected onUpdate callback to fire")
	}
}

// --- Integration-style test using mock FeatureServiceClient (no real gRPC/proto marshal) ---

type mockSubStream struct {
	ctx context.Context
	ch  <-chan *pb.GetFeatureResponse
}

func (m *mockSubStream) Recv() (*pb.GetFeatureResponse, error) {
	select {
	case <-m.ctx.Done():
		return nil, io.EOF
	case r, ok := <-m.ch:
		if !ok {
			return nil, io.EOF
		}
		return r, nil
	}
}

func (m *mockSubStream) Header() (metadata.MD, error) { return nil, nil }
func (m *mockSubStream) Trailer() metadata.MD         { return nil }
func (m *mockSubStream) CloseSend() error             { return nil }
func (m *mockSubStream) Context() context.Context     { return m.ctx }
func (m *mockSubStream) SendMsg(any) error            { return nil }
func (m *mockSubStream) RecvMsg(any) error            { return nil }

type mockStatsStream struct {
	ctx context.Context
	ch  chan<- *pb.SendStatsRequest
}

func (m *mockStatsStream) Send(req *pb.SendStatsRequest) error {
	select {
	case <-m.ctx.Done():
		return m.ctx.Err()
	case m.ch <- req:
		return nil
	}
}
func (m *mockStatsStream) CloseAndRecv() (*emptypb.Empty, error) { return &emptypb.Empty{}, nil }
func (m *mockStatsStream) Header() (metadata.MD, error)          { return nil, nil }
func (m *mockStatsStream) Trailer() metadata.MD                  { return nil }
func (m *mockStatsStream) CloseSend() error                      { return nil }
func (m *mockStatsStream) Context() context.Context              { return m.ctx }
func (m *mockStatsStream) SendMsg(any) error                     { return nil }
func (m *mockStatsStream) RecvMsg(any) error                     { return nil }

type mockFeatureClient struct {
	subCh   chan *pb.GetFeatureResponse
	statsCh chan *pb.SendStatsRequest
}

func (m *mockFeatureClient) Subscribe(ctx context.Context, in *pb.GetAllFeatureRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.GetFeatureResponse], error) {
	_ = in
	_ = opts
	return &mockSubStream{ctx: ctx, ch: m.subCh}, nil
}
func (m *mockFeatureClient) Stats(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[pb.SendStatsRequest, emptypb.Empty], error) {
	_ = opts
	return &mockStatsStream{ctx: ctx, ch: m.statsCh}, nil
}

func TestIntegration_MockClient_NoPanicAndStatsFlow(t *testing.T) {
	m := &mockFeatureClient{subCh: make(chan *pb.GetFeatureResponse, 4), statsCh: make(chan *pb.SendStatsRequest, 4)}
	c := &Client{
		client:             m,
		serviceName:        "billing",
		features:           make(map[string]FeatureConfig),
		statsCh:            make(chan string, 64),
		autoStats:          true,
		statsFlushInterval: 30 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelRoot = cancel
	gotUpdate := make(chan UpdateEvent, 1)
	c.onUpdate = func(ev UpdateEvent) { gotUpdate <- ev }

	c.wg.Add(2)
	go func() { defer c.wg.Done(); c.runSubscriber(ctx) }()
	go func() { defer c.wg.Done(); c.runStats(ctx) }()

	// push update to mock subscribe stream
	m.subCh <- &pb.GetFeatureResponse{Version: 1, Features: []*pb.FeatureItem{{Name: "flag", All: 100}}}

	select {
	case <-gotUpdate:
	case <-time.After(1 * time.Second):
		t.Fatalf("did not receive update via mock subscribe")
	}

	if !c.IsEnabled("flag", "user-1", nil) {
		t.Fatalf("expected flag enabled after update")
	}

	// expect stats get flushed to mock
	select {
	case r := <-m.statsCh:
		if r.FeatureName != "flag" || r.ServiceName != "billing" {
			t.Fatalf("unexpected stats: %#v", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("stats not flushed to server in time")
	}

	// shutdown
	c.cancelRoot()
	c.wg.Wait()
}

// --- Real in-process gRPC server integration test ---

type realTestFeatureServer struct {
	pb.UnimplementedFeatureServiceServer
	updates       []*pb.GetFeatureResponse
	statsReceived chan *pb.SendStatsRequest
}

func (s *realTestFeatureServer) Subscribe(_ *pb.GetAllFeatureRequest, stream grpc.ServerStreamingServer[pb.GetFeatureResponse]) error {
	for _, u := range s.updates {
		if err := stream.Send(u); err != nil {
			return err
		}
	}
	<-stream.Context().Done()
	return stream.Context().Err()
}

func (s *realTestFeatureServer) Stats(stream grpc.ClientStreamingServer[pb.SendStatsRequest, emptypb.Empty]) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&emptypb.Empty{})
		}
		if err != nil {
			return err
		}
		select {
		case s.statsReceived <- req:
		default:
		}
	}
}

func TestIntegration_RealServer_NoPanicAndStatsFlow(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	srv := &realTestFeatureServer{
		updates:       []*pb.GetFeatureResponse{{Version: 1, Features: []*pb.FeatureItem{{Name: "flag", All: 100}}}},
		statsReceived: make(chan *pb.SendStatsRequest, 16),
	}
	pb.RegisterFeatureServiceServer(grpcSrv, srv)
	go func() { _ = grpcSrv.Serve(lis) }()
	defer grpcSrv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gotUpdate := make(chan UpdateEvent, 1)
	cli, err := New(ctx, lis.Addr().String(), "billing", Options{AutoSendStats: true, StatsFlushInterval: 50 * time.Millisecond, OnUpdate: func(ev UpdateEvent) { gotUpdate <- ev }})
	if err != nil {
		t.Fatalf("New client: %v", err)
	}
	defer func() { _ = cli.Close() }()

	select {
	case <-gotUpdate:
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive update from real server")
	}

	if !cli.IsEnabled("flag", "user-1", nil) {
		t.Fatalf("expected flag enabled from server update")
	}

	select {
	case req := <-srv.statsReceived:
		if req.FeatureName != "flag" || req.ServiceName != "billing" {
			t.Fatalf("unexpected stats: %#v", req)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not receive stats in time")
	}
}
