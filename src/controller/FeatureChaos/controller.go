package FeatureChaos

import (
	"gitlab.com/devpro_studio/FeatureChaos/src/service/FeatureService"
	"gitlab.com/devpro_studio/Paranoia/paranoia/controller"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/pkg/server/grpc"
	grpc2 "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type Controller struct {
	controller.Mock
	UnimplementedFeatureServiceServer
	featureService FeatureService.Interface
}

func NewController(name string) *Controller {
	return &Controller{
		Mock: controller.Mock{
			NamePkg: name,
		},
	}
}

func (t *Controller) Init(app interfaces.IEngine, _ map[string]interface{}) error {
	app.GetPkg(interfaces.PkgServer, "grpc").(grpc.IGrpc).RegisterService(&FeatureService_ServiceDesc, t)
	t.featureService = app.GetModule(interfaces.ModuleService, "feature").(FeatureService.Interface)

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
			features := t.featureService.GetNewFeature(request.ServiceName, lastVersion)

			if len(features) > 0 {
				resp := &GetFeatureResponse{
					Features: make([]*FeatureItem, 0, len(features)),
				}

				for _, feature := range features {
					if feature.Version > lastVersion {
						lastVersion = feature.Version
					}

					props := make([]*PropsItem, len(feature.Keys))

					for i, key := range feature.Keys {
						items := make(map[string]int32, len(key.Params))

						for _, param := range key.Params {
							items[param.Name] = int32(param.Value)
						}

						props[i] = &PropsItem{
							All:  int32(key.Value),
							Name: key.Key,
							Item: items,
						}
					}

					resp.Features = append(resp.Features, &FeatureItem{
						All:   int32(feature.Value),
						Name:  feature.Name,
						Props: props,
					})
				}

				resp.Version = lastVersion

				err := response.Send(resp)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (t *Controller) Stats(request grpc2.ClientStreamingServer[SendStatsRequest, emptypb.Empty]) error {
	return status.Errorf(codes.Unimplemented, "method Stats not implemented")
}
