package FeatureKeyRepository

import (
	"context"
	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
)

type Interface interface {
	GetFeatureKeys(c context.Context, featureIds []uuid.UUID) []*db.FeatureKey
}
