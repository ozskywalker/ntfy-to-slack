FROM golang:1.22.3-alpine3.19 AS build-env
ADD . /src
RUN cd /src && go build -o ntfy-to-slack

FROM alpine
WORKDIR /app
COPY --from=build-env /src/ntfy-to-slack /app/
ENTRYPOINT ./ntfy-to-slack