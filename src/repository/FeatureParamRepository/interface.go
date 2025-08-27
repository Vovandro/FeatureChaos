package FeatureParamRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Interface interface {
	ListAllParams(c context.Context) map[uuid.UUID][]*db.FeatureParam
	ListParams(c context.Context, keyId uuid.UUID) []*db.FeatureParam
	CreateParam(c context.Context, featureId uuid.UUID, keyId uuid.UUID, name string, value int) (uuid.UUID, error)
	UpdateParam(c context.Context, featureId uuid.UUID, keyId uuid.UUID, paramId uuid.UUID, name string, value int) error
	DeleteParam(c context.Context, paramId uuid.UUID) error

	DeleteAllByKeyId(c context.Context, tx postgres.SQLTx, keyId uuid.UUID) error
	DeleteAllByFeatureId(c context.Context, tx postgres.SQLTx, featureId uuid.UUID) error
}
