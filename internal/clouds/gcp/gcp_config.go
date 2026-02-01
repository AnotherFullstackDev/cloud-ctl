package gcp

type CloudRunConfig struct {
	ServiceName string `mapstructure:"service_name"`
	ProjectID   string `mapstructure:"project_id"`
	Region      string `mapstructure:"region"`
}
