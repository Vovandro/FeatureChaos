package AdminHTTP

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

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
