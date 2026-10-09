#!/usr/bin/env bash
# Build (and optionally push) the prebuilt dev container image used by docker-compose.yml.
#
#   .devcontainer/build-image.sh 1            # build locally for this machine's platform
#   .devcontainer/build-image.sh 1 --push     # multi-arch build, push to ghcr.io
#
# Pushing needs `docker login ghcr.io` (PAT with write:packages) and a buildx builder with
# the docker-container driver (`docker buildx create --use`). After the first push, set the
# package to Public in GitHub so attendees can pull it without logging in.
set -euo pipefail

VERSION="${1:?usage: build-image.sh <version-tag> [--push]}"
IMAGE="ghcr.io/henryken/order-mgmt-devcontainer:${VERSION}"
cd "$(dirname "$0")/.."

args=(build --workspace-folder . --config .devcontainer/prebuild/devcontainer.json --image-name "$IMAGE")
if [[ "${2:-}" == "--push" ]]; then
  args+=(--platform linux/amd64,linux/arm64 --push true)
fi

npx --yes @devcontainers/cli "${args[@]}"
echo "Built $IMAGE"
