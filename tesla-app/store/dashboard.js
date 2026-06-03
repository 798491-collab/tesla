import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const DASHBOARD_TYPES = [
  { key: 'instrument', label: '极简仪表盘', desc: '横屏双环仪表', icon: 'Dashboard' },
  { key: 'cockpit', label: '驾驶舱仪表盘', desc: '实时数据 + 地图 + 控制', icon: 'Speedometer' },
]

const DASHBOARD_ROUTES = {
  instrument: '/pages/dashboard/instrument',
  cockpit: '/pages/dashboard/cockpit',
}

export const useDashboardStore = defineStore('dashboard', () => {
  const stored = uni.getStorageSync('dashboardType')
  const dashboardType = ref(DASHBOARD_ROUTES[stored] ? stored : 'instrument')

  const currentDashboard = computed(() => {
    return DASHBOARD_TYPES.find(d => d.key === dashboardType.value) || DASHBOARD_TYPES[0]
  })

  const currentRoute = computed(() => {
    return DASHBOARD_ROUTES[dashboardType.value] || DASHBOARD_ROUTES.instrument
  })

  const dashboardList = computed(() => DASHBOARD_TYPES)

  const setDashboardType = (type) => {
    const found = DASHBOARD_TYPES.find(d => d.key === type)
    if (found) {
      dashboardType.value = type
      uni.setStorageSync('dashboardType', type)
    }
  }

  const getDashboardRoute = (type) => {
    return DASHBOARD_ROUTES[type] || DASHBOARD_ROUTES.instrument
  }

  return {
    dashboardType,
    currentDashboard,
    currentRoute,
    dashboardList,
    setDashboardType,
    getDashboardRoute,
  }
})
