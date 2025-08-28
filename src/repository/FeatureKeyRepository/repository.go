package FeatureKeyRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ActivationValuesRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Repository struct {
	repository.Mock
	logger                     interfaces.ILogger
	db                         postgres.IPostgres
	activationValuesRepository ActivationValuesRepository.Interface
	featureParamRepository     FeatureParamRepository.Interface
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
	t.db = app.GetPkg(interfaces.PkgDatabase, names.DatabasePrimary).(postgres.IPostgres)
	t.activationValuesRepository = app.GetModule(interfaces.ModuleRepository, names.ActivationValuesRepository).(ActivationValuesRepository.Interface)
	t.featureParamRepository = app.GetModule(interfaces.ModuleRepository, names.FeatureParamRepository).(FeatureParamRepository.Interface)

	return nil
}

func (t *Repository) ListAllKeys(c context.Context) map[uuid.UUID][]*db.FeatureKey {
	rows, err := t.db.Query(c, `
		SELECT
			ak.id,
			ak.feature_id,
			ak.key,
			ak.description,
			av.value AS value
		FROM activation_keys AS ak
		LEFT JOIN activation_values AS av ON av.activation_key_id = ak.id
			AND av.activation_param_id IS NULL
		WHERE ak.deleted_at IS NULL`)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}

	defer rows.Close()
	res := make(map[uuid.UUID][]*db.FeatureKey)

	for rows.Next() {
		var item db.FeatureKey
		if err := rows.Scan(&item.Id, &item.FeatureId, &item.Key, &item.Description, &item.Value); err != nil {
			t.logger.Error(c, err)
			return nil
		}

		if _, ok := res[item.FeatureId]; !ok {
			res[item.FeatureId] = make([]*db.FeatureKey, 0)
		}

		res[item.FeatureId] = append(res[item.FeatureId], &item)
	}

	return res
}

func (t *Repository) ListKeys(c context.Context, featureId uuid.UUID) []*db.FeatureKey {
	rows, err := t.db.Query(c, `
		SELECT
			ak.id,
			ak.key,
			ak.description,
			av.value AS value
		FROM activation_keys AS ak
		LEFT JOIN activation_values AS av ON av.activation_key_id = ak.id
			AND av.activation_param_id IS NULL
		WHERE ak.feature_id = $1 AND ak.deleted_at IS NULL`, featureId)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}

	defer rows.Close()
	res := make([]*db.FeatureKey, 0)
	for rows.Next() {
		var item db.FeatureKey
		if err := rows.Scan(&item.Id, &item.Key, &item.Description, &item.Value); err != nil {
			t.logger.Error(c, err)
			return nil
		}
		res = append(res, &item)
	}

	return res
}

func (t *Repository) CreateKey(c context.Context, featureId uuid.UUID, key string, description string, value int) (uuid.UUID, error) {
	tx, err := t.db.BeginTx(c)
	if err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	defer tx.Rollback(c)

	// Try to restore an existing soft-deleted key first
	row, err := tx.QueryRow(c, `
UPDATE activation_keys
SET description = $3, deleted_at = NULL
WHERE feature_id = $1 AND key = $2 AND deleted_at IS NOT NULL
RETURNING id
`, featureId, key, description)

	if err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	var id uuid.UUID
	if scanErr := row.Scan(&id); scanErr != nil {
		// No soft-deleted row restored; insert a new one
		newId := uuid.New()
		row, err = tx.QueryRow(c, `
INSERT INTO activation_keys (id, feature_id, key, description)
VALUES ($1, $2, $3, $4)
RETURNING id
`, newId, featureId, key, description)
		if err != nil {
			t.logger.Error(c, err)
			return uuid.Nil, err
		}

		if scanErr2 := row.Scan(&id); scanErr2 != nil {
			return uuid.Nil, scanErr2
		}
	}

	if _, err := t.activationValuesRepository.InsertValue(c, tx, featureId, &id, nil, value); err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	if err := tx.Commit(c); err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	return id, nil
}

func (t *Repository) UpdateKey(c context.Context, featureId uuid.UUID, keyId uuid.UUID, key string, description string, value int) error {
	tx, err := t.db.BeginTx(c)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	defer tx.Rollback(c)

	err = tx.Exec(c, `UPDATE activation_keys SET key = $2, description = CASE WHEN $3 = '' THEN description ELSE $3 END WHERE id = $1 AND deleted_at IS NULL`, keyId, key, description)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	if _, err := t.activationValuesRepository.InsertValue(c, tx, featureId, &keyId, nil, value); err != nil {
		t.logger.Error(c, err)
		return err
	}

	return tx.Commit(c)
}

func (t *Repository) DeleteKey(c context.Context, keyId uuid.UUID) error {
	tx, err := t.db.BeginTx(c)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	defer tx.Rollback(c)

	err = tx.Exec(c, `UPDATE activation_keys SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, keyId)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	if err := t.activationValuesRepository.DeleteByKeyId(c, tx, keyId); err != nil {
		t.logger.Error(c, err)
		return err
	}

	if err := t.featureParamRepository.DeleteAllByKeyId(c, tx, keyId); err != nil {
		t.logger.Error(c, err)
		return err
	}

	if err := tx.Commit(c); err != nil {
		t.logger.Error(c, err)
		return err
	}

	return nil
}

func (t *Repository) DeleteAllByFeatureId(c context.Context, tx postgres.SQLTx, featureId uuid.UUID) error {
	return tx.Exec(c, `DELETE FROM activation_keys WHERE feature_id = $1`, featureId)
}
