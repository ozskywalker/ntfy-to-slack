FROM golang:1.24.2 AS build-env
ADD . /src
RUN cd /src && go build -o ntfy-to-slack

FROM alpine
WORKDIR /app
COPY --from=build-env /src/ntfy-to-slack /app/
ENTRYPOINT ./ntfy-to-slack