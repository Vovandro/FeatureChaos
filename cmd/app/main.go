package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	AdminHTTP "gitlab.com/devpro_studio/FeatureChaos/src/controller/AdminHTTP"
	"gitlab.com/devpro_studio/FeatureChaos/src/controller/FeatureChaos"
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
	sentry_log "gitlab.com/devpro_studio/Paranoia/pkg/logger/sentry-log"
	std_log "gitlab.com/devpro_studio/Paranoia/pkg/logger/std-log"
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

	s.PushPkg(memory.New("secondary")).
		PushPkg(redis.New("primary")).
		PushPkg(postgres.New("primary")).
		PushPkg(grpc.New("grpc")).
		PushPkg(httpSrv.New("http")).
		PushModule(FeatureRepository.New("feature")).
		PushModule(FeatureParamRepository.New("feature_param")).
		PushModule(FeatureKeyRepository.New("feature_key")).
		PushModule(ActivationValuesRepository.New("activation_values")).
		PushModule(ServiceAccessRepository.New("service_access")).
		PushModule(StatsRepository.New("stats")).
		PushModule(FeatureService.New("feature")).
		PushModule(StatsService.New("stats")).
		PushModule(FeatureChaos.NewController("grpc_controller")). // inner space
		PushModule(AdminHTTP.New("http_admin"))

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
