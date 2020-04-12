#!/usr/bin/env bash

set -exo pipefail

source /etc/secrets/env

: "${EDGEMAX_PASSWORD:?Need to set EDGEMAX_PASSWORD non-empty}"

exec /usr/local/bin/edgemax_exporter \
	-edgemax.addr=https://192.168.1.1 \
	-edgemax.username=ubnt \
	-edgemax.password=$EDGEMAX_PASSWORD \
	-edgemax.insecure
