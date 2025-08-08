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
	AdminService "gitlab.com/devpro_studio/FeatureChaos/src/service/AdminService"
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
}

func New(name string) *Controller {
	return &Controller{Mock: controller.Mock{NamePkg: name}}
}

func (t *Controller) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	t.adminSvc = app.GetModule(interfaces.ModuleService, "admin").(AdminService.Interface)
	t.features = app.GetModule(interfaces.ModuleRepository, "feature").(FeatureRepository.Interface)
	t.keys = app.GetModule(interfaces.ModuleRepository, "feature_key").(FeatureKeyRepository.Interface)
	t.params = app.GetModule(interfaces.ModuleRepository, "feature_param").(FeatureParamRepository.Interface)
	http := app.GetPkg(interfaces.PkgServer, "http").(httpSrv.IHttp)

	// static
	http.PushRoute("GET", "/", t.indexPage, nil)
	http.PushRoute("GET", "/index.js", t.indexJS, nil)
	// features
	http.PushRoute("GET", "/api/features", t.listFeatures, nil)
	http.PushRoute("GET", "/api/features/{id}/keys", t.listKeys, nil)
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
	type resp struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Value       int    `json:"value"`
		Version     int64  `json:"version"`
	}
	out := make([]resp, 0, len(items))
	for _, it := range items {
		out = append(out, resp{ID: it.Id.String(), Name: it.Name, Description: it.Description, Value: it.Value, Version: it.Version})
	}
	respondJSON(ctx, http.StatusOK, out)
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
