#!/bin/bash

# Create public Docker registry in Google Cloud
# This creates an Artifact Registry repository with public read access

set -e

# Configuration
PROJECT_ID="production-471918"
REGION="us-central1"
REPOSITORY_NAME="hyperion-public"
DESCRIPTION="Public Docker registry for Hyperion MCP server and related images"

echo "Creating public Docker registry in Google Cloud..."
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo "Repository: $REPOSITORY_NAME"
echo ""

# Create the Artifact Registry repository
echo "Step 1: Creating Artifact Registry repository..."
gcloud artifacts repositories create $REPOSITORY_NAME \
    --repository-format=docker \
    --location=$REGION \
    --description="$DESCRIPTION" \
    --project=$PROJECT_ID

echo "✓ Repository created successfully"
echo ""

# Make the repository publicly readable
echo "Step 2: Setting public read access..."
gcloud artifacts repositories add-iam-policy-binding $REPOSITORY_NAME \
    --location=$REGION \
    --member=allUsers \
    --role=roles/artifactregistry.reader \
    --project=$PROJECT_ID

echo "✓ Public read access granted"
echo ""

# Display repository information
echo "=========================================="
echo "Public Docker Registry Created!"
echo "=========================================="
echo ""
echo "Registry URL:"
echo "  ${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}"
echo ""
echo "To push images:"
echo "  1. Authenticate: gcloud auth configure-docker ${REGION}-docker.pkg.dev"
echo "  2. Tag image: docker tag <local-image> ${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}/<image-name>:<tag>"
echo "  3. Push image: docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}/<image-name>:<tag>"
echo ""
echo "To pull images (no authentication required):"
echo "  docker pull ${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}/<image-name>:<tag>"
echo ""
echo "Example for MCP server:"
echo "  docker tag hyperion-coordinator-mcp:latest ${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}/hyperion-coordinator-mcp:latest"
echo "  docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}/hyperion-coordinator-mcp:latest"
echo ""
echo "Public users can pull:"
echo "  docker pull ${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}/hyperion-coordinator-mcp:latest"
echo ""
