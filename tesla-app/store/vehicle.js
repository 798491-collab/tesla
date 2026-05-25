import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getUserVehicles, getVehicleState } from '@/api/vehicle.js'

export const useVehicleStore = defineStore('vehicle', () => {
  const vehicles = ref([])
  const currentVehicle = ref(null)
  const vehicleState = ref(null)
  const loading = ref(false)

  const vehicleList = computed(() => vehicles.value)
  const hasVehicles = computed(() => vehicles.value.length > 0)

  const fetchVehicles = async () => {
    loading.value = true
    try {
      const res = await getUserVehicles()
      vehicles.value = res.data || []
      if (vehicles.value.length > 0) {
        const savedVIN = uni.getStorageSync('currentVehicleVIN')
        const saved = savedVIN ? vehicles.value.find(v => v.vin === savedVIN) : null
        if (saved) {
          currentVehicle.value = saved
        } else if (!currentVehicle.value) {
          currentVehicle.value = vehicles.value[0]
        }
      }
      return res.data
    } catch (err) {
      console.error('获取车辆列表失败:', err)
      // 不抛出错误，避免页面崩溃
      return []
    } finally {
      loading.value = false
    }
  }

  const selectVehicle = (vehicle) => {
    currentVehicle.value = vehicle
    if (vehicle?.vin) {
      uni.setStorageSync('currentVehicleVIN', vehicle.vin)
    }
  }

  const fetchVehicleState = async (vin) => {
    if (!vin) return
    try {
      const res = await getVehicleState(vin)
      vehicleState.value = res.data
      return res.data
    } catch (err) {
      console.error('获取车辆状态失败:', err)
    }
  }

  const clearVehicles = () => {
    vehicles.value = []
    currentVehicle.value = null
    vehicleState.value = null
    uni.removeStorageSync('currentVehicleVIN')
  }

  return {
    vehicles,
    currentVehicle,
    vehicleState,
    loading,
    vehicleList,
    hasVehicles,
    fetchVehicles,
    selectVehicle,
    setCurrentVehicle: selectVehicle,
    fetchVehicleState,
    clearVehicles
  }
})
