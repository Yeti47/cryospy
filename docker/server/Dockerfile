# syntax=docker/dockerfile:1.6

##############################
# Build stage
##############################
FROM golang:1.24 AS builder

WORKDIR /src

# Copy server sources (core, capture-server, dashboard)
COPY server /src/server

# Prepare output directory for compiled binaries
RUN mkdir -p /out

# Pre-fetch dependencies for each module
RUN cd server/core && go mod download
RUN cd server/capture-server && go mod download
RUN cd server/dashboard && go mod download

# Build binaries
RUN cd server/capture-server \
    && CGO_ENABLED=1 go build -o /out/capture-server .
RUN cd server/dashboard \
    && CGO_ENABLED=1 go build -o /out/dashboard .

##############################
# Runtime stage
##############################
FROM debian:bookworm-slim AS runtime

ENV CRYOSPY_HOME=/home/cryospy \
    HOME=/home/cryospy \
    GIN_MODE=release

# Install runtime dependencies (ffmpeg for video processing, tini for proper signal handling)
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ffmpeg \
        ca-certificates \
        tini \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd --create-home --shell /usr/sbin/nologin cryospy

WORKDIR /opt/cryospy

# Copy binaries
COPY --from=builder /out/capture-server /usr/local/bin/capture-server
COPY --from=builder /out/dashboard /usr/local/bin/dashboard

# Copy dashboard web assets
COPY server/dashboard/web /opt/cryospy/web

# Copy entrypoint script
COPY docker/server/cryospy-entrypoint.sh /usr/local/bin/cryospy-entrypoint.sh
RUN chmod +x /usr/local/bin/cryospy-entrypoint.sh

# Ensure data directory is exposed for persistence
VOLUME ["/home/cryospy"]

# Expose dashboard and capture-server ports
EXPOSE 8080 8081

USER cryospy

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/cryospy-entrypoint.sh"]
CMD []
