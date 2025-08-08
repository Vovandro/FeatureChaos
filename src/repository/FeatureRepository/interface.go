package FeatureRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
)

type Interface interface {
	GetFeaturesList(c context.Context, ids []uuid.UUID) []*db.Feature
	ListFeatures(c context.Context) []*db.Feature
	CreateFeature(c context.Context, name string, description string) (uuid.UUID, error)
	UpdateFeature(c context.Context, id uuid.UUID, name string, description string) error
	DeleteFeature(c context.Context, id uuid.UUID) error
}
