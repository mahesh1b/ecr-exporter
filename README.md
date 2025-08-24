# ECR Prometheus Exporter

A Prometheus exporter for AWS Elastic Container Registry (ECR) that provides comprehensive metrics about your container repositories and images.

## Metrics Exported

- `ecr_repositories_total` - Total number of ECR repositories
- `ecr_images_total` - Number of images in each ECR repository
- `ecr_image_size_max_bytes` - Maximum image size in each repository
- `ecr_image_size_min_bytes` - Minimum image size in each repository  
- `ecr_image_size_avg_bytes` - Average image size in each repository
- `ecr_latest_push_timestamp` - Latest image push date and time (Unix timestamp)
- `ecr_latest_pull_timestamp` - Latest image pull date and time (Unix timestamp)
- `ecr_scrape_errors_total` - Total number of scrape errors
- `ecr_scrape_duration_seconds` - Duration of the scrape operation

## Prerequisites

- AWS credentials configured (via AWS CLI, IAM role, or environment variables)
- Required IAM permissions:
  - `ecr:DescribeRepositories`
  - `ecr:DescribeImages`

## Usage

### Running Locally

```bash
# Install dependencies
go mod tidy

# Run the exporter
go run .
```

The exporter will start on port 8080. Metrics are available at `http://localhost:8080/metrics`.

### Running with Docker

```bash
# Build the image
docker build -t ecr-exporter .

# Run with AWS credentials
docker run -p 8080:8080 \
  -e AWS_ACCESS_KEY_ID=your_access_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret_key \
  -e AWS_REGION=us-east-1 \
  -e LOG_LEVEL=info \
  ecr-exporter

# Run with debug logging
docker run -p 8080:8080 \
  -e AWS_ACCESS_KEY_ID=your_access_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret_key \
  -e AWS_REGION=us-east-1 \
  -e LOG_LEVEL=debug \
  ecr-exporter
```

### Environment Variables

**AWS Configuration:**
- `AWS_REGION` - AWS region (default: us-east-1)
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `AWS_SESSION_TOKEN` - AWS session token (for temporary credentials)

**Application Configuration:**
- `LOG_LEVEL` - Log level (default: info)
  - Valid values: `debug`, `info`, `warn`, `error`, `fatal`, `panic`
  - Example: `LOG_LEVEL=debug`

## Logging Configuration

The exporter uses structured logging in logfmt format (key=value pairs) for easy parsing by log aggregation systems.

### Log Levels

Set the `LOG_LEVEL` environment variable to control verbosity:

- `debug` - Detailed debugging information including API calls
- `info` - General operational messages (default)
- `warn` - Warning messages for non-critical issues
- `error` - Error messages for failures
- `fatal` - Fatal errors that cause the application to exit
- `panic` - Panic-level errors

### Examples

```bash
# Run with debug logging
LOG_LEVEL=debug go run .

# Run with minimal logging
LOG_LEVEL=error go run .

# Default (info level)
go run .
```

### Log Format

All logs are output in logfmt format:
```
time=2025-08-24T10:30:45Z level=info msg="Starting ECR Prometheus Exporter"
time=2025-08-24T10:30:45Z level=info msg="AWS connectivity test successful"
time=2025-08-24T10:30:50Z level=debug msg="Processing repository 1/20: my-repo"
```

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'ecr-exporter'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 300s  # ECR API has rate limits, so scrape less frequently
```

## Sample Queries

```promql
# Number of repositories
ecr_repositories_total

# Repositories with most images
topk(10, ecr_images_total)

# Average image size across all repositories
avg(ecr_image_size_avg_bytes) by (repository_name)

# Time since last push (in hours)
(time() - ecr_latest_push_timestamp) / 3600

# Repositories not pulled recently (> 30 days)
ecr_latest_pull_timestamp < (time() - 30*24*3600)
```

## IAM Policy

Minimum required IAM policy:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ecr:DescribeRepositories",
                "ecr:DescribeImages"
            ],
            "Resource": "*"
        }
    ]
}
```