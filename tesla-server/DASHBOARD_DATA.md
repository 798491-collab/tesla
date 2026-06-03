# Tesla 仪表盘数据手册

> 基于 Fleet Telemetry 双通道融合架构
> 最后更新：2026-05-31

---

## 一、数据延迟分级

| 层级 | 来源 | 延迟 | 适用场景 |
|------|------|------|---------|
| L1 实时流 | Fleet Telemetry WebSocket | ~550ms | 动画、圆环、实时指示 |
| L2 状态流 | Fleet Telemetry WebSocket | ~550ms ~ 60s | 状态图标、数值显示 |
| L3 轮询 | REST API vehicle_data | 3~60s | 兜底补充 |

> 实际延迟 = interval_seconds + 500ms（刷新周期）+ 网络延迟（<50ms）

---

## 二、核心仪表区

### 2.1 速度表

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 车速 | speed | ~550ms | km/h 或 mph | 主仪表核心，EMA 平滑系数 0.6 |
| 档位 | gear | ~550ms | P/R/N/D | 档位指示灯 |
| 驱动就绪 | drive_rail | ~550ms | boolean | READY 指示灯 |
| 定速巡航速度 | cruise_set_speed | ~550ms | km/h | 巡航设定值 |
| 巡航跟车距离 | cruise_follow_distance | ~550ms | 1-7 | 跟车距离档位 |
| 速度限制 | current_limit_mph | ~550ms | mph | 限速值 |

### 2.2 功率表

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 驱动功率 | power | ~550ms | kW | 正值加速/负值制动 |
| 电池电压 | pack_voltage | ~10.5s | V | 电池包电压 |
| 电池电流 | pack_current | ~10.5s | A | 正值放电/负值充电 |
| 油门踏板 | pedal_position | ~550ms | 0-100 | 踏板深度 |
| 刹车踏板 | brake_pedal | ~550ms | boolean | 刹车指示 |

### 2.3 G力显示

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 横向加速度 | lateral_acceleration | ~550ms | m/s² | 左右转弯G值 |
| 纵向加速度 | longitudinal_acceleration | ~550ms | m/s² | 加速/制动G值 |

---

## 三、电池 & 充电区

### 3.1 电量

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 电量百分比 | soc | ~10.5s | % | 主电量显示 |
| 剩余电量 | energy_remaining | ~10.5s | kWh | 剩余可用电量 |
| 理想续航 | range_km | ~10.5s | km | 理想条件续航 |
| 估算续航 | range_km（EstBatteryRange） | ~10.5s | km | 考虑驾驶条件的续航 |
| 总里程 | odometer_km | ~60.5s | km | 车辆总里程 |
| 充电上限 | charge_limit_soc | ~10.5s | % | 充电目标百分比 |

### 3.2 充电实时

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 充电状态 | charge_state | ~550ms | string | Disconnected/Charging/Complete/Stopped |
| DC充电功率 | dc_charging_power | ~10.5s | kW | 超充/快充功率 |
| AC充电功率 | ac_charging_power | ~10.5s | kW | 家充/慢充功率 |
| 合并充电功率 | charge_power | ~10.5s | kW | dc + ac 合并值 |
| 充电器电压 | voltage | ~10.5s | V | 充电器输入电压 |
| 充电电流 | ampere | ~10.5s | A | 充电电流 |
| 充电速率 | charge_speed | ~10.5s | km/h | 每小时增加的续航 |
| 充满时间 | minutes_to_full | ~60.5s | 分钟 | 预计充满时间 |
| 已充入电量 | dc_charging_energy_in | ~10.5s | kWh | 本次充电已充入 |
| 快充连接 | fast_charger_present | ~550ms | boolean | 是否连接超充 |
| 快充类型 | fast_charger_type | ~10.5s | enum | 超充类型 |
| 充电口开 | charge_port_door_open | ~550ms | boolean | 充电口门状态 |
| 充电使能 | charge_enable_request | ~10.5s | boolean | 充电是否启用 |
| 电池加热 | battery_heater_on | ~10.5s | boolean | 电池预热状态 |

### 3.3 充电曲线数据

| 数据 | 字段名 | 延迟 | 用途 |
|------|--------|------|------|
| soc + dc_charging_power | 每 ~10.5s 采样 | 充电功率曲线 |
| soc + charge_speed | 每 ~10.5s 采样 | 充电速率曲线 |
| pack_voltage + pack_current | 每 ~10.5s 采样 | 电压/电流曲线 |

---

## 四、空调 & 舒适区

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 车内温度 | inside_temp | ~550ms | ℃ | 车内温度计 |
| 车外温度 | outside_temp | ~550ms | ℃ | 车外温度计 |
| 空调开关 | is_ac_on | ~550ms | boolean | 空调总开关 |
| AC开关 | hvac_ac_enabled | ~550ms | boolean | 压缩机开关 |
| 设定温度 | driver_temp_setting | ~550ms | ℃ | 驾驶侧温度设定 |
| 副驾温度 | passenger_temp_setting | ~550ms | ℃ | 副驾温度设定 |
| 风扇速度 | hvac_fan_speed | ~550ms | 0-7 | 风扇等级 |
| 自动模式 | hvac_auto_mode | ~550ms | enum | AUTO 模式 |
| 除霜模式 | defrost_mode | ~550ms | enum | 前/后除霜 |
| 预调节除霜 | defrost_for_preconditioning | ~550ms | boolean | 预调节引起的除霜 |
| 气候保持 | climate_keeper_mode | ~550ms | enum | 爱犬/露营/自定义 |
| 过热保护 | cabin_overheat_protection_mode | ~10.5s | enum | 驾驶舱过热保护 |
| 方向盘加热 | steering_wheel_heater | ~550ms | boolean | 方向盘加热开关 |
| 方向盘加热等级 | hvac_steering_wheel_heat_level | ~550ms | 0-3 | 加热等级 |
| 方向盘自动 | hvac_steering_wheel_heat_auto | ~550ms | boolean | 自动加热 |
| 左前座椅加热 | seat_heater_left | ~550ms | 0-3 | 座椅加热等级 |
| 右前座椅加热 | seat_heater_right | ~550ms | 0-3 | 座椅加热等级 |
| 左后座椅加热 | seat_heater_rear_left | ~550ms | 0-3 | 座椅加热等级 |
| 右后座椅加热 | seat_heater_rear_right | ~550ms | 0-3 | 座椅加热等级 |
| 左前座椅冷却 | climate_seat_cooling_front_left | ~550ms | 0-3 | 座椅通风等级 |
| 右前座椅冷却 | climate_seat_cooling_front_right | ~550ms | 0-3 | 座椅通风等级 |
| 左前座椅自动 | auto_seat_climate_left | ~550ms | boolean | 自动座椅空调 |
| 右前座椅自动 | auto_seat_climate_right | ~550ms | boolean | 自动座椅空调 |

---

## 五、车辆状态区

| 数据 | 字段名 | 延迟 | 说明 |
|------|--------|------|------|
| 锁车状态 | locked | ~550ms | 已锁/未锁 |
| 左前门 | door_fl | ~550ms | 开/关 |
| 右前门 | door_fr | ~550ms | 开/关 |
| 左后门 | door_rl | ~550ms | 开/关 |
| 右后门 | door_rr | ~550ms | 开/关 |
| 后备箱 | trunk_open | ~550ms | 开/关 |
| 前备箱 | frunk_open | ~550ms | 开/关 |
| 驾驶侧车窗 | fd_window | ~550ms | 开/关/通风 |
| 副驾侧车窗 | fp_window | ~550ms | 开/关/通风 |
| 哨兵模式 | sentry_mode | ~550ms | 开/关 |
| 服务模式 | service_mode | ~550ms | 开/关 |
| 访客模式 | guest_mode_enabled | ~550ms | 开/关 |
| 中控屏状态 | center_display_state | ~550ms | 屏幕 on/off |

---

## 六、安全辅助区

| 数据 | 字段名 | 延迟 | 说明 |
|------|--------|------|------|
| 安全带 | driver_seat_belt | ~550ms | 已系/未系 |
| 驾驶员在座 | driver_seat_occupied | ~550ms | 在座/空座 |
| 盲区摄像头 | automatic_blind_spot_camera | ~550ms | 开/关 |
| 盲区警告音 | blind_spot_collision_warning_chime | ~550ms | 开/关 |
| 前方碰撞预警 | forward_collision_warning | ~550ms | 灵敏度等级 |
| 车道偏离辅助 | lane_departure_avoidance | ~550ms | 辅助等级 |
| 紧急车道偏离 | emergency_lane_departure_avoidance | ~550ms | 开/关 |
| 自动紧急制动 | automatic_emergency_braking_off | ~550ms | 是否关闭 |

---

## 七、灯光区

| 数据 | 字段名 | 延迟 | 说明 |
|------|--------|------|------|
| 远光灯 | lights_high_beams | ~550ms | 开/关 |
| 危险灯 | lights_hazards_active | ~550ms | 双闪开/关 |
| 转向灯 | lights_turn_signal | ~550ms | 左/右/双闪/无 |

---

## 八、定位 & 导航区

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 纬度 | latitude | ~550ms | ° | 地图中心 |
| 经度 | longitude | ~550ms | ° | 地图中心 |
| 方向 | heading | ~550ms | ° | 0=北, 90=东 |
| GPS状态 | gps_state | ~550ms | boolean | 信号锁定 |
| 目的地纬度 | destination_latitude | ~10.5s | ° | 导航终点 |
| 目的地经度 | destination_longitude | ~10.5s | ° | 导航终点 |
| 目的地名称 | destination_name | ~550ms | string | 导航终点名称 |

---

## 九、媒体区

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 播放状态 | media_playback_status | ~550ms | enum | Playing/Paused/Stopped |
| 音频源 | media_audio_source | ~550ms | string | FM/USB/Bluetooth/Spotify等 |
| 音量 | media_volume | ~550ms | 0-11 | 当前音量 |
| 曲目标题 | now_playing_title | ~550ms | string | 歌曲名 |
| 艺术家 | now_playing_artist | ~550ms | string | 歌手名 |
| 专辑 | now_playing_album | ~550ms | string | 专辑名 |
| 曲目时长 | now_playing_duration | ~550ms | ms | 总时长 |
| 播放进度 | now_playing_elapsed | ~550ms | ms | 当前位置 |

---

## 十、胎压区

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 前左胎压 | tpms_fl | ~60.5s | bar/psi | 左前轮胎压 |
| 前右胎压 | tpms_fr | ~60.5s | bar/psi | 右前轮胎压 |
| 后左胎压 | tpms_rl | ~60.5s | bar/psi | 左后轮胎压 |
| 后右胎压 | tpms_rr | ~60.5s | bar/psi | 右后轮胎压 |

---

## 十一、动力总成区（高级）

| 数据 | 字段名 | 延迟 | 单位 | 说明 |
|------|--------|------|------|------|
| 前驱状态 | di_state_f | ~550ms | enum | 逆变器状态 |
| 后驱状态 | di_state_r | ~550ms | enum | 逆变器状态 |
| 前驱实际扭矩 | di_torque_actual_f | ~550ms | Nm | 前驱输出扭矩 |
| 后驱实际扭矩 | di_torque_actual_r | ~550ms | Nm | 后驱输出扭矩 |
| 前驱定子温度 | di_stator_temp_f | ~550ms | ℃ | 电机温度 |
| 后驱定子温度 | di_stator_temp_r | ~550ms | ℃ | 电机温度 |
| 前驱散热器温度 | di_heatsink_tf | ~550ms | ℃ | 散热器温度 |
| 后驱散热器温度 | di_heatsink_tr | ~550ms | ℃ | 散热器温度 |
| 前轴转速 | di_axle_speed_f | ~550ms | rpm | 轴转速 |
| 后轴转速 | di_axle_speed_r | ~550ms | rpm | 轴转速 |

---

## 十二、推导字段

以下字段由后端或前端从原始数据推导，无需额外配置：

| 推导字段 | 推导逻辑 | 用途 |
|---------|---------|------|
| charging | charge_state === 'Charging' \|\| charge_state === 'Complete' | 充电指示灯 |
| driving | gear === 'D' \|\| gear === 'R' \|\| gear === 'N' | 行驶指示灯 |
| charge_power | dc_charging_power + ac_charging_power | 合并充电功率 |
| seat_heater | { left, right, rear_left, rear_right } | 座椅加热组合对象 |
| odometer_km | odometer × 1.60934 | 英里转公里 |
| range_km | ideal_battery_range × 1.60934 | 英里转公里 |
| charge_speed | charge_rate_mile_per_hour × 1.60934 | 英里转公里 |
| minutes_to_full | estimated_hours_to_charge_termination × 60 | 小时转分钟 |

---

## 十三、仪表盘布局参考

### 横屏布局（16:9 大屏）

```
┌──────────┬──────────────────────────────┬──────────┐
│          │                              │          │
│  车外温度 │        速度圆环 82 km/h       │  电量 71% │
│  🌡️ 32°C │          ● D                 │  🔋 45kWh│
│          │     巡航 120 km/h             │  🛣️ 328km│
│  车内温度 │                              │          │
│  🌡️ 24°C │    G力球 ⬈ 0.3g             │  充电中   │
│          │                              │  23 kW   │
│  空调 AUTO│                              │  ⏱️ 1h32m│
│  24°C    │                              │          │
├──────────┼──────────────────────────────┼──────────┤
│ 🔒 已锁  │                              │ 🎵 晴天  │
│ 🛡️ 哨兵  │       实时地图               │ 🎤 周杰伦 │
│ ❄️ 除霜  │       550ms 刷新             │ 🔊 ████░ │
│ 🪑 加热L2│       方向标记旋转            │ ▶ 2:30/4:12│
│ 💡 远光  │                              │          │
│ 🛞 胎压OK│                              │          │
└──────────┴──────────────────────────────┴──────────┘
```

### 竖屏布局（手机）

```
┌─────────────────────┐
│     82 km/h  ● D    │
│     巡航 120 km/h   │
├─────────────────────┤
│ 🔋 71%  328km  45kWh│
│ ⚡ 充电中 23kW      │
│ ⏱️ 1h 32min 充满    │
├─────────────────────┤
│ 🌡️ 外32°C 内24°C   │
│ ❄️ AUTO 24°C        │
├─────────────────────┤
│ 🔒 已锁 🛡️ 哨兵     │
│ 4门关 窗关 箱关      │
├─────────────────────┤
│ 🗺️ 实时地图         │
│ 🧭 东北 45°         │
├─────────────────────┤
│ 🎵 晴天 - 周杰伦    │
│ ▶ ████████░░ 2:30   │
└─────────────────────┘
```

---

## 十四、动画实现建议

| 动画类型 | 数据源 | 推荐帧率 | 平滑方式 | CSS/Canvas |
|---------|--------|---------|---------|-----------|
| 速度圆环 | speed | 60fps 渲染 | EMA α=0.6 | Canvas |
| 功率圆环 | power | 60fps 渲染 | EMA α=0.3 | Canvas |
| G力球 | lateral/longitudinal_acceleration | 60fps 渲染 | EMA α=0.3 | Canvas |
| 档位切换 | gear | 即时 | 无需平滑 | CSS transition |
| 地图标记旋转 | heading | 60fps 渲染 | 线性插值 | CSS transform |
| 充电脉冲 | charge_power | 30fps 渲染 | EMA α=0.2 | CSS animation |
| 播放进度条 | now_playing_elapsed | 60fps 渲染 | 线性插值 | CSS width |
| 温度变化 | inside/outside_temp | 即时 | CSS transition | CSS transition |

> EMA（指数移动平均）公式：`smoothed = α × newValue + (1 - α) × prevValue`
> α 越大响应越快，α 越小越平滑

---

## 十五、数据总量统计

| 区域 | 可显示数据数 | 550ms 级 | 10s 级 | 60s 级 |
|------|------------|---------|--------|--------|
| 核心仪表 | 9 | 7 | 0 | 0 |
| 电池充电 | 17 | 3 | 12 | 2 |
| 空调舒适 | 22 | 20 | 2 | 0 |
| 车辆状态 | 13 | 13 | 0 | 0 |
| 安全辅助 | 8 | 8 | 0 | 0 |
| 灯光 | 3 | 3 | 0 | 0 |
| 定位导航 | 7 | 5 | 2 | 0 |
| 媒体 | 8 | 8 | 0 | 0 |
| 胎压 | 4 | 0 | 0 | 4 |
| 动力总成 | 10 | 10 | 0 | 0 |
| **合计** | **101** | **77** | **16** | **6** |

> 77 个数据点可达 ~550ms 级刷新，完全可以实现流畅的仪表盘动画效果。
