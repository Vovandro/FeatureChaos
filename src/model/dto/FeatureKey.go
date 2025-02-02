package dto

import "github.com/google/uuid"

type FeatureKey struct {
	Id          uuid.UUID
	Key         string
	Description string
	Value       int
	Params      []FeatureParam
}
