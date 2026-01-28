#!/bin/bash

# CoCli Release Process Automation Script
# Usage: ./scripts/release.sh [version]

set -e

VERSION=${1:-"$(date +%Y.%m.%d)"}

echo "Starting CoCli Release Process v${VERSION}"
echo "================================================"

# Step 1: Quality Assurance
echo "Step 1: Running quality assurance checks..."
echo "   Running tests..."
go test ./...

echo "   Verifying build..."
go build -o cocli-test
rm cocli-test

# Step 2: Cross-platform build
echo ""
echo "Step 2: Building cross-platform binaries..."
chmod +x scripts/build.sh
./scripts/build.sh ${VERSION}

# Step 3: Update releases
echo ""
echo "Step 3: Updating release directory..."
cp dist/cocli-darwin-arm64 releases/
cp dist/cocli-darwin-amd64 releases/
cp dist/cocli-linux-amd64 releases/
cp dist/cocli-linux-arm64 releases/
chmod +x releases/cocli-*

# Step 4: Validation
echo ""
echo "Step 4: Validating binaries..."
echo "   File sizes and permissions:"
ls -la releases/cocli-*

echo ""
echo "   Architecture verification:"
file releases/cocli-darwin-arm64
file releases/cocli-darwin-amd64

# Step 5: Cleanup
echo ""
echo "Step 5: Cleaning up..."
rm -rf dist/

echo ""
echo "Release process complete!"
echo ""
echo "Summary:"
echo "   Version: v${VERSION}"
echo "   Binaries updated in releases/"
echo "   All platforms built successfully"
echo ""
echo "Manual steps remaining:"
echo "   1. Review changes: git status"
echo "   2. Add binaries: git add releases/cocli-*"
echo "   3. Add source changes: git add [modified files]"
echo "   4. Commit: git commit -m 'Update release binaries v${VERSION}'"
echo "   5. Tag: git tag -a v${VERSION} -m 'Release v${VERSION}'"
echo "   6. Push: git push origin main --tags"
echo "   7. Create GitHub release with binaries"