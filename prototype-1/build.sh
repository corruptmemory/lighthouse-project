#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "Building prototype-1..."
go build -o bin/prototype-1 .
echo "Done: bin/prototype-1"
