import { get } from '@/utils/request.js'

export const getTripLogs = (vin, start, end) => {
  const params = {}
  if (start) params.start = start
  if (end) params.end = end
  return get(`/api/trip/${vin}/logs`, params)
}

export const getTripStats = (vin, start, end) => {
  return get(`/api/trip/${vin}/stats`, { start, end })
}

export const getMonthlyStats = (vin) => {
  return get(`/api/trip/${vin}/monthly-stats`)
}

export const getMonthlyTripList = (vin) => {
  return get(`/api/trip/${vin}/monthly-list`)
}

export const getTripPoints = (vin, tripId) => {
  return get(`/api/trip/${vin}/points/${tripId}`)
}
