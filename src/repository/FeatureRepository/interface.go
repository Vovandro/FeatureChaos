package FeatureRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
)

type Interface interface {
	GetFeatureName(c context.Context, id uuid.UUID) (string, error)
	ListFeatures(c context.Context) []*db.Feature

	CreateFeature(c context.Context, name string, description string, value int) (uuid.UUID, error)
	UpdateFeature(c context.Context, id uuid.UUID, name string, description string, value int) error
	DeleteFeature(c context.Context, id uuid.UUID) error
}
