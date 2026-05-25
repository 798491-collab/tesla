#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

SERVER_DIR="$PROJECT_DIR/tesla-server"
APP_DIR="$PROJECT_DIR/tesla-app"
VCP_DIR="/opt/tesla-vcp"
VCP_KEYS_DIR="$VCP_DIR/keys"
VCP_BIN="/root/go/bin/tesla-http-proxy"
SERVER_BIN="$SERVER_DIR/tesla-server"
SERVER_PORT=1255
VCP_PORT=4443
DOMAIN=""
DB_USER="root"
DB_PASSWORD=""
DB_NAME="teslaapp"
DB_HOST="localhost"
DB_PORT="3306"
REDIS_HOST="localhost"
REDIS_PORT="6379"
REDIS_PASSWORD=""
TESLA_CLIENT_ID=""
TESLA_CLIENT_SECRET=""
JWT_SECRET=""
TENCENT_MAP_KEY=""
DEPLOY_ALL=true
DEPLOY_BACKEND=false
DEPLOY_FRONTEND=false
DEPLOY_VCP=false
DEPLOY_NGINX=false
SKIP_ENV=false

print_banner() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                                                  ║${NC}"
    echo -e "${CYAN}║        Tesla 远程控制平台 - 一键部署脚本         ║${NC}"
    echo -e "${CYAN}║                                                  ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}▶ $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_ok() {
    echo -e "  ${GREEN}✓ $1${NC}"
}

print_warn() {
    echo -e "  ${YELLOW}⚠ $1${NC}"
}

print_err() {
    echo -e "  ${RED}✗ $1${NC}"
}

print_info() {
    echo -e "  ${CYAN}ℹ $1${NC}"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_err "此脚本需要 root 权限运行"
        echo "  请使用: sudo bash $0"
        exit 1
    fi
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --backend)  DEPLOY_ALL=false; DEPLOY_BACKEND=true; shift ;;
            --frontend) DEPLOY_ALL=false; DEPLOY_FRONTEND=true; shift ;;
            --vcp)      DEPLOY_ALL=false; DEPLOY_VCP=true; shift ;;
            --nginx)    DEPLOY_ALL=false; DEPLOY_NGINX=true; shift ;;
            --skip-env) SKIP_ENV=true; shift ;;
            --domain=*) DOMAIN="${1#*=}"; shift ;;
            --help)
                echo "用法: sudo bash deploy.sh [选项]"
                echo ""
                echo "选项:"
                echo "  --backend      仅部署后端"
                echo "  --frontend     仅部署前端"
                echo "  --vcp          仅部署VCP代理"
                echo "  --nginx        仅配置Nginx"
                echo "  --domain=域名  指定域名"
                echo "  --skip-env     跳过.env配置（使用已有配置）"
                echo "  --help         显示帮助"
                exit 0
                ;;
            *) print_err "未知参数: $1"; exit 1 ;;
        esac
    done
}

check_command() {
    if command -v "$1" &> /dev/null; then
        local version=$($1 --version 2>&1 | head -1)
        print_ok "$1 已安装: $version"
        return 0
    else
        print_err "$1 未安装"
        return 1
    fi
}

check_environment() {
    print_step "检查服务器环境"

    local missing=0

    if ! check_command go; then
        missing=1
        print_info "安装 Go: https://go.dev/dl/ 或执行: snap install go --classic"
    fi

    if ! check_command mysql && ! check_command mariadb; then
        if systemctl is-active --quiet mysql || systemctl is-active --quiet mariadb; then
            print_ok "MySQL/MariaDB 服务运行中"
        else
            missing=1
            print_info "安装 MySQL: apt install mysql-server"
        fi
    else
        if systemctl is-active --quiet mysql || systemctl is-active --quiet mariadb; then
            print_ok "MySQL/MariaDB 服务运行中"
        fi
    fi

    if systemctl is-active --quiet redis-server || systemctl is-active --quiet redis; then
        print_ok "Redis 服务运行中"
    else
        if command -v redis-server &> /dev/null || command -v redis-cli &> /dev/null; then
            print_warn "Redis 已安装但未运行，尝试启动..."
            systemctl start redis-server 2>/dev/null || systemctl start redis 2>/dev/null || true
        else
            missing=1
            print_info "安装 Redis: apt install redis-server"
        fi
    fi

    if ! check_command nginx; then
        missing=1
        print_info "安装 Nginx: apt install nginx"
    fi

    if ! check_command node; then
        missing=1
        print_info "安装 Node.js: curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash - && apt install nodejs"
    else
        local node_version=$(node -v)
        print_ok "Node.js $node_version"
    fi

    if ! check_command npm; then
        missing=1
        print_info "安装 npm: apt install npm"
    fi

    if [[ $missing -eq 1 ]]; then
        echo ""
        print_err "缺少必要依赖，请先安装上述组件后重新运行"
        echo ""
        read -p "是否继续部署（已安装的组件仍会部署）？[y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

ask_domain() {
    if [[ -z "$DOMAIN" ]]; then
        echo ""
        read -p "请输入你的域名（如 www.example.com）: " DOMAIN
        if [[ -z "$DOMAIN" ]]; then
            print_err "域名不能为空"
            exit 1
        fi
    fi
    print_info "域名: $DOMAIN"
}

configure_env() {
    if [[ "$SKIP_ENV" == true ]]; then
        if [[ -f "$SERVER_DIR/.env" ]]; then
            print_ok "使用已有 .env 配置"
            return
        else
            print_warn "未找到 .env 文件，将重新配置"
        fi
    fi

    print_step "配置环境变量"

    if [[ -f "$SERVER_DIR/.env" ]]; then
        read -p "已存在 .env 文件，是否覆盖？[y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_ok "保留已有 .env 配置"
            return
        fi
    fi

    echo ""
    echo -e "${CYAN}请输入配置信息（直接回车使用默认值）:${NC}"
    echo ""

    read -p "MySQL 用户 [root]: " input_db_user
    DB_USER="${input_db_user:-root}"

    read -sp "MySQL 密码: " input_db_pass
    echo
    DB_PASSWORD="$input_db_pass"

    read -p "MySQL 数据库名 [teslaapp]: " input_db_name
    DB_NAME="${input_db_name:-teslaapp}"

    read -p "MySQL 主机 [localhost]: " input_db_host
    DB_HOST="${input_db_host:-localhost}"

    read -p "MySQL 端口 [3306]: " input_db_port
    DB_PORT="${input_db_port:-3306}"

    read -p "Redis 主机 [localhost]: " input_redis_host
    REDIS_HOST="${input_redis_host:-localhost}"

    read -p "Redis 端口 [6379]: " input_redis_port
    REDIS_PORT="${input_redis_port:-6379}"

    read -sp "Redis 密码（无密码直接回车）: " input_redis_pass
    echo
    REDIS_PASSWORD="$input_redis_pass"

    echo ""
    echo -e "${YELLOW}Tesla 开发者配置（从 https://developer.tesla.cn 获取）:${NC}"
    read -p "Tesla Client ID: " TESLA_CLIENT_ID
    read -sp "Tesla Client Secret: " TESLA_CLIENT_SECRET
    echo

    JWT_SECRET=$(openssl rand -hex 32)
    print_info "已自动生成 JWT 密钥"

    read -p "腾讯地图 Key（可选，直接回车跳过）: " TENCENT_MAP_KEY

    cat > "$SERVER_DIR/.env" << EOF
SERVER_PORT=1255
GIN_MODE=release

DB_HOST=$DB_HOST
DB_PORT=$DB_PORT
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_NAME=$DB_NAME

REDIS_HOST=$REDIS_HOST
REDIS_PORT=$REDIS_PORT
REDIS_PASSWORD=$REDIS_PASSWORD
REDIS_DB=0

TESLA_CLIENT_ID=$TESLA_CLIENT_ID
TESLA_CLIENT_SECRET=$TESLA_CLIENT_SECRET
TESLA_REDIRECT_URI=https://$DOMAIN/api/tesla/callback
TESLA_AUTH_URL=https://auth.tesla.cn/oauth2/v3/authorize
TESLA_TOKEN_URL=https://auth.tesla.cn/oauth2/v3/token
TESLA_FLEET_API_URL=https://fleet-api.prd.cn.vn.cloud.tesla.cn
TESLA_AUDIENCE=https://fleet-api.prd.cn.vn.cloud.tesla.cn
TESLA_VCP_URL=https://127.0.0.1:$VCP_PORT
TESLA_PARTNER_DOMAIN=$DOMAIN
TESLA_FRONTEND_CALLBACK_URL=https://$DOMAIN/#/pages/callback/callback

JWT_SECRET=$JWT_SECRET
JWT_EXPIRES_IN=86400

TENCENT_MAP_KEY=$TENCENT_MAP_KEY

AI_API_KEY=
AI_MODEL=glm-4-flash
AI_BASE_URL=https://open.bigmodel.cn/api/paas/v4
EOF

    chmod 600 "$SERVER_DIR/.env"
    print_ok ".env 配置文件已生成"
}

init_database() {
    print_step "初始化数据库"

    if [[ -z "$DB_PASSWORD" ]] && [[ -f "$SERVER_DIR/.env" ]]; then
        DB_PASSWORD=$(grep "^DB_PASSWORD=" "$SERVER_DIR/.env" | cut -d'=' -f2-)
        DB_USER=$(grep "^DB_USER=" "$SERVER_DIR/.env" | cut -d'=' -f2-)
        DB_NAME=$(grep "^DB_NAME=" "$SERVER_DIR/.env" | cut -d'=' -f2-)
        DB_HOST=$(grep "^DB_HOST=" "$SERVER_DIR/.env" | cut -d'=' -f2-)
        DB_PORT=$(grep "^DB_PORT=" "$SERVER_DIR/.env" | cut -d'=' -f2-)
    fi

    if [[ -z "$DB_PASSWORD" ]]; then
        read -sp "请输入 MySQL root 密码: " DB_PASSWORD
        echo
    fi

    if mysql -u"$DB_USER" -p"$DB_PASSWORD" -h"$DB_HOST" -P"$DB_PORT" -e "USE $DB_NAME" 2>/dev/null; then
        print_ok "数据库 $DB_NAME 已存在"
        read -p "是否重新初始化（会清空数据）？[y/N] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            mysql -u"$DB_USER" -p"$DB_PASSWORD" -h"$DB_HOST" -P"$DB_PORT" < "$SERVER_DIR/database.sql"
            print_ok "数据库已重新初始化"
        fi
    else
        mysql -u"$DB_USER" -p"$DB_PASSWORD" -h"$DB_HOST" -P"$DB_PORT" < "$SERVER_DIR/database.sql"
        print_ok "数据库初始化完成"
    fi
}

deploy_vcp() {
    print_step "部署 VCP 代理（Tesla 命令签名服务）"

    mkdir -p "$VCP_KEYS_DIR"

    if [[ -f "$VCP_KEYS_DIR/private.pem" ]]; then
        print_ok "密钥文件已存在，跳过生成"
    else
        print_info "生成 EC 密钥对（prime256v1）..."
        openssl ecparam -name prime256v1 -genkey -noout -out "$VCP_KEYS_DIR/private.pem"
        openssl ec -in "$VCP_KEYS_DIR/private.pem" -pubout -out "$VCP_KEYS_DIR/public.pem"
        print_ok "密钥对已生成"
    fi

    if [[ -f "$VCP_KEYS_DIR/tls-key.pem" ]]; then
        print_ok "TLS 证书已存在，跳过生成"
    else
        print_info "生成自签名 TLS 证书..."
        openssl req -x509 -newkey rsa:4096 -keyout "$VCP_KEYS_DIR/tls-key.pem" \
            -out "$VCP_KEYS_DIR/tls-cert.pem" -days 3650 -nodes \
            -subj "/CN=localhost"
        print_ok "TLS 证书已生成"
    fi

    if [[ -f "$VCP_BIN" ]]; then
        print_ok "tesla-http-proxy 已编译"
    else
        print_info "编译 tesla-http-proxy..."
        if [[ ! -d "/tmp/vehicle-command" ]]; then
            git clone https://github.com/teslamotors/vehicle-command.git /tmp/vehicle-command
        fi
        cd /tmp/vehicle-command
        go build -o "$VCP_BIN" ./cmd/tesla-http-proxy
        print_ok "tesla-http-proxy 编译完成"
    fi

    cat > /etc/systemd/system/tesla-http-proxy.service << EOF
[Unit]
Description=Tesla HTTP Proxy
After=network.target

[Service]
Type=simple
ExecStart=$VCP_BIN \\
  -tls-key $VCP_KEYS_DIR/tls-key.pem \\
  -cert $VCP_KEYS_DIR/tls-cert.pem \\
  -key-file $VCP_KEYS_DIR/private.pem \\
  -host 127.0.0.1 \\
  -port $VCP_PORT \\
  -verbose
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable tesla-http-proxy

    if systemctl is-active --quiet tesla-http-proxy; then
        systemctl restart tesla-http-proxy
        print_ok "tesla-http-proxy 已重启"
    else
        systemctl start tesla-http-proxy
        print_ok "tesla-http-proxy 已启动"
    fi

    sleep 2
    if systemctl is-active --quiet tesla-http-proxy; then
        print_ok "tesla-http-proxy 运行正常（端口 $VCP_PORT）"
    else
        print_err "tesla-http-proxy 启动失败，请检查: journalctl -u tesla-http-proxy"
    fi

    deploy_public_key
}

deploy_public_key() {
    print_info "托管公钥到网站..."

    local web_root="/www/wwwroot/$DOMAIN"
    if [[ ! -d "$web_root" ]]; then
        web_root="/var/www/$DOMAIN"
        mkdir -p "$web_root"
    fi

    local pem_dir="$web_root/.well-known/appspecific"
    mkdir -p "$pem_dir"
    cp "$VCP_KEYS_DIR/public.pem" "$pem_dir/com.tesla.3p.public-key.pem"
    print_ok "公钥已托管到 $pem_dir/com.tesla.3p.public-key.pem"

    print_info "验证公钥格式..."
    if head -1 "$VCP_KEYS_DIR/public.pem" | grep -q "BEGIN PUBLIC KEY"; then
        print_ok "公钥格式正确（PEM 格式）"
    else
        print_err "公钥格式错误！重新生成..."
        openssl ec -in "$VCP_KEYS_DIR/private.pem" -pubout -out "$VCP_KEYS_DIR/public.pem"
        cp "$VCP_KEYS_DIR/public.pem" "$pem_dir/com.tesla.3p.public-key.pem"
        print_ok "公钥已重新生成"
    fi
}

deploy_backend() {
    print_step "编译和部署后端服务"

    cd "$SERVER_DIR"

    if [[ ! -f "go.mod" ]]; then
        print_err "未找到 go.mod，请确认项目目录正确"
        exit 1
    fi

    print_info "安装 Go 依赖..."
    go mod tidy
    print_ok "依赖安装完成"

    print_info "编译 tesla-server..."
    CGO_ENABLED=0 go build -o "$SERVER_BIN" cmd/main.go
    print_ok "编译完成: $SERVER_BIN"

    cat > /etc/systemd/system/tesla-server.service << EOF
[Unit]
Description=Tesla Server
After=network.target mysql.service redis.service tesla-http-proxy.service
Wants=tesla-http-proxy.service

[Service]
Type=simple
WorkingDirectory=$SERVER_DIR
ExecStart=$SERVER_BIN
Restart=always
RestartSec=5
EnvironmentFile=$SERVER_DIR/.env

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable tesla-server

    if systemctl is-active --quiet tesla-server; then
        systemctl restart tesla-server
        print_ok "tesla-server 已重启"
    else
        systemctl start tesla-server
        print_ok "tesla-server 已启动"
    fi

    sleep 3
    if systemctl is-active --quiet tesla-server; then
        print_ok "tesla-server 运行正常（端口 $SERVER_PORT）"
    else
        print_err "tesla-server 启动失败，请检查: journalctl -u tesla-server -n 50"
        journalctl -u tesla-server -n 20 --no-pager
    fi
}

deploy_frontend() {
    print_step "编译和部署前端"

    cd "$APP_DIR"

    if [[ ! -f "package.json" ]]; then
        print_err "未找到 package.json，请确认项目目录正确"
        exit 1
    fi

    if [[ ! -f ".env" ]]; then
        print_info "生成前端 .env 配置..."
        cat > "$APP_DIR/.env" << EOF
VITE_API_BASE_URL=https://$DOMAIN
VITE_WS_BASE_URL=wss://$DOMAIN/api/ws
VITE_TENCENT_MAP_KEY=$TENCENT_MAP_KEY
VITE_TENCENT_MAP_STYLE_DARK=2
EOF
        print_ok "前端 .env 已生成"
    fi

    print_info "安装 npm 依赖..."
    npm install
    print_ok "依赖安装完成"

    print_info "编译 H5 版本..."
    npm run build:h5
    print_ok "前端编译完成"

    local dist_dir="$APP_DIR/dist/build/web"
    if [[ ! -d "$dist_dir" ]]; then
        print_err "编译产物未找到: $dist_dir"
        exit 1
    fi
    print_ok "编译产物: $dist_dir"
}

configure_nginx() {
    print_step "配置 Nginx 反向代理"

    ask_domain

    local nginx_conf="/etc/nginx/sites-available/$DOMAIN"
    local nginx_enabled="/etc/nginx/sites-enabled/$DOMAIN"

    if [[ -f "$nginx_conf" ]]; then
        read -p "Nginx 配置已存在，是否覆盖？[y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_ok "保留已有 Nginx 配置"
            return
        fi
    fi

    read -p "SSL 证书路径（留空则使用 Let's Encrypt 自动申请）: " ssl_cert_path
    read -p "SSL 密钥路径: " ssl_key_path

    local use_ssl=false
    if [[ -n "$ssl_cert_path" ]] && [[ -n "$ssl_key_path" ]] && [[ -f "$ssl_cert_path" ]] && [[ -f "$ssl_key_path" ]]; then
        use_ssl=true
    fi

    local web_root="/www/wwwroot/$DOMAIN"
    if [[ ! -d "$web_root" ]]; then
        web_root="/var/www/$DOMAIN"
    fi

    if [[ "$use_ssl" == true ]]; then
        cat > "$nginx_conf" << EOF
server {
    listen 80;
    server_name $DOMAIN;
    return 301 https://\$host\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;

    ssl_certificate     $ssl_cert_path;
    ssl_certificate_key $ssl_key_path;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    client_max_body_size 50m;

    location /.well-known/appspecific/com.tesla.3p.public-key.pem {
        alias $VCP_KEYS_DIR/public.pem;
        default_type application/x-pem-file;
        add_header Access-Control-Allow-Origin *;
    }

    location /api/ws {
        proxy_pass http://127.0.0.1:$SERVER_PORT;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:$SERVER_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 60s;
    }

    location / {
        root $web_root/tesla-app/dist/build/web;
        try_files \$uri \$uri/ /index.html;
    }

    access_log /var/log/nginx/$DOMAIN.access.log;
    error_log  /var/log/nginx/$DOMAIN.error.log;
}
EOF
    else
        print_info "未提供 SSL 证书，生成 HTTP 配置（建议后续配置 HTTPS）..."
        cat > "$nginx_conf" << EOF
server {
    listen 80;
    server_name $DOMAIN;

    client_max_body_size 50m;

    location /.well-known/appspecific/com.tesla.3p.public-key.pem {
        alias $VCP_KEYS_DIR/public.pem;
        default_type application/x-pem-file;
        add_header Access-Control-Allow-Origin *;
    }

    location /api/ws {
        proxy_pass http://127.0.0.1:$SERVER_PORT;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:$SERVER_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 60s;
    }

    location / {
        root $web_root/tesla-app/dist/build/web;
        try_files \$uri \$uri/ /index.html;
    }

    access_log /var/log/nginx/$DOMAIN.access.log;
    error_log  /var/log/nginx/$DOMAIN.error.log;
}
EOF

        print_warn "Tesla Fleet API 要求 HTTPS，请尽快配置 SSL 证书"
        print_info "可使用 Let's Encrypt: certbot --nginx -d $DOMAIN"
    fi

    if [[ ! -L "$nginx_enabled" ]]; then
        ln -s "$nginx_conf" "$nginx_enabled"
    fi

    nginx -t 2>/dev/null
    if [[ $? -eq 0 ]]; then
        systemctl reload nginx
        print_ok "Nginx 配置已生效"
    else
        print_err "Nginx 配置有误，请检查: nginx -t"
        nginx -t
    fi
}

register_partner() {
    print_step "注册 Tesla Partner Account"

    if ! systemctl is-active --quiet tesla-server; then
        print_err "tesla-server 未运行，无法注册"
        return
    fi

    read -p "是否现在注册 Partner Account？[Y/n] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        return
    fi

    local result=$(curl -s -X POST "http://localhost:$SERVER_PORT/api/tesla/partner/register?domain=$DOMAIN")
    echo "  返回: $result"

    if echo "$result" | grep -q '"code":200'; then
        print_ok "Partner Account 注册成功"
    else
        print_warn "Partner Account 注册失败，请检查公钥格式和可访问性"
        print_info "手动注册: curl -s -X POST 'http://localhost:$SERVER_PORT/api/tesla/partner/register?domain=$DOMAIN'"
    fi
}

print_summary() {
    print_step "部署完成 - 状态检查"

    echo ""
    echo -e "${CYAN}┌──────────────────────────────────────────────────┐${NC}"
    echo -e "${CYAN}│                 服务状态检查                      │${NC}"
    echo -e "${CYAN}├──────────────────────────────────────────────────┤${NC}"

    if systemctl is-active --quiet tesla-http-proxy; then
        echo -e "  ${GREEN}✓${NC} VCP 代理      运行中 (端口 $VCP_PORT)"
    else
        echo -e "  ${RED}✗${NC} VCP 代理      未运行"
    fi

    if systemctl is-active --quiet tesla-server; then
        echo -e "  ${GREEN}✓${NC} 后端服务      运行中 (端口 $SERVER_PORT)"
    else
        echo -e "  ${RED}✗${NC} 后端服务      未运行"
    fi

    if systemctl is-active --quiet nginx; then
        echo -e "  ${GREEN}✓${NC} Nginx         运行中"
    else
        echo -e "  ${RED}✗${NC} Nginx         未运行"
    fi

    if systemctl is-active --quiet mysql || systemctl is-active --quiet mariadb; then
        echo -e "  ${GREEN}✓${NC} MySQL         运行中"
    else
        echo -e "  ${RED}✗${NC} MySQL         未运行"
    fi

    if systemctl is-active --quiet redis-server || systemctl is-active --quiet redis; then
        echo -e "  ${GREEN}✓${NC} Redis         运行中"
    else
        echo -e "  ${RED}✗${NC} Redis         未运行"
    fi

    echo -e "${CYAN}├──────────────────────────────────────────────────┤${NC}"
    echo -e "  域名: $DOMAIN"
    echo -e "  API:  https://$DOMAIN/api/"
    echo -e "  WS:   wss://$DOMAIN/api/ws"
    echo -e "${CYAN}└──────────────────────────────────────────────────┘${NC}"

    echo ""
    echo -e "${YELLOW}后续步骤:${NC}"
    echo "  1. 确保 SSL 证书已配置（Tesla 要求 HTTPS）"
    echo "  2. 注册 Partner Account（如未注册）:"
    echo "     curl -s -X POST 'http://localhost:$SERVER_PORT/api/tesla/partner/register?domain=$DOMAIN'"
    echo "  3. 验证公钥可访问:"
    echo "     curl -s 'https://$DOMAIN/.well-known/appspecific/com.tesla.3p.public-key.pem' | head -1"
    echo "  4. 在手机浏览器打开配对链接（Safari/Chrome）:"
    echo "     https://tesla.cn/_ak/$DOMAIN?vin=你的VIN"
    echo ""
    echo -e "${YELLOW}常用命令:${NC}"
    echo "  查看后端日志:  journalctl -u tesla-server -f"
    echo "  查看VCP日志:   journalctl -u tesla-http-proxy -f"
    echo "  重启后端:      systemctl restart tesla-server"
    echo "  重启VCP:       systemctl restart tesla-http-proxy"
    echo "  重载Nginx:     systemctl reload nginx"
    echo ""
}

main() {
    print_banner
    check_root
    parse_args "$@"

    if [[ "$DEPLOY_ALL" == true ]]; then
        ask_domain
        check_environment
        configure_env
        init_database
        deploy_vcp
        deploy_backend
        deploy_frontend
        configure_nginx
        register_partner
    else
        if [[ "$DEPLOY_VCP" == true ]]; then
            ask_domain
            deploy_vcp
        fi
        if [[ "$DEPLOY_BACKEND" == true ]]; then
            configure_env
            init_database
            deploy_backend
        fi
        if [[ "$DEPLOY_FRONTEND" == true ]]; then
            ask_domain
            deploy_frontend
        fi
        if [[ "$DEPLOY_NGINX" == true ]]; then
            configure_nginx
        fi
    fi

    print_summary
}

main "$@"
