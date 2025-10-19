package AdminHTTP

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

func (t *Controller) listFeatures(c context.Context, ctx httpSrv.ICtx) {
	var req GetFeaturesRequest
	if err := req.FromRequest(ctx); err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	items, count, err := t.activationValues.GetFeatures(c, req.ServiceId, req.Page, t.config.PageSize, req.Find, req.IsDeprecated, t.config.DeprecatedTime)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	totalPages := (count + t.config.PageSize - 1) / t.config.PageSize

	if len(items) == 0 {
		respondJSON(ctx, http.StatusOK, GetFeaturesResponse{
			Page:       req.Page,
			TotalPages: totalPages,
			Features:   make([]Feature, 0),
		})
		return
	}

	features := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		features = append(features, item.Id)
	}

	access, err := t.access.GetAccessByFeatures(c, features)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	out := GetFeaturesResponse{
		Page:       req.Page,
		TotalPages: totalPages,
		Features:   make([]Feature, 0, len(items)),
	}

	for _, it := range items {
		used := false
		if t.stats != nil {
			used = t.stats.IsUsed(context.Background(), it.Name)
		}

		svcResp := make([]Service, 0)

		if svcs, ok := access[it.Id]; ok {
			for _, svc := range svcs {
				svcResp = append(svcResp, Service{
					ID:   svc.ServiceId.String(),
					Name: svc.Name,
				})
			}
		}

		keyResp := make([]Key, 0)
		for _, key := range it.Keys {
			k := Key{
				ID:     key.Id.String(),
				Name:   key.Key,
				Value:  key.Value,
				Params: make([]Param, 0),
			}

			for _, param := range key.Params {
				k.Params = append(k.Params, Param{
					ID:    param.Id.String(),
					Name:  param.Name,
					Value: param.Value,
				})
			}

			keyResp = append(keyResp, k)
		}

		out.Features = append(out.Features, Feature{
			ID:           it.Id.String(),
			Name:         it.Name,
			Description:  it.Description,
			Value:        it.Value,
			Used:         used,
			Services:     svcResp,
			Keys:         keyResp,
			IsDeprecated: time.Since(it.UpdatedAt) > t.config.DeprecatedTime,
			CreatedAt:    it.CreatedAt,
			UpdatedAt:    it.UpdatedAt,
		})
	}

	respondJSON(ctx, http.StatusOK, out)
}

type featureCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Value       int    `json:"value"`
}

func (t *Controller) createFeature(c context.Context, ctx httpSrv.ICtx) {
	var req featureCreateReq
	if err := parseJSON(ctx, &req); err != nil || req.Name == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id, err := t.features.CreateFeature(c, req.Name, req.Description, req.Value)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"id": id.String()})
}

type featureUpdateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Value       int    `json:"value"`
}

func (t *Controller) updateFeature(c context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req featureUpdateReq
	if err := parseJSON(ctx, &req); err != nil || req.Name == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := t.features.UpdateFeature(c, id, req.Name, req.Description, req.Value); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusOK, map[string]string{"status": "ok"})
}

func (t *Controller) deleteFeature(c context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := t.features.DeleteFeature(c, id); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}
