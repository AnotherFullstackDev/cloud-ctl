package render

import (
	"context"
	"fmt"
)

type Provider struct {
	serviceID string
	config    Config
	api       *ApiClient
}

func MustNewProvider(serviceID string, cfg Config) *Provider {
	api := MustNewApiClient("https://api.render.com")

	return &Provider{
		serviceID: serviceID,
		config:    cfg,
		api:       api,
	}
}

func (p *Provider) DeployService(ctx context.Context) error {
	err := p.api.DeployService(ctx, p.config.ServiceID, p.config.DeploymentKey)
	if err != nil {
		return fmt.Errorf("deploying service %s: %w", p.config.ServiceID, err)
	}
	return nil
}
