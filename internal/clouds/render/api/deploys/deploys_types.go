package deploys

type ClearCache string

// clear do_not_clear
const (
	ClearCacheClear      ClearCache = "clear"
	ClearCacheDoNotClear ClearCache = "do_not_clear"
)

type TriggerDeployInput struct {
	ClearCache ClearCache `json:"clearCache,omitempty"`
	CommitID   string     `json:"commitId,omitempty"`
	ImageID    string     `json:"imageId,omitempty"`
}
