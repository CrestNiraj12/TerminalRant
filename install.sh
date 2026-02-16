#!/usr/bin/env bash
set -euo pipefail

REPO_OWNER="CrestNiraj12"
REPO_NAME="terminalrant"
BINARY_NAME="terminalrant"
PINNED_VERSION="0.1.0"
TMP_DIR=""

DEFAULT_INSTALL_DIR="/usr/local/bin"
if [ -n "${INSTALL_DIR:-}" ]; then
  INSTALL_DIR="${INSTALL_DIR}"
elif [ -w "${DEFAULT_INSTALL_DIR}" ]; then
  INSTALL_DIR="${DEFAULT_INSTALL_DIR}"
else
  INSTALL_DIR="${HOME}/.local/bin"
fi
if [ -n "${VERSION:-}" ] && [ "${VERSION#v}" != "${PINNED_VERSION}" ]; then
  printf '[install] error: this installer is pinned to v%s; requested VERSION=%s\n' "${PINNED_VERSION}" "${VERSION}" >&2
  exit 1
fi
VERSION="${PINNED_VERSION}"

log() {
  printf '[install] %s\n' "$*"
}

fail() {
  printf '[install] error: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

cleanup() {
  if [ -n "${TMP_DIR}" ] && [ -d "${TMP_DIR}" ]; then
    rm -rf "${TMP_DIR}"
  fi
}

resolve_os() {
  local uname_s
  uname_s="$(uname -s)"
  case "${uname_s}" in
    Linux*)
      printf 'linux'
      ;;
    Darwin*)
      printf 'darwin'
      ;;
    MINGW*|MSYS*|CYGWIN*)
      printf 'windows'
      ;;
    *)
      fail "unsupported OS: ${uname_s}"
      ;;
  esac
}

resolve_arch() {
  local uname_m
  uname_m="$(uname -m)"
  case "${uname_m}" in
    x86_64|amd64)
      printf 'amd64'
      ;;
    arm64|aarch64)
      printf 'arm64'
      ;;
    *)
      fail "unsupported architecture: ${uname_m}"
      ;;
  esac
}

main() {
  need_cmd curl
  need_cmd mktemp

  local os arch version ext archive_name download_url
  os="$(resolve_os)"
  arch="$(resolve_arch)"

  if [ "${os}" = "windows" ]; then
    [ "${arch}" = "amd64" ] || fail "windows arm64 release is not published"
    ext="zip"
    need_cmd unzip
  else
    ext="tar.gz"
    need_cmd tar
  fi

  version="${VERSION#v}"
  archive_name="${BINARY_NAME}_${version}_${os}_${arch}.${ext}"
  download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/v${version}/${archive_name}"

  log "detected target: ${os}/${arch}"
  log "install version: v${version}"
  log "download: ${download_url}"

  TMP_DIR="$(mktemp -d)"
  trap cleanup EXIT

  curl -fL "${download_url}" -o "${TMP_DIR}/${archive_name}" || fail "download failed"

  if [ "${ext}" = "zip" ]; then
    unzip -q "${TMP_DIR}/${archive_name}" -d "${TMP_DIR}"
  else
    tar -xzf "${TMP_DIR}/${archive_name}" -C "${TMP_DIR}"
  fi

  [ -f "${TMP_DIR}/${BINARY_NAME}" ] || fail "binary not found in archive"

  mkdir -p "${INSTALL_DIR}" || fail "unable to create install dir: ${INSTALL_DIR}"
  if command -v install >/dev/null 2>&1; then
    install -m 0755 "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}" || fail "install failed"
  else
    cp "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}" || fail "copy failed"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}" || fail "chmod failed"
  fi

  log "installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
  log "run '${BINARY_NAME} --version' to verify"
}

main "$@"
