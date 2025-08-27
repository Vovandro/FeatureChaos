#!/usr/bin/env bash
set -euo pipefail

# Generate protobufs from the main proto for:
# - Server application (Go) under src/controller/FeatureChaos
# - Go SDK under sdk/fc_sdk_go/pb
# - PHP SDK under sdk/fc_sdk_php/src (if plugins available)
# - Python SDK under sdk/fc_sdk_py/featurechaos/pb (if grpc_tools available)

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC_PROTO="$REPO_ROOT/proto/FeatureChaos.proto"

if [[ ! -f "$SRC_PROTO" ]]; then
  echo "[ERR] Main proto not found: $SRC_PROTO" >&2
  exit 1
fi

echo "[INFO] Using main proto: $SRC_PROTO"

require_bin() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[ERR] Required tool not found: $1" >&2
    exit 1
  fi
}

# Try to ensure protoc and plugins/tools are installed
ensure_dependencies() {
  # Prefer adding common Go bin dirs to PATH
  export GOBIN="${GOBIN:-}"
  if [[ -z "${GOBIN}" ]]; then
    # derive from GOPATH or default
    if command -v go >/dev/null 2>&1; then
      GOBIN="$(go env GOPATH 2>/dev/null)/bin"
    else
      GOBIN="$HOME/go/bin"
    fi
  fi
  export PATH="$GOBIN:$PATH"

  # protoc (install via Homebrew on macOS)
  if ! command -v protoc >/dev/null 2>&1; then
    if [[ "$(uname -s)" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
      echo "[INFO] Installing protoc via Homebrew..."
      brew install protobuf >/dev/null
    fi
  fi
  if ! command -v protoc >/dev/null 2>&1; then
    echo "[ERR] protoc is required. Install it manually (e.g., brew install protobuf) and rerun." >&2
    exit 1
  fi

  # Go plugins
  if command -v go >/dev/null 2>&1; then
    local PGO_VER="v1.34.2"
    local PGRPC_VER="v1.5.1"
    echo "[INFO] Installing/updating Go protoc plugins ($PGO_VER, $PGRPC_VER)..."
    GOFLAGS="${GOFLAGS:-}" GO111MODULE=on go install google.golang.org/protobuf/cmd/protoc-gen-go@"$PGO_VER" >/dev/null
    GOFLAGS="${GOFLAGS:-}" GO111MODULE=on go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@"$PGRPC_VER" >/dev/null
  else
    echo "[WARN] 'go' is not available; skipping Go plugin installation"
  fi

  # PHP grpc_php_plugin (install via Homebrew on macOS)
  if ! command -v grpc_php_plugin >/dev/null 2>&1; then
    if [[ "$(uname -s)" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
      echo "[INFO] Installing grpc (php plugin) via Homebrew..."
      brew install grpc >/dev/null || true
    fi
  fi
  if ! command -v grpc_php_plugin >/dev/null 2>&1; then
    echo "[ERR] grpc_php_plugin is required for PHP gRPC stubs. Install grpc (e.g., brew install grpc) and rerun." >&2
    exit 1
  fi

  # Python grpc_tools (mandatory)
  if ! command -v python3 >/dev/null 2>&1; then
    echo "[ERR] python3 is required for Python SDK generation." >&2
    exit 1
  fi
  if ! python3 -c 'import grpc_tools.protoc' >/dev/null 2>&1; then
    if python3 -m pip --version >/dev/null 2>&1; then
      echo "[INFO] Installing Python grpcio-tools..."
      python3 -m pip install --user grpcio-tools >/dev/null || true
    fi
  fi
  if ! python3 -c 'import grpc_tools.protoc' >/dev/null 2>&1; then
    echo "[ERR] python grpc_tools.protoc not available. Install via: python3 -m pip install --user grpcio-tools" >&2
    exit 1
  fi
}

gen_go_app() {
  echo "[INFO] Generating Go server protobufs..."
  require_bin protoc
  require_bin protoc-gen-go
  require_bin protoc-gen-go-grpc

  out_dir="$REPO_ROOT/src/controller/FeatureChaos"
  mkdir -p "$out_dir"

  protoc -I "$REPO_ROOT/proto" \
    --go_out=paths=source_relative:"$out_dir" \
    --go-grpc_out=paths=source_relative:"$out_dir" \
    "$SRC_PROTO"
}

gen_go_sdk() {
  echo "[INFO] Generating Go SDK protobufs..."
  require_bin protoc
  require_bin protoc-gen-go
  require_bin protoc-gen-go-grpc

  sdk_proto_dir="$REPO_ROOT/sdk/fc_sdk_go/pb"
  mkdir -p "$sdk_proto_dir"

  # Create SDK-specific proto with proper go_package
  sdk_proto="$sdk_proto_dir/FeatureChaos.proto"
  go_pkg='option go_package = "gitlab.com/devpro_studio/FeatureChaos/sdk/fc_sdk_go/pb;pb";'
  awk -v pkg="$go_pkg" '
    BEGIN{replaced=0}
    /^option[[:space:]]+go_package/ {print pkg; replaced=1; next}
    {print}
    END{if(replaced==0){print pkg}}
  ' "$SRC_PROTO" > "$sdk_proto"

  protoc -I "$sdk_proto_dir" \
    --go_out=paths=source_relative:"$sdk_proto_dir" \
    --go-grpc_out=paths=source_relative,require_unimplemented_servers=false:"$sdk_proto_dir" \
    "$sdk_proto"
}

gen_php_sdk() {
  echo "[INFO] Generating PHP SDK protobufs..."
  out_dir="$REPO_ROOT/sdk/fc_sdk_php/src"
  mkdir -p "$out_dir"
  require_bin protoc
  require_bin grpc_php_plugin
  protoc -I "$REPO_ROOT/proto" --php_out="$out_dir" "$SRC_PROTO"
  protoc -I "$REPO_ROOT/proto" --grpc_out="$out_dir" --plugin=protoc-gen-grpc="$(command -v grpc_php_plugin)" "$SRC_PROTO"
}

gen_py_sdk() {
  echo "[INFO] Generating Python SDK protobufs..."
  out_dir="$REPO_ROOT/sdk/fc_sdk_py/featurechaos/pb"
  mkdir -p "$out_dir"
  # ensure package importability
  [[ -f "$out_dir/__init__.py" ]] || touch "$out_dir/__init__.py"
  python3 -m grpc_tools.protoc -I "$REPO_ROOT/proto" \
    --python_out="$out_dir" \
    --grpc_python_out="$out_dir" \
    "$SRC_PROTO"
}

ensure_dependencies
gen_go_app
gen_go_sdk
gen_php_sdk
gen_py_sdk

echo "[DONE] Protobuf generation completed."
