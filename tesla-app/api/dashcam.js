import { get } from '@/utils/request.js'

export const getVehicleTracks = (vin, startTime, endTime) => {
  return get(`/api/vehicle/${vin}/tracks`, { start: startTime, end: endTime })
}

export const getDashcamEvents = (vin, options = {}) => {
  return get(`/api/dashcam/${vin}/events`, { type: options.type, limit: options.limit, offset: options.offset })
}
