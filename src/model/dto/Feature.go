package dto

import (
	"time"

	"github.com/google/uuid"
)

type Feature struct {
	Id          uuid.UUID
	Name        string
	Description string
	Version     int64
	Value       int
	IsDeleted   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Keys        []FeatureKey
}
