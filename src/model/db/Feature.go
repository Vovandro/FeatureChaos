package db

import "github.com/google/uuid"

type Feature struct {
	Id          uuid.UUID
	Name        string
	Description string
	Value       int
	Version     int64
}
