#!/usr/bin/env bash
set -euo pipefail

# Helper commands: start-service java|go|python|ts|dotnet, kill-service
sudo ln -sf /workspaces/.devcontainer/bin/start-service /usr/local/bin/start-service
sudo ln -sf /workspaces/.devcontainer/bin/kill-service /usr/local/bin/kill-service

# Tools (HTTPie, temporal, uv) are installed in .devcontainer/Dockerfile. The caches these
# commands fill (Gradle, npm, uv, Go modules) live in /workspaces/.devcache (see
# docker-compose.yml), so they are fast after the first create.

(cd order-service-java && ./gradlew --no-daemon classes)
(cd order-service-go && go build ./...)
(cd order-service-python && uv sync)
npm ci --prefix order-service-ts
(cd order-service-dotnet && dotnet restore)

