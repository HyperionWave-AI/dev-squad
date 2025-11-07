# Docker Registry Setup

## Overview

Hyperion uses Google Cloud Artifact Registry to host prebuilt Docker images. This allows users to run Hyperion without building images locally.

## Registry Configuration

- **Project ID**: `production-471918`
- **Region**: `us-central1`
- **Repository**: `hyperion-public`
- **Registry URL**: `us-central1-docker.pkg.dev/production-471918/hyperion-public`

## Available Images

1. **hyperion-coordinator-mcp** - MCP Server with HTTP transport
2. **hyperion-http-bridge** - HTTP bridge connecting MCP server to UI
3. **hyperion-ui** - React dashboard UI

## For Users (Running Hyperion)

### Using Prebuilt Images

The default `docker-compose.yml` is configured to pull prebuilt images from the registry:

```bash
# Pull and start all services with prebuilt images
docker compose up -d

# Or pull images first
docker compose pull
docker compose up -d
```

**No authentication required for pulling images!**

### Switching to Local Development

Use `docker-compose.dev.yml` for local development with hot-reload:

```bash
# Development mode with local builds and hot-reload
docker compose -f docker-compose.dev.yml up -d
```

## For Contributors (Building & Publishing Images)

### Prerequisites

1. Google Cloud SDK installed
2. Authenticated with GCP:
   ```bash
   gcloud auth login
   gcloud config set project production-471918
   ```

### Building and Pushing Images

```bash
# Build all images and push to registry
./scripts/build-and-push-images.sh

# Push existing local images without rebuilding
./scripts/build-and-push-images.sh --skip-build
```

### Manual Build and Push

```bash
# Authenticate Docker with GCP
gcloud auth configure-docker us-central1-docker.pkg.dev

# Build image
docker build -f coordinator/mcp-server/Dockerfile \
  -t hyperion-coordinator-mcp:latest \
  coordinator/mcp-server

# Tag for registry
docker tag hyperion-coordinator-mcp:latest \
  us-central1-docker.pkg.dev/production-471918/hyperion-public/hyperion-coordinator-mcp:latest

# Push to registry
docker push us-central1-docker.pkg.dev/production-471918/hyperion-public/hyperion-coordinator-mcp:latest
```

## Docker Compose Configurations

### Production (`docker-compose.yml`)
- Uses prebuilt images from registry
- No local builds required
- Fast startup
- Suitable for end users

### Development (`docker-compose.dev.yml`)
- Builds images locally
- Mounts source code for hot-reload
- Suitable for development and testing

## Architecture

```
docker-compose.yml (Production)
├── hyperion-mcp-server
│   └── image: us-central1-docker.pkg.dev/.../hyperion-coordinator-mcp:latest
├── hyperion-http-bridge
│   └── image: us-central1-docker.pkg.dev/.../hyperion-http-bridge:latest
├── hyperion-ui
│   └── image: us-central1-docker.pkg.dev/.../hyperion-ui:latest
├── mongodb (mongo:7.0)
├── qdrant (qdrant/qdrant:latest)
└── embedding-service (ghcr.io/huggingface/text-embeddings-inference:cpu-latest)

docker-compose.dev.yml (Development)
├── hyperion-mcp-server
│   └── build: coordinator/mcp-server
│   └── volumes: [source code mounted for hot-reload]
├── hyperion-http-bridge
│   └── build: coordinator/mcp-http-bridge
├── hyperion-ui
│   └── build: coordinator/ui
└── [same infrastructure services]
```

## Registry Access Control

Currently, the registry has organization-level restrictions preventing public (`allUsers`) access due to GCP policies:

```
constraints/iam.allowedPolicyMemberDomains
```

### Workarounds:

1. **Authenticated Users**: Contributors authenticate with `gcloud auth login`
2. **Service Account**: For CI/CD, use a service account with `artifactregistry.reader` role
3. **Domain Restriction**: May need to add specific Google Workspace domain

### Future: Public Registry

To make the registry fully public (if org policy allows):

```bash
gcloud artifacts repositories add-iam-policy-binding hyperion-public \
  --location=us-central1 \
  --member=allUsers \
  --role=roles/artifactregistry.reader
```

## Troubleshooting

### Pull Access Denied

**Problem**: `denied: Permission "artifactregistry.repositories.downloadArtifacts" denied`

**Solution**:
1. Authenticate: `gcloud auth configure-docker us-central1-docker.pkg.dev`
2. Or use service account key: `gcloud auth activate-service-account --key-file=key.json`

### Image Not Found

**Problem**: `manifest for ... not found`

**Solution**:
1. Verify image exists: `gcloud artifacts docker images list --repository=hyperion-public --location=us-central1`
2. Build and push: `./scripts/build-and-push-images.sh`

### Organization Policy Error

**Problem**: `User allUsers is not in permitted organization`

**Solution**: This is expected. Use authenticated access for now. Public access requires org policy changes.

## CI/CD Integration

For GitHub Actions or other CI/CD:

```yaml
- name: Authenticate to Google Cloud
  uses: google-github-actions/auth@v1
  with:
    credentials_json: ${{ secrets.GCP_SA_KEY }}

- name: Configure Docker
  run: gcloud auth configure-docker us-central1-docker.pkg.dev

- name: Build and Push
  run: ./scripts/build-and-push-images.sh
```

## Version Management

Current strategy: **Latest tag only**
- All images use `:latest` tag
- Simple for users (always get newest version)
- No version tracking complexity

Future strategy (if needed):
- Semantic versioning: `:1.0.0`, `:1.0`, `:1`
- Git commit SHA: `:abc123f`
- Date-based: `:2025-01-10`

## Cost

Artifact Registry pricing:
- **Storage**: $0.10/GB per month
- **Egress**: Free within same region, standard network rates otherwise
- **Estimated cost**: < $5/month for Hyperion images

## Related Files

- `docker-compose.yml` - Production configuration with prebuilt images
- `docker-compose.dev.yml` - Development configuration with local builds
- `scripts/build-and-push-images.sh` - Build and publish script
- `scripts/create-public-registry.sh` - Registry creation script
