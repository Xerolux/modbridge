#!/bin/bash
# Build script that fixes go:embed issue with files starting with underscore

echo "Building frontend..."
cd frontend
npm run build
cd ..

echo "Fixing files that start with underscore (go:embed compatibility)..."
# Find all files in dist/assets that start with underscore and rename them
find frontend/dist/assets -name "_*" -type f | while read file; do
    dir=$(dirname "$file")
    basename=$(basename "$file")
    # Remove leading underscore
    newname="${basename#_}"
    mv "$file" "$dir/$newname"
    # Update references in other files
    find frontend/dist -name "*.js" -o -name "*.html" | xargs sed -i "s|/$basename|/$newname|g"
    echo "Renamed: $basename -> $newname"
done

echo "Copying to pkg/web/dist..."
rm -rf pkg/web/dist
cp -r frontend/dist pkg/web/dist

echo "Build complete!"
