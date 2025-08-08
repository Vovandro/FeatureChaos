package FeatureParamRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
)

type Interface interface {
	GetFeatureParams(c context.Context, featureIds []uuid.UUID) []*db.FeatureParam
	GetParamsByKeyId(c context.Context, keyId uuid.UUID) []*db.FeatureParam
	CreateParam(c context.Context, featureId uuid.UUID, keyId uuid.UUID, name string) (uuid.UUID, error)
	UpdateParam(c context.Context, paramId uuid.UUID, name string) error
	DeleteParam(c context.Context, paramId uuid.UUID) error
	GetFeatureIdByParamId(c context.Context, paramId uuid.UUID) (uuid.UUID, error)
}
