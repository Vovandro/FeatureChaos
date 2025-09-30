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

func NewForTest(db postgres.IPostgres, cache redis.IRedis, logger interfaces.ILogger) *Repository {
	return &Repository{
		db:     db,
		cache:  cache,
		logger: logger,
	}
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
	SELECT av.feature_id, f.name, av.activation_key_id, ak.key, av.activation_param_id, ap.name, av.value, av.v, av.deleted_at
	FROM activation_values av
	JOIN service_access sa ON sa.feature_id = av.feature_id
	JOIN services s ON s.id = sa.service_id
	JOIN features f ON f.id = av.feature_id
	LEFT JOIN activation_keys ak ON ak.id = av.activation_key_id
	LEFT JOIN activation_params ap ON ap.id = av.activation_param_id
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

	// Aggregate preserving encounter order and defaulting absent values to -1
	type featureAgg struct {
		name      string
		value     int
		valueSet  bool
		isDeleted bool
	}
	type keyAgg struct {
		featureID uuid.UUID
		name      string
		value     int
		valueSet  bool
		isDeleted bool
		params    []dto.FeatureParam
	}

	featureOrder := make([]uuid.UUID, 0)
	featureById := make(map[uuid.UUID]*featureAgg)
	keysOrderByFeature := make(map[uuid.UUID][]uuid.UUID)
	keyById := make(map[uuid.UUID]*keyAgg)

	ensureFeature := func(id uuid.UUID, name string) *featureAgg {
		if f, ok := featureById[id]; ok {
			return f
		}
		f := &featureAgg{name: name}
		featureById[id] = f
		featureOrder = append(featureOrder, id)
		return f
	}
	ensureKey := func(featureID uuid.UUID, keyID uuid.UUID, keyName string) *keyAgg {
		if k, ok := keyById[keyID]; ok {
			return k
		}
		k := &keyAgg{featureID: featureID, name: keyName, params: make([]dto.FeatureParam, 0)}
		keyById[keyID] = k
		keysOrderByFeature[featureID] = append(keysOrderByFeature[featureID], keyID)
		return k
	}

	for _, v := range values {
		f := ensureFeature(v.FeatureID, v.FeatureName)
		// Mark feature deleted only if the feature-level row itself is deleted
		if v.KeyId == nil && v.ParamId == nil && v.DeletedAt != nil {
			f.isDeleted = true
		}

		if v.KeyId == nil {
			// Feature-level value
			f.value = v.Value
			f.valueSet = true
			continue
		}

		// Key-level or param-level
		keyID := *v.KeyId
		keyName := ""
		if v.KeyName != nil {
			keyName = *v.KeyName
		}
		k := ensureKey(v.FeatureID, keyID, keyName)
		// Mark key deleted only if the key-level row itself is deleted
		if v.ParamId == nil && v.DeletedAt != nil {
			k.isDeleted = true
		}

		if v.ParamId == nil {
			// key-level value
			k.value = v.Value
			k.valueSet = true
		} else {
			// param-level value
			paramName := ""
			if v.ParamName != nil {
				paramName = *v.ParamName
			}
			k.params = append(k.params, dto.FeatureParam{
				Id:        *v.ParamId,
				Name:      paramName,
				Value:     v.Value,
				IsDeleted: v.DeletedAt != nil,
			})
		}
	}

	// Build DTOs preserving order and filling defaults for missing levels
	result := make([]*dto.Feature, 0, len(featureOrder))
	for _, fid := range featureOrder {
		fAgg := featureById[fid]
		feat := &dto.Feature{
			ID:        fid,
			Name:      fAgg.name,
			Value:     fAgg.value,
			IsDeleted: fAgg.isDeleted,
		}
		if !fAgg.valueSet {
			feat.Value = -1
		}

		if order, ok := keysOrderByFeature[fid]; ok {
			feat.Keys = make([]dto.FeatureKey, 0, len(order))
			for _, kid := range order {
				kAgg := keyById[kid]
				keyDto := dto.FeatureKey{
					Id:        kid,
					Key:       kAgg.name,
					Value:     kAgg.value,
					IsDeleted: kAgg.isDeleted,
					Params:    kAgg.params,
				}
				if !kAgg.valueSet {
					keyDto.Value = -1
				}
				feat.Keys = append(feat.Keys, keyDto)
			}
		}
		result = append(result, feat)
	}

	return cachedVersion, result, nil
}
