package FeatureService

import (
	"context"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/dto"
)

type Interface interface {
	GetNewFeature(c context.Context, serviceName string, lastVersion int64) []*dto.Feature
}
