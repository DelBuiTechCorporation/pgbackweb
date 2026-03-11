# syntax=docker/dockerfile:1.7

FROM golang:1.25-trixie AS golang
FROM debian:trixie

LABEL org.opencontainers.image.source="https://github.com/eduardolat/pgbackweb"

ARG TARGETARCH
ARG TARGETPLATFORM

WORKDIR /app

ENV DEBIAN_FRONTEND="noninteractive"
ENV PATH="${PATH}:/usr/local/go/bin"

# Install Golang from official image
COPY --from=golang /usr/local/go /usr/local/go

# Validate supported architectures early.
RUN case "${TARGETARCH}" in \
        amd64|arm64) echo "Building for ${TARGETPLATFORM}" ;; \
        *) echo "Unsupported architecture: ${TARGETPLATFORM}" && exit 1 ;; \
    esac

# Install APT dependencies with cache mounts for faster rebuilds.
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    set -e && \
    apt update && apt install -y postgresql-common && \
    /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y && \
    apt update && apt install -y \
        git \
        npm \
        tzdata \
        unzip \
        wget \
        postgresql-client-13 \
        postgresql-client-14 \
        postgresql-client-15 \
        postgresql-client-16 \
        postgresql-client-17 \
        postgresql-client-18

# Install Node.js and arch-specific tools in a dedicated layer.
RUN --mount=type=cache,target=/root/.npm,sharing=locked \
    set -e && \
    npm install -g n@latest && \
    n 22.19.0 && \
    mkdir -p /tmp/downloads && cd /tmp/downloads && \
    case "${TARGETARCH}" in \
        arm64) \
            task_pkg="task_linux_arm64.tar.gz" && \
            goose_bin="goose_linux_arm64" && \
            sqlc_pkg="sqlc_1.30.0_linux_arm64.tar.gz" && \
            lint_pkg="golangci-lint-2.5.0-linux-arm64.tar.gz" && \
            lint_dir="golangci-lint-2.5.0-linux-arm64" ;; \
        amd64) \
            task_pkg="task_linux_amd64.tar.gz" && \
            goose_bin="goose_linux_x86_64" && \
            sqlc_pkg="sqlc_1.30.0_linux_amd64.tar.gz" && \
            lint_pkg="golangci-lint-2.5.0-linux-amd64.tar.gz" && \
            lint_dir="golangci-lint-2.5.0-linux-amd64" ;; \
    esac && \
    wget --no-verbose "https://github.com/go-task/task/releases/download/v3.45.4/${task_pkg}" && \
    tar -xzf "${task_pkg}" && \
    mv ./task /usr/local/bin/task && \
    wget --no-verbose "https://github.com/pressly/goose/releases/download/v3.25.0/${goose_bin}" && \
    mv "./${goose_bin}" /usr/local/bin/goose && \
    wget --no-verbose "https://github.com/sqlc-dev/sqlc/releases/download/v1.30.0/${sqlc_pkg}" && \
    tar -xzf "${sqlc_pkg}" && \
    mv ./sqlc /usr/local/bin/sqlc && \
    wget --no-verbose "https://github.com/golangci/golangci-lint/releases/download/v2.5.0/${lint_pkg}" && \
    tar -xzf "${lint_pkg}" && \
    mv "./${lint_dir}/golangci-lint" /usr/local/bin/golangci-lint && \
    chmod +x /usr/local/bin/task /usr/local/bin/goose /usr/local/bin/sqlc /usr/local/bin/golangci-lint && \
    git config --global --add safe.directory '*' && \
    rm -rf /tmp/downloads

# Keep compatibility with previous image behavior.
RUN mkdir -p /backups && chmod 777 /backups

# Copy and install Node dependencies first for better layer cache hit.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm,sharing=locked npm ci

# Copy and install Go dependencies first for better layer cache hit.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download

# Copy source code only after dependency layers.
COPY . .

# Build app with Go caches mounted for faster incremental builds.
RUN --mount=type=cache,target=/root/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    set -e && \
    task fixperms && \
    task build && \
    cp ./dist/change-password /usr/local/bin/change-password && \
    chmod +x /usr/local/bin/change-password

EXPOSE 8085
CMD ["task", "servem"]
