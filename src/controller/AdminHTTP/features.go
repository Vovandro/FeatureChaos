package AdminHTTP

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

func (t *Controller) listFeatures(c context.Context, ctx httpSrv.ICtx) {
	items := t.features.ListFeatures(c)
	access, err := t.access.GetAccess(c)
	allKeys := t.keys.ListAllKeys(c)
	allParams := t.params.ListAllParams(c)

	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var svcMap map[uuid.UUID][]*db.ServiceAccess
	if t.access != nil && len(access) > 0 {
		svcMap = make(map[uuid.UUID][]*db.ServiceAccess)
		for _, a := range access {
			svcMap[a.FeatureId] = append(svcMap[a.FeatureId], a)
		}
	}

	type resp struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Value       int    `json:"value"`
		Version     int64  `json:"version"`
		Used        bool   `json:"used"`
		Services    []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"services"`
		Keys []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Value  int    `json:"value"`
			Params []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Value int    `json:"value"`
			} `json:"params"`
		} `json:"keys"`
	}
	out := make([]resp, 0, len(items))
	for _, it := range items {
		used := false
		if t.stats != nil {
			used = t.stats.IsUsed(context.Background(), it.Name)
		}
		svcs, ok := svcMap[it.Id]
		if !ok {
			svcs = make([]*db.ServiceAccess, 0)
		}
		svcResp := make([]struct {
			ID   string "json:\"id\""
			Name string "json:\"name\""
		}, len(svcs))

		for i, s := range svcs {
			svcResp[i] = struct {
				ID   string "json:\"id\""
				Name string "json:\"name\""
			}{ID: s.ServiceId.String(), Name: s.Name}
		}

		keys, ok := allKeys[it.Id]
		if !ok {
			keys = make([]*db.FeatureKey, 0)
		}
		keyResp := make([]struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Value  int    `json:"value"`
			Params []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Value int    `json:"value"`
			} `json:"params"`
		}, len(keys))

		for i, k := range keys {
			params, ok := allParams[k.Id]
			if !ok {
				params = make([]*db.FeatureParam, 0)
			}
			paramResp := make([]struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Value int    `json:"value"`
			}, len(params))

			for j, p := range params {
				paramResp[j] = struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Value int    `json:"value"`
				}{ID: p.Id.String(), Name: p.Name, Value: p.Value}
			}

			keyResp[i] = struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Value  int    `json:"value"`
				Params []struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Value int    `json:"value"`
				} `json:"params"`
			}{ID: k.Id.String(), Name: k.Key, Value: k.Value, Params: paramResp}
		}
		out = append(out, resp{ID: it.Id.String(), Name: it.Name, Description: it.Description, Value: it.Value, Version: it.Version, Used: used, Services: svcResp, Keys: keyResp})
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
