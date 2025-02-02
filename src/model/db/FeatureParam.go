package db

import "github.com/google/uuid"

type FeatureParam struct {
	Id        uuid.UUID
	FeatureId uuid.UUID
	KeyId     uuid.UUID
	Key       string
	Value     int
	Version   int64
}
