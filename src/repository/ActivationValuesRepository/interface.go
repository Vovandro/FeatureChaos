package ActivationValuesRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/dto"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Interface interface {
	InsertValue(c context.Context, tx postgres.SQLTx, featureId uuid.UUID, keyId *uuid.UUID, paramId *uuid.UUID, value int) (int64, error)

	GetNewByServiceName(c context.Context, serviceName string, lastVersion int64) (int64, []*dto.Feature, error)

	DeleteByFeatureId(c context.Context, tx postgres.SQLTx, featureId uuid.UUID) error
	DeleteByKeyId(c context.Context, tx postgres.SQLTx, keyId uuid.UUID) error
	DeleteByParamId(c context.Context, tx postgres.SQLTx, paramId uuid.UUID) error
}
