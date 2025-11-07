package image

type BuildConfig struct {
	Cmd []string          `mapstructure:"cmd"`
	Env map[string]string `mapstructure:"env"`
	Dir string            `mapstructure:"dir"`
}

type Config struct {
	Repository string          `mapstructure:"repository"`
	Tag        string          `mapstructure:"tag"`
	Ghcr       *GhcrDestConfig `mapstructure:"ghcr"`
	Build      BuildConfig     `mapstructure:"build"`
}

// GhcrDestConfig -  Github container registry destination config
type GhcrDestConfig struct {
	Username   string `mapstructure:"username"`
	Owner      string `mapstructure:"owner"`
	Repository string `mapstructure:"repository"`
	Tag        string `mapstructure:"tag"`
}
