package ServiceAccessRepository

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/cache/redis"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
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
SELECT DISTINCT sa.feature_id
FROM services s
JOIN service_access sa ON sa.service_id = s.id
JOIN activation_values av ON av.feature_id = sa.feature_id
WHERE s.name = $1 AND av.v > $2 AND av.deleted_at IS NULL
`, serviceName, lastVersion)

	if err != nil {
		t.logger.Error(c, err)
		return nil
	}

	defer rows.Close()
	res := make([]uuid.UUID, 0)

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.logger.Error(c, err)
			continue
		}
		res = append(res, id)
	}

	t.cache.Set(c, "feature_version_check:"+serviceName, cachedVersion, time.Hour)

	return res
}
