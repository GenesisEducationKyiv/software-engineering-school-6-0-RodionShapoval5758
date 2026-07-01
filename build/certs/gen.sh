#!/usr/bin/env bash
# Generates a private CA and mTLS leaf certs for the subscription <-> monitoring
# gRPC (CatalogService) link. Output goes to ./certs (gitignored), relative to
# the repo root regardless of where this script is invoked from.
#
# Usage: build/certs/gen.sh [--force]
#   --force   regenerate the CA even if certs/ca.key already exists.
#             Without it, re-running is safe and only (re)issues leaf certs
#             signed by the existing CA, so already-distributed leaves aren't
#             silently invalidated by an accidental CA rotation.
set -euo pipefail

FORCE=0
if [[ "${1:-}" == "--force" ]]; then
  FORCE=1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
out_dir="${repo_root}/certs"
mkdir -p "${out_dir}"
cd "${out_dir}"

if [[ -f ca.key && "${FORCE}" -eq 0 ]]; then
  echo "certs/ca.key already exists, reusing existing CA (pass --force to regenerate)"
else
  echo "generating root CA"
  openssl genrsa -out ca.key 4096
  openssl req -x509 -new -key ca.key -sha256 -days 3650 \
    -subj "/CN=github-release-notification-ca" -out ca.crt
fi

gen_leaf() {
  local name="$1" eku="$2" san_ext="$3"

  echo "generating ${name} leaf cert"
  openssl genrsa -out "${name}.key" 2048
  # Leaf keys are read inside containers by a non-root user (e.g. "app") that
  # doesn't match the host file owner, so they need to be world-readable.
  # ca.key never leaves the host and stays at its default restrictive mode.
  chmod 644 "${name}.key"
  openssl req -new -key "${name}.key" -subj "/CN=${name}" -out "${name}.csr"
  openssl x509 -req -in "${name}.csr" -CA ca.crt -CAkey ca.key -CAcreateserial \
    -days 90 -sha256 \
    -extfile <(printf "%s\nextendedKeyUsage=%s\n" "${san_ext}" "${eku}") \
    -out "${name}.crt"
  rm -f "${name}.csr"
}

gen_leaf "subscription" "serverAuth" "subjectAltName=DNS:subscription,DNS:localhost,IP:127.0.0.1"
gen_leaf "monitoring" "clientAuth" "subjectAltName=DNS:monitoring"

echo "done. certs written to ${out_dir}"
