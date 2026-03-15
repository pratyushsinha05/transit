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
# PHASE 1: PREREQUISITES CHECK
# ============================================================================
echo -e "${BLUE}[1/3] Checking prerequisites...${NC}"

command -v make >/dev/null 2>&1 || {
	echo -e "${RED}✗ Make not found${NC}"
	exit 1
}

command -v docker >/dev/null 2>&1 || {
	echo -e "${RED}✗ Docker not found${NC}"
	exit 1
}

command -v npm >/dev/null 2>&1 || {
	echo -e "${RED}✗ npm not found. Please install Node.js.${NC}"
	exit 1
}
echo -e "${GREEN}  ✓ All prerequisites installed${NC}"
echo ""

# ============================================================================
# PHASE 2: DEPLOY BACKEND
# ============================================================================
echo -e "${BLUE}[2/3] Deploying Backend via Docker Stack...${NC}"
echo "Running Makefile commands..."

# Check backend dependencies
make check-deps
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Backend dependency check failed. Please resolve the issues above.${NC}"
    exit 1
fi

# Spin up Docker stack
make docker-up
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to start Docker stack.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Backend deployed successfully${NC}"
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
echo -e "${GREEN}Backend Services:${NC}"
echo "  - API Server: http://localhost:8080"
echo "  - PostgreSQL: localhost:5432"
echo "  - Redis:      localhost:6379"
echo ""
echo -e "${GREEN}Frontend Status:${NC}"
echo "  - Built files are ready in: ./frontend/dist"
echo "  - To serve frontend locally, you can run:"
echo "      cd frontend && npm run preview"
echo ""
echo -e "${YELLOW}To view backend logs:${NC} make docker-logs"
echo -e "${YELLOW}To stop all services:${NC} make docker-down"
echo ""

echo -e "${BLUE}Starting frontend preview server...${NC}"
cd frontend && npm run preview
