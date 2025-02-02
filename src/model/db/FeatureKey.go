package db

import "github.com/google/uuid"

type FeatureKey struct {
	Id          uuid.UUID
	FeatureId   uuid.UUID
	Key         string
	Description string
	Value       int
	Version     int64
}
