package PublicHTTP

import (
	"context"
	"encoding/json"
	"net/http"

	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/service/FeatureService"
	"gitlab.com/devpro_studio/FeatureChaos/src/service/StatsService"
	"gitlab.com/devpro_studio/Paranoia/paranoia/controller"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

type Controller struct {
	controller.Mock
	featureService FeatureService.Interface
	statsService   StatsService.Interface
}

func New(name string) *Controller {
	return &Controller{Mock: controller.Mock{NamePkg: name}}
}

func (t *Controller) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	// resolve dependencies
	t.featureService = app.GetModule(interfaces.ModuleService, names.FeatureService).(FeatureService.Interface)
	t.statsService = app.GetModule(interfaces.ModuleService, names.StatsService).(StatsService.Interface)

	// mount routes on public HTTP server
	http := app.GetPkg(interfaces.PkgServer, names.HttpPublicServer).(httpSrv.IHttp)
	http.PushRoute("POST", "/api/updates", t.getUpdates, nil)
	http.PushRoute("POST", "/api/stats", t.postStats, nil)
	return nil
}

// helpers
func respondJSON(ctx httpSrv.ICtx, status int, v any) {
	b, _ := json.Marshal(v)
	ctx.GetResponse().Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.GetResponse().SetStatus(status)
	ctx.GetResponse().SetBody(b)
}

func parseJSON[T any](ctx httpSrv.ICtx, out *T) error {
	defer ctx.GetRequest().GetBody().Close()
	dec := json.NewDecoder(ctx.GetRequest().GetBody())
	return dec.Decode(out)
}

// Request/Response shapes mirror the gRPC proto messages but in JSON (single-shot polling)
type updatesRequest struct {
	ServiceName string `json:"service_name"`
	LastVersion int64  `json:"last_version"`
}

type propsItem struct {
	All  int32            `json:"all"`
	Name string           `json:"name"`
	Item map[string]int32 `json:"item"`
}

type featureItem struct {
	All   int32       `json:"all"`
	Name  string      `json:"name"`
	Props []propsItem `json:"props"`
}

// Deleted item kinds: 0=FEATURE, 1=KEY, 2=PARAM (matches proto enum order)
type deletedItem struct {
	Kind        int    `json:"kind"`
	FeatureName string `json:"feature_name"`
	KeyName     string `json:"key_name,omitempty"`
	ParamName   string `json:"param_name,omitempty"`
}

type updatesResponse struct {
	Version  int64         `json:"version"`
	Features []featureItem `json:"features"`
	Deleted  []deletedItem `json:"deleted"`
}

func (t *Controller) getUpdates(c context.Context, ctx httpSrv.ICtx) {
	var req updatesRequest
	if err := parseJSON(ctx, &req); err != nil || req.ServiceName == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	version, features := t.featureService.GetNewFeature(c, req.ServiceName, req.LastVersion)
	resp := updatesResponse{Version: version, Features: make([]featureItem, 0, len(features)), Deleted: make([]deletedItem, 0)}

	for _, feature := range features {
		if feature.IsDeleted {
			resp.Deleted = append(resp.Deleted, deletedItem{Kind: 0, FeatureName: feature.Name})
			continue
		}

		props := make([]propsItem, 0, len(feature.Keys))
		for _, key := range feature.Keys {
			if key.IsDeleted {
				resp.Deleted = append(resp.Deleted, deletedItem{Kind: 1, FeatureName: feature.Name, KeyName: key.Key})
				continue
			}
			items := make(map[string]int32, len(key.Params))
			for _, param := range key.Params {
				if param.IsDeleted {
					resp.Deleted = append(resp.Deleted, deletedItem{Kind: 2, FeatureName: feature.Name, KeyName: key.Key, ParamName: param.Name})
					continue
				}
				items[param.Name] = int32(param.Value)
			}
			props = append(props, propsItem{All: int32(key.Value), Name: key.Key, Item: items})
		}

		resp.Features = append(resp.Features, featureItem{All: int32(feature.Value), Name: feature.Name, Props: props})
	}

	respondJSON(ctx, http.StatusOK, resp)
}

type statsRequest struct {
	ServiceName string   `json:"service_name"`
	Features    []string `json:"features"`
	FeatureName string   `json:"feature_name"`
}

func (t *Controller) postStats(c context.Context, ctx httpSrv.ICtx) {
	var req statsRequest
	if err := parseJSON(ctx, &req); err != nil || req.ServiceName == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if len(req.Features) == 0 && req.FeatureName == "" {
		respondJSON(ctx, http.StatusBadRequest, map[string]string{"error": "features or feature_name required"})
		return
	}

	if len(req.Features) > 0 {
		for _, feat := range req.Features {
			if feat == "" {
				continue
			}
			t.statsService.SetStat(c, req.ServiceName, feat)
		}
	} else if req.FeatureName != "" {
		t.statsService.SetStat(c, req.ServiceName, req.FeatureName)
	}

	respondJSON(ctx, http.StatusOK, map[string]string{"status": "ok"})
}
