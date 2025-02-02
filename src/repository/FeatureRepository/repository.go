package FeatureRepository

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
	db     postgres.IPostgres
	logger interfaces.ILogger
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
    features.id, name, description, value, v 
FROM features
LEFT JOIN activation_values ON activation_values.feature_id = features.id
WHERE features.id IN ($1) 
  AND activation_key_id IS NULL 
  AND deleted_at IS NULL
`, ids)

	if err != nil {
		t.logger.Error(c, err)
		return nil
	}
	defer rows.Close()
	res := make([]*db.Feature, 0)

	for rows.Next() {
		var item db.Feature
		rows.Scan(&item.Id, &item.Name, &item.Description, &item.Value, &item.Version)
		res = append(res, &item)
	}

	return res
}
