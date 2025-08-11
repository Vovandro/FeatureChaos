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

// Services CRUD
func (t *Repository) ListServices(c context.Context) []Service {
	rows, err := t.db.Query(c, `SELECT id, name FROM services ORDER BY name`)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	out := make([]Service, 0)
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.Id, &s.Name); err != nil {
			t.logger.Error(c, err)
			continue
		}
		out = append(out, s)
	}
	return out
}

func (t *Repository) CreateService(c context.Context, name string) (uuid.UUID, error) {
	id := uuid.New()
	if err := t.db.Exec(c, `INSERT INTO services(id, name) VALUES($1,$2)`, id, name); err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}
	return id, nil
}

func (t *Repository) DeleteService(c context.Context, id uuid.UUID) error {
	if err := t.db.Exec(c, `DELETE FROM services WHERE id=$1`, id); err != nil {
		t.logger.Error(c, err)
		return err
	}
	return nil
}

// Feature-Service bindings
func (t *Repository) GetServicesByFeature(c context.Context, featureId uuid.UUID) []Service {
	rows, err := t.db.Query(c, `SELECT s.id, s.name FROM service_access sa JOIN services s ON s.id = sa.service_id WHERE sa.feature_id = $1 ORDER BY s.name`, featureId)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	out := make([]Service, 0)
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.Id, &s.Name); err != nil {
			t.logger.Error(c, err)
			continue
		}
		out = append(out, s)
	}
	return out
}

func (t *Repository) GetServiceById(c context.Context, id uuid.UUID) (Service, bool) {
	rows, err := t.db.Query(c, `SELECT id, name FROM services WHERE id=$1`, id)
	if err != nil {
		t.logger.Error(c, err)
		return Service{}, false
	}
	defer rows.Close()
	if rows.Next() {
		var s Service
		if err := rows.Scan(&s.Id, &s.Name); err != nil {
			t.logger.Error(c, err)
			return Service{}, false
		}
		return s, true
	}
	return Service{}, false
}

func (t *Repository) HasAnyAccessByService(c context.Context, id uuid.UUID) bool {
	rows, err := t.db.Query(c, `SELECT 1 FROM service_access WHERE service_id=$1 LIMIT 1`, id)
	if err != nil {
		t.logger.Error(c, err)
		return false
	}
	defer rows.Close()
	return rows.Next()
}

func (t *Repository) GetServicesByFeatureList(c context.Context, featureIds []uuid.UUID) map[uuid.UUID][]Service {
	rows, err := t.db.Query(c, `SELECT sa.feature_id, s.id, s.name FROM service_access sa JOIN services s ON s.id = sa.service_id WHERE sa.feature_id = ANY($1)`, featureIds)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	out := make(map[uuid.UUID][]Service)
	for rows.Next() {
		var fid uuid.UUID
		var s Service
		if err := rows.Scan(&fid, &s.Id, &s.Name); err != nil {
			t.logger.Error(c, err)
			continue
		}
		out[fid] = append(out[fid], s)
	}
	return out
}

func (t *Repository) AddAccess(c context.Context, featureId uuid.UUID, serviceId uuid.UUID) error {
	// Avoid relying on ON CONFLICT without a unique index; check existence first
	rows, err := t.db.Query(c, `SELECT 1 FROM service_access WHERE feature_id=$1 AND service_id=$2 LIMIT 1`, featureId, serviceId)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}
	defer rows.Close()
	if rows.Next() {
		// already linked
		return nil
	}
	id := uuid.New()
	if err := t.db.Exec(c, `INSERT INTO service_access(id, feature_id, service_id) VALUES($1,$2,$3)`, id, featureId, serviceId); err != nil {
		t.logger.Error(c, err)
		return err
	}
	return nil
}

func (t *Repository) RemoveAccess(c context.Context, featureId uuid.UUID, serviceId uuid.UUID) error {
	if err := t.db.Exec(c, `DELETE FROM service_access WHERE feature_id=$1 AND service_id=$2`, featureId, serviceId); err != nil {
		t.logger.Error(c, err)
		return err
	}
	return nil
}
