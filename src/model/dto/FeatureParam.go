package dto

import "github.com/google/uuid"

type FeatureParam struct {
	Id    uuid.UUID
	Name  string
	Value int
}
