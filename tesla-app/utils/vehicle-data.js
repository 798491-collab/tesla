import { reactive } from 'vue'
import { getVehicleState } from '@/api/vehicle.js'
import { startSimulator, stopSimulator, isSimulatorMode, onBLEData, BLEState } from './ble.js'
import { wsConnect, wsDisconnect, wsOn, wsOff, wsSwitchVIN, wsIsConnected } from './websocket.js'

// 需要在遥测流中"持久化"的字段（增量推送，且过期后不需要被擦除的字段）
// 这些字段会同时写入 state 层，确保遥测过期后仍保留在 data 中
const PERSISTENT_TELEMETRY_FIELDS = new Set([
  // 驾驶状态
  'gear', 'latitude', 'longitude', 'heading',
  // 电池核心
  'soc', 'battery_level', 'usable_soc', 'range_km', 'battery_temp',
  'energy_remaining', 'odometer_km', 'rated_range_km',
  // 充电状态
  'charge_state', 'charging_state', 'charge_power',
  'dc_charging_power', 'ac_charging_power', 'charge_amps', 'charger_voltage',
  'fast_charger_present', 'fast_charger_type', 'supercharging',
  'charge_limit_soc', 'charge_speed', 'added_energy', 'charge_energy_added',
  'minutes_to_full', 'time_to_full_charge', 'charger_phases',
  'charge_port_door_open', 'charge_port_latch', 'charge_port_open',
  'charge_current_request', 'charge_current_request_max',
  'dc_charging_energy_in', 'ac_charging_energy_in',
  'charge_port_cold_weather_mode', 'charge_enable_request',
  // 电池健康
  'module_temp_max', 'module_temp_min', 'num_module_temp_max', 'num_module_temp_min',
  'brick_voltage_max', 'brick_voltage_min', 'num_brick_voltage_max', 'num_brick_voltage_min',
  'battery_heater_on', 'bms_state', 'bms_full_charge_complete',
  'dcdc_enable', 'isolation_resistance',
  'lifetime_energy_used', 'preconditioning_enabled',
  'pack_voltage', 'pack_current',
  // 温控
  'inside_temp', 'outside_temp',
  'driver_temp_setting', 'passenger_temp_setting', 'hvac_fan_speed',
  'steering_wheel_heater', 'is_ac_on', 'is_climate_on',
  // 车辆状态
  'locked', 'sentry_mode', 'voltage', 'ampere'
])

const vehicleStore = reactive({
  realtime: {},
  state: {},
  data: {},
  stateOutput: null,
  source: 'ws', // 'ws' | 'cloud' | 'ble'
  realtimeSource: null,
  realtimeUpdatedAt: 0,
  bleConnected: false,
  bleScanning: false,
  bleState: 'idle',
  loading: false,
  error: null,
  vin: '',
  pollState: 'sleeping',
  commandState: 'idle',
  lastCommand: '',
  commandLatencyMs: 0,
  analysisNotification: null
})

let bleUnsubscribe = null
let realtimeExpiryTimer = null

// 优化：使用节流/时间戳比对思想，避免高频创建销毁定时器
function scheduleRealtimeExpiry() {
  if (realtimeExpiryTimer) return // 如果定时器已经在跑了，不要动它，让它自己到期检测

  realtimeExpiryTimer = setTimeout(() => {
    const age = Date.now() - vehicleStore.realtimeUpdatedAt
    if (age >= 9500) {
      // 确实超时了，触发重新合并（擦除过期实时数据）
      rebuildMergedData()
      realtimeExpiryTimer = null
    } else {
      // 期间有新数据进来了，重新计算剩余时间继续等
      realtimeExpiryTimer = null
      scheduleRealtimeExpiry()
    }
  }, 9500)
}

export function useVehicleData() { return vehicleStore }

export function initVehicleData(vin) {
  vehicleStore.vin = vin
  if (vehicleStore.source === 'ble' && vehicleStore.bleConnected) return
  if (wsIsConnected() && vehicleStore.vin === vin) {
    fetchInitialState(vin)
    return
  }
  startWSStream(vin)
  fetchInitialState(vin)
}

export function destroyVehicleData() {
  stopWSStream()
  if (realtimeExpiryTimer) { clearTimeout(realtimeExpiryTimer); realtimeExpiryTimer = null }
  if (bleUnsubscribe) { bleUnsubscribe(); bleUnsubscribe = null }
  stopSimulator()
}

export function suspendVehicleData() {
  if (realtimeExpiryTimer) { clearTimeout(realtimeExpiryTimer); realtimeExpiryTimer = null }
}

function mergeRealtime(partial) {
  if (!partial || typeof partial !== 'object') return
  
  vehicleStore.realtimeUpdatedAt = Date.now()

  const mapping = {
    // 驾驶
    speed: 'speed', gear: 'gear', power: 'power', pedal_position: 'pedal_position',
    brake_pedal: 'brake_pedal', drive_rail: 'drive_rail', cruise_set_speed: 'cruise_set_speed',
    lateral_acceleration: 'lateral_acceleration', longitudinal_acceleration: 'longitudinal_acceleration',
    latitude: 'latitude', longitude: 'longitude', heading: 'heading', gps_state: 'gps_state',
    // 电池核心
    soc: 'soc', battery_level: 'battery_level', usable_soc: 'usable_soc',
    range_km: 'range_km', battery_temp: 'battery_temp', energy_remaining: 'energy_remaining',
    odometer_km: 'odometer_km', rated_range_km: 'rated_range_km',
    // 充电
    dc_charging_power: 'dc_charging_power', ac_charging_power: 'ac_charging_power',
    pack_voltage: 'pack_voltage', pack_current: 'pack_current',
    charge_amps: 'charge_amps', charger_voltage: 'charger_voltage',
    charge_state: 'charge_state', fast_charger_present: 'fast_charger_present',
    fast_charger_type: 'fast_charger_type', supercharging: 'supercharging',
    charge_limit_soc: 'charge_limit_soc', charge_speed: 'charge_speed',
    added_energy: 'added_energy', charge_energy_added: 'charge_energy_added',
    minutes_to_full: 'minutes_to_full', time_to_full_charge: 'time_to_full_charge',
    charger_phases: 'charger_phases',
    charge_port_door_open: 'charge_port_door_open', charge_port_latch: 'charge_port_latch',
    charge_port_open: 'charge_port_open',
    charge_current_request: 'charge_current_request', charge_current_request_max: 'charge_current_request_max',
    dc_charging_energy_in: 'dc_charging_energy_in', ac_charging_energy_in: 'ac_charging_energy_in',
    charge_port_cold_weather_mode: 'charge_port_cold_weather_mode',
    charge_enable_request: 'charge_enable_request',
    // 电池健康
    module_temp_max: 'module_temp_max', module_temp_min: 'module_temp_min',
    num_module_temp_max: 'num_module_temp_max', num_module_temp_min: 'num_module_temp_min',
    brick_voltage_max: 'brick_voltage_max', brick_voltage_min: 'brick_voltage_min',
    num_brick_voltage_max: 'num_brick_voltage_max', num_brick_voltage_min: 'num_brick_voltage_min',
    battery_heater_on: 'battery_heater_on', bms_state: 'bms_state',
    bms_full_charge_complete: 'bms_full_charge_complete',
    dcdc_enable: 'dcdc_enable', isolation_resistance: 'isolation_resistance',
    lifetime_energy_used: 'lifetime_energy_used', preconditioning_enabled: 'preconditioning_enabled',
    // 温控
    inside_temp: 'inside_temp', outside_temp: 'outside_temp',
    driver_temp_setting: 'driver_temp_setting', passenger_temp_setting: 'passenger_temp_setting',
    hvac_fan_speed: 'hvac_fan_speed', steering_wheel_heater: 'steering_wheel_heater',
    is_ac_on: 'is_ac_on', is_climate_on: 'is_climate_on',
    // 车辆状态
    locked: 'locked', sentry_mode: 'sentry_mode',
    voltage: 'voltage', ampere: 'ampere'
  }

  for (const [srcKey, dstKey] of Object.entries(mapping)) {
    if (partial[srcKey] !== undefined && partial[srcKey] !== null) {
      vehicleStore.realtime[dstKey] = partial[srcKey]
      
      // 【关键修复】：只有特定需要持久化的增量流字段才写入 state，车速、定位等高频遥测绝不写入 state！
      if (PERSISTENT_TELEMETRY_FIELDS.has(dstKey)) {
        vehicleStore.state[dstKey] = partial[srcKey]
      }
    }
  }

  // 特殊字段订正
  if (partial.soc !== undefined) vehicleStore.realtime.soc = partial.soc
  if (partial.charge_state !== undefined) vehicleStore.realtime.charging_state = partial.charge_state
  if (partial.charger_voltage !== undefined) vehicleStore.realtime.voltage = partial.charger_voltage
  if (partial.charge_amps !== undefined) vehicleStore.realtime.ampere = partial.charge_amps

  if (partial.dc_charging_power !== undefined || partial.ac_charging_power !== undefined) {
    const dc = partial.dc_charging_power !== undefined ? partial.dc_charging_power : (vehicleStore.realtime.dc_charging_power || 0)
    const ac = partial.ac_charging_power !== undefined ? partial.ac_charging_power : (vehicleStore.realtime.ac_charging_power || 0)
    vehicleStore.realtime.charge_power = Math.round((dc + ac) * 10) / 10
  }

  if (partial.state_output) {
    vehicleStore.stateOutput = partial.state_output
  }

  rebuildMergedData()
  scheduleRealtimeExpiry()
}

function mergeState(partial) {
  if (!partial || typeof partial !== 'object') return
  if (partial.state_output) vehicleStore.stateOutput = partial.state_output
  
  for (const [key, value] of Object.entries(partial)) {
    if (key === 'state_output') continue
    if (value !== undefined && value !== null) {
      vehicleStore.state[key] = value
    }
  }
  rebuildMergedData()
}

// 三路数据融合核心管道
function rebuildMergedData() {
  // 如果当前是 BLE 模拟器模式，直接以 state 为准输出，切断 WS 遥测合并，防止闪烁
  if (vehicleStore.source === 'ble') {
    vehicleStore.data = { ...vehicleStore.state }
    return
  }

  // 1. 基础层：大包状态
  const merged = { ...vehicleStore.state }

  // 2. 遥测层：检查是否过期
  const realtimeAge = Date.now() - vehicleStore.realtimeUpdatedAt
  const realtimeFresh = vehicleStore.realtimeUpdatedAt > 0 && realtimeAge < 10000

  if (realtimeFresh) {
    for (const [key, value] of Object.entries(vehicleStore.realtime)) {
      if (value !== undefined && value !== null) {
        merged[key] = value // 覆盖 state 中的旧数据
      }
    }
  } else {
    // 遥测过期，清除瞬时状态字段（速度、功率、踏板等），但保留位置和航向（最后已知位置）
    const highFreqFields = ['speed', 'power', 'pedal_position', 'brake_pedal']
    highFreqFields.forEach(f => { delete merged[f] })
  }

  // 3. 衍生状态计算
  if (merged.charger_voltage !== undefined && merged.voltage === undefined) merged.voltage = merged.charger_voltage
  if (merged.charge_amps !== undefined && merged.ampere === undefined) merged.ampere = merged.charge_amps
  if (merged.charge_state !== undefined && merged.charging_state === undefined) merged.charging_state = merged.charge_state
  
  if ((merged.dc_charging_power !== undefined || merged.ac_charging_power !== undefined) && merged.charge_power === undefined) {
    merged.charge_power = Math.round(((merged.dc_charging_power || 0) + (merged.ac_charging_power || 0)) * 10) / 10
  }
  if (merged.charge_power !== undefined && merged.charge_power !== null) {
    merged.charge_power = Math.round(Number(merged.charge_power) * 10) / 10
  }

  // 组装 seat_heater 对象（后端发的是5个独立字段，前端部分页面期望对象）
  if (merged.seat_heater_left !== undefined || merged.seat_heater_right !== undefined ||
      merged.seat_heater_rear_left !== undefined || merged.seat_heater_rear_right !== undefined ||
      merged.seat_heater_rear_center !== undefined) {
    merged.seat_heater = {
      left: merged.seat_heater_left || 0,
      right: merged.seat_heater_right || 0,
      rear_left: merged.seat_heater_rear_left || 0,
      rear_right: merged.seat_heater_rear_right || 0,
      rear_center: merged.seat_heater_rear_center || 0
    }
  }

  const cs = merged.charging_state || merged.charge_state
  merged.charging = cs === 'Charging' || cs === 'Complete'
  merged.driving = ['D', 'R', 'N'].includes(merged.gear)

  if (vehicleStore.stateOutput) merged.state_output = vehicleStore.stateOutput

  vehicleStore.data = merged
  vehicleStore.error = null
}

async function fetchInitialState(vin) {
  if (!vin) return
  vehicleStore.loading = true
  try {
    const res = await getVehicleState(vin)
    const data = res.data || {}
    mergeState(data)
    if (!wsIsConnected()) {
      vehicleStore.source = 'cloud'
    }
    vehicleStore.error = null
  } catch (err) {
    vehicleStore.error = err.message
  } finally {
    vehicleStore.loading = false
  }
}

// ================= WS 事件回调群 =================
function startWSStream(vin) {
  stopWSStream()
  const events = [
    'vehicle_state', 'realtime_update', 'state_update', 'online_state',
    'poll_state', 'command_state', 'media_state', 'open', 'close',
    'trip_ended', 'charging_ended', 'analysis_complete'
  ]
  const handlers = [
    onWSVehicleState, onWSRealtimeUpdate, onWSStateUpdate, onWSOnlineState,
    onWSPollState, onWSCommandState, onWSMediaState, onWSOpen, onWSClose,
    onWSTripEnded, onWSChargingEnded, onWSAnalysisComplete
  ]
  events.forEach((ev, i) => wsOn(ev, handlers[i]))
  wsConnect(vin)
}

function stopWSStream() {
  const events = [
    'vehicle_state', 'realtime_update', 'state_update', 'online_state',
    'poll_state', 'command_state', 'media_state', 'open', 'close',
    'trip_ended', 'charging_ended', 'analysis_complete'
  ]
  const handlers = [
    onWSVehicleState, onWSRealtimeUpdate, onWSStateUpdate, onWSOnlineState,
    onWSPollState, onWSCommandState, onWSMediaState, onWSOpen, onWSClose,
    onWSTripEnded, onWSChargingEnded, onWSAnalysisComplete
  ]
  events.forEach((ev, i) => wsOff(ev, handlers[i]))
  wsDisconnect()
}

function onWSRealtimeUpdate(data) {
  if (vehicleStore.source === 'ble') return
  mergeRealtime(data)
  vehicleStore.realtimeSource = 'telemetry'
  vehicleStore.source = 'ws'
}

function onWSStateUpdate(data) {
  if (vehicleStore.source === 'ble') return
  mergeState(data)
  if (vehicleStore.realtimeSource !== 'telemetry') vehicleStore.source = 'ws'
}

function onWSVehicleState(data) {
  if (vehicleStore.source === 'ble') return
  mergeState(data)
  if (vehicleStore.source !== 'ble') vehicleStore.source = 'ws'
}

function onWSOnlineState(data) {
  if (data.online !== undefined) vehicleStore.state.online = data.online
  if (data.state !== undefined) vehicleStore.state.online_state = data.state
  if (data.state_output) vehicleStore.stateOutput = data.state_output
  if (vehicleStore.source !== 'ble') {
    rebuildMergedData()
    vehicleStore.source = 'ws'
  }
}

function onWSPollState(data) { if (data.poll_state) vehicleStore.pollState = data.poll_state }

function onWSCommandState(data) {
  if (data.command_state) vehicleStore.commandState = data.command_state
  if (data.last_command) vehicleStore.lastCommand = data.last_command
  if (data.latency_ms !== undefined) vehicleStore.commandLatencyMs = data.latency_ms
  if (['success', 'failed'].includes(data.command_state)) {
    setTimeout(() => {
      if (vehicleStore.commandState === data.command_state) vehicleStore.commandState = 'idle'
    }, 3000)
  }
}

function onWSMediaState(data) {
  mergeState(data)
}

function onWSTripEnded(data) { setNotification('trip_ended', data.ref_id, data.status, { tripId: data.trip_id }) }
function onWSChargingEnded(data) { setNotification('charging_ended', data.ref_id, data.status, { chargeId: data.charge_id }) }
function onWSAnalysisComplete(data) { setNotification('analysis_complete', data.ref_id, data.status, { analysisType: data.type }) }

function setNotification(type, refId, status, extra = {}) {
  vehicleStore.analysisNotification = { type, refId, status, timestamp: Date.now(), ...extra }
}

function onWSOpen() { if (vehicleStore.source !== 'ble') vehicleStore.source = 'ws' }
function onWSClose() {
  if (vehicleStore.source === 'ws') vehicleStore.source = 'cloud'
  vehicleStore.realtimeSource = null
}

// ================= BLE / 模拟器控制 =================
export function startSimulatorMode() {
  if (bleUnsubscribe) { bleUnsubscribe(); bleUnsubscribe = null }

  bleUnsubscribe = onBLEData((event) => {
    if (event.type === 'state_change') {
      vehicleStore.bleState = event.to
      vehicleStore.bleScanning = event.to === BLEState.SCANNING
    } else if (event.type === 'data') {
      // 模拟器全量覆盖到 state
      vehicleStore.state = { ...vehicleStore.state, ...event.data }
      vehicleStore.source = 'ble'
      vehicleStore.bleConnected = true
      rebuildMergedData()
    } else if (event.type === 'connect') {
      vehicleStore.bleConnected = true
      vehicleStore.source = 'ble'
    } else if (event.type === 'disconnect') {
      vehicleStore.bleConnected = false
      // 【关键修复】：智能判定降级策略，避免强行写死 ws 导致没网时卡死
      vehicleStore.source = wsIsConnected() ? 'ws' : 'cloud'
      rebuildMergedData()
    }
  })

  startSimulator()
  vehicleStore.source = 'ble'
  vehicleStore.bleConnected = true
}

export function stopSimulatorMode() {
  stopSimulator()
  vehicleStore.bleConnected = false
  vehicleStore.source = wsIsConnected() ? 'ws' : 'cloud'
  if (bleUnsubscribe) { bleUnsubscribe(); bleUnsubscribe = null }
  rebuildMergedData()
}

// 导出状态查询方法
export function getSource() { return vehicleStore.source }
export function isBLEMode() { return vehicleStore.source === 'ble' }
export function isCloudMode() { return vehicleStore.source === 'cloud' }
export function isWSMode() { return vehicleStore.source === 'ws' }
export function isTelemetryMode() { return vehicleStore.realtimeSource === 'telemetry' }