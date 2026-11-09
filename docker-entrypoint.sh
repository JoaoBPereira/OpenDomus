#!/bin/sh
# Writes the runtime config.json of both web apps from the container environment, then starts
# the backend and nginx. Lets the same image point at any API URL without rebuilding.
set -eu

html=/usr/share/nginx/html

if [ -n "${BACKEND_SERVER_URL:-}" ]; then
  printf '{\n  "serverURL": "%s",\n  "priceKwh": %s\n}\n' "$BACKEND_SERVER_URL" "${PRICE_KWH:-0}" > "$html/frontend/config.json"
  printf '{\n  "serverUrl": "%s"\n}\n' "$BACKEND_SERVER_URL" > "$html/bo/config.json"
  echo "config.json written for $BACKEND_SERVER_URL"
fi

./od &
exec nginx -g 'daemon off;'
