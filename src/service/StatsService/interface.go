package StatsService

import "context"

type Interface interface {
	SetStat(c context.Context, serviceName string, featureName string)
	IsUsed(c context.Context, featureName string) bool
	IsServiceUsed(c context.Context, serviceName string) bool
}
