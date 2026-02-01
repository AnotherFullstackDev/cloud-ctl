package registry

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/google"
)

// GcpArtifactRegistryConfig - GCP Artifact Registry / GCR destination config
type GcpArtifactRegistryConfig string

type GcpArtifactRegistry struct {
	config GcpArtifactRegistryConfig
}

func NewGcpArtifactRegistry(config GcpArtifactRegistryConfig) Registry {
	return &GcpArtifactRegistry{config}
}

func (r *GcpArtifactRegistry) GetAuthType() AuthType {
	return AuthTypeKeychain
}

func (r *GcpArtifactRegistry) GetAuthentication() (authn.Authenticator, error) {
	return nil, nil
}

func (r *GcpArtifactRegistry) ResetAuthentication() error { return nil }

func (r *GcpArtifactRegistry) GetKeychain() authn.Keychain {
	// google.Keychain automatically handles:
	// 1. Application Default Credentials (ADC) via GOOGLE_APPLICATION_CREDENTIALS env var
	// 2. gcloud CLI credentials (fallback)
	// 3. Compute Engine/GKE/Cloud Run service account credentials
	return google.Keychain
}

func (r *GcpArtifactRegistry) GetImageRef() (string, error) {
	imageID := string(r.config)

	// Check if it's Artifact Registry format: <region>-docker.pkg.dev/<project>/<repository>/<image>:<tag>
	if strings.Contains(imageID, ".pkg.dev") {
		return r.validateArtifactRegistryFormat(imageID)
	}

	// Check if it's GCR format: gcr.io/<project>/<image>:<tag> or <region>.gcr.io/<project>/<image>:<tag>
	if strings.Contains(imageID, "gcr.io") {
		return r.validateGCRFormat(imageID)
	}

	return "", fmt.Errorf("%w - invalid GCP registry image format: %s, expected Artifact Registry (<region>-docker.pkg.dev/<project>/<repository>/<image>:<tag>) or GCR (gcr.io/<project>/<image>:<tag>)", lib.BadUserInputError, imageID)
}

// validateArtifactRegistryFormat validates format: <region>-docker.pkg.dev/<project>/<repository>/<image>:<tag>
func (r *GcpArtifactRegistry) validateArtifactRegistryFormat(imageID string) (string, error) {
	parts := strings.Split(imageID, "/")
	// Expected: [<region>-docker.pkg.dev, <project>, <repository>, <image>:<tag>]
	if len(parts) != 4 {
		return "", fmt.Errorf("%w - invalid Artifact Registry image format: %s, expected format: <region>-docker.pkg.dev/<project>/<repository>/<image>:<tag>", lib.BadUserInputError, imageID)
	}
	slog.Debug("split Artifact Registry image into parts", "parts", parts)

	registryHost := parts[0]
	if !strings.HasSuffix(registryHost, "-docker.pkg.dev") {
		return "", fmt.Errorf("%w - invalid Artifact Registry host: %s, expected format: <region>-docker.pkg.dev", lib.BadUserInputError, registryHost)
	}

	imageAndTag := parts[3]
	tagParts := strings.SplitN(imageAndTag, ":", 2)
	if len(tagParts) != 2 || tagParts[1] == "" {
		return "", fmt.Errorf("%w - invalid Artifact Registry image format: %s, missing tag", lib.BadUserInputError, imageID)
	}
	slog.Debug("split into image and tag parts", "image_tag_parts", tagParts)

	return imageID, nil
}

// validateGCRFormat validates format: gcr.io/<project>/<image>:<tag> or <region>.gcr.io/<project>/<image>:<tag>
func (r *GcpArtifactRegistry) validateGCRFormat(imageID string) (string, error) {
	parts := strings.Split(imageID, "/")
	// Expected: [gcr.io or <region>.gcr.io, <project>, <image>:<tag>] or more nested paths
	if len(parts) < 3 {
		return "", fmt.Errorf("%w - invalid GCR image format: %s, expected format: gcr.io/<project>/<image>:<tag> or <region>.gcr.io/<project>/<image>:<tag>", lib.BadUserInputError, imageID)
	}
	slog.Debug("split GCR image into parts", "parts", parts)

	registryHost := parts[0]
	if registryHost != "gcr.io" && !strings.HasSuffix(registryHost, ".gcr.io") {
		return "", fmt.Errorf("%w - invalid GCR host: %s, expected gcr.io or <region>.gcr.io", lib.BadUserInputError, registryHost)
	}

	// The last part should contain the tag
	lastPart := parts[len(parts)-1]
	tagParts := strings.SplitN(lastPart, ":", 2)
	if len(tagParts) != 2 || tagParts[1] == "" {
		return "", fmt.Errorf("%w - invalid GCR image format: %s, missing tag", lib.BadUserInputError, imageID)
	}
	slog.Debug("split into image and tag parts", "image_tag_parts", tagParts)

	return imageID, nil
}
