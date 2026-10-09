#!/usr/bin/env bash
set -euo pipefail

# HTTPie (`http`) for the README's example requests
sudo apt-get update -y
sudo apt-get install -y --no-install-recommends httpie

# Temporal CLI. Defaults to localhost:7233, which is the sidecar Temporal server.
curl -sSf https://temporal.download/cli.sh | sh
sudo ln -sf "$HOME/.temporalio/bin/temporal" /usr/local/bin/temporal

npm install --prefix downstream-services
(cd order-service && ./gradlew --no-daemon classes)
