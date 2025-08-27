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
