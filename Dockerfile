FROM golang:1.11.5-alpine AS build

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY . /app

RUN go build ./cmd/edgemax_exporter

FROM alpine:3.8

COPY --from=build /app/edgemax_exporter /usr/local/bin/edgemax_exporter

ENTRYPOINT ["/usr/local/bin/edgemax_exporter"]
