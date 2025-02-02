package FeatureParamRepository

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

func (t *Repository) GetFeatureParams(c context.Context, featureIds []uuid.UUID) []*db.FeatureParam {
	rows, err := t.db.Query(c, `
SELECT 
    feature_params.id, feature_params.feature_id, feature_params.activation_id, name, value, v
FROM feature_params
LEFT JOIN activation_values ON activation_values.feature_param_id = feature_params.id
WHERE feature_params.feature_id IN ($1) 
  AND deleted_at IS NULL
`, featureIds)
	if err != nil {
		return nil
	}
	defer rows.Close()

	res := make([]*db.FeatureParam, 0)

	for rows.Next() {
		var item db.FeatureParam
		rows.Scan(&item.Id, &item.FeatureId, &item.KeyId, &item.Key, &item.Value, &item.Version)
		res = append(res, &item)
	}

	return res
}
