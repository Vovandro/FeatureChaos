package AdminService

import (
	"context"

	"github.com/google/uuid"
)

type Interface interface {
	CreateFeature(ctx context.Context, name string, description string) (uuid.UUID, error)
	UpdateFeature(ctx context.Context, id uuid.UUID, name string, description string) error
	DeleteFeature(ctx context.Context, id uuid.UUID) error
	SetFeatureValue(ctx context.Context, id uuid.UUID, value int) (int64, error)

	CreateKey(ctx context.Context, featureId uuid.UUID, key string, description string) (uuid.UUID, error)
	UpdateKey(ctx context.Context, keyId uuid.UUID, key string, description string) error
	DeleteKey(ctx context.Context, keyId uuid.UUID) error
	SetKeyValue(ctx context.Context, keyId uuid.UUID, value int) (int64, error)

	CreateParam(ctx context.Context, keyId uuid.UUID, name string) (uuid.UUID, error)
	UpdateParam(ctx context.Context, paramId uuid.UUID, name string) error
	DeleteParam(ctx context.Context, paramId uuid.UUID) error
	SetParamValue(ctx context.Context, paramId uuid.UUID, value int) (int64, error)
}
