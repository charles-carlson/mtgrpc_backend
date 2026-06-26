#!/bin/bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$REPO_ROOT"

protoc \
  --go_out=. \
  --go_opt=module=backend_nonsense \
  --go-grpc_out=. \
  --go-grpc_opt=module=backend_nonsense \
  proto/cards.proto

echo "proto generated successfully"
