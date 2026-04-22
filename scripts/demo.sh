#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

IMAGE_NAME="demo-cache"
IMAGE_TAG="latest"
IMAGE_REF="${IMAGE_NAME}:${IMAGE_TAG}"
CONTEXT_DIR="$REPO_ROOT/examples/simple-app"
TEMP_FILE="$CONTEXT_DIR/demo-temp.txt"

log() {
  printf '\n%s\n' "=== $1 ==="
}

section_header() {
  printf '\n%s\n' "╔════════════════════════════════════════════════════════╗"
  printf '%s\n' "║ $1"
  printf '%s\n' "╚════════════════════════════════════════════════════════╝"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$1" >&2
    exit 1
  fi
}

cleanup() {
  rm -f "$TEMP_FILE"
}

trap cleanup EXIT

require_cmd go

if [[ ! -d "$CONTEXT_DIR" ]]; then
  printf 'Example context not found: %s\n' "$CONTEXT_DIR" >&2
  exit 1
fi

cd "$REPO_ROOT"

log "Building Docksmith binary"
go build -o docksmith .

section_header "DEMO: Cache Hit vs Cache Miss with Performance Measurement"

log "Environment"
printf 'Go version: '
go version
printf 'Working directory: %s\n' "$REPO_ROOT"
printf 'Image reference: %s\n' "$IMAGE_REF"

# Clean up old images first
if ./docksmith images 2>/dev/null | grep -q "$IMAGE_NAME"; then
  log "Removing existing image from previous runs"
  ./docksmith rmi "$IMAGE_REF" || true
fi

section_header "PHASE 1: FIRST BUILD (all steps execute, CACHE MISS)"

log "Build #1 - Creating image (all steps run)"
START=$(date +%s.%N)
./docksmith build -t "$IMAGE_REF" "$CONTEXT_DIR"
END=$(date +%s.%N)
TIME1=$(echo "$END - $START" | bc)
printf 'Build #1 elapsed time: %.3f seconds\n' "$TIME1"

log "Listing images"
./docksmith images

section_header "PHASE 2: REBUILD UNCHANGED (same inputs, CACHE HIT)"

log "Build #2 - Rebuilding with no changes (expect cache hits)"
START=$(date +%s.%N)
./docksmith build -t "$IMAGE_REF" "$CONTEXT_DIR"
END=$(date +%s.%N)
TIME2=$(echo "$END - $START" | bc)
printf 'Build #2 elapsed time: %.3f seconds\n' "$TIME2"

SPEEDUP=$(echo "scale=2; $TIME1 / $TIME2" | bc)
printf 'Speedup (Build #1 / Build #2): %.2fx\n' "$SPEEDUP"

section_header "PHASE 3: TRIGGER CACHE MISS (modify source file)"

log "Modifying a file in the build context"
echo "Modified on $(date)" >> "$TEMP_FILE"
printf 'Created temp file: %s\n' "$TEMP_FILE"

log "Build #3 - Rebuilding after source change (expect cache miss)"
START=$(date +%s.%N)
./docksmith build -t "$IMAGE_REF" "$CONTEXT_DIR"
END=$(date +%s.%N)
TIME3=$(echo "$END - $START" | bc)
printf 'Build #3 elapsed time: %.3f seconds\n' "$TIME3"

section_header "PHASE 4: REBUILD AGAIN (cache restored, CACHE HIT)"

log "Build #4 - Rebuilding with restored state (expect cache hits again)"
START=$(date +%s.%N)
./docksmith build -t "$IMAGE_REF" "$CONTEXT_DIR"
END=$(date +%s.%N)
TIME4=$(echo "$END - $START" | bc)
printf 'Build #4 elapsed time: %.3f seconds\n' "$TIME4"

section_header "PHASE 5: RUNTIME DEMO"

log "Running container with default CMD"
./docksmith run "$IMAGE_REF"

log "Running container with override command"
./docksmith run "$IMAGE_REF" sh -c 'echo "Override: Custom command execution"'

section_header "PHASE 6: CLEANUP"

log "Removing image"
./docksmith rmi "$IMAGE_REF"

log "Final image list"
./docksmith images

section_header "SUMMARY"
printf '\n'
printf 'Performance Results:\n'
printf '  Build #1 (cache miss, first run):     %.3f seconds\n' "$TIME1"
printf '  Build #2 (cache hit, no changes):     %.3f seconds\n' "$TIME2"
printf '  Build #3 (cache miss, file changed):  %.3f seconds\n' "$TIME3"
printf '  Build #4 (cache hit, restored):       %.3f seconds\n' "$TIME4"
printf '\n'
printf 'Cache Performance:\n'
printf '  First cache hit speedup (Build #1 vs #2): %.2fx faster\n' "$SPEEDUP"
printf '  Second cache hit speedup (Build #3 vs #4): %.2fx faster\n' "$(echo "scale=2; $TIME3 / $TIME4" | bc)"
printf '\n'
printf 'Key observations:\n'
printf '  - Build #1 & #3 show "CACHE MISS" messages in output (full execution)\n'
printf '  - Build #2 & #4 show "CACHE HIT" messages (skipped steps)\n'
printf '  - Timing clearly shows speedup from cache reuse\n'
printf '\n'

log "Demo complete - ready to present to professor"