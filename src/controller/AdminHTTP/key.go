package AdminHTTP

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

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
