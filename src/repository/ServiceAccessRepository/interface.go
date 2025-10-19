package ServiceAccessRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
)

type Interface interface {
	ListServices(c context.Context) []db.Service
	CreateService(c context.Context, name string) (uuid.UUID, error)
	DeleteService(c context.Context, id uuid.UUID) error

	GetAccess(c context.Context) ([]*db.ServiceAccess, error)
	GetAccessByFeatures(c context.Context, featureIds []uuid.UUID) (map[uuid.UUID][]*db.ServiceAccess, error)
	AddAccess(c context.Context, featureId uuid.UUID, serviceId uuid.UUID) error
	RemoveAccess(c context.Context, featureId uuid.UUID, serviceId uuid.UUID) error
}
