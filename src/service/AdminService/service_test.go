package AdminService

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	AVRepo "gitlab.com/devpro_studio/FeatureChaos/src/repository/ActivationValuesRepository"
	KeyRepo "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureKeyRepository"
	ParamRepo "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	FeatRepo "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureRepository"
)

type fakeFeatRepo struct {
	createErr, updErr, delErr error
	created                   uuid.UUID
}

func (f *fakeFeatRepo) GetFeaturesList(context.Context, []uuid.UUID) []*db.Feature { return nil }
func (f *fakeFeatRepo) ListFeatures(context.Context) []*db.Feature                 { return nil }
func (f *fakeFeatRepo) CreateFeature(context.Context, string, string) (uuid.UUID, error) {
	if f.createErr != nil {
		return uuid.Nil, f.createErr
	}
	if f.created == uuid.Nil {
		f.created = uuid.New()
	}
	return f.created, nil
}
func (f *fakeFeatRepo) UpdateFeature(context.Context, uuid.UUID, string, string) error {
	return f.updErr
}
func (f *fakeFeatRepo) DeleteFeature(context.Context, uuid.UUID) error { return f.delErr }

type fakeKeyRepo struct {
	featureIdByKey map[uuid.UUID]uuid.UUID
	err            error
}

func (f *fakeKeyRepo) GetFeatureKeys(context.Context, []uuid.UUID) []*db.FeatureKey { return nil }
func (f *fakeKeyRepo) CreateKey(context.Context, uuid.UUID, string, string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (f *fakeKeyRepo) UpdateKey(context.Context, uuid.UUID, string, string) error { return nil }
func (f *fakeKeyRepo) DeleteKey(context.Context, uuid.UUID) error                 { return nil }
func (f *fakeKeyRepo) DeleteByFeatureId(context.Context, uuid.UUID) error         { return nil }
func (f *fakeKeyRepo) GetFeatureIdByKeyId(context.Context, uuid.UUID) (uuid.UUID, error) {
	for k, v := range f.featureIdByKey {
		_ = k
		return v, f.err
	}
	return uuid.Nil, f.err
}

type fakeParamRepo struct {
	featureIdByParam map[uuid.UUID]uuid.UUID
	err              error
}

func (f *fakeParamRepo) GetFeatureParams(context.Context, []uuid.UUID) []*db.FeatureParam { return nil }
func (f *fakeParamRepo) GetParamsByKeyId(context.Context, uuid.UUID) []*db.FeatureParam   { return nil }
func (f *fakeParamRepo) CreateParam(context.Context, uuid.UUID, uuid.UUID, string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (f *fakeParamRepo) UpdateParam(context.Context, uuid.UUID, string) error { return nil }
func (f *fakeParamRepo) DeleteParam(context.Context, uuid.UUID) error         { return nil }
func (f *fakeParamRepo) GetFeatureIdByParamId(context.Context, uuid.UUID) (uuid.UUID, error) {
	for k, v := range f.featureIdByParam {
		_ = k
		return v, f.err
	}
	return uuid.Nil, f.err
}

type fakeAVRepo struct {
	v   int64
	err error
}

func (f *fakeAVRepo) InsertValue(context.Context, uuid.UUID, *uuid.UUID, *uuid.UUID, int) (int64, error) {
	return f.v, f.err
}
func (f *fakeAVRepo) DeleteByFeatureId(context.Context, uuid.UUID) error { return f.err }
func (f *fakeAVRepo) DeleteByKeyId(context.Context, uuid.UUID) error     { return f.err }
func (f *fakeAVRepo) DeleteByParamId(context.Context, uuid.UUID) error   { return f.err }

func TestCreateFeature_HappyPath(t *testing.T) {
	feat := &fakeFeatRepo{}
	av := &fakeAVRepo{v: 1}
	svc := &Service{features: FeatRepo.Interface(feat), values: AVRepo.Interface(av)}
	id, err := svc.CreateFeature(context.Background(), "name", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == uuid.Nil {
		t.Fatalf("expected non-nil id")
	}
}

func TestSetKeyValue_UsesFeatureIdFromKey(t *testing.T) {
	fid := uuid.New()
	kid := uuid.New()
	keyRepo := &fakeKeyRepo{featureIdByKey: map[uuid.UUID]uuid.UUID{kid: fid}}
	av := &fakeAVRepo{v: 2}
	svc := &Service{keys: KeyRepo.Interface(keyRepo), values: AVRepo.Interface(av)}

	v, err := svc.SetKeyValue(context.Background(), kid, 123)
	if err != nil || v != 2 {
		t.Fatalf("unexpected result v=%d err=%v", v, err)
	}
}

func TestSetParamValue_ErrorOnLookup(t *testing.T) {
	param := &fakeParamRepo{err: errors.New("lookup failed")}
	av := &fakeAVRepo{v: 0}
	svc := &Service{params: ParamRepo.Interface(param), values: AVRepo.Interface(av)}
	if _, err := svc.SetParamValue(context.Background(), uuid.New(), 5); err == nil {
		t.Fatalf("expected error")
	}
}
