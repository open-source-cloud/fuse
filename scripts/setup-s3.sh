#!/bin/bash

# Wait for LocalStack to be ready
echo "Waiting for LocalStack to be ready..."
until curl -s http://localstack:4566/_localstack/health | grep -q '"s3": "available"'; do
  echo "LocalStack S3 not ready yet, waiting..."
  sleep 2
done

echo "LocalStack S3 is ready! Setting up buckets..."

# Configure AWS CLI to use LocalStack
export AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID:-test}
export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:-test}
export AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-us-east-1}

export WORKFLOW_BUCKET_NAME=${WORKFLOW_BUCKET_NAME:-fuse-workflows}

aws --endpoint-url=http://localstack:4566 s3api create-bucket --bucket $WORKFLOW_BUCKET_NAME

echo "🎉 S3 setup completed successfully!"
echo "LocalStack S3 endpoint: http://localhost:4566"
echo "Created buckets:"
echo "  - $WORKFLOW_BUCKET_NAME"