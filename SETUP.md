# Fuse Engine - Docker Compose Setup

This setup includes MongoDB (Bitnami) and LocalStack with S3 functionality.

## Services

### MongoDB (Bitnami)

- **Image**: `bitnami/mongodb:latest`
- **Port**: `27017`
- **Volume**: `mongodb_data:/bitnami/mongodb`

### LocalStack

- **Image**: `localstack/localstack:latest`
- **Port**: `4566` (main endpoint)
- **Services**: S3, SQS, SNS
- **Volume**: `localstack_data:/tmp/localstack`

### S3 Setup Container

- Automatically runs after LocalStack is ready
- Creates three S3 buckets:
  - `fuse-app-bucket` (private)
  - `fuse-uploads` (public read)
  - `fuse-backups` (private)

## Environment Variables

Create a `.env` file with the following variables:

```env
# Env configuration
APP_NAME="Fuse"

SERVER_PORT=9090
SERVER_HOST="localhost"

DB_DRIVER="mongodb"
DB_HOST="localhost"
DB_PORT=27017
DB_NAME="fuse"
DB_USER="fuse"
DB_PASS="password123"
DB_TLS=false
```

Or just copy the example from .env.example

## Usage

1. **Start all services**:

   ```bash
   docker compose up -d
   ```

2. **View logs**:

   ```bash
   docker compose logs -f
   ```

3. **Stop services**:

   ```bash
   docker compose down
   ```

4. **Reset data** (removes all volumes):
   ```bash
   docker compose down -v
   ```

## Accessing Services

- **MongoDB**: `mongodb://admin:password@localhost:27017/fuse`
- **LocalStack S3**: `http://localhost:4566`
- **S3 Buckets** (via AWS CLI):
  ```bash
  aws --endpoint-url=http://localhost:4566 s3 ls
  ```

## S3 Configuration

The S3 setup script (`scripts/setup-s3.sh`) automatically:

- Waits for LocalStack to be ready
- Creates required S3 buckets
- Sets bucket policies
- Provides confirmation of setup

## Network

All services run on the `fuse-network` bridge network for internal communication.
