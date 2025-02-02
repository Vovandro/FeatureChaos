package ServiceAccessRepository

import (
	"context"
	"github.com/google/uuid"
)

type Interface interface {
	GetNewByServiceName(c context.Context, serviceName string, lastVersion int64) []uuid.UUID
}
