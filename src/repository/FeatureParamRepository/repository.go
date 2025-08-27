package FeatureParamRepository

import (
	"context"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ActivationValuesRepository"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
)

type Repository struct {
	repository.Mock
	logger                     interfaces.ILogger
	db                         postgres.IPostgres
	activationValuesRepository ActivationValuesRepository.Interface
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

	return nil
}

func (t *Repository) ListAllParams(c context.Context) map[uuid.UUID][]*db.FeatureParam {
	rows, err := t.db.Query(c, `
		SELECT
			ap.id,
			ap.name,
			ap.activation_id,
			av.value AS value
		FROM activation_params AS ap
		LEFT JOIN activation_values AS av ON av.activation_param_id = ap.id
		WHERE ap.deleted_at IS NULL`)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	res := make(map[uuid.UUID][]*db.FeatureParam)
	for rows.Next() {
		var item db.FeatureParam
		if err := rows.Scan(&item.Id, &item.Name, &item.KeyId, &item.Value); err != nil {
			t.logger.Error(c, err)
			return nil
		}
		if _, ok := res[item.KeyId]; !ok {
			res[item.KeyId] = make([]*db.FeatureParam, 0)
		}
		res[item.KeyId] = append(res[item.KeyId], &item)
	}
	return res
}

func (t *Repository) ListParams(c context.Context, keyId uuid.UUID) []*db.FeatureParam {
	rows, err := t.db.Query(c, `
		SELECT
			ap.id,
			ap.name,
			av.value AS value
		FROM activation_params AS ap
		LEFT JOIN activation_values AS av ON av.activation_param_id = ap.id
		WHERE ap.deleted_at IS NULL
			AND ap.activation_id = $1`, keyId)

	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	res := make([]*db.FeatureParam, 0)
	for rows.Next() {
		var item db.FeatureParam
		if err := rows.Scan(&item.Id, &item.Name, &item.Value); err != nil {
			t.logger.Error(c, err)
			return nil
		}
		res = append(res, &item)
	}
	return res
}

func (t *Repository) CreateParam(c context.Context, featureId uuid.UUID, keyId uuid.UUID, name string, value int) (uuid.UUID, error) {
	tx, err := t.db.BeginTx(c)
	if err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	defer tx.Rollback(c)

	// Try to restore an existing soft-deleted param first
	row, err := tx.QueryRow(c, `
UPDATE activation_params
SET deleted_at = NULL
WHERE activation_id = $1 AND name = $2 AND deleted_at IS NOT NULL
RETURNING id
`, keyId, name)
	if err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	var id uuid.UUID
	if scanErr := row.Scan(&id); scanErr != nil {
		// No soft-deleted row restored; insert a new one
		newId := uuid.New()
		row, err = tx.QueryRow(c, `
INSERT INTO activation_params (id, feature_id, activation_id, name)
VALUES ($1, $2, $3, $4)
RETURNING id
`, newId, featureId, keyId, name)
		if err != nil {
			t.logger.Error(c, err)
			return uuid.Nil, err
		}
		if scanErr2 := row.Scan(&id); scanErr2 != nil {
			return uuid.Nil, scanErr2
		}
	}

	if _, err := t.activationValuesRepository.InsertValue(c, tx, featureId, &keyId, &id, value); err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	if err := tx.Commit(c); err != nil {
		t.logger.Error(c, err)
		return uuid.Nil, err
	}

	return id, nil
}

func (t *Repository) UpdateParam(c context.Context, featureId uuid.UUID, keyId uuid.UUID, paramId uuid.UUID, name string, value int) error {
	tx, err := t.db.BeginTx(c)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	defer tx.Rollback(c)

	err = tx.Exec(c, `UPDATE activation_params SET name = $2 WHERE id = $1 AND deleted_at IS NULL`, paramId, name)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	if _, err := t.activationValuesRepository.InsertValue(c, tx, featureId, &keyId, &paramId, value); err != nil {
		t.logger.Error(c, err)
		return err
	}

	return tx.Commit(c)
}

func (t *Repository) DeleteParam(c context.Context, paramId uuid.UUID) error {
	tx, err := t.db.BeginTx(c)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	defer tx.Rollback(c)

	err = tx.Exec(c, `UPDATE activation_params SET deleted_at = NOW() WHERE id = $1`, paramId)
	if err != nil {
		t.logger.Error(c, err)
		return err
	}

	if err := t.activationValuesRepository.DeleteByParamId(c, tx, paramId); err != nil {
		t.logger.Error(c, err)
		return err
	}
	if err := tx.Commit(c); err != nil {
		t.logger.Error(c, err)
		return err
	}

	return nil
}

func (t *Repository) DeleteAllByKeyId(c context.Context, tx postgres.SQLTx, keyId uuid.UUID) error {
	return tx.Exec(c, `DELETE FROM activation_params WHERE activation_id = $1`, keyId)
}

func (t *Repository) DeleteAllByFeatureId(c context.Context, tx postgres.SQLTx, featureId uuid.UUID) error {
	return tx.Exec(c, `DELETE FROM activation_params WHERE feature_id = $1`, featureId)
}
