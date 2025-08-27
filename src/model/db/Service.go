package db

import "github.com/google/uuid"

type Service struct {
	Id   uuid.UUID
	Name string
}
