package FeatureChaos

import (
	"io"
	"time"

	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/service/FeatureService"
	"gitlab.com/devpro_studio/FeatureChaos/src/service/StatsService"
	"gitlab.com/devpro_studio/Paranoia/paranoia/controller"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/pkg/server/grpc"
	grpc2 "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Controller struct {
	controller.Mock
	UnimplementedFeatureServiceServer
	featureService FeatureService.Interface
	statsService   StatsService.Interface
}

func NewController(name string) *Controller {
	return &Controller{
		Mock: controller.Mock{
			NamePkg: name,
		},
	}
}

func (t *Controller) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	app.GetPkg(interfaces.PkgServer, names.GrpcServer).(grpc.IGrpc).RegisterService(&FeatureService_ServiceDesc, t)
	t.featureService = app.GetModule(interfaces.ModuleService, names.FeatureService).(FeatureService.Interface)
	t.statsService = app.GetModule(interfaces.ModuleService, names.StatsService).(StatsService.Interface)
	return nil
}

func (t *Controller) Subscribe(request *GetAllFeatureRequest, response grpc2.ServerStreamingServer[GetFeatureResponse]) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	lastVersion := request.LastVersion

	for {
		select {
		case <-response.Context().Done():
			return nil

		case <-ticker.C:
			version, features := t.featureService.GetNewFeature(response.Context(), request.ServiceName, lastVersion)

			if len(features) > 0 {
				resp := &GetFeatureResponse{
					Features: make([]*FeatureItem, 0, len(features)),
					Deleted:  make([]*GetFeatureResponse_DeletedItem, 0),
				}

				for _, feature := range features {
					// If feature is deleted, record and skip deeper levels
					if feature.IsDeleted {
						resp.Deleted = append(resp.Deleted, &GetFeatureResponse_DeletedItem{
							Kind:        GetFeatureResponse_DeletedItem_FEATURE,
							FeatureName: feature.Name,
						})
						continue
					}

					props := make([]*PropsItem, 0, len(feature.Keys))

					for _, key := range feature.Keys {
						// If key is deleted, record and skip params
						if key.IsDeleted {
							resp.Deleted = append(resp.Deleted, &GetFeatureResponse_DeletedItem{
								Kind:        GetFeatureResponse_DeletedItem_KEY,
								FeatureName: feature.Name,
								KeyName:     key.Key,
							})
							continue
						}

						items := make(map[string]int32, len(key.Params))

						for _, param := range key.Params {
							// If param is deleted, record and skip adding to map
							if param.IsDeleted {
								resp.Deleted = append(resp.Deleted, &GetFeatureResponse_DeletedItem{
									Kind:        GetFeatureResponse_DeletedItem_PARAM,
									FeatureName: feature.Name,
									KeyName:     key.Key,
									ParamName:   param.Name,
								})
								continue
							}
							items[param.Name] = int32(param.Value)
						}

						props = append(props, &PropsItem{
							All:  int32(key.Value),
							Name: key.Key,
							Item: items,
						})
					}

					resp.Features = append(resp.Features, &FeatureItem{
						All:   int32(feature.Value),
						Name:  feature.Name,
						Props: props,
					})
				}

				resp.Version = version

				err := response.Send(resp)
				if err != nil {
					return err
				}
			}
		}
	}

}

func (t *Controller) Stats(request grpc2.ClientStreamingServer[SendStatsRequest, emptypb.Empty]) error {
	for {
		req, err := request.Recv()
		if err != nil {
			// graceful close on EOF
			if err == io.EOF {
				return request.SendAndClose(&emptypb.Empty{})
			}
			return err
		}

		t.statsService.SetStat(request.Context(), req.ServiceName, req.FeatureName)
	}
}
