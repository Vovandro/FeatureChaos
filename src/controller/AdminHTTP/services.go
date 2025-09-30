package AdminHTTP

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

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
