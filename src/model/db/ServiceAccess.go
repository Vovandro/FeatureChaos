package db

import "github.com/google/uuid"

type ServiceAccess struct {
	ID        uuid.UUID
	FeatureId uuid.UUID
	ServiceId uuid.UUID
	Name      string
}
