#!/bin/bash
set -e
# Build script for ModBridge frontend + Go embed preparation
# Note: Vite is configured to never generate underscore-prefixed filenames
# (see frontend/vite.config.js rollupOptions) so no post-build renaming needed.

echo "Building frontend..."
cd frontend
npm install
npm run build
cd ..

echo "Copying to pkg/web/dist..."
rm -rf pkg/web/dist
cp -r frontend/dist pkg/web/dist

echo "Build complete!"
