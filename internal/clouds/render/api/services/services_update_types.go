package services

type UpdateServiceEnvSpecificDetails interface {
	isUpdateServiceEnvSpecificDetails()
}

type MaintenanceMode struct {
	Enabled bool   `json:"enabled"`
	Uri     string `json:"uri"`
}

type ServiceCache struct {
	Profile string `json:"profile"`
}

type UpdateServiceDockerSpecificDetails struct {
	DockerCommand         string `json:"dockerCommand,omitempty"`
	DockerContext         string `json:"dockerContext,omitempty"`
	DockerfilePath        string `json:"dockerfilePath,omitempty"`
	RegistryCredentialsID string `json:"registryCredentialsId,omitempty"`
}

func (d *UpdateServiceDockerSpecificDetails) isUpdateServiceEnvSpecificDetails() {}

type UpdateServiceNativeEnvironmentSpecificDetails struct {
	BuildCommand string `json:"buildCommand,omitempty"`
	StartCommand string `json:"startCommand,omitempty"`
}

func (d *UpdateServiceNativeEnvironmentSpecificDetails) isUpdateServiceEnvSpecificDetails() {}

type UpdateServiceDetails interface {
	isUpdateServiceDetails()
}

type UpdateServiceDetailsStaticSite struct {
	BuildCommand               string                 `json:"buildCommand,omitempty"`
	PublishPath                string                 `json:"publishPath,omitempty"`
	PullRequestPreviewsEnabled string                 `json:"pullRequestPreviewsEnabled,omitempty"` // yes | no, deprecated - use previews
	Previews                   *Previews              `json:"previews,omitempty"`
	RenderSubdomainPolicy      *RenderSubdomainPolicy `json:"renderSubdomainPolicy,omitempty"`
	IpAllowList                *IpAllowList           `json:"ipAllowList,omitempty"`
}

func (d *UpdateServiceDetailsStaticSite) isUpdateServiceDetails() {}

type UpdateServiceDetailsWebService struct {
	EnvSpecificDetails         UpdateServiceEnvSpecificDetails `json:"envSpecificDetails,omitempty"`
	MaintenanceMode            *MaintenanceMode                `json:"maintenanceMode"`
	Plan                       ServicePlan                     `json:"plan"`
	PreDeployCommand           string                          `json:"preDeployCommand,omitempty"`
	PullRequestPreviewsEnabled string                          `json:"pullRequestPreviewsEnabled,omitempty"` // yes | no, deprecated - use previews
	Previews                   *Previews                       `json:"previews,omitempty"`
	Runtime                    ServiceRuntime                  `json:"runtime"`
	MaxShutdownDelaySeconds    int64                           `json:"maxShutdownDelaySeconds"`
	RenderSubdomainPolicy      RenderSubdomainPolicy           `json:"renderSubdomainPolicy,omitempty"`
	IpAllowList                *IpAllowList                    `json:"ipAllowList,omitempty"`
	Cache                      *ServiceCache                   `json:"cache,omitempty"`
}

func (d *UpdateServiceDetailsWebService) isUpdateServiceDetails() {}

type UpdateServiceDetailsPrivateService struct {
	EnvSpecificDetails         UpdateServiceEnvSpecificDetails `json:"envSpecificDetails,omitempty"`
	Plan                       ServicePlan                     `json:"plan,omitempty"`
	PreDeployCommand           string                          `json:"preDeployCommand,omitempty"`
	PullRequestPreviewsEnabled string                          `json:"pullRequestPreviewsEnabled,omitempty"` // yes | no, deprecated - use previews
	Previews                   *Previews                       `json:"previews,omitempty"`
	Runtime                    ServiceRuntime                  `json:"runtime"`
	MaxShutdownDelaySeconds    int64                           `json:"maxShutdownDelaySeconds"`
}

func (d *UpdateServiceDetailsPrivateService) isUpdateServiceDetails() {}

type UpdateServiceDetailsBackgroundWorker struct {
	EnvSpecificDetails         UpdateServiceEnvSpecificDetails `json:"envSpecificDetails,omitempty"`
	Plan                       ServicePlan                     `json:"plan,omitempty"`
	PreDeployCommand           string                          `json:"preDeployCommand,omitempty"`
	PullRequestPreviewsEnabled string                          `json:"pullRequestPreviewsEnabled,omitempty"` // yes | no, deprecated - use previews
	Previews                   *Previews                       `json:"previews,omitempty"`
	Runtime                    ServiceRuntime                  `json:"runtime"`
	MaxShutdownDelaySeconds    int64                           `json:"maxShutdownDelaySeconds"`
}

func (d *UpdateServiceDetailsBackgroundWorker) isUpdateServiceDetails() {}

type UpdateServiceDetailsCronJob struct {
	EnvSpecificDetails UpdateServiceEnvSpecificDetails `json:"envSpecificDetails,omitempty"`
	Plan               ServicePlan                     `json:"plan,omitempty"`
	Schedule           string                          `json:"schedule,omitempty"`
	Runtime            ServiceRuntime                  `json:"runtime,omitempty"`
}

type UpdateServiceImage struct {
	OwnerID              string `json:"ownerId"`
	RegistryCredentialID string `json:"registryCredentialId,omitempty"`
	ImagePath            string `json:"imagePath"`
}

type UpdateServiceInput struct {
	AutoDeploy     *ServiceAutoDeploy   `json:"autoDeploy,omitempty"`
	Repo           string               `json:"repo,omitempty"`
	Branch         *string              `json:"branch,omitempty"`
	Image          *UpdateServiceImage  `json:"image,omitempty"`
	Name           string               `json:"name,omitempty"`
	BuildFilter    *ServiceBuildFilter  `json:"buildFilter,omitempty"`
	RootDir        string               `json:"rootDir,omitempty"`
	ServiceDetails UpdateServiceDetails `json:"serviceDetails,omitempty"`
}
