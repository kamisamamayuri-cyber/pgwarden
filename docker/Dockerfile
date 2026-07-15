FROM golang:1.25-trixie AS golang
FROM debian:trixie AS backend
LABEL org.opencontainers.image.source="https://github.com/kamisamamayuri-cyber/pgwarden"
# Declare ARG to make it available in the RUN commands
ARG TARGETPLATFORM
RUN echo "Building for ${TARGETPLATFORM}"
RUN if [ "${TARGETPLATFORM}" != "linux/amd64" ] && [ "${TARGETPLATFORM}" != "linux/arm64" ]; then \
        echo "Unsupported architecture: ${TARGETPLATFORM}" && \
        exit 1; \
    fi

WORKDIR /app
ENV DEBIAN_FRONTEND="noninteractive"
ENV GOBIN="/usr/local/go/bin"
ENV PATH="$PATH:/usr/local/go/bin"

# Install golang from docker image
COPY --from=golang /usr/local/go /usr/local/go

# Tool versions
ARG TASK_VERSION=3.40.0
ARG GOOSE_VERSION=3.25.0
ARG SQLC_VERSION=1.30.0
ARG GOLANGCI_VERSION=2.5.0

# Install system dependencies and prepare system
RUN set -e && \
    # Create a temp dir for downloads
    mkdir /tmp/downloads && cd /tmp/downloads && \

    # Add PostgreSQL APT repository
    # https://www.postgresql.org/download/linux/debian/
    apt update && apt install -y postgresql-common && \
    /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y && \

    # Install APT packages
    apt update && apt install -y \
        wget unzip tzdata git npm \
        postgresql-client-13 postgresql-client-14 \
        postgresql-client-15 postgresql-client-16 \
        postgresql-client-17 postgresql-client-18 && \

    # Install NodeJS
    apt install -y npm && \
    npm install -g n@latest && \
    n 22.19.0 && \

    # Install downloadable binaries
    if [ "${TARGETPLATFORM}" = "linux/arm64" ]; then \
        echo "Downloading arm64 binaries" && \
        # Install task
        wget --no-verbose https://github.com/go-task/task/releases/download/v${TASK_VERSION}/task_linux_arm64.tar.gz && \
        tar -xzf task_linux_arm64.tar.gz && \
        mv ./task /usr/local/bin/task && \
        # Install goose
        wget --no-verbose https://github.com/pressly/goose/releases/download/v${GOOSE_VERSION}/goose_linux_arm64 && \
        mv ./goose_linux_arm64 /usr/local/bin/goose && \
        # Install sqlc
        wget --no-verbose https://github.com/sqlc-dev/sqlc/releases/download/v${SQLC_VERSION}/sqlc_${SQLC_VERSION}_linux_arm64.tar.gz && \
        tar -xzf sqlc_${SQLC_VERSION}_linux_arm64.tar.gz && \
        mv ./sqlc /usr/local/bin/sqlc && \
        # Install golangci-lint
        wget --no-verbose https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_VERSION}/golangci-lint-${GOLANGCI_VERSION}-linux-arm64.tar.gz && \
        tar -xzf golangci-lint-${GOLANGCI_VERSION}-linux-arm64.tar.gz && \
        mv ./golangci-lint-${GOLANGCI_VERSION}-linux-arm64/golangci-lint /usr/local/bin/golangci-lint; \
    else \
        echo "Downloading amd64 binaries" && \
        # Install task
        wget --no-verbose https://github.com/go-task/task/releases/download/v${TASK_VERSION}/task_linux_amd64.tar.gz && \
        tar -xzf task_linux_amd64.tar.gz && \
        mv ./task /usr/local/bin/task && \
        # Install goose
        wget --no-verbose https://github.com/pressly/goose/releases/download/v${GOOSE_VERSION}/goose_linux_x86_64 && \
        mv ./goose_linux_x86_64 /usr/local/bin/goose && \
        # Install sqlc
        wget --no-verbose https://github.com/sqlc-dev/sqlc/releases/download/v${SQLC_VERSION}/sqlc_${SQLC_VERSION}_linux_amd64.tar.gz && \
        tar -xzf sqlc_${SQLC_VERSION}_linux_amd64.tar.gz && \
        mv ./sqlc /usr/local/bin/sqlc && \
        # Install golangci-lint
        wget --no-verbose https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_VERSION}/golangci-lint-${GOLANGCI_VERSION}-linux-amd64.tar.gz && \
        tar -xzf golangci-lint-${GOLANGCI_VERSION}-linux-amd64.tar.gz && \
        mv ./golangci-lint-${GOLANGCI_VERSION}-linux-amd64/golangci-lint /usr/local/bin/golangci-lint; \
    fi && \

    # Make downloaded binaries executable
    chmod +x /usr/local/bin/* && \

    # Default git config
    # https://github.com/golangci/golangci-lint/issues/4033
    git config --global --add safe.directory '*' && \

    # Clean up downloads and APT cache
    mkdir -p /app && cd /app && rm -rf /tmp/downloads && \
    rm -rf /var/lib/apt/lists/* && \

    # Create backups dir
    mkdir /backups && \
    chmod 777 /backups

##############
# START HERE #
##############

# Copy and install node dependencies
COPY package.json .
COPY package-lock.json .
RUN npm install

# Copy and install go dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the rest of the files
COPY . .

# Build the app
RUN set -e && \
    task fixperms && \
    task build && \
    cp ./dist/change-password /usr/local/bin/change-password && \
    cp ./dist/discover /usr/local/bin/pgwarden-discover && \
    chmod +x /usr/local/bin/change-password /usr/local/bin/pgwarden-discover

# Run the app
EXPOSE 8085
CMD ["task", "servem"]
