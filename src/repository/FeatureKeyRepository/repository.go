package FeatureKeyRepository

import (
	"context"
	"github.com/google/uuid"
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
    feature_keys.id, feature_keys.feature_id, key, description, value, v
FROM feature_keys
LEFT JOIN activation_values ON activation_values.id = feature_keys.activation_key_id
WHERE feature_keys.id IN ($1)
  AND activation_param_id IS NOT NULL
  AND deleted_at IS NULL
`, featureIds)
	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	res := make([]*db.FeatureKey, 0)

	for rows.Next() {
		var item db.FeatureKey
		rows.Scan(&item.Id, &item.FeatureId, &item.Key, &item.Description, &item.Value, &item.Version)
		res = append(res, &item)
	}

	return res
}
