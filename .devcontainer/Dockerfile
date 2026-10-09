FROM mcr.microsoft.com/devcontainers/base:ubuntu

# Repo-independent tools live in the image so Docker caches them across rebuilds.
RUN apt-get update \
    && apt-get install -y --no-install-recommends httpie \
    && rm -rf /var/lib/apt/lists/*

# Temporal CLI (defaults to TEMPORAL_ADDRESS, set in docker-compose.yml)
RUN curl -sSf https://temporal.download/cli.sh | sh \
    && install -m 0755 /root/.temporalio/bin/temporal /usr/local/bin/temporal \
    && rm -rf /root/.temporalio

# uv runs the Python order service
RUN curl -LsSf https://astral.sh/uv/install.sh | UV_UNMANAGED_INSTALL=/usr/local/bin sh
