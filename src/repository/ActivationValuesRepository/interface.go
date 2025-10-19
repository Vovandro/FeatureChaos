package ActivationValuesRepository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/dto"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Interface interface {
	InsertValue(c context.Context, tx postgres.SQLTx, featureId uuid.UUID, keyId *uuid.UUID, paramId *uuid.UUID, value int) (int64, error)

	GetNewByServiceName(c context.Context, serviceName string, lastVersion int64) (int64, []*dto.Feature, error)

	GetFeatures(c context.Context, serviceId string, page int, pageSize int, find string, isDeprecated bool, deprecatedTime time.Duration) ([]*dto.Feature, int, error)

	DeleteByFeatureId(c context.Context, tx postgres.SQLTx, featureId uuid.UUID) error
	DeleteByKeyId(c context.Context, tx postgres.SQLTx, keyId uuid.UUID) error
	DeleteByParamId(c context.Context, tx postgres.SQLTx, paramId uuid.UUID) error
}
