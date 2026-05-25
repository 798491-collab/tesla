const BLEState = {
  IDLE: 'idle',
  INITIALIZING: 'initializing',
  SCANNING: 'scanning',
  CONNECTING: 'connecting',
  CONNECTED: 'connected',
  RECONNECTING: 'reconnecting',
  FAILED: 'failed'
}

const VALID_TRANSITIONS = {
  [BLEState.IDLE]: [BLEState.INITIALIZING, BLEState.SCANNING],
  [BLEState.INITIALIZING]: [BLEState.SCANNING, BLEState.IDLE, BLEState.FAILED],
  [BLEState.SCANNING]: [BLEState.CONNECTING, BLEState.IDLE, BLEState.FAILED],
  [BLEState.CONNECTING]: [BLEState.CONNECTED, BLEState.RECONNECTING, BLEState.IDLE, BLEState.FAILED],
  [BLEState.CONNECTED]: [BLEState.RECONNECTING, BLEState.IDLE, BLEState.FAILED],
  [BLEState.RECONNECTING]: [BLEState.CONNECTED, BLEState.FAILED, BLEState.IDLE],
  [BLEState.FAILED]: [BLEState.IDLE, BLEState.INITIALIZING]
}

const TESLA_SERVICE_UUID = '00000211-b2d1-43f0-9b88-960cebf8b91e'
const CHAR_WRITE = '00000212-b2d1-43f0-9b88-960cebf8b91e'
const CHAR_INDICATE = '00000213-b2d1-43f0-9b88-960cebf8b91e'
const CHAR_VERSION = '00000214-b2d1-43f0-9b88-960cebf8b91e'

let bleState = {
  connectionState: BLEState.IDLE,
  deviceId: null,
  deviceName: null,
  listeners: new Set(),
  lastData: null,
  packetCount: 0,
  reconnectAttempts: 0,
  lastError: null,
  simulatorActive: false,
  serviceUUID: null,
  writeChar: null,
  indicateChar: null,
  versionChar: null
}

function transitionTo(newState) {
  const valid = VALID_TRANSITIONS[bleState.connectionState] || []
  if (!valid.includes(newState)) {
    console.warn(`[BLE] Invalid transition: ${bleState.connectionState} → ${newState}`)
    return false
  }
  const oldState = bleState.connectionState
  bleState.connectionState = newState
  console.log(`[BLE] State: ${oldState} → ${newState}`)
  notifyListeners({ type: 'state_change', from: oldState, to: newState })
  return true
}

export function getBLEState() {
  return {
    connectionState: bleState.connectionState,
    deviceId: bleState.deviceId,
    deviceName: bleState.deviceName,
    packetCount: bleState.packetCount,
    reconnectAttempts: bleState.reconnectAttempts,
    lastError: bleState.lastError
  }
}

export function isConnected() {
  return bleState.connectionState === BLEState.CONNECTED
}

export function isReconnecting() {
  return bleState.connectionState === BLEState.RECONNECTING
}

export function getConnectionState() {
  return bleState.connectionState
}

export function computeBLEName(vin) {
  if (!vin) return null
  const last6 = vin.slice(-6)
  return `Tesla ${last6}`
}

export async function initBLE() {
  if (bleState.connectionState !== BLEState.IDLE) {
    console.warn('[BLE] Already initialized or in progress')
    return
  }

  transitionTo(BLEState.INITIALIZING)

  try {
    await requestBLEPermissions()

    await new Promise((resolve, reject) => {
      uni.openBluetoothAdapter({
        success: resolve,
        fail: reject
      })
    })
    console.log('[BLE] Bluetooth adapter opened')
    transitionTo(BLEState.IDLE)
  } catch (e) {
    transitionTo(BLEState.FAILED)
    bleState.lastError = e.errMsg || e.message || 'Bluetooth init failed'
    throw e
  }
}

function requestBLEPermissions() {
  return new Promise((resolve) => {
    const system = uni.getSystemInfoSync()
    const platform = system.platform || ''

    console.log('[BLE] Platform:', platform, 'System:', system.system)

    if (platform !== 'android') {
      resolve()
      return
    }

    const androidVersion = parseInt((system.system || '').replace(/Android\s*/i, '')) || 0

    if (androidVersion < 12) {
      resolve()
      return
    }

    if (typeof plus !== 'undefined' && plus.android && plus.android.requestPermissions) {
      console.log('[BLE] Requesting Android 12+ BLE permissions...')
      plus.android.requestPermissions(
        [
          'android.permission.BLUETOOTH_SCAN',
          'android.permission.BLUETOOTH_CONNECT',
          'android.permission.ACCESS_FINE_LOCATION'
        ],
        (result) => {
          console.log('[BLE] Permission result - granted:', result.granted, 'denied:', result.denied)
          resolve()
        },
        (error) => {
          console.warn('[BLE] Permission request error:', error)
          resolve()
        }
      )
    } else {
      resolve()
    }
  })
}

export function closeBLE() {
  cancelReconnectJS()

  return new Promise((resolve) => {
    if (bleState.deviceId) {
      uni.closeBLEConnection({
        deviceId: bleState.deviceId,
        complete: () => {
          resetState()
          uni.closeBluetoothAdapter({ complete: () => resolve() })
        }
      })
    } else {
      resetState()
      uni.closeBluetoothAdapter({ complete: () => resolve() })
    }
  })
}

function resetState() {
  bleState.connectionState = BLEState.IDLE
  bleState.deviceId = null
  bleState.deviceName = null
  bleState.reconnectAttempts = 0
  bleState.lastError = null
  bleState.serviceUUID = null
  bleState.writeChar = null
  bleState.indicateChar = null
  bleState.versionChar = null
}

export function scanVehicle(vin, rssiThreshold = -70, timeout = 10000) {
  if (bleState.connectionState !== BLEState.IDLE) {
    console.warn('[BLE] Cannot scan in state:', bleState.connectionState)
    return Promise.resolve([])
  }

  transitionTo(BLEState.SCANNING)
  const found = []

  const targetName = vin ? computeBLEName(vin) : null
  console.log('[BLE] Scanning for Tesla BLE, target name:', targetName || 'any')

  return new Promise((resolve) => {
    uni.startBluetoothDevicesDiscovery({
      services: [TESLA_SERVICE_UUID],
      allowDuplicatesKey: false,
      success: () => {
        setTimeout(() => {
          uni.stopBluetoothDevicesDiscovery()
          if (bleState.connectionState === BLEState.SCANNING) {
            transitionTo(BLEState.IDLE)
          }
          resolve(found)
        }, timeout)

        uni.onBluetoothDeviceFound((res) => {
          for (const device of (res.devices || [])) {
            const name = device.name || device.localName || ''
            const isTesla = name.startsWith('Tesla') || name.startsWith('S')
            if (isTesla && device.RSSI && device.RSSI > rssiThreshold) {
              if (!found.find(d => d.deviceId === device.deviceId)) {
                found.push({
                  deviceId: device.deviceId,
                  name: name,
                  RSSI: device.RSSI
                })
                console.log('[BLE] Found Tesla:', name, 'RSSI:', device.RSSI)
              }
            }
          }
        })
      },
      fail: () => {
        transitionTo(BLEState.FAILED)
        resolve(found)
      }
    })
  })
}

export async function connectVehicle(deviceId) {
  if (bleState.connectionState !== BLEState.IDLE) {
    throw new Error(`Cannot connect in state: ${bleState.connectionState}`)
  }

  bleState.deviceId = deviceId
  transitionTo(BLEState.CONNECTING)

  try {
    await doConnect(deviceId)
  } catch (e) {
    bleState.lastError = e.message || 'Connect failed'
    if (bleState.connectionState === BLEState.CONNECTING) {
      transitionTo(BLEState.FAILED)
    }
    throw e
  }
}

function doConnect(deviceId) {
  return new Promise((resolve, reject) => {
    uni.stopBluetoothDevicesDiscovery()

    uni.createBLEConnection({
      deviceId,
      timeout: 15000,
      success: () => {
        transitionTo(BLEState.CONNECTED)
        notifyListeners({ type: 'connect' })

        uni.onBLEConnectionStateChange((res) => {
          if (!res.connected && res.deviceId === bleState.deviceId) {
            handleDisconnect()
          }
        })

        uni.getBLEDeviceServices({
          deviceId,
          success: (res) => {
            const service = (res.services || []).find(s =>
              s.uuid.toLowerCase() === TESLA_SERVICE_UUID.toLowerCase()
            )
            if (!service) {
              console.error('[BLE] Tesla service not found. Available services:', res.services?.map(s => s.uuid))
              reject(new Error('Tesla BLE service not found'))
              return
            }

            bleState.serviceUUID = service.uuid
            console.log('[BLE] Found Tesla service:', service.uuid)

            uni.getBLEDeviceCharacteristics({
              deviceId,
              serviceId: service.uuid,
              success: (res) => {
                console.log('[BLE] Characteristics:', res.characteristics?.map(c => `${c.uuid} [${c.properties?.write ? 'W' : ''}${c.properties?.indicate ? 'I' : ''}${c.properties?.read ? 'R' : ''}${c.properties?.notify ? 'N' : ''}]`))

                for (const char of (res.characteristics || [])) {
                  const uuid = char.uuid.toLowerCase()
                  if (uuid === CHAR_WRITE.toLowerCase()) bleState.writeChar = char
                  else if (uuid === CHAR_INDICATE.toLowerCase()) bleState.indicateChar = char
                  else if (uuid === CHAR_VERSION.toLowerCase()) bleState.versionChar = char
                }

                const subscribeIndicate = () => {
                  if (!bleState.indicateChar) {
                    console.warn('[BLE] Indicate characteristic not found')
                    return Promise.resolve()
                  }
                  return new Promise((r) => {
                    uni.notifyBLECharacteristicValueChange({
                      deviceId,
                      serviceId: service.uuid,
                      characteristicId: bleState.indicateChar.uuid,
                      state: true,
                      success: () => {
                        console.log('[BLE] Subscribed to Indicate characteristic')
                        r()
                      },
                      fail: (err) => {
                        console.warn('[BLE] Subscribe Indicate failed:', err)
                        r()
                      }
                    })
                  })
                }

                const readVersion = () => {
                  if (!bleState.versionChar) return Promise.resolve()
                  return new Promise((r) => {
                    uni.readBLECharacteristicValue({
                      deviceId,
                      serviceId: service.uuid,
                      characteristicId: bleState.versionChar.uuid,
                      success: () => console.log('[BLE] Read version requested'),
                      fail: () => {},
                      complete: () => r()
                    })
                  })
                }

                uni.onBLECharacteristicValueChange((res) => {
                  const charId = res.characteristicId.toLowerCase()
                  const value = res.value

                  if (charId === CHAR_INDICATE.toLowerCase()) {
                    handleVehicleResponse(value)
                  } else if (charId === CHAR_VERSION.toLowerCase()) {
                    console.log('[BLE] Version response, length:', value.byteLength)
                  }
                })

                Promise.all([subscribeIndicate(), readVersion()])
                  .then(() => resolve(true))
                  .catch(() => resolve(true))
              },
              fail: reject
            })
          },
          fail: reject
        })
      },
      fail: reject
    })
  })
}

function handleVehicleResponse(value) {
  if (!value || value.byteLength === 0) return

  bleState.packetCount++
  const bytes = new Uint8Array(value)
  console.log('[BLE] Vehicle response, length:', bytes.length, 'hex:', Array.from(bytes.slice(0, 20)).map(b => b.toString(16).padStart(2, '0')).join(' '))

  notifyListeners({
    type: 'data',
    raw: value,
    bytes: bytes,
    length: bytes.length
  })
}

export function sendCommand(data) {
  if (!bleState.writeChar || !bleState.serviceUUID || !bleState.deviceId) {
    console.warn('[BLE] Cannot send command: not connected or missing characteristics')
    return Promise.reject(new Error('Not connected'))
  }

  return new Promise((resolve, reject) => {
    uni.writeBLECharacteristicValue({
      deviceId: bleState.deviceId,
      serviceId: bleState.serviceUUID,
      characteristicId: bleState.writeChar.uuid,
      value: data,
      success: resolve,
      fail: reject
    })
  })
}

function handleDisconnect() {
  const wasConnected = bleState.connectionState === BLEState.CONNECTED
  const wasReconnecting = bleState.connectionState === BLEState.RECONNECTING

  notifyListeners({ type: 'disconnect' })

  if (wasConnected && bleState.deviceId) {
    startReconnectJS()
  } else if (wasReconnecting) {
    scheduleReconnectJS()
  } else {
    transitionTo(BLEState.IDLE)
  }
}

let jsReconnectTimer = null
let jsReconnectAttempts = 0
const JS_MAX_RECONNECT = 5
const JS_RECONNECT_BASE_DELAY = 2000
const JS_LONG_CYCLE_DELAY = 60000

function startReconnectJS() {
  jsReconnectAttempts = 0
  transitionTo(BLEState.RECONNECTING)
  scheduleReconnectJS()
}

function scheduleReconnectJS() {
  cancelReconnectJS()

  if (jsReconnectAttempts >= JS_MAX_RECONNECT) {
    console.warn('[BLE] Max reconnect attempts, entering long cycle')
    jsReconnectTimer = setTimeout(() => {
      jsReconnectAttempts = 0
      scheduleReconnectJS()
    }, JS_LONG_CYCLE_DELAY)
    return
  }

  const delay = Math.min(JS_RECONNECT_BASE_DELAY * Math.pow(2, jsReconnectAttempts), 30000)
  jsReconnectAttempts++
  bleState.reconnectAttempts = jsReconnectAttempts

  console.log(`[BLE] Reconnect attempt ${jsReconnectAttempts} in ${delay}ms`)

  jsReconnectTimer = setTimeout(async () => {
    if (bleState.connectionState !== BLEState.RECONNECTING) return
    try {
      await doConnect(bleState.deviceId)
      jsReconnectAttempts = 0
      bleState.reconnectAttempts = 0
      notifyListeners({ type: 'reconnect_success' })
    } catch (e) {
      console.warn('[BLE] Reconnect failed:', e.message)
      scheduleReconnectJS()
    }
  }, delay)
}

function cancelReconnectJS() {
  if (jsReconnectTimer) {
    clearTimeout(jsReconnectTimer)
    jsReconnectTimer = null
  }
}

export function onBLEData(callback) {
  bleState.listeners.add(callback)
  return () => bleState.listeners.delete(callback)
}

function notifyListeners(event) {
  for (const cb of bleState.listeners) {
    try { cb(event) } catch (e) {}
  }
}

export function getLastBLEData() { return bleState.lastData }

// ===== BLE Simulator =====

let simulatorTimer = null
let simulatorSpeed = 0
let simulatorTargetSpeed = 0
let simulatorBattery = 78
let simulatorInsideTemp = 22.5
let simulatorOutsideTemp = 18.0
let simulatorCharging = false
let simulatorShiftState = 'P'

export function isSimulatorMode() {
  return bleState.simulatorActive === true
}

export function startSimulator() {
  if (simulatorTimer) return

  bleState.simulatorActive = true
  bleState.connectionState = BLEState.CONNECTED
  bleState.deviceId = 'SIMULATOR'
  bleState.deviceName = 'Tesla Simulator'

  notifyListeners({ type: 'state_change', from: BLEState.IDLE, to: BLEState.CONNECTED })
  notifyListeners({ type: 'connect' })

  simulatorTimer = setInterval(() => {
    if (simulatorShiftState === 'D') {
      if (simulatorSpeed < simulatorTargetSpeed) {
        simulatorSpeed = Math.min(simulatorSpeed + 2 + Math.random() * 3, simulatorTargetSpeed)
      } else if (simulatorSpeed > simulatorTargetSpeed) {
        simulatorSpeed = Math.max(simulatorSpeed - 3 - Math.random() * 2, simulatorTargetSpeed)
      }
      simulatorSpeed += (Math.random() - 0.5) * 1.5
      simulatorSpeed = Math.max(0, simulatorSpeed)
    } else if (simulatorShiftState === 'R') {
      simulatorSpeed = Math.min(simulatorSpeed + 1, 10)
    } else {
      if (simulatorSpeed > 0.5) {
        simulatorSpeed = Math.max(0, simulatorSpeed - 2 - Math.random() * 2)
      } else {
        simulatorSpeed = 0
      }
    }

    if (simulatorCharging) {
      simulatorBattery = Math.min(100, simulatorBattery + 0.02)
    }

    const data = {
      speed: Math.round(simulatorSpeed * 10) / 10,
      shift_state: simulatorShiftState,
      battery_level: Math.round(simulatorBattery),
      charging_state: simulatorCharging ? 'Charging' : 'Idle',
      charger_power: simulatorCharging ? Math.round((30 + Math.random() * 70) * 10) / 10 : 0,
      battery_range_km: Math.round(simulatorBattery * 5.8 * 10) / 10,
      inside_temp: Math.round((simulatorInsideTemp + (Math.random() - 0.5) * 0.3) * 10) / 10,
      outside_temp: Math.round((simulatorOutsideTemp + (Math.random() - 0.5) * 0.5) * 10) / 10,
      is_ac_on: true,
      locked: true,
      door_fl: false,
      door_fr: false,
      door_rl: false,
      door_rr: false,
      trunk_open: false,
      frunk_open: false,
      sentry_mode: false,
      charge_port_open: simulatorCharging,
      fast_charger_present: simulatorCharging && simulatorBattery < 80,
      heading: Math.round(Math.random() * 360),
      seat_heater_left: 0,
      steering_wheel_heater: false
    }

    bleState.packetCount++
    bleState.lastData = data
    notifyListeners({ type: 'data', data })
  }, 200)
}

export function stopSimulator() {
  if (simulatorTimer) {
    clearInterval(simulatorTimer)
    simulatorTimer = null
  }
  bleState.simulatorActive = false
  bleState.connectionState = BLEState.IDLE
  bleState.deviceId = null
  bleState.deviceName = null
  notifyListeners({ type: 'disconnect' })
  notifyListeners({ type: 'state_change', from: BLEState.CONNECTED, to: BLEState.IDLE })
}

export function setSimulatorShiftState(state) {
  simulatorShiftState = state
  if (state === 'P' || state === 'N') {
    simulatorTargetSpeed = 0
  } else if (state === 'D') {
    simulatorTargetSpeed = 40 + Math.random() * 80
  } else if (state === 'R') {
    simulatorTargetSpeed = 5
  }
}

export function setSimulatorCharging(charging) {
  simulatorCharging = charging
  if (charging) {
    simulatorShiftState = 'P'
    simulatorSpeed = 0
  }
}

export function setSimulatorSpeed(target) {
  simulatorTargetSpeed = target
}

export { TESLA_SERVICE_UUID, CHAR_WRITE, CHAR_INDICATE, CHAR_VERSION, BLEState }
