FROM golang:1.11.5-alpine AS build

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY . /app

ARG go_arch
RUN GOARCH=${go_arch} go build ./cmd/edgemax_exporter

# arm v7 3.11
FROM alpine@sha256:c5ea49127cd44d0f50eafda229a056bb83b6e691883c56fd863d42675fae3909

COPY --from=build /app/edgemax_exporter /usr/local/bin/edgemax_exporter
COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
