package AdminHTTP

import (
	"strconv"
	"time"

	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

type GetFeaturesRequest struct {
	ServiceId    string
	Find         string
	IsDeprecated bool
	Page         int
}

type GetFeaturesResponse struct {
	Page       int       `json:"page"`
	TotalPages int       `json:"total_pages"`
	Features   []Feature `json:"features"`
}

type Feature struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Value        int       `json:"value"`
	Used         bool      `json:"used"`
	IsDeprecated bool      `json:"is_deprecated"`
	Services     []Service `json:"services"`
	Keys         []Key     `json:"keys"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Service struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type Key struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Value  int     `json:"value"`
	Params []Param `json:"params"`
}

type Param struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func (t *GetFeaturesRequest) FromRequest(ctx httpSrv.ICtx) error {
	t.ServiceId = ctx.GetRequest().GetQuery().Get("service_id")
	t.Find = ctx.GetRequest().GetQuery().Get("find")
	t.IsDeprecated = ctx.GetRequest().GetQuery().Get("is_deprecated") == "true"

	t.Page = 1
	page := ctx.GetRequest().GetQuery().Get("page")
	if page != "" {
		pageInt, err := strconv.Atoi(page)
		if err == nil && pageInt > 0 {
			t.Page = pageInt
		}
	}

	return nil
}
