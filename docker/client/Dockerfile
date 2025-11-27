# syntax=docker/dockerfile:1

# Builder stage compiles the capture client using the gocv toolchain.
FROM gocv/opencv:latest AS builder

WORKDIR /src

COPY client/capture-client/go.mod client/capture-client/go.sum ./
RUN go mod download

COPY client/capture-client/ ./

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o capture-client .

# Runtime stage ships only the compiled binary and runtime assets.
FROM gocv/opencv:latest

WORKDIR /app

ENV CONFIG_DIR=/config \
    CONFIG_FILE=/config/config.json

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Create runtime directories and copy binaries/assets.
RUN mkdir -p /app/config

COPY --from=builder /src/capture-client /usr/local/bin/capture-client
COPY client/capture-client/config.example.json /app/config/config.example.json
COPY scripts/configure-client.sh /usr/local/bin/configure-client.sh
COPY docker/client/entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chmod +x /usr/local/bin/capture-client \
    /usr/local/bin/configure-client.sh \
    /usr/local/bin/entrypoint.sh

# Persist configuration, logs, and temporary buffers outside the container lifecycle.
VOLUME ["/config"]

# Default device path can be overridden at runtime if needed.
ENV CAMERA_DEVICE=/dev/video0

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD []
