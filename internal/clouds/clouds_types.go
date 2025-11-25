package clouds

import (
	"context"
)

type ImageRegistry interface {
	GetImageRef() (string, error)
}

type CloudProvider interface {
	DeployServiceFromImage(ctx context.Context, registry ImageRegistry) error
}
