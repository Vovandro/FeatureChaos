package AdminHTTP

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	FeatureKeyRepository "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureKeyRepository"
	FeatureParamRepository "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	FeatureRepository "gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureRepository"
	ServiceAccessRepository "gitlab.com/devpro_studio/FeatureChaos/src/repository/ServiceAccessRepository"
	AdminService "gitlab.com/devpro_studio/FeatureChaos/src/service/AdminService"
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
	adminSvc AdminService.Interface
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
	t.adminSvc = app.GetModule(interfaces.ModuleService, "admin").(AdminService.Interface)
	t.features = app.GetModule(interfaces.ModuleRepository, "feature").(FeatureRepository.Interface)
	t.keys = app.GetModule(interfaces.ModuleRepository, "feature_key").(FeatureKeyRepository.Interface)
	t.params = app.GetModule(interfaces.ModuleRepository, "feature_param").(FeatureParamRepository.Interface)
	t.stats = app.GetModule(interfaces.ModuleService, "stats").(StatsService.Interface)
	t.access = app.GetModule(interfaces.ModuleRepository, "service_access").(ServiceAccessRepository.Interface)
	http := app.GetPkg(interfaces.PkgServer, "http").(httpSrv.IHttp)

	// static
	http.PushRoute("GET", "/", t.indexPage, nil)
	http.PushRoute("GET", "/index.js", t.indexJS, nil)
	// features
	http.PushRoute("GET", "/api/features", t.listFeatures, nil)
	http.PushRoute("GET", "/api/features/{id}/keys", t.listKeys, nil)
	http.PushRoute("GET", "/api/features/{id}/services", t.listFeatureServices, nil)
	http.PushRoute("POST", "/api/features/{id}/services/{sid}", t.addFeatureService, nil)
	http.PushRoute("DELETE", "/api/features/{id}/services/{sid}", t.removeFeatureService, nil)
	// services
	http.PushRoute("GET", "/api/services", t.listServices, nil)
	http.PushRoute("POST", "/api/services", t.createService, nil)
	http.PushRoute("DELETE", "/api/services/{id}", t.deleteService, nil)
	http.PushRoute("POST", "/api/features", t.createFeature, nil)
	http.PushRoute("PUT", "/api/features/{id}", t.updateFeature, nil)
	http.PushRoute("DELETE", "/api/features/{id}", t.deleteFeature, nil)
	http.PushRoute("POST", "/api/features/{id}/value", t.setFeatureValue, nil)

	// keys
	http.PushRoute("POST", "/api/features/{id}/keys", t.createKey, nil)
	http.PushRoute("PUT", "/api/keys/{id}", t.updateKey, nil)
	http.PushRoute("DELETE", "/api/keys/{id}", t.deleteKey, nil)
	http.PushRoute("POST", "/api/keys/{id}/value", t.setKeyValue, nil)

	// params
	http.PushRoute("POST", "/api/keys/{id}/params", t.createParam, nil)
	http.PushRoute("PUT", "/api/params/{id}", t.updateParam, nil)
	http.PushRoute("DELETE", "/api/params/{id}", t.deleteParam, nil)
	http.PushRoute("POST", "/api/params/{id}/value", t.setParamValue, nil)
	http.PushRoute("GET", "/api/keys/{id}/params", t.listParams, nil)

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

type featureCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (t *Controller) createFeature(_ context.Context, ctx httpSrv.ICtx) {
	var req featureCreateReq
	if err := parseJSON(ctx, &req); err != nil || req.Name == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id, err := t.adminSvc.CreateFeature(context.Background(), req.Name, req.Description)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"id": id.String()})
}

type featureUpdateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (t *Controller) updateFeature(_ context.Context, ctx httpSrv.ICtx) {
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
	if err := t.adminSvc.UpdateFeature(context.Background(), id, req.Name, req.Description); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusOK, map[string]string{"status": "ok"})
}

func (t *Controller) deleteFeature(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	// prevent deletion when feature is active by statistics
	// check activity by feature name
	if t.stats != nil {
		// need feature name to check usage; fetch list and find match
		feats := t.features.GetFeaturesList(context.Background(), []uuid.UUID{id})
		if len(feats) == 1 {
			if t.stats.IsUsed(context.Background(), feats[0].Name) {
				respondJSON(ctx, http.StatusConflict, map[string]string{"error": "feature is active"})
				return
			}
		}
	}
	if err := t.adminSvc.DeleteFeature(context.Background(), id); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}

type setValueReq struct {
	Value int `json:"value"`
}

func (t *Controller) setFeatureValue(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req setValueReq
	if err := parseJSON(ctx, &req); err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	v, err := t.adminSvc.SetFeatureValue(context.Background(), id, req.Value)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusOK, map[string]any{"version": v})
}

func (t *Controller) listFeatures(_ context.Context, ctx httpSrv.ICtx) {
	items := t.features.ListFeatures(context.Background())
	// Batch fetch services for all features to avoid per-item queries (nil-safe)
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.Id)
	}
	var svcMap map[uuid.UUID][]ServiceAccessRepository.Service
	if t.access != nil && len(ids) > 0 {
		svcMap = t.access.GetServicesByFeatureList(context.Background(), ids)
	} else {
		svcMap = make(map[uuid.UUID][]ServiceAccessRepository.Service)
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
	}
	out := make([]resp, 0, len(items))
	for _, it := range items {
		used := false
		if t.stats != nil {
			used = t.stats.IsUsed(context.Background(), it.Name)
		}
		svcs := svcMap[it.Id]
		svcResp := make([]struct {
			ID   string "json:\"id\""
			Name string "json:\"name\""
		}, len(svcs))
		for i, s := range svcs {
			svcResp[i] = struct {
				ID   string "json:\"id\""
				Name string "json:\"name\""
			}{ID: s.Id.String(), Name: s.Name}
		}
		out = append(out, resp{ID: it.Id.String(), Name: it.Name, Description: it.Description, Value: it.Value, Version: it.Version, Used: used, Services: svcResp})
	}
	respondJSON(ctx, http.StatusOK, out)
}

// Services CRUD endpoints
func (t *Controller) listServices(_ context.Context, ctx httpSrv.ICtx) {
	svcs := t.access.ListServices(context.Background())
	type resp struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
	}
	out := make([]resp, len(svcs))
	for i, s := range svcs {
		active := false
		if t.stats != nil {
			active = t.stats.IsServiceUsed(context.Background(), s.Name)
		}
		out[i] = resp{ID: s.Id.String(), Name: s.Name, Active: active}
	}
	respondJSON(ctx, http.StatusOK, out)
}

func (t *Controller) createService(_ context.Context, ctx httpSrv.ICtx) {
	var body struct {
		Name string `json:"name"`
	}
	if err := parseJSON(ctx, &body); err != nil || body.Name == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id, err := t.access.CreateService(context.Background(), body.Name)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"id": id.String()})
}

func (t *Controller) deleteService(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	// prevent deletion when service is active by statistics
	if t.stats != nil {
		if s, ok := t.access.GetServiceById(context.Background(), id); ok {
			if t.stats.IsServiceUsed(context.Background(), s.Name) {
				respondJSON(ctx, http.StatusConflict, map[string]string{"error": "service is active"})
				return
			}
		}
	}
	if err := t.access.DeleteService(context.Background(), id); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}

// Feature-service binding endpoints
func (t *Controller) listFeatureServices(_ context.Context, ctx httpSrv.ICtx) {
	fidStr := ctx.GetRouterValue("id")
	fid, err := uuid.Parse(fidStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid feature id"})
		return
	}
	svcs := t.access.GetServicesByFeature(context.Background(), fid)
	type resp struct{ ID, Name string }
	out := make([]resp, len(svcs))
	for i, s := range svcs {
		out[i] = resp{ID: s.Id.String(), Name: s.Name}
	}
	respondJSON(ctx, http.StatusOK, out)
}

func (t *Controller) addFeatureService(_ context.Context, ctx httpSrv.ICtx) {
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
	if err := t.access.AddAccess(context.Background(), fid, sid); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"status": "ok"})
}

func (t *Controller) removeFeatureService(_ context.Context, ctx httpSrv.ICtx) {
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
	if err := t.access.RemoveAccess(context.Background(), fid, sid); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}

type keyCreateReq struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

func (t *Controller) createKey(_ context.Context, ctx httpSrv.ICtx) {
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
	id, err := t.adminSvc.CreateKey(context.Background(), featureId, req.Key, req.Description)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"id": id.String()})
}

type keyUpdateReq struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

func (t *Controller) updateKey(_ context.Context, ctx httpSrv.ICtx) {
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
	if err := t.adminSvc.UpdateKey(context.Background(), id, req.Key, req.Description); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusOK, map[string]string{"status": "ok"})
}

func (t *Controller) deleteKey(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := t.adminSvc.DeleteKey(context.Background(), id); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}

func (t *Controller) setKeyValue(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req setValueReq
	if err := parseJSON(ctx, &req); err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	v, err := t.adminSvc.SetKeyValue(context.Background(), id, req.Value)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusOK, map[string]any{"version": v})
}

type paramCreateReq struct {
	KeyId uuid.UUID `json:"key_id"`
	Name  string    `json:"name"`
}

func (t *Controller) createParam(_ context.Context, ctx httpSrv.ICtx) {
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
	id, err := t.adminSvc.CreateParam(context.Background(), keyId, body.Name)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusCreated, map[string]string{"id": id.String()})
}

func (t *Controller) updateParam(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := parseJSON(ctx, &req); err != nil || req.Name == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := t.adminSvc.UpdateParam(context.Background(), id, req.Name); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusOK, map[string]string{"status": "ok"})
}

func (t *Controller) deleteParam(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := t.adminSvc.DeleteParam(context.Background(), id); err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusNoContent, nil)
}

func (t *Controller) setParamValue(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req setValueReq
	if err := parseJSON(ctx, &req); err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	v, err := t.adminSvc.SetParamValue(context.Background(), id, req.Value)
	if err != nil {
		respondJSON(ctx, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(ctx, http.StatusOK, map[string]any{"version": v})
}

// Simple admin UI
func (t *Controller) indexPage(_ context.Context, ctx httpSrv.ICtx) {
	respondHTML(ctx, http.StatusOK, tplIndexHTML)
}

func (t *Controller) indexJS(_ context.Context, ctx httpSrv.ICtx) {
	respondJS(ctx, http.StatusOK, tplIndexJS)
}

func (t *Controller) listKeys(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	featureId, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid feature id"})
		return
	}
	items := t.keys.GetFeatureKeys(context.Background(), []uuid.UUID{featureId})
	type resp struct {
		ID      string `json:"id"`
		Key     string `json:"key"`
		Value   int    `json:"value"`
		Version int64  `json:"version"`
	}
	out := make([]resp, 0, len(items))
	for _, it := range items {
		out = append(out, resp{ID: it.Id.String(), Key: it.Key, Value: it.Value, Version: it.Version})
	}
	respondJSON(ctx, http.StatusOK, out)
}

func (t *Controller) listParams(_ context.Context, ctx httpSrv.ICtx) {
	idStr := ctx.GetRouterValue("id")
	keyId, err := uuid.Parse(idStr)
	if err != nil {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid key id"})
		return
	}
	items := t.params.GetParamsByKeyId(context.Background(), keyId)
	type resp struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Value   int    `json:"value"`
		Version int64  `json:"version"`
	}
	out := make([]resp, 0, len(items))
	for _, it := range items {
		out = append(out, resp{ID: it.Id.String(), Name: it.Key, Value: it.Value, Version: it.Version})
	}
	respondJSON(ctx, http.StatusOK, out)
}
