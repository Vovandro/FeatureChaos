package ActivationValuesRepository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/cache/redis"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Repository struct {
	repository.Mock
	db     postgres.IPostgres
	cache  redis.IRedis
	logger interfaces.ILogger
}

type dbRunner interface {
	Exec(ctx context.Context, query string, args ...interface{}) error
	QueryRow(ctx context.Context, query string, args ...interface{}) (postgres.SQLRow, error)
}

type txRunner struct{ tx pgx.Tx }

func (t txRunner) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := t.tx.Exec(ctx, query, args...)
	return err
}
func (t txRunner) QueryRow(ctx context.Context, query string, args ...interface{}) (postgres.SQLRow, error) {
	return t.tx.QueryRow(ctx, query, args...), nil
}

const txCtxKey = "pgx_tx"

func (t *Repository) runner(ctx context.Context) dbRunner {
	if v := ctx.Value(txCtxKey); v != nil {
		if tx, ok := v.(pgx.Tx); ok && tx != nil {
			return txRunner{tx: tx}
		}
	}
	return t.db
}

func New(name string) *Repository {
	return &Repository{Mock: repository.Mock{NamePkg: name}}
}

func (t *Repository) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	t.logger = app.GetLogger()
	t.db = app.GetPkg(interfaces.PkgDatabase, "primary").(postgres.IPostgres)
	t.cache = app.GetPkg(interfaces.PkgCache, "primary").(redis.IRedis)
	return nil
}

func (t *Repository) InsertValue(c context.Context, featureId uuid.UUID, keyId *uuid.UUID, paramId *uuid.UUID, value int) (int64, error) {
	v, err := t.nextVersion(c, featureId)
	if err != nil {
		return 0, err
	}
	var key any
	var param any
	if keyId != nil {
		key = *keyId
	}
	if paramId != nil {
		param = *paramId
	}
	err = t.runner(c).Exec(c, `INSERT INTO activation_values (id, feature_id, activation_key_id, activation_param_id, value, deleted_at, v)
VALUES ($1, $2, $3, $4, $5, NULL, $6)`, uuid.New(), featureId, key, param, value, v)
	if err != nil {
		return 0, err
	}
	if bumpErr := t.bumpGlobalVersion(c); bumpErr != nil {
		t.logger.Error(c, fmt.Errorf("bump global version: %w", bumpErr))
	}
	return v, nil
}

func (t *Repository) DeleteByFeatureId(c context.Context, featureId uuid.UUID) error {
	return t.runner(c).Exec(c, `DELETE FROM activation_values WHERE feature_id = $1`, featureId)
}

func (t *Repository) DeleteByKeyId(c context.Context, keyId uuid.UUID) error {
	return t.runner(c).Exec(c, `DELETE FROM activation_values WHERE activation_key_id = $1`, keyId)
}

func (t *Repository) DeleteByParamId(c context.Context, paramId uuid.UUID) error {
	return t.runner(c).Exec(c, `DELETE FROM activation_values WHERE activation_param_id = $1`, paramId)
}

func (t *Repository) nextVersion(c context.Context, featureId uuid.UUID) (int64, error) {
	row, err := t.runner(c).QueryRow(c, `SELECT COALESCE(MAX(v), 0) FROM activation_values WHERE feature_id = $1`, featureId)
	if err != nil {
		return 0, err
	}
	var max int64
	if err := row.Scan(&max); err != nil {
		return 0, err
	}
	return max + 1, nil
}

func (t *Repository) bumpGlobalVersion(c context.Context) error {
	vStr, err := t.cache.Get(c, "feature_version")
	if err != nil || vStr == "" {
		return t.cache.Set(c, "feature_version", 1, 24*time.Hour)
	}
	var cur int
	_, _ = fmt.Sscanf(vStr, "%d", &cur)
	return t.cache.Set(c, "feature_version", cur+1, 24*time.Hour)
}
