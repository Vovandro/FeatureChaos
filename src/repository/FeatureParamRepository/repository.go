package FeatureParamRepository

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

func (t *Repository) GetFeatureParams(c context.Context, featureIds []uuid.UUID) []*db.FeatureParam {
	rows, err := t.db.Query(c, `
SELECT
    ap.id,
    ap.feature_id,
    ap.activation_id,
    ap.name,
    COALESCE(av.value, 0) AS value,
    COALESCE(av.v, 0)     AS v
FROM activation_params AS ap
LEFT JOIN activation_values AS av
    ON av.activation_param_id = ap.id
   AND av.deleted_at IS NULL
WHERE ap.feature_id = ANY($1)
`, featureIds)
	if err != nil {
		return nil
	}
	defer rows.Close()

	res := make([]*db.FeatureParam, 0)

	for rows.Next() {
		var item db.FeatureParam
		if err := rows.Scan(&item.Id, &item.FeatureId, &item.KeyId, &item.Key, &item.Value, &item.Version); err != nil {
			t.logger.Error(c, err)
			continue
		}
		res = append(res, &item)
	}

	// rows.Err() is not available in the wrapped driver; errors are surfaced via Scan above.

	return res
}

func (t *Repository) GetParamsByKeyId(c context.Context, keyId uuid.UUID) []*db.FeatureParam {
	rows, err := t.db.Query(c, `
SELECT
    ap.id,
    ap.feature_id,
    ap.activation_id,
    ap.name,
    COALESCE(av.value, 0) AS value,
    COALESCE(av.v, 0)     AS v
FROM activation_params AS ap
LEFT JOIN activation_values AS av
    ON av.activation_param_id = ap.id
   AND av.deleted_at IS NULL
WHERE ap.activation_id = $1
`, keyId)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	res := make([]*db.FeatureParam, 0)
	for rows.Next() {
		var item db.FeatureParam
		if err := rows.Scan(&item.Id, &item.FeatureId, &item.KeyId, &item.Key, &item.Value, &item.Version); err != nil {
			t.logger.Error(c, err)
			continue
		}
		res = append(res, &item)
	}
	return res
}

func (t *Repository) CreateParam(c context.Context, featureId uuid.UUID, keyId uuid.UUID, name string) (uuid.UUID, error) {
	id := uuid.New()
	err := t.runner(c).Exec(c, `INSERT INTO activation_params (id, feature_id, activation_id, name) VALUES ($1, $2, $3, $4)`, id, featureId, keyId, name)
	if err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}
	return id, nil
}

func (t *Repository) UpdateParam(c context.Context, paramId uuid.UUID, name string) error {
	err := t.runner(c).Exec(c, `UPDATE activation_params SET name = $2 WHERE id = $1`, paramId, name)
	if err != nil {
		t.logger.Error(c, err)
	}
	return err
}

func (t *Repository) DeleteParam(c context.Context, paramId uuid.UUID) error {
	err := t.runner(c).Exec(c, `DELETE FROM activation_params WHERE id = $1`, paramId)
	if err != nil {
		t.logger.Error(c, err)
	}
	return err
}

func (t *Repository) GetFeatureIdByParamId(c context.Context, paramId uuid.UUID) (uuid.UUID, error) {
	var featureId uuid.UUID
	row, err := t.runner(c).QueryRow(c, `SELECT feature_id FROM activation_params WHERE id = $1`, paramId)
	if err != nil {
		return uuid.Nil, err
	}
	if err := row.Scan(&featureId); err != nil {
		return uuid.Nil, err
	}
	return featureId, nil
}
