package FeatureService

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	KeyRepo "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureKeyRepository"
	ParamRepo "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	FeatRepo "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureRepository"
	AccessRepo "gitlab.com/devpro_studio/FeatureChaos/src/repository/ServiceAccessRepository"
)

// Fakes for repository interfaces
type fakeAccessRepo struct{ ids []uuid.UUID }

func (f *fakeAccessRepo) GetNewByServiceName(_ context.Context, _ string, _ int64) []uuid.UUID {
	return f.ids
}

type fakeFeatureRepo struct{ features []*db.Feature }

func (f *fakeFeatureRepo) GetFeaturesList(_ context.Context, _ []uuid.UUID) []*db.Feature {
	return f.features
}
func (f *fakeFeatureRepo) ListFeatures(_ context.Context) []*db.Feature { return f.features }
func (f *fakeFeatureRepo) CreateFeature(_ context.Context, _ string, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (f *fakeFeatureRepo) UpdateFeature(_ context.Context, _ uuid.UUID, _ string, _ string) error {
	return nil
}
func (f *fakeFeatureRepo) DeleteFeature(_ context.Context, _ uuid.UUID) error { return nil }

type fakeKeyRepo struct{ keys []*db.FeatureKey }

func (f *fakeKeyRepo) GetFeatureKeys(_ context.Context, _ []uuid.UUID) []*db.FeatureKey {
	return f.keys
}
func (f *fakeKeyRepo) CreateKey(_ context.Context, _ uuid.UUID, _ string, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (f *fakeKeyRepo) UpdateKey(_ context.Context, _ uuid.UUID, _ string, _ string) error { return nil }
func (f *fakeKeyRepo) DeleteKey(_ context.Context, _ uuid.UUID) error                     { return nil }
func (f *fakeKeyRepo) DeleteByFeatureId(_ context.Context, _ uuid.UUID) error             { return nil }
func (f *fakeKeyRepo) GetFeatureIdByKeyId(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

type fakeParamRepo struct{ params []*db.FeatureParam }

func (f *fakeParamRepo) GetFeatureParams(_ context.Context, _ []uuid.UUID) []*db.FeatureParam {
	return f.params
}
func (f *fakeParamRepo) GetParamsByKeyId(_ context.Context, _ uuid.UUID) []*db.FeatureParam {
	return f.params
}
func (f *fakeParamRepo) CreateParam(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (f *fakeParamRepo) UpdateParam(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (f *fakeParamRepo) DeleteParam(_ context.Context, _ uuid.UUID) error           { return nil }
func (f *fakeParamRepo) GetFeatureIdByParamId(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func TestGetNewFeature_NoAccessIds(t *testing.T) {
	svc := &Service{}
	// inject fakes via exported fields through Init is not possible, so set directly
	svc.accessRepository = &fakeAccessRepo{ids: nil}
	svc.featureRepository = &fakeFeatureRepo{}
	svc.featureKeyRepository = &fakeKeyRepo{}
	svc.featureParamRepository = &fakeParamRepo{}

	got := svc.GetNewFeature(context.Background(), "svc", 0)
	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestGetNewFeature_AssembleHierarchyAndVersion(t *testing.T) {
	fid := uuid.New()
	kid := uuid.New()
	pid := uuid.New()
	access := &fakeAccessRepo{ids: []uuid.UUID{fid}}
	features := &fakeFeatureRepo{features: []*db.Feature{{Id: fid, Name: "f1", Description: "d", Value: 10, Version: 1}}}
	keys := &fakeKeyRepo{keys: []*db.FeatureKey{{Id: kid, FeatureId: fid, Key: "k1", Description: "kd", Value: 5, Version: 2}}}
	params := &fakeParamRepo{params: []*db.FeatureParam{{Id: pid, FeatureId: fid, KeyId: kid, Key: "p1", Value: 1, Version: 3}}}

	svc := &Service{
		accessRepository:       AccessRepo.Interface(access),
		featureRepository:      FeatRepo.Interface(features),
		featureKeyRepository:   KeyRepo.Interface(keys),
		featureParamRepository: ParamRepo.Interface(params),
	}

	got := svc.GetNewFeature(context.Background(), "svc", 0)
	if len(got) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(got))
	}
	f := got[0]
	if f.ID != fid || f.Name != "f1" || f.Description != "d" || f.Value != 10 {
		t.Fatalf("unexpected feature: %#v", f)
	}
	// version should be max of feature, key, and param versions => 3
	if f.Version != 3 {
		t.Fatalf("expected version 3, got %d", f.Version)
	}
	if len(f.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(f.Keys))
	}
	k := f.Keys[0]
	if k.Id != kid || k.Key != "k1" || k.Description != "kd" || k.Value != 5 {
		t.Fatalf("unexpected key: %#v", k)
	}
	if len(k.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(k.Params))
	}
	p := k.Params[0]
	if p.Id != pid || p.Name != "p1" || p.Value != 1 {
		t.Fatalf("unexpected param: %#v", p)
	}
}
