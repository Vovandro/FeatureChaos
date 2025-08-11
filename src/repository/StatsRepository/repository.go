package StatsRepository

import (
	"context"
	"time"

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
	t.cache = app.GetPkg(interfaces.PkgCache, "primary").(redis.IRedis)
	return nil
}

func (t *Repository) SetStat(c context.Context, serviceName string, featureName string) {
	// Mark per-service usage and also set a global used flag with TTL auto-refresh
	_ = t.cache.Set(c, "stat:"+featureName+":"+serviceName, 1, 0)
	// Global used flag to query quickly in UI. Refresh expiration to keep key around.
	_ = t.cache.Set(c, "stat_used:"+featureName, 1, 24*time.Hour)
	_ = t.cache.Set(c, "stat_service_used:"+serviceName, 1, 24*time.Hour)
}

func (t *Repository) IsUsed(c context.Context, featureName string) bool {
	if t.cache == nil {
		return false
	}
	if t.cache.Has(c, "stat_used:"+featureName) {
		// extend TTL so frequently visited UIs keep this warm
		_ = t.cache.Expire(c, "stat_used:"+featureName, 24*time.Hour)
		return true
	}
	return false
}

func (t *Repository) IsServiceUsed(c context.Context, serviceName string) bool {
	if t.cache == nil {
		return false
	}
	if t.cache.Has(c, "stat_service_used:"+serviceName) {
		_ = t.cache.Expire(c, "stat_service_used:"+serviceName, 24*time.Hour)
		return true
	}
	return false
}
