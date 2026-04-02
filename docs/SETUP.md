# Fuse Engine - Docker Compose Setup

This setup includes LocalStack with AWS resources (S3, SQS, SNS).

## Services

### LocalStack

- **Image**: `localstack/localstack:latest`
- **Port**: `4566` (main endpoint)
- **Services**: S3, SQS, SNS
- **Volume**: `localstack_data:/var/lib/localstack`

### Localstack Setup Container

- Automatically runs after LocalStack is ready

## Environment Variables

Create a `.env` file from the .env.example

```bash
cp .env.example .env
```

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
