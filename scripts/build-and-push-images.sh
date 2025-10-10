#!/bin/bash

# Build and push Docker images to Google Cloud Artifact Registry
# Usage: ./build-and-push-images.sh [--skip-build]

set -e

# Configuration
PROJECT_ID="production-471918"
REGION="us-central1"
REPOSITORY="hyperion-public"
REGISTRY_URL="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY}"

# Parse arguments
SKIP_BUILD=false
if [[ "$1" == "--skip-build" ]]; then
    SKIP_BUILD=true
    echo "⚠️  Skipping build, will only push existing images"
fi

echo "=========================================="
echo "Building and Pushing Hyperion Images"
echo "=========================================="
echo "Registry: ${REGISTRY_URL}"
echo ""

# Authenticate with Google Cloud
echo "Step 1: Authenticating with Google Cloud..."
gcloud auth configure-docker ${REGION}-docker.pkg.dev --quiet
echo "✓ Authenticated"
echo ""

# Navigate to project root
cd "$(dirname "$0")/.."
PROJECT_ROOT=$(pwd)
echo "Project root: ${PROJECT_ROOT}"
echo ""

# Image definitions: local-name | context-path | dockerfile-path | registry-name
IMAGES=(
    "hyperion-coordinator-mcp|coordinator/mcp-server|Dockerfile|hyperion-coordinator-mcp"
    "hyperion-http-bridge|coordinator|mcp-http-bridge/Dockerfile.combined|hyperion-http-bridge"
    "hyperion-ui|coordinator/ui|Dockerfile|hyperion-ui"
)

# Build and push each image
for image_config in "${IMAGES[@]}"; do
    IFS='|' read -r LOCAL_NAME CONTEXT DOCKERFILE REGISTRY_NAME <<< "$image_config"

    FULL_REGISTRY_TAG="${REGISTRY_URL}/${REGISTRY_NAME}:latest"

    echo "=========================================="
    echo "Image: ${LOCAL_NAME}"
    echo "=========================================="

    if [ "$SKIP_BUILD" = false ]; then
        echo "Building image..."
        echo "  Context: ${CONTEXT}"
        echo "  Dockerfile: ${DOCKERFILE}"

        docker build \
            -f "${CONTEXT}/${DOCKERFILE}" \
            -t "${LOCAL_NAME}:latest" \
            "${CONTEXT}"

        echo "✓ Built: ${LOCAL_NAME}:latest"
    else
        echo "⚠️  Skipping build (using existing local image)"
    fi

    echo ""
    echo "Tagging for registry..."
    docker tag "${LOCAL_NAME}:latest" "${FULL_REGISTRY_TAG}"
    echo "✓ Tagged: ${FULL_REGISTRY_TAG}"

    echo ""
    echo "Pushing to registry..."
    docker push "${FULL_REGISTRY_TAG}"
    echo "✓ Pushed: ${FULL_REGISTRY_TAG}"
    echo ""
done

echo "=========================================="
echo "✅ All Images Built and Pushed Successfully!"
echo "=========================================="
echo ""
echo "Images available at:"
for image_config in "${IMAGES[@]}"; do
    IFS='|' read -r LOCAL_NAME CONTEXT DOCKERFILE REGISTRY_NAME <<< "$image_config"
    echo "  ${REGISTRY_URL}/${REGISTRY_NAME}:latest"
done
echo ""
echo "To use these images, update docker-compose.yml:"
echo "  hyperion-mcp-server:"
echo "    image: ${REGISTRY_URL}/hyperion-coordinator-mcp:latest"
echo ""
echo "  hyperion-http-bridge:"
echo "    image: ${REGISTRY_URL}/hyperion-http-bridge:latest"
echo ""
echo "  hyperion-ui:"
echo "    image: ${REGISTRY_URL}/hyperion-ui:latest"
echo ""
