package FeatureService

import (
	"context"
	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/dto"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureKeyRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ServiceAccessRepository"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/service"
)

type Service struct {
	service.Mock
	accessRepository       ServiceAccessRepository.Interface
	featureRepository      FeatureRepository.Interface
	featureKeyRepository   FeatureKeyRepository.Interface
	featureParamRepository FeatureParamRepository.Interface
}

func New(name string) *Service {
	return &Service{
		Mock: service.Mock{
			NamePkg: name,
		},
	}
}

func (t *Service) Init(app interfaces.IEngine, cfg map[string]interface{}) error {
	t.accessRepository = app.GetModule(interfaces.ModuleRepository, "service_access").(ServiceAccessRepository.Interface)
	t.featureRepository = app.GetModule(interfaces.ModuleRepository, "feature").(FeatureRepository.Interface)
	t.featureKeyRepository = app.GetModule(interfaces.ModuleRepository, "feature_key").(FeatureKeyRepository.Interface)
	t.featureParamRepository = app.GetModule(interfaces.ModuleRepository, "feature_param").(FeatureParamRepository.Interface)

	return nil
}

func (t *Service) GetNewFeature(c context.Context, serviceName string, lastVersion int64) []*dto.Feature {
	featuresIds := t.accessRepository.GetNewByServiceName(c, serviceName, lastVersion)

	if featuresIds == nil || len(featuresIds) == 0 {
		return nil
	}

	features := t.featureRepository.GetFeaturesList(c, featuresIds)

	if features == nil || len(features) == 0 {
		return nil
	}

	keys := t.featureKeyRepository.GetFeatureKeys(c, featuresIds)

	groupsKey := make(map[uuid.UUID][]*db.FeatureKey, len(keys))

	for _, key := range keys {
		groupsKey[key.FeatureId] = append(groupsKey[key.FeatureId], key)
	}

	params := t.featureParamRepository.GetFeatureParams(c, featuresIds)

	groupsParam := make(map[uuid.UUID][]*db.FeatureParam, len(params))

	for _, param := range params {
		groupsParam[param.KeyId] = append(groupsParam[param.KeyId], param)
	}

	res := make([]*dto.Feature, len(features))

	for i, feature := range features {
		res[i] = &dto.Feature{
			ID:          feature.Id,
			Name:        feature.Name,
			Description: feature.Description,
			Version:     feature.Version,
			Value:       feature.Value,
		}

		if keys, ok := groupsKey[feature.Id]; ok {
			res[i].Keys = make([]dto.FeatureKey, len(keys))
			for j, key := range keys {
				res[i].Keys[j].Id = key.Id
				res[i].Keys[j].Key = key.Key
				res[i].Keys[j].Description = key.Description
				res[i].Keys[j].Value = key.Value

				if res[i].Version < key.Version {
					res[i].Version = key.Version
				}

				if params, ok2 := groupsParam[key.Id]; ok2 {
					res[i].Keys[j].Params = make([]dto.FeatureParam, len(params))
					for k, param := range params {
						res[i].Keys[j].Params[k].Id = param.Id
						res[i].Keys[j].Params[k].Name = param.Key
						res[i].Keys[j].Params[k].Value = param.Value

						if res[i].Version < param.Version {
							res[i].Version = param.Version
						}
					}
				}
			}
		}
	}

	return res
}
