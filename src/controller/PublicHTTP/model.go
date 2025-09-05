package PublicHTTP

// Request/Response shapes mirror the gRPC proto messages but in JSON (single-shot polling)
type updatesRequest struct {
	ServiceName string `json:"service_name"`
	LastVersion int64  `json:"last_version"`
}

type statsRequest struct {
	ServiceName string   `json:"service_name"`
	Features    []string `json:"features"`
	FeatureName string   `json:"feature_name"`
}

type propsItem struct {
	All  int32            `json:"all"`
	Name string           `json:"name"`
	Item map[string]int32 `json:"item"`
}

type featureItem struct {
	All   int32       `json:"all"`
	Name  string      `json:"name"`
	Props []propsItem `json:"props"`
}

// Deleted item kinds: 0=FEATURE, 1=KEY, 2=PARAM (matches proto enum order)
type deletedItem struct {
	Kind        int    `json:"kind"`
	FeatureName string `json:"feature_name"`
	KeyName     string `json:"key_name,omitempty"`
	ParamName   string `json:"param_name,omitempty"`
}

type updatesResponse struct {
	Version  int64         `json:"version"`
	Features []featureItem `json:"features"`
	Deleted  []deletedItem `json:"deleted"`
}
