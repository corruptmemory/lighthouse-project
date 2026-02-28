#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "Building prototype-2..."
go build -o bin/prototype-2 .
echo "Done: bin/prototype-2"
