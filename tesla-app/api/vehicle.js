import { get, post, del } from '@/utils/request.js'

export const getAuthURL = () => {
  return get('/api/tesla/auth')
}

export const bindVehicle = (data) => {
  return post('/api/tesla/bind', data)
}

export const getUserVehicles = () => {
  return get('/api/tesla/vehicles')
}

export const getVehicleDetail = (vin) => {
  return get(`/api/tesla/vehicle/${vin}/detail`)
}

export const unbindVehicle = (vin) => {
  return del(`/api/tesla/unbind/${vin}`)
}

export const getVehicleState = (vin) => {
  return get(`/api/vehicle/${vin}/state`)
}

export const refreshVehicleState = (vin) => {
  return post(`/api/vehicle/${vin}/refresh`)
}

export const wakeVehicle = (vin) => {
  return post(`/api/vehicle/${vin}/wake`)
}

export const getFleetStatus = (vin) => {
  return get(`/api/tesla/vehicle/${vin}/fleet-status`)
}

export const getPairingURL = (vin) => {
  return get(`/api/tesla/vehicle/${vin}/pairing-url`)
}

export const checkPublicKeyHosting = (domain) => {
  return get('/api/tesla/partner/check-hosting', { domain })
}

export const checkPartnerPublicKey = (domain) => {
  return get('/api/tesla/partner/check-public-key', { domain })
}

export const getVehicleData = (vin) => {
  return get(`/api/vehicle/${vin}/data`)
}

export const getVehicleConfig = (vin) => {
  return get(`/api/vehicle/${vin}/config`)
}

export const getVehicleHealth = (vin) => {
  return get(`/api/vehicle/${vin}/health`)
}
