package services

import (
	"encoding/json"
	"fmt"
	"time"
)

type ServiceType string

const (
	ServiceTypeStaticSite       ServiceType = "static_site"
	ServiceTypeWebService       ServiceType = "web_service"
	ServiceTypePrivateService   ServiceType = "private_service"
	ServiceTypeBackgroundWorker ServiceType = "background_worker"
	ServiceTypeCronJob          ServiceType = "cron_job"
)

type ServiceAutoDeploy string

const (
	ServiceAutoDeployYes ServiceAutoDeploy = "yes"
	ServiceAutoDeployNo  ServiceAutoDeploy = "no"
)

type ServiceNotifyOnFail string

const (
	ServiceNotifyOnFailDefault ServiceNotifyOnFail = "default"
	ServiceNotifyOnFailNotify  ServiceNotifyOnFail = "notify"
	ServiceNotifyOnFailIgnore  ServiceNotifyOnFail = "ignore"
)

type ServiceSuspended string

const (
	ServiceSuspendedSuspended    ServiceSuspended = "suspended"
	ServiceSuspendedNotSuspended ServiceSuspended = "not_suspended"
)

type ServicePreviewGeneration string

const (
	ServicePreviewGenerationOff       ServicePreviewGeneration = "off"
	ServicePreviewGenerationManual    ServicePreviewGeneration = "manual"
	ServicePreviewGenerationAutomatic ServicePreviewGeneration = "automatic"
)

type ServiceBuildPlan string

const (
	ServiceBuildPlanStarter     ServiceBuildPlan = "starter"
	ServiceBuildPlanPerformance ServiceBuildPlan = "performance"
)

type RenderSubdomainPolicy string

const (
	RenderSubdomainPolicyEnabled  RenderSubdomainPolicy = "enabled"
	RenderSubdomainPolicyDisabled RenderSubdomainPolicy = "disabled"
)

type ServiceRuntime string

const (
	ServiceRuntimeDocker ServiceRuntime = "docker"
	ServiceRuntimeElixir ServiceRuntime = "elixir"
	ServiceRuntimeGo     ServiceRuntime = "go"
	ServiceRuntimeNode   ServiceRuntime = "node"
	ServiceRuntimePython ServiceRuntime = "python"
	ServiceRuntimeRuby   ServiceRuntime = "ruby"
	ServiceRuntimeRust   ServiceRuntime = "rust"
	ServiceRuntimeImage  ServiceRuntime = "image"
)

type WebServiceProtocol string

const (
	WebServiceProtocolTCP WebServiceProtocol = "TCP"
	WebServiceProtocolUDP WebServiceProtocol = "UDP"
)

type ServicePlan string

// starter starter_plus standard standard_plus pro pro_plus pro_max pro_ultra free custom
const (
	ServicePlanStarter      ServicePlan = "starter"
	ServicePlanStarterPlus  ServicePlan = "starter_plus"
	ServicePlanStandard     ServicePlan = "standard"
	ServicePlanStandardPlus ServicePlan = "standard_plus"
	ServicePlanPro          ServicePlan = "pro"
	ServicePlanProPlus      ServicePlan = "pro_plus"
	ServicePlanProMax       ServicePlan = "pro_max"
	ServicePlanProUltra     ServicePlan = "pro_ultra"
	ServicePlanFree         ServicePlan = "free"
	ServicePlanCustom       ServicePlan = "custom"
)

type ServiceRegion string

// frankfurt oregon ohio singapore virginia
const (
	ServiceRegionFrankfurt ServiceRegion = "frankfurt"
	ServiceRegionOregon    ServiceRegion = "oregon"
	ServiceRegionOhio      ServiceRegion = "ohio"
	ServiceRegionSingapore ServiceRegion = "singapore"
	ServiceRegionVirginia  ServiceRegion = "virginia"
)

type ServiceRegistry string

// GITHUB GITLAB DOCKER GOOGLE_ARTIFACT AWS_ECR
const (
	ServiceRegistryGITHUB          ServiceRegistry = "GITHUB"
	ServiceRegistryGITLAB          ServiceRegistry = "GITLAB"
	ServiceRegistryDOCKER          ServiceRegistry = "DOCKER"
	ServiceRegistryGOOGLE_ARTIFACT ServiceRegistry = "GOOGLE_ARTIFACT"
	ServiceRegistryAWS_ECR         ServiceRegistry = "AWS_ECR"
)

type IpAllowList struct {
	CidrBlock   string `json:"cidrBlock"`
	Description string `json:"description"`
}

type ParentServer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Previews struct {
	Generation ServicePreviewGeneration `json:"generation"`
}

type StaticSiteDetails struct {
	BuildCommand          string                `json:"buildCommand"`
	IpAllowList           []IpAllowList         `json:"ipAllowList"`
	ParentServer          ParentServer          `json:"parentServer"`
	PublishPath           string                `json:"publishPath"`
	Previews              Previews              `json:"previews"`
	Url                   string                `json:"url"`
	BuildPlan             ServiceBuildPlan      `json:"buildPlan"`
	RenderSubdomainPolicy RenderSubdomainPolicy `json:"renderSubdomainPolicy"`
}

type ServiceAutoscalingCriteria struct {
	Enabled    bool  `json:"enabled"`
	Percentage int64 `json:"percentage"`
}

type ServiceAutoscaling struct {
	Enabled  bool  `json:"enabled"`
	Min      int64 `json:"min"`
	Max      int64 `json:"max"`
	Criteria struct {
		Cpu    ServiceAutoscalingCriteria `json:"cpu"`
		Memory ServiceAutoscalingCriteria `json:"memory"`
	} `json:"criteria"`
}

type ServiceDisc struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SizeGB    string `json:"sizeGB"`
	MountPath string `json:"mountPath"`
}

type OpenPort struct {
	Port     int64              `json:"port"`
	Protocol WebServiceProtocol `json:"protocol"`
}

type WebServiceDetails struct {
	Autoscaling                ServiceAutoscaling     `json:"autoscaling"`
	Cache                      ServiceCache           `json:"cache"`
	Disc                       ServiceDisc            `json:"dist"`
	Env                        ServiceRuntime         `json:"env"`
	EnvSpecificDetails         RuntimeSpecificDetails `json:"envSpecificDetails"`
	HealthCheckPath            string                 `json:"healthCheckPath"`
	IpAllowList                []IpAllowList          `json:"ipAllowList"`
	MaintenanceMode            MaintenanceMode        `json:"maintenanceMode"`
	NumInstances               int64                  `json:"numInstances"`
	OpenPorts                  []OpenPort             `json:"openPorts"`
	ParentServer               ParentServer           `json:"parentServer"`
	Plan                       ServicePlan            `json:"plan"`
	PullRequestPreviewsEnabled string                 `json:"pullRequestPreviewsEnabled"` // yes | no, deprecated - use previews
	Previews                   Previews               `json:"previews"`
	Region                     ServiceRegion          `json:"region"`
	SshAddress                 string                 `json:"sshAddress"`
	Url                        string                 `json:"url"`
	BuildPlan                  ServiceBuildPlan       `json:"buildPlan"`
	MaxShutdownDelaySeconds    int64                  `json:"maxShutdownDelaySeconds"`
	RenderSubdomainPolicy      RenderSubdomainPolicy  `json:"renderSubdomainPolicy"`
}

type PrivateServiceDetails struct {
	Autoscaling                ServiceAutoscaling     `json:"autoscaling"`
	Disc                       ServiceDisc            `json:"disc"`
	Env                        ServiceRuntime         `json:"env"`
	EnvSpecificDetails         RuntimeSpecificDetails `json:"envSpecificDetails"`
	NumInstances               int64                  `json:"numInstances"`
	OpenPorts                  []OpenPort             `json:"openPorts"`
	ParentServer               ParentServer           `json:"parentServer"`
	Plan                       ServicePlan            `json:"plan"`
	PullRequestPreviewsEnabled string                 `json:"pullRequestPreviewsEnabled"` // yes | no, deprecated - use previews
	Previews                   Previews               `json:"previews"`
	Region                     ServiceRegion          `json:"region"`
	Runtime                    ServiceRuntime         `json:"runtime"`
	SshAddress                 string                 `json:"sshAddress"`
	Url                        string                 `json:"url"`
	BuildPlan                  ServiceBuildPlan       `json:"buildPlan"`
	MaxShutdownDelaySeconds    int64                  `json:"maxShutdownDelaySeconds"`
}

type BackgroundWorkerDetails struct {
	Autoscaling                ServiceAutoscaling     `json:"autoscaling"`
	Disc                       ServiceDisc            `json:"disc"`
	Env                        ServiceRuntime         `json:"env"`
	EnvSpecificDetails         RuntimeSpecificDetails `json:"envSpecificDetails"`
	NumInstances               int64                  `json:"numInstances"`
	ParentServer               ParentServer           `json:"parentServer"`
	Plan                       ServicePlan            `json:"plan"`
	PullRequestPreviewsEnabled string                 `json:"pullRequestPreviewsEnabled"` // yes | no, deprecated - use previews
	Previews                   Previews               `json:"previews"`
	Region                     ServiceRegion          `json:"region"`
	Runtime                    ServiceRuntime         `json:"runtime"`
	SshAddress                 string                 `json:"sshAddress"`
	BuildPlan                  ServiceBuildPlan       `json:"buildPlan"`
	MaxShutdownDelaySeconds    int64                  `json:"maxShutdownDelaySeconds"`
}

type CronJobDetails struct {
	Env                ServiceRuntime         `json:"env"`
	EnvSpecificDetails RuntimeSpecificDetails `json:"envSpecificDetails"`
	LastSuccessfulRun  time.Time              `json:"lastSuccessfulRun"`
	Plan               ServicePlan            `json:"plan"`
	Region             ServiceRegion          `json:"region"`
	Runtime            ServiceRuntime         `json:"runtime"`
	Schedule           string                 `json:"schedule"`
	BuildPlan          ServiceBuildPlan       `json:"buildPlan"`
}

type RuntimeSpecificDetails json.RawMessage

type DockerDetails struct {
	DockerCommand      string             `json:"dockerCommand"`
	DockerContext      string             `json:"dockerContext"`
	DockerfilePath     string             `json:"dockerfilePath"`
	RegistryCredential RegistryCredential `json:"registryCredential"`
}

type NativeEnvironmentDetails struct {
	BuildCommand     string `json:"buildCommand"`
	StartCommand     string `json:"startCommand"`
	PreDeployCommand string `json:"preDeployCommand"`
}

func (d *RuntimeSpecificDetails) DockerDetails() (*DockerDetails, error) {
	var details DockerDetails
	if err := json.Unmarshal(*d, &details); err != nil {
		return nil, fmt.Errorf("could not unmarshal env specific details as DockerDetails: %w", err)
	}
	return &details, nil
}

func (d *RuntimeSpecificDetails) NativeEnvironmentDetails() (*NativeEnvironmentDetails, error) {
	var details NativeEnvironmentDetails
	if err := json.Unmarshal(*d, &details); err != nil {
		return nil, fmt.Errorf("could not unmarshal env specific details as NativeEnvironmentDetails: %w", err)
	}
	return &details, nil
}

type ServiceDetails json.RawMessage

// UnmarshalJSON makes ServiceDetails behave like json.RawMessage.
// It stores the raw JSON representation as-is.
func (s *ServiceDetails) UnmarshalJSON(data []byte) error {
	if s == nil {
		return fmt.Errorf("ServiceDetails: UnmarshalJSON on nil receiver")
	}

	// Copy input to avoid retaining references to the input buffer.
	// This matches json.RawMessage behavior.
	*s = append((*s)[:0], data...)
	return nil
}

// MarshalJSON returns the stored raw JSON bytes.
// If nil, it encodes as JSON null.
func (s ServiceDetails) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return s, nil
}

func (s *ServiceDetails) DecodeInto(v any) error {
	if err := json.Unmarshal(*s, v); err != nil {
		return fmt.Errorf("could not unmarshal service details: %w", err)
	}
	return nil
}

func (d *ServiceDetails) StaticSiteDetails() (*StaticSiteDetails, error) {
	var details StaticSiteDetails
	if err := json.Unmarshal(*d, &details); err != nil {
		return nil, fmt.Errorf("could not unmarshal service details as StaticSiteDetails: %w", err)
	}
	return &details, nil
}

func (d *ServiceDetails) WebServiceDetails() (*WebServiceDetails, error) {
	var details WebServiceDetails
	if err := json.Unmarshal(*d, &details); err != nil {
		return nil, fmt.Errorf("could not unmarshal service details as WebServiceDetails: %w", err)
	}
	return &details, nil
}

func (d *ServiceDetails) PrivateServiceDetails() (*PrivateServiceDetails, error) {
	var details PrivateServiceDetails
	if err := json.Unmarshal(*d, &details); err != nil {
		return nil, fmt.Errorf("could not unmarshal service details as PrivateServiceDetails: %w", err)
	}
	return &details, nil
}

func (d *ServiceDetails) BackgroundWorkerDetails() (*BackgroundWorkerDetails, error) {
	var details BackgroundWorkerDetails
	if err := json.Unmarshal(*d, &details); err != nil {
		return nil, fmt.Errorf("could not unmarshal service details as BackgroundWorkerDetails: %w", err)
	}
	return &details, nil
}

func (d *ServiceDetails) CronJobDetails() (*CronJobDetails, error) {
	var details CronJobDetails
	if err := json.Unmarshal(*d, &details); err != nil {
		return nil, fmt.Errorf("could not unmarshal service details as CronJobDetails: %w", err)
	}
	return &details, nil
}

type RegistryCredential struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Registry  ServiceRegistry `json:"registry"`
	Username  string          `json:"username"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type ServiceBuildFilter struct {
	Paths        []string `json:"paths"`
	IgnoredPaths []string `json:"ignoredPaths"`
}

type Service struct {
	Id                 string              `json:"id"`
	AutoDeploy         ServiceAutoDeploy   `json:"autoDeploy"`
	Branch             string              `json:"branch"`
	BuildFilter        ServiceBuildFilter  `json:"buildFilter"`
	CreatedAt          time.Time           `json:"createdAt"`
	DashboardUrl       string              `json:"dashboardUrl"`
	EnvironmentId      string              `json:"environmentId"`
	ImagePath          string              `json:"imagePath"`
	Name               string              `json:"name"`
	NotifyOnFail       ServiceNotifyOnFail `json:"notifyOnFail"`
	OwnerId            string              `json:"ownerId"`
	RegistryCredential RegistryCredential  `json:"registryCredential"`
	Repo               string              `json:"repo"`
	RootDir            string              `json:"rootDir"`
	Slug               string              `json:"slug"`
	Suspended          ServiceSuspended    `json:"suspended"`
	Suspenders         []string            `json:"suspenders"`
	Type               ServiceType         `json:"type"`
	UpdatedAt          time.Time           `json:"updatedAt"`
	ServiceDetails     ServiceDetails      `json:"serviceDetails"`
}
