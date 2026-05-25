# Tesla 中国区车联网平台

适用于：

- 国行 Tesla（LRW 开头）
- 中国区 Fleet API
- Go
- uniapp（H5 / App）
- Redis
- MySQL
- 腾讯地图
- WebSocket 实时通信
- 行程分析
- 充电分析
- 车辆控制
- VCP 虚拟钥匙配对
- 生产环境

## 开源协议

本项目采用 [AGPL-3.0](LICENSE) 开源协议。

核心要求：

- 修改后的代码**必须开源**
- 部署为网络服务也**必须公开源码**
- 必须保留原作者版权声明
- 不得使用原作者商标推广衍生产品

## 项目结构

```
├── tesla-server/          # Go 后端
│   ├── cmd/main.go        # 入口
│   ├── config/            # 配置
│   ├── internal/
│   │   ├── tesla/         # Tesla OAuth + Fleet API
│   │   ├── vcp/           # VCP 签名命令代理
│   │   ├── vehicle/       # 车辆状态管理
│   │   ├── ws/            # WebSocket 实时通信
│   │   ├── charging/      # 充电分析
│   │   ├── trip/          # 行程分析
│   │   └── ai/            # AI 分析
│   ├── models/            # 数据模型
│   ├── routes/            # 路由
│   ├── .env.example       # 环境变量示例
│   └── DEPLOY.md          # 完整部署指南
│
├── tesla-app/             # UniApp 前端
│   ├── api/               # API 接口
│   ├── components/        # 组件
│   ├── pages/             # 页面
│   │   ├── dashboard/     # 仪表盘
│   │   ├── control/       # 车辆控制
│   │   ├── vehicle/       # 车辆管理
│   │   ├── charging/      # 充电记录
│   │   ├── trip/          # 行程记录
│   │   ├── ai/            # AI 分析
│   │   └── ...
│   ├── store/             # Pinia 状态管理
│   ├── styles/            # 主题样式
│   ├── utils/             # 工具函数
│   ├── .env.example       # 环境变量示例
│   └── manifest.json.example  # 应用配置示例
│
├── LICENSE                # AGPL-3.0 开源协议
├── .gitignore             # Git 忽略规则
└── 项目说明.md             # 本文件
```

## 快速开始

### 后端

```bash
cd tesla-server
cp .env.example .env
# 编辑 .env 填入实际配置
go mod tidy
go build -o tesla-server cmd/main.go
./tesla-server
```

### 前端

```bash
cd tesla-app
cp .env.example .env
cp manifest.json.example manifest.json
# 编辑 .env 和 manifest.json 填入实际配置
npm install
npm run dev:h5
```

### 完整部署

参见 [DEPLOY.md](tesla-server/DEPLOY.md)，包含 VCP 代理部署、公钥配对、Nginx 配置等完整步骤。

***

# 一、系统总体架构

```
uniapp ←── WebSocket ──→ Go API Server ←── HTTP ──→ Tesla Fleet API（中国区）
                              ↕                         ↕
                         Redis 缓存              VCP Proxy（签名命令）
                              ↕
                         MySQL 持久化
```

### VCP 签名命令流程

```
App → Go后端 → VCP Proxy(用私钥签名) → Fleet API → 车辆
                                                              │
                                                  车辆验证公钥是否在信任链
                                                  ❌ public key not paired = 未配对
```

**VCP 命令成功的三个前提：**

1. VCP 代理运行中，且加载了私钥
2. 公钥已注册到 Tesla Partner Account
3. 公钥已配对到目标车辆（用户通过 Tesla App 确认）

***

# 二、数据通信架构（核心）

## 前端：纯 WebSocket 驱动，无 HTTP 轮询

前端数据更新完全依赖 WebSocket，不使用 HTTP 轮询。

### WebSocket 连接

```
wss://domain/api/ws?token=xxx          # 全局车辆状态
wss://domain/api/ws/vin/{vin}?token=xxx # 指定车辆状态
```

### 消息格式

服务端推送：

```json
{
  "type": "vehicle_state",
  "vin": "LRW...",
  "data": { "speed": 80, "battery_level": 65, ... },
  "ts": 1779284400123
}
```

前端发送：

```json
{
  "type": "ping"
}
```

### 降级策略

| 场景           | 行为                               |
| ------------ | -------------------------------- |
| WebSocket 正常 | 实时接收服务端推送，零延迟                    |
| WebSocket 断开 | 自动启动 HTTP fallback 轮询（3\~30s 间隔） |
| WebSocket 重连 | 自动停止 fallback，恢复实时推送             |
| 首次进入页面       | 拉取一次初始状态，然后依赖 WebSocket          |

### 数据源标识

| 数据源        | 说明                       |
| ---------- | ------------------------ |
| `ws`       | WebSocket 实时推送（默认）       |
| `ble`      | 蓝牙直连（低延迟）                |
| `fallback` | WebSocket 断开时的 HTTP 降级轮询 |

***

## 后端：轮询 Tesla Fleet API + WebSocket 广播

后端根据车辆状态动态调整轮询频率，获取数据后通过 WebSocket 实时广播给前端。

### 轮询策略

| 车辆状态          | 轮询频率 | 说明            |
| ------------- | ---- | ------------- |
| driving       | 2 秒  | 行驶中，高频获取速度/位置 |
| charging      | 15 秒 | 充电中，中频获取充电进度  |
| parked online | 60 秒 | 在线停车，低频检查     |
| asleep        | 不轮询  | 休眠中，避免唤醒车辆    |
| offline       | 不轮询  | 离线，无法通信       |

### WebSocket 广播流程

```
Tesla Fleet API → 后端轮询获取数据 → Redis 缓存更新 → WebSocket Hub 广播 → 所有订阅客户端
```

### WebSocket 服务端配置

| 配置              | 值    | 说明               |
| --------------- | ---- | ---------------- |
| WriteBufferSize | 4096 | 写缓冲区，适配车辆状态 JSON |
| Send 通道缓冲       | 256  | 防止高频推送时丢弃消息      |
| WriteTimeout    | 5s   | 写超时，避免慢连接阻塞      |
| ReadTimeout     | 120s | 读超时，配合 30s 心跳    |
| Ping 间隔         | 30s  | 服务端心跳检测          |

***

# 三、VCP 代理与虚拟钥匙配对

中国区 Tesla 车辆控制命令（解锁、开后备箱、空调等）必须通过 VCP（Vehicle Command Protocol）代理签名发送。

## 部署架构

```
/opt/tesla-vcp/keys/
├── private.pem      # EC 私钥（prime256v1）
├── public.pem       # PEM 格式公钥（必须用 openssl ec -pubout 生成）
├── tls-key.pem      # VCP 代理 TLS 私钥
└── tls-cert.pem     # VCP 代理 TLS 证书
```

## 关键步骤

1. **生成密钥对**：`openssl ecparam -name prime256v1 -genkey -noout -out private.pem`
2. **生成 PEM 公钥**：`openssl ec -in private.pem -pubout -out public.pem`（⚠️ 必须是 PEM 格式）
3. **托管公钥**：`https://域名/.well-known/appspecific/com.tesla.3p.public-key.pem`
4. **注册 Partner Account**：`POST /api/tesla/partner/register?domain=域名`
5. **车辆配对**：在手机打开 `https://tesla.cn/_ak/域名?vin=VIN`，Tesla App 确认

## 配对 API

| API                                           | 方法   | 说明                 |
| --------------------------------------------- | ---- | ------------------ |
| `/api/tesla/vehicle/:vin/pairing-url`         | GET  | 获取配对链接             |
| `/api/tesla/vehicle/:vin/fleet-status`        | GET  | 检查配对状态             |
| `/api/tesla/partner/register?domain=`         | POST | 注册 Partner Account |
| `/api/tesla/partner/check-public-key?domain=` | GET  | 检查公钥注册状态           |
| `/api/tesla/partner/check-hosting?domain=`    | GET  | 检查公钥托管状态           |

## 常见错误

| 错误                                        | 原因          | 修复                        |
| ----------------------------------------- | ----------- | ------------------------- |
| `Invalid EC public key`                   | 公钥不是 PEM 格式 | `openssl ec -pubout` 重新生成 |
| `public key not paired`                   | 虚拟钥匙未配对     | 手机打开配对链接                  |
| `Tesla Vehicle Command Protocol required` | 走了旧 REST 路径 | 确认 VCP 代理运行               |

***

# 四、腾讯地图集成

## 腾讯位置服务（LBS）

官方：[腾讯位置服务](https://lbs.qq.com/)

## 推荐功能

| 功能    | 用途       |
| ----- | -------- |
| 地图展示  | 实时车辆位置   |
| 逆地理编码 | 经纬度转地址   |
| 路径规划  | 行程轨迹     |
| POI   | 充电地点识别   |
| 行政区解析 | 城市分析     |
| 围栏    | 家/公司自动识别 |

***

# 五、腾讯地图接口（核心）

## 1. 经纬度转地址（逆地理编码）

接口：

```
GET https://apis.map.qq.com/ws/geocoder/v1/
```

参数：

| 参数       | 说明       |
| -------- | -------- |
| location | 纬度,经度    |
| key      | 腾讯地图 KEY |

示例：

```
https://apis.map.qq.com/ws/geocoder/v1/?location=31.2304,121.4737&key=xxxx
```

返回：

```json
{
  "result": {
    "address": "上海市浦东新区..."
  }
}
```

推荐用途：行程终点地址（公司、家、商场、高速服务区、超充站）

***

# 六、轨迹地图

## trip\_points

每次行驶记录：

```
latitude
longitude
speed
heading
battery_level
```

前端使用腾讯地图 polyline 显示行程轨迹：

```
起点 → 路线 → 终点
```

***

# 七、充电地点识别

充电结束后调用腾讯逆地理编码，自动识别：

| 类型     | 示例       |
| ------ | -------- |
| 家充     | 家        |
| 公司充电   | 公司       |
| 超充站    | Tesla 超充 |
| 第三方充电站 | 国家电网     |

***

# 八、家庭/公司地点识别

用户首次设置家坐标、公司坐标。

判断：距离 < 200 米，则自动识别为家/公司。

***

# 九、腾讯地图距离计算

行程轨迹优化，避免 GPS 漂移。

推荐使用腾讯地图距离接口：

```
https://apis.map.qq.com/ws/distance/v1/
```

用途：比 Tesla odometer 更精准分析城市道路、停车移动、轨迹修正。

***

# 十、充电分析系统

## 充电地点记录

新增字段：

```
address
city
district
poi_name
```

## 充电价格统计

数据库字段：

```
price_per_kwh  -- 电价(元/kWh)
total_cost     -- 总费用(元)
```

### 功能说明

1. 充电详情页添加价格：每条充电记录可添加电价，自动计算总费用
2. 月度统计总花费：充电列表页显示每月总费用
3. 费用统计维度：单月总花费、单次充电费用、平均电价分析

## 自动统计

| 数据     | 说明       |
| ------ | -------- |
| 家充次数   | 家        |
| 超充次数   | Tesla 超充 |
| 外部充电次数 | 第三方      |
| 每城市充电  | 城市分析     |
| 月度充电费用 | 总花费统计    |

***

# 十一、行程分析系统

## trip\_logs 增加

```
start_address
end_address
start_city
end_city
avg_consumption
```

百公里能耗 = 耗电量(kWh) / 行驶距离(km) × 100

***

# 十二、地图实时车辆显示

## 首页地图

显示：实时车辆位置、当前速度、电量、在线状态

腾讯地图 marker 根据 shift\_state 切换：

| 状态       | 图标 |
| -------- | -- |
| driving  | 行驶 |
| charging | 充电 |
| parked   | 停车 |
| asleep   | 离线 |

***

# 十三、围栏功能

地理围栏支持：到家提醒、离家提醒、到公司提醒、自动统计通勤。

***

# 十四、腾讯地图调用频率（优化）

不要每次推送都逆地理编码。

正确做法：

- 行程开始：调用一次
- 行程结束：调用一次
- 充电结束：调用一次

***

# 十五、Redis 缓存策略

车辆状态：

```
tesla:vehicle:{vin}:state
```

TTL：5 分钟

地图缓存：

```
tesla:geocode:{lat}:{lng}
```

TTL：7 天

***

# 十六、车辆控制接口（完整）

| 功能    | 接口                                            |
| ----- | --------------------------------------------- |
| 唤醒车辆  | POST /api/vcp/wake                            |
| 锁车    | POST /api/vcp/door\_lock                      |
| 解锁    | POST /api/vcp/door\_unlock                    |
| 开空调   | POST /api/vcp/auto\_conditioning\_start       |
| 关空调   | POST /api/vcp/auto\_conditioning\_stop        |
| 后备箱   | POST /api/vcp/actuate\_trunk                  |
| 前备箱   | POST /api/vcp/actuate\_frunk                  |
| 哨兵模式  | POST /api/vcp/set\_sentry\_mode               |
| 闪灯    | POST /api/vcp/flash\_lights                   |
| 鸣笛    | POST /api/vcp/honk\_horn                      |
| 开始充电  | POST /api/vcp/charge\_start                   |
| 停止充电  | POST /api/vcp/charge\_stop                    |
| 充电限值  | POST /api/vcp/set\_charge\_limit              |
| 开充电口  | POST /api/vcp/charge\_port\_door\_open        |
| 关充电口  | POST /api/vcp/charge\_port\_door\_close       |
| 设置温度  | POST /api/vcp/set\_temps                      |
| 座椅加热  | POST /api/vcp/remote\_seat\_heater            |
| 方向盘加热 | POST /api/vcp/remote\_steering\_wheel\_heater |

所有控制命令均通过 VCP 代理签名发送。

***

# 十七、控制接口限流（必须）

| 接口            | 限制         |
| ------------- | ---------- |
| wake\_up      | 5 分钟最多 1 次 |
| command       | 最低间隔 2 秒   |
| vehicle\_data | 最低 5 秒     |

***

# 十八、双主题适配

## 支持浅色/深色主题切换

### 实现方式

- CSS 变量定义主题色
- Pinia 状态管理主题模式
- 所有页面适配双主题

### 主题变量

```
--bg-page          -- 页面背景
--bg-card          -- 卡片背景
--text-primary     -- 主要文字
--color-primary    -- 主题色
--dark-page-*      -- 深色页面专用变量
```

### 适配页面

- 首页仪表盘
- 车辆详情页
- 充电记录页
- 行驶记录页
- 车辆控制页
- 个人中心页
- AI 分析页

***

# 十九、推荐前端页面

| 页面   | 功能             |
| ---- | -------------- |
| 首页   | 地图、车辆状态、电量、温度  |
| 行程页面 | 轨迹回放、行驶统计、耗电分析 |
| 充电页面 | 充电记录、电费分析、地点统计 |
| 控制页面 | 锁车、空调、后备箱、哨兵模式 |
| 车辆页面 | 车辆管理、虚拟钥匙配对    |

***

# 二十、腾讯地图推荐 SDK

uniapp 推荐：腾讯地图 uni-app SDK

***

# 二十一、最终生产方案（稳定版）

## 核心原则

1. 前端永远不直接请求 Tesla API
2. 所有 Tesla API 仅后端访问
3. 前端通过 WebSocket 实时接收数据，不使用 HTTP 轮询
4. WebSocket 断开时自动降级为 HTTP fallback 轮询
5. 车辆休眠时立即停止后端轮询
6. 中国区控制命令必须通过 VCP 代理签名

***

# 二十二、最终推荐技术栈

| 模块     | 技术                      |
| ------ | ----------------------- |
| 后端     | Go + Gin                |
| 数据库    | MySQL（表名统一 tesla\_ 前缀）  |
| 缓存     | Redis                   |
| 实时通信   | WebSocket               |
| 地图     | 腾讯地图                    |
| 前端     | uniapp + Pinia + Vue3   |
| VCP 代理 | tesla-http-proxy（Go 编译） |
| 部署     | Linux + Nginx + HTTPS   |

***

# 二十三、Nginx WebSocket 配置（必须）

```nginx
location /api/ws {
    proxy_pass http://127.0.0.1:1255;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
    proxy_connect_timeout 60s;
    proxy_buffering off;
}

# 公钥托管 - Tesla Fleet API 要求
location /.well-known/appspecific/com.tesla.3p.public-key.pem {
    alias /opt/tesla-vcp/keys/public.pem;
    default_type application/x-pem-file;
    add_header Access-Control-Allow-Origin *;
}
```

***

# 二十四、用户认证

## Token 机制

- 后端 JWT 生成，默认有效期 24 小时（可通过 JWT\_EXPIRES\_IN 环境变量调整）
- 登录返回 `token` + `expires`（秒数）+ `expiresAt`（Unix 时间戳）
- 前端 Pinia 持久化存储 token 和 expiresAt
- 请求前本地检查过期，过期自动跳转登录页
- 服务端 401 响应自动清除本地存储并跳转

## 数据库表

| 表名                            | 说明         |
| ----------------------------- | ---------- |
| tesla\_users                  | 用户表        |
| tesla\_user\_tokens           | 用户 Token 表 |
| tesla\_vehicles               | 车辆表        |
| tesla\_oauth\_accounts        | OAuth 账户表  |
| tesla\_vehicle\_command\_logs | 命令日志表      |
| tesla\_charging\_sessions     | 充电记录表      |
| tesla\_trip\_logs             | 行程记录表      |
| tesla\_trip\_points           | 行程轨迹点表     |

***

# 二十五、车辆图片

使用 Tesla 官方 Compositor 接口根据车辆配置渲染图片。

```
https://static-assets.tesla.com/v1/compositor/?view=STUD_3QTR&model=my&options=PPSW,MTY03,MDLY
```

| 参数      | 说明                                               |
| ------- | ------------------------------------------------ |
| view    | 视角：STUD\_3QTR（45度）、STUD\_SIDE（侧面）、STUD\_REAR（后视） |
| model   | 车型：my/m3/ms/mx                                   |
| options | 车辆配置码（颜色、轮毂等），来自 Fleet API option\_codes         |
| bkba    | 暗黑模式：1=黑色背景，不传=白色背景                              |

后端在绑定时自动生成 vehicle\_image URL 并存入数据库，前端直接使用。

***

# 二十六、环境变量

## 后端（tesla-server/.env）

| 变量名                    | 必填 | 说明                  |
| ---------------------- | -- | ------------------- |
| `SERVER_PORT`          | 否  | 服务端口，默认 8080        |
| `DB_HOST`              | 否  | MySQL 地址            |
| `DB_PASSWORD`          | 是  | MySQL 密码            |
| `TESLA_CLIENT_ID`      | 是  | Tesla Client ID     |
| `TESLA_CLIENT_SECRET`  | 是  | Tesla Client Secret |
| `TESLA_VCP_URL`        | 是  | VCP 代理地址            |
| `TESLA_PARTNER_DOMAIN` | 是  | Partner 域名          |
| `JWT_SECRET`           | 是  | JWT 签名密钥            |

## 前端（tesla-app/.env）

| 变量名                    | 说明          |
| ---------------------- | ----------- |
| `VITE_API_BASE_URL`    | 后端 API 地址   |
| `VITE_TENCENT_MAP_KEY` | 腾讯地图 Key    |
| `VITE_WECHAT_APPID`    | 微信小程序 AppID |

***

# 二十七、踩坑记录

| 问题                                        | 原因          | 修复                        |
| ----------------------------------------- | ----------- | ------------------------- |
| `Invalid EC public key`                   | 公钥不是 PEM 格式 | `openssl ec -pubout` 重新生成 |
| `public key not paired`                   | 虚拟钥匙未配对     | 手机打开配对链接                  |
| `Tesla Vehicle Command Protocol required` | 走了旧 REST 路径 | 确认 VCP 代理运行               |
| 配对链接不调起 App                               | 用了微信浏览器     | 用 Safari/Chrome 打开        |
| `bind: address already in use`            | 端口被占用       | 先 `pkill` 旧服务             |

