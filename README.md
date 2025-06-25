# k8s-controller

[![Overall CI/CD](https://github.com/oleksandr-san/k8s-controller/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/oleksandr-san/k8s-controller/actions/workflows/ci.yaml)
[![Go Version](https://img.shields.io/badge/Go-1.24-blue)](https://golang.org/)

A lightweight Kubernetes controller application built with Go that provides HTTP API endpoints for interacting with Kubernetes resources and includes a dynamic informer system for watching cluster events.

## Features

- **HTTP Server**: FastHTTP-based web server with request logging and middleware
- **Kubernetes Integration**: Dynamic informer system for watching Kubernetes resources
- **CLI Interface**: Cobra-based command-line interface with multiple subcommands
- **Kubernetes API Operations**: Built-in commands for applying, listing, and deleting Kubernetes resources
- **Configurable Logging**: Structured logging with multiple levels (trace, debug, info, warn, error)
- **Docker Support**: Multi-stage Docker build with distroless base image
- **Helm Chart**: Ready-to-deploy Helm chart for Kubernetes
- **CI/CD Pipeline**: GitHub Actions workflow with automated testing and Docker image publishing

## Architecture

```
k8s-controller/
├── cmd/                    # CLI commands and application entry points
│   ├── root.go            # Root command with logging configuration
│   ├── server.go          # HTTP server command
│   ├── k8sapi.go          # Kubernetes API base command
│   ├── k8sapi_apply.go    # Apply Kubernetes resources
│   ├── k8sapi_list.go     # List Kubernetes resources
│   └── k8sapi_delete.go   # Delete Kubernetes resources
├── pkg/                   # Core packages
│   ├── informer/          # Dynamic Kubernetes informer implementation
│   └── testutil/          # Testing utilities with envtest support
├── charts/                # Helm chart for deployment
└── .github/workflows/     # CI/CD pipeline configuration
```

## Prerequisites

- Go 1.24.0 or later
- Kubernetes cluster access (local or remote)
- kubectl configured with cluster access
- Docker (for containerization)
- Helm 3.x (for deployment)

## Installation

### From Source

```bash
git clone https://github.com/oleksandr-san/k8s-controller.git
cd k8s-controller
make build
```

### Using Docker

```bash
docker pull ghcr.io/oleksandr-san/k8s-controller:latest
```

### Using Helm

```bash
helm install k8s-controller ./charts/k8s-controller
```

## Usage

### Basic Commands

Display help and version information:
```bash
./k8s-controller --help
./k8s-controller --log-level debug
```

### HTTP Server

Start the HTTP server with Kubernetes informer:
```bash
./k8s-controller server --port 8080 --kubeconfig ~/.kube/config
```

Server options:
- `--port`: Server port (default: 8080)
- `--kubeconfig`: Path to kubeconfig file (default: ~/.kube/config)
- `--in-cluster`: Use in-cluster Kubernetes configuration
- `--log-level`: Set logging level (trace, debug, info, warn, error)
- `--enable-leader-election`: Enable leader election for controller manager (default: true)
- `--leader-election-namespace`: Namespace for leader election (default: default)
- `--metrics-port`: Port for controller manager metrics (default: 8081)

### Kubernetes API Operations

List Kubernetes resources:
```bash
./k8s-controller k8sapi list --resource deployments --namespace default
```

Apply Kubernetes manifests:
```bash
./k8s-controller k8sapi apply --filename deployment.yaml
```

Delete Kubernetes resources:
```bash
./k8s-controller k8sapi delete --resource deployment --name my-app --namespace default
```

## Configuration

### Environment Variables

The application supports configuration via environment variables:

- `LOG_LEVEL`: Set logging level (trace, debug, info, warn, error)
- `APP_PORT`: Server port
- `KUBECONFIG`: Path to kubeconfig file
- `ENABLE_LEADER_ELECTION`: Enable leader election for controller manager
- `LEADER_ELECTION_NAMESPACE`: Namespace for leader election
- `APP_METRICS_PORT`: Port for controller manager metrics

### Logging

The application uses structured logging with zerolog. Log levels can be configured:

- `trace`: Most verbose, includes caller information and console output
- `debug`: Debug information with caller details
- `info`: General information (default)
- `warn`: Warning messages
- `error`: Error messages only

## Development

### Building

```bash
# Build the application
make build

# Build with specific OS/architecture
GOOS=linux GOARCH=amd64 make build

# Build Docker image
make docker-build
```

### Testing

```bash
# Run tests with envtest
make test

# Run tests with coverage
make test-coverage

# Set up envtest environment
make envtest
```

### Code Quality

```bash
# Format code
make format

# Run linter
make lint
```

## Deployment

### Docker Deployment

```bash
docker run -p 8080:8080 \
  -v ~/.kube/config:/root/.kube/config:ro \
  ghcr.io/oleksandr-san/k8s-controller:latest \
  server --kubeconfig /root/.kube/config
```

### Kubernetes Deployment

Using the provided Helm chart:

```bash
# Install with default values
helm install k8s-controller ./charts/k8s-controller

# Install with custom values
helm install k8s-controller ./charts/k8s-controller \
  --set image.tag=latest \
  --set service.port=8080 \
  --set replicaCount=2
```

### Helm Chart Configuration

Key configuration options in `values.yaml`:

- `replicaCount`: Number of replicas
- `image.repository`: Container image repository
- `image.tag`: Container image tag
- `service.port`: Service port
- `ingress.enabled`: Enable ingress
- `resources`: Resource limits and requests
- `autoscaling.enabled`: Enable horizontal pod autoscaling
- `leaderElection.enabled`: Enable leader election for controller manager
- `leaderElection.namespace`: Namespace for leader election

## API Endpoints

When running in server mode, the application exposes:

- `GET /`: Welcome endpoint returning server status
- Request logging with unique request IDs
- Health check capabilities

## Monitoring and Observability

- Structured JSON logging with request tracing
- Request ID tracking across requests
- HTTP request metrics (method, path, status, latency)
- Kubernetes resource event logging

## CI/CD Pipeline

The project includes a GitHub Actions workflow that:

- Builds and tests the application on Go 1.24.4
- Runs security scans with Trivy
- Builds and pushes Docker images to GitHub Container Registry
- Packages and uploads Helm charts as artifacts
- Supports both branch and tag-based deployments

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the terms specified in the [LICENSE](LICENSE) file.

## Support

For questions, issues, or contributions, please:

1. Check existing [GitHub Issues](https://github.com/oleksandr-san/k8s-controller/issues)
2. Create a new issue with detailed information
3. Follow the project's contribution guidelines

## Roadmap

- [ ] Add more Kubernetes resource types support
- [ ] Implement custom resource definitions (CRDs)
- [ ] Add metrics and monitoring endpoints
- [ ] Enhance error handling and recovery
- [ ] Add configuration validation
- [ ] Implement admission controllers