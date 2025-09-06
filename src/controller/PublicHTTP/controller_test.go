package PublicHTTP

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ActivationValuesRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/service/FeatureService"
	"gitlab.com/devpro_studio/Paranoia/pkg/cache/redis"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
	"gitlab.com/devpro_studio/Paranoia/pkg/logger/mock_log"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

func TestController_getUpdates(t *testing.T) {
	type test struct {
		name      string
		reqBody   string
		resCode   int
		resData   *updatesResponse
		mockPg    *postgres.Mock
		mockRedis *redis.Mock
	}

	tests := []test{
		{
			name:    "empty data",
			reqBody: `{"service_name": "test", "last_version": 0}`,
			resCode: http.StatusOK,
			resData: &updatesResponse{
				Version:  -1,
				Features: []featureItem{},
				Deleted:  []deletedItem{},
			},
			mockPg:    &postgres.Mock{},
			mockRedis: &redis.Mock{},
		},
		{
			name:    "one feature data",
			reqBody: `{"service_name": "test", "last_version": 0}`,
			resCode: http.StatusOK,
			resData: &updatesResponse{
				Version: 1,
				Features: []featureItem{
					{
						All:   100,
						Name:  "test_feature",
						Props: []propsItem{},
					},
				},
				Deleted: []deletedItem{},
			},
			mockPg: &postgres.Mock{
				QueryFunc: func(c context.Context, query string, args ...any) (postgres.SQLRows, error) {
					return &postgres.MockRows{
						Values: [][]any{
							{
								uuid.New().String(), // feature_id
								"test_feature",      // feature_name
								nil,                 // key_id
								nil,                 // key_name
								nil,                 // param_id
								nil,                 // param_name
								100,                 // value
								int64(1),            // v
								nil,                 // deleted_at
							},
						},
					}, nil
				},
			},
			mockRedis: &redis.Mock{
				Data: map[string]string{
					"feature_version": "1",
				},
			},
		},
		{
			name:    "features data",
			reqBody: `{"service_name": "test", "last_version": 0}`,
			resCode: http.StatusOK,
			resData: &updatesResponse{
				Version: 1,
				Features: []featureItem{
					{
						All:  100,
						Name: "test_feature",
						Props: []propsItem{
							{
								All:  99,
								Name: "test_key",
								Item: map[string]int32{},
							},
							{
								All:  98,
								Name: "test_key2",
								Item: map[string]int32{
									"test_param_1": 1,
									"test_param_2": 2,
								},
							},
						},
					},
				},
				Deleted: []deletedItem{},
			},
			mockPg: &postgres.Mock{
				QueryFunc: func(c context.Context, query string, args ...any) (postgres.SQLRows, error) {
					featureId := uuid.New()
					keyId := uuid.New()
					return &postgres.MockRows{
						Values: [][]any{
							{
								featureId.String(), // feature_id
								"test_feature",     // feature_name
								nil,                // key_id
								nil,                // key_name
								nil,                // param_id
								nil,                // param_name
								100,                // value
								int64(1),           // v
								nil,                // deleted_at
							},
							{
								featureId.String(),  // feature_id
								"test_feature",      // feature_name
								uuid.New().String(), // key_id
								"test_key",          // key_name
								nil,                 // param_id
								nil,                 // param_name
								99,                  // value
								int64(2),            // v
								nil,                 // deleted_at
							},
							{
								featureId.String(), // feature_id
								"test_feature",     // feature_name
								keyId.String(),     // key_id
								"test_key2",        // key_name
								nil,                // param_id
								nil,                // param_name
								98,                 // value
								int64(2),           // v
								nil,                // deleted_at
							},
							{
								featureId.String(),  // feature_id
								"test_feature",      // feature_name
								keyId.String(),      // key_id
								"test_key2",         // key_name
								uuid.New().String(), // param_id
								"test_param_1",      // param_name
								1,                   // value
								int64(2),            // v
								nil,                 // deleted_at
							},
							{
								featureId.String(),  // feature_id
								"test_feature",      // feature_name
								keyId.String(),      // key_id
								"test_key2",         // key_name
								uuid.New().String(), // param_id
								"test_param_2",      // param_name
								2,                   // value
								int64(2),            // v
								nil,                 // deleted_at
							},
						},
					}, nil
				},
			},
			mockRedis: &redis.Mock{
				Data: map[string]string{
					"feature_version": "1",
				},
			},
		},
		{
			name:    "without old data",
			reqBody: `{"service_name": "test", "last_version": 1}`,
			resCode: http.StatusOK,
			resData: &updatesResponse{
				Version: 2,
				Features: []featureItem{
					{
						All:  -1,
						Name: "test_feature",
						Props: []propsItem{
							{
								All:  99,
								Name: "test_key",
								Item: map[string]int32{},
							},
							{
								All:  -1,
								Name: "test_key2",
								Item: map[string]int32{
									"test_param_1": 1,
									"test_param_2": 2,
								},
							},
						},
					},
				},
				Deleted: []deletedItem{},
			},
			mockPg: &postgres.Mock{
				QueryFunc: func(c context.Context, query string, args ...any) (postgres.SQLRows, error) {
					featureId := uuid.New()
					keyId := uuid.New()
					return &postgres.MockRows{
						Values: [][]any{
							{
								featureId.String(),  // feature_id
								"test_feature",      // feature_name
								uuid.New().String(), // key_id
								"test_key",          // key_name
								nil,                 // param_id
								nil,                 // param_name
								99,                  // value
								int64(2),            // v
								nil,                 // deleted_at
							},
							{
								featureId.String(),  // feature_id
								"test_feature",      // feature_name
								keyId.String(),      // key_id
								"test_key2",         // key_name
								uuid.New().String(), // param_id
								"test_param_1",      // param_name
								1,                   // value
								int64(2),            // v
								nil,                 // deleted_at
							},
							{
								featureId.String(),  // feature_id
								"test_feature",      // feature_name
								keyId.String(),      // key_id
								"test_key2",         // key_name
								uuid.New().String(), // param_id
								"test_param_2",      // param_name
								2,                   // value
								int64(2),            // v
								nil,                 // deleted_at
							},
						},
					}, nil
				},
			},
			mockRedis: &redis.Mock{
				Data: map[string]string{
					"feature_version": "2",
				},
			},
		},
		{
			name:    "deleted features data",
			reqBody: `{"service_name": "test", "last_version": 1}`,
			resCode: http.StatusOK,
			resData: &updatesResponse{
				Version:  2,
				Features: []featureItem{},
				Deleted: []deletedItem{
					{
						Kind:        0,
						FeatureName: "test_feature",
					},
					{
						Kind:        1,
						FeatureName: "test_feature_2",
						KeyName:     "test_key",
					},
					{
						Kind:        2,
						FeatureName: "test_feature_3",
						KeyName:     "test_key2",
						ParamName:   "test_param_1",
					},
					{
						Kind:        2,
						FeatureName: "test_feature_3",
						KeyName:     "test_key2",
						ParamName:   "test_param_2",
					},
				},
			},
			mockPg: &postgres.Mock{
				QueryFunc: func(c context.Context, query string, args ...any) (postgres.SQLRows, error) {
					featureId := uuid.New()
					keyId := uuid.New()
					return &postgres.MockRows{
						Values: [][]any{
							{
								uuid.New().String(), // feature_id
								"test_feature",      // feature_name
								nil,                 // key_id
								nil,                 // key_name
								nil,                 // param_id
								nil,                 // param_name
								100,                 // value
								int64(1),            // v
								time.Now(),          // deleted_at
							},
							{
								uuid.New().String(), // feature_id
								"test_feature_2",    // feature_name
								uuid.New().String(), // key_id
								"test_key",          // key_name
								nil,                 // param_id
								nil,                 // param_name
								99,                  // value
								int64(2),            // v
								time.Now(),          // deleted_at
							},
							{
								featureId.String(),  // feature_id
								"test_feature_3",    // feature_name
								keyId.String(),      // key_id
								"test_key2",         // key_name
								uuid.New().String(), // param_id
								"test_param_1",      // param_name
								1,                   // value
								int64(2),            // v
								time.Now(),          // deleted_at
							},
							{
								featureId.String(),  // feature_id
								"test_feature_3",    // feature_name
								keyId.String(),      // key_id
								"test_key2",         // key_name
								uuid.New().String(), // param_id
								"test_param_2",      // param_name
								2,                   // value
								int64(2),            // v
								time.Now(),          // deleted_at
							},
						},
					}, nil
				},
			},
			mockRedis: &redis.Mock{
				Data: map[string]string{
					"feature_version": "2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Controller{
				featureService: FeatureService.NewForTest(
					ActivationValuesRepository.NewForTest(
						tt.mockPg,
						tt.mockRedis,
						mock_log.New(true),
					),
				),
			}

			ctx := httpSrv.HttpCtxPool.Get().(*httpSrv.HttpCtx)
			ctx.Fill(httptest.NewRequest("POST", "/api/updates", bytes.NewBufferString(tt.reqBody)))
			c.getUpdates(context.Background(), ctx)

			if tt.resCode != ctx.GetResponse().GetStatus() {
				t.Errorf("expected code %d, got %d", tt.resCode, ctx.GetResponse().GetStatus())
			}

			var body updatesResponse
			json.Unmarshal(ctx.GetResponse().GetBody(), &body)

			if !reflect.DeepEqual(*tt.resData, body) {
				t.Errorf("expected data %v, got %v", tt.resData, body)
			}
		})
	}
}
