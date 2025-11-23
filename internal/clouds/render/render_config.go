package render

type Config struct {
	ProjectID    string                       `mapstructure:"project_id"`
	EnvID        string                       `mapstructure:"env_id"`
	ServiceID    string                       `mapstructure:"service_id"`
	Environments map[string]EnvironmentConfig `mapstructure:"environments"`
}

type EnvironmentConfig struct {
	ProjectID string `mapstructure:"project_id"`
	EnvID     string `mapstructure:"env_id"`
	ServiceID string `mapstructure:"service_id"`
}
