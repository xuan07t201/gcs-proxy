#!/bin/bash

# GCS Proxy Deployment Script for Google Cloud Run

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ID="${GOOGLE_CLOUD_PROJECT}"
BUCKET_NAME="${GCS_BUCKET_NAME}"
REGION="${CLOUD_RUN_REGION:-us-central1}"
SERVICE_NAME="${CLOUD_RUN_SERVICE:-gcs-proxy}"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    if [ -z "$PROJECT_ID" ]; then
        print_error "GOOGLE_CLOUD_PROJECT environment variable is not set"
        exit 1
    fi
    
    if [ -z "$BUCKET_NAME" ]; then
        print_error "GCS_BUCKET_NAME environment variable is not set"
        exit 1
    fi
    
    # Check if gcloud is installed and authenticated
    if ! command -v gcloud &> /dev/null; then
        print_error "gcloud CLI is not installed"
        exit 1
    fi
    
    # Check if authenticated
    if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q "."; then
        print_error "Not authenticated with gcloud. Run: gcloud auth login"
        exit 1
    fi
    
    print_status "Prerequisites check passed"
}

# Enable required APIs
enable_apis() {
    print_status "Enabling required Google Cloud APIs..."
    
    gcloud services enable cloudbuild.googleapis.com \
        run.googleapis.com \
        containerregistry.googleapis.com \
        --project="$PROJECT_ID"
    
    print_status "APIs enabled successfully"
}

# Build and deploy using Cloud Build
deploy_with_cloudbuild() {
    print_status "Building and deploying with Cloud Build..."
    
    gcloud builds submit --config cloudbuild.yaml \
        --substitutions _BUCKET_NAME="$BUCKET_NAME" \
        --project="$PROJECT_ID"
    
    print_status "Deployment completed successfully"
}

# Manual deployment (alternative to Cloud Build)
deploy_manual() {
    print_status "Building Docker image..."
    
    # Build image
    docker build -t gcr.io/"$PROJECT_ID"/gcs-proxy:latest .
    
    print_status "Pushing image to Container Registry..."
    docker push gcr.io/"$PROJECT_ID"/gcs-proxy:latest
    
    print_status "Deploying to Cloud Run..."
    gcloud run deploy "$SERVICE_NAME" \
        --image gcr.io/"$PROJECT_ID"/gcs-proxy:latest \
        --region "$REGION" \
        --platform managed \
        --allow-unauthenticated \
        --set-env-vars GOOGLE_CLOUD_PROJECT_ID="$PROJECT_ID",GCS_BUCKET_NAME="$BUCKET_NAME",NODE_ENV=production \
        --memory 512Mi \
        --cpu 1 \
        --concurrency 80 \
        --max-instances 100 \
        --min-instances 0 \
        --timeout 300s \
        --project="$PROJECT_ID"
    
    print_status "Manual deployment completed successfully"
}

# Get service URL
get_service_url() {
    SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" \
        --region "$REGION" \
        --project "$PROJECT_ID" \
        --format "value(status.url)")
    
    print_status "Service deployed successfully!"
    print_status "Service URL: $SERVICE_URL"
    print_status "Health check: $SERVICE_URL/health"
}

# Main deployment function
main() {
    print_status "Starting GCS Proxy deployment..."
    print_status "Project: $PROJECT_ID"
    print_status "Bucket: $BUCKET_NAME"
    print_status "Region: $REGION"
    print_status "Service: $SERVICE_NAME"
    
    check_prerequisites
    enable_apis
    
    # Choose deployment method
    if [ "$1" = "manual" ]; then
        deploy_manual
    else
        deploy_with_cloudbuild
    fi
    
    get_service_url
    
    print_status "Deployment completed! 🎉"
}

# Help function
show_help() {
    echo "Usage: $0 [manual]"
    echo ""
    echo "Environment variables required:"
    echo "  GOOGLE_CLOUD_PROJECT - Your GCP project ID"
    echo "  GCS_BUCKET_NAME - Your GCS bucket name"
    echo ""
    echo "Optional environment variables:"
    echo "  CLOUD_RUN_REGION - Cloud Run region (default: us-central1)"
    echo "  CLOUD_RUN_SERVICE - Service name (default: gcs-proxy)"
    echo ""
    echo "Options:"
    echo "  manual    Use manual deployment instead of Cloud Build"
    echo "  help      Show this help message"
}

# Parse arguments
case "$1" in
    help|--help|-h)
        show_help
        ;;
    *)
        main "$@"
        ;;
esac