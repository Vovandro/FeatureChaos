package FeatureKeyRepository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Repository struct {
	repository.Mock
	logger interfaces.ILogger
	db     postgres.IPostgres
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
	return &Repository{
		Mock: repository.Mock{
			NamePkg: name,
		},
	}
}

func (t *Repository) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	t.logger = app.GetLogger()
	t.db = app.GetPkg(interfaces.PkgDatabase, "primary").(postgres.IPostgres)
	return nil
}

func (t *Repository) GetFeatureKeys(c context.Context, featureIds []uuid.UUID) []*db.FeatureKey {
	rows, err := t.db.Query(c, `
SELECT
    ak.id,
    ak.feature_id,
    ak.key,
    ak.description,
    COALESCE(vv.value, 0) AS value,
    COALESCE(vv.v, 0)     AS v
FROM activation_keys AS ak
LEFT JOIN LATERAL (
    SELECT value, v
    FROM activation_values av
    WHERE av.activation_key_id = ak.id
      AND av.activation_param_id IS NULL
      AND av.deleted_at IS NULL
    ORDER BY v DESC
    LIMIT 1
) vv ON true
WHERE ak.feature_id = ANY($1)
`, featureIds)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	res := make([]*db.FeatureKey, 0)

	for rows.Next() {
		var item db.FeatureKey
		if err := rows.Scan(&item.Id, &item.FeatureId, &item.Key, &item.Description, &item.Value, &item.Version); err != nil {
			t.logger.Error(c, err)
			continue
		}
		res = append(res, &item)
	}

	// rows.Err() is not available in the wrapped driver; errors are surfaced via Scan above.

	return res
}

func (t *Repository) CreateKey(c context.Context, featureId uuid.UUID, key string, description string) (uuid.UUID, error) {
	id := uuid.New()
	err := t.runner(c).Exec(c, `INSERT INTO activation_keys (id, feature_id, key, description) VALUES ($1, $2, $3, $4)`, id, featureId, key, description)
	if err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}
	return id, nil
}

func (t *Repository) UpdateKey(c context.Context, keyId uuid.UUID, key string, description string) error {
	err := t.runner(c).Exec(c, `UPDATE activation_keys SET key = $2, description = $3 WHERE id = $1`, keyId, key, description)
	if err != nil {
		t.logger.Error(c, err)
	}
	return err
}

func (t *Repository) DeleteKey(c context.Context, keyId uuid.UUID) error {
	err := t.runner(c).Exec(c, `DELETE FROM activation_keys WHERE id = $1`, keyId)
	if err != nil {
		t.logger.Error(c, err)
	}
	return err
}

func (t *Repository) GetFeatureIdByKeyId(c context.Context, keyId uuid.UUID) (uuid.UUID, error) {
	var featureId uuid.UUID
	row, err := t.runner(c).QueryRow(c, `SELECT feature_id FROM activation_keys WHERE id = $1`, keyId)
	if err != nil {
		return uuid.Nil, err
	}
	if err := row.Scan(&featureId); err != nil {
		return uuid.Nil, err
	}
	return featureId, nil
}

func (t *Repository) DeleteByFeatureId(c context.Context, featureId uuid.UUID) error {
	return t.runner(c).Exec(c, `DELETE FROM activation_keys WHERE feature_id = $1`, featureId)
}
