package ServiceAccessRepository

import (
	"context"

	"github.com/google/uuid"
)

type Interface interface {
	GetNewByServiceName(c context.Context, serviceName string, lastVersion int64) []uuid.UUID
	// Services CRUD
	ListServices(c context.Context) []Service
	CreateService(c context.Context, name string) (uuid.UUID, error)
	DeleteService(c context.Context, id uuid.UUID) error
	GetServiceById(c context.Context, id uuid.UUID) (Service, bool)
	HasAnyAccessByService(c context.Context, id uuid.UUID) bool
	// Feature-Service bindings
	GetServicesByFeature(c context.Context, featureId uuid.UUID) []Service
	GetServicesByFeatureList(c context.Context, featureIds []uuid.UUID) map[uuid.UUID][]Service
	AddAccess(c context.Context, featureId uuid.UUID, serviceId uuid.UUID) error
	RemoveAccess(c context.Context, featureId uuid.UUID, serviceId uuid.UUID) error
}

type Service struct {
	Id   uuid.UUID
	Name string
}
