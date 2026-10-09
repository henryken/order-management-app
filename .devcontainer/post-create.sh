#!/usr/bin/env bash
set -euo pipefail

# HTTPie (`http`) for the README's example requests
sudo apt-get update -y
sudo apt-get install -y --no-install-recommends httpie

# Temporal CLI. Defaults to localhost:7233, which is the sidecar Temporal server.
curl -sSf https://temporal.download/cli.sh | sh
sudo ln -sf "$HOME/.temporalio/bin/temporal" /usr/local/bin/temporal

# uv runs the Python order service
curl -LsSf https://astral.sh/uv/install.sh | sh
sudo ln -sf "$HOME/.local/bin/uv" /usr/local/bin/uv

npm install --prefix downstream-services
(cd order-service-java && ./gradlew --no-daemon classes)
(cd order-service-go && go mod download)
(cd order-service-python && uv sync)
npm install --prefix order-service-ts
