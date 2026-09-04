#!/bin/bash

# ============================================================================
# TRANSIT POC - FULL STACK DEPLOYMENT SCRIPT
# ============================================================================
# This script handles the end-to-end deployment of both the Go backend
# and the React frontend.
# ============================================================================

# Color Output
BLUE='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║            TRANSIT POC FULL STACK DEPLOYMENT                   ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# ============================================================================
# PHASE 1: PREREQUISITES & PORT CHECKS
# ============================================================================
echo -e "${BLUE}[1/3] Checking prerequisites and environment...${NC}"

command -v make >/dev/null 2>&1 || {
	echo -e "${RED}✗ Make not found. Please install make.${NC}"
	exit 1
}

command -v docker >/dev/null 2>&1 || {
	echo -e "${RED}✗ Docker not found. Please install Docker.${NC}"
	exit 1
}

command -v npm >/dev/null 2>&1 || {
	echo -e "${RED}✗ npm not found. Please install Node.js.${NC}"
	exit 1
}

command -v node >/dev/null 2>&1 || {
	echo -e "${RED}✗ Node.js not found. Please install Node.js.${NC}"
	exit 1
}

# Verify Docker daemon is running
docker info >/dev/null 2>&1 || {
	echo -e "${RED}✗ Docker daemon is not running. Please start Docker Desktop.${NC}"
	exit 1
}

# Check for conflicting local ports (5432, 6379, 8080, 4173)
check_port() {
	local port=$1
	local name=$2
	if lsof -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
		local proc
		proc=$(lsof -iTCP:"${port}" -sTCP:LISTEN -F c 2>/dev/null | grep '^c' | cut -c2- | head -1)
		if [ "$proc" != "docker" ] && [ "$proc" != "com.docker" ] && [ "$proc" != "docker-proxy" ]; then
			echo -e "${RED}✗ Port ${port} (${name}) is already in use by '${proc}'.${NC}"
			echo -e "${YELLOW}Please stop the conflicting process before deploying.${NC}"
			exit 1
		fi
	fi
}

check_port 5432 "PostgreSQL"
check_port 6379 "Redis"
check_port 8080 "Backend API"
check_port 4173 "Frontend Preview"

echo -e "${GREEN}  ✓ All prerequisites installed and ports available${NC}"
echo ""

# ============================================================================
# PHASE 2: DEPLOY BACKEND (Rebuilds image, starts stack, waits for health)
# ============================================================================
echo -e "${BLUE}[2/3] Deploying Backend via Docker Stack...${NC}"
echo "Building fresh backend image and starting services..."

# Spin up Docker stack (docker-up runs build-docker and waits for health)
make docker-up
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to start Docker stack or backend failed health check.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Backend deployed and verified healthy${NC}"
echo ""

# ============================================================================
# PHASE 3: DEPLOY FRONTEND
# ============================================================================
echo -e "${BLUE}[3/3] Deploying Frontend...${NC}"
FRONTEND_DIR="./frontend"

if [ ! -d "$FRONTEND_DIR" ]; then
    echo -e "${RED}✗ Frontend directory '$FRONTEND_DIR' not found.${NC}"
    exit 1
fi

cd "$FRONTEND_DIR" || exit 1

echo "Installing frontend dependencies..."
npm install
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ npm install failed.${NC}"
    exit 1
fi

echo "Building frontend..."
npm run build
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Frontend build failed.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Frontend built successfully${NC}"
cd ..
echo ""

# ============================================================================
# SUMMARY
# ============================================================================
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                      DEPLOYMENT COMPLETE                       ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Services Available:${NC}"
echo "  - Frontend UI:  http://localhost:4173"
echo "  - Backend API:  http://localhost:8080"
echo "  - Health Check: http://localhost:8080/health"
echo "  - WebSocket:    ws://localhost:8080/ws"
echo "  - PostgreSQL:   localhost:5432"
echo "  - Redis:        localhost:6379"
echo ""
echo -e "${YELLOW}Useful Commands:${NC}"
echo "  - View backend logs: make docker-logs"
echo "  - Stop all services: make docker-down"
echo ""

echo -e "${BLUE}Starting frontend preview server at http://localhost:4173...${NC}"
cd frontend && npm run preview
