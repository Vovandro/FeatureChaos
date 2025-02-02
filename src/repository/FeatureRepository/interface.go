package FeatureRepository

import (
	"context"
	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
)

type Interface interface {
	GetFeaturesList(c context.Context, ids []uuid.UUID) []*db.Feature
}
