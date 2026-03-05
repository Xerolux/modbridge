#!/bin/bash

# ModBridge Deployment Script
# Deploy to remote server via SSH and compile from source

set -e  # Exit on any error

# Configuration
REMOTE_USER="${REMOTE_USER:-basti}"
REMOTE_HOST="${REMOTE_HOST:-192.168.178.196}"
REMOTE_PORT="${REMOTE_PORT:-22}"
REMOTE_PATH="${REMOTE_PATH:-/home/modbridge}"
GO_VERSION="1.26"
NODE_VERSION="22"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== ModBridge Deployment Script ===${NC}"
echo "Target: $REMOTE_USER@$REMOTE_HOST:$REMOTE_PORT"
echo "Deploy Path: $REMOTE_PATH"
echo ""

# Check if SSH key exists
if [ ! -z "$SSH_KEY" ]; then
    SSH_OPTS="-i $SSH_KEY"
elif [ -f ~/.ssh/id_rsa ]; then
    SSH_OPTS="-i ~/.ssh/id_rsa"
else
    SSH_OPTS=""
fi

SSH_CMD="ssh $SSH_OPTS -p $REMOTE_PORT $REMOTE_USER@$REMOTE_HOST"

# Step 1: Check connectivity
echo -e "${YELLOW}1. Testing SSH connection...${NC}"
if $SSH_CMD "echo 'SSH connection OK'" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ SSH connection successful${NC}"
else
    echo -e "${RED}✗ SSH connection failed${NC}"
    echo "Please ensure:"
    echo "  1. Server is reachable at $REMOTE_HOST:$REMOTE_PORT"
    echo "  2. User '$REMOTE_USER' can connect via SSH"
    echo "  3. SSH key is configured (use: export SSH_KEY=/path/to/key)"
    exit 1
fi

# Step 2: Check if Go and Node are installed
echo -e "${YELLOW}2. Checking required tools on server...${NC}"
$SSH_CMD bash << 'REMOTE_SCRIPT'
    echo "Checking Go..."
    if ! command -v go &> /dev/null; then
        echo "⚠ Go is not installed. Please install Go 1.26+"
        exit 1
    fi
    GO_V=$(go version | awk '{print $3}')
    echo "✓ Go version: $GO_V"

    echo "Checking Node..."
    if ! command -v npm &> /dev/null; then
        echo "⚠ Node/npm is not installed. Installing dependencies will fail."
        echo "  Please install Node.js 22+ on the server"
    else
        NODE_V=$(node --version)
        echo "✓ Node version: $NODE_V"
    fi

    echo "Checking Git..."
    if ! command -v git &> /dev/null; then
        echo "⚠ Git is not installed"
        exit 1
    fi
    echo "✓ Git is installed"
REMOTE_SCRIPT

# Step 3: Clone or pull repository
echo -e "${YELLOW}3. Cloning/Updating repository...${NC}"
$SSH_CMD bash << REMOTE_SCRIPT
    if [ -d "$REMOTE_PATH/.git" ]; then
        echo "Repository exists, pulling latest changes..."
        cd $REMOTE_PATH
        git fetch origin
        git checkout main
        git pull origin main
    else
        echo "Cloning repository..."
        mkdir -p $(dirname $REMOTE_PATH)
        git clone https://github.com/Xerolux/modbridge.git $REMOTE_PATH
        cd $REMOTE_PATH
    fi
    echo "✓ Repository ready"
REMOTE_SCRIPT

# Step 4: Build frontend
echo -e "${YELLOW}4. Building frontend...${NC}"
$SSH_CMD bash << REMOTE_SCRIPT
    cd $REMOTE_PATH/frontend
    echo "Installing npm dependencies..."
    npm ci --prefer-offline
    echo "Building Vue application..."
    npm run build
    echo "Copying frontend to pkg/web/dist..."
    rm -rf $REMOTE_PATH/pkg/web/dist
    cp -r $REMOTE_PATH/frontend/dist $REMOTE_PATH/pkg/web/dist
    echo "✓ Frontend built and copied successfully"
REMOTE_SCRIPT

# Step 5: Build Go binary
echo -e "${YELLOW}5. Building Go binary...${NC}"
$SSH_CMD bash << REMOTE_SCRIPT
    cd $REMOTE_PATH
    echo "Downloading Go dependencies..."
    go mod download
    go mod verify

    echo "Building binary (CGO_ENABLED=1 for SQLite3)..."
    CGO_ENABLED=1 go build \
        -ldflags="-s -w -X main.Version=\$(git describe --tags --always --dirty)" \
        -o modbridge-new \
        ./main.go

    echo "✓ Binary compiled: modbridge-new"
REMOTE_SCRIPT

# Step 6: Backup and switch binary
echo -e "${YELLOW}6. Backing up and switching binary...${NC}"
$SSH_CMD bash << REMOTE_SCRIPT
    cd $REMOTE_PATH

    # Backup old binary
    if [ -f modbridge ]; then
        BACKUP_NAME="modbridge.backup.\$(date +%Y%m%d_%H%M%S)"
        cp modbridge "\$BACKUP_NAME"
        echo "✓ Old binary backed up: \$BACKUP_NAME"
    fi

    # Switch new binary
    mv modbridge-new modbridge
    chmod +x modbridge
    echo "✓ New binary activated: modbridge"
REMOTE_SCRIPT

# Step 7: Restart service (if systemd service exists)
echo -e "${YELLOW}7. Checking for systemd service...${NC}"
if $SSH_CMD "systemctl is-active --quiet modbridge 2>/dev/null; echo \$?" | grep -q "0"; then
    echo "Found active modbridge service. Restarting..."
    $SSH_CMD "sudo systemctl restart modbridge" || {
        echo -e "${YELLOW}⚠ Could not restart service (may need sudo password)${NC}"
        echo "You can manually restart with: sudo systemctl restart modbridge"
    }
    sleep 2
    if $SSH_CMD "systemctl is-active --quiet modbridge"; then
        echo -e "${GREEN}✓ Service restarted successfully${NC}"
    else
        echo -e "${RED}✗ Service failed to start${NC}"
        echo "Check logs with: sudo journalctl -u modbridge -n 50"
    fi
else
    echo -e "${YELLOW}⚠ No systemd service found${NC}"
    echo "You can start manually with: cd $REMOTE_PATH && ./modbridge"
fi

# Step 8: Health check
echo -e "${YELLOW}8. Running health check...${NC}"
$SSH_CMD bash << REMOTE_SCRIPT
    sleep 2
    if curl -s http://localhost:8080/api/health | grep -q "ok"; then
        echo "✓ Health check passed"
    else
        echo "⚠ Health check failed or service not responding"
    fi
REMOTE_SCRIPT

echo ""
echo -e "${GREEN}=== Deployment Complete ===${NC}"
echo "Server: $REMOTE_USER@$REMOTE_HOST"
echo "Path: $REMOTE_PATH"
echo ""
echo "Next steps:"
echo "  1. Verify the deployment: curl http://$REMOTE_HOST:8080/api/health"
echo "  2. Check logs: ssh $REMOTE_USER@$REMOTE_HOST 'tail -f $REMOTE_PATH/proxy.log'"
echo "  3. Rollback if needed: ssh $REMOTE_USER@$REMOTE_HOST 'cd $REMOTE_PATH && ls -lh modbridge*'"
