package db

import (
	"time"

	"github.com/google/uuid"
)

type ActivationValues struct {
	FeatureID   uuid.UUID
	FeatureName string
	KeyId       *uuid.UUID
	KeyName     *string
	ParamId     *uuid.UUID
	ParamName   *string
	Value       int
	V           int64
	DeletedAt   *time.Time
}

type ActivationValuesFull struct {
	FeatureId          uuid.UUID
	FeatureName        string
	FeatureDescription string
	FeatureCreatedAt   time.Time
	FeatureUpdatedAt   time.Time
	KeyId              *uuid.UUID
	KeyName            *string
	ParamId            *uuid.UUID
	ParamName          *string
	Value              int
}
