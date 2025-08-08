package FeatureKeyRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
)

type Interface interface {
	GetFeatureKeys(c context.Context, featureIds []uuid.UUID) []*db.FeatureKey
	CreateKey(c context.Context, featureId uuid.UUID, key string, description string) (uuid.UUID, error)
	UpdateKey(c context.Context, keyId uuid.UUID, key string, description string) error
	DeleteKey(c context.Context, keyId uuid.UUID) error
	DeleteByFeatureId(c context.Context, featureId uuid.UUID) error
	GetFeatureIdByKeyId(c context.Context, keyId uuid.UUID) (uuid.UUID, error)
}
