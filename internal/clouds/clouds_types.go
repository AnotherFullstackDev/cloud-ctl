package clouds

import (
	"context"
)

type ImageRegistry interface {
	GetImageRef() string
}

type CloudProvider interface {
	DeployServiceFromImage(ctx context.Context, registry ImageRegistry) error
}
