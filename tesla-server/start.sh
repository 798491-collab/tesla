#!/bin/bash
APP_DIR="/www/wwwroot/tesla"
CERTS_DIR="$APP_DIR/certs"
CERT_SRC_DIR="/www/server/panel/vhost/cert/chen.sfytdds.com"
VCP_KEYS_DIR="/opt/tesla-vcp/keys"

mkdir -p "$CERTS_DIR"

sync_if_newer() {
    local src="$1" dst="$2"
    if [ ! -f "$src" ]; then
        echo "[cert-sync] Source not found: $src"
        return
    fi
    if [ ! -f "$dst" ] || [ "$src" -nt "$dst" ]; then
        cp -f "$src" "$dst"
        chmod 644 "$dst"
        echo "[cert-sync] Synced: $src -> $dst"
    fi
}

sync_if_newer "$VCP_KEYS_DIR/private.pem" "$CERTS_DIR/private.pem"
sync_if_newer "$CERT_SRC_DIR/fullchain.pem" "$CERTS_DIR/fullchain.pem"
sync_if_newer "$CERT_SRC_DIR/privkey.pem" "$CERTS_DIR/privkey.pem"

chown -R www:www "$CERTS_DIR" 2>/dev/null

cd "$APP_DIR"
exec ./tesla-server
