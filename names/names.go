package names

const (
	CacheMemory                = "secondary"
	CacheRedis                 = "primary"
	DatabasePrimary            = "primary"
	HttpServer                 = "http"
	HttpPublicServer           = "http_public"
	GrpcServer                 = "grpc"
	FeatureKeyRepository       = "feature_key"
	FeatureParamRepository     = "feature_param"
	FeatureRepository          = "feature"
	ActivationValuesRepository = "activation_values"
	ServiceAccessRepository    = "service_access"
	StatsRepository            = "stats"
	FeatureService             = "feature"
	StatsService               = "stats"
	FeatureChaosController     = "grpc_controller"
	AdminHTTP                  = "http_admin"
	PublicHTTP                 = "http_public"
)
