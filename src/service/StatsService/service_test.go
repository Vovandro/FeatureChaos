package StatsService

import (
	"context"
	"testing"
)

type fakeStatsRepo struct {
	called    bool
	svc, feat string
}

func (f *fakeStatsRepo) SetStat(_ context.Context, serviceName string, featureName string) {
	f.called = true
	f.svc = serviceName
	f.feat = featureName
}

func TestSetStatDelegation(t *testing.T) {
	repo := &fakeStatsRepo{}
	svc := &Service{statsRepository: repo}
	svc.SetStat(context.Background(), "s", "f")
	if !repo.called || repo.svc != "s" || repo.feat != "f" {
		t.Fatalf("repo not called correctly: %#v", repo)
	}
}
