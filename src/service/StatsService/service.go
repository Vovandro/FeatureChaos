package StatsService

import (
	"context"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/StatsRepository"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/service"
)

type Service struct {
	service.Mock
	statsRepository StatsRepository.Interface
}

func New(name string) *Service {
	return &Service{
		Mock: service.Mock{
			NamePkg: name,
		},
	}
}

func (t *Service) Init(app interfaces.IEngine, cfg map[string]interface{}) error {
	t.statsRepository = app.GetModule(interfaces.ModuleRepository, "stats").(StatsRepository.Interface)
	return nil
}

func (t *Service) SetStat(c context.Context, serviceName string, featureName string) {
	t.statsRepository.SetStat(c, serviceName, featureName)
}
