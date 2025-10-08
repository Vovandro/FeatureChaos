package AdminHTTP

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"

	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureKeyRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ServiceAccessRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/service/StatsService"
	"gitlab.com/devpro_studio/Paranoia/paranoia/controller"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
	"gitlab.com/devpro_studio/go_utils/decode"
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

	config Config
}

type Config struct {
	AppUrl string `yaml:"app_url"`
}

func New(name string) *Controller {
	return &Controller{Mock: controller.Mock{NamePkg: name}}
}

func (t *Controller) Init(app interfaces.IEngine, cfg map[string]interface{}) error {
	t.features = app.GetModule(interfaces.ModuleRepository, names.FeatureRepository).(FeatureRepository.Interface)
	t.keys = app.GetModule(interfaces.ModuleRepository, names.FeatureKeyRepository).(FeatureKeyRepository.Interface)
	t.params = app.GetModule(interfaces.ModuleRepository, names.FeatureParamRepository).(FeatureParamRepository.Interface)
	t.stats = app.GetModule(interfaces.ModuleService, names.StatsService).(StatsService.Interface)
	t.access = app.GetModule(interfaces.ModuleRepository, names.ServiceAccessRepository).(ServiceAccessRepository.Interface)
	http := app.GetPkg(interfaces.PkgServer, names.HttpServer).(httpSrv.IHttp)

	err := decode.Decode(cfg, &t.config, "yaml", decode.DecoderStrongFoundDst)
	if err != nil {
		return err
	}

	t.config.AppUrl = strings.TrimRight(t.config.AppUrl, "/")

	tplIndexHTML = strings.ReplaceAll(tplIndexHTML, "{{APP_URL}}", t.config.AppUrl)
	tplIndexJS = strings.ReplaceAll(tplIndexJS, "{{APP_URL}}", t.config.AppUrl)

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

// Simple admin UI
func (t *Controller) indexPage(_ context.Context, ctx httpSrv.ICtx) {
	respondHTML(ctx, http.StatusOK, tplIndexHTML)
}

func (t *Controller) indexJS(_ context.Context, ctx httpSrv.ICtx) {
	respondJS(ctx, http.StatusOK, tplIndexJS)
}
