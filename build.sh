#!/bin/bash

# Build script for hibp-checker
# Cross-compiles for Windows x64, Linux x64, and macOS (Intel + Apple Silicon)

set -e

APP_NAME="hibp-checker"
OUTPUT_DIR="dist"

# Clean and create output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

echo "Building $APP_NAME..."

# Windows x64
echo "  -> Windows x64"
GOOS=windows GOARCH=amd64 go build -o "$OUTPUT_DIR/${APP_NAME}-windows-amd64.exe"

# Linux x64
echo "  -> Linux x64"
GOOS=linux GOARCH=amd64 go build -o "$OUTPUT_DIR/${APP_NAME}-linux-amd64"

# macOS Intel
echo "  -> macOS Intel (amd64)"
GOOS=darwin GOARCH=amd64 go build -o "$OUTPUT_DIR/${APP_NAME}-darwin-amd64"

# macOS Apple Silicon
echo "  -> macOS Apple Silicon (arm64)"
GOOS=darwin GOARCH=arm64 go build -o "$OUTPUT_DIR/${APP_NAME}-darwin-arm64"

echo ""
echo "Build complete! Binaries are in the '$OUTPUT_DIR' directory:"
ls -lh "$OUTPUT_DIR"
