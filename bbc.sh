#!/usr/bin/env bash
#
# bbc.sh — manage the bbc-pipeline-ui container lifecycle.
#
# Usage: ./bbc.sh <command>
#
#   build    Build the Docker image (Nix builds the Go binary + webapp inside)
#   start    Build the image if missing, then run the container (webapp mode)
#   stop     Stop and remove the container
#   restart  Stop then start
#   status   Show container state (+ best-effort health check)
#   logs     Follow container logs
#   help     Show this help
#
# Configuration (all overridable via environment variables):
#   BBC_IMAGE        image name            (default: bbc-pipeline-ui)
#   BBC_CONTAINER    container name        (default: bbc-pipeline-ui)
#   BBC_CONFIG_FILE  host config path      (default: ./config.yml)
#   BBC_PORT         host port to publish  (default: 8080)
#
# The host config file is mounted at /config.yml inside the container.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

IMAGE_NAME="${BBC_IMAGE:-bbc-pipeline-ui}"
CONTAINER_NAME="${BBC_CONTAINER:-bbc-pipeline-ui}"
CONFIG_FILE="${BBC_CONFIG_FILE:-${SCRIPT_DIR}/config.yml}"
PORT="${BBC_PORT:-8080}"
RESTART_POLICY="${BBC_RESTART:-unless-stopped}"

log() { printf '\033[1;34m[bbc]\033[0m %s\n' "$*"; }
err() { printf '\033[1;31m[bbc]\033[0m %s\n' "$*" >&2; }

image_exists()     { docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; }
container_exists() { docker container inspect "$CONTAINER_NAME" >/dev/null 2>&1; }

container_state() {
  docker inspect --format '{{.State.Status}}' "$CONTAINER_NAME" 2>/dev/null || true
}

build() {
  log "Building image '${IMAGE_NAME}' (Nix builds the Go binary + webapp)…"
  docker build -t "$IMAGE_NAME" "$SCRIPT_DIR"
  log "Image '${IMAGE_NAME}' built."
}

ensure_config() {
  while [[ ! -f "$CONFIG_FILE" ]]; do
    err "Config file not found: ${CONFIG_FILE}"
    echo
    echo "  1) Run the first-time setup wizard"
    echo "  2) Set BBC_CONFIG_FILE to another path"
    echo "  q) Abort"
    echo
    read -r -p "Choose an option [1/2/q]: " choice || choice=""
    case "$choice" in
      1) run_wizard ;;
      2)
        read -r -p "Config file path: " new_path || new_path=""
        if [[ -n "$new_path" ]]; then
          CONFIG_FILE="$new_path"
        fi
        ;;
      q|Q|"")
        err "Aborted."
        exit 1
        ;;
      *)
        err "Unknown option: ${choice}"
        ;;
    esac
  done
}

# Run the app's interactive first-run wizard inside a throwaway container,
# writing the resulting config to the host path in CONFIG_FILE.
run_wizard() {
  if ! image_exists; then
    log "Image not found — building it now."
    build
  fi

  local cfg_dir cfg_base cfg_dir_abs
  cfg_dir="$(dirname "$CONFIG_FILE")"
  cfg_base="$(basename "$CONFIG_FILE")"
  mkdir -p "$cfg_dir"
  cfg_dir_abs="$(cd "$cfg_dir" && pwd)"

  log "Starting the interactive setup wizard…"
  log "Config will be written to: ${cfg_dir_abs}/${cfg_base}"
  log "After the wizard completes, quit the TUI with 'q'."

  # Success is measured by whether the config file is created below, not the
  # container exit code (the user quits the TUI, possibly with ctrl+c).
  docker run --rm -it \
    --user "$(id -u):$(id -g)" \
    -v "${cfg_dir_abs}:/cfg" \
    -e "BBC_CONFIG=/cfg/${cfg_base}" \
    "$IMAGE_NAME" || true

  if [[ -f "$CONFIG_FILE" ]]; then
    log "Configuration saved to ${CONFIG_FILE}."
  else
    err "Wizard finished but ${CONFIG_FILE} was not created."
  fi
}

start() {
  ensure_config

  if ! image_exists; then
    log "Image not found — building it now."
    build
  fi

  if [[ "$(container_state)" == "running" ]]; then
    log "Container '${CONTAINER_NAME}' is already running."
    return 0
  fi

  if container_exists; then
    log "Removing stopped container '${CONTAINER_NAME}'…"
    docker rm -f "$CONTAINER_NAME" >/dev/null
  fi

  log "Starting container '${CONTAINER_NAME}' (webapp on http://localhost:${PORT})…"
  docker run -d \
    --name "$CONTAINER_NAME" \
    --restart "$RESTART_POLICY" \
    -p "${PORT}:8080" \
    -v "$CONFIG_FILE:/config.yml" \
    "$IMAGE_NAME" \
    --webapp "0.0.0.0:8080"

  log "Started. Webapp: http://localhost:${PORT}  (config: ${CONFIG_FILE})"
}

stop() {
  if ! container_exists; then
    log "Container '${CONTAINER_NAME}' does not exist."
    return 0
  fi
  log "Stopping container '${CONTAINER_NAME}'…"
  docker stop "$CONTAINER_NAME" >/dev/null
  docker rm "$CONTAINER_NAME" >/dev/null
  log "Stopped and removed."
}

restart() {
  stop
  start
}

status() {
  if ! container_exists; then
    log "Container '${CONTAINER_NAME}' does not exist (not created)."
    return 0
  fi

  local state
  state="$(container_state)"
  printf '\033[1;34m[bbc]\033[0m Container: %s\n' "$CONTAINER_NAME"
  docker ps -a --filter "name=^/${CONTAINER_NAME}$" --format '  state: {{.Status}}\n  ports: {{.Ports}}\n  image: {{.Image}}'

  if [[ "$state" == "running" ]] && command -v curl >/dev/null 2>&1; then
    local health
    health="$(curl -sf "http://localhost:${PORT}/api/health" 2>/dev/null || true)"
    if [[ -n "$health" ]]; then
      log "health: ${health}"
    fi
  fi
}

logs() {
  if ! container_exists; then
    err "Container '${CONTAINER_NAME}' does not exist."
    exit 1
  fi
  docker logs -f "$CONTAINER_NAME"
}

usage() {
  sed -n '2,21p' "$0" | sed 's/^# \{0,1\}//'
}

case "${1:-help}" in
  build)   build ;;
  start)   start ;;
  stop)    stop ;;
  restart) restart ;;
  status)  status ;;
  logs)    logs ;;
  help|-h|--help) usage ;;
  *)
    err "Unknown command: ${1:-}"
    usage
    exit 1
    ;;
esac
