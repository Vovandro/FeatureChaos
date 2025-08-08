package AdminService

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ActivationValuesRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureKeyRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureRepository"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/service"
)

type Service struct {
	service.Mock
	features FeatureRepository.Interface
	keys     FeatureKeyRepository.Interface
	params   FeatureParamRepository.Interface
	values   ActivationValuesRepository.Interface
}

func New(name string) *Service {
	return &Service{Mock: service.Mock{NamePkg: name}}
}

func (t *Service) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	t.features = app.GetModule(interfaces.ModuleRepository, "feature").(FeatureRepository.Interface)
	t.keys = app.GetModule(interfaces.ModuleRepository, "feature_key").(FeatureKeyRepository.Interface)
	t.params = app.GetModule(interfaces.ModuleRepository, "feature_param").(FeatureParamRepository.Interface)
	t.values = app.GetModule(interfaces.ModuleRepository, "activation_values").(ActivationValuesRepository.Interface)
	return nil
}

func (t *Service) CreateFeature(ctx context.Context, name string, description string) (uuid.UUID, error) {
	id, err := t.features.CreateFeature(ctx, name, description)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := t.values.InsertValue(ctx, id, nil, nil, 0); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (t *Service) UpdateFeature(ctx context.Context, id uuid.UUID, name string, description string) error {
	return t.features.UpdateFeature(ctx, id, name, description)
}

func (t *Service) DeleteFeature(ctx context.Context, id uuid.UUID) error {
	if err := t.values.DeleteByFeatureId(ctx, id); err != nil {
		return err
	}
	// keys/params deletion assumed handled externally; we remove the feature row
	return t.features.DeleteFeature(ctx, id)
}

func (t *Service) SetFeatureValue(ctx context.Context, id uuid.UUID, value int) (int64, error) {
	return t.values.InsertValue(ctx, id, nil, nil, value)
}

func (t *Service) CreateKey(ctx context.Context, featureId uuid.UUID, key string, description string) (uuid.UUID, error) {
	return t.keys.CreateKey(ctx, featureId, key, description)
}

func (t *Service) UpdateKey(ctx context.Context, keyId uuid.UUID, key string, description string) error {
	return t.keys.UpdateKey(ctx, keyId, key, description)
}

func (t *Service) DeleteKey(ctx context.Context, keyId uuid.UUID) error {
	if err := t.values.DeleteByKeyId(ctx, keyId); err != nil {
		return err
	}
	return t.keys.DeleteKey(ctx, keyId)
}

func (t *Service) SetKeyValue(ctx context.Context, keyId uuid.UUID, value int) (int64, error) {
	featureId, err := t.keys.GetFeatureIdByKeyId(ctx, keyId)
	if err != nil {
		return 0, err
	}
	return t.values.InsertValue(ctx, featureId, &keyId, nil, value)
}

func (t *Service) CreateParam(ctx context.Context, keyId uuid.UUID, name string) (uuid.UUID, error) {
	featureId, err := t.keys.GetFeatureIdByKeyId(ctx, keyId)
	if err != nil {
		return uuid.Nil, err
	}
	return t.params.CreateParam(ctx, featureId, keyId, name)
}

func (t *Service) UpdateParam(ctx context.Context, paramId uuid.UUID, name string) error {
	return t.params.UpdateParam(ctx, paramId, name)
}

func (t *Service) DeleteParam(ctx context.Context, paramId uuid.UUID) error {
	if err := t.values.DeleteByParamId(ctx, paramId); err != nil {
		return err
	}
	return t.params.DeleteParam(ctx, paramId)
}

func (t *Service) SetParamValue(ctx context.Context, paramId uuid.UUID, value int) (int64, error) {
	featureId, err := t.params.GetFeatureIdByParamId(ctx, paramId)
	if err != nil {
		return 0, err
	}
	return t.values.InsertValue(ctx, featureId, nil, &paramId, value)
}
