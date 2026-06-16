# Tesla Fleet Telemetry 字段映射文档

> 基于 `fleet-telemetry v0.9.0` proto 定义，共 261 个有效字段（不含 Field_Unknown=0）
> 显式映射 193 个，Semi-truck 专用 12 个跳过，Deprecated/Experimental 21 个跳过，其余由 default 分支自动处理

---

## 一、实时驾驶数据（realtime_update）

推送通道：`realtime_update` | 写入：`realtimeFields` | 频率：约 1 秒

### 1.1 行驶状态

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_VehicleSpeed | `speed` | float64 | × 1.60934 (mph→km/h) | 车速 |
| Field_Gear | `gear` | string | ShiftState 枚举→P/R/N/D | 挡位，未知挡位跳过 |
| Field_PedalPosition | `pedal_position` | float64 | 直接取值 | 油门踏板位置 (%) |
| Field_CruiseSetSpeed | `cruise_set_speed` | float64 | × 1.60934 (mph→km/h) | 巡航设定速度 |
| Field_LateralAcceleration | `lateral_acceleration` | float64 | 直接取值 | 横向加速度 (g) |
| Field_LongitudinalAcceleration | `longitudinal_acceleration` | float64 | 直接取值 | 纵向加速度 (g) |
| Field_BrakePedal | `brake_pedal` | bool | 直接取布尔值 | 刹车踏板是否踩下 |
| Field_DriveRail | `drive_rail` | bool | 直接取布尔值 | 驱动轨道状态 |

### 1.2 位置信息

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_Location | `latitude` | float64 | WGS84→GCJ02 坐标纠偏 | 纬度 |
| Field_Location | `longitude` | float64 | WGS84→GCJ02 坐标纠偏 | 经度 |
| Field_GpsHeading | `heading` | int | 直接取值 | GPS 航向角 |
| Field_GpsState | `gps_state` | int | 直接取值 | GPS 状态 |

### 1.3 电池/充电实时数据

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_Soc | `soc` | float64 | 直接取值 | 电池电量 (%) |
| Field_BatteryLevel | `battery_level` | float64 | 直接取值 | 电池等级 |
| Field_DCChargingPower | `dc_charging_power` | float64 | 直接取值 | 直流充电功率 (kW) |
| Field_ACChargingPower | `ac_charging_power` | float64 | 直接取值 | 交流充电功率 (kW) |
| Field_PackVoltage | `pack_voltage` | float64 | 直接取值 | 电池包电压 (V) |
| Field_PackCurrent | `pack_current` | float64 | 直接取值 | 电池包电流 (A) |
| Field_EnergyRemaining | `energy_remaining` | float64 | 直接取值 | 剩余能量 (kWh) |
| Field_ChargeAmps | `charge_amps` | float64 | 直接取值 | 充电电流 (A) |
| Field_ChargerVoltage | `charger_voltage` | float64 | 直接取值 | 充电器电压 (V) |
| Field_ChargeState | `charge_state` | string | ChargingState 枚举 | 充电状态：Disconnected/NoPower/Starting/Charging/Complete/Stopped |
| Field_DetailedChargeState | `charge_state` | string | DetailedChargeState 枚举 | 详细充电状态（覆盖 ChargeState） |
| Field_FastChargerPresent | `fast_charger_present` | bool | 直接取布尔值 | 是否连接快充 |

---

## 二、车辆状态数据（state_update）

推送通道：`state_update` | 写入：`stateFields` | 频率：变化时推送

### 2.1 车门/车窗/车锁

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_Locked | `locked` | bool | 直接取布尔值 | 车辆是否锁定 |
| Field_DoorState | `door_fl` | bool | DoorState 提取 | 左前门是否打开 |
| Field_DoorState | `door_fr` | bool | DoorState 提取 | 右前门是否打开 |
| Field_DoorState | `door_rl` | bool | DoorState 提取 | 左后门是否打开 |
| Field_DoorState | `door_rr` | bool | DoorState 提取 | 右后门是否打开 |
| Field_DoorState | `trunk_open` | bool | DoorState 提取 | 后备箱是否打开 |
| Field_DoorState | `frunk_open` | bool | DoorState 提取 | 前备箱是否打开 |
| Field_DoorState | `door_open` | bool | 四门任一打开为 true | 任意门是否打开 |
| Field_FdWindow | `fd_window` | bool | 直接取布尔值 | 左前窗是否开启 |
| Field_FpWindow | `fp_window` | bool | 直接取布尔值 | 右前窗是否开启 |
| Field_RdWindow | `rd_window` | bool | 直接取布尔值 | 左后窗是否开启 |
| Field_RpWindow | `rp_window` | bool | 直接取布尔值 | 右后窗是否开启 |
| Field_SentryMode | `sentry_mode` | bool | SentryModeState 枚举→非 Off/Unknown 为 true | 哨兵模式 |

### 2.2 空调/气候控制

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_InsideTemp | `inside_temp` | float64 | 直接取值 | 车内温度 (°C) |
| Field_OutsideTemp | `outside_temp` | float64 | 直接取值 | 车外温度 (°C) |
| Field_HvacPower | `hvac_power` | bool | HvacPowerState 枚举→非 Off/Unknown 为 true | 空调是否开启 |
| Field_HvacPower | `is_ac_on` | bool | 同 hvac_power | 空调是否开启（别名） |
| Field_HvacPower | `is_climate_on` | bool | 同 hvac_power | 气候控制是否开启（别名） |
| Field_HvacLeftTemperatureRequest | `driver_temp_setting` | float64 | 直接取值 | 驾驶员设定温度 (°C) |
| Field_HvacRightTemperatureRequest | `passenger_temp_setting` | float64 | 直接取值 | 副驾设定温度 (°C) |
| Field_HvacACEnabled | `hvac_ac_enabled` | bool | 直接取布尔值 | A/C 是否启用 |
| Field_HvacFanSpeed | `hvac_fan_speed` | int | 直接取值 | 风扇速度等级 |
| Field_HvacFanStatus | `hvac_fan_status` | int | 直接取值 | 风扇运行状态 |
| Field_HvacAutoMode | `hvac_auto_mode` | int | 直接取值 | 自动空调模式 |
| Field_ClimateKeeperMode | `climate_keeper_mode` | int | 直接取值 | 气候保持模式 |
| Field_DefrostMode | `defrost_mode` | int | 直接取值 | 除霜模式 |
| Field_DefrostForPreconditioning | `defrost_for_preconditioning` | bool | 直接取布尔值 | 预热除霜 |
| Field_RearDefrostEnabled | `rear_defrost_enabled` | bool | 直接取布尔值 | 后窗除霜 |
| Field_RearDisplayHvacEnabled | `rear_display_hvac_enabled` | bool | 直接取布尔值 | 后排空调显示启用 |
| Field_HvacSteeringWheelHeatLevel | `steering_wheel_heater` | bool | v > 0 为 true | 方向盘加热是否开启 |
| Field_HvacSteeringWheelHeatLevel | `hvac_steering_wheel_heat_level` | int | 直接取值 | 方向盘加热等级 |
| Field_HvacSteeringWheelHeatAuto | `hvac_steering_wheel_heat_auto` | bool | 直接取布尔值 | 方向盘加热自动 |
| Field_CabinOverheatProtectionMode | `cabin_overheat_protection_mode` | int | 直接取值 | 过热保护模式 |
| Field_CabinOverheatProtectionTemperatureLimit | `cabin_overheat_protection_temperature_limit` | int | 直接取值 | 过热保护温度限制 |
| Field_BatteryHeaterOn | `battery_heater_on` | bool | 直接取布尔值 | 电池加热器是否开启 |
| Field_NotEnoughPowerToHeat | `not_enough_power_to_heat` | bool | 直接取布尔值 | 电力不足无法加热 |

### 2.3 座椅

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_SeatHeaterLeft | `seat_heater_left` | int | 直接取值 | 左前座椅加热等级 (0=关) |
| Field_SeatHeaterRight | `seat_heater_right` | int | 直接取值 | 右前座椅加热等级 |
| Field_SeatHeaterRearLeft | `seat_heater_rear_left` | int | 直接取值 | 左后座椅加热等级 |
| Field_SeatHeaterRearRight | `seat_heater_rear_right` | int | 直接取值 | 右后座椅加热等级 |
| Field_SeatHeaterRearCenter | `seat_heater_rear_center` | int | 直接取值 | 后排中间座椅加热等级 |
| Field_RearSeatHeaters | `rear_seat_heaters` | int | 直接取值 | 后排座椅加热器数量 |
| Field_AutoSeatClimateLeft | `auto_seat_climate_left` | bool | 直接取布尔值 | 左前自动座椅气候 |
| Field_AutoSeatClimateRight | `auto_seat_climate_right` | bool | 直接取布尔值 | 右前自动座椅气候 |
| Field_ClimateSeatCoolingFrontLeft | `climate_seat_cooling_front_left` | int | 直接取值 | 左前座椅通风等级 |
| Field_ClimateSeatCoolingFrontRight | `climate_seat_cooling_front_right` | int | 直接取值 | 右前座椅通风等级 |
| Field_SeatVentEnabled | `seat_vent_enabled` | bool | 直接取布尔值 | 座椅通风启用 |
| Field_DriverSeatBelt | `driver_seat_belt` | bool | 直接取布尔值 | 驾驶员安全带 |
| Field_PassengerSeatBelt | `passenger_seat_belt` | bool | BuckleStatus 枚举→Latched 为 true | 副驾安全带 |
| Field_DriverSeatOccupied | `driver_seat_occupied` | bool | 直接取布尔值 | 驾驶员座是否有人 |

### 2.4 充电详情

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_ChargeLimitSoc | `charge_limit_soc` | int | 直接取值 | 充电限制 (%) |
| Field_ChargePortDoorOpen | `charge_port_door_open` | bool | 直接取布尔值 | 充电口盖是否打开 |
| Field_ChargePortLatch | `charge_port_latch` | string | ChargePortLatchValue 枚举 | 充电口锁：Disengaged/Engaged/Blocking |
| Field_ChargePort | `charge_port` | string | ChargePortValue 枚举 | 充电口类型：US/EU/GB/CCS |
| Field_ChargeRateMilePerHour | `charge_speed` | float64 | × 1.60934 (mph→km/h) | 充电速度 |
| Field_ChargeEnableRequest | `charge_enable_request` | bool | 直接取布尔值 | 充电使能请求 |
| Field_ChargeCurrentRequest | `charge_current_request` | int | 直接取值 | 请求充电电流 (A) |
| Field_ChargeCurrentRequestMax | `charge_current_request_max` | int | 直接取值 | 最大请求充电电流 (A) |
| Field_ChargerPhases | `charger_phases` | int | 直接取值 | 充电器相数 |
| Field_ChargePortColdWeatherMode | `charge_port_cold_weather_mode` | bool | 直接取布尔值 | 充电口寒冷天气模式 |
| Field_ChargingCableType | `charging_cable_type` | int | 直接取值 | 充电线缆类型 |
| Field_FastChargerType | `fast_charger_type` | int | 直接取值 | 快充类型 |
| Field_TimeToFullCharge | `time_to_full_charge` | float64 | 直接取值 | 充满预计时间 (小时) |
| Field_EstimatedHoursToChargeTermination | `minutes_to_full` | float64 | × 60 (小时→分钟) | 充满预计时间 (分钟) |
| Field_DCChargingEnergyIn | `dc_charging_energy_in` | float64 | 直接取值 | 直流充电累计能量 (kWh) |
| Field_ACChargingEnergyIn | `ac_charging_energy_in` | float64 | 直接取值 | 交流充电累计能量 (kWh) |
| Field_SuperchargerSessionTripPlanner | `supercharger_session_trip_planner` | bool | 直接取布尔值 | 超充行程规划 |

### 2.5 充电调度

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_ScheduledChargingStartTime | `scheduled_charging_start_time` | float64 | 直接取值 | 定时充电开始时间 |
| Field_ScheduledChargingPending | `scheduled_charging_pending` | bool | 直接取布尔值 | 定时充电是否等待中 |
| Field_ScheduledChargingMode | `scheduled_charging_mode` | string | ScheduledChargingModeValue 枚举 | 定时充电模式：Off/StartAt/DepartBy |
| Field_ScheduledDepartureTime | `scheduled_departure_time` | float64 | 直接取值 | 定时出发时间 |
| Field_PreconditioningEnabled | `preconditioning_enabled` | bool | 直接取布尔值 | 预处理启用 |

### 2.6 续航/里程

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_IdealBatteryRange | `range_km` | float64 | × 1.60934 (miles→km) | 理想续航里程 (km) |
| Field_EstBatteryRange | `range_km` | float64 | × 1.60934 (miles→km)，仅当 range_km 为 nil 时写入 | 估算续航里程 (km) |
| Field_RatedRange | `rated_range_km` | float64 | × 1.60934 (miles→km) | 额定续航里程 (km) |
| Field_Odometer | `odometer_km` | float64 | × 1.60934 (miles→km) | 总里程 (km) |
| Field_MilesSinceReset | `km_since_reset` | float64 | × 1.60934 (miles→km) | 重置后行驶里程 (km) |
| Field_SelfDrivingMilesSinceReset | `self_driving_km_since_reset` | float64 | × 1.60934 (miles→km) | 重置后自动驾驶里程 (km) |

### 2.7 胎压

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_TpmsPressureFl | `tpms_fl` | float64 | 直接取值 | 左前胎压 (bar) |
| Field_TpmsPressureFr | `tpms_fr` | float64 | 直接取值 | 右前胎压 (bar) |
| Field_TpmsPressureRl | `tpms_rl` | float64 | 直接取值 | 左后胎压 (bar) |
| Field_TpmsPressureRr | `tpms_rr` | float64 | 直接取值 | 右后胎压 (bar) |
| Field_TpmsLastSeenPressureTimeFl | `tpms_last_seen_pressure_time_fl` | float64 | 直接取值 | 左前胎压最后更新时间 |
| Field_TpmsLastSeenPressureTimeFr | `tpms_last_seen_pressure_time_fr` | float64 | 直接取值 | 右前胎压最后更新时间 |
| Field_TpmsLastSeenPressureTimeRl | `tpms_last_seen_pressure_time_rl` | float64 | 直接取值 | 左后胎压最后更新时间 |
| Field_TpmsLastSeenPressureTimeRr | `tpms_last_seen_pressure_time_rr` | float64 | 直接取值 | 右后胎压最后更新时间 |
| Field_TpmsHardWarnings | `tpms_hard_warnings` | bool | 直接取布尔值 | 胎压硬警告 |
| Field_TpmsSoftWarnings | `tpms_soft_warnings` | bool | 直接取布尔值 | 胎压软警告 |

### 2.8 电池诊断

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_BMSState | `bms_state` | int | 直接取值 | BMS 状态 |
| Field_BmsFullchargecomplete | `bms_full_charge_complete` | bool | 直接取布尔值 | BMS 充满完成 |
| Field_BrickVoltageMax | `brick_voltage_max` | int | 直接取值 | 最高电芯电压 |
| Field_BrickVoltageMin | `brick_voltage_min` | int | 直接取值 | 最低电芯电压 |
| Field_NumBrickVoltageMax | `num_brick_voltage_max` | int | 直接取值 | 最高电压电芯编号 |
| Field_NumBrickVoltageMin | `num_brick_voltage_min` | int | 直接取值 | 最低电压电芯编号 |
| Field_ModuleTempMax | `module_temp_max` | float64 | 直接取值 | 模块最高温度 (°C) |
| Field_ModuleTempMin | `module_temp_min` | float64 | 直接取值 | 模块最低温度 (°C) |
| Field_NumModuleTempMax | `num_module_temp_max` | int | 直接取值 | 最高温度模块编号 |
| Field_NumModuleTempMin | `num_module_temp_min` | int | 直接取值 | 最低温度模块编号 |
| Field_IsolationResistance | `isolation_resistance` | float64 | 直接取值 | 绝缘电阻 |
| Field_Hvil | `hvil` | int | 直接取值 | HVIL 状态 |
| Field_DCDCEnable | `dcdc_enable` | bool | 直接取布尔值 | DC-DC 转换器启用 |
| Field_LifetimeEnergyUsed | `lifetime_energy_used` | float64 | 直接取值 | 终身累计能耗 (kWh) |
| Field_LifetimeEnergyGainedRegen | `lifetime_energy_gained_regen` | float64 | 直接取值 | 终身再生制动回收能量 (kWh) |

### 2.9 灯光

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_LightsHazardsActive | `lights_hazards_active` | bool | 直接取布尔值 | 危险警示灯 |
| Field_LightsHighBeams | `lights_high_beams` | bool | 直接取布尔值 | 远光灯 |
| Field_LightsTurnSignal | `lights_turn_signal` | string | TurnSignalState 枚举 | 转向灯：left/right/off |
| Field_WiperHeatEnabled | `wiper_heat_enabled` | bool | 直接取布尔值 | 雨刮加热 |

### 2.10 安全/驾驶辅助

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_SpeedLimitMode | `speed_limit_mode` | bool | 直接取布尔值 | 限速模式 |
| Field_CurrentLimitMph | `current_limit_mph` | float64 | 直接取值 | 当前限速 (mph) |
| Field_SpeedLimitWarning | `speed_limit_warning` | string | SpeedAssistLevel 枚举 | 限速警告：None/Display/Chime |
| Field_CruiseFollowDistance | `cruise_follow_distance` | int | 直接取值 | 巡航跟车距离 |
| Field_AutomaticBlindSpotCamera | `automatic_blind_spot_camera` | bool | 直接取布尔值 | 自动盲区摄像头 |
| Field_BlindSpotCollisionWarningChime | `blind_spot_collision_warning_chime` | bool | 直接取布尔值 | 盲区碰撞警告提示音 |
| Field_ForwardCollisionWarning | `forward_collision_warning` | int | 直接取值 | 前方碰撞警告 |
| Field_LaneDepartureAvoidance | `lane_departure_avoidance` | int | 直接取值 | 车道偏离避让 |
| Field_EmergencyLaneDepartureAvoidance | `emergency_lane_departure_avoidance` | bool | 直接取布尔值 | 紧急车道偏离避让 |
| Field_AutomaticEmergencyBrakingOff | `automatic_emergency_braking_off` | bool | 直接取布尔值 | 自动紧急制动关闭 |
| Field_BrakePedalPos | `brake_pedal_pos` | float64 | 直接取值 | 刹车踏板位置 (%) |

### 2.11 安全/模式

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_PinToDriveEnabled | `pin_to_drive_enabled` | bool | 直接取布尔值 | PIN 驾驶启用 |
| Field_ValetModeEnabled | `valet_mode_enabled` | bool | 直接取布尔值 | 代客泊车模式 |
| Field_GuestModeEnabled | `guest_mode_enabled` | bool | 直接取布尔值 | 访客模式 |
| Field_GuestModeMobileAccessState | `guest_mode_mobile_access_state` | string | GuestModeMobileAccess 枚举→.String() | 访客模式移动访问状态 |
| Field_ServiceMode | `service_mode` | bool | 直接取布尔值 | 服务模式 |
| Field_RemoteStartEnabled | `remote_start_enabled` | bool | 直接取布尔值 | 远程启动启用 |
| Field_PairedPhoneKeyAndKeyFobQty | `paired_phone_key_and_key_fob_qty` | int | 直接取值 | 配对钥匙数量 |
| Field_HomelinkNearby | `homelink_nearby` | bool | 直接取布尔值 | Homelink 附近 |
| Field_HomelinkDeviceCount | `homelink_device_count` | int | 直接取值 | Homelink 设备数量 |

### 2.12 导航/行程

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_DestinationLocation | `destination_latitude` | float64 | GetLocationValue | 目的地纬度 |
| Field_DestinationLocation | `destination_longitude` | float64 | GetLocationValue | 目的地经度 |
| Field_DestinationName | `destination_name` | string | 直接取字符串 | 目的地名称 |
| Field_OriginLocation | `origin_latitude` | float64 | GetLocationValue | 出发地纬度 |
| Field_OriginLocation | `origin_longitude` | float64 | GetLocationValue | 出发地经度 |
| Field_MilesToArrival | `km_to_arrival` | float64 | × 1.60934 (miles→km) | 到达剩余里程 (km) |
| Field_MinutesToArrival | `minutes_to_arrival` | float64 | 直接取值 | 到达剩余时间 (分钟) |
| Field_RouteLastUpdated | `route_last_updated` | float64 | 直接取值 | 路线更新时间 |
| Field_ExpectedEnergyPercentAtTripArrival | `expected_energy_percent_at_arrival` | float64 | 直接取值 | 到达时预期电量 (%) |
| Field_RouteTrafficMinutesDelay | `route_traffic_minutes_delay` | float64 | 直接取值 | 路线交通延迟 (分钟) |

### 2.13 地理围栏

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_LocatedAtHome | `located_at_home` | bool | 直接取布尔值 | 是否在家 |
| Field_LocatedAtWork | `located_at_work` | bool | 直接取布尔值 | 是否在工作地点 |
| Field_LocatedAtFavorite | `located_at_favorite` | bool | 直接取布尔值 | 是否在收藏地点 |

### 2.14 车辆信息

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_VehicleName | `vehicle_name` | string | 直接取字符串 | 车辆名称 |
| Field_CarType | `car_type` | string | 直接取字符串 | 车型 |
| Field_Trim | `trim` | string | 直接取字符串 | 内饰配置 |
| Field_ExteriorColor | `exterior_color` | string | 直接取字符串 | 外观颜色 |
| Field_RoofColor | `roof_color` | string | 直接取字符串 | 车顶颜色 |
| Field_WheelType | `wheel_type` | string | 直接取字符串 | 轮毂类型 |
| Field_EfficiencyPackage | `efficiency_package` | string | 直接取字符串 | 效率包 |
| Field_Version | `version` | string | 直接取字符串 | 软件版本 |
| Field_EuropeVehicle | `europe_vehicle` | bool | 直接取布尔值 | 是否欧洲车辆 |
| Field_RightHandDrive | `right_hand_drive` | bool | 直接取布尔值 | 是否右舵 |
| Field_CenterDisplay | `center_display_state` | int | 直接取值 | 中控屏状态 |

### 2.15 Powershare (V2G/V2L)

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_PowershareStatus | `powershare_status` | string | PowershareState 枚举→.String() | Powershare 状态 |
| Field_PowershareType | `powershare_type` | string | PowershareTypeStatus 枚举→.String() | Powershare 类型 |
| Field_PowershareStopReason | `powershare_stop_reason` | string | PowershareStopReasonStatus 枚举→.String() | Powershare 停止原因 |
| Field_PowershareHoursLeft | `powershare_hours_left` | float64 | 直接取值 | Powershare 剩余小时 |
| Field_PowershareInstantaneousPowerKW | `powershare_instantaneous_power_kw` | float64 | 直接取值 | Powershare 瞬时功率 (kW) |

### 2.16 软件更新

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_SoftwareUpdateVersion | `software_update_version` | string | 直接取字符串 | 软件更新版本号 |
| Field_SoftwareUpdateDownloadPercentComplete | `software_update_download_percent` | float64 | 直接取值 | 下载进度 (%) |
| Field_SoftwareUpdateInstallationPercentComplete | `software_update_installation_percent` | float64 | 直接取值 | 安装进度 (%) |
| Field_SoftwareUpdateExpectedDurationMinutes | `software_update_expected_duration_minutes` | float64 | 直接取值 | 预期安装时长 (分钟) |
| Field_SoftwareUpdateScheduledStartTime | `software_update_scheduled_start_time` | float64 | 直接取值 | 计划安装时间 |

### 2.17 Cybertruck 专用

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_OffroadLightbarPresent | `offroad_lightbar_present` | bool | 直接取布尔值 | 越野灯条 |
| Field_TonneauOpenPercent | `tonneau_open_percent` | float64 | 直接取值 | 后盖打开百分比 |
| Field_TonneauPosition | `tonneau_position` | string | TonneauPositionState 枚举→.String() | 后盖位置 |
| Field_TonneauTentMode | `tonneau_tent_mode` | string | TonneauTentModeState 枚举→.String() | 后盖帐篷模式 |
| Field_SunroofInstalled | `sunroof_installed` | string | SunroofInstalledState 枚举→.String() | 天窗安装状态 |

### 2.18 用户偏好设置

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_SettingDistanceUnit | `setting_distance_unit` | string | DistanceUnit 枚举→.String() | 距离单位 |
| Field_SettingTemperatureUnit | `setting_temperature_unit` | string | TemperatureUnit 枚举→.String() | 温度单位 |
| Field_Setting24HourTime | `setting_24_hour_time` | bool | 直接取布尔值 | 24 小时制 |
| Field_SettingTirePressureUnit | `setting_tire_pressure_unit` | string | PressureUnit 枚举→.String() | 胎压单位 |
| Field_SettingChargeUnit | `setting_charge_unit` | string | ChargeUnitPreference 枚举→.String() | 充电单位 |

### 2.19 驱动逆变器诊断

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_DiStateR | `di_state_r` | string | DriveInverterState 枚举→.String() | 后驱逆变器状态 |
| Field_DiStateF | `di_state_f` | string | DriveInverterState 枚举→.String() | 前驱逆变器状态 |
| Field_DiStateREL | `di_state_rel` | string | DriveInverterState 枚举→.String() | 后左逆变器状态 |
| Field_DiStateRER | `di_state_rer` | string | DriveInverterState 枚举→.String() | 后右逆变器状态 |
| Field_DiHeatsinkTR | `di_heatsink_t_r` | float64 | 直接取值 | 后驱散热器温度 (°C) |
| Field_DiHeatsinkTF | `di_heatsink_t_f` | float64 | 直接取值 | 前驱散热器温度 (°C) |
| Field_DiHeatsinkTREL | `di_heatsink_t_rel` | float64 | 直接取值 | 后左散热器温度 (°C) |
| Field_DiHeatsinkTRER | `di_heatsink_t_rer` | float64 | 直接取值 | 后右散热器温度 (°C) |
| Field_DiAxleSpeedR | `di_axle_speed_r` | float64 | 直接取值 | 后驱轴速 (rpm) |
| Field_DiAxleSpeedF | `di_axle_speed_f` | float64 | 直接取值 | 前驱轴速 (rpm) |
| Field_DiAxleSpeedREL | `di_axle_speed_rel` | float64 | 直接取值 | 后左轴速 (rpm) |
| Field_DiAxleSpeedRER | `di_axle_speed_rer` | float64 | 直接取值 | 后右轴速 (rpm) |
| Field_DiTorquemotor | `di_torque_motor` | float64 | 直接取值 | 逆变器扭矩指令 (Nm) |
| Field_DiSlaveTorqueCmd | `di_slave_torque_cmd` | float64 | 直接取值 | 从驱扭矩指令 (Nm) |
| Field_DiTorqueActualR | `di_torque_actual_r` | float64 | 直接取值 | 后驱实际扭矩 (Nm) |
| Field_DiTorqueActualF | `di_torque_actual_f` | float64 | 直接取值 | 前驱实际扭矩 (Nm) |
| Field_DiTorqueActualREL | `di_torque_actual_rel` | float64 | 直接取值 | 后左实际扭矩 (Nm) |
| Field_DiTorqueActualRER | `di_torque_actual_rer` | float64 | 直接取值 | 后右实际扭矩 (Nm) |
| Field_DiStatorTempR | `di_stator_temp_r` | float64 | 直接取值 | 后驱定子温度 (°C) |
| Field_DiStatorTempF | `di_stator_temp_f` | float64 | 直接取值 | 前驱定子温度 (°C) |
| Field_DiStatorTempREL | `di_stator_temp_rel` | float64 | 直接取值 | 后左定子温度 (°C) |
| Field_DiStatorTempRER | `di_stator_temp_rer` | float64 | 直接取值 | 后右定子温度 (°C) |
| Field_DiVBatR | `di_vbat_r` | float64 | 直接取值 | 后驱电池电压 (V) |
| Field_DiVBatF | `di_vbat_f` | float64 | 直接取值 | 前驱电池电压 (V) |
| Field_DiVBatREL | `di_vbat_rel` | float64 | 直接取值 | 后左电池电压 (V) |
| Field_DiVBatRER | `di_vbat_rer` | float64 | 直接取值 | 后右电池电压 (V) |
| Field_DiMotorCurrentR | `di_motor_current_r` | float64 | 直接取值 | 后驱电机电流 (A) |
| Field_DiMotorCurrentF | `di_motor_current_f` | float64 | 直接取值 | 前驱电机电流 (A) |
| Field_DiMotorCurrentREL | `di_motor_current_rel` | float64 | 直接取值 | 后左电机电流 (A) |
| Field_DiMotorCurrentRER | `di_motor_current_rer` | float64 | 直接取值 | 后右电机电流 (A) |
| Field_DiInverterTR | `di_inverter_t_r` | float64 | 直接取值 | 后驱逆变器温度 (°C) |
| Field_DiInverterTF | `di_inverter_t_f` | float64 | 直接取值 | 前驱逆变器温度 (°C) |
| Field_DiInverterTREL | `di_inverter_t_rel` | float64 | 直接取值 | 后左逆变器温度 (°C) |
| Field_DiInverterTRER | `di_inverter_t_rer` | float64 | 直接取值 | 后右逆变器温度 (°C) |

---

## 三、媒体数据（media_state）

推送通道：`media_state` | 写入：`mediaFields` | 频率：变化时推送

| Proto 字段 | 前端 key | 类型 | 转换规则 | 说明 |
|---|---|---|---|---|
| Field_MediaPlaybackStatus | `media_playback_status` | string | MediaStatus 枚举→Stopped/Playing/Paused | 播放状态 |
| Field_MediaPlaybackSource | `media_audio_source` | string | 直接取字符串 | 音频来源 |
| Field_MediaAudioVolume | `media_volume` | int | 直接取值 | 音量 |
| Field_MediaAudioVolumeIncrement | `media_audio_volume_increment` | int | 直接取值 | 音量步进 |
| Field_MediaAudioVolumeMax | `media_audio_volume_max` | int | 直接取值 | 最大音量 |
| Field_MediaNowPlayingTitle | `now_playing_title` | string | 直接取字符串 | 当前曲目名 |
| Field_MediaNowPlayingArtist | `now_playing_artist` | string | 直接取字符串 | 当前艺术家 |
| Field_MediaNowPlayingAlbum | `now_playing_album` | string | 直接取字符串 | 当前专辑 |
| Field_MediaNowPlayingStation | `now_playing_station` | string | 直接取字符串 | 电台名称 |
| Field_MediaNowPlayingDuration | `now_playing_duration` | int | 直接取值 | 曲目总时长 (秒) |
| Field_MediaNowPlayingElapsed | `now_playing_elapsed` | int | 直接取值 | 已播放时长 (秒) |

---

## 四、未映射字段

### 4.1 Semi-truck 专用（不需要映射）

| Proto 字段 | 编号 | 说明 |
|---|---|---|
| Field_SemitruckTpmsPressureRe1L0 | 73 | 半挂车后轴1左外胎压 |
| Field_SemitruckTpmsPressureRe1L1 | 74 | 半挂车后轴1左内胎压 |
| Field_SemitruckTpmsPressureRe1R0 | 75 | 半挂车后轴1右外胎压 |
| Field_SemitruckTpmsPressureRe1R1 | 76 | 半挂车后轴1右内胎压 |
| Field_SemitruckTpmsPressureRe2L0 | 77 | 半挂车后轴2左外胎压 |
| Field_SemitruckTpmsPressureRe2L1 | 78 | 半挂车后轴2左内胎压 |
| Field_SemitruckTpmsPressureRe2R0 | 79 | 半挂车后轴2右外胎压 |
| Field_SemitruckTpmsPressureRe2R1 | 80 | 半挂车后轴2右内胎压 |
| Field_SemitruckPassengerSeatFoldPosition | 97 | 半挂车副驾折叠位置 |
| Field_LifetimeEnergyUsedDrive | 103 | 半挂车驱动终身能耗 |
| Field_SemitruckTractorParkBrakeStatus | 104 | 半挂车牵引车驻车制动 |
| Field_SemitruckTrailerParkBrakeStatus | 105 | 半挂车挂车驻车制动 |

### 4.2 Deprecated/Experimental（不需要映射）

| Proto 字段 | 编号 | 说明 |
|---|---|---|
| Field_Deprecated_1 | 162 | 已废弃 |
| Field_Deprecated_2 | 100 | 已废弃 |
| Field_Deprecated_3 | 257 | 已废弃 |
| Field_Experimental_1 ~ Experimental_15 | 119~178 | 实验性字段（15个） |

### 4.3 default 分支自动处理

未在上述列表中显式映射的字段，由 `default` 分支自动处理：
- 以 `toSnakeCase(Field名去前缀)` 作为 key
- 按 proto Value 类型分发：BooleanValue→bool, DoubleValue/FloatValue→float64, IntValue→int, LongValue→int64, StringValue→string
- 未知类型跳过
- 统一写入 `stateFields`

---

## 五、单位转换汇总

| 原始单位 | 目标单位 | 乘数 | 涉及字段 |
|---|---|---|---|
| mph | km/h | 1.60934 | VehicleSpeed, CruiseSetSpeed, ChargeRateMilePerHour, CurrentLimitMph |
| miles | km | 1.60934 | Odometer, IdealBatteryRange, EstBatteryRange, RatedRange, MilesToArrival, MilesSinceReset, SelfDrivingMilesSinceReset |
| hours | minutes | 60 | EstimatedHoursToChargeTermination |
| WGS84 | GCJ02 | 坐标纠偏 | Location (latitude/longitude) |

---

## 六、枚举转换汇总

| Proto 枚举 | 转换结果 | 涉及字段 |
|---|---|---|
| ShiftState | P/R/N/D (未知跳过) | Gear |
| ChargingState | Disconnected/NoPower/Starting/Charging/Complete/Stopped | ChargeState |
| DetailedChargeStateValue | 同 ChargingState | DetailedChargeState |
| SentryModeState | bool (非Off/Unknown为true) | SentryMode |
| HvacPowerState | bool (非Off/Unknown为true) | HvacPower |
| ChargePortLatchValue | Disengaged/Engaged/Blocking/Unknown | ChargePortLatch |
| ChargePortValue | US/EU/GB/CCS/Unknown | ChargePort |
| TurnSignalState | left/right/off | LightsTurnSignal |
| BuckleStatus | bool (Latched为true) | PassengerSeatBelt |
| ScheduledChargingModeValue | Off/StartAt/DepartBy/Unknown | ScheduledChargingMode |
| SpeedAssistLevel | None/Display/Chime/Unknown | SpeedLimitWarning |
| MediaStatus | Stopped/Playing/Paused/Unknown | MediaPlaybackStatus |
| DriveInverterState | .String() | DiStateR/F/REL/RER |
| PowershareState | .String() | PowershareStatus |
| PowershareTypeStatus | .String() | PowershareType |
| PowershareStopReasonStatus | .String() | PowershareStopReason |
| TonneauPositionState | .String() | TonneauPosition |
| TonneauTentModeState | .String() | TonneauTentMode |
| SunroofInstalledState | .String() | SunroofInstalled |
| GuestModeMobileAccess | .String() | GuestModeMobileAccessState |
| DistanceUnit | .String() | SettingDistanceUnit |
| TemperatureUnit | .String() | SettingTemperatureUnit |
| PressureUnit | .String() | SettingTirePressureUnit |
| ChargeUnitPreference | .String() | SettingChargeUnit |

---

## 七、数据流架构

```
特斯拉车辆
  ├── Fleet REST API (轮询 5-30s) ──→ ExtractRealtimeFromSimple ──→ ws.BroadcastRealtimeUpdate
  │                                  → ExtractStateFromSimple ──→ ws.BroadcastStateUpdate
  │                                  → ExtractMediaFromSimple ──→ ws.BroadcastMediaUpdate
  │                                  → 完整数据 ──→ ws.BroadcastVehicleState + 状态引擎
  │
  └── Fleet Telemetry (推送 ~1s) ──→ processProtobufTelemetry
                                       → realtimeFields ──→ updateRealtimeFields ──→ Redis + WebSocket + 状态引擎
                                       → stateFields ──→ updateVehicleStateFields ──→ Redis + WebSocket + 状态引擎
                                       → mediaFields ──→ updateMediaFields ──→ Redis + WebSocket + 状态引擎

前端 (vehicle-data.js)
  ├── onWSRealtimeUpdate → mergeRealtime (EMA平滑 + 写入realtime+state)
  ├── onWSStateUpdate → mergeState (写入state)
  ├── onWSMediaState → 写入data + state
  ├── onWSVehicleState → mergeData (完整数据)
  └── rebuildMergedData → 合并 data + state + realtime → 最终展示数据
```

### 零值语义保障

| 场景 | 遥测处理 | Fleet API 处理 |
|---|---|---|
| speed=0（车停了） | 推送0，前端接受 | speed:null→Go解析为0→不推送，不覆盖 |
| gear 未变化 | 不推送，前端保留上次值 | shift_state:null→Go解析为""→不推送，不覆盖 |
| locked=false | 推送false，前端接受 | 推送false，前端接受 |
| seat_heater=0 | 推送0，前端接受 | 推送0，前端接受 |
| charger_voltage=0 | 推送0，前端接受 | 不推送，避免覆盖遥测充电数据 |
