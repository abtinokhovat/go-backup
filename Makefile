.PHONY: build install uninstall start stop status restart enable disable schedule

# Variables
BINARY_NAME=backup-agent
SERVICE_NAME=go-backup
INSTALL_DIR=/usr/local/bin
SERVICE_DIR=/etc/systemd/system
CONFIG_DIR=/etc/$(SERVICE_NAME)

# Build the binary
build:
	go build -o $(BINARY_NAME) .

# Install the service and timer
install: build
	@echo "Installing $(SERVICE_NAME)..."
	@sudo mkdir -p $(INSTALL_DIR)
	@sudo cp $(BINARY_NAME) $(INSTALL_DIR)/
	@sudo cp dara-backup.sh $(INSTALL_DIR)/
	@sudo chmod +x $(INSTALL_DIR)/dara-backup.sh
	@sudo cp $(SERVICE_NAME).service $(SERVICE_DIR)/
	@sudo cp $(SERVICE_NAME).timer $(SERVICE_DIR)/
	@sudo mkdir -p $(CONFIG_DIR)
	@sudo cp config.yaml $(CONFIG_DIR)/
	@sudo systemctl daemon-reload
	@echo "Installation complete. Use 'make enable' to enable the service."

# Uninstall the service and timer
uninstall:
	@echo "Uninstalling $(SERVICE_NAME)..."
	@sudo systemctl stop $(SERVICE_NAME).timer
	@sudo systemctl disable $(SERVICE_NAME).timer
	@sudo rm -f $(SERVICE_DIR)/$(SERVICE_NAME).service
	@sudo rm -f $(SERVICE_DIR)/$(SERVICE_NAME).timer
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo rm -f $(INSTALL_DIR)/dara-backup.sh
	@sudo rm -rf $(CONFIG_DIR)
	@sudo systemctl daemon-reload
	@echo "Uninstallation complete."

# Service management commands
start:
	@sudo systemctl start $(SERVICE_NAME).timer

stop:
	@sudo systemctl stop $(SERVICE_NAME).timer

status:
	@sudo systemctl status $(SERVICE_NAME).timer

restart:
	@sudo systemctl restart $(SERVICE_NAME).timer

enable:
	@sudo systemctl enable $(SERVICE_NAME).timer
	@sudo systemctl start $(SERVICE_NAME).timer

disable:
	@sudo systemctl disable $(SERVICE_NAME).timer
	@sudo systemctl stop $(SERVICE_NAME).timer

# Schedule management
schedule:
	@echo "Current schedule:"
	@sudo systemctl list-timers $(SERVICE_NAME).timer
	@echo "\nTo modify the schedule, edit $(SERVICE_DIR)/$(SERVICE_NAME).timer"
	@echo "Available schedule formats:"
	@echo "  OnCalendar=daily              # Run daily at 00:00:00"
	@echo "  OnCalendar=hourly             # Run every hour"
	@echo "  OnCalendar=weekly             # Run weekly on Sunday at 00:00:00"
	@echo "  OnCalendar=*-*-* 12:00:00     # Run daily at 12:00:00"
	@echo "  OnCalendar=Mon *-*-* 12:00:00 # Run every Monday at 12:00:00"
	@echo "After editing, run: sudo systemctl daemon-reload && sudo systemctl restart $(SERVICE_NAME).timer" 