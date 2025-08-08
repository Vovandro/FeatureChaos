package StatsRepository

import (
	"context"

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
	t.cache.Set(c, "stat:"+featureName+":"+serviceName, 1, 0)
}
