package container_image

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/build/pipeline"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/container_image/registry"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/placeholders"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/term"
)

type Service struct {
	config               Config
	registry             registry.Registry
	placeholdersResolver *placeholders.Service
	pipelineService      *pipeline.Service
}

// recompressedLayer implements v1.Layer with proper DiffID support for recompressed layers.
type recompressedLayer struct {
	compressedData   []byte
	uncompressedData []byte
	mediaType        types.MediaType
	diffID           v1.Hash
	digest           v1.Hash
}

func newRecompressedLayer(compressedData, uncompressedData []byte, mediaType types.MediaType) (v1.Layer, error) {
	// Compute DiffID (hash of uncompressed content)
	diffIDHash := sha256.Sum256(uncompressedData)
	diffID := v1.Hash{
		Algorithm: "sha256",
		Hex:       hex.EncodeToString(diffIDHash[:]),
	}

	// Compute Digest (hash of compressed content)
	digestHash := sha256.Sum256(compressedData)
	digest := v1.Hash{
		Algorithm: "sha256",
		Hex:       hex.EncodeToString(digestHash[:]),
	}

	return &recompressedLayer{
		compressedData:   compressedData,
		uncompressedData: uncompressedData,
		mediaType:        mediaType,
		diffID:           diffID,
		digest:           digest,
	}, nil
}

func (l *recompressedLayer) Digest() (v1.Hash, error) {
	return l.digest, nil
}

func (l *recompressedLayer) DiffID() (v1.Hash, error) {
	return l.diffID, nil
}

func (l *recompressedLayer) Compressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.compressedData)), nil
}

func (l *recompressedLayer) Uncompressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.uncompressedData)), nil
}

func (l *recompressedLayer) Size() (int64, error) {
	return int64(len(l.compressedData)), nil
}

func (l *recompressedLayer) MediaType() (types.MediaType, error) {
	return l.mediaType, nil
}

func NewService(config Config, registry registry.Registry, resolver *placeholders.Service, pipeline *pipeline.Service) *Service {
	return &Service{
		config,
		registry,
		resolver,
		pipeline,
	}
}

func (s *Service) GetRegistry() registry.Registry {
	return s.registry
}

func (s *Service) BuildImage(ctx context.Context) error {
	if s.config.Build == nil {
		slog.InfoContext(ctx, "no image build configured")
		return nil
	}

	switch {
	case len(s.config.Build.Cmd) > 0:
		return s.buildImageViaCmd(ctx, s.config.Build.Cmd, s.config.Build.Env, s.config.Build.Dir)
	case s.config.Build.Pipeline != nil:
		return s.pipelineService.ProcessPipeline(ctx, s.config.Image)
	}

	return fmt.Errorf("no image build strategy configured")
}

func (s *Service) buildImageViaCmd(ctx context.Context, cmd []string, env map[string]string, dir string) error {
	if len(cmd) <= 0 {
		return fmt.Errorf("no command provided for image build")
	}

	resolvedCmd := make([]string, 0, len(cmd))
	for _, c := range cmd {
		resolvedC, err := s.placeholdersResolver.ResolvePlaceholders(c)
		if err != nil {
			return fmt.Errorf("resolving placeholder in build command '%s': %w", c, err)
		}
		resolvedCmd = append(resolvedCmd, resolvedC)
	}

	args := resolvedCmd
	if len(args) == 1 {
		args = []string{"sh", "-c", resolvedCmd[0]}
	}

	environment := os.Environ()
	for k, v := range env {
		resolvedValue, err := s.placeholdersResolver.ResolvePlaceholders(v)
		if err != nil {
			return fmt.Errorf("resolving placeholder in build env var '%s'='%s': %w", k, v, err)
		}

		environment = append(environment, fmt.Sprintf("%s=%s", k, resolvedValue))
	}

	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.Env = environment
	command.Dir = dir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	slog.InfoContext(ctx, "running image build command", "args", command.Args)

	if err := command.Run(); err != nil {
		return fmt.Errorf("running image build command: %w", err)
	}

	return nil
}

// recompressImage recompresses all layers of an image using the specified compression algorithm and level.
// This can significantly reduce image size and improve push/pull performance when using zstd compression.
func (s *Service) recompressImage(ctx context.Context, img v1.Image, algorithm CompressionAlgorithm, level int) (v1.Image, error) {
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("getting image layers: %w", err)
	}

	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("getting image config: %w", err)
	}

	// Calculate original image size
	var originalSize int64
	for _, layer := range layers {
		size, err := layer.Size()
		if err != nil {
			slog.WarnContext(ctx, "failed to get layer size", "error", err)
			continue
		}
		originalSize += size
	}

	// Start with an empty image and add recompressed layers
	result := empty.Image

	// Set the config
	result, err = mutate.ConfigFile(result, configFile)
	if err != nil {
		return nil, fmt.Errorf("setting image config: %w", err)
	}

	slog.InfoContext(ctx, "recompressing image layers",
		"algorithm", algorithm,
		"level", level,
		"layer_count", len(layers),
		"original_size_mb", fmt.Sprintf("%.2f", float64(originalSize)/(1024*1024)))

	// Track new size during recompression
	var newSize int64

	for i, layer := range layers {
		recompressedLayer, layerSize, err := s.recompressLayer(ctx, layer, algorithm, level, i)
		if err != nil {
			return nil, fmt.Errorf("recompressing layer %d: %w", i, err)
		}
		newSize += layerSize

		result, err = mutate.AppendLayers(result, recompressedLayer)
		if err != nil {
			return nil, fmt.Errorf("appending recompressed layer %d: %w", i, err)
		}
	}

	var savingsPercent float64
	if originalSize > 0 {
		savingsPercent = (1 - float64(newSize)/float64(originalSize)) * 100
	}

	slog.InfoContext(ctx, "image recompression complete",
		"original_size_mb", fmt.Sprintf("%.2f", float64(originalSize)/(1024*1024)),
		"new_size_mb", fmt.Sprintf("%.2f", float64(newSize)/(1024*1024)),
		"savings_percent", fmt.Sprintf("%.1f%%", savingsPercent))

	return result, nil
}

// recompressLayer recompresses a single layer with the specified compression algorithm.
// Returns the recompressed layer, its size in bytes, and any error.
func (s *Service) recompressLayer(ctx context.Context, layer v1.Layer, algorithm CompressionAlgorithm, level int, layerIndex int) (v1.Layer, int64, error) {
	// Get uncompressed content
	uncompressed, err := layer.Uncompressed()
	if err != nil {
		return nil, 0, fmt.Errorf("getting uncompressed layer: %w", err)
	}
	defer uncompressed.Close()

	// Read all uncompressed data
	uncompressedData, err := io.ReadAll(uncompressed)
	if err != nil {
		return nil, 0, fmt.Errorf("reading uncompressed layer: %w", err)
	}

	// Determine media type based on compression algorithm
	var mediaType types.MediaType
	var compressedData []byte

	switch algorithm {
	case CompressionZstd:
		mediaType = types.OCILayerZStd
		compressedData, err = compressWithZstd(uncompressedData, level)
		if err != nil {
			return nil, 0, fmt.Errorf("compressing with zstd: %w", err)
		}
	case CompressionGzip:
		mediaType = types.OCILayer
		compressedData, err = compressWithGzip(uncompressedData, level)
		if err != nil {
			return nil, 0, fmt.Errorf("compressing with gzip: %w", err)
		}
	case CompressionNone:
		mediaType = types.OCIUncompressedLayer
		compressedData = uncompressedData
	default:
		return nil, 0, fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}

	originalSize, _ := layer.Size()
	newSize := int64(len(compressedData))
	slog.DebugContext(ctx, "recompressed layer",
		"layer_index", layerIndex,
		"algorithm", algorithm,
		"original_size", originalSize,
		"new_size", newSize,
		"ratio", fmt.Sprintf("%.2f%%", float64(newSize)/float64(len(uncompressedData))*100))

	// Create a new layer from the compressed data with proper DiffID
	newLayer, err := newRecompressedLayer(compressedData, uncompressedData, mediaType)
	if err != nil {
		return nil, 0, fmt.Errorf("creating recompressed layer: %w", err)
	}

	return newLayer, newSize, nil
}

// compressWithZstd compresses data using zstd algorithm at the specified level.
func compressWithZstd(data []byte, level int) ([]byte, error) {
	// Map level to zstd encoder level (1-22, with reasonable defaults)
	encoderLevel := zstd.EncoderLevelFromZstd(level)
	if level <= 0 {
		encoderLevel = zstd.SpeedDefault // level 3
	}

	var buf bytes.Buffer
	encoder, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(encoderLevel))
	if err != nil {
		return nil, fmt.Errorf("creating zstd encoder: %w", err)
	}

	_, err = encoder.Write(data)
	if err != nil {
		encoder.Close()
		return nil, fmt.Errorf("writing to zstd encoder: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing zstd encoder: %w", err)
	}

	return buf.Bytes(), nil
}

// compressWithGzip compresses data using gzip algorithm at the specified level.
func compressWithGzip(data []byte, level int) ([]byte, error) {
	// Map level to gzip level (1-9)
	if level <= 0 {
		level = gzip.DefaultCompression
	}
	if level > 9 {
		level = 9
	}

	var buf bytes.Buffer
	encoder, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, fmt.Errorf("creating gzip encoder: %w", err)
	}

	_, err = encoder.Write(data)
	if err != nil {
		encoder.Close()
		return nil, fmt.Errorf("writing to gzip encoder: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip encoder: %w", err)
	}

	return buf.Bytes(), nil
}

// TODO: add check for image architecture compatibility with target registry/platform
func (s *Service) PushImage(ctx context.Context) error {
	destRef, err := s.registry.GetImageRef()
	if err != nil {
		return fmt.Errorf("getting image reference from registry: %w", err)
	}
	if destRef == "" {
		return fmt.Errorf("container registry returned empty image reference")
	}

	resolvedImage, err := s.placeholdersResolver.ResolvePlaceholders(s.config.Image)
	srcRef, err := name.NewTag(resolvedImage)
	if err != nil {
		return fmt.Errorf("parsing source image tag: %w", err)
	}

	image, err := daemon.Image(srcRef, daemon.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("getting image from local daemon: %w", err)
	}

	// Apply compression if configured
	if s.config.Compression != nil && s.config.Compression.Algorithm != "" {
		level := s.config.Compression.Level
		if level <= 0 {
			// Set sensible defaults based on algorithm
			switch s.config.Compression.Algorithm {
			case CompressionZstd:
				level = 3 // Good balance for zstd per AWS testing
			case CompressionGzip:
				level = 6 // Default gzip level
			}
		}

		image, err = s.recompressImage(ctx, image, s.config.Compression.Algorithm, level)
		if err != nil {
			return fmt.Errorf("recompressing image with %s: %w", s.config.Compression.Algorithm, err)
		}
	}

	destTag, err := name.NewTag(destRef)
	if err != nil {
		return fmt.Errorf("parsing destination image tag: %w", err)
	}

	// Determine authentication method based on registry auth type
	authType := s.registry.GetAuthType()
	var authOption remote.Option
	if authType == registry.AuthTypeKeychain {
		keychain := s.registry.GetKeychain()
		authOption = remote.WithAuthFromKeychain(keychain)
	} else {
		auth, err := s.registry.GetAuthentication()
		if err != nil {
			return fmt.Errorf("getting registry authentication: %w", err)
		}
		authOption = remote.WithAuth(auth)
	}

	var stdout io.Writer = os.Stdout
	stderr := os.Stderr
	tty := false

	if f, ok := stdout.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		tty = true
	}

	imageConfig, err := image.ConfigFile()
	if err != nil {
		return fmt.Errorf("getting image config file: %w", err)
	}

	slog.InfoContext(ctx, "pushing image to remote registry",
		"source", srcRef,
		"dest", destTag,
		"os", imageConfig.OS,
		"architecture", imageConfig.Architecture)

	startTime := time.Now()
	for {
		progressChan := make(chan v1.Update, 32)

		go func() {
			var lastUpdateTime time.Time
			for update := range progressChan {
				if !tty {
					continue
				}

				if update.Error != nil {
					fmt.Fprintf(stderr, "Error: %v\n", update.Error)
					continue
				}
				if update.Total <= 0 {
					continue
				}
				if time.Since(lastUpdateTime) <= 500*time.Millisecond {
					continue
				}
				lastUpdateTime = time.Now()

				percentage := float64(update.Complete) / float64(update.Total) * 100

				fmt.Fprintf(stdout, "Image push: %.2f%% complete\n", percentage)
			}
		}()

		maxUploadJobs := int(math.Min(16, float64(runtime.NumCPU())))
		options := []remote.Option{
			remote.WithContext(ctx),
			authOption,
			remote.WithProgress(progressChan),
			remote.WithJobs(maxUploadJobs),
			remote.WithPlatform(v1.Platform{
				Architecture: imageConfig.Architecture,
				OS:           imageConfig.OS,
				OSFeatures:   imageConfig.OSFeatures,
				OSVersion:    imageConfig.OSVersion,
				Variant:      imageConfig.Variant,
			}),
		}
		if err := remote.Write(destTag, image, options...); err != nil {
			var registryErr *transport.Error
			if errors.As(err, &registryErr) {
				isUnauthorizedErr := false
				if registryErr.StatusCode == http.StatusUnauthorized || registryErr.StatusCode == http.StatusForbidden {
					isUnauthorizedErr = true
				}
				for _, desc := range registryErr.Errors {
					if desc.Code == transport.UnauthorizedErrorCode || desc.Code == transport.DeniedErrorCode {
						isUnauthorizedErr = true
					}
				}
				if isUnauthorizedErr {
					slog.WarnContext(ctx, "unauthorized error pushing image to registry, resetting authentication and retrying", "error", err)

					err = s.registry.ResetAuthentication()
					if err != nil {
						return fmt.Errorf("resetting registry authentication after unauthorized error: %w", err)
					}
					// Only refresh auth option for authenticator type; keychain handles refresh internally
					if authType == registry.AuthTypeAuthenticator {
						auth, err := s.registry.GetAuthentication()
						if err != nil {
							return fmt.Errorf("getting registry authentication after reset: %w", err)
						}
						authOption = remote.WithAuth(auth)
					}
					continue
				}
			}
			return fmt.Errorf("pushing image to remote registry: %w", err)
		}

		break
	}

	slog.InfoContext(ctx, "image pushed successfully",
		"source", srcRef,
		"destination", destRef,
		"duration", fmt.Sprintf("%f seconds", time.Since(startTime).Seconds()))

	return nil
}
