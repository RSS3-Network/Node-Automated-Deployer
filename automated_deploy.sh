#!/bin/bash

# Deployer information
DEPLOYER_NAME="Node-Automated-Deployer"
DEPLOYER_VERSION="v0.3.0"
DEPLOYER_RELEASE_URL="https://github.com/RSS3-Network/$DEPLOYER_NAME/releases"

# The version of RSS3 Node to deploy
NODE_VERSION="v0.5.9"

# Detect the operating system
OS=$(uname -s)
ARCH=$(uname -m)

# Function to check if Docker is installed and running
check_docker() {
    if ! command -v docker &> /dev/null; then
        echo "Docker is not installed. Please install Docker: https://docs.docker.com/engine/install/ and rerun this script."
        exit 1
    fi
    if ! docker ps &> /dev/null; then
        echo "Docker is not running. Please start Docker and rerun this script."
        exit 1
    fi
}

check_docker

# Function to check if Docker Compose is installed
check_docker_compose() {
    if command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    elif command -v docker &> /dev/null && docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    else
        echo "Docker Compose is not installed. Please install Docker Compose: https://docs.docker.com/compose/install/ and rerun this script."
        exit 1
    fi
}

check_docker_compose

# Function to install curl if it's not already installed
install_curl() {
    if ! command -v curl &> /dev/null; then
        echo "curl is not installed. Installing curl..."
        case $OS in
            Linux)
                if command -v apt-get &> /dev/null; then
                    sudo apt-get update
                    sudo apt-get install -y curl
                elif command -v yum &> /dev/null; then
                    sudo yum install -y curl
                elif command -v pacman &> /dev/null; then
                    sudo pacman -Syu curl
                else
                    echo "Unsupported package manager. Please install curl manually, or download the Deployer manually from $DEPLOYER_RELEASE_URL."
                    exit 1
                fi
                ;;
            MINGW*|MSYS*|CYGWIN*)
                echo "Please install curl manually on Windows."
                exit 1
                ;;
            *)
                echo "Unsupported operating system: $OS"
                exit 1
                ;;
        esac
    fi
}

# Install curl if not installed
install_curl

# Determine the correct file name based on OS and architecture
case $OS in
    Darwin)
        case $ARCH in
            arm64)
                FILE="${DEPLOYER_NAME}_Darwin_arm64.tar.gz"
                ;;
            x86_64)
                FILE="${DEPLOYER_NAME}_Darwin_x86_64.tar.gz"
                ;;
            *)
                echo "Unsupported architecture: $ARCH"
                exit 1
                ;;
        esac
        ;;
    Linux)
        case $ARCH in
            arm64)
                FILE="${DEPLOYER_NAME}_Linux_arm64.tar.gz"
                ;;
            i386)
                FILE="${DEPLOYER_NAME}_Linux_i386.tar.gz"
                ;;
            x86_64)
                FILE="${DEPLOYER_NAME}_Linux_x86_64.tar.gz"
                ;;
            *)
                echo "Unsupported architecture: $ARCH"
                exit 1
                ;;
        esac
        ;;
    MINGW*|MSYS*|CYGWIN*)
        case $ARCH in
            arm64)
                FILE="${DEPLOYER_NAME}_Windows_arm64.zip"
                ;;
            i386)
                FILE="${DEPLOYER_NAME}_Windows_i386.zip"
                ;;
            x86_64)
                FILE="${DEPLOYER_NAME}_Windows_x86_64.zip"
                ;;
            *)
                echo "Unsupported architecture: $ARCH"
                exit 1
                ;;
        esac
        ;;
    *)
        echo "Unsupported operating system: $OS"
        exit 1
        ;;
esac

# URL for the download
URL="$DEPLOYER_RELEASE_URL/download/$DEPLOYER_VERSION/$FILE"

echo "Downloading $URL..."

# Download the file
curl -L $URL -o $FILE

# Check if the download was successful
if [ $? -eq 0 ]; then
    echo "Download successful, extracting the file..."

    # Extract the file
    case $FILE in
        *.tar.gz)
            tar -xzf $FILE
            ;;
        *.zip)
            unzip $FILE
            ;;
        *)
            echo "Unsupported file format: $FILE"
            exit 1
            ;;
    esac

    # Check if the extraction was successful
    if [ $? -eq 0 ]; then
        echo "Extraction successful."
    else
        echo "Extraction failed."
        exit 1
    fi
else
    echo "Download failed."
    exit 1
fi

# Get the directory of the script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Check if config.yaml exists in the script's directory and run Deployer if it does
if [ -f "$SCRIPT_DIR/config.yaml" ]; then
    echo "config.yaml found in the script's directory, moving it to config folder..."

    mkdir -p "$SCRIPT_DIR/config"
    mv "$SCRIPT_DIR/config.yaml" "$SCRIPT_DIR/config/config.yaml"
    echo "DO NOT DELETE/MOVE THE config FOLDER OR ITS CONTENTS!"

    export NODE_VERSION
    echo "running the deployer..."
    "$SCRIPT_DIR/node-automated-deployer" > "$SCRIPT_DIR/docker-compose.yaml"

    # Check if docker-compose.yaml was successfully created
    if [ -f "$SCRIPT_DIR/docker-compose.yaml" ]; then
        echo "docker-compose.yaml created, starting Docker Compose..."
        (cd "$SCRIPT_DIR" && $COMPOSE_CMD up -d)

        # Check if Docker Compose started successfully
        if [ $? -eq 0 ]; then
            echo "Deployment process completed successfully."
            echo "🎉 Welcome to the RSS3 Network!"
        else
            echo "Failed to start Docker Compose."
            exit 1
        fi
    else
        echo "Failed to create docker-compose.yaml."
        exit 1
    fi
else
    echo "config.yaml not found, please create a config.yaml file or generate one at https://explorer.rss3.io."
fi
