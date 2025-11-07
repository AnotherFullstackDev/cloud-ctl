package render

type Config struct {
	ServiceID     string `mapstructure:"service_id"`
	DeploymentKey string `mapstructure:"key"`
}
