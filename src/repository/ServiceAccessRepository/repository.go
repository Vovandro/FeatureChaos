package ServiceAccessRepository

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
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
	t.cache = app.GetPkg(interfaces.PkgCache, names.CacheRedis).(redis.IRedis)
	t.db = app.GetPkg(interfaces.PkgDatabase, names.DatabasePrimary).(postgres.IPostgres)

	return nil
}

// Services CRUD
func (t *Repository) ListServices(c context.Context) []db.Service {
	rows, err := t.db.Query(c, `SELECT id, name FROM services ORDER BY name`)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	out := make([]db.Service, 0)
	for rows.Next() {
		var s db.Service
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

func (t *Repository) AddAccess(c context.Context, featureId uuid.UUID, serviceId uuid.UUID) error {
	if err := t.db.Exec(c, `INSERT INTO service_access(id, feature_id, service_id) VALUES($1,$2,$3) ON CONFLICT (feature_id, service_id) DO NOTHING`, uuid.New(), featureId, serviceId); err != nil {
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

func (t *Repository) GetAccess(c context.Context) ([]*db.ServiceAccess, error) {
	rows, err := t.db.Query(c, `SELECT service_access.id, feature_id, service_id, name FROM service_access JOIN services ON service_access.service_id = services.id`)
	if err != nil {
		t.logger.Error(c, err)
		return nil, err
	}
	defer rows.Close()
	out := make([]*db.ServiceAccess, 0)
	for rows.Next() {
		s := &db.ServiceAccess{}
		if err := rows.Scan(&s.ID, &s.FeatureId, &s.ServiceId, &s.Name); err != nil {
			t.logger.Error(c, err)
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func (t *Repository) GetAccessByFeatures(c context.Context, featureIds []uuid.UUID) (map[uuid.UUID][]*db.ServiceAccess, error) {
	// Short-circuit to avoid building an invalid IN () clause
	if len(featureIds) == 0 {
		return map[uuid.UUID][]*db.ServiceAccess{}, nil
	}

	// Build placeholders and args with strongly typed UUIDs to satisfy pgx encoders
	placeholders := make([]string, 0, len(featureIds))
	args := make([]any, 0, len(featureIds))
	for i := range featureIds {
		placeholders = append(placeholders, "$"+strconv.Itoa(i+1))
		args = append(args, featureIds[i])
	}

	query := `
    SELECT service_access.id, feature_id, service_id, name
    FROM service_access
    JOIN services ON service_access.service_id = services.id
    WHERE feature_id IN (` + strings.Join(placeholders, ",") + `)
    `

	rows, err := t.db.Query(c, query, args...)
	if err != nil {
		t.logger.Error(c, err)
		return nil, err
	}

	defer rows.Close()
	out := make(map[uuid.UUID][]*db.ServiceAccess)

	for rows.Next() {
		s := &db.ServiceAccess{}
		if err := rows.Scan(&s.ID, &s.FeatureId, &s.ServiceId, &s.Name); err != nil {
			t.logger.Error(c, err)
			continue
		}

		if _, ok := out[s.FeatureId]; !ok {
			out[s.FeatureId] = make([]*db.ServiceAccess, 0)
		}
		out[s.FeatureId] = append(out[s.FeatureId], s)
	}

	return out, nil
}
