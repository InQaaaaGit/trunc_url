#!/bin/bash

# Пример сборки с передачей информации о версии через ldflags
VERSION="v1.0.0"
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "Building shortener with build info:"
echo "Version: $VERSION"
echo "Date: $DATE" 
echo "Commit: $COMMIT"

# Сборка shortener
go build -ldflags \
  "-X main.buildVersion=$VERSION \
   -X main.buildDate=$DATE \
   -X main.buildCommit=$COMMIT" \
  -o shortener ./cmd/shortener

# Сборка api  
go build -ldflags \
  "-X main.buildVersion=$VERSION \
   -X main.buildDate=$DATE \
   -X main.buildCommit=$COMMIT" \
  -o api ./cmd/api

echo "Build completed!"
echo ""
echo "To test build info, run:"
echo "./shortener"
echo "or"
echo "./api" 