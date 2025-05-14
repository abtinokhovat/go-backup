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

## Configuration

The backup agent is configured using a YAML file located at `/etc/go-backup/config.yaml`. The configuration file is copied during installation, but you can modify it at any time.

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

The service runs as a systemd timer, which provides flexible scheduling options. By default, it runs daily at 12:00.

### Available Schedule Formats

- Daily at midnight: `OnCalendar=daily`
- Every hour: `OnCalendar=hourly`
- Weekly (Sunday at midnight): `OnCalendar=weekly`
- Daily at specific time: `OnCalendar=*-*-* 12:00:00`
- Weekly on specific day: `OnCalendar=Mon *-*-* 12:00:00`

To modify the schedule:

1. Edit the timer file: `/etc/systemd/system/go-backup.timer`
2. Reload and restart the service:

```bash
sudo systemctl daemon-reload
make restart
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
