package FeatureRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ActivationValuesRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureKeyRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Repository struct {
	repository.Mock
	db                         postgres.IPostgres
	logger                     interfaces.ILogger
	activationValuesRepository ActivationValuesRepository.Interface
	featureParamRepository     FeatureParamRepository.Interface
	featureKeyRepository       FeatureKeyRepository.Interface
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
	t.activationValuesRepository = app.GetModule(interfaces.ModuleRepository, names.ActivationValuesRepository).(ActivationValuesRepository.Interface)
	t.featureParamRepository = app.GetModule(interfaces.ModuleRepository, names.FeatureParamRepository).(FeatureParamRepository.Interface)
	t.featureKeyRepository = app.GetModule(interfaces.ModuleRepository, names.FeatureKeyRepository).(FeatureKeyRepository.Interface)

	return nil
}

func (t *Repository) GetFeatureName(c context.Context, id uuid.UUID) (string, error) {
	row, err := t.db.QueryRow(c, `
SELECT
    f.name
FROM features AS f
WHERE f.id = $1
  AND f.deleted_at IS NULL
`, id)

	if err != nil {
		t.logger.Error(c, err)
		return "", err
	}

	var name string
	if err := row.Scan(&name); err != nil {
		t.logger.Error(c, err)
		return "", err
	}

	return name, nil
}

func (t *Repository) ListFeatures(c context.Context) []*db.Feature {
	rows, err := t.db.Query(c, `
SELECT
    f.id,
    f.name,
    f.description,
    av.value AS value,
    av.v     AS v
FROM features AS f
JOIN activation_values av ON av.feature_id = f.id
WHERE av.activation_key_id IS NULL
  AND f.deleted_at IS NULL
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

func (t *Repository) CreateFeature(c context.Context, name string, description string, value int) (uuid.UUID, error) {
	tx, err := t.db.BeginTx(c)
	if err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	defer tx.Rollback(c)

	// Try to restore an existing soft-deleted feature first
	row, err := tx.QueryRow(c, `
UPDATE features
SET description = $2, deleted_at = NULL
WHERE name = $1 AND deleted_at IS NOT NULL
RETURNING id
`, name, description)

	if err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	var id uuid.UUID
	if scanErr := row.Scan(&id); scanErr != nil {
		// No soft-deleted row restored; insert a new one
		newId := uuid.New()
		row, err = tx.QueryRow(c, `
INSERT INTO features (id, name, description)
VALUES ($1, $2, $3)
RETURNING id
`, newId, name, description)
		if err != nil {
			t.logger.Error(c, err)
			return uuid.Nil, err
		}
		if scanErr2 := row.Scan(&id); scanErr2 != nil {
			return uuid.Nil, scanErr2
		}
	}

	if _, err := t.activationValuesRepository.InsertValue(c, tx, id, nil, nil, value); err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	if err := tx.Commit(c); err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	return id, nil
}

func (t *Repository) UpdateFeature(c context.Context, id uuid.UUID, name string, description string, value int) error {
	tx, err := t.db.BeginTx(c)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	defer tx.Rollback(c)

	err = tx.Exec(c, `UPDATE features SET name = $2, description = $3 WHERE id = $1 AND deleted_at IS NULL`, id, name, description)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	if _, err := t.activationValuesRepository.InsertValue(c, tx, id, nil, nil, value); err != nil {
		t.logger.Error(c, err)
		return err
	}

	if err := tx.Commit(c); err != nil {
		t.logger.Error(c, err)
		return err
	}

	return nil
}

func (t *Repository) DeleteFeature(c context.Context, id uuid.UUID) error {
	tx, err := t.db.BeginTx(c)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	defer tx.Rollback(c)

	err = tx.Exec(c, `UPDATE features SET deleted_at = NOW() WHERE id = $1`, id)
	if err != nil {
		t.logger.Error(c, err)
	}

	if err := t.activationValuesRepository.DeleteByFeatureId(c, tx, id); err != nil {
		t.logger.Error(c, err)
		return err
	}

	if err := t.featureParamRepository.DeleteAllByFeatureId(c, tx, id); err != nil {
		t.logger.Error(c, err)
		return err
	}

	if err := t.featureKeyRepository.DeleteAllByFeatureId(c, tx, id); err != nil {
		t.logger.Error(c, err)
		return err
	}

	if err := tx.Commit(c); err != nil {
		t.logger.Error(c, err)
		return err
	}

	return nil
}
