package AdminHTTP

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	FeatureKeyRepository "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureKeyRepository"
	FeatureParamRepository "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	FeatureRepository "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureRepository"
	ServiceAccessRepository "gitlab.com/devpro_studio/FeatureChaos/src/repository/ServiceAccessRepository"
	StatsService "gitlab.com/devpro_studio/FeatureChaos/src/service/StatsService"
	"gitlab.com/devpro_studio/Paranoia/paranoia/controller"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

//go:embed templates/index.html
var tplIndexHTML string

//go:embed templates/index.js
var tplIndexJS string

type Controller struct {
	controller.Mock
	features FeatureRepository.Interface
	keys     FeatureKeyRepository.Interface
	params   FeatureParamRepository.Interface
	stats    StatsService.Interface
	access   ServiceAccessRepository.Interface
}

func New(name string) *Controller {
	return &Controller{Mock: controller.Mock{NamePkg: name}}
}

func (t *Controller) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	t.features = app.GetModule(interfaces.ModuleRepository, names.FeatureRepository).(FeatureRepository.Interface)
	t.keys = app.GetModule(interfaces.ModuleRepository, names.FeatureKeyRepository).(FeatureKeyRepository.Interface)
	t.params = app.GetModule(interfaces.ModuleRepository, names.FeatureParamRepository).(FeatureParamRepository.Interface)
	t.stats = app.GetModule(interfaces.ModuleService, names.StatsService).(StatsService.Interface)
	t.access = app.GetModule(interfaces.ModuleRepository, names.ServiceAccessRepository).(ServiceAccessRepository.Interface)
	http := app.GetPkg(interfaces.PkgServer, names.HttpServer).(httpSrv.IHttp)

	// static
	http.PushRoute("GET", "/", t.indexPage, nil)
	http.PushRoute("GET", "/index.js", t.indexJS, nil)

	// features
	http.PushRoute("GET", "/api/features", t.listFeatures, nil)
	http.PushRoute("POST", "/api/features", t.createFeature, nil)
	http.PushRoute("PUT", "/api/features/{id}", t.updateFeature, nil)
	http.PushRoute("DELETE", "/api/features/{id}", t.deleteFeature, nil)

	// services
	http.PushRoute("GET", "/api/services", t.listServices, nil)
	http.PushRoute("POST", "/api/services", t.createService, nil)
	http.PushRoute("DELETE", "/api/services/{id}", t.deleteService, nil)
	http.PushRoute("POST", "/api/features/{id}/services/{sid}", t.addFeatureService, nil)
	http.PushRoute("DELETE", "/api/features/{id}/services/{sid}", t.removeFeatureService, nil)

	// keys
	http.PushRoute("POST", "/api/features/{id}/keys", t.createKey, nil)
	http.PushRoute("PUT", "/api/keys/{id}", t.updateKey, nil)
	http.PushRoute("DELETE", "/api/keys/{id}", t.deleteKey, nil)

	// params
	http.PushRoute("POST", "/api/keys/{id}/params", t.createParam, nil)
	http.PushRoute("PUT", "/api/params/{id}", t.updateParam, nil)
	http.PushRoute("DELETE", "/api/params/{id}", t.deleteParam, nil)

	return nil
}

func respondJSON(ctx httpSrv.ICtx, status int, v any) {
	b, _ := json.Marshal(v)
	ctx.GetResponse().Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.GetResponse().SetStatus(status)
	ctx.GetResponse().SetBody(b)
}

func respondHTML(ctx httpSrv.ICtx, status int, s string) {
	ctx.GetResponse().Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.GetResponse().SetStatus(status)
	ctx.GetResponse().SetBody([]byte(s))
}

func respondJS(ctx httpSrv.ICtx, status int, s string) {
	ctx.GetResponse().Header().Set("Content-Type", "text/javascript; charset=utf-8")
	ctx.GetResponse().SetStatus(status)
	ctx.GetResponse().SetBody([]byte(s))
}

func parseJSON[T any](ctx httpSrv.ICtx, out *T) error {
	defer ctx.GetRequest().GetBody().Close()
	dec := json.NewDecoder(ctx.GetRequest().GetBody())
	return dec.Decode(out)
}

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

// Services CRUD endpoints
func (t *Controller) listServices(c context.Context, ctx httpSrv.ICtx) {
	svcs := t.access.ListServices(c)
	type resp struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
	}
	out := make([]resp, len(svcs))
	for i, s := range svcs {
		active := false
		if t.stats != nil {
			active = t.stats.IsServiceUsed(c, s.Name)
		}
		out[i] = resp{ID: s.Id.String(), Name: s.Name, Active: active}
	}
	respondJSON(ctx, http.StatusOK, out)
}

func (t *Controller) createService(c context.Context, ctx httpSrv.ICtx) {
	var body struct {
		Name string `json:"name"`
	}
	if err := parseJSON(ctx, &body); err != nil || body.Name == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id, err := t.access.CreateService(c, body.Name)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"id": id.String()})
}

func (t *Controller) deleteService(c context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := t.access.DeleteService(c, id); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}

// Feature-service binding endpoints
func (t *Controller) addFeatureService(c context.Context, ctx httpSrv.ICtx) {
	fidStr := ctx.GetRouterValue("id")
	sidStr := ctx.GetRouterValue("sid")
	fid, err := uuid.Parse(fidStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid feature id"})
		return
	}
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid service id"})
		return
	}
	if err := t.access.AddAccess(c, fid, sid); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"status": "ok"})
}

func (t *Controller) removeFeatureService(c context.Context, ctx httpSrv.ICtx) {
	fidStr := ctx.GetRouterValue("id")
	sidStr := ctx.GetRouterValue("sid")
	fid, err := uuid.Parse(fidStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid feature id"})
		return
	}
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid service id"})
		return
	}
	if err := t.access.RemoveAccess(c, fid, sid); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}

type keyCreateReq struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Value       int    `json:"value"`
}

func (t *Controller) createKey(c context.Context, ctx httpSrv.ICtx) {
	featureIdStr := ctx.GetRouterValue("id")
	featureId, err := uuid.Parse(featureIdStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid feature id"})
		return
	}
	var req keyCreateReq
	if err := parseJSON(ctx, &req); err != nil || req.Key == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id, err := t.keys.CreateKey(c, featureId, req.Key, req.Description, req.Value)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"id": id.String()})
}

type keyUpdateReq struct {
	FeatureId   uuid.UUID `json:"feature_id"`
	Key         string    `json:"key"`
	Description string    `json:"description"`
	Value       int       `json:"value"`
}

func (t *Controller) updateKey(c context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req keyUpdateReq
	if err := parseJSON(ctx, &req); err != nil || req.Key == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := t.keys.UpdateKey(c, id, req.FeatureId, req.Key, req.Description, req.Value); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusOK, map[string]string{"status": "ok"})
}

func (t *Controller) deleteKey(c context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := t.keys.DeleteKey(c, id); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}

type paramCreateReq struct {
	FeatureId uuid.UUID `json:"feature_id"`
	Name      string    `json:"name"`
	Value     int       `json:"value"`
}

func (t *Controller) createParam(c context.Context, ctx httpSrv.ICtx) {
	keyIdStr := ctx.GetRouterValue("id")
	keyId, err := uuid.Parse(keyIdStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid key id"})
		return
	}
	var body paramCreateReq
	if err := parseJSON(ctx, &body); err != nil || body.Name == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id, err := t.params.CreateParam(c, body.FeatureId, keyId, body.Name, body.Value)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"id": id.String()})
}

func (t *Controller) updateParam(c context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req struct {
		FeatureId uuid.UUID `json:"feature_id"`
		KeyId     uuid.UUID `json:"key_id"`
		Name      string    `json:"name"`
		Value     int       `json:"value"`
	}
	if err := parseJSON(ctx, &req); err != nil || req.Name == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := t.params.UpdateParam(c, id, req.FeatureId, req.KeyId, req.Name, req.Value); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusOK, map[string]string{"status": "ok"})
}

func (t *Controller) deleteParam(c context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := t.params.DeleteParam(c, id); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}

// Simple admin UI
func (t *Controller) indexPage(_ context.Context, ctx httpSrv.ICtx) {
	respondHTML(ctx, http.StatusOK, tplIndexHTML)
}

func (t *Controller) indexJS(_ context.Context, ctx httpSrv.ICtx) {
	respondJS(ctx, http.StatusOK, tplIndexJS)
}
