package AdminHTTP

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/model/db"
	AdminSvc "gitlab.com/devpro_studio/FeatureChaos/src/service/AdminService"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

// Minimal fakes for http package interfaces
type fakeRequest struct{ body io.ReadCloser }

func (r *fakeRequest) GetBody() io.ReadCloser { return r.body }
func (r *fakeRequest) GetBodySize() int64 {
	if r.body == nil {
		return 0
	}
	return 0
}
func (r *fakeRequest) GetCookie() httpSrv.ICookie { return &fakeCookie{} }
func (r *fakeRequest) GetHeader() httpSrv.IHeader { return &fakeHeader{h: http.Header{}} }
func (r *fakeRequest) GetMethod() string          { return "" }
func (r *fakeRequest) GetURI() string             { return "" }
func (r *fakeRequest) GetQuery() httpSrv.IQuery   { return &fakeQuery{} }
func (r *fakeRequest) GetRemoteIP() string        { return "" }
func (r *fakeRequest) GetRemoteHost() string      { return "" }
func (r *fakeRequest) GetUserAgent() string       { return "" }

type fakeResponse struct {
	header *fakeHeader
	status int
	body   []byte
}

func (r *fakeResponse) Header() httpSrv.IHeader {
	if r.header == nil {
		r.header = &fakeHeader{h: http.Header{}}
	}
	return r.header
}
func (r *fakeResponse) SetStatus(code int)      { r.status = code }
func (r *fakeResponse) SetBody(b []byte)        { r.body = b }
func (r *fakeResponse) Clear()                  { r.header = &fakeHeader{h: http.Header{}}; r.status = 0; r.body = nil }
func (r *fakeResponse) GetBody() []byte         { return r.body }
func (r *fakeResponse) GetStatus() int          { return r.status }
func (r *fakeResponse) Cookie() httpSrv.ICookie { return &fakeCookie{} }

type fakeCtx struct {
	req    httpSrv.IRequest
	resp   httpSrv.IResponse
	params map[string]string
}

func (c *fakeCtx) GetRequest() httpSrv.IRequest             { return c.req }
func (c *fakeCtx) GetResponse() httpSrv.IResponse           { return c.resp }
func (c *fakeCtx) GetRouterValue(k string) string           { return c.params[k] }
func (c *fakeCtx) GetUserValue(string) (interface{}, error) { return nil, nil }
func (c *fakeCtx) PushUserValue(string, interface{})        {}
func (c *fakeCtx) SetRouteProps(map[string]string)          {}

type fakeCookie struct{}

func (f *fakeCookie) Set(string, string, string, time.Duration) {}
func (f *fakeCookie) Get(string) string                         { return "" }
func (f *fakeCookie) GetAsMap() map[string]string               { return map[string]string{} }

type fakeHeader struct{ h http.Header }

func (f *fakeHeader) Add(k, v string)               { f.h.Add(k, v) }
func (f *fakeHeader) Set(k, v string)               { f.h.Set(k, v) }
func (f *fakeHeader) Get(k string) string           { return f.h.Get(k) }
func (f *fakeHeader) Values(k string) []string      { return f.h.Values(k) }
func (f *fakeHeader) Del(k string)                  { f.h.Del(k) }
func (f *fakeHeader) GetAsMap() map[string][]string { return f.h }

type fakeQuery struct{}

func (f *fakeQuery) Get(string) string { return "" }

// Fakes for Admin service and repositories
type fakeAdmin struct {
	id      uuid.UUID
	version int64
	err     error
}

func (f *fakeAdmin) CreateFeature(context.Context, string, string) (uuid.UUID, error) {
	return f.id, f.err
}
func (f *fakeAdmin) UpdateFeature(context.Context, uuid.UUID, string, string) error { return f.err }
func (f *fakeAdmin) DeleteFeature(context.Context, uuid.UUID) error                 { return f.err }
func (f *fakeAdmin) SetFeatureValue(context.Context, uuid.UUID, int) (int64, error) {
	return f.version, f.err
}
func (f *fakeAdmin) CreateKey(context.Context, uuid.UUID, string, string) (uuid.UUID, error) {
	return uuid.New(), f.err
}
func (f *fakeAdmin) UpdateKey(context.Context, uuid.UUID, string, string) error { return f.err }
func (f *fakeAdmin) DeleteKey(context.Context, uuid.UUID) error                 { return f.err }
func (f *fakeAdmin) SetKeyValue(context.Context, uuid.UUID, int) (int64, error) {
	return f.version, f.err
}
func (f *fakeAdmin) CreateParam(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	return uuid.New(), f.err
}
func (f *fakeAdmin) UpdateParam(context.Context, uuid.UUID, string) error { return f.err }
func (f *fakeAdmin) DeleteParam(context.Context, uuid.UUID) error         { return f.err }
func (f *fakeAdmin) SetParamValue(context.Context, uuid.UUID, int) (int64, error) {
	return f.version, f.err
}

type fakeFeatures struct{ items []*db.Feature }

func (f *fakeFeatures) GetFeaturesList(context.Context, []uuid.UUID) []*db.Feature { return f.items }
func (f *fakeFeatures) ListFeatures(context.Context) []*db.Feature                 { return f.items }
func (f *fakeFeatures) CreateFeature(context.Context, string, string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (f *fakeFeatures) UpdateFeature(context.Context, uuid.UUID, string, string) error { return nil }
func (f *fakeFeatures) DeleteFeature(context.Context, uuid.UUID) error                 { return nil }

type fakeKeys struct{ items []*db.FeatureKey }

func (f *fakeKeys) GetFeatureKeys(context.Context, []uuid.UUID) []*db.FeatureKey { return f.items }
func (f *fakeKeys) CreateKey(context.Context, uuid.UUID, string, string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (f *fakeKeys) UpdateKey(context.Context, uuid.UUID, string, string) error { return nil }
func (f *fakeKeys) DeleteKey(context.Context, uuid.UUID) error                 { return nil }
func (f *fakeKeys) DeleteByFeatureId(context.Context, uuid.UUID) error         { return nil }
func (f *fakeKeys) GetFeatureIdByKeyId(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

type fakeParams struct{ items []*db.FeatureParam }

func (f *fakeParams) GetFeatureParams(context.Context, []uuid.UUID) []*db.FeatureParam {
	return f.items
}
func (f *fakeParams) GetParamsByKeyId(context.Context, uuid.UUID) []*db.FeatureParam { return f.items }
func (f *fakeParams) CreateParam(context.Context, uuid.UUID, uuid.UUID, string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (f *fakeParams) UpdateParam(context.Context, uuid.UUID, string) error { return nil }
func (f *fakeParams) DeleteParam(context.Context, uuid.UUID) error         { return nil }
func (f *fakeParams) GetFeatureIdByParamId(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

var _ AdminSvc.Interface = (*fakeAdmin)(nil)

func TestCreateFeature_OK(t *testing.T) {
	fid := uuid.New()
	ctl := &Controller{adminSvc: &fakeAdmin{id: fid}}
	body, _ := json.Marshal(map[string]any{"name": "feature", "description": "desc"})
	ctx := &fakeCtx{
		req:    &fakeRequest{body: io.NopCloser(bytes.NewBuffer(body))},
		resp:   &fakeResponse{},
		params: map[string]string{},
	}
	ctl.createFeature(context.Background(), ctx)
	resp := ctx.resp.(*fakeResponse)
	if resp.status != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.status)
	}
	var out map[string]string
	if err := json.Unmarshal(resp.body, &out); err != nil || out["id"] == "" {
		t.Fatalf("invalid response body: %s, err=%v", string(resp.body), err)
	}
}

func TestCreateFeature_BadBody(t *testing.T) {
	ctl := &Controller{adminSvc: &fakeAdmin{}}
	ctx := &fakeCtx{req: &fakeRequest{body: io.NopCloser(bytes.NewBufferString("{}"))}, resp: &fakeResponse{}}
	ctl.createFeature(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", ctx.resp.(*fakeResponse).status)
	}
}

func TestListFeatures_OK(t *testing.T) {
	fid := uuid.New()
	ctl := &Controller{features: &fakeFeatures{items: []*db.Feature{{Id: fid, Name: "n", Description: "d", Value: 7, Version: 3}}}}
	ctx := &fakeCtx{req: &fakeRequest{body: io.NopCloser(bytes.NewBuffer(nil))}, resp: &fakeResponse{}}
	ctl.listFeatures(context.Background(), ctx)
	resp := ctx.resp.(*fakeResponse)
	if resp.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.status)
	}
	var out []map[string]any
	if err := json.Unmarshal(resp.body, &out); err != nil || len(out) != 1 {
		t.Fatalf("invalid json: %s err=%v", string(resp.body), err)
	}
}

func newCtx(body []byte, params map[string]string) *fakeCtx {
	return &fakeCtx{
		req:    &fakeRequest{body: io.NopCloser(bytes.NewBuffer(body))},
		resp:   &fakeResponse{},
		params: params,
	}
}

func TestUpdateFeature_OK_And_BadInputs(t *testing.T) {
	fid := uuid.New()
	ctl := &Controller{adminSvc: &fakeAdmin{}}

	// OK
	body, _ := json.Marshal(map[string]string{"name": "n2", "description": "d2"})
	ctx := newCtx(body, map[string]string{"id": fid.String()})
	ctl.updateFeature(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}

	// Bad ID
	ctx = newCtx(body, map[string]string{"id": "bad"})
	ctl.updateFeature(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad id, got %d", ctx.resp.(*fakeResponse).status)
	}

	// Bad body
	ctx = newCtx([]byte("{}"), map[string]string{"id": fid.String()})
	ctl.updateFeature(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad body, got %d", ctx.resp.(*fakeResponse).status)
	}
}

func TestDeleteFeature_OK_And_BadID(t *testing.T) {
	fid := uuid.New()
	ctl := &Controller{adminSvc: &fakeAdmin{}}

	// OK
	ctx := newCtx(nil, map[string]string{"id": fid.String()})
	ctl.deleteFeature(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", ctx.resp.(*fakeResponse).status)
	}

	// Bad ID
	ctx = newCtx(nil, map[string]string{"id": "bad"})
	ctl.deleteFeature(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", ctx.resp.(*fakeResponse).status)
	}
}

func TestSetFeatureValue_OK_And_Errors(t *testing.T) {
	fid := uuid.New()
	ctl := &Controller{adminSvc: &fakeAdmin{version: 42}}

	// OK
	body, _ := json.Marshal(map[string]int{"value": 5})
	ctx := newCtx(body, map[string]string{"id": fid.String()})
	ctl.setFeatureValue(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}

	// Bad ID
	ctx = newCtx(body, map[string]string{"id": "bad"})
	ctl.setFeatureValue(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad id, got %d", ctx.resp.(*fakeResponse).status)
	}

	// Bad body
	ctx = newCtx([]byte("not json"), map[string]string{"id": fid.String()})
	ctl.setFeatureValue(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad body, got %d", ctx.resp.(*fakeResponse).status)
	}
}

func TestKeys_CRUD_And_Value(t *testing.T) {
	fid := uuid.New()
	kid := uuid.New()
	ctl := &Controller{adminSvc: &fakeAdmin{version: 7}}

	// createKey OK
	body, _ := json.Marshal(map[string]string{"key": "k", "description": "d"})
	ctx := newCtx(body, map[string]string{"id": fid.String()})
	ctl.createKey(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusCreated {
		t.Fatalf("expected 201, got %d", ctx.resp.(*fakeResponse).status)
	}

	// createKey bad feature id
	ctx = newCtx(body, map[string]string{"id": "bad"})
	ctl.createKey(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", ctx.resp.(*fakeResponse).status)
	}

	// updateKey OK
	body2, _ := json.Marshal(map[string]string{"key": "k2", "description": "d2"})
	ctx = newCtx(body2, map[string]string{"id": kid.String()})
	ctl.updateKey(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}

	// updateKey bad id
	ctx = newCtx(body2, map[string]string{"id": "bad"})
	ctl.updateKey(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", ctx.resp.(*fakeResponse).status)
	}

	// setKeyValue OK
	body3, _ := json.Marshal(map[string]int{"value": 9})
	ctx = newCtx(body3, map[string]string{"id": kid.String()})
	ctl.setKeyValue(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}

	// deleteKey OK
	ctx = newCtx(nil, map[string]string{"id": kid.String()})
	ctl.deleteKey(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", ctx.resp.(*fakeResponse).status)
	}
}

func TestListKeys_OK_And_BadID(t *testing.T) {
	fid := uuid.New()
	ctl := &Controller{keys: &fakeKeys{items: []*db.FeatureKey{{Id: uuid.New(), FeatureId: fid, Key: "k", Description: "d", Value: 1, Version: 2}}}}

	// OK
	ctx := newCtx(nil, map[string]string{"id": fid.String()})
	ctl.listKeys(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}

	// Bad id
	ctx = newCtx(nil, map[string]string{"id": "bad"})
	ctl.listKeys(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", ctx.resp.(*fakeResponse).status)
	}
}

func TestParams_CRUD_And_Value(t *testing.T) {
	kid := uuid.New()
	pid := uuid.New()
	ctl := &Controller{adminSvc: &fakeAdmin{version: 11}}

	// createParam OK (name from body, key id from path)
	body, _ := json.Marshal(map[string]string{"name": "p"})
	ctx := newCtx(body, map[string]string{"id": kid.String()})
	ctl.createParam(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusCreated {
		t.Fatalf("expected 201, got %d", ctx.resp.(*fakeResponse).status)
	}

	// updateParam OK
	body2, _ := json.Marshal(map[string]string{"name": "p2"})
	ctx = newCtx(body2, map[string]string{"id": pid.String()})
	ctl.updateParam(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}

	// setParamValue OK
	body3, _ := json.Marshal(map[string]int{"value": 3})
	ctx = newCtx(body3, map[string]string{"id": pid.String()})
	ctl.setParamValue(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}

	// deleteParam OK
	ctx = newCtx(nil, map[string]string{"id": pid.String()})
	ctl.deleteParam(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", ctx.resp.(*fakeResponse).status)
	}
}

func TestListParams_OK_And_BadID(t *testing.T) {
	kid := uuid.New()
	ctl := &Controller{params: &fakeParams{items: []*db.FeatureParam{{Id: uuid.New(), FeatureId: uuid.New(), KeyId: kid, Key: "p", Value: 1, Version: 2}}}}

	// OK
	ctx := newCtx(nil, map[string]string{"id": kid.String()})
	ctl.listParams(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}

	// Bad id
	ctx = newCtx(nil, map[string]string{"id": "bad"})
	ctl.listParams(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", ctx.resp.(*fakeResponse).status)
	}
}

func TestIndexPages_OK(t *testing.T) {
	ctl := &Controller{}
	// index html
	ctx := newCtx(nil, nil)
	ctl.indexPage(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}
	// index js
	ctx = newCtx(nil, nil)
	ctl.indexJS(context.Background(), ctx)
	if ctx.resp.(*fakeResponse).status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ctx.resp.(*fakeResponse).status)
	}
}
