package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/devpro_studio/FeatureChaos/names"
	"gitlab.com/devpro_studio/FeatureChaos/src/controller/AdminHTTP"
	"gitlab.com/devpro_studio/FeatureChaos/src/controller/FeatureChaos"
	"gitlab.com/devpro_studio/FeatureChaos/src/controller/PublicHTTP"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ActivationValuesRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureKeyRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureParamRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/FeatureRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/ServiceAccessRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/repository/StatsRepository"
	"gitlab.com/devpro_studio/FeatureChaos/src/service/FeatureService"
	"gitlab.com/devpro_studio/FeatureChaos/src/service/StatsService"
	"gitlab.com/devpro_studio/Paranoia/paranoia"
	"gitlab.com/devpro_studio/Paranoia/paranoia/interfaces"
	"gitlab.com/devpro_studio/Paranoia/pkg/cache/memory"
	"gitlab.com/devpro_studio/Paranoia/pkg/cache/redis"
	"gitlab.com/devpro_studio/Paranoia/pkg/database/postgres"
	"gitlab.com/devpro_studio/Paranoia/pkg/logger/sentry_log"
	"gitlab.com/devpro_studio/Paranoia/pkg/logger/std_log"
	"gitlab.com/devpro_studio/Paranoia/pkg/server/grpc"
	httpSrv "gitlab.com/devpro_studio/Paranoia/pkg/server/http"
)

func main() {
	s := paranoia.New("feature chaos", "cfg.yaml")

	cfg := s.GetConfig()

	if len(cfg.GetConfigItem(interfaces.PkgLogger, "sentry")) > 0 {
		s.PushPkg(sentry_log.New("sentry"))
	}

	if len(cfg.GetConfigItem(interfaces.PkgLogger, "std")) > 0 {
		s.PushPkg(std_log.New("std"))
	}

	s.PushPkg(memory.New(names.CacheMemory)).
		PushPkg(redis.New(names.CacheRedis)).
		PushPkg(postgres.New(names.DatabasePrimary)).
		PushModule(FeatureRepository.New(names.FeatureRepository)).
		PushModule(FeatureParamRepository.New(names.FeatureParamRepository)).
		PushModule(FeatureKeyRepository.New(names.FeatureKeyRepository)).
		PushModule(ActivationValuesRepository.New(names.ActivationValuesRepository)).
		PushModule(ServiceAccessRepository.New(names.ServiceAccessRepository)).
		PushModule(StatsRepository.New(names.StatsRepository)).
		PushModule(FeatureService.New(names.FeatureService)).
		PushModule(StatsService.New(names.StatsService))

	if len(cfg.GetConfigItem(interfaces.PkgServer, names.HttpPublicServer)) > 0 {
		s.PushPkg(httpSrv.New(names.HttpPublicServer)).
			PushModule(PublicHTTP.New(names.PublicHTTP))
	}

	if len(cfg.GetConfigItem(interfaces.PkgServer, names.GrpcServer)) > 0 {
		s.PushPkg(grpc.New(names.GrpcServer)).
			PushModule(FeatureChaos.NewController(names.FeatureChaosController))
	}

	if len(cfg.GetConfigItem(interfaces.PkgServer, names.HttpServer)) > 0 {
		s.PushPkg(httpSrv.New(names.HttpServer)).
			PushModule(AdminHTTP.New(names.AdminHTTP))
	}

	err := s.Init()
	if err != nil {
		panic(err)
	}
	defer s.Stop()

	s.GetLogger().Info(context.Background(), "start feature chaos service")

	// Wait for syscall stop
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
}
