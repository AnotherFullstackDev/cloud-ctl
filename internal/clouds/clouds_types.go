package clouds

import "context"

type CloudProvider interface {
	DeployService(ctx context.Context) error
}
