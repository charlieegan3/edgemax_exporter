#!/usr/bin/env bash

set -exo pipefail

: "${ENV_PATH:?Need to set ENV_PATH non-empty}"

source $ENV_PATH

: "${EDGEMAX_PASSWORD:?Need to set EDGEMAX_PASSWORD non-empty}"

exec /usr/local/bin/edgemax_exporter \
	-edgemax.addr=https://192.168.1.1 \
	-edgemax.username=ubnt \
	-edgemax.password=$EDGEMAX_PASSWORD \
	-edgemax.insecure
