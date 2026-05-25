# Tesla 远程控制平台 - 完整部署指南

## 目录

1. [系统架构](#1-系统架构)
2. [服务器环境要求](#2-服务器环境要求)
3. [Tesla Developer 配置](#3-tesla-developer-配置)
4. [VCP 代理部署（非 Docker）](#4-vcp-代理部署非-docker)
5. [公钥生成与托管](#5-公钥生成与托管)
6. [Partner Account 注册](#6-partner-account-注册)
7. [虚拟钥匙配对](#7-虚拟钥匙配对)
8. [后端服务部署](#8-后端服务部署)
9. [前端部署](#9-前端部署)
10. [环境变量清单](#10-环境变量清单)
11. [踩坑记录与故障排查](#11-踩坑记录与故障排查)

---

## 1. 系统架构

```
用户手机 App (UniApp)
       │
       ▼
  Nginx 反向代理 (:443/:80)
       │
       ├── /api/* ──────────► Go 后端 (:1255)
       │                         │
       │                         ├── Fleet API (车辆数据)
       │                         │   https://fleet-api.prd.cn.vn.cloud.tesla.cn
       │                         │
       │                         └── VCP 代理 (签名命令) ◄── 关键！
       │                             https://127.0.0.1:4443
       │
       └── /* ──────────────► 前端静态文件

VCP 签名命令流程：
  App → Go后端 → VCP Proxy(用私钥签名) → Fleet API → 车辆
                                                                  │
                                                      车辆验证公钥是否在信任链
                                                      ❌ public key not paired = 未配对
```

**核心前提**：VCP 命令必须满足三个条件才能成功：
1. ✅ VCP 代理运行中，且加载了私钥
2. ✅ 公钥已注册到 Tesla Partner Account
3. ✅ 公钥已配对到目标车辆（用户通过 Tesla App 确认）

---

## 2. 服务器环境要求

| 组件 | 最低版本 | 说明 |
|------|---------|------|
| Go | 1.22+ | 后端编译 + VCP 代理编译 |
| MySQL | 5.7+ | 数据存储 |
| Redis | 6.0+ | 缓存/锁/状态 |
| Nginx | 1.18+ | 反向代理 + HTTPS + 公钥托管 |

---

## 3. Tesla Developer 配置

### 3.1 注册开发者账号

1. 访问 https://developer.tesla.cn （中国区）
2. 创建 Tesla 账号，启用 MFA
3. 创建应用，填写：
   - **应用名称**：你的应用名
   - **描述**：应用用途
   - **域名**：你的域名（如 `www.baidu.com`）
   - **Scopes**：勾选 `vehicle_device_data`、`vehicle_cmds`、`vehicle_charging_cmds`、`vehicle_location`

### 3.2 获取 Client ID 和 Secret

在开发者控制台中获取：
- `Client ID`
- `Client Secret`

记录下来，后面配置环境变量要用。

---

## 4. VCP 代理部署（非 Docker）

VCP 代理（tesla-http-proxy）负责用你的私钥对命令进行签名。**没有它，所有控制命令都会失败。**

### 4.1 创建工作目录和密钥

```bash
mkdir -p /opt/tesla-vcp/keys
cd /opt/tesla-vcp/keys

# 生成私钥（prime256v1 是 Tesla 唯一支持的曲线）
openssl ecparam -name prime256v1 -genkey -noout -out private.pem

# ⚠️ 重要：必须用 -pubout 生成 PEM 格式公钥，不能用原始格式！
openssl ec -in private.pem -pubout -out public.pem

# 生成自签名 TLS 证书（VCP 代理需要）
openssl req -x509 -newkey rsa:4096 -keyout tls-key.pem -out tls-cert.pem \
  -days 3650 -nodes -subj "/CN=localhost"

# 验证密钥文件
ls -la /opt/tesla-vcp/keys/
cat /opt/tesla-vcp/keys/public.pem
```

**验证公钥格式必须是 PEM 格式：**
```
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
-----END PUBLIC KEY-----
```

⚠️ **如果公钥是原始十六进制格式（如 `041384d8...`），Tesla 会返回 `Invalid EC public key`！必须用 `openssl ec -pubout` 生成 PEM 格式。**

### 4.2 编译 VCP 代理

```bash
# 确保 Go 已安装
go version

# 编译
git clone https://github.com/teslamotors/vehicle-command.git /tmp/vehicle-command
cd /tmp/vehicle-command
go build -o /root/go/bin/tesla-http-proxy ./cmd/tesla-http-proxy

# 验证
tesla-http-proxy --help
```

### 4.3 启动 VCP 代理

```bash
nohup /root/go/bin/tesla-http-proxy \
  -tls-key /opt/tesla-vcp/keys/tls-key.pem \
  -cert /opt/tesla-vcp/keys/tls-cert.pem \
  -key-file /opt/tesla-vcp/keys/private.pem \
  -host 127.0.0.1 \
  -port 4443 \
  -verbose \
  > /opt/tesla-vcp/proxy.log 2>&1 &
```

### 4.4 验证 VCP 代理运行

```bash
# 检查进程
ps aux | grep tesla-http-proxy

# 检查端口
ss -tlnp | grep 4443

# 测试连接（会返回 401，但说明代理在运行）
curl -k https://localhost:4443/api/1/vehicles
```

### 4.5 配置 systemd 自动启动（推荐）

```bash
cat > /etc/systemd/system/tesla-http-proxy.service << 'EOF'
[Unit]
Description=Tesla HTTP Proxy
After=network.target

[Service]
Type=simple
ExecStart=/root/go/bin/tesla-http-proxy \
  -tls-key /opt/tesla-vcp/keys/tls-key.pem \
  -cert /opt/tesla-vcp/keys/tls-cert.pem \
  -key-file /opt/tesla-vcp/keys/private.pem \
  -host 127.0.0.1 \
  -port 4443 \
  -verbose
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable tesla-http-proxy
systemctl start tesla-http-proxy
systemctl status tesla-http-proxy
```

---

## 5. 公钥生成与托管

### 5.1 生成密钥对

已在 [4.1 节](#41-创建工作目录和密钥) 完成。

### 5.2 托管公钥到网站

Tesla 要求公钥必须通过 HTTPS 公开可访问：

```
https://你的域名/.well-known/appspecific/com.tesla.3p.public-key.pem
```

#### 方案 A：直接放到网站根目录（最快）

```bash
mkdir -p /www/wwwroot/你的域名/.well-known/appspecific/
cp /opt/tesla-vcp/keys/public.pem /www/wwwroot/你的域名/.well-known/appspecific/com.tesla.3p.public-key.pem
```

#### 方案 B：通过 Nginx 托管（推荐）

```nginx
server {
    listen 443 ssl;
    server_name 你的域名;

    # 公钥托管 - Tesla Fleet API 要求
    location /.well-known/appspecific/com.tesla.3p.public-key.pem {
        alias /opt/tesla-vcp/keys/public.pem;
        default_type application/x-pem-file;
        add_header Access-Control-Allow-Origin *;
    }

    # ... 其他配置
}
```

```bash
nginx -t && nginx -s reload
```

### 5.3 验证公钥可访问

```bash
curl -s "https://你的域名/.well-known/appspecific/com.tesla.3p.public-key.pem" | head -3
```

**必须返回 PEM 格式：**
```
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
-----END PUBLIC KEY-----
```

⚠️ **如果返回原始十六进制（如 `041384d8...`），说明公钥格式错误！重新执行：**
```bash
openssl ec -in /opt/tesla-vcp/keys/private.pem -pubout -out /opt/tesla-vcp/keys/public.pem
cp /opt/tesla-vcp/keys/public.pem /www/wwwroot/你的域名/.well-known/appspecific/com.tesla.3p.public-key.pem
```

---

## 6. Partner Account 注册

将你的公钥注册到 Tesla Fleet API，这样 Tesla 才知道你的应用身份。

### 6.1 通过 API 注册

```bash
curl -s -X POST "http://localhost:1255/api/tesla/partner/register?domain=你的域名"
```

成功返回：
```json
{"code":200,"message":"Partner account registered successfully"}
```

⚠️ **如果返回 `Invalid EC public key`**：说明公钥文件格式不对或不可访问，回到第 5 步修复。

### 6.2 验证注册状态

```bash
curl -s "http://localhost:1255/api/tesla/partner/check-public-key?domain=你的域名"
```

### 6.3 检查公钥托管状态

```bash
curl -s "http://localhost:1255/api/tesla/partner/check-hosting?domain=你的域名"
```

---

## 7. 虚拟钥匙配对

这是**最关键的一步**！即使 VCP 代理运行正常、公钥已注册，如果公钥没有配对到车辆，所有命令都会返回 `public key not paired`。

### 7.1 配对原理

```
1. 用户在手机浏览器打开配对链接 → https://tesla.cn/_ak/你的域名?vin=VIN
2. 链接调起 Tesla App → 弹出确认框 "是否添加第三方钥匙？"
3. 用户点击确认 → Tesla App 发送指令到车辆
4. 车辆将公钥添加到信任链 → 配对完成
```

### 7.2 通过 App 配对（推荐）

1. 打开 App → 我的车辆页面
2. 每辆车下方有「虚拟钥匙」状态卡片
3. 如果显示「未配对」，点击「配对」按钮
4. 弹出配对引导弹窗，点击「打开配对链接」
5. 链接会调起 Tesla App，确认添加钥匙
6. 返回 App，点击「已配对？点击检查状态」

### 7.3 手动配对

在手机浏览器（Safari/Chrome，不要用微信内置浏览器）中打开：

```
https://tesla.cn/_ak/你的域名?vin=你的VIN
```

⚠️ **注意事项：**
- 必须用 **Safari（iOS）** 或 **Chrome（Android）** 打开，微信/QQ 浏览器不支持 Universal Link
- 车辆必须**在线**才能完成配对
- 如果没有自动调起 Tesla App，长按链接选择"在 Tesla 中打开"

### 7.4 验证配对状态

```bash
curl -s -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:1255/api/tesla/vehicle/YOUR_VIN/fleet-status"
```

成功返回：
```json
{
  "code": 200,
  "data": {
    "key_paired": true,
    "command_protocol_required": true,
    "signed_command_available": true
  }
}
```

**`key_paired: true` 表示配对成功，所有控制命令即可正常使用。**

---

## 8. 后端服务部署

### 8.1 编译

```bash
cd /www/wwwroot/你的域名/tesla-server

# 安装依赖
go mod tidy

# 编译
go build -o tesla-server cmd/main.go
```

### 8.2 配置环境变量

创建 `.env` 文件：

```bash
cat > /www/wwwroot/你的域名/tesla-server/.env << 'EOF'
# 服务器配置
SERVER_PORT=1255
GIN_MODE=release

# 数据库
DB_HOST=localhost
DB_PORT=3306
DB_USER=你的用户名
DB_PASSWORD=你的密码
DB_NAME=teslaapp

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Tesla OAuth（中国区）
TESLA_CLIENT_ID=你的ClientID
TESLA_CLIENT_SECRET=你的ClientSecret
TESLA_REDIRECT_URI=https://你的域名/api/tesla/callback
TESLA_AUTH_URL=https://auth.tesla.cn/oauth2/v3/authorize
TESLA_TOKEN_URL=https://auth.tesla.cn/oauth2/v3/token
TESLA_FLEET_API_URL=https://fleet-api.prd.cn.vn.cloud.tesla.cn
TESLA_AUDIENCE=https://fleet-api.prd.cn.vn.cloud.tesla.cn

# VCP 代理地址（关键！）
TESLA_VCP_URL=https://127.0.0.1:4443

# Partner 域名（用于生成配对链接，关键！）
TESLA_PARTNER_DOMAIN=你的域名

# 前端回调地址
TESLA_FRONTEND_CALLBACK_URL=https://你的域名/#/pages/callback/callback

# JWT
JWT_SECRET=你的JWT密钥
JWT_EXPIRES_IN=86400

# 地图
TENCENT_MAP_KEY=你的腾讯地图Key

# AI（可选）
AI_API_KEY=
AI_MODEL=glm-4-flash
AI_BASE_URL=https://open.bigmodel.cn/api/paas/v4
EOF
```

### 8.3 启动服务

```bash
# 停止旧服务
pkill tesla-server

# 启动新服务
cd /www/wwwroot/你的域名/tesla-server
nohup ./tesla-server > tesla-server.log 2>&1 &

# 检查启动状态
sleep 2
tail -20 tesla-server.log

# 确认端口监听
ss -tlnp | grep 1255
```

### 8.4 配置 systemd（推荐，自动重启）

```bash
cat > /etc/systemd/system/tesla-server.service << 'EOF'
[Unit]
Description=Tesla Server
After=network.target mysql.service redis.service

[Service]
Type=simple
WorkingDirectory=/www/wwwroot/你的域名/tesla-server
ExecStart=/www/wwwroot/你的域名/tesla-server/tesla-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable tesla-server
systemctl start tesla-server
systemctl status tesla-server
```

### 8.5 Nginx 反向代理配置

```nginx
server {
    listen 443 ssl http2;
    server_name 你的域名;

    ssl_certificate     /path/to/your/cert.pem;
    ssl_certificate_key /path/to/your/key.pem;

    # 公钥托管 - Tesla Fleet API 要求
    location /.well-known/appspecific/com.tesla.3p.public-key.pem {
        alias /opt/tesla-vcp/keys/public.pem;
        default_type application/x-pem-file;
        add_header Access-Control-Allow-Origin *;
    }

    # API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:1255;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
    }

    # WebSocket
    location /api/ws {
        proxy_pass http://127.0.0.1:1255;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # 前端静态文件
    location / {
        root /www/wwwroot/你的域名/tesla-app/dist/build/web;
        try_files $uri $uri/ /index.html;
    }
}
```

---

## 9. 前端部署

```bash
cd /www/wwwroot/你的域名/tesla-app

# 安装依赖
npm install

# 编译 H5
npm run build:h5

# 编译产物在 dist/build/web/
```

---

## 10. 环境变量清单

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `SERVER_PORT` | 否 | 8080 | 服务端口 |
| `GIN_MODE` | 否 | release | Gin 模式 |
| `DB_HOST` | 否 | localhost | MySQL 地址 |
| `DB_PORT` | 否 | 3306 | MySQL 端口 |
| `DB_USER` | 否 | root | MySQL 用户 |
| `DB_PASSWORD` | 是 | | MySQL 密码 |
| `DB_NAME` | 否 | tesla_platform | 数据库名 |
| `REDIS_HOST` | 否 | localhost | Redis 地址 |
| `REDIS_PORT` | 否 | 6379 | Redis 端口 |
| `REDIS_PASSWORD` | 否 | | Redis 密码 |
| `TESLA_CLIENT_ID` | 是 | | Tesla 开发者 Client ID |
| `TESLA_CLIENT_SECRET` | 是 | | Tesla 开发者 Client Secret |
| `TESLA_REDIRECT_URI` | 是 | | OAuth 回调地址 |
| `TESLA_VCP_URL` | **是** | | VCP 代理地址，如 `https://127.0.0.1:4443` |
| `TESLA_PARTNER_DOMAIN` | **是** | | Partner 域名，如 `www.baidu,com` |
| `TESLA_FRONTEND_CALLBACK_URL` | 否 | | 前端回调地址 |
| `JWT_SECRET` | 是 | | JWT 签名密钥 |
| `TENCENT_MAP_KEY` | 否 | | 腾讯地图 Key |

---

## 11. 踩坑记录与故障排查

### 🚨 坑1：公钥格式错误 → `Invalid EC public key`

**现象**：注册 Partner Account 时返回 `Invalid EC public key`

**原因**：公钥文件是原始 EC 点格式（十六进制），不是 PEM 格式。

**错误示例**：
```
041384d860090f7b649baf7b957c14258670b843fb319c59206814a9b24ff312702aa5de427b7c2d10f639a21a8c8a1b90b6fdcb15ae4293fb744066ef10019f0c
```

**正确格式**：
```
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
-----END PUBLIC KEY-----
```

**修复**：
```bash
openssl ec -in /opt/tesla-vcp/keys/private.pem -pubout -out /opt/tesla-vcp/keys/public.pem
cp /opt/tesla-vcp/keys/public.pem /www/wwwroot/你的域名/.well-known/appspecific/com.tesla.3p.public-key.pem
```

### 🚨 坑2：缺少 TESLA_PARTNER_DOMAIN 环境变量

**现象**：配对链接无法生成，返回 `Partner domain not configured`

**原因**：`.env` 文件中没有 `TESLA_PARTNER_DOMAIN` 配置。

**修复**：在 `.env` 中添加：
```
TESLA_PARTNER_DOMAIN=你的域名
```

### 🚨 坑3：配对链接不调起 Tesla App

**现象**：打开 `https://tesla.cn/_ak/域名` 后跳转到网页而不是 Tesla App

**原因**：
1. 用了微信/QQ 内置浏览器打开（不支持 Universal Link）
2. 手机没有安装 Tesla App

**修复**：
- 用 **Safari（iOS）** 或 **Chrome（Android）** 打开
- 确保已安装并登录 Tesla App
- iOS 可以长按链接 → 选择"在 Tesla 中打开"

### 🚨 坑4：所有命令返回 `public key not paired`

**现象**：VCP 代理正常、Partner Account 已注册，但命令仍被拒绝

**原因**：虚拟钥匙未配对到车辆

**修复**：在手机上打开配对链接完成配对，参见第 7 节

### 🚨 坑5：`Tesla Vehicle Command Protocol required`

**现象**：命令返回此错误

**原因**：请求走了旧的 REST API 路径，没有经过 VCP 代理签名

**修复**：确认 `TESLA_VCP_URL` 环境变量已设置，且 VCP 代理在运行

### 🚨 坑6：端口冲突 → `bind: address already in use`

**现象**：启动后端时端口被占用

**修复**：
```bash
# 先停掉旧服务
pkill tesla-server
sleep 1
# 再启动
nohup ./tesla-server > tesla-server.log 2>&1 &
```

### 🚨 坑7：`command too frequent, please wait 2 seconds`

**原因**：命令发送间隔小于 2 秒

**修复**：等待 2 秒后重试

---

## 快速部署检查清单

部署完成后，按顺序验证：

- [ ] MySQL 连接正常
- [ ] Redis 连接正常
- [ ] VCP 代理运行中（`ps aux | grep tesla-http-proxy`，端口 4443）
- [ ] 公钥文件可通过 HTTPS 访问（必须是 PEM 格式 `-----BEGIN PUBLIC KEY-----`）
- [ ] Go 后端启动成功（端口 1255）
- [ ] `.env` 中 `TESLA_VCP_URL` 和 `TESLA_PARTNER_DOMAIN` 已配置
- [ ] Partner Account 注册成功（`/api/tesla/partner/register`）
- [ ] 用户 OAuth 授权成功
- [ ] 车辆绑定成功
- [ ] **虚拟钥匙配对成功**（`key_paired: true`）
- [ ] 控制命令正常执行
