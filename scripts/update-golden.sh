#!/usr/bin/env bash
set -e

# Regenerate golden test files
# This script regenerates all golden .pb files by running luakit build on each script

cd "$(dirname "$0")/.."
GOLDEN_DIR="test/integration/testdata/golden"
SCRIPT_DIR="test/integration/golden_scripts"

echo "Regenerating golden files..."

for script in "$SCRIPT_DIR"/*.lua; do
    script_name=$(basename "$script" .lua)
    output_file="$GOLDEN_DIR/${script_name}.pb"

    echo "Processing $script_name..."

    if ./dist/luakit build -o "$output_file" "$script"; then
        echo "  ✓"
    else
        echo "  ✗ FAILED"
    fi
    echo ""
done

echo "Done. Run 'go test ./test/integration/ -run TestA' to verify."
