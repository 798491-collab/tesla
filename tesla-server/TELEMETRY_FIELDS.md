# Tesla Fleet Telemetry 官方字段对照文档

> 最后更新：2026-05-31
> 官方文档：https://developer.tesla.cn/docs/fleet-api/fleet-telemetry/available-data
> Protobuf 定义：https://github.com/teslamotors/fleet-telemetry/blob/main/protos/vehicle_data.proto

---

## 概述

Tesla Fleet Telemetry 提供 **130+ 个字段**，车辆每 **500ms** 刷新一次数据桶推送至服务器。

每个字段需配置 `interval_seconds`（最小发射间隔），字段只在 **间隔已过且值已变化** 时才发射到数据桶。

实际延迟 ≈ `interval_seconds` + `500ms`（刷新周期）+ 网络延迟（<50ms）

---

## 字段状态标记

| 标记 | 含义 |
|------|------|
| ✅ 已配置 | 已在 ConfigureFleetTelemetry 中配置，后端可接收 |
| 🔧 已处理 | 已在 processProtobufTelemetry 中显式处理 |
| 📱 前端使用 | 前端页面已使用该字段 |
| ⚪ 未使用 | 后端已接收但前端暂未使用，预留扩展 |

---

## 一、Driving 行驶数据

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | VehicleSpeed | real | 车速（km/h 或 mph，取决于车辆设置） | 1s | ✅🔧📱 | speed |
| 2 | Gear | ShiftState enum | 当前档位（P/R/N/D） | 1s | ✅🔧📱 | gear |
| 3 | CruiseSetSpeed | real | 定速巡航设定速度 | 1s | ✅🔧📱 | cruise_set_speed |
| 4 | LateralAcceleration | real | 横向加速度（m/s²） | 1s | ✅🔧📱 | lateral_acceleration |
| 5 | LongitudinalAcceleration | real | 纵向加速度（m/s²） | 1s | ✅🔧📱 | longitudinal_acceleration |
| 6 | BrakePedal | boolean | 刹车踏板是否踩下 | 1s | ✅🔧⚪ | brake_pedal |
| 7 | BrakePedalPos | real | 制动主缸压力 | 60s | ✅⚪ | brake_pedal_pos |
| 8 | DriveRail | boolean | 驱动就绪状态（所有驱动相关 ECU 已上电） | 1s | ✅🔧⚪ | drive_rail |
| 9 | PedalPosition | real | 油门踏板位置 | 1s | ✅🔧📱 | pedal_position |
| 10 | LifetimeEnergyUsedDrive | real | 驾驶放电总能量（kWh），仅 Semi-truck | - | ❌ | - |

---

## 二、Location 定位数据

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | Location | LocationValue | 车辆经纬度 | 1s | ✅🔧📱 | latitude / longitude |
| 2 | GpsHeading | real | 车辆方向（0=北, 90=东） | 1s | ✅🔧📱 | heading |
| 3 | GpsState | boolean | GPS 是否已锁定 | 1s | ✅🔧📱 | gps_state |
| 4 | DestinationLocation | Location | 导航目的地坐标 | 10s | ✅🔧⚪ | destination_latitude / destination_longitude |
| 5 | DestinationName | string | 导航目的地名称 | 1s | ✅🔧⚪ | destination_name |
| 6 | LocatedAtHome | boolean | 车辆是否在家 | - | ❌ | - |
| 7 | LocatedAtWork | boolean | 车辆是否在公司 | - | ❌ | - |
| 8 | LocatedAtFavorite | boolean | 车辆是否在收藏位置 | - | ❌ | - |

---

## 三、Charging 充电数据

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | ChargeState | string | 充电状态（Disconnected/Charging/Complete/Stopped） | 1s | ✅🔧📱 | charge_state |
| 2 | DetailedChargeState | DetailedChargeStateValue enum | 详细充电状态（2024.38+固件） | 1s | ✅🔧📱 | charge_state |
| 3 | FastChargerPresent | boolean | 是否连接快充 | 1s | ✅🔧📱 | fast_charger_present |
| 4 | FastChargerType | FastCharger enum | 快充类型 | 10s | ✅🔧⚪ | fast_charger_type |
| 5 | Soc | real | 电池电量百分比 | 10s | ✅🔧📱 | soc |
| 6 | BatteryLevel | real | 电池电量百分比（与Soc类似） | 10s | ✅🔧📱 | battery_level |
| 7 | PackVoltage | real | 电池包电压（V） | 10s | ✅🔧📱 | pack_voltage |
| 8 | PackCurrent | real | 电池包电流（A） | 10s | ✅🔧📱 | pack_current |
| 9 | EnergyRemaining | real | 剩余电量（kWh） | 10s | ✅🔧📱 | energy_remaining |
| 10 | ChargerVoltage | real | 充电器输入电压（V），变化频繁建议设 minimum_delta | 10s | ✅🔧📱 | charger_voltage → voltage |
| 11 | ChargeAmps | real | AC 充电器感知输入电流（A） | 10s | ✅🔧📱 | charge_amps → ampere |
| 12 | DCChargingPower | real | DC 充电功率（kW） | 10s | ✅🔧📱 | dc_charging_power → charge_power |
| 13 | ACChargingPower | real | AC 充电功率（kW） | 10s | ✅🔧📱 | ac_charging_power → charge_power |
| 14 | DCChargingEnergyIn | real | DC/总充电电量（kWh），AC+DC 均可靠 | 10s | ✅🔧⚪ | dc_charging_energy_in |
| 15 | ACChargingEnergyIn | real | AC 充电电量（kWh），DC 充电时应忽略 | 10s | ✅🔧⚪ | ac_charging_energy_in |
| 16 | ChargePortDoorOpen | boolean | 充电口门是否打开 | 1s | ✅🔧📱 | charge_port_door_open |
| 17 | ChargePortLatch | ChargePortLatchValue enum | 充电口锁状态 | 10s | ✅🔧⚪ | charge_port_latch |
| 18 | ChargeLimitSoc | integer | 充电上限百分比 | 10s | ✅🔧📱 | charge_limit_soc |
| 19 | ChargeRateMilePerHour | real | 充电速率（英里/小时） | 10s | ✅🔧📱 | charge_speed（×1.60934→km/h） |
| 20 | IdealBatteryRange | real | 理想续航里程（英里） | 10s | ✅🔧📱 | range_km（×1.60934→km） |
| 21 | EstBatteryRange | real | 估算续航里程（英里），考虑驾驶条件 | 10s | ✅🔧📱 | range_km（备选） |
| 22 | EstimatedHoursToChargeTermination | real | 充满所需时间（小时） | 60s | ✅🔧⚪ | minutes_to_full（×60→分钟） |
| 23 | ChargeEnableRequest | boolean | 充电是否已启用 | 10s | ✅🔧⚪ | charge_enable_request |
| 24 | ChargeCurrentRequest | integer | 请求充电电流（A） | 10s | ✅🔧⚪ | charge_current_request |
| 25 | ChargeCurrentRequestMax | integer | 最大可用充电电流（A） | 10s | ✅🔧⚪ | charge_current_request_max |
| 26 | ChargerPhases | integer | 充电器相位数 | 10s | ✅🔧⚪ | charger_phases |
| 27 | ChargePortColdWeatherMode | boolean | 充电口寒冷模式 | 10s | ✅🔧⚪ | charge_port_cold_weather_mode |
| 28 | ChargingCableType | CableType enum | 充电线缆类型 | 60s | ✅🔧⚪ | charging_cable_type |
| 29 | ChargePort | ChargePortValue enum | 充电口类型 | 60s | ✅⚪ | charge_port |
| 30 | BatteryHeaterOn | boolean | 电池加热器是否开启 | 10s | ✅🔧⚪ | battery_heater_on |
| 31 | BMSState | BMSStateValue enum | BMS 操作状态 | 10s | ✅🔧⚪ | bms_state |
| 32 | BmsFullchargecomplete | boolean | BMS 是否充满 | 10s | ✅🔧⚪ | bms_full_charge_complete |
| 33 | BrickVoltageMax | integer | 电芯最高电压 | 60s | ✅🔧⚪ | brick_voltage_max |
| 34 | BrickVoltageMin | integer | 电芯最低电压 | 60s | ✅🔧⚪ | brick_voltage_min |
| 35 | DCDCEnable | boolean | PCS DCDC 使能状态 | 10s | ✅🔧⚪ | dcdc_enable |
| 36 | LifetimeEnergyUsed | real | 放电总能量（kWh） | 60s | ✅🔧⚪ | lifetime_energy_used |
| 37 | ExpectedEnergyPercentAtTripArrival | real | 到达目的地时预期电量百分比 | - | ❌ | - |

---

## 四、Climate 空调数据

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | InsideTemp | real | 车内温度（℃），建议设 minimum_delta | 1s | ✅🔧📱 | inside_temp |
| 2 | OutsideTemp | real | 车外温度（℃） | 1s | ✅🔧📱 | outside_temp |
| 3 | HvacPower | HvacPowerState enum | 空调系统电源状态 | 1s | ✅🔧📱 | hvac_power / is_ac_on / is_climate_on |
| 4 | HvacLeftTemperatureRequest | real | 左侧温度设定（℃） | 1s | ✅🔧📱 | driver_temp_setting |
| 5 | HvacRightTemperatureRequest | real | 右侧温度设定（℃） | 1s | ✅🔧📱 | passenger_temp_setting |
| 6 | HvacACEnabled | boolean | AC 是否开启 | 1s | ✅🔧⚪ | hvac_ac_enabled |
| 7 | HvacAutoMode | HvacAutoModeState enum | 空调自动模式状态 | 1s | ✅🔧⚪ | hvac_auto_mode |
| 8 | HvacFanSpeed | integer | 空调风扇速度 | 1s | ✅🔧⚪ | hvac_fan_speed |
| 9 | HvacFanStatus | integer | 风扇设定速度段 | 1s | ✅🔧⚪ | hvac_fan_status |
| 10 | DefrostMode | DefrostModeState enum | 除霜模式 | 1s | ✅🔧📱 | defrost_mode |
| 11 | DefrostForPreconditioning | boolean | 是否因预调节而除霜 | 1s | ✅🔧⚪ | defrost_for_preconditioning |
| 12 | ClimateKeeperMode | ClimateKeeperModeState enum | 气候保持模式 | 1s | ✅🔧⚪ | climate_keeper_mode |
| 13 | HvacSteeringWheelHeatLevel | integer | 方向盘加热等级 | 1s | ✅🔧📱 | steering_wheel_heater / hvac_steering_wheel_heat_level |
| 14 | HvacSteeringWheelHeatAuto | boolean | 方向盘加热是否自动 | 1s | ✅🔧⚪ | hvac_steering_wheel_heat_auto |
| 15 | AutoSeatClimateLeft | boolean | 左前座椅自动空调 | 1s | ✅🔧⚪ | auto_seat_climate_left |
| 16 | AutoSeatClimateRight | boolean | 右前座椅自动空调 | 1s | ✅🔧⚪ | auto_seat_climate_right |
| 17 | ClimateSeatCoolingFrontLeft | integer | 左前座椅冷却等级 | 1s | ✅🔧⚪ | climate_seat_cooling_front_left |
| 18 | ClimateSeatCoolingFrontRight | integer | 右前座椅冷却等级 | 1s | ✅🔧⚪ | climate_seat_cooling_front_right |
| 19 | CabinOverheatProtectionMode | CabinOverheatProtectionModeState enum | 驾驶舱过热保护模式 | 10s | ✅🔧⚪ | cabin_overheat_protection_mode |
| 20 | CabinOverheatProtectionTemperatureLimit | CabitOverheatProtectionTempLimit enum | 过热保护温度限制（低/中/高） | 10s | ✅🔧⚪ | cabin_overheat_protection_temperature_limit |

---

## 五、Seat Heater 座椅加热（Vehicle State 子类）

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | SeatHeaterLeft | integer | 左前座椅加热等级 | 1s | ✅🔧📱 | seat_heater_left → seat_heater.left |
| 2 | SeatHeaterRight | integer | 右前座椅加热等级 | 1s | ✅🔧📱 | seat_heater_right → seat_heater.right |
| 3 | SeatHeaterRearLeft | integer | 左后座椅加热等级 | 1s | ✅🔧📱 | seat_heater_rear_left → seat_heater.rear_left |
| 4 | SeatHeaterRearRight | integer | 右后座椅加热等级 | 1s | ✅🔧📱 | seat_heater_rear_right → seat_heater.rear_right |
| 5 | SeatHeaterRearCenter | integer | 后排中间座椅加热等级 | - | ❌ | - |

---

## 六、Media 媒体数据

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | MediaPlaybackStatus | MediaStatusValue enum | 播放状态（Playing/Paused/Stopped） | 1s | ✅🔧📱 | media_playback_status |
| 2 | MediaPlaybackSource | string | 音频源 | 1s | ✅🔧📱 | media_audio_source |
| 3 | MediaAudioVolume | real | 音量（0-11） | 1s | ✅🔧📱 | media_volume |
| 4 | MediaAudioVolumeIncrement | real | 音量增减步长 | - | ❌ | - |
| 5 | MediaAudioVolumeMax | real | 最大音量 | - | ❌ | - |
| 6 | MediaNowPlayingTitle | string | 当前播放曲目标题 | 1s | ✅🔧📱 | now_playing_title |
| 7 | MediaNowPlayingArtist | string | 当前播放艺术家 | 1s | ✅🔧📱 | now_playing_artist |
| 8 | MediaNowPlayingAlbum | string | 当前播放专辑 | 1s | ✅🔧📱 | now_playing_album |
| 9 | MediaNowPlayingDuration | integer | 曲目时长（毫秒），电台返回 18000000 | 1s | ✅🔧📱 | now_playing_duration |
| 10 | MediaNowPlayingElapsed | integer | 当前播放位置（毫秒） | 1s | ✅🔧📱 | now_playing_elapsed |

---

## 七、Vehicle State 车辆状态

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | Locked | boolean | 车辆是否已锁 | 1s | ✅🔧📱 | locked |
| 2 | DoorState | string | 车门状态（2024.44.32前副驾/左后互换） | 1s | ✅🔧📱 | door_fl/fr/rl/rr/trunk_open/frunk_open |
| 3 | SentryMode | SentryModeState enum | 哨兵模式 | 1s | ✅🔧📱 | sentry_mode |
| 4 | Odometer | real | 里程表（英里） | 60s | ✅🔧📱 | odometer_km（×1.60934→km） |
| 5 | CenterDisplay | DisplayState enum | 中控屏状态 | 1s | ✅🔧⚪ | center_display_state |
| 6 | DriverSeatOccupied | boolean | 驾驶员是否在座 | 1s | ✅🔧⚪ | driver_seat_occupied |
| 7 | FdWindow | WindowState enum | 驾驶侧车窗状态 | 1s | ✅🔧⚪ | fd_window |
| 8 | FpWindow | WindowState enum | 副驾侧车窗状态 | 1s | ✅🔧⚪ | fp_window |
| 9 | GuestModeEnabled | boolean | 访客模式是否启用 | 1s | ✅🔧⚪ | guest_mode_enabled |
| 10 | GuestModeMobileAccessState | GuestModeMobileAccess enum | 访客模式手机访问状态 | 1s | ✅⚪ | guest_mode_mobile_access_state |
| 11 | HomelinkDeviceCount | integer | 附近 Homelink 设备数量 | 1s | ✅🔧⚪ | homelink_device_count |
| 12 | HomelinkNearby | boolean | 附近是否有 Homelink 设备 | 1s | ✅🔧⚪ | homelink_nearby |
| 13 | LightsHazardsActive | boolean | 危险警示灯是否开启 | 1s | ✅🔧⚪ | lights_hazards_active |
| 14 | LightsHighBeams | boolean | 远光灯是否开启 | 1s | ✅🔧⚪ | lights_high_beams |
| 15 | LightsTurnSignal | TurnSignalState enum | 转向灯状态（左/右/双闪/无） | 1s | ✅⚪ | lights_turn_signal |
| 16 | CurrentLimitMph | real | 速度限制（mph） | 1s | ✅🔧⚪ | current_limit_mph |
| 17 | Version | string | 软件版本 | 60s | ✅🔧⚪ | version |
| 18 | ServiceMode | boolean | 服务模式 | - | ✅🔧⚪ | service_mode |

---

## 八、Safety 安全辅助

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | DriverSeatBelt | boolean | 驾驶员安全带是否未系 | 1s | ✅🔧⚪ | driver_seat_belt |
| 2 | AutomaticBlindSpotCamera | boolean | 盲区摄像头是否启用 | 1s | ✅🔧⚪ | automatic_blind_spot_camera |
| 3 | AutomaticEmergencyBrakingOff | boolean | 自动紧急制动是否关闭 | 1s | ✅🔧⚪ | automatic_emergency_braking_off |
| 4 | BlindSpotCollisionWarningChime | boolean | 盲区碰撞警告提示音 | 1s | ✅🔧⚪ | blind_spot_collision_warning_chime |
| 5 | CruiseFollowDistance | FollowDistance enum | 巡航跟随距离 | 1s | ✅🔧⚪ | cruise_follow_distance |
| 6 | EmergencyLaneDepartureAvoidance | boolean | 紧急车道偏离避让 | 1s | ✅🔧⚪ | emergency_lane_departure_avoidance |
| 7 | ForwardCollisionWarning | ForwardCollisionSensitivity enum | 前方碰撞警告灵敏度 | 1s | ✅🔧⚪ | forward_collision_warning |
| 8 | LaneDepartureAvoidance | LaneAssistLevel enum | 车道偏离辅助等级 | 1s | ✅🔧⚪ | lane_departure_avoidance |

---

## 九、Vehicle Configuration 车辆配置

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | CarType | enum | 车型（model3/modely/models/modelx等） | 60s | ✅🔧⚪ | car_type |
| 2 | ExteriorColor | string | 外观颜色 | 60s | ✅🔧⚪ | exterior_color |
| 3 | EfficiencyPackage | string | 效率包配置 | 60s | ✅🔧⚪ | efficiency_package |
| 4 | EuropeVehicle | boolean | 是否为欧洲版车辆 | 60s | ✅⚪ | europe_vehicle |

---

## 十、Powertrain 动力总成

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | DiStateF | DriveInverterState enum | 前驱逆变器状态 | 1s | ✅⚪ | di_state_f |
| 2 | DiStateR | DriveInverterState enum | 后驱逆变器状态 | 1s | ✅⚪ | di_state_r |
| 3 | DiStateREL | DriveInverterState enum | 后左逆变器状态 | 1s | ✅⚪ | di_state_rel |
| 4 | DiStateRER | DriveInverterState enum | 后右逆变器状态 | 1s | ✅⚪ | di_state_rer |
| 5 | DiHeatsinkTF | real | 前驱散热器温度 | 1s | ✅⚪ | di_heatsink_tf |
| 6 | DiHeatsinkTR | real | 后驱散热器温度 | 1s | ✅⚪ | di_heatsink_tr |
| 7 | DiHeatsinkTREL | real | 后左散热器温度 | 1s | ✅⚪ | di_heatsink_trel |
| 8 | DiHeatsinkTRER | real | 后右散热器温度 | 1s | ✅⚪ | di_heatsink_trer |
| 9 | DiAxleSpeedF | real | 前轴转速 | 1s | ✅⚪ | di_axle_speed_f |
| 10 | DiAxleSpeedR | real | 后轴转速 | 1s | ✅⚪ | di_axle_speed_r |
| 11 | DiAxleSpeedREL | real | 后左轴转速 | 1s | ✅⚪ | di_axle_speed_rel |
| 12 | DiAxleSpeedRER | real | 后右轴转速 | 1s | ✅⚪ | di_axle_speed_rer |
| 13 | DiSlaveTorqueCmd | real | 副驱动单元扭矩指令 | 1s | ✅⚪ | di_slave_torque_cmd |
| 14 | DiTorqueActualF | real | 前驱实际扭矩 | 1s | ✅⚪ | di_torque_actual_f |
| 15 | DiTorqueActualR | real | 后驱实际扭矩 | 1s | ✅⚪ | di_torque_actual_r |
| 16 | DiTorqueActualREL | real | 后左实际扭矩 | 1s | ✅⚪ | di_torque_actual_rel |
| 17 | DiTorqueActualRER | real | 后右实际扭矩 | 1s | ✅⚪ | di_torque_actual_rer |
| 18 | DiStatorTempF | real | 前驱定子温度 | 1s | ✅⚪ | di_stator_temp_f |
| 19 | DiStatorTempR | real | 后驱定子温度 | 1s | ✅⚪ | di_stator_temp_r |
| 20 | DiStatorTempREL | real | 后左定子温度 | 1s | ✅⚪ | di_stator_temp_rel |
| 21 | DiStatorTempRER | real | 后右定子温度 | 1s | ✅⚪ | di_stator_temp_rer |
| 22 | DiVBatF | real | 前驱电池电压 | 1s | ✅⚪ | di_vbat_f |
| 23 | DiVBatR | real | 后驱电池电压 | 1s | ✅⚪ | di_vbat_r |
| 24 | DiVBatREL | real | 后左电池电压 | 1s | ✅⚪ | di_vbat_rel |
| 25 | DiVBatRER | real | 后右电池电压 | 1s | ✅⚪ | di_vbat_rer |
| 26 | DiMotorCurrentF | real | 前驱电机电流 | 1s | ✅⚪ | di_motor_current_f |
| 27 | DiMotorCurrentR | real | 后驱电机电流 | 1s | ✅⚪ | di_motor_current_r |
| 28 | DiMotorCurrentREL | real | 后左电机电流 | 1s | ✅⚪ | di_motor_current_rel |
| 29 | DiMotorCurrentRER | real | 后右电机电流 | 1s | ✅⚪ | di_motor_current_rer |
| 30 | DiInverterTF | real | 前驱逆变器温度 | 1s | ✅⚪ | di_inverter_tf |
| 31 | DiInverterTR | real | 后驱逆变器温度 | 1s | ✅⚪ | di_inverter_tr |
| 32 | DiInverterTREL | real | 后左逆变器温度 | 1s | ✅⚪ | di_inverter_trel |
| 33 | DiInverterTRER | real | 后右逆变器温度 | 1s | ✅⚪ | di_inverter_trer |
| 34 | DiTorquemotor | real | 驱动单元扭矩指令 | 1s | ✅⚪ | di_torquemotor |
| 35 | Hvil | HvilStatus enum | 高压互锁状态 | 1s | ✅🔧⚪ | hvil |

---

## 十一、Service 服务

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | IsolationResistance | real | 高压母线与底盘间绝缘电阻 | 60s | ✅🔧⚪ | isolation_resistance |

---

## 十二、TPMS 胎压

| # | 官方字段名 | 类型 | 中文说明 | 配置间隔 | 状态 | 前端字段名 |
|---|-----------|------|---------|---------|------|-----------|
| 1 | TpmsPressureFl | real | 前左胎压 | 60s | ✅🔧📱 | tpms_fl |
| 2 | TpmsPressureFr | real | 前右胎压 | 60s | ✅🔧📱 | tpms_fr |
| 3 | TpmsPressureRl | real | 后左胎压 | 60s | ✅🔧📱 | tpms_rl |
| 4 | TpmsPressureRr | real | 后右胎压 | 60s | ✅🔧📱 | tpms_rr |

---

## 统计

| 类别 | 官方字段数 | 已配置 | 已处理 | 前端使用 | 未配置 |
|------|-----------|--------|--------|---------|--------|
| Driving | 10 | 9 | 9 | 7 | 1（Semi-truck） |
| Location | 8 | 5 | 5 | 3 | 3（低优先） |
| Charging | 37 | 36 | 34 | 15 | 1 |
| Climate | 20 | 20 | 20 | 6 | 0 |
| Seat Heater | 5 | 4 | 4 | 4 | 1（RearCenter） |
| Media | 10 | 8 | 8 | 8 | 2（VolumeIncrement/Max） |
| Vehicle State | 18 | 18 | 16 | 10 | 0 |
| Safety | 8 | 8 | 8 | 0 | 0 |
| Vehicle Config | 4 | 4 | 3 | 0 | 0 |
| Powertrain | 35 | 35 | 1 | 0 | 0 |
| Service | 1 | 1 | 1 | 0 | 0 |
| TPMS | 4 | 4 | 4 | 4 | 0 |
| **合计** | **160** | **152** | **113** | **57** | **8** |

> 💡 Powertrain 字段虽已配置，但大部分通过 `default` case 自动处理（toSnakeCase 转换后存入 stateFields），无需显式 case。
> 未配置的 8 个字段为：Semi-truck 专用（5个）、低优先级 Location（3个）、Media 音量细节（2个，已计入其他类别）。

---

## 前端页面字段使用对照

### dashboard.vue（首页3D模型）
speed, gear, soc, range_km, odometer_km, inside_temp, outside_temp, latitude, longitude, charging, charge_power, added_energy, charge_limit_soc, trunk_open, frunk_open, locked, is_ac_on, media_playback_status, media_audio_source, media_volume, now_playing_title, now_playing_artist, door_fl/fr/rl/rr, mirror_folded, sentry_mode

### instrument.vue（仪表盘）
speed, soc, range_km, gear, outside_temp, inside_temp, odometer_km, charging, charge_power, voltage, ampere, power, latitude, longitude, heading

### detail.vue（车辆详情）
soc, range_km, charging, charge_power, charging_state, charge_speed, voltage, ampere, added_energy, inside_temp, outside_temp, is_ac_on, speed, gear, odometer_km, heading, locked, sentry_mode, door_fl/fr/rl/rr, frunk_open, trunk_open, tpms_fl/fr/rl/rr

### control.vue（控制页）
locked, is_ac_on, trunk_open, frunk_open, sentry_mode, charging, charge_port_door_open, inside_temp, seat_heater, steering_wheel_heater

### location.vue（地图页）
latitude, longitude, heading, speed, gear, driving, charging, soc, range_km

---

## 兼容映射说明

| Telemetry 字段 | → 前端旧字段 | 说明 |
|---------------|-------------|------|
| charger_voltage | voltage | 充电电压 |
| charge_amps | ampere | 充电电流 |
| charge_state | charging_state | 充电状态 |
| dc_charging_power + ac_charging_power | charge_power | 合并计算充电功率 |
| seat_heater_left/right/rear_* | seat_heater (object) | 组合为对象 |
| charge_state | charging (boolean) | 推导：Charging/Complete=true |
| gear | driving (boolean) | 推导：D/R/N=true |
| odometer (miles) | odometer_km | ×1.60934 转换 |
| ideal_battery_range (miles) | range_km | ×1.60934 转换 |
| charge_rate_mile_per_hour | charge_speed | ×1.60934 转换 |
