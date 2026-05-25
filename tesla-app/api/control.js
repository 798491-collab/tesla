import { get, post } from '@/utils/request.js'

export const getCommands = () => {
  return get('/api/vcp/commands')
}

export const doorLock = (vin) => {
  return post('/api/vcp/door_lock', { vin })
}

export const doorUnlock = (vin) => {
  return post('/api/vcp/door_unlock', { vin })
}

export const autoConditioningStart = (vin) => {
  return post('/api/vcp/auto_conditioning_start', { vin })
}

export const autoConditioningStop = (vin) => {
  return post('/api/vcp/auto_conditioning_stop', { vin })
}

export const honkHorn = (vin) => {
  return post('/api/vcp/honk_horn', { vin })
}

export const flashLights = (vin) => {
  return post('/api/vcp/flash_lights', { vin })
}

export const actuateTrunk = (vin) => {
  return post('/api/vcp/actuate_trunk', { vin })
}

export const actuateFrunk = (vin) => {
  return post('/api/vcp/actuate_frunk', { vin })
}

export const setSentryMode = (vin, on) => {
  return post('/api/vcp/set_sentry_mode', { vin, on })
}

export const chargeStart = (vin) => {
  return post('/api/vcp/charge_start', { vin })
}

export const chargeStop = (vin) => {
  return post('/api/vcp/charge_stop', { vin })
}

export const setChargeLimit = (vin, percent) => {
  return post('/api/vcp/set_charge_limit', { vin, percent })
}

export const chargePortDoorOpen = (vin) => {
  return post('/api/vcp/charge_port_door_open', { vin })
}

export const chargePortDoorClose = (vin) => {
  return post('/api/vcp/charge_port_door_close', { vin })
}

export const setTemps = (vin, driver_temp, passenger_temp) => {
  return post('/api/vcp/set_temps', { vin, driver_temp, passenger_temp })
}

export const remoteSeatHeater = (vin, heater, level) => {
  return post('/api/vcp/remote_seat_heater', { vin, heater, level })
}

export const remoteSteeringWheelHeater = (vin) => {
  return post('/api/vcp/remote_steering_wheel_heater', { vin })
}

export const windowControl = (vin, command) => {
  return post('/api/vcp/window_control', { vin, command })
}

export const mirrorFold = (vin) => {
  return post('/api/vcp/mirror_fold', { vin })
}

export const mirrorUnfold = (vin) => {
  return post('/api/vcp/mirror_unfold', { vin })
}

export const lightControl = (vin, data) => {
  return post('/api/vcp/light_control', { vin, ...data })
}
