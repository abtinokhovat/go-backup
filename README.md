# GoBackupDB

A robust backup agent written in Go that runs as a Linux systemd service, providing automated backup capabilities with flexible scheduling.

## Prerequisites

- Linux operating system with systemd
- Go 1.16 or higher
- Root/sudo access for service installation
- Make utility

## Installation

1. Clone the repository:

```bash
git clone <repository-url>
cd backup-agent
```

2. Build and install the service:

```bash
make install
```

This will:

- Build the Go binary
- Install the binary to `/usr/local/bin`
- Install the systemd service and timer
- Set up the configuration directory at `/etc/go-backup`
- Copy the configuration file

3. Enable the service to start automatically:

```bash
make enable
```

## Makefile Commands

The backup agent provides several make commands for managing the service and its operations:

### Installation Commands

- `make build` - Build the Go binary (outputs to `build/` directory)
- `make install` - Install the service, binary, and configuration files
- `make uninstall` - Remove all installed files and services
- `make install-delete-timer` - Install the deletion timer service
- `make uninstall-delete-timer` - Remove the deletion timer service

### Service Management Commands

- `make start` - Start the backup service
- `make stop` - Stop the backup service
- `make restart` - Restart the backup service
- `make status` - Check the current status of the service
- `make enable` - Enable and start the service (starts automatically on boot)
- `make disable` - Disable and stop the service (won't start on boot)

### Scheduling Commands

- `make schedule` - View current backup schedule and available scheduling options
- `make delete-schedule` - Delete the current backup schedule

### Deletion Timer Commands

- `make start-delete` - Start the deletion timer
- `make stop-delete` - Stop the deletion timer
- `make restart-delete` - Restart the deletion timer
- `make status-delete` - Check the status of the deletion timer
- `make enable-delete` - Enable and start the deletion timer
- `make disable-delete` - Disable and stop the deletion timer

### Development Commands

- `make test` - Run all tests
- `make lint` - Run linters
- `make fmt` - Format Go code
- `make vet` - Run Go vet
- `make deps` - Install dependencies

## Configuration

The backup agent is configured using a YAML file located at `/etc/go-backup/config.yaml`. The configuration file is copied during installation, but you can modify it at any time.

### Encryption

The backup agent supports AES-256-GCM encryption for your backups. To enable encryption:

1. Generate a secure encryption key:

```bash
# Generate a random 32-byte key and encode it in base64
openssl rand -base64 32
```

2. Add the generated key to your configuration:

```yaml
encryption:
  enabled: true
  key: "your-generated-base64-key-here" # The key generated in step 1
```

Important security notes:

- Keep your encryption key secure and never share it
- Store a backup of your encryption key in a secure location
- If you lose the encryption key, you won't be able to decrypt your backups
- The key must be exactly 32 bytes when decoded from base64
- The key is used for AES-256-GCM encryption, which provides both confidentiality and authenticity

Example configuration structure:

```yaml
# Add your configuration details here
# Example:
backup:
  source: /path/to/source
  destination: /path/to/destination
  retention: 7d
```

## Service Management

The backup agent provides several make commands for easy service management:

### Basic Commands

- `make start` - Start the backup service
- `make stop` - Stop the backup service
- `make status` - Check the current status of the service
- `make restart` - Restart the backup service

### Service Lifecycle

- `make enable` - Enable and start the service (starts automatically on boot)
- `make disable` - Disable and stop the service (won't start on boot)

### Scheduling

- `make schedule` - View current schedule and available scheduling options
- `make delete-schedule` - Delete the current backup schedule

The service runs as a systemd timer, which provides flexible scheduling options. By default, it runs daily at 12:00.

### Deletion Timer Setup

1. Install the deletion timer:

```bash
make install-delete-timer
```

2. Enable the deletion timer:

```bash
make enable-delete
```

3. View deletion timer status:

```bash
make status-delete
```

4. To stop or disable the deletion timer:

```bash
make stop-delete    # Stop the timer
make disable-delete # Disable and stop the timer
```

## Uninstallation

To completely remove the backup agent:

```bash
make uninstall
```

This will:

- Stop and disable the service
- Remove the binary and script files
- Remove the systemd service and timer files
- Remove the configuration directory

## Service Details

### Service File

The service is defined in `go-backup.service` and runs as a oneshot service, meaning it:

- Executes the backup process
- Completes the task
- Exits until the next scheduled run

### Timer File

The timer is defined in `go-backup.timer` and controls when the service runs.

### Logs

View service logs using:

```bash
journalctl -u go-backup.service
```

View timer logs using:

```bash
journalctl -u go-backup.timer
```

## Troubleshooting

1. **Service won't start**

   - Check service status: `make status`
   - View logs: `journalctl -u go-backup.service`
   - Verify configuration: `/etc/go-backup/config.yaml`

2. **Timer not running**

   - Check timer status: `make schedule`
   - Verify timer file: `/etc/systemd/system/go-backup.timer`
   - Check system time: `timedatectl status`

3. **Permission issues**
   - Verify the service is running as root
   - Check file permissions in backup locations
   - Ensure configuration file is readable

## Security Considerations

- The service runs as root to ensure access to all necessary files
- Configuration files are stored in `/etc/go-backup/`
- Binary and script files are installed in `/usr/local/bin/`
- Systemd service files are in `/etc/systemd/system/`

## Development

To modify the backup agent:

1. Make your changes to the Go code
2. Build the binary: `make build`
3. Reinstall the service: `make install`
4. Restart the service: `make restart`

## License

[Add your license information here]
