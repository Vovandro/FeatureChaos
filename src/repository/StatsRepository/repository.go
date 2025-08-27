package StatsRepository

import (
	"context"
	"time"

	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/cache/redis"
)

type Repository struct {
	repository.Mock
	cache redis.IRedis
}

func New(name string) *Repository {
	return &Repository{
		Mock: repository.Mock{
			NamePkg: name,
		},
	}
}

func (t *Repository) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	t.cache = app.GetPkg(interfaces.PkgCache, names.CacheRedis).(redis.IRedis)

	return nil
}

func (t *Repository) SetStat(c context.Context, serviceName string, featureName string) {
	_ = t.cache.Set(c, "stat_used:"+featureName, 1, 30*time.Minute)
	_ = t.cache.Set(c, "stat_service_used:"+serviceName, 1, 30*time.Minute)
}

func (t *Repository) IsUsed(c context.Context, featureName string) bool {
	return t.cache.Has(c, "stat_used:"+featureName)
}

func (t *Repository) IsServiceUsed(c context.Context, serviceName string) bool {
	return t.cache.Has(c, "stat_service_used:"+serviceName)
}
