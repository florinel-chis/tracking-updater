# Magento Tracking Updater Microservice

A Go microservice that processes CSV files containing order tracking information and updates shipments in Magento 2 via the REST API.

## Features

- Monitors a directory for new CSV files with tracking information
- Processes files containing order numbers, tracking numbers, carrier codes, and titles
- Automatically retrieves order and shipment information from Magento 2
- Updates tracking information for shipments via the Magento 2 REST API
- Handles errors gracefully and provides detailed logging
- Moves processed files to success/failure directories

## Requirements

- Go 1.21 or higher
- Magento 2 installation with REST API access
- A valid API token for Magento 2

## Installation

### From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/tracking-updater.git
   cd tracking-updater
   ```

2. Build the application:
   ```bash
   go build -o tracking-updater ./cmd/server
   ```

### Using Docker

1. Build the Docker image:
   ```bash
   docker build -t tracking-updater .
   ```

## Configuration

Create a `config.yaml` file with the following structure:

```yaml
magento:
  base_url: "https://yourdomain.com/rest/V1"
  token: "your_magento_api_token"
  timeout: 30s
  max_retries: 3
  retry_backoff: 1s

file_watch:
  directory: "/path/to/watch"
  file_pattern: "^\\d{8}_\\d{6}\\.csv$"
  processed_dir: "/path/to/processed"
  failed_dir: "/path/to/failed"
  poll_interval: 5s
  max_concurrency: 5
  batch_size: 50
  file_process_time: 10m

log:
  level: "info"
  format: "json"
  file: "/path/to/logs/tracking-updater.log"
  enable_file: true
```

### Configuration Parameters

#### Magento Configuration

- `base_url`: The base URL for the Magento REST API
- `token`: Your Magento API access token
- `timeout`: HTTP request timeout
- `max_retries`: Maximum number of retry attempts for failed requests
- `retry_backoff`: Time to wait between retry attempts

#### File Watching Configuration

- `directory`: The directory to watch for new CSV files
- `file_pattern`: Regular expression pattern for matching valid file names
- `processed_dir`: Directory to move successfully processed files to
- `failed_dir`: Directory to move files that failed processing to
- `poll_interval`: Interval between directory scans
- `max_concurrency`: Maximum number of concurrent file processing workers
- `batch_size`: Number of records to process in a batch
- `file_process_time`: Maximum time to spend processing a file

#### Logging Configuration

- `level`: Log level (debug, info, warn, error)
- `format`: Log format (text, json)
- `file`: Path to the log file
- `enable_file`: Whether to write logs to a file

## Usage

### Running from Source

```bash
./tracking-updater --config config.yaml
```

### Running with Docker

```bash
docker run -d \
  --name tracking-updater \
  -v /path/to/config.yaml:/app/config.yaml \
  -v /path/to/watch:/data/watch \
  -v /path/to/processed:/data/processed \
  -v /path/to/failed:/data/failed \
  -v /path/to/logs:/data/logs \
  tracking-updater
```

## CSV File Format

The CSV files should have the following columns:

- `order_number`: The Magento increment ID/order number
- `tracking_number`: The tracking number for the shipment
- `carrier_code`: The carrier code (as defined in Magento)
- `title`: The title/name of the shipping carrier

Example:

```csv
order_number,tracking_number,carrier_code,title
1000000001,1ZX23456789,ups,UPS
1000000002,123456789012,fedex,FedEx
```

## Best Practices

1. Always ensure your Magento API token has the appropriate permissions
2. Monitor the logs for any errors or issues
3. Ensure sufficient disk space for log files and processed files
4. Consider setting up log rotation for the log files
5. Regularly clean up old processed files

## Troubleshooting

- If files are not being processed, check the file pattern configuration
- If tracking updates are failing, verify your Magento API credentials
- Check the logs for detailed error information
- Ensure all required directories exist and are writable

## License

MIT