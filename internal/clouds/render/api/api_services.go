package api

import (
	"context"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/render/api/services"
)

func (c *Client) ListServices(ctx context.Context, input services.ListServicesInput) (services.ListServicesResponse, error) {
	var servicesList services.ListServicesResponse
	resp, err := c.NewGetRequest(ctx, c.URLf("/services")).WriteBodyTo(&servicesList).Do()
	return servicesList, MapResponseToError(resp, err)
}

func (c *Client) RetrieveService(ctx context.Context, serviceID string) (services.Service, error) {
	// TODO: think on adding this method to the http requests library (DoInto())
	//_, err := c.NewGetRequest(ctx, c.URLf("/services/%s", serviceID)).DoInto(&service)
	var service services.Service
	resp, err := c.NewGetRequest(ctx, c.URLf("/services/%s", serviceID)).WriteBodyTo(&service).Do()
	return service, MapResponseToError(resp, err)
}

func (c *Client) UpdateService(ctx context.Context, serviceID string, input services.UpdateServiceInput) error {
	resp, err := c.NewPatchRequest(ctx, c.URLf("/services/%s", serviceID), input).Do()
	return MapResponseToError(resp, err)
}
