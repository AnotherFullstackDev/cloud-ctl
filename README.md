# Cloud CTL
Cloud CTL is a command-line tool designed to simplify interaction with and deployment of compute cloud resources across multiple cloud service providers. It provides a unified interface for users to interact with various cloud platforms.

## Status
Currently, it is pretty mich work in progress. New features are being added regularly. But also breaking changes might occur. At this stage no guarantees are made regarding stability or backward compatibility.

## Features
- **Multi-Cloud Support**: Deploy and observe containers across different cloud providers (Render, Railway is coming, AWS is coming, GCP is coming, Azure is coming).
- **Container Management**: Automatically builds and deploys Docker containers to the cloud (at this point only CLI command for docker build is supported).
- **Container Registries**: Supports pushing and deployment of images from popular container registries (GitHub Container Registry, Docker Hub is coming, AWS ECR is coming, GCP Container Registry is coming, Azure Container Registry is coming).
- **Credentials storage**: Securely store and manage cloud provider credentials locally.

## Commands
- `cloudctl service deploy [service_name]`: Build a docker container and deploy to the specified cloud provider.

## Config
Cloud CTL uses a configuration file `cloudctl.yaml` located at the root of the project.

Config file structure:
```yaml
services:
  SERVICE_NAME:
    # Configuration of the image to build and push
    image:
      repository: "LOCAL IMAGE"
      tag: "LOCAL TAG"
      build:
        cmd: "COMMAND TO BUILD DOCKER IMAGE"
      # Github Container Registry configuration
      ghcr:
        username: "Github USERNAME"
        owner: "Github USERNAME or ORGANIZATION NAME"
        repository: "REPOSITORY"
        tag: "REMOTE TAG TO ASSIGN TO THE PUSHED IMAGE"
    render:
      service_id: "RENDER SERVICE ID"
```
