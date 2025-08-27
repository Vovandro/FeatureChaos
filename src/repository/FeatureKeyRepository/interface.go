package FeatureKeyRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Interface interface {
	ListAllKeys(c context.Context) map[uuid.UUID][]*db.FeatureKey
	ListKeys(c context.Context, featureId uuid.UUID) []*db.FeatureKey
	CreateKey(c context.Context, featureId uuid.UUID, key string, description string, value int) (uuid.UUID, error)
	UpdateKey(c context.Context, featureId uuid.UUID, keyId uuid.UUID, key string, description string, value int) error
	DeleteKey(c context.Context, keyId uuid.UUID) error

	DeleteAllByFeatureId(c context.Context, tx postgres.SQLTx, featureId uuid.UUID) error
}
