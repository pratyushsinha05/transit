#!/bin/sh
# ============================================================================
# Health Check Script for Transit Backend
# ============================================================================
# Used by Docker health check and Kubernetes readiness probes
# Exit 0 = healthy, Exit 1 = unhealthy
# ============================================================================

set -e

# Configuration
HOST="${HEALTH_HOST:-localhost}"
PORT="${SERVER_PORT:-8080}"
ENDPOINT="${HEALTH_ENDPOINT:-/health}"
TIMEOUT="${HEALTH_TIMEOUT:-5}"

# Perform health check
if command -v wget >/dev/null 2>&1; then
    # Use wget (available in Alpine)
    wget --no-verbose --tries=1 --timeout="${TIMEOUT}" --spider "http://${HOST}:${PORT}${ENDPOINT}"
elif command -v curl >/dev/null 2>&1; then
    # Fallback to curl
    curl -sf --max-time "${TIMEOUT}" "http://${HOST}:${PORT}${ENDPOINT}" > /dev/null
else
    # No HTTP client available, try netcat
    if command -v nc >/dev/null 2>&1; then
        nc -z "${HOST}" "${PORT}"
    else
        echo "No health check tool available (wget, curl, or nc)"
        exit 1
    fi
fi

# If we got here, health check passed
exit 0
