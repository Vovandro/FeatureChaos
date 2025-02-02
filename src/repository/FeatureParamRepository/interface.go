package FeatureParamRepository

import (
	"context"
	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
)

type Interface interface {
	GetFeatureParams(c context.Context, featureIds []uuid.UUID) []*db.FeatureParam
}
