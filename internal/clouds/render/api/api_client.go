package api

import (
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
)

type Client struct {
	*lib.ApiClient
}

func MustNewClient(baseURL, apiKey string) *Client {
	client := lib.MustNewProtectedApiClient(baseURL, apiKey)
	return &Client{ApiClient: client}
}
