import { reactive } from 'vue'
import { getVehicleState } from '@/api/vehicle.js'
import { startSimulator, stopSimulator, isSimulatorMode, onBLEData, BLEState } from './ble.js'
import { wsConnect, wsDisconnect, wsOn, wsOff, wsSwitchVIN, wsIsConnected } from './websocket.js'
import { getRefreshInterval } from './vehicle-state.js'

const EMA_ALPHA = 0.3
const EMA_FIELDS = ['speed', 'inside_temp', 'outside_temp', 'charge_power', 'range_km', 'charge_amps', 'soc']

const vehicleStore = reactive({
  data: {},
  stateOutput: null,
  source: 'ws',
  bleConnected: false,
  bleScanning: false,
  bleState: 'idle',
  loading: false,
  error: null,
  vin: '',
  pollState: 'sleeping'
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
      if (emaState[field] === undefined) {
        emaState[field] = current
      } else {
        emaState[field] = emaState[field] * (1 - EMA_ALPHA) + current * EMA_ALPHA
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

function mergeData(partial) {
  if (!partial || typeof partial !== 'object') return
  if (partial.state_output) {
    vehicleStore.stateOutput = partial.state_output
  }
  const { state_output, ...rest } = partial
  vehicleStore.data = { ...vehicleStore.data, ...rest }
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
  wsOn('online_state', onWSOnlineState)
  wsOn('poll_state', onWSPollState)
  wsOn('open', onWSOpen)
  wsOn('close', onWSClose)

  wsConnect(vin)
}

function stopWSStream() {
  wsOff('vehicle_state', onWSVehicleState)
  wsOff('online_state', onWSOnlineState)
  wsOff('poll_state', onWSPollState)
  wsOff('open', onWSOpen)
  wsOff('close', onWSClose)
  wsDisconnect()
}

function onWSVehicleState(data) {
  const smoothed = applyEMA(data)
  mergeData(smoothed)
  vehicleStore.source = 'ws'
  stopFallbackPolling()
}

function onWSOnlineState(data) {
  if (data.state_output) {
    vehicleStore.stateOutput = data.state_output
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
  if (d.gear === 'D' || d.gear === 'R') return 3000
  if (d.charging === true) return 5000
  return 15000
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
