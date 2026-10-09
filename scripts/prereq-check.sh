#!/usr/bin/env bash
#
# Pre-class check for the Order Management workshop (devcontainer setup).
# Self-contained: does not need the workshop repository.
# Run on your HOST laptop (macOS, Linux, or WSL), not inside a container.
#
#   curl -fsSL <URL> | bash                    # check tools, network, ports
#   curl -fsSL <URL> | bash -s -- --warm       # also pre-pull the Docker images (do this at home)
#
# Exit code is non-zero if any REQUIRED check fails.

# Everything lives in main() and is invoked on the last line, so a truncated
# download can never execute half a script.
main() {
  set -u

  # Keep in sync with .devcontainer/docker-compose.yml
  local IMAGES="
temporalio/temporal:1.8.3
ghcr.io/shopify/toxiproxy:2.12.0
ghcr.io/henryken/toxihub:latest
mcr.microsoft.com/devcontainers/base:ubuntu
"
  local MIN_DOCKER_MEM_GB=4
  local MIN_FREE_DISK_GB=10
  # Only the ports the devcontainer forwards to the host (see forwardPorts in devcontainer.json)
  local PORTS="8080 7233 8233 7072 8474"
  local HOSTS="registry-1.docker.io ghcr.io mcr.microsoft.com services.gradle.org repo.maven.apache.org registry.npmjs.org temporal.download"

  local WARM=0 arg
  for arg in "$@"; do
    case "$arg" in
      --warm) WARM=1 ;;
      -h|--help) echo "Usage: bash prereq-check.sh [--warm]"; return 0 ;;
      *) echo "Unknown option: $arg (try --help)"; return 2 ;;
    esac
  done

  local GREEN="" YELLOW="" RED="" BOLD="" RESET=""
  if [ -t 1 ]; then
    GREEN=$'\033[32m'; YELLOW=$'\033[33m'; RED=$'\033[31m'; BOLD=$'\033[1m'; RESET=$'\033[0m'
  fi

  FAILS=0
  WARNS=0
  pass() { echo "  ${GREEN}[ OK ]${RESET} $1"; }
  warn() { echo "  ${YELLOW}[WARN]${RESET} $1"; [ -n "${2:-}" ] && echo "         -> $2"; WARNS=$((WARNS + 1)); }
  fail() { echo "  ${RED}[FAIL]${RESET} $1"; [ -n "${2:-}" ] && echo "         -> $2"; FAILS=$((FAILS + 1)); }
  section() { echo; echo "${BOLD}$1${RESET}"; }
  have() { command -v "$1" >/dev/null 2>&1; }

  local OS INSTALL_DOCKER
  OS="$(uname -s)"
  case "$OS" in
    Darwin) INSTALL_DOCKER="Install Docker Desktop: https://www.docker.com/products/docker-desktop/ (or 'brew install --cask docker')" ;;
    *)      INSTALL_DOCKER="Install Docker: https://docs.docker.com/get-docker/ (on Windows use Docker Desktop with the WSL2 backend)" ;;
  esac

  echo "${BOLD}Order Management workshop: pre-class check${RESET}"
  echo "Host: $OS $(uname -m)"

  # ------------------------------------------------------------------------
  section "1. Core tools"

  if have git; then
    pass "git $(git --version | awk '{print $3}')"
  else
    fail "git not found" "Install from https://git-scm.com/downloads"
  fi

  local DOCKER_OK=0
  if ! have docker; then
    fail "docker not found" "$INSTALL_DOCKER"
  elif ! docker info >/dev/null 2>&1; then
    fail "docker is installed but the daemon is not running" "Start Docker Desktop, wait until it says 'running', then re-run this check"
  else
    DOCKER_OK=1
    pass "docker $(docker version --format '{{.Server.Version}}' 2>/dev/null) (daemon running)"
  fi

  if [ "$DOCKER_OK" = 1 ]; then
    if docker compose version >/dev/null 2>&1; then
      pass "docker compose $(docker compose version --short 2>/dev/null)"
    elif have docker-compose; then
      warn "only legacy 'docker-compose' found" "Update Docker so the 'docker compose' plugin is available (the devcontainer needs it)"
    else
      fail "docker compose not found" "Update Docker Desktop to a recent version"
    fi
  fi

  # ------------------------------------------------------------------------
  section "2. Editor (devcontainer support)"

  if have code; then
    if code --list-extensions 2>/dev/null | grep -qi '^ms-vscode-remote.remote-containers$'; then
      pass "VS Code with the Dev Containers extension"
    else
      fail "VS Code found, but the Dev Containers extension is missing" \
           "Run: code --install-extension ms-vscode-remote.remote-containers"
    fi
  else
    warn "'code' command not found on PATH" \
         "Install VS Code (https://code.visualstudio.com) and, in VS Code, run 'Shell Command: Install code command in PATH'. Ignore this if your instructor told you to use IntelliJ."
  fi

  # ------------------------------------------------------------------------
  section "3. Docker resources"

  if [ "$DOCKER_OK" = 1 ]; then
    local MEM_BYTES MEM_GB FREE_KB FREE_GB
    MEM_BYTES="$(docker info --format '{{.MemTotal}}' 2>/dev/null || echo 0)"
    MEM_GB=$(( MEM_BYTES / 1024 / 1024 / 1024 ))
    # Docker reports slightly under the configured amount, so allow a 0.5 GB margin
    if [ "$MEM_BYTES" -ge $(( (MIN_DOCKER_MEM_GB * 1024 - 512) * 1024 * 1024 )) ]; then
      pass "Docker memory: ~${MEM_GB} GB"
    else
      fail "Docker memory is ~${MEM_GB} GB (need at least ${MIN_DOCKER_MEM_GB} GB)" \
           "Docker Desktop > Settings > Resources > Memory"
    fi

    FREE_KB="$(df -Pk "$HOME" | awk 'NR==2 {print $4}')"
    FREE_GB=$(( FREE_KB / 1024 / 1024 ))
    if [ "$FREE_GB" -ge "$MIN_FREE_DISK_GB" ]; then
      pass "Free disk space: ~${FREE_GB} GB"
    else
      fail "Only ~${FREE_GB} GB free disk (need at least ${MIN_FREE_DISK_GB} GB)" "Free up space; images and build caches are large"
    fi
  else
    warn "Skipped (Docker not available)"
  fi

  # ------------------------------------------------------------------------
  section "4. Network access (registries and package hosts)"

  local h
  if have curl; then
    for h in $HOSTS; do
      # Any HTTP response (even 401/404) proves we can reach the host; only a connection failure is a problem.
      if curl -s -o /dev/null --connect-timeout 5 --max-time 10 "https://$h/"; then
        pass "$h"
      else
        fail "cannot reach $h" "Check Wi-Fi/VPN/proxy. Corporate networks often block ghcr.io and registry hosts."
      fi
    done
  else
    warn "curl not found, skipping network checks"
  fi

  # ------------------------------------------------------------------------
  section "5. Local ports (forwarded from the container to your laptop)"

  # Prints "<pid> <command>" of the process listening on port $1, or nothing if free.
  # Returns 2 if no tool is available to tell.
  port_owner() {
    if have lsof; then
      lsof -nP -iTCP:"$1" -sTCP:LISTEN -Fpc +c 0 2>/dev/null | awk '
        /^p/ { pid = substr($0, 2) }
        /^c/ { print pid, substr($0, 2); exit }'
    elif have nc; then
      nc -z 127.0.0.1 "$1" >/dev/null 2>&1 && echo "? unknown"
    else
      return 2
    fi
  }

  # Editors/Docker forward container ports to the host themselves, so seeing them
  # on a port means a dev container is probably already running, not a real conflict.
  is_expected_owner() {
    case "$(echo "$1" | tr '[:upper:]' '[:lower:]')" in
      idea*|*jetbrains*|*intellij*|code|code\ helper*|electron|com.docker*|docker*|vpnkit*|rancher*|colima|limactl|lima*|podman*|gvproxy) return 0 ;;
      *) return 1 ;;
    esac
  }

  local p owner pid cmd CHECKED=0 CONFLICTS=0 EXP_PORTS="" EXP_OWNERS=""
  for p in $PORTS; do
    owner="$(port_owner "$p")"
    [ "$?" = 2 ] && break
    CHECKED=1
    [ -z "$owner" ] && continue
    pid="${owner%% *}"
    cmd="${owner#* }"
    if is_expected_owner "$cmd"; then
      EXP_PORTS="$EXP_PORTS $p"
      case "$EXP_OWNERS" in *"'$cmd' (pid $pid)"*) ;; *) EXP_OWNERS="$EXP_OWNERS, '$cmd' (pid $pid)" ;; esac
    else
      CONFLICTS=$((CONFLICTS + 1))
      fail "port $p is held by '$cmd' (pid $pid)" \
           "Stop that program before class, or the workshop services cannot start on this port."
    fi
  done
  if [ "$CHECKED" = 0 ]; then
    warn "neither lsof nor nc available, skipped port check"
  fi
  if [ -n "$EXP_PORTS" ]; then
    warn "ports in use by${EXP_OWNERS#,}:$EXP_PORTS" \
         "Looks like a dev container session that is already open. Fine if that is your workshop container; otherwise close it first."
  fi
  if [ "$CHECKED" = 1 ] && [ "$CONFLICTS" = 0 ] && [ -z "$EXP_PORTS" ]; then
    pass "all workshop ports are free"
  fi

  # ------------------------------------------------------------------------
  section "6. Pre-pull Docker images"

  local img
  if [ "$WARM" = 0 ]; then
    echo "  Skipped. Re-run with --warm to download the images now (recommended at home, not on class Wi-Fi)."
  elif [ "$DOCKER_OK" = 0 ]; then
    warn "Cannot pre-pull (Docker not available)"
  else
    for img in $IMAGES; do
      echo "  Pulling $img ..."
      if docker pull --quiet "$img" >/dev/null 2>&1; then
        pass "$img"
      else
        fail "could not pull $img" \
             "Check network access (see section 4). On Apple Silicon/ARM the image may also lack an arm64 build; tell your instructor."
      fi
    done
    echo
    echo "  Note: further tools are downloaded when you first open the project in its dev container."
  fi

  # ------------------------------------------------------------------------
  echo
  if [ "$FAILS" -gt 0 ]; then
    echo "${RED}${BOLD}$FAILS required check(s) failed${RESET}, $WARNS warning(s). Fix the items marked [FAIL] and re-run."
    return 1
  fi
  echo "${GREEN}${BOLD}All required checks passed${RESET} ($WARNS warning(s))."
  if [ "$WARM" = 0 ]; then
    echo
    echo "Tip: re-run with '--warm' (curl -fsSL <URL> | bash -s -- --warm) to pre-download the images."
  fi
  return 0
}

main "$@"
