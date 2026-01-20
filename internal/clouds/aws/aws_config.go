package aws

type EcsConfig struct {
	ARN           string  `mapstructure:"arn"`
	ContainerName *string `mapstructure:"container_name"`
}
