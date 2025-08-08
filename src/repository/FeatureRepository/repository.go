package FeatureRepository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Repository struct {
	repository.Mock
	db     postgres.IPostgres
	logger interfaces.ILogger
}

type dbRunner interface {
	Exec(ctx context.Context, query string, args ...interface{}) error
	Query(ctx context.Context, query string, args ...interface{}) (postgres.SQLRows, error)
}

type txRunner struct{ tx pgx.Tx }

func (t txRunner) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := t.tx.Exec(ctx, query, args...)
	return err
}
func (t txRunner) Query(ctx context.Context, query string, args ...interface{}) (postgres.SQLRows, error) {
	// wrap pgx.Rows into a type that implements Close() error if needed; here we fallback to non-tx for selects
	return nil, fmt.Errorf("tx runner query not supported in this repository")
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

func (t *Repository) GetFeaturesList(c context.Context, ids []uuid.UUID) []*db.Feature {
	rows, err := t.db.Query(c, `
SELECT
    f.id,
    f.name,
    f.description,
    COALESCE(av.value, 0) AS value,
    COALESCE(av.v, 0)     AS v
FROM features AS f
LEFT JOIN activation_values AS av
    ON av.feature_id = f.id
   AND av.activation_key_id IS NULL
   AND av.activation_param_id IS NULL
   AND av.deleted_at IS NULL
WHERE f.id = ANY($1)
`, ids)

	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	res := make([]*db.Feature, 0)

	for rows.Next() {
		var item db.Feature
		if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.Value, &item.Version); err != nil {
			t.logger.Error(c, err)
			continue
		}
		res = append(res, &item)
	}

	// rows.Err() is not available in the wrapped driver; errors are surfaced via Scan above.

	return res
}

func (t *Repository) ListFeatures(c context.Context) []*db.Feature {
	rows, err := t.db.Query(c, `
SELECT
    f.id,
    f.name,
    f.description,
    COALESCE(av.value, 0) AS value,
    COALESCE(av.v, 0)     AS v
FROM features AS f
LEFT JOIN activation_values AS av
    ON av.feature_id = f.id
   AND av.activation_key_id IS NULL
   AND av.activation_param_id IS NULL
   AND av.deleted_at IS NULL
`)

	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	res := make([]*db.Feature, 0)

	for rows.Next() {
		var item db.Feature
		if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.Value, &item.Version); err != nil {
			t.logger.Error(c, err)
			continue
		}
		res = append(res, &item)
	}

	return res
}

func (t *Repository) CreateFeature(c context.Context, name string, description string) (uuid.UUID, error) {
	id := uuid.New()
	err := t.runner(c).Exec(c, `INSERT INTO features (id, name, description) VALUES ($1, $2, $3)`, id, name, description)
	if err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}
	return id, nil
}

func (t *Repository) UpdateFeature(c context.Context, id uuid.UUID, name string, description string) error {
	err := t.runner(c).Exec(c, `UPDATE features SET name = $2, description = $3 WHERE id = $1`, id, name, description)
	if err != nil {
		t.logger.Error(c, err)
	}
	return err
}

func (t *Repository) DeleteFeature(c context.Context, id uuid.UUID) error {
	err := t.runner(c).Exec(c, `DELETE FROM features WHERE id = $1`, id)
	if err != nil {
		t.logger.Error(c, err)
	}
	return err
}
