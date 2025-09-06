package FeatureService

import (
	"context"

	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/dto"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ActivationValuesRepository"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/service"
)

type Service struct {
	service.Mock
	activationValuesRepository ActivationValuesRepository.Interface
}

func New(name string) *Service {
	return &Service{
		Mock: service.Mock{
			NamePkg: name,
		},
	}
}

func NewForTest(activationValuesRepository ActivationValuesRepository.Interface) *Service {
	return &Service{
		activationValuesRepository: activationValuesRepository,
	}
}

func (t *Service) Init(app interfaces.IEngine, cfg map[string]interface{}) error {
	t.activationValuesRepository = app.GetModule(interfaces.ModuleRepository, names.ActivationValuesRepository).(ActivationValuesRepository.Interface)

	return nil
}

func (t *Service) GetNewFeature(c context.Context, serviceName string, lastVersion int64) (int64, []*dto.Feature) {
	version, features, err := t.activationValuesRepository.GetNewByServiceName(c, serviceName, lastVersion)
	if err != nil {
		return 0, nil
	}

	return version, features
}
