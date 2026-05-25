import { get, post } from '@/utils/request.js'

export const getTripAnalysis = (vin, refId) => {
  return get(`/api/ai/trip/${vin}/${encodeURIComponent(refId)}`)
}

export const getChargingAnalysis = (vin, refId) => {
  return get(`/api/ai/charging/${vin}/${encodeURIComponent(refId)}`)
}

export const getVehicleAnalysis = (vin, date) => {
  const params = date ? { date } : {}
  return get(`/api/ai/vehicle/${vin}`, params)
}

export const triggerTripAnalysis = (vin, refId) => {
  return post(`/api/ai/trip/${vin}/${encodeURIComponent(refId)}`, {})
}

export const triggerChargingAnalysis = (vin, refId) => {
  return post(`/api/ai/charging/${vin}/${encodeURIComponent(refId)}`, {})
}

export const triggerVehicleAnalysis = (vin, date) => {
  const params = date ? `?date=${date}` : ''
  return post(`/api/ai/vehicle/${vin}${params}`, {})
}

export const getAnalysisHistory = (vin, type, limit) => {
  const params = {}
  if (type) params.type = type
  if (limit) params.limit = limit
  return get(`/api/ai/history/${vin}`, params)
}

export const getAnalysisList = (vin, type, page, pageSize) => {
  const params = {}
  if (type) params.type = type
  if (page) params.page = page
  if (pageSize) params.page_size = pageSize
  return get(`/api/ai/list/${vin}`, params)
}

export const getLatestAnalysis = (vin, type) => {
  return get(`/api/ai/latest/${vin}/${type}`)
}
