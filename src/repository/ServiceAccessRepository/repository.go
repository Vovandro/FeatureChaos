package ServiceAccessRepository

import (
	"context"
	"github.com/google/uuid"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/cache/redis"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
	"strconv"
	"time"
)

type Repository struct {
	repository.Mock
	logger interfaces.ILogger
	cache  redis.IRedis
	db     postgres.IPostgres
}

func New(name string) *Repository {
	return &Repository{
		Mock: repository.Mock{
			NamePkg: name,
		},
	}
}

func (t *Repository) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	t.logger = app.GetLogger()
	t.cache = app.GetPkg(interfaces.PkgCache, "primary").(redis.IRedis)
	t.db = app.GetPkg(interfaces.PkgDatabase, "primary").(postgres.IPostgres)
	return nil
}

func (t *Repository) GetNewByServiceName(c context.Context, serviceName string, lastVersion int64) []uuid.UUID {
	cachedVersionStr, err := t.cache.Get(c, "feature_version")
	if err != nil {
		cachedVersionStr = "-1"
	}
	checkedVersionStr, err := t.cache.Get(c, "feature_version_check:"+serviceName)
	if err != nil {
		checkedVersionStr = "-1"
	}

	cachedVersion, _ := strconv.ParseInt(cachedVersionStr, 10, 64)
	checkedVersion, _ := strconv.ParseInt(checkedVersionStr, 10, 64)

	if checkedVersion == lastVersion && cachedVersion <= checkedVersion {
		t.cache.Set(c, "feature_version_check:"+serviceName, cachedVersion, time.Hour)
		return nil
	}

	rows, err := t.db.Query(c, `
SELECT feature_id FROM services
LEFT JOIN service_access ON service_access.service_id = services.id
WHERE name = $1 AND version > $2
`, serviceName, lastVersion)

	if err != nil {
		t.logger.Error(c, err)
		return nil
	}

	defer rows.Close()
	res := make([]uuid.UUID, 0)

	for rows.Next() {
		var id uuid.UUID
		rows.Scan(&id)
		res = append(res, id)
	}

	t.cache.Set(c, "feature_version_check:"+serviceName, cachedVersion, time.Hour)

	return res
}
