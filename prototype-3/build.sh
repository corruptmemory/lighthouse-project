#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

HTMX_VERSION="2.0.4"
HTMX_FILE="static/vendor/htmx.min.js"

if [[ "${1:-}" == "--refresh-htmx" ]] || [[ ! -f "$HTMX_FILE" ]]; then
    echo "Downloading htmx ${HTMX_VERSION}..."
    curl -sL "https://unpkg.com/htmx.org@${HTMX_VERSION}/dist/htmx.min.js" -o "$HTMX_FILE"
    echo "Downloaded: $HTMX_FILE"
fi

mkdir -p widgets

echo "Building prototype-3..."
go build -o bin/prototype-3 .
echo "Done: bin/prototype-3"
