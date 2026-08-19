FROM golang:1.26.6-alpine3.23 AS build-env
ADD . /src
WORKDIR /src
RUN go build -o /src/ntfy-to-slack -v ./cmd/ntfy-to-slack || (echo "Build failed" && exit 1)

FROM alpine:3.23
WORKDIR /app
COPY --from=build-env /src/ntfy-to-slack /app/

ENTRYPOINT ["/app/ntfy-to-slack"]