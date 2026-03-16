#!/bin/bash

# ============================================================================
# TRANSIT POC - FULL STACK CLEANUP SCRIPT
# ============================================================================
# This script handles the cleanup of both the Go backend (Docker containers,
# volumes, artifacts) and the React frontend (node_modules, dist).
# ============================================================================

# Color Output
BLUE='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║            TRANSIT POC FULL STACK CLEANUP                      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# ============================================================================
# PHASE 1: CLEANING BACKEND
# ============================================================================
echo -e "${BLUE}[1/2] Cleaning Backend via Makefile...${NC}"

if command -v make >/dev/null 2>&1; then
    make clean-all
else
    echo -e "${RED}✗ Make not found, cannot clean backend via Makefile${NC}"
    echo -e "${YELLOW}Attempting manual docker cleanup...${NC}"
    if command -v docker >/dev/null 2>&1; then
        docker compose -f ./infra/docker-compose.yml down -v --remove-orphans
    fi
fi

echo ""

# ============================================================================
# PHASE 2: CLEANING FRONTEND
# ============================================================================
echo -e "${BLUE}[2/2] Cleaning Frontend...${NC}"
FRONTEND_DIR="./frontend"

if [ -d "$FRONTEND_DIR" ]; then
    echo "Removing frontend build artifacts and dependencies..."
    
    if [ -d "$FRONTEND_DIR/dist" ]; then
        rm -rf "$FRONTEND_DIR/dist"
        echo "  - Removed dist/"
    fi
    
    if [ -d "$FRONTEND_DIR/node_modules" ]; then
        rm -rf "$FRONTEND_DIR/node_modules"
        echo "  - Removed node_modules/"
    fi
    
    echo -e "${GREEN}✓ Frontend cleaned${NC}"
else
    echo -e "${YELLOW}⚠ Frontend directory '$FRONTEND_DIR' not found, skipping.${NC}"
fi

echo ""

# ============================================================================
# SUMMARY
# ============================================================================
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                      CLEANUP COMPLETE                          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Note: This performed a deep clean. You will need to run 'bash deploy.sh'${NC}"
echo -e "${YELLOW}to reinstall frontend dependencies and rebuild images/containers.${NC}"
echo ""
