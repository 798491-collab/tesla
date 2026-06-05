import { reactive } from 'vue'
import { getVehicleState } from '@/api/vehicle.js'
import { startSimulator, stopSimulator, isSimulatorMode, onBLEData, BLEState } from './ble.js'
import { wsConnect, wsDisconnect, wsOn, wsOff, wsSwitchVIN, wsIsConnected } from './websocket.js'
import { getRefreshInterval } from './vehicle-state.js'

const EMA_ALPHA = 1.0
const EMA_ALPHA_FAST = 1.0
const EMA_FIELDS = ['speed', 'inside_temp', 'outside_temp', 'charge_power', 'range_km', 'charge_amps', 'soc']
const EMA_FAST_FIELDS = ['speed']

const L1_REALTIME_FIELDS = new Set([
  'speed', 'gear', 'power', 'pedal_position', 'brake_pedal', 'drive_rail', 'cruise_set_speed',
  'lateral_acceleration', 'longitudinal_acceleration',
  'latitude', 'longitude', 'heading', 'gps_state',
  'soc', 'battery_level', 'dc_charging_power', 'ac_charging_power',
  'pack_voltage', 'pack_current', 'energy_remaining',
  'charge_amps', 'charger_voltage', 'charge_state', 'fast_charger_present'
])

const L2_STATE_FIELDS = new Set([
  'locked', 'door_open', 'door_fl', 'door_fr', 'door_rl', 'door_rr',
  'trunk_open', 'frunk_open', 'windows_open',
  'fd_window', 'fp_window', 'rd_window', 'rp_window',
  'sentry_mode',
  'valet_mode_enabled', 'service_mode',
  'inside_temp', 'outside_temp', 'driver_temp_setting', 'passenger_temp_setting',
  'is_ac_on', 'is_climate_on', 'seat_heater_left', 'seat_heater_right',
  'seat_heater_rear_left', 'seat_heater_rear_right', 'seat_heater_rear_center',
  'steering_wheel_heater',
  'defrost_mode', 'hvac_power', 'hvac_ac_enabled', 'hvac_auto_mode', 'hvac_fan_speed',
  'climate_keeper_mode',
  'hvac_steering_wheel_heat_level', 'hvac_steering_wheel_heat_auto',
  'defrost_for_preconditioning', 'auto_seat_climate_left', 'auto_seat_climate_right',
  'climate_seat_cooling_front_left', 'climate_seat_cooling_front_right',
  'cabin_overheat_protection_mode', 'cabin_overheat_protection_temperature_limit',
  'seat_vent_enabled', 'rear_display_hvac_enabled', 'wiper_heat_enabled',
  'charge_port_door_open', 'charge_port_latch', 'charge_limit_soc', 'minutes_to_full',
  'charging_state', 'charge_speed', 'voltage', 'ampere', 'charge_power', 'added_energy',
  'fast_charger_type', 'battery_heater_on',
  'dc_charging_energy_in', 'ac_charging_energy_in',
  'charging_cable_type', 'charge_enable_request',
  'charge_current_request', 'charge_current_request_max', 'charger_phases',
  'charge_port_cold_weather_mode', 'bms_state', 'bms_full_charge_complete',
  'dcdc_enable', 'lifetime_energy_used',
  'module_temp_max', 'module_temp_min', 'num_module_temp_max', 'num_module_temp_min',
  'brick_voltage_max', 'brick_voltage_min', 'num_brick_voltage_max', 'num_brick_voltage_min',
  'preconditioning_enabled', 'not_enough_power_to_heat', 'supercharger_session_trip_planner',
  'tpms_fl', 'tpms_fr', 'tpms_rl', 'tpms_rr',
  'tpms_last_seen_pressure_time_fl', 'tpms_last_seen_pressure_time_fr',
  'tpms_last_seen_pressure_time_rl', 'tpms_last_seen_pressure_time_rr',
  'tpms_soft_warnings', 'tpms_hard_warnings',
  'car_type', 'trim', 'exterior_color', 'wheel_type',
  'version', 'vehicle_name', 'roof_color',
  'europe_vehicle', 'right_hand_drive',
  'efficiency_package', 'rear_seat_heaters', 'sunroof_installed',
  'remote_start_enabled',
  'odometer_km', 'range_km',
  'driver_seat_belt', 'driver_seat_occupied',
  'center_display_state', 'mirror_folded',
  'battery_temp', 'usable_soc',
  'lights_high_beams', 'lights_hazards_active', 'lights_turn_signal',
  'current_limit_mph', 'cruise_follow_distance',
  'automatic_blind_spot_camera', 'blind_spot_collision_warning_chime',
  'forward_collision_warning', 'lane_departure_avoidance',
  'emergency_lane_departure_avoidance', 'automatic_emergency_braking_off',
  'guest_mode_enabled',
  'destination_latitude', 'destination_longitude', 'destination_name',
  'powershare_status', 'powershare_type', 'powershare_instantaneous_power_kw',
  'powershare_hours_left', 'powershare_stop_reason',
  'software_update_version', 'software_update_download_percent',
  'software_update_expected_duration_minutes', 'software_update_installation_percent',
  'software_update_scheduled_start_time',
  'tonneau_position', 'tonneau_open_percent', 'tonneau_tent_mode', 'offroad_lightbar_present',
  'route_traffic_minutes_delay', 'miles_to_arrival', 'minutes_to_arrival',
  'expected_energy_percent_at_trip_arrival', 'located_at_home', 'located_at_work',
  'located_at_favorite', 'route_last_updated',
  'pin_to_drive_enabled', 'paired_phone_key_and_key_fob_qty',
  'passenger_seat_belt', 'miles_since_reset', 'self_driving_miles_since_reset',
  'isolation_resistance', 'hvil',
  'setting_24_hour_time', 'setting_charge_unit', 'setting_distance_unit',
  'setting_temperature_unit', 'setting_tire_pressure_unit',
  'media_audio_volume_increment', 'media_audio_volume_max',
])

const L1_REALTIME_WS_FIELDS = new Set([
  'speed', 'gear', 'power', 'pedal_position', 'brake_pedal', 'drive_rail', 'cruise_set_speed',
  'lateral_acceleration', 'longitudinal_acceleration',
  'latitude', 'longitude', 'heading', 'gps_state',
  'soc', 'battery_level', 'dc_charging_power', 'ac_charging_power',
  'pack_voltage', 'pack_current', 'energy_remaining',
  'charge_amps', 'charger_voltage', 'charge_state', 'fast_charger_present',
  'updated_at'
])

const vehicleStore = reactive({
  realtime: {},
  state: {},
  data: {},
  stateOutput: null,
  source: 'ws',
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

let fallbackTimer = null
let bleUnsubscribe = null
let emaState = {}

export function useVehicleData() {
  return vehicleStore
}

export function initVehicleData(vin) {
  vehicleStore.vin = vin
  if (vehicleStore.source === 'ble' && vehicleStore.bleConnected) {
    return
  }
  startWSStream(vin)
  fetchInitialState(vin)
}

export function destroyVehicleData() {
  stopWSStream()
  stopFallbackPolling()
  if (bleUnsubscribe) {
    bleUnsubscribe()
    bleUnsubscribe = null
  }
  stopSimulator()
}

function applyEMA(rawData) {
  const smoothed = { ...rawData }
  for (const field of EMA_FIELDS) {
    if (smoothed[field] !== undefined && smoothed[field] !== null) {
      const current = Number(smoothed[field])
      if (isNaN(current)) continue
      const alpha = EMA_FAST_FIELDS.includes(field) ? EMA_ALPHA_FAST : EMA_ALPHA
      if (emaState[field] === undefined) {
        emaState[field] = current
      } else {
        emaState[field] = emaState[field] * (1 - alpha) + current * alpha
      }
      if (field === 'speed') {
        smoothed[field] = Math.round(emaState[field] * 10) / 10
      } else if (field === 'soc') {
        smoothed[field] = Math.round(emaState[field])
      } else {
        smoothed[field] = Math.round(emaState[field] * 10) / 10
      }
    }
  }
  return smoothed
}

function mergeRealtime(partial) {
  if (!partial || typeof partial !== 'object') return
  const mapping = {
    speed: 'speed',
    gear: 'gear',
    power: 'power',
    pedal_position: 'pedal_position',
    brake_pedal: 'brake_pedal',
    drive_rail: 'drive_rail',
    cruise_set_speed: 'cruise_set_speed',
    lateral_acceleration: 'lateral_acceleration',
    longitudinal_acceleration: 'longitudinal_acceleration',
    latitude: 'latitude',
    longitude: 'longitude',
    heading: 'heading',
    gps_state: 'gps_state',
    soc: 'soc',
    battery_level: 'battery_level',
    dc_charging_power: 'dc_charging_power',
    ac_charging_power: 'ac_charging_power',
    pack_voltage: 'pack_voltage',
    pack_current: 'pack_current',
    energy_remaining: 'energy_remaining',
    charge_amps: 'charge_amps',
    charger_voltage: 'charger_voltage',
    charge_state: 'charge_state',
    fast_charger_present: 'fast_charger_present'
  }
  for (const [srcKey, dstKey] of Object.entries(mapping)) {
    if (partial[srcKey] !== undefined && partial[srcKey] !== null) {
      vehicleStore.realtime[dstKey] = partial[srcKey]
    }
  }
  if (partial.soc !== undefined) {
    vehicleStore.realtime.soc = partial.soc
  }
  if (partial.charge_state !== undefined) {
    vehicleStore.realtime.charging_state = partial.charge_state
  }
  if (partial.charger_voltage !== undefined) {
    vehicleStore.realtime.voltage = partial.charger_voltage
  }
  if (partial.charge_amps !== undefined) {
    vehicleStore.realtime.ampere = partial.charge_amps
  }
  if (partial.dc_charging_power !== undefined || partial.ac_charging_power !== undefined) {
    vehicleStore.realtime.charge_power = (partial.dc_charging_power || 0) + (partial.ac_charging_power || 0)
  }
  if (partial.updated_at) {
    vehicleStore.realtimeUpdatedAt = partial.updated_at
  }
  rebuildMergedData()
}

function mergeState(partial) {
  if (!partial || typeof partial !== 'object') return
  if (partial.state_output) {
    vehicleStore.stateOutput = partial.state_output
    delete partial.state_output
  }
  for (const [key, value] of Object.entries(partial)) {
    if (value !== undefined && value !== null) {
      vehicleStore.state[key] = value
    }
  }
  rebuildMergedData()
}

function mergeData(partial) {
  if (!partial || typeof partial !== 'object') return
  if (partial.state_output) {
    vehicleStore.stateOutput = partial.state_output
  }
  const { state_output, ...rest } = partial
  vehicleStore.data = { ...vehicleStore.data, ...rest }
  vehicleStore.error = null
}

function rebuildMergedData() {
  const merged = { ...vehicleStore.state }

  const realtimeAge = Date.now() - vehicleStore.realtimeUpdatedAt
  const realtimeFresh = vehicleStore.realtimeUpdatedAt > 0 && realtimeAge < 10000

  if (realtimeFresh) {
    for (const [key, value] of Object.entries(vehicleStore.realtime)) {
      if (value !== undefined && value !== null) {
        merged[key] = value
      }
    }
  }

  if (vehicleStore.data) {
    for (const [key, value] of Object.entries(vehicleStore.data)) {
      if (merged[key] === undefined || merged[key] === null) {
        merged[key] = value
      }
    }
  }

  if (merged.charger_voltage !== undefined && merged.voltage === undefined) {
    merged.voltage = merged.charger_voltage
  }
  if (merged.charge_amps !== undefined && merged.ampere === undefined) {
    merged.ampere = merged.charge_amps
  }
  if (merged.charge_state !== undefined && merged.charging_state === undefined) {
    merged.charging_state = merged.charge_state
  }
  if ((merged.dc_charging_power || merged.ac_charging_power) && !merged.charge_power) {
    merged.charge_power = (merged.dc_charging_power || 0) + (merged.ac_charging_power || 0)
  }
  if (merged.seat_heater_left !== undefined && !merged.seat_heater) {
    merged.seat_heater = {
      left: merged.seat_heater_left || 0,
      right: merged.seat_heater_right || 0,
      rear_left: merged.seat_heater_rear_left || 0,
      rear_right: merged.seat_heater_rear_right || 0,
    }
  }

  // Derive windows_open from individual window states if not already set
  if (merged.windows_open === undefined &&
      (merged.fd_window !== undefined || merged.fp_window !== undefined ||
       merged.rd_window !== undefined || merged.rp_window !== undefined)) {
    merged.windows_open = !!(merged.fd_window || merged.fp_window || merged.rd_window || merged.rp_window)
  }

  // Derive door_open from individual door states if not already set
  if (merged.door_open === undefined &&
      (merged.door_fl !== undefined || merged.door_fr !== undefined ||
       merged.door_rl !== undefined || merged.door_rr !== undefined)) {
    merged.door_open = !!(merged.door_fl || merged.door_fr || merged.door_rl || merged.door_rr)
  }

  // 始终根据 charging_state 推导 charging 布尔值，避免状态卡住
  const cs = merged.charging_state || merged.charge_state
  if (cs) {
    merged.charging = cs === 'Charging' || cs === 'Complete'
  } else if (merged.charging === undefined) {
    merged.charging = false
  }
  if (merged.driving === undefined) {
    const g = merged.gear
    merged.driving = g === 'D' || g === 'R' || g === 'N'
  }

  if (vehicleStore.stateOutput) {
    merged.state_output = vehicleStore.stateOutput
  }

  vehicleStore.data = merged
  vehicleStore.error = null
}

async function fetchInitialState(vin) {
  if (!vin) return
  vehicleStore.loading = true
  try {
    const res = await getVehicleState(vin)
    const data = res.data || {}
    const smoothed = applyEMA(data)
    if (!wsIsConnected()) {
      mergeData(smoothed)
      vehicleStore.source = 'cloud'
    } else {
      for (const key of Object.keys(vehicleStore.data)) {
        if (!(key in smoothed)) {
          smoothed[key] = vehicleStore.data[key]
        }
      }
      mergeData(smoothed)
    }
    vehicleStore.error = null
  } catch (err) {
    vehicleStore.error = err.message
  } finally {
    vehicleStore.loading = false
  }
}

function startWSStream(vin) {
  stopWSStream()

  wsOn('vehicle_state', onWSVehicleState)
  wsOn('realtime_update', onWSRealtimeUpdate)
  wsOn('state_update', onWSStateUpdate)
  wsOn('online_state', onWSOnlineState)
  wsOn('poll_state', onWSPollState)
  wsOn('command_state', onWSCommandState)
  wsOn('media_state', onWSMediaState)
  wsOn('open', onWSOpen)
  wsOn('close', onWSClose)
  wsOn('trip_ended', onWSTripEnded)
  wsOn('charging_ended', onWSChargingEnded)
  wsOn('analysis_complete', onWSAnalysisComplete)

  wsConnect(vin)
}

function stopWSStream() {
  wsOff('vehicle_state', onWSVehicleState)
  wsOff('realtime_update', onWSRealtimeUpdate)
  wsOff('state_update', onWSStateUpdate)
  wsOff('online_state', onWSOnlineState)
  wsOff('poll_state', onWSPollState)
  wsOff('command_state', onWSCommandState)
  wsOff('media_state', onWSMediaState)
  wsOff('open', onWSOpen)
  wsOff('close', onWSClose)
  wsOff('trip_ended', onWSTripEnded)
  wsOff('charging_ended', onWSChargingEnded)
  wsOff('analysis_complete', onWSAnalysisComplete)
  wsDisconnect()
}

function onWSRealtimeUpdate(data) {
  const smoothed = applyEMA(data)
  mergeRealtime(smoothed)
  vehicleStore.realtimeSource = 'telemetry'
  vehicleStore.source = 'ws'
  stopFallbackPolling()
}

function onWSStateUpdate(data) {
  mergeState(data)
  if (vehicleStore.realtimeSource !== 'telemetry') {
    vehicleStore.source = 'ws'
  }
}

function onWSVehicleState(data) {
  const smoothed = applyEMA(data)
  mergeData(smoothed)
  if (data.gear) vehicleStore.data.gear = data.gear
  if (data.charging !== undefined) vehicleStore.data.charging = data.charging
  if (data.locked !== undefined) vehicleStore.data.locked = data.locked
  if (data.door_open !== undefined) vehicleStore.data.door_open = data.door_open
  if (data.door_fl !== undefined) vehicleStore.data.door_fl = data.door_fl
  if (data.door_fr !== undefined) vehicleStore.data.door_fr = data.door_fr
  if (data.door_rl !== undefined) vehicleStore.data.door_rl = data.door_rl
  if (data.door_rr !== undefined) vehicleStore.data.door_rr = data.door_rr
  if (data.trunk_open !== undefined) vehicleStore.data.trunk_open = data.trunk_open
  if (data.frunk_open !== undefined) vehicleStore.data.frunk_open = data.frunk_open
  if (data.sentry_mode !== undefined) vehicleStore.data.sentry_mode = data.sentry_mode
  if (data.is_ac_on !== undefined) vehicleStore.data.is_ac_on = data.is_ac_on
  if (data.charge_port_door_open !== undefined) vehicleStore.data.charge_port_door_open = data.charge_port_door_open
  if (data.charge_port_latch !== undefined) vehicleStore.data.charge_port_latch = data.charge_port_latch
  if (data.windows_open !== undefined) vehicleStore.data.windows_open = data.windows_open
  if (data.driving !== undefined) vehicleStore.data.driving = data.driving

  // Window states
  if (data.fd_window !== undefined) vehicleStore.data.fd_window = data.fd_window
  if (data.fp_window !== undefined) vehicleStore.data.fp_window = data.fp_window
  if (data.rd_window !== undefined) vehicleStore.data.rd_window = data.rd_window
  if (data.rp_window !== undefined) vehicleStore.data.rp_window = data.rp_window

  // Mirror
  if (data.mirror_folded !== undefined) vehicleStore.data.mirror_folded = data.mirror_folded

  // Battery temp
  if (data.battery_temp !== undefined) vehicleStore.data.battery_temp = data.battery_temp

  // Seat heaters
  if (data.seat_heater_rear_left !== undefined) vehicleStore.data.seat_heater_rear_left = data.seat_heater_rear_left
  if (data.seat_heater_rear_right !== undefined) vehicleStore.data.seat_heater_rear_right = data.seat_heater_rear_right
  if (data.seat_heater_rear_center !== undefined) vehicleStore.data.seat_heater_rear_center = data.seat_heater_rear_center

  // Usable SOC
  if (data.usable_soc !== undefined) vehicleStore.data.usable_soc = data.usable_soc

  // Added energy
  if (data.added_energy !== undefined) vehicleStore.data.added_energy = data.added_energy

  // HVAC details
  if (data.hvac_power !== undefined) vehicleStore.data.hvac_power = data.hvac_power
  if (data.hvac_ac_enabled !== undefined) vehicleStore.data.hvac_ac_enabled = data.hvac_ac_enabled
  if (data.hvac_auto_mode !== undefined) vehicleStore.data.hvac_auto_mode = data.hvac_auto_mode
  if (data.hvac_fan_speed !== undefined) vehicleStore.data.hvac_fan_speed = data.hvac_fan_speed
  if (data.driver_temp_setting !== undefined) vehicleStore.data.driver_temp_setting = data.driver_temp_setting
  if (data.passenger_temp_setting !== undefined) vehicleStore.data.passenger_temp_setting = data.passenger_temp_setting
  if (data.defrost_mode !== undefined) vehicleStore.data.defrost_mode = data.defrost_mode
  if (data.steering_wheel_heater !== undefined) vehicleStore.data.steering_wheel_heater = data.steering_wheel_heater
  if (data.climate_keeper_mode !== undefined) vehicleStore.data.climate_keeper_mode = data.climate_keeper_mode

  // Charge details
  if (data.charge_limit_soc !== undefined) vehicleStore.data.charge_limit_soc = data.charge_limit_soc
  if (data.minutes_to_full !== undefined) vehicleStore.data.minutes_to_full = data.minutes_to_full
  if (data.fast_charger_type !== undefined) vehicleStore.data.fast_charger_type = data.fast_charger_type
  if (data.charge_speed !== undefined) vehicleStore.data.charge_speed = data.charge_speed
  if (data.battery_heater_on !== undefined) vehicleStore.data.battery_heater_on = data.battery_heater_on

  // Vehicle config
  if (data.car_type !== undefined) vehicleStore.data.car_type = data.car_type
  if (data.version !== undefined) vehicleStore.data.version = data.version
  if (data.exterior_color !== undefined) vehicleStore.data.exterior_color = data.exterior_color

  // Lights
  if (data.lights_high_beams !== undefined) vehicleStore.data.lights_high_beams = data.lights_high_beams
  if (data.lights_hazards_active !== undefined) vehicleStore.data.lights_hazards_active = data.lights_hazards_active
  if (data.lights_turn_signal !== undefined) vehicleStore.data.lights_turn_signal = data.lights_turn_signal

  // Driving detail
  if (data.brake_pedal !== undefined) vehicleStore.data.brake_pedal = data.brake_pedal
  if (data.drive_rail !== undefined) vehicleStore.data.drive_rail = data.drive_rail
  if (data.pedal_position !== undefined) vehicleStore.data.pedal_position = data.pedal_position

  // Odometer
  if (data.odometer_km !== undefined) vehicleStore.data.odometer_km = data.odometer_km

  // Center display
  if (data.center_display_state !== undefined) vehicleStore.data.center_display_state = data.center_display_state

  // Guest/Service mode
  if (data.guest_mode_enabled !== undefined) vehicleStore.data.guest_mode_enabled = data.guest_mode_enabled
  if (data.service_mode !== undefined) vehicleStore.data.service_mode = data.service_mode

  // Destination
  if (data.destination_latitude !== undefined) vehicleStore.data.destination_latitude = data.destination_latitude
  if (data.destination_longitude !== undefined) vehicleStore.data.destination_longitude = data.destination_longitude
  if (data.destination_name !== undefined) vehicleStore.data.destination_name = data.destination_name
  if (data.miles_to_arrival !== undefined) vehicleStore.data.miles_to_arrival = data.miles_to_arrival
  if (data.minutes_to_arrival !== undefined) vehicleStore.data.minutes_to_arrival = data.minutes_to_arrival

  // Supercharging
  if (data.supercharging !== undefined) vehicleStore.data.supercharging = data.supercharging

  if (data.media_playback_status !== undefined) vehicleStore.data.media_playback_status = data.media_playback_status
  if (data.media_audio_source !== undefined) vehicleStore.data.media_audio_source = data.media_audio_source
  if (data.media_volume !== undefined) vehicleStore.data.media_volume = data.media_volume
  if (data.media_audio_volume_increment !== undefined) vehicleStore.data.media_audio_volume_increment = data.media_audio_volume_increment
  if (data.media_audio_volume_max !== undefined) vehicleStore.data.media_audio_volume_max = data.media_audio_volume_max
  if (data.now_playing_title !== undefined) vehicleStore.data.now_playing_title = data.now_playing_title
  if (data.now_playing_artist !== undefined) vehicleStore.data.now_playing_artist = data.now_playing_artist
  if (data.now_playing_album !== undefined) vehicleStore.data.now_playing_album = data.now_playing_album
  if (data.now_playing_duration !== undefined) vehicleStore.data.now_playing_duration = data.now_playing_duration
  if (data.now_playing_elapsed !== undefined) vehicleStore.data.now_playing_elapsed = data.now_playing_elapsed
  if (data.now_playing_station !== undefined) vehicleStore.data.now_playing_station = data.now_playing_station
  if (data.state_output) {
    vehicleStore.stateOutput = data.state_output
  }
  vehicleStore.source = 'ws'
  stopFallbackPolling()
}

function onWSOnlineState(data) {
  if (data.state_output) {
    vehicleStore.stateOutput = data.state_output
  } else if (data.state) {
    if (vehicleStore.stateOutput) {
      vehicleStore.stateOutput = {
        ...vehicleStore.stateOutput,
        state: {
          ...vehicleStore.stateOutput.state,
          online_state: data.state,
          online: data.online,
        },
      }
    } else {
      vehicleStore.stateOutput = {
        vin: vehicleStore.vin,
        state: {
          online_state: data.state,
          online: data.online,
          confidence: 0.5,
          changed_at: Math.floor(Date.now() / 1000),
        },
        drive: { drive_state: 'parked', speed: 0, gear: 'P' },
        charge: { charge_state: 'disconnected', battery_level: 0 },
        lock: { lock_state: 'locked', doors_open: false },
        command: { command_state: 'idle', last_command: '', latency_ms: 0 },
        meta: { last_success_at: 0, last_fail_at: 0, state_lock_until: 0, state_transition_count: 0, last_state_source: 'ws' },
      }
    }
  }
  mergeData(data)
  if (vehicleStore.source !== 'ble') {
    vehicleStore.source = 'ws'
  }
}

function onWSPollState(data) {
  if (data.poll_state) {
    vehicleStore.pollState = data.poll_state
  }
}

function onWSCommandState(data) {
  if (data.command_state) {
    vehicleStore.commandState = data.command_state
  }
  if (data.last_command) {
    vehicleStore.lastCommand = data.last_command
  }
  if (data.latency_ms !== undefined) {
    vehicleStore.commandLatencyMs = data.latency_ms
  }
  if (data.command_state === 'success' || data.command_state === 'failed') {
    setTimeout(() => {
      if (vehicleStore.commandState === data.command_state) {
        vehicleStore.commandState = 'idle'
      }
    }, 3000)
  }
}

function onWSMediaState(data) {
  if (data.media_playback_status !== undefined) vehicleStore.data.media_playback_status = data.media_playback_status
  if (data.media_audio_source !== undefined) vehicleStore.data.media_audio_source = data.media_audio_source
  if (data.media_volume !== undefined) vehicleStore.data.media_volume = data.media_volume
  if (data.media_audio_volume_increment !== undefined) vehicleStore.data.media_audio_volume_increment = data.media_audio_volume_increment
  if (data.media_audio_volume_max !== undefined) vehicleStore.data.media_audio_volume_max = data.media_audio_volume_max
  if (data.now_playing_title !== undefined) vehicleStore.data.now_playing_title = data.now_playing_title
  if (data.now_playing_artist !== undefined) vehicleStore.data.now_playing_artist = data.now_playing_artist
  if (data.now_playing_album !== undefined) vehicleStore.data.now_playing_album = data.now_playing_album
  if (data.now_playing_duration !== undefined) vehicleStore.data.now_playing_duration = data.now_playing_duration
  if (data.now_playing_elapsed !== undefined) vehicleStore.data.now_playing_elapsed = data.now_playing_elapsed
  if (data.now_playing_station !== undefined) vehicleStore.data.now_playing_station = data.now_playing_station
}

function onWSTripEnded(data) {
  console.log('[VehicleData] Trip ended, AI analysis started:', data)
  vehicleStore.analysisNotification = {
    type: 'trip_ended',
    refId: data.ref_id,
    tripId: data.trip_id,
    status: data.status,
    timestamp: Date.now()
  }
}

function onWSChargingEnded(data) {
  console.log('[VehicleData] Charging ended, AI analysis started:', data)
  vehicleStore.analysisNotification = {
    type: 'charging_ended',
    refId: data.ref_id,
    chargeId: data.charge_id,
    status: data.status,
    timestamp: Date.now()
  }
}

function onWSAnalysisComplete(data) {
  console.log('[VehicleData] Analysis complete:', data)
  vehicleStore.analysisNotification = {
    type: 'analysis_complete',
    analysisType: data.type,
    refId: data.ref_id,
    status: data.status,
    timestamp: Date.now()
  }
}

function onWSOpen() {
  console.log('[VehicleData] WebSocket connected')
  vehicleStore.source = 'ws'
  stopFallbackPolling()
}

function onWSClose() {
  console.log('[VehicleData] WebSocket disconnected, starting fallback polling')
  if (vehicleStore.source === 'ws') {
    vehicleStore.source = 'cloud'
  }
  vehicleStore.realtimeSource = null
  startFallbackPolling()
}

function startFallbackPolling() {
  stopFallbackPolling()
  const poll = async () => {
    if (!vehicleStore.vin) return
    if (wsIsConnected()) {
      stopFallbackPolling()
      return
    }
    vehicleStore.loading = true
    try {
      const res = await getVehicleState(vehicleStore.vin)
      const data = res.data || {}
      const smoothed = applyEMA(data)
      mergeData(smoothed)
      vehicleStore.source = 'cloud'
      vehicleStore.error = null
    } catch (err) {
      vehicleStore.error = err.message
    } finally {
      vehicleStore.loading = false
    }
    stopFallbackPolling()
    fallbackTimer = setInterval(poll, getFallbackInterval())
  }
  poll()
}

function stopFallbackPolling() {
  if (fallbackTimer) {
    clearInterval(fallbackTimer)
    fallbackTimer = null
  }
}

function getFallbackInterval() {
  if (vehicleStore.stateOutput) {
    return getRefreshInterval(vehicleStore.stateOutput)
  }
  const d = vehicleStore.data
  if (d.gear === 'D' || d.gear === 'R') return 2000
  if (d.charging === true) return 3000
  return 5000
}

export function startSimulatorMode() {
  if (bleUnsubscribe) {
    bleUnsubscribe()
    bleUnsubscribe = null
  }

  bleUnsubscribe = onBLEData((event) => {
    if (event.type === 'state_change') {
      vehicleStore.bleState = event.to
      vehicleStore.bleScanning = event.to === BLEState.SCANNING
    } else if (event.type === 'data') {
      mergeData(event.data)
      vehicleStore.source = 'ble'
      vehicleStore.bleConnected = true
    } else if (event.type === 'connect') {
      vehicleStore.bleConnected = true
      vehicleStore.source = 'ble'
    } else if (event.type === 'disconnect') {
      vehicleStore.bleConnected = false
      vehicleStore.source = 'ws'
    }
  })

  startSimulator()
  vehicleStore.source = 'ble'
  vehicleStore.bleConnected = true
}

export function stopSimulatorMode() {
  stopSimulator()
  vehicleStore.bleConnected = false
  vehicleStore.source = 'ws'
  if (bleUnsubscribe) {
    bleUnsubscribe()
    bleUnsubscribe = null
  }
}

export function getSource() { return vehicleStore.source }
export function isBLEMode() { return vehicleStore.source === 'ble' }
export function isCloudMode() { return vehicleStore.source === 'cloud' }
export function isWSMode() { return vehicleStore.source === 'ws' }
export function isTelemetryMode() { return vehicleStore.realtimeSource === 'telemetry' }
