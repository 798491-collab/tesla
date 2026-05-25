const API_BASE = import.meta.env.VITE_API_BASE_URL || 'https://your-domain.com'
const WS_BASE = API_BASE.replace(/^https?/, 'wss') + '/api/ws'

let socketTask = null
let reconnectTimer = null
let heartbeatTimer = null
let reconnectAttempts = 0
const MAX_RECONNECT_ATTEMPTS = 10
const RECONNECT_INTERVALS = [1000, 2000, 3000, 5000, 8000, 10000, 15000, 20000, 30000, 60000]
let listeners = {}
let currentVIN = ''
let isConnected = false
let isManualClose = false

function getToken() {
  return uni.getStorageSync('token') || ''
}

function getReconnectInterval() {
  const idx = Math.min(reconnectAttempts, RECONNECT_INTERVALS.length - 1)
  return RECONNECT_INTERVALS[idx]
}

export function wsConnect(vin) {
  if (!vin) return
  currentVIN = vin
  isManualClose = false

  if (socketTask) {
    wsDisconnect()
  }

  const token = getToken()
  if (!token) {
    console.warn('[WS] No token, skip connect')
    return
  }

  const url = `${WS_BASE}/vin/${vin}?token=${token}`
  console.log('[WS] Connecting to:', url.replace(token, '***'))

  socketTask = uni.connectSocket({
    url: url,
    complete: () => {}
  })

  socketTask.onOpen(() => {
    console.log('[WS] Connected for VIN:', currentVIN)
    isConnected = true
    reconnectAttempts = 0
    startHeartbeat()
    emit('open', { vin: currentVIN })
  })

  socketTask.onMessage((res) => {
    try {
      const msg = JSON.parse(res.data)
      if (msg.type === 'pong') {
        return
      }
      if (msg.vin && msg.vin !== currentVIN) {
        return
      }
      emit('message', msg)
      if (msg.type === 'vehicle_state') {
        emit('vehicle_state', msg.data)
      } else if (msg.type === 'online_state') {
        emit('online_state', msg.data)
      } else if (msg.type === 'poll_state') {
        emit('poll_state', msg.data)
      }
    } catch (e) {
      console.warn('[WS] Parse message error:', e)
    }
  })

  socketTask.onError((err) => {
    console.error('[WS] Error:', err)
    isConnected = false
    emit('error', err)
  })

  socketTask.onClose((res) => {
    console.log('[WS] Closed, code:', res.code, 'reason:', res.reason)
    isConnected = false
    socketTask = null
    stopHeartbeat()
    emit('close', { code: res.code, reason: res.reason })

    if (!isManualClose && reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
      scheduleReconnect()
    }
  })
}

export function wsDisconnect() {
  isManualClose = true
  stopHeartbeat()
  clearReconnect()

  if (socketTask) {
    socketTask.close({
      code: 1000,
      reason: 'manual close'
    })
    socketTask = null
  }
  isConnected = false
}

export function wsIsConnected() {
  return isConnected
}

export function wsSwitchVIN(vin) {
  if (vin === currentVIN && isConnected) return
  wsDisconnect()
  setTimeout(() => {
    wsConnect(vin)
  }, 300)
}

function scheduleReconnect() {
  if (reconnectTimer) return
  reconnectAttempts++
  const interval = getReconnectInterval()
  console.log(`[WS] Reconnect in ${interval}ms (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`)

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    wsConnect(currentVIN)
  }, interval)
}

function clearReconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  reconnectAttempts = 0
}

function startHeartbeat() {
  stopHeartbeat()
  heartbeatTimer = setInterval(() => {
    if (socketTask && isConnected) {
      socketTask.send({
        data: JSON.stringify({ type: 'ping' }),
        fail: () => {}
      })
    }
  }, 30000)
}

function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
}

export function wsOn(event, callback) {
  if (!listeners[event]) {
    listeners[event] = []
  }
  listeners[event].push(callback)
}

export function wsOff(event, callback) {
  if (!listeners[event]) return
  if (callback) {
    listeners[event] = listeners[event].filter(cb => cb !== callback)
  } else {
    delete listeners[event]
  }
}

function emit(event, data) {
  if (!listeners[event]) return
  for (const cb of listeners[event]) {
    try {
      cb(data)
    } catch (e) {
      console.error(`[WS] Listener error for ${event}:`, e)
    }
  }
}
