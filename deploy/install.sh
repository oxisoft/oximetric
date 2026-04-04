#!/usr/bin/env bash
set -euo pipefail

REPO_RAW="https://raw.githubusercontent.com/oxisoft/oximetric/main"

echo ""
echo "  ╔═══════════════════════════════════════╗"
echo "  ║         OXI Metric Installer          ║"
echo "  ║   Privacy-first on-premises analytics ║"
echo "  ╚═══════════════════════════════════════╝"
echo ""

# --- Database ---
echo "Database engine:"
echo "  1) SQLite  — zero config, single file, great for <100K DAU"
echo "  2) PostgreSQL — concurrent writes, better for high traffic"
echo ""
read -rp "Choose [1/2] (default: 1): " DB_CHOICE < /dev/tty
DB_CHOICE="${DB_CHOICE:-1}"

# --- Watchtower ---
echo ""
echo "Auto-updates with Watchtower?"
echo "  Watchtower checks daily for new OXI Metric versions and restarts automatically."
echo ""
read -rp "Enable auto-updates? [y/N] (default: N): " WT_CHOICE < /dev/tty
WT_CHOICE="${WT_CHOICE:-n}"

# --- Determine compose file ---
case "${DB_CHOICE}" in
  2)
    case "${WT_CHOICE}" in
      [yY]*) COMPOSE_FILE="docker-compose.postgres.watchtower.yml" ;;
      *)     COMPOSE_FILE="docker-compose.postgres.yml" ;;
    esac
    ;;
  *)
    case "${WT_CHOICE}" in
      [yY]*) COMPOSE_FILE="docker-compose.sqlite.watchtower.yml" ;;
      *)     COMPOSE_FILE="docker-compose.sqlite.yml" ;;
    esac
    ;;
esac

# --- Admin credentials ---
echo ""
echo "Admin account setup:"
echo ""
read -rp "Admin username (default: admin): " ADMIN_USER < /dev/tty
ADMIN_USER="${ADMIN_USER:-admin}"

while true; do
  read -rsp "Admin password (min 8 characters): " ADMIN_PASS < /dev/tty
  echo ""
  if [ "${#ADMIN_PASS}" -ge 8 ]; then
    break
  fi
  echo "  Password too short. Minimum 8 characters."
done

# --- JWT Secret ---
echo ""
echo "JWT secret (used to sign authentication tokens):"
echo "  1) Generate automatically (recommended)"
echo "  2) Enter manually"
echo ""
read -rp "Choose [1/2] (default: 1): " JWT_CHOICE < /dev/tty
JWT_CHOICE="${JWT_CHOICE:-1}"

if [ "${JWT_CHOICE}" = "2" ]; then
  while true; do
    read -rsp "JWT secret (min 32 characters): " JWT_SECRET < /dev/tty
    echo ""
    if [ "${#JWT_SECRET}" -ge 32 ]; then
      break
    fi
    echo "  Too short. Minimum 32 characters."
  done
else
  JWT_SECRET=$(head -c 48 /dev/urandom | base64 | tr -d '/+=' | head -c 48)
  echo "  Generated: ${JWT_SECRET:0:8}..."
fi

# --- Domain name ---
echo ""
read -rp "Public domain name (e.g. https://analytics.yourcompany.com, or leave empty): " DOMAIN_NAME < /dev/tty
DOMAIN_NAME="${DOMAIN_NAME:-}"

# --- PostgreSQL password ---
DB_PASSWORD=""
if [ "${DB_CHOICE}" = "2" ]; then
  echo ""
  DB_PASSWORD=$(head -c 24 /dev/urandom | base64 | tr -d '/+=' | head -c 24)
  echo "PostgreSQL password generated automatically."
fi

# --- Download compose file ---
echo ""
echo "Downloading ${COMPOSE_FILE}..."
curl -fsSL "${REPO_RAW}/deploy/${COMPOSE_FILE}" -o docker-compose.yml

# --- Create .env ---
cat > .env << EOF
OXIMETRIC_ADMIN_USERNAME=${ADMIN_USER}
OXIMETRIC_ADMIN_PASSWORD=${ADMIN_PASS}
OXIMETRIC_JWT_SECRET=${JWT_SECRET}
EOF

if [ -n "${DOMAIN_NAME}" ]; then
  echo "OXIMETRIC_DOMAIN_NAME=${DOMAIN_NAME}" >> .env
fi

if [ -n "${DB_PASSWORD}" ]; then
  echo "OXIMETRIC_DB_PASSWORD=${DB_PASSWORD}" >> .env
fi

# --- Summary ---
echo ""
echo "  ✓ docker-compose.yml downloaded"
echo "  ✓ .env created"
echo ""
echo "  Configuration:"
echo "    Database:     $([ "${DB_CHOICE}" = "2" ] && echo "PostgreSQL" || echo "SQLite")"
echo "    Auto-updates: $(echo "${WT_CHOICE}" | grep -qi '^y' && echo "Yes (Watchtower)" || echo "No")"
echo "    Admin user:   ${ADMIN_USER}"
echo "    Domain:       ${DOMAIN_NAME:-not set (http://localhost:6940)}"
echo ""
echo "  Start OXI Metric:"
echo ""
echo "    docker compose up -d"
echo ""
echo "  Then open: ${DOMAIN_NAME:-http://localhost:6940}"
echo ""
