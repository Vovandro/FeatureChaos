package StatsService

import "context"

type Interface interface {
	SetStat(c context.Context, serviceName string, featureName string)
}
