#!/bin/bash

set -e

VERSION=${1:-$(git describe --tags --always 2>/dev/null || echo "dev")}
BINARY_NAME=luakit
DIST_DIR=dist

echo "======================================"
echo "Building release artifacts for $VERSION"
echo "======================================"
echo

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "Building for linux/amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" -o "${DIST_DIR}/${BINARY_NAME}-${VERSION}-linux-amd64" ./cmd/luakit
echo "Built ${BINARY_NAME}-${VERSION}-linux-amd64"

echo "Building for linux/arm64..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=${VERSION}" -o "${DIST_DIR}/${BINARY_NAME}-${VERSION}-linux-arm64" ./cmd/luakit
echo "Built ${BINARY_NAME}-${VERSION}-linux-arm64"

echo "Building for darwin/amd64..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" -o "${DIST_DIR}/${BINARY_NAME}-${VERSION}-darwin-amd64" ./cmd/luakit
echo "Built ${BINARY_NAME}-${VERSION}-darwin-amd64"

echo "Building for darwin/arm64..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=${VERSION}" -o "${DIST_DIR}/${BINARY_NAME}-${VERSION}-darwin-arm64" ./cmd/luakit
echo "Built ${BINARY_NAME}-${VERSION}-darwin-arm64"

echo
echo "======================================"
echo "Generating checksums"
echo "======================================"
echo

cd "$DIST_DIR"
shasum -a 256 "${BINARY_NAME}-"* > checksums.txt
echo "Generated checksums.txt:"
cat checksums.txt
cd ..

echo
echo "======================================"
echo "Summary"
echo "======================================"
echo "Built binaries:"
ls -lh "$DIST_DIR/${BINARY_NAME}-"* | while read line; do
    size=$(echo $line | awk '{print $5}')
    name=$(echo $line | awk '{print $9}')
    echo "  $name ($size)"
done
echo
echo "Checksums file: $DIST_DIR/checksums.txt"
echo
echo "All release artifacts ready for version: $VERSION"
echo
echo "To create the release:"
echo "  1. git tag -a $VERSION -m Release\ $VERSION"
echo "  2. git push origin $VERSION"
echo "  3. Monitor the workflow at: https://github.com/kasuboski/luakit/actions"
