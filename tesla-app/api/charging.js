import { get, post } from '@/utils/request.js'

export const getChargingLogs = (vin, start, end) => {
  const params = {}
  if (start) params.start = start
  if (end) params.end = end
  return get(`/api/charging/${vin}/logs`, params)
}

export const getChargingStats = (vin, start, end) => {
  return get(`/api/charging/${vin}/stats`, { start, end })
}

export const getMonthlyChargingStats = (vin) => {
  return get(`/api/charging/${vin}/monthly-stats`)
}

export const getMonthlyChargingList = (vin) => {
  return get(`/api/charging/${vin}/monthly-list`)
}

// 更新充电记录价格
export const updateChargingPrice = (logId, data) => {
  return post(`/api/charging/log/${logId}/price`, data)
}
