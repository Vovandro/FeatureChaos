package ActivationValuesRepository

import (
	"context"

	"github.com/google/uuid"
)

type Interface interface {
	InsertValue(c context.Context, featureId uuid.UUID, keyId *uuid.UUID, paramId *uuid.UUID, value int) (int64, error)
	DeleteByFeatureId(c context.Context, featureId uuid.UUID) error
	DeleteByKeyId(c context.Context, keyId uuid.UUID) error
	DeleteByParamId(c context.Context, paramId uuid.UUID) error
}
