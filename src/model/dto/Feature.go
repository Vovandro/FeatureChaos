package dto

import "github.com/google/uuid"

type Feature struct {
	ID          uuid.UUID
	Name        string
	Description string
	Version     int64
	Value       int
	Keys        []FeatureKey
}
