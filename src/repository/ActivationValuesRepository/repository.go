package ActivationValuesRepository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/dto"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/paranoia/repository"
	"gitlab.com/devpro_studio/Paranoia/pkg/cache/redis"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
	"gitlab.com/devpro_studio/go_utils/dataUtils"
)

type Repository struct {
	repository.Mock
	db     postgres.IPostgres
	cache  redis.IRedis
	logger interfaces.ILogger
}

func New(name string) *Repository {
	return &Repository{Mock: repository.Mock{NamePkg: name}}
}

func (t *Repository) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	t.logger = app.GetLogger()
	t.db = app.GetPkg(interfaces.PkgDatabase, names.DatabasePrimary).(postgres.IPostgres)
	t.cache = app.GetPkg(interfaces.PkgCache, names.CacheRedis).(redis.IRedis)

	return nil
}

func (t *Repository) InsertValue(c context.Context, tx postgres.SQLTx, featureId uuid.UUID, keyId *uuid.UUID, paramId *uuid.UUID, value int) (int64, error) {
	v, err := t.nextVersion(c, tx)
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

	// Try to restore/update existing row (handles soft-deleted as well)
	var updatedId uuid.UUID
	row, err := tx.QueryRow(c, `
UPDATE activation_values
SET value = $4, deleted_at = NULL, v = $5
WHERE feature_id = $1
  AND activation_key_id IS NOT DISTINCT FROM $2
  AND activation_param_id IS NOT DISTINCT FROM $3
RETURNING id
`, featureId, key, param, value, v)
	if err != nil {
		return 0, err
	}
	if scanErr := row.Scan(&updatedId); scanErr == nil {
		if bumpErr := t.bumpGlobalVersion(c, v); bumpErr != nil {
			t.logger.Error(c, fmt.Errorf("bump global version: %w", bumpErr))
		}
		return v, nil
	}

	// If nothing was updated, insert a new row; ON CONFLICT covers races among active rows
	err = tx.Exec(c, `
INSERT INTO activation_values (id, feature_id, activation_key_id, activation_param_id, value, deleted_at, v)
VALUES ($1, $2, $3, $4, $5, NULL, $6)
ON CONFLICT (
    feature_id,
    COALESCE(activation_key_id, '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE(activation_param_id, '00000000-0000-0000-0000-000000000000'::uuid)
) WHERE (deleted_at IS NULL)
DO UPDATE SET value = EXCLUDED.value, deleted_at = NULL, v = EXCLUDED.v
`, uuid.New(), featureId, key, param, value, v)
	if err != nil {
		return 0, err
	}

	if bumpErr := t.bumpGlobalVersion(c, v); bumpErr != nil {
		t.logger.Error(c, fmt.Errorf("bump global version: %w", bumpErr))
	}

	return v, nil
}

func (t *Repository) DeleteByFeatureId(c context.Context, tx postgres.SQLTx, featureId uuid.UUID) error {
	v, err := t.nextVersion(c, tx)
	if err != nil {
		return err
	}

	err = tx.Exec(c, `UPDATE activation_values SET deleted_at = NOW(), v = $1 WHERE feature_id = $2`, v, featureId)
	if err != nil {
		return err
	}

	err = tx.Exec(c, `DELETE FROM activation_values WHERE feature_id = $1 AND activation_key_id IS NOT NULL`, featureId)
	if err != nil {
		return err
	}

	if bumpErr := t.bumpGlobalVersion(c, v); bumpErr != nil {
		t.logger.Error(c, fmt.Errorf("bump global version: %w", bumpErr))
	}

	return nil
}

func (t *Repository) DeleteByKeyId(c context.Context, tx postgres.SQLTx, keyId uuid.UUID) error {
	v, err := t.nextVersion(c, tx)
	if err != nil {
		return err
	}

	err = tx.Exec(c, `UPDATE activation_values SET deleted_at = NOW(), v = $1 WHERE activation_key_id = $2`, v, keyId)
	if err != nil {
		return err
	}

	err = tx.Exec(c, `DELETE FROM activation_values WHERE activation_key_id = $1 AND activation_param_id IS NOT NULL`, keyId)
	if err != nil {
		return err
	}

	if bumpErr := t.bumpGlobalVersion(c, v); bumpErr != nil {
		t.logger.Error(c, fmt.Errorf("bump global version: %w", bumpErr))
	}

	return nil
}

func (t *Repository) DeleteByParamId(c context.Context, tx postgres.SQLTx, paramId uuid.UUID) error {
	v, err := t.nextVersion(c, tx)
	if err != nil {
		return err
	}

	err = tx.Exec(c, `UPDATE activation_values SET deleted_at = NOW(), v = $1 WHERE activation_param_id = $2`, v, paramId)
	if err != nil {
		return err
	}

	if bumpErr := t.bumpGlobalVersion(c, v); bumpErr != nil {
		t.logger.Error(c, fmt.Errorf("bump global version: %w", bumpErr))
	}

	return nil
}

func (t *Repository) nextVersion(c context.Context, tx postgres.SQLTx) (int64, error) {
	row, err := tx.QueryRow(c, `SELECT COALESCE(MAX(v), 0) FROM activation_values`)
	if err != nil {
		return 0, err
	}
	var max int64
	if err := row.Scan(&max); err != nil {
		return 0, err
	}
	return max + 1, nil
}

func (t *Repository) bumpGlobalVersion(c context.Context, v int64) error {
	return t.cache.Set(c, "feature_version", v, 365*24*time.Hour)
}

func (t *Repository) GetNewByServiceName(c context.Context, serviceName string, lastVersion int64) (int64, []*dto.Feature, error) {
	cachedVersionStr, err := t.cache.Get(c, "feature_version")
	if err != nil {
		cachedVersionStr = "-1"
	}

	cachedVersion, _ := strconv.ParseInt(cachedVersionStr, 10, 64)

	if cachedVersion <= lastVersion {
		return cachedVersion, nil, nil
	}

	rows, err := t.db.Query(c, `
	SELECT av.feature_id, f.name, av.activation_key_id, ak.name, av.activation_param_id, ap.name, av.value, av.v, av.deleted_at
	FROM activation_values av
	JOIN service_access sa ON sa.feature_id = av.feature_id
	JOIN services s ON s.id = sa.service_id
	JOIN features f ON f.id = av.feature_id
	JOIN activation_keys ak ON ak.id = av.activation_key_id
	JOIN activation_params ap ON ap.id = av.activation_param_id
	WHERE s.name = $1 AND av.v > $2
`, serviceName, lastVersion)

	if err != nil {
		t.logger.Error(c, err)
		return lastVersion, nil, err
	}

	defer rows.Close()
	values := make([]db.ActivationValues, 0)

	for rows.Next() {
		var f db.ActivationValues
		if err := rows.Scan(&f.FeatureID, &f.FeatureName, &f.KeyId, &f.KeyName, &f.ParamId, &f.ParamName, &f.Value, &f.V, &f.DeletedAt); err != nil {
			t.logger.Error(c, err)
			continue
		}

		values = append(values, f)
	}

	params := make(map[uuid.UUID][]dto.FeatureParam)

	for _, v := range values {
		if v.ParamId != nil {
			if _, ok := params[*v.ParamId]; !ok {
				params[*v.ParamId] = make([]dto.FeatureParam, 0)
			}
			params[*v.ParamId] = append(params[*v.ParamId], dto.FeatureParam{
				Id:        *v.ParamId,
				Name:      *v.ParamName,
				Value:     v.Value,
				IsDeleted: v.DeletedAt != nil,
			})
		}
	}

	keys := make(map[uuid.UUID][]dto.FeatureKey)

	for _, v := range values {
		if v.KeyId != nil && v.ParamId == nil {
			if _, ok := keys[v.FeatureID]; !ok {
				keys[v.FeatureID] = make([]dto.FeatureKey, 0)
			}
			featureKey := dto.FeatureKey{
				Id:        *v.KeyId,
				Key:       *v.KeyName,
				Value:     int(v.Value),
				IsDeleted: v.DeletedAt != nil,
			}

			if _, ok := params[*v.KeyId]; ok {
				featureKey.Params = params[*v.KeyId]
			}

			keys[v.FeatureID] = append(keys[v.FeatureID], featureKey)
		}
	}

	features := make(map[uuid.UUID]*dto.Feature)

	for _, v := range values {
		if v.KeyId == nil {
			feature := &dto.Feature{
				ID:        v.FeatureID,
				Name:      v.FeatureName,
				Value:     v.Value,
				IsDeleted: v.DeletedAt != nil,
			}

			if _, ok := keys[v.FeatureID]; ok {
				feature.Keys = keys[v.FeatureID]
			}

			features[v.FeatureID] = feature
		}
	}

	return cachedVersion, dataUtils.MapValues(features), nil
}
