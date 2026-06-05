<template>
	<view class="cyber-dashboard">
		<view class="status-bar">
			<view class="status-left">
				<text class="time">{{ currentTime }}</text>
				<view class="signal-bars">
					<view class="bar" v-for="i in 4" :key="i" :class="{ active: i <= signalLevel }"></view>
				</view>
				<text class="tag-5g">5G</text>
			</view>
			<view class="status-center">
				<text class="ready-tag" :class="{ active: isReady }">READY</text>
				<view class="gears">
					<text v-for="g in ['P','R','N','D']" :key="g" :class="{ active: currentGear === g }">{{ g }}</text>
				</view>
			</view>
			<view class="status-right">
				<text class="ap-status" v-if="autopilotEnabled">AP</text>
				<text class="temp">{{ outsideTempDisplay }}°C</text>
			</view>
		</view>

		<view class="main-container">

			<!-- 左侧：地图（独占左侧，跨两行） -->
			<view class="card-box map-panel" :class="{ 'map-dark-filter': useMapFilter }">
				<map v-if="hasLocation" id="mapBg" class="map-bg" :latitude="navLat" :longitude="navLng"
					:scale="hasDestination ? 17 : 16" :enable-3D="true" :show-compass="false"
					:show-location="true" :enable-zoom="true" :enable-scroll="true"
					:enable-rotate="hasDestination" :rotate="navHeading"
					:markers="navMarkers" :polyline="navRoutePolyline" :enable-traffic="true"
					:subkey="tencentMapKey" @updated="onMapUpdated" />
				<view v-else class="map-placeholder">
					<text class="map-placeholder-text">等待定位...</text>
				</view>
				<view class="nav-overlay" v-if="hasDestination">
					<template v-if="currentStep">
						<text class="nav-turn-icon">{{ turnIcon }}</text>
						<view class="nav-info">
							<text class="dist">{{ currentStepDistance }}</text>
							<text class="road">{{ currentStep.road_name || currentStep.instruction }}</text>
						</view>
					</template>
					<template v-else>
						<text class="arrow">↑</text>
						<view class="nav-info">
							<text class="dist">{{ navDistanceKm }} km 后</text>
							<text class="road">{{ destinationName }}</text>
						</view>
					</template>
				</view>
				<view class="map-bar" v-if="hasDestination">
					<text class="bar-item">{{ etaTime }} 到达</text>
					<text class="bar-item highlight">{{ etaMinutes }} 分钟</text>
					<text class="bar-item">{{ navDistanceKm }} 公里</text>
				</view>
			</view>

			<!-- 右上：时速 -->
			<view class="card-box speed-panel">
				<view class="speed-value">
					<text class="num">{{ Math.round(speed) }}</text>
					<text class="unit">km/h</text>
				</view>
				<view class="speed-limits">
					<view class="cruise" v-if="cruiseSetSpeed > 0">
						<Icon name="Navigate" :size="10" color="#00a2ff" />
						<text>{{ cruiseSetSpeed }}</text>
					</view>
					<text class="limit" v-if="speedLimit > 0">{{ speedLimit }}</text>
				</view>
				<view class="mini-trip">
					<text>{{ tripDistance }} km</text>
					<text class="split">|</text>
					<text>{{ tripEfficiency }} Wh/km</text>
				</view>
			</view>

			<!-- 右上右：电量车况 -->
			<view class="card-box right-panel">
				<view class="car-info">
					<text class="model">{{ vehicleModelLabel }}</text>
					<text class="chk-ok">● {{ vehicleStatusText }}</text>
				</view>
				<view class="battery-box">
					<view class="soc-header">
						<text class="soc-num">{{ batteryPercent }}<text class="pct">%</text></text>
						<text class="range">{{ rangeKm }} km</text>
					</view>
					<view class="battery-bar">
						<view class="battery-fill" :style="{ width: batteryPercent + '%', background: batteryGradient }"></view>
					</view>
				</view>
				<view class="status-shortcuts">
					<view class="ico" :class="{ active: locked }" @click="toggleLock">
						<Icon :name="locked ? 'LockClosed' : 'LockOpen'" :size="16" :color="locked ? '#00a2ff' : '#5c6e88'" />
					</view>
					<view class="ico" :class="{ active: climateOn }" @click="toggleClimate">
						<Icon name="Snow" :size="16" :color="climateOn ? '#00a2ff' : '#5c6e88'" />
					</view>
					<view class="ico" :class="{ active: sentryOn }" @click="toggleSentry">
						<Icon name="Shield" :size="16" :color="sentryOn ? '#00a2ff' : '#5c6e88'" />
					</view>
				</view>
			</view>

			<!-- 右下左：ADAS -->
			<view class="card-box adas-panel">
				<view class="adas-view">
					<view class="road-line line-l"></view>
					<image class="ego-car" src="/static/dashboard/car_rear.png" mode="aspectFit"
						:style="{ transform: `translateX(${laneOffset}px)` }"></image>
					<view class="road-line line-r"></view>
				</view>
				<view class="adas-indicators">
					<view class="ind" :class="{ active: leftIndicator }">
						<Icon name="ChevronBack" :size="12" :color="leftIndicator ? '#facc15' : '#394556'" />
					</view>
					<view class="ind" :class="{ active: rightIndicator }">
						<Icon name="ChevronForward" :size="12" :color="rightIndicator ? '#facc15' : '#394556'" />
					</view>
					<view class="ind" :class="{ active: highBeam }">
						<Icon name="Flash" :size="12" :color="highBeam ? '#60a5fa' : '#394556'" />
					</view>
					<view class="ind" :class="{ active: autoHold }">
						<text class="ah-text" :style="{ color: autoHold ? '#4ade80' : '#394556' }">A</text>
					</view>
				</view>
			</view>

			<!-- 右下右：功率+胎压（原音乐位置） -->
			<view class="card-box power-chart-panel">
				<view class="panel-header">
					<text class="title">功率</text>
					<text class="power-value" :class="{ negative: drivePower < 0 }">{{ drivePower > 0 ? '+' : '' }}{{ Math.round(drivePower) }} kW</text>
				</view>
				<view class="power-body">
					<view class="chart-area">
						<canvas canvas-id="powerChartCanvas" class="power-canvas" id="powerChartCanvas"></canvas>
					</view>
					<view class="tpms-side">
						<view class="tpms-col"><text :class="{ warn: !tireNormal[0] }">{{ tpmsValuesShort[0] }}</text><text :class="{ warn: !tireNormal[2] }">{{ tpmsValuesShort[2] }}</text></view>
						<image class="car-top-mini" src="/static/dashboard/car_top.png" mode="aspectFit"></image>
						<view class="tpms-col"><text :class="{ warn: !tireNormal[1] }">{{ tpmsValuesShort[1] }}</text><text :class="{ warn: !tireNormal[3] }">{{ tpmsValuesShort[3] }}</text></view>
					</view>
				</view>
			</view>

		</view>
	</view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { useVehicleStore } from '@/store/vehicle'
import { useVehicleData, initVehicleData, destroyVehicleData } from '@/utils/vehicle-data'
import { useThemeStore } from '@/store/theme'
import Icon from '@/components/Icon/Icon.vue'
import {
  doorLock, doorUnlock, autoConditioningStart, autoConditioningStop,
  setSentryMode
} from '@/api/control.js'

// ===== 基础状态 =====
const vehicleStore = useVehicleStore()
const vehicleDataStore = useVehicleData()
const themeStore = useThemeStore()
const currentVehicle = computed(() => vehicleStore.currentVehicle)
const vehicleData = computed(() => vehicleDataStore.data)

// ===== 主题判断 =====
const isDarkTheme = computed(() => themeStore.resolvedTheme === 'dark' || themeStore.resolvedTheme === 'visionpro')

// 主题切换时重置地图样式标记
watch(isDarkTheme, () => {
  isStyleSet = false
  setTimeout(() => applyMapDarkStyle(), 300)
})

// ===== 地图墨渊主题（仪表盘始终使用墨渊） =====
const tencentMapKey = import.meta.env.VITE_TENCENT_MAP_KEY || ''
const darkStyleId = import.meta.env.VITE_TENCENT_MAP_STYLE_DARK || '2'
let isStyleSet = false // 防止重复设置
let mapContext = null
const useMapFilter = ref(true) // CSS filter 作为降级，setMapStyle 成功后移除

const onMapUpdated = () => {
  applyMapDarkStyle()
}

// 强制应用地图墨渊主题（仪表盘始终暗色，无需判断主题）
function applyMapDarkStyle() {
  if (isStyleSet) return
  
  try {
    if (!mapContext) {
      mapContext = uni.createMapContext('mapBg', getCurrentInstance())
    }
    if (mapContext?.setMapStyle) {
      mapContext.setMapStyle({
        styleId: darkStyleId,
        success: () => {
          console.log('[HorizontalDashboard] 地图墨渊主题设置成功')
          isStyleSet = true
          useMapFilter.value = false // SDK 暗色生效，移除 CSS 滤镜避免反转折线颜色
        },
        fail: (err) => {
          console.warn('[HorizontalDashboard] 地图主题设置失败，尝试整数参数:', err)
          try {
            mapContext.setMapStyle(parseInt(darkStyleId) || 2)
            isStyleSet = true
            useMapFilter.value = false
          } catch (e) {}
        }
      })
    }
  } catch (e) {
    console.warn('[HorizontalDashboard] applyMapDarkStyle 异常:', e)
  }
}

// ===== 强制横屏+全屏 =====
onMounted(() => {
  // #ifdef APP-PLUS
  plus.screen.lockOrientation('landscape-primary')
  // 全屏沉浸模式，隐藏导航栏
  plus.navigator.setFullscreen(true)
  plus.navigator.hideSystemNavigation()
  // #endif
  uni.setKeepScreenOn({ keepScreenOn: true })
  // 初始化车辆数据
  if (currentVehicle.value?.vin) {
    initVehicleData(currentVehicle.value.vin)
  }
  updateTime()
  timeTimer = setInterval(updateTime, 1000)
  powerTimer = setInterval(() => {
    powerHistory.value.push(drivePower.value)
    if (powerHistory.value.length > 60) powerHistory.value.shift()
    drawPowerChart()
  }, 1000)
  routeTimer = setInterval(() => fetchRoute(), 30000)
  calcCanvasSize()
  setTimeout(() => drawPowerChart(), 300)
  // 延迟强制设置地图墨渊主题
  setTimeout(() => applyMapDarkStyle(), 800)
})

onUnmounted(() => {
  // #ifdef APP-PLUS
  plus.screen.unlockOrientation()
  plus.navigator.setFullscreen(false)
  plus.navigator.showSystemNavigation()
  // #endif
  uni.setKeepScreenOn({ keepScreenOn: false })
  if (timeTimer) clearInterval(timeTimer)
  if (powerTimer) clearInterval(powerTimer)
  if (routeTimer) clearInterval(routeTimer)
  destroyVehicleData()
})

onShow(() => {
  // #ifdef APP-PLUS
  plus.screen.lockOrientation('landscape-primary')
  plus.navigator.setFullscreen(true)
  plus.navigator.hideSystemNavigation()
  // #endif
  if (currentVehicle.value?.vin) {
    initVehicleData(currentVehicle.value.vin)
    setTimeout(() => fetchRoute(), 2000)
  }
  // 重新应用地图暗色主题
  setTimeout(() => applyMapDarkStyle(), 500)
})

onHide(() => {
  // #ifdef APP-PLUS
  plus.screen.unlockOrientation()
  plus.navigator.setFullscreen(false)
  plus.navigator.showSystemNavigation()
  // #endif
  destroyVehicleData()
})

// ===== 时间 =====
const currentTime = ref('')
let timeTimer = null
function updateTime() {
  const now = new Date()
  currentTime.value = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`
}

// ===== 顶部状态栏数据 =====
const signalLevel = computed(() => {
  const lat = vehicleDataStore.commandLatencyMs
  if (!lat || lat < 0) return 2
  if (lat < 100) return 4
  if (lat < 300) return 3
  if (lat < 1000) return 2
  return 1
})
const isReady = computed(() => vehicleData.value?.drive_rail === true)
const currentGear = computed(() => vehicleData.value?.gear || 'P')
const autopilotEnabled = computed(() => {
  const so = vehicleDataStore.stateOutput
  return so?.drive?.autopilot_state === 'Enabled'
})
const outsideTempDisplay = computed(() => {
  const t = vehicleData.value?.outside_temp
  return t !== null && t !== undefined ? Math.round(t) : '--'
})

// ===== 左侧速度数据 =====
const speed = computed(() => vehicleData.value?.speed || 0)
const cruiseSetSpeed = computed(() => vehicleData.value?.cruise_set_speed || 0)
const speedLimit = computed(() => vehicleData.value?.current_limit_mph || 0)
const tripDistance = computed(() => {
  const v = vehicleData.value?.miles_since_reset
  if (v !== null && v !== undefined) return (v * 1.60934).toFixed(1)
  return '0.0'
})
const tripEfficiency = computed(() => {
  const so = vehicleDataStore.stateOutput
  if (so?.drive?.efficiency_wh_per_km) return Math.round(so.drive.efficiency_wh_per_km)
  return 0
})

// ===== 中间导航数据 =====
const hasLocation = computed(() => {
  const lat = vehicleData.value?.latitude
  const lng = vehicleData.value?.longitude
  return lat && lng && lat !== 0 && lng !== 0
})
const navLat = computed(() => vehicleData.value?.latitude || 39.9042)
const navLng = computed(() => vehicleData.value?.longitude || 116.4074)

const hasDestination = computed(() => {
  const lat = vehicleData.value?.destination_latitude
  const lng = vehicleData.value?.destination_longitude
  return lat && lng && lat !== 0 && lng !== 0
})
const destinationLat = computed(() => vehicleData.value?.destination_latitude || 0)
const destinationLng = computed(() => vehicleData.value?.destination_longitude || 0)
const destinationName = computed(() => vehicleData.value?.destination_name || '目的地')

const navMarkers = computed(() => {
  const markers = []
  // 有目的地时不显示车辆位置标记（show-location 已显示定位点）
  if (hasDestination.value) {
    markers.push({
      id: 2, latitude: destinationLat.value, longitude: destinationLng.value,
      title: destinationName.value, iconPath: '/static/car-marker.png',
      width: 24, height: 24, anchor: { x: 0.5, y: 1 },
      callout: {
        content: destinationName.value, color: '#fff', fontSize: 12,
        bgColor: 'rgba(4,9,22,0.85)', borderRadius: 4, padding: 4,
        display: 'ALWAYS', textAlign: 'center'
      }
    })
  }
  return markers
})

// 导航路线
const routePoints = ref([])
const navSteps = ref([])
const routeDistance = ref(0) // 腾讯路线规划返回的距离（米）
const routeDuration = ref(0) // 腾讯路线规划返回的时间（秒）

// 车头方向（用于地图旋转跟随车头）
const navHeading = computed(() => {
  const h = vehicleData.value?.heading
  return h != null ? Math.round(h) : 0
})

const navRoutePolyline = computed(() => {
  if (!hasDestination.value) return []
  if (routePoints.value.length > 1) {
    return [{
      points: routePoints.value,
      color: '#00a2ff', width: 6, arrowLine: true,
      borderColor: '#0066cc', borderWidth: 2
    }]
  }
  // 降级：直线虚线
  return [{
    points: [
      { latitude: navLat.value, longitude: navLng.value },
      { latitude: destinationLat.value, longitude: destinationLng.value }
    ],
    color: '#00a2ff', width: 4, arrowLine: true, dottedLine: true
  }]
})

const navDistanceKm = computed(() => {
  // 优先使用腾讯路线规划的精确距离
  if (routeDistance.value > 0) return (routeDistance.value / 1000).toFixed(1)
  const miles = vehicleData.value?.miles_to_arrival || 0
  return (miles * 1.60934).toFixed(1)
})
const etaTime = computed(() => {
  const mins = routeDuration.value > 0
    ? Math.round(routeDuration.value / 60)
    : (vehicleData.value?.minutes_to_arrival || 0)
  if (mins <= 0) return '--:--'
  const now = new Date()
  now.setMinutes(now.getMinutes() + Math.round(mins))
  return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`
})
const etaMinutes = computed(() => {
  if (routeDuration.value > 0) return Math.round(routeDuration.value / 60)
  return Math.round(vehicleData.value?.minutes_to_arrival || 0)
})

// 当前导航步骤（根据车辆位置匹配最近的步骤）
const currentStepIdx = ref(0)

const currentStep = computed(() => {
  if (navSteps.value.length === 0) return null
  return navSteps.value[currentStepIdx.value] || navSteps.value[0] || null
})

const currentStepDistance = computed(() => {
  if (!currentStep.value) return ''
  const d = currentStep.value.distance || 0
  if (d >= 1000) return (d / 1000).toFixed(1) + ' km'
  return d + ' m'
})

// 转向图标映射
const turnIcon = computed(() => {
  if (!currentStep.value) return '↑'
  const act = currentStep.value.act_desc || ''
  const dir = currentStep.value.dir_desc || ''
  if (act.includes('左转掉头') || act.includes('左后转')) return '↩'
  if (act.includes('右后转')) return '↪'
  if (act.includes('左转')) return '←'
  if (act.includes('右转')) return '→'
  if (act.includes('偏左转') || act.includes('靠左')) return '↖'
  if (act.includes('偏右转') || act.includes('靠右')) return '↗'
  if (act.includes('直行')) return '↑'
  if (act.includes('环岛')) return '⟳'
  if (dir.includes('东')) return '→'
  if (dir.includes('西')) return '←'
  if (dir.includes('南')) return '↓'
  if (dir.includes('北')) return '↑'
  return '↑'
})

// 腾讯地图 polyline 解压
// 腾讯地图返回的 polyline 是压缩坐标：第一个点是绝对坐标(lat,lng)，后续点是相对偏移量(需除以1e6)
function decodePolyline(polyline) {
  if (!polyline || polyline.length < 2) return []
  const points = []
  let lat = polyline[0]
  let lng = polyline[1]
  points.push({ latitude: lat, longitude: lng })
  for (let i = 2; i < polyline.length; i += 2) {
    lat += polyline[i] / 1e6
    lng += polyline[i + 1] / 1e6
    points.push({ latitude: lat, longitude: lng })
  }
  return points
}

// 路线规划
async function fetchRoute() {
  if (!hasDestination.value || !hasLocation.value) return
  const key = import.meta.env.VITE_TENCENT_MAP_KEY
  if (!key) return
  try {
    const from = `${navLat.value},${navLng.value}`
    const to = `${destinationLat.value},${destinationLng.value}`
    const heading = vehicleData.value?.heading || 0
    const url = `https://apis.map.qq.com/ws/direction/v1/driving/?from=${from}&to=${to}&heading=${heading}&key=${key}&output=json`
    const res = await new Promise((resolve, reject) => {
      uni.request({ url, method: 'GET', success: resolve, fail: reject })
    })
    if (res.statusCode === 200 && res.data?.status === 0 && res.data?.result?.routes?.length) {
      const route = res.data.result.routes[0]
      // 解压 polyline
      const points = decodePolyline(route.polyline)
      if (points.length > 1) routePoints.value = points
      // 解析导航步骤
      if (route.steps && route.steps.length > 0) {
        navSteps.value = route.steps
        currentStepIdx.value = 0
      }
      // 保存路线距离和时间
      routeDistance.value = route.distance || 0
      routeDuration.value = route.duration || 0
    }
  } catch (e) {
    console.warn('[HorizontalDashboard] 路线规划失败:', e)
  }
}

// 车辆位置变化时更新当前导航步骤
watch([navLat, navLng], () => {
  if (navSteps.value.length > 1 && routePoints.value.length > 1) {
    updateCurrentStep()
  }
})

// 根据车辆位置匹配当前导航步骤
function updateCurrentStep() {
  if (!navSteps.value.length || routePoints.value.length < 2) return
  const curLat = navLat.value
  const curLng = navLng.value
  // 找到路线中距离车辆最近的点索引
  let minDist = Infinity
  let closestIdx = 0
  for (let i = 0; i < routePoints.value.length; i++) {
    const p = routePoints.value[i]
    const d = Math.abs(p.latitude - curLat) + Math.abs(p.longitude - curLng)
    if (d < minDist) {
      minDist = d
      closestIdx = i
    }
  }
  // 根据最近点索引找到对应的步骤
  for (let i = navSteps.value.length - 1; i >= 0; i--) {
    const step = navSteps.value[i]
    const startIdx = step.polyline_idx?.[0] || 0
    if (closestIdx >= startIdx) {
      currentStepIdx.value = i
      break
    }
  }
}

// 目的地变化时自动规划路线
watch(hasDestination, (val) => {
  if (val) {
    routePoints.value = []
    navSteps.value = []
    currentStepIdx.value = 0
    routeDistance.value = 0
    routeDuration.value = 0
    setTimeout(() => fetchRoute(), 500)
  } else {
    routePoints.value = []
    navSteps.value = []
    currentStepIdx.value = 0
    routeDistance.value = 0
    routeDuration.value = 0
  }
})

// 目的地坐标变化时重新规划
watch([destinationLat, destinationLng], () => {
  if (hasDestination.value) {
    routePoints.value = []
    navSteps.value = []
    currentStepIdx.value = 0
    routeDistance.value = 0
    routeDuration.value = 0
    setTimeout(() => fetchRoute(), 500)
  }
})

// ===== 右侧车辆状态 =====
const vehicleModelLabel = computed(() => {
  return currentVehicle.value?.display_name || currentVehicle.value?.vehicle_name || 'Tesla'
})
const vehicleStatusText = computed(() => {
  if (vehicleData.value?.charging) return '充电中'
  if (vehicleData.value?.driving) return '行驶中'
  if (isReady.value) return '系统就绪'
  return '待机中'
})
const batteryPercent = computed(() => Math.round(vehicleData.value?.soc || 0))
const rangeKm = computed(() => Math.round(vehicleData.value?.range_km || 0))
const batteryGradient = computed(() => {
  const p = batteryPercent.value
  if (p > 60) return 'linear-gradient(90deg, #00bfff, #00ff88)'
  if (p >= 20) return 'linear-gradient(90deg, #facc15, #ff8c00)'
  return 'linear-gradient(90deg, #ef4444, #ff6b6b)'
})
const locked = computed(() => vehicleData.value?.locked !== false)
const climateOn = computed(() => vehicleData.value?.is_ac_on === true)
const sentryOn = computed(() => vehicleData.value?.sentry_mode === true)

// ===== 底部功率数据 =====
const drivePower = computed(() => vehicleData.value?.power || 0)
const powerHistory = ref([])

// ===== ADAS =====
const laneOffset = computed(() => {
  const latAccel = vehicleData.value?.lateral_acceleration || 0
  return Math.max(-15, Math.min(15, latAccel * 8))
})
const leftIndicator = computed(() => vehicleData.value?.lights_turn_signal === 'left')
const rightIndicator = computed(() => vehicleData.value?.lights_turn_signal === 'right')
const highBeam = computed(() => vehicleData.value?.lights_high_beams === true)
const autoHold = computed(() => {
  const so = vehicleDataStore.stateOutput
  return so?.drive?.auto_hold === true
})

// ===== 胎压 =====
const tpmsValues = computed(() => {
  const d = vehicleData.value
  return [d?.tpms_fl || 0, d?.tpms_fr || 0, d?.tpms_rl || 0, d?.tpms_rr || 0]
})
const tireNormal = computed(() => tpmsValues.value.map(p => p >= 2.0 && p <= 3.2))
const tpmsValuesShort = computed(() => tpmsValues.value.map(v => v ? v.toFixed(1) : '0.0'))

// ===== 命令执行 =====
const currentVIN = computed(() => currentVehicle.value?.vin)
const executeCommand = async (commandFn, commandName) => {
  if (!currentVIN.value) {
    uni.showToast({ title: '请先选择车辆', icon: 'none' })
    return false
  }
  uni.showLoading({ title: '执行中...' })
  try {
    await commandFn(currentVIN.value)
    uni.hideLoading()
    uni.showToast({ title: `${commandName}成功`, icon: 'success' })
    return true
  } catch (err) {
    uni.hideLoading()
    uni.showToast({ title: err.message || `${commandName}失败`, icon: 'none' })
    return false
  }
}
const toggleLock = async () => {
  if (locked.value) await executeCommand(doorUnlock, '解锁')
  else await executeCommand(doorLock, '上锁')
}
const toggleClimate = async () => {
  if (climateOn.value) await executeCommand(autoConditioningStop, '关闭空调')
  else await executeCommand(autoConditioningStart, '开启空调')
}
const toggleSentry = async () => {
  if (currentVIN.value) {
    const newState = !sentryOn.value
    await executeCommand((vin) => setSentryMode(vin, newState), newState ? '开启哨兵' : '关闭哨兵')
  }
}

// ===== 功率图表 Canvas =====
const chartW = ref(200)
const chartH = ref(60)
let powerTimer = null
let routeTimer = null

function calcCanvasSize() {
  const sysInfo = uni.getSystemInfoSync()
  const w = Math.max(sysInfo.windowWidth, sysInfo.windowHeight)
  const h = Math.min(sysInfo.windowWidth, sysInfo.windowHeight)
  chartW.value = Math.floor(w * 0.35)
  chartH.value = Math.floor(h * 0.22)
}

function drawPowerChart() {
  const ctx = uni.createCanvasContext('powerChartCanvas')
  if (!ctx) return
  const w = chartW.value
  const h = chartH.value
  const pad = 4

  ctx.clearRect(0, 0, w, h)

  // 零线
  ctx.beginPath()
  ctx.moveTo(pad, h / 2)
  ctx.lineTo(w - pad, h / 2)
  ctx.lineWidth = 0.5
  ctx.strokeStyle = 'rgba(255,255,255,0.1)'
  ctx.stroke()

  const history = powerHistory.value
  if (history.length > 1) {
    const maxP = 80
    const minP = -80
    const range = maxP - minP

    ctx.beginPath()
    history.forEach((p, i) => {
      const x = pad + (i / (history.length - 1)) * (w - pad * 2)
      const normalized = (Math.max(minP, Math.min(maxP, p)) - minP) / range
      const y = pad + (1 - normalized) * (h - pad * 2)
      if (i === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    })
    ctx.lineWidth = 1.5
    ctx.strokeStyle = '#00ff88'
    ctx.stroke()

    // 填充
    const lastX = pad + (w - pad * 2)
    ctx.lineTo(lastX, h / 2)
    ctx.lineTo(pad, h / 2)
    ctx.closePath()
    ctx.fillStyle = 'rgba(0,255,136,0.06)'
    ctx.fill()
  }

  ctx.draw()
}

watch(() => powerHistory.value.length, () => drawPowerChart())
</script>

<style lang="scss" scoped>
.cyber-dashboard {
	width: 100vw;
	height: 100vh;
	background-color: #040814;
	color: #ffffff;
	font-family: Arial, Helvetica, sans-serif;
	display: flex;
	flex-direction: column;
	overflow: hidden;
	box-sizing: border-box;
}

/* 1. 顶栏 */
.status-bar {
	height: 44rpx;
	padding: 0 30rpx;
	display: flex;
	justify-content: space-between;
	align-items: center;
	background-color: #060d20;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
	font-size: 20rpx;
	box-sizing: border-box;

	.status-left, .status-right {
		display: flex;
		align-items: center;
		gap: 15rpx;
		color: #7a8ba4;
		.time { color: #ffffff; font-weight: bold; }
	}
	
	.signal-bars {
		display: flex;
		align-items: flex-end;
		gap: 2rpx;
		height: 16rpx;
		.bar {
			width: 4rpx;
			background: rgba(255,255,255,0.15);
			border-radius: 1rpx;
			&:nth-child(1) { height: 4rpx; }
			&:nth-child(2) { height: 8rpx; }
			&:nth-child(3) { height: 12rpx; }
			&:nth-child(4) { height: 16rpx; }
			&.active { background: #00ff88; }
		}
	}
	
	.status-center {
		display: flex;
		align-items: center;
		gap: 40rpx;
		
		.ready-tag { color: #394556; font-weight: bold; letter-spacing: 2rpx;
			&.active { color: #00ff88; }
		}
		.gears {
			display: flex; gap: 15rpx; color: #394556; font-weight: bold;
			text.active { color: #ffffff; text-shadow: 0 0 8rpx #ffffff; }
		}
	}
	
	.ap-status {
		color: #00ff88; font-weight: bold; font-size: 18rpx;
		padding: 2rpx 8rpx; border: 1rpx solid #00ff88; border-radius: 4rpx;
	}
}

/* 2. 主区域栅格 */
.main-container {
	height: calc(100vh - 44rpx);
	padding: 12rpx;
	display: grid;
	grid-template-columns: 1fr 1fr 1fr;
	grid-template-rows: 1fr 1fr;
	gap: 12rpx;
	box-sizing: border-box;
	overflow: hidden;
}

/* 3. 通用卡片 */
.card-box {
	background: linear-gradient(135deg, #091329 0%, #050b18 100%);
	border: 1px solid #162a4e;
	border-radius: 8rpx;
	position: relative;
	overflow: hidden;
	display: flex;
	flex-direction: column;
	box-sizing: border-box;
}

/* ================= 左侧：地图（跨两行） ================= */
.map-panel {
	grid-row: 1 / 3;
	grid-column: 1;
	
	.map-bg { width: 100%; height: 100%; position: absolute; z-index: 1; }
	
	// 暗色主题 CSS 滤镜（第三重保障）
	&.map-dark-filter .map-bg {
		filter: invert(90%) hue-rotate(180deg) brightness(0.95) contrast(0.9);
		transition: filter 0.3s ease;
	}
	
	.map-placeholder {
		width: 100%; height: 100%;
		display: flex; align-items: center; justify-content: center;
		.map-placeholder-text { font-size: 24rpx; color: #394556; }
	}
	
	.nav-overlay {
		position: absolute; top: 12rpx; left: 12rpx; z-index: 2;
		background: rgba(4, 9, 22, 0.9); border: 1px solid #00a2ff;
		border-radius: 6rpx; padding: 8rpx 15rpx;
		display: flex; align-items: center; gap: 12rpx;
		
		.arrow { font-size: 30rpx; color: #00a2ff; font-weight: bold; }
		.nav-turn-icon { font-size: 36rpx; color: #00ff88; font-weight: bold; }
		.nav-info {
			display: flex; flex-direction: column;
			.dist { font-size: 20rpx; font-weight: bold; }
			.road { font-size: 16rpx; color: #7a8ba4; max-width: 200rpx; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
		}
	}
	
	.map-bar {
		position: absolute; bottom: 0; left: 0; right: 0; z-index: 2;
		height: 54rpx; background: rgba(4, 9, 22, 0.95);
		display: flex; align-items: center; justify-content: space-around;
		font-size: 18rpx; border-top: 1px solid rgba(255, 255, 255, 0.05);
		color: #7a8ba4;
		.highlight { color: #00ff88; font-weight: bold; }
	}
}

/* ================= 右上左：时速 ================= */
.speed-panel {
	justify-content: space-between;
	padding: 15rpx;
	
	.speed-value {
		display: flex;
		align-items: baseline;
		justify-content: center;
		margin-top: 10rpx;
		.num { font-size: 76rpx; font-weight: bold; line-height: 1; font-family: sans-serif; }
		.unit { font-size: 18rpx; color: #5c6e88; margin-left: 6rpx; }
	}
	
	.speed-limits {
		display: flex;
		justify-content: center;
		gap: 30rpx;
		font-size: 20rpx;
		.cruise {
			display: flex; align-items: center; gap: 4rpx;
			color: #00a2ff; font-weight: bold;
		}
		.limit { 
			color: #ff4d4d; border: 2rpx solid #ff4d4d; border-radius: 50%;
			width: 28rpx; height: 28rpx; text-align: center; line-height: 24rpx; font-size: 16rpx;
		}
	}
	
	.mini-trip {
		border-top: 1px solid rgba(255, 255, 255, 0.05);
		padding-top: 10rpx;
		display: flex;
		justify-content: center;
		gap: 15rpx;
		font-size: 16rpx;
		color: #7a8ba4;
		.split { color: #162a4e; }
	}
}

/* ================= 右上右：电量车况 ================= */
.right-panel {
	justify-content: space-between;
	padding: 12rpx;
	
	.car-info {
		display: flex;
		justify-content: space-between;
		align-items: center;
		.model { font-size: 18rpx; font-weight: bold; color: #7a8ba4; letter-spacing: 2rpx; }
		.chk-ok { font-size: 14rpx; color: #00ff88; }
	}
	
	.battery-box {
		.soc-header {
			display: flex;
			justify-content: space-between;
			align-items: baseline;
			.soc-num { font-size: 32rpx; font-weight: bold; .pct { font-size: 16rpx; } }
			.range { font-size: 16rpx; color: #00ff88; font-weight: bold; }
		}
		.battery-bar {
			height: 6rpx; background: #101f38; border-radius: 3rpx; margin-top: 6rpx; overflow: hidden;
			.battery-fill { height: 100%; border-radius: 3rpx; transition: width 0.5s ease; }
		}
	}
	
	.status-shortcuts {
		display: flex; justify-content: space-around;
		border-top: 1px solid rgba(255, 255, 255, 0.05); padding-top: 8rpx;
		.ico { opacity: 0.3; 
			&.active { opacity: 1; text-shadow: 0 0 8rpx #00a2ff; }
		}
	}
}

/* ================= 下左：功率+胎压（左右各半） ================= */
.power-chart-panel {
	padding: 10rpx;
	
	.panel-header {
		display: flex; justify-content: space-between; align-items: center; font-size: 16rpx; flex-shrink: 0;
		.title { color: #5c6e88; }
		.power-value { font-size: 14rpx; font-weight: bold; color: #00ff88;
			&.negative { color: #facc15; }
		}
	}
	
	.power-body {
		flex: 1; display: flex; flex-direction: row; padding-top: 4rpx; overflow: hidden;
	}
	
	.chart-area {
		flex: 1; display: flex; min-width: 0;
	}
	
	.power-canvas {
		width: 100%;
		height: 100%;
	}
	
	.tpms-side {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: space-around;
		border-left: 1px solid rgba(255, 255, 255, 0.05);
		margin-left: 8rpx;
		padding-left: 8rpx;
		
		.car-top-mini { width: 28rpx; height: 50rpx; }
		.tpms-col {
			display: flex; flex-direction: column; justify-content: space-around; height: 80%;
			font-size: 13rpx; font-weight: bold; color: #00ff88;
			.warn { color: #ff4d4d; }
		}
	}
}

/* ================= 下中：ADAS ================= */
.adas-panel {
	justify-content: center;
	align-items: center;
	
	.adas-view {
		width: 80%; flex: 1; position: relative; display: flex; justify-content: center; align-items: center;
		
		.road-line {
			position: absolute; bottom: 0; width: 4rpx; height: 80%; background: rgba(0, 162, 255, 0.3);
			border-radius: 2rpx;
		}
		.line-l { left: 20rpx; transform: rotate(8deg); border-left: 2rpx dashed #00a2ff; }
		.line-r { right: 20rpx; transform: rotate(-8deg); border-right: 2rpx dashed #00a2ff; }
		
		.ego-car { width: 50rpx; height: 70rpx; z-index: 2; transition: transform 0.3s ease; }
	}
	
	.adas-indicators {
		display: flex; justify-content: center; gap: 8rpx; padding: 6rpx 0;
		.ind {
			width: 28rpx; height: 28rpx;
			display: flex; align-items: center; justify-content: center;
			border-radius: 4rpx;
			background: rgba(255, 255, 255, 0.04);
			&.active { background: rgba(255, 255, 255, 0.08); }
		}
		.ah-text { font-size: 12rpx; font-weight: 700; }
	}
}
</style>
