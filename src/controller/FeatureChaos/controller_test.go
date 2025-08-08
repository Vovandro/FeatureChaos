package FeatureChaos

import (
	"context"
	"io"
	"testing"
	"time"

	"gitlab.com/devpro_studio/FeatureChaos/src/model/dto"
	FSvc "gitlab.com/devpro_studio/FeatureChaos/src/service/FeatureService"
	SSvc "gitlab.com/devpro_studio/FeatureChaos/src/service/StatsService"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type fakeFeatureSvc struct{ resp []*dto.Feature }

func (f *fakeFeatureSvc) GetNewFeature(context.Context, string, int64) []*dto.Feature { return f.resp }

type fakeStatsSvc struct{ called int }

func (f *fakeStatsSvc) SetStat(context.Context, string, string) { f.called++ }

// simple in-memory stream for server streaming
type memStream struct {
	ctx  context.Context
	sent []*GetFeatureResponse
}

func (m *memStream) Context() context.Context         { return m.ctx }
func (m *memStream) Send(r *GetFeatureResponse) error { m.sent = append(m.sent, r); return nil }
func (m *memStream) SetHeader(metadata.MD) error      { return nil }
func (m *memStream) SendHeader(metadata.MD) error     { return nil }
func (m *memStream) SetTrailer(metadata.MD)           {}
func (m *memStream) SendMsg(any) error                { return nil }
func (m *memStream) RecvMsg(any) error                { return nil }

// client stream for Stats
type memClientStream struct {
	ctx    context.Context
	in     []*SendStatsRequest
	idx    int
	closed bool
}

func (m *memClientStream) Context() context.Context { return m.ctx }
func (m *memClientStream) Recv() (*SendStatsRequest, error) {
	if m.idx >= len(m.in) {
		return nil, io.EOF
	}
	v := m.in[m.idx]
	m.idx++
	return v, nil
}
func (m *memClientStream) SendAndClose(*emptypb.Empty) error { m.closed = true; return nil }
func (m *memClientStream) SetHeader(metadata.MD) error       { return nil }
func (m *memClientStream) SendHeader(metadata.MD) error      { return nil }
func (m *memClientStream) SetTrailer(metadata.MD)            {}
func (m *memClientStream) SendMsg(any) error                 { return nil }
func (m *memClientStream) RecvMsg(any) error                 { return nil }

var _ FSvc.Interface = (*fakeFeatureSvc)(nil)
var _ SSvc.Interface = (*fakeStatsSvc)(nil)

func TestSubscribe_SendsConvertedFeatures(t *testing.T) {
	c := &Controller{}
	c.featureService = &fakeFeatureSvc{resp: []*dto.Feature{{Name: "n", Value: 1, Version: 5}}}
	stream := &memStream{ctx: context.Background()}

	// run a single tick; stop by cancelling context
	ctx, cancel := context.WithCancel(context.Background())
	stream.ctx = ctx
	go func() { time.Sleep(1200 * time.Millisecond); cancel() }()
	_ = c.Subscribe(&GetAllFeatureRequest{ServiceName: "svc", LastVersion: 0}, stream)
	if len(stream.sent) == 0 {
		t.Fatalf("expected at least one message sent")
	}
	if stream.sent[len(stream.sent)-1].Version != 5 {
		t.Fatalf("expected version 5, got %d", stream.sent[len(stream.sent)-1].Version)
	}
}

func TestStats_ConsumesUntilEOF(t *testing.T) {
	c := &Controller{}
	fs := &fakeStatsSvc{}
	c.statsService = fs
	st := &memClientStream{ctx: context.Background(), in: []*SendStatsRequest{{ServiceName: "s", FeatureName: "f"}}}
	if err := c.Stats(st); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !st.closed || fs.called != 1 {
		t.Fatalf("stream not closed or stats not recorded")
	}
}
