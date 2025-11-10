package api

import (
	"context"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/render/api/deploys"
)

func (c *Client) TriggerDeploy(ctx context.Context, serviceID string, input deploys.TriggerDeployInput) error {
	_, err := c.NewPostRequest(ctx, c.URLf("/services/%s/deploys", serviceID), input).Do()
	return err
}
