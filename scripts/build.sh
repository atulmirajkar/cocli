#!/bin/bash

# Build script for cross-platform binaries
# Creates binaries for macOS (Apple Silicon & Intel) and Linux (x86_64 & ARM64)

set -e

OUTPUT_DIR="dist"
VERSION=${1:-"1.0.0"}

echo "🔨 Building CoCli v${VERSION}"
echo "================================================"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Define build targets
TARGETS=(
    "darwin:arm64:cocli-darwin-arm64"
    "darwin:amd64:cocli-darwin-amd64"
    "linux:amd64:cocli-linux-amd64"
    "linux:arm64:cocli-linux-arm64"
)

# Build each target
for target in "${TARGETS[@]}"; do
    IFS=':' read -r goos goarch output <<< "$target"
    
    echo ""
    echo "📦 Building for ${goos}/${goarch}..."
    
    GOOS="$goos" GOARCH="$goarch" go build \
        -o "$OUTPUT_DIR/$output" \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        .
    
    # Make executable
    chmod +x "$OUTPUT_DIR/$output"
    
    # Get file size
    SIZE=$(ls -lh "$OUTPUT_DIR/$output" | awk '{print $5}')
    echo "   ✓ Created: $OUTPUT_DIR/$output ($SIZE)"
done

echo ""
echo "✨ Build complete!"
echo ""
echo "📋 Files created in $OUTPUT_DIR/:"
ls -lh "$OUTPUT_DIR/"
echo ""
echo "🚀 Ready for distribution!"
