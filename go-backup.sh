#!/bin/bash

# Check if .env file path is provided as an argument
if [ -z "$1" ]; then
  echo "Usage: $0 path_to_env_file"
  exit 1
fi

# Load environment variables from the specified .env file
export $(grep -v '^#' "$1" | xargs)

# Run the Golang binary
./backup-agent