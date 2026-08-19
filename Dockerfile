FROM golang:1.26.6-alpine3.23 AS build-env
RUN apk add --no-cache git
WORKDIR /src

# Download modules in their own layer so editing source files doesn't bust
# the module cache on every rebuild.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo development) && \
    GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo unknown) && \
    GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "") && \
    BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    CGO_ENABLED=0 go build -o /src/ntfy-to-slack -v \
      -ldflags "-s -w \
        -X github.com/ozskywalker/ntfy-to-slack/internal/version.Version=${VERSION} \
        -X github.com/ozskywalker/ntfy-to-slack/internal/version.GitCommit=${GIT_COMMIT} \
        -X github.com/ozskywalker/ntfy-to-slack/internal/version.GitTag=${GIT_TAG} \
        -X github.com/ozskywalker/ntfy-to-slack/internal/version.BuildDate=${BUILD_DATE}" \
      ./cmd/ntfy-to-slack || (echo "Build failed" && exit 1)

FROM alpine:3.23
RUN adduser -D -u 10001 ntfy-to-slack
WORKDIR /app
COPY --from=build-env /src/ntfy-to-slack /app/
USER ntfy-to-slack

ENTRYPOINT ["/app/ntfy-to-slack"]
