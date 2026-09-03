#!/usr/bin/env bash
set -euo pipefail

# Find repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

CHECK="${1:-all}"

check_layering() {
    echo "Checking layering rule (CLAUDE.md §3.2)..."
    local violations
    violations=$(grep -rn '"transit-backend/internal/database"' \
        "${REPO_ROOT}/backend/internal/handlers" \
        "${REPO_ROOT}/backend/internal/services" \
        --include="*.go" 2>/dev/null | grep -v '_test.go' || true)

    if [ -n "${violations}" ]; then
        echo "✗ Layering violation: handlers or services import database directly:"
        echo "${violations}"
        return 1
    fi

    echo "✓ Layering rule passed (no handlers or services import database directly)"
    return 0
}

case "${CHECK}" in
    --check=layering|layering)
        check_layering
        ;;
    --check=all|all|"")
        check_layering
        ;;
    *)
        echo "Unknown check option: ${CHECK}"
        echo "Usage: $0 [--check=layering|--check=all]"
        exit 1
        ;;
esac
