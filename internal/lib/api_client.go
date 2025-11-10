package lib

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/AnotherFullstackDev/httpreqx"
)

type ApiClient struct {
	baseURL *url.URL
	*httpreqx.HttpClient
}

func NewProtectedApiClient(baseURL, apiKey string) (*ApiClient, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse base url: %w", err)
	}

	httpClient := httpreqx.NewHttpClient().
		SetBodyMarshaler(httpreqx.NewJSONBodyMarshaler()).
		SetBodyUnmarshaler(httpreqx.NewJSONBodyUnmarshaler()).
		SetHeaders(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
			"Content-Type":  "application/json",
			"Accept":        "application/json",
		}).
		SetDumpOnError().
		SetStackTraceEnabled(false)

	return &ApiClient{
		baseURL:    base,
		HttpClient: httpClient,
	}, nil
}

func MustNewProtectedApiClient(baseURL, apiKey string) *ApiClient {
	client, err := NewProtectedApiClient(baseURL, apiKey)
	if err != nil {
		log.Fatalf("could not create api client: %v", err)
	}
	return client
}

func (c *ApiClient) buildUrl(path string) *url.URL {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}

	return c.baseURL.JoinPath(segments...)
}

func (c *ApiClient) URL(path string) string {
	return c.buildUrl(path).String()
}

func (c *ApiClient) URLf(format string, a ...interface{}) string {
	path := fmt.Sprintf(format, a...)
	return c.buildUrl(path).String()
}

func (c *ApiClient) URLWithQuery(query url.Values, path string) string {
	u := c.buildUrl(path)
	u.RawQuery = query.Encode()
	return u.String()
}

func (c *ApiClient) URLWithQueryf(query url.Values, format string, a ...interface{}) string {
	path := fmt.Sprintf(format, a...)
	return c.URLWithQuery(query, path)
}
