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
					:enable-rotate="true" :rotate="navHeading"
					:markers="navMarkers" :polyline="navRoutePolyline" :enable-traffic="true"
					:subkey="tencentMapKey" @updated="onMapUpdated" />
				<view v-else class="map-placeholder">
					<text class="map-placeholder-text">等待定位...</text>
				</view>

				<!-- 导航模式：导航指令横幅 -->
				<view class="navi-banner" v-if="naviActive && hasDestination">
					<text class="navi-instruction">{{ naviInstruction || destinationName }}</text>
				</view>

				<!-- 非导航模式：小号导航提示 -->
				<view class="nav-overlay" v-if="!naviActive && hasDestination">
					<template v-if="currentStep">
						<view class="nav-info">
							<text class="dist">{{ currentStepDistance }}</text>
							<text class="road">{{ currentStep.road_name || currentStep.instruction || naviInstruction }}</text>
						</view>
					</template>
					<template v-else>
						<view class="nav-info">
							<text class="dist">{{ navDistanceKm }} km 后</text>
							<text class="road">{{ destinationName }}</text>
						</view>
					</template>
				</view>

				<view class="map-bar" v-if="hasDestination">
					<view class="bar-info">
						<text class="bar-row">{{ etaTime }} 到达，{{ etaRemaining }}</text>
						<text class="bar-row">{{ navDistanceKm }} km</text>
					</view>
					<view class="bar-voice-btn" :class="{ muted: !naviVoiceEnabled }" @click="naviVoiceEnabled = !naviVoiceEnabled">
						<text class="voice-icon">{{ naviVoiceEnabled ? '🔊' : '🔇' }}</text>
					</view>
				</view>
			</view>

			<!-- 右上：时速 -->
			<view class="card-box speed-panel">
				<view class="speed-top-row">
					<view class="cruise" :class="{ active: autopilotEnabled }" v-if="autopilotEnabled && cruiseSetSpeed > 0">
						<text>{{ cruiseSetSpeed }}</text>
					</view>
					<view class="limit">
						<text v-if="speedLimit > 0">{{ speedLimit }}</text>
					</view>
				</view>
				<view class="speed-value">
					<text class="num">{{ Math.round(speed) }}</text>
					<text class="unit">km/h</text>
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
						<Icon :name="locked ? 'LockClosed' : 'LockOpen'" :size="16" :color="locked ? '#fff' : '#555'" />
					</view>
					<view class="ico" :class="{ active: climateOn }" @click="toggleClimate">
						<Icon name="Snow" :size="16" :color="climateOn ? '#fff' : '#555'" />
					</view>
					<view class="ico" :class="{ active: sentryOn }" @click="toggleSentry">
						<Icon name="Shield" :size="16" :color="sentryOn ? '#fff' : '#555'" />
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
						<Icon name="ChevronBack" :size="12" :color="leftIndicator ? '#e02020' : '#333'" />
					</view>
					<view class="ind" :class="{ active: rightIndicator }">
						<Icon name="ChevronForward" :size="12" :color="rightIndicator ? '#e02020' : '#333'" />
					</view>
					<view class="ind" :class="{ active: highBeam }">
						<Icon name="Flash" :size="12" :color="highBeam ? '#fff' : '#333'" />
					</view>
					<view class="ind" :class="{ active: autoHold }">
						<text class="ah-text" :style="{ color: autoHold ? '#e02020' : '#333' }">A</text>
					</view>
				</view>
			</view>

			<!-- 右下右：功率+胎压（原音乐位置） -->
			<view class="card-box power-chart-panel">
				<view class="panel-header">
					<text class="title">功率</text>
					<text class="power-value" :class="{ negative: drivePower < 0, high: Math.abs(drivePower) > 20 }">{{ drivePower > 0 ? '+' : '' }}{{ Math.round(drivePower) }} kW</text>
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
import { ref, computed, onMounted, onUnmounted, watch, nextTick, getCurrentInstance } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { useVehicleStore } from '@/store/vehicle'
import { useVehicleData, initVehicleData, destroyVehicleData } from '@/utils/vehicle-data'
import { useThemeStore } from '@/store/theme'
import { getDisplayStateLabel } from '@/utils/vehicle-state'
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
  if (naviCheckTimer) clearInterval(naviCheckTimer)
  stopNaviVoice()
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
// 限速值来自 Tesla 车辆数据（腾讯地图不支持道路限速）
const speedLimit = computed(() => vehicleData.value?.current_limit_mph || 0)
const tripDistance = computed(() => {
  const v = vehicleData.value?.miles_since_reset
  if (v !== null && v !== undefined) return (v * 1.60934).toFixed(1)
  return '0.0'
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
  // 车辆当前位置标记（汽车图标）
  if (hasLocation.value) {
    markers.push({
      id: 1, latitude: navLat.value, longitude: navLng.value,
      iconPath: '/static/car-marker.png',
      width: 32, height: 32, anchor: { x: 0.5, y: 0.5 },
      rotate: navHeading.value
    })
  }
  // 目的地标记
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
      color: '#e02020', width: 6, arrowLine: true,
      borderColor: '#8b0000', borderWidth: 2
    }]
  }
  // 降级：直线虚线
  return [{
    points: [
      { latitude: navLat.value, longitude: navLng.value },
      { latitude: destinationLat.value, longitude: destinationLng.value }
    ],
    color: '#e02020', width: 4, arrowLine: true, dottedLine: true
  }]
})

const navDistanceKm = computed(() => {
  const miles = vehicleData.value?.miles_to_arrival || 0
  if (miles > 0) return (miles * 1.60934).toFixed(1)
  if (routeDistance.value > 0) return (routeDistance.value / 1000).toFixed(1)
  return '0.0'
})
const etaTime = computed(() => {
  const mins = vehicleData.value?.minutes_to_arrival || 0
  if (mins > 0) {
    const now = new Date()
    now.setMinutes(now.getMinutes() + Math.round(mins))
    return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`
  }
  if (routeDuration.value > 0) {
    const now = new Date()
    now.setMinutes(now.getMinutes() + Math.round(routeDuration.value / 60))
    return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`
  }
  return '--:--'
})
const etaMinutes = computed(() => {
  const mins = vehicleData.value?.minutes_to_arrival || 0
  if (mins > 0) return Math.round(mins)
  if (routeDuration.value > 0) return Math.round(routeDuration.value / 60)
  return 0
})
const etaRemaining = computed(() => {
  const mins = etaMinutes.value
  if (mins <= 0) return '0分钟'
  if (mins < 60) return `${mins}分钟`
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return m > 0 ? `${h}小时${m}分钟` : `${h}小时`
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

// 导航指令文本（类似高德/百度导航格式）
const naviInstruction = computed(() => {
  if (!currentStep.value) return ''
  const step = currentStep.value
  const act = step.act_desc || ''
  const road = step.road?.name || step.road_name || ''
  const dist = step.distance || 0

  // 构建距离文本
  const distText = dist >= 1000 ? (dist / 1000).toFixed(1) + '公里' : dist + '米'

  // 根据动作类型生成指令
  if (act.includes('到达')) return '到达目的地'
  if (act.includes('左转掉头') || act.includes('左后转')) return road ? `掉头后进入${road}` : '掉头'
  if (act.includes('右后转')) return road ? `右转掉头进入${road}` : '右转掉头'
  if (act.includes('左转') && act.includes('偏')) return road ? `靠左行驶${distText}进入${road}` : `靠左行驶${distText}`
  if (act.includes('右转') && act.includes('偏')) return road ? `靠右行驶${distText}进入${road}` : `靠右行驶${distText}`
  if (act.includes('左转')) return road ? `行驶${distText}后左转进入${road}` : `行驶${distText}后左转`
  if (act.includes('右转')) return road ? `行驶${distText}后右转进入${road}` : `行驶${distText}后右转`
  if (act.includes('环岛')) return road ? `进入环岛，驶出后进入${road}` : '进入环岛'

  // 直行
  if (road) return `沿${road}行驶${distText}`
  return `直行${distText}`
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

// ===== 增强内嵌导航 =====
const naviActive = ref(false)        // 导航模式是否激活
const naviVoiceEnabled = ref(true)   // 语音播报开关
const lastSpokenStep = ref(-1)       // 上次播报的步骤索引，防止重复播报
const lastRerouteTime = ref(0)       // 上次重规划时间
const offRouteDistance = 200         // 偏航判定距离（米）

// 有目的地时自动进入导航模式
watch(hasDestination, (val) => {
  if (val) {
    naviActive.value = true
    startNaviVoice()
  } else {
    naviActive.value = false
    stopNaviVoice()
  }
})

// ===== TTS 语音播报 =====
let ttsEngine = null

function startNaviVoice() {
  // #ifdef APP-PLUS
  console.log('[HorizontalDashboard] startNaviVoice 被调用, ttsEngine:', ttsEngine)
  if (ttsEngine != null) return
  try {
    console.log('[HorizontalDashboard] plus.audio:', typeof plus.audio, 'createSpeech:', typeof plus.audio?.createSpeech)
    // 使用 plus.audio.createSpeech 进行语音合成（TTS）
    if (plus.audio && typeof plus.audio.createSpeech === 'function') {
      console.log('[HorizontalDashboard] 使用 plus.audio.createSpeech 模式')
      ttsEngine = 'audio-tts'
    } else {
      console.warn('[HorizontalDashboard] plus.audio.createSpeech 不可用')
    }
  } catch (e) {
    console.warn('[HorizontalDashboard] TTS初始化失败:', e)
    ttsEngine = null
  }
  console.log('[HorizontalDashboard] startNaviVoice 结束, ttsEngine:', ttsEngine)
  // #endif
}

function stopNaviVoice() {
  // #ifdef APP-PLUS
  if (ttsEngine != null) {
    try {
      if (ttsEngine !== 'audio-tts' && typeof ttsEngine.stop === 'function') {
        ttsEngine.stop()
      }
    } catch (e) {}
    ttsEngine = null
  }
  // #endif
}

function speakNavi(text) {
  console.log('[HorizontalDashboard] speakNavi 被调用:', text, '语音开关:', naviVoiceEnabled.value, '导航激活:', naviActive.value)
  if (!naviVoiceEnabled.value || !naviActive.value) return
  // #ifdef APP-PLUS
  // 懒初始化 TTS 引擎
  if (ttsEngine == null) {
    startNaviVoice()
  }
  try {
    console.log('[HorizontalDashboard] TTS引擎状态:', ttsEngine, 'plus.audio:', typeof plus.audio)
    if (ttsEngine === 'audio-tts') {
      // 使用 plus.audio.createSpeech 进行语音合成
      if (plus.audio && typeof plus.audio.createSpeech === 'function') {
        console.log('[HorizontalDashboard] 使用 plus.audio.createSpeech 播报:', text)
        const speech = plus.audio.createSpeech(text, {
          lang: 'zh-CN',
          rate: 1.0,
          pitch: 1.0,
          volume: 0.8
        })
        if (speech && typeof speech.start === 'function') {
          speech.start()
          speech.onerror = (e) => {
            console.warn('[HorizontalDashboard] TTS播报错误:', e.message)
          }
        }
      } else {
        console.warn('[HorizontalDashboard] plus.audio.createSpeech 不可用')
      }
      return
    }
    if (ttsEngine != null && typeof ttsEngine.speak === 'function') {
      console.log('[HorizontalDashboard] 使用引擎 speak:', text)
      ttsEngine.speak(text)
    } else {
      console.warn('[HorizontalDashboard] 没有可用的 TTS 方法')
    }
  } catch (e) {
    console.warn('[HorizontalDashboard] TTS播报失败:', e)
  }
  // #endif
  // #ifdef H5
  try {
    if ('speechSynthesis' in window) {
      const utterance = new SpeechSynthesisUtterance(text)
      utterance.lang = 'zh-CN'
      utterance.rate = 1.0
      utterance.volume = 0.8
      window.speechSynthesis.cancel()
      window.speechSynthesis.speak(utterance)
    }
  } catch (e) {
    console.warn('[HorizontalDashboard] H5 TTS播报失败:', e)
  }
  // #endif
}

// 监听导航步骤变化，播报语音
watch(currentStepIdx, (newIdx) => {
  console.log('[HorizontalDashboard] currentStepIdx 变化:', newIdx, 'naviActive:', naviActive.value, 'lastSpokenStep:', lastSpokenStep.value)
  if (!naviActive.value || newIdx < 0 || newIdx === lastSpokenStep.value) return
  const step = navSteps.value[newIdx]
  if (!step) return

  lastSpokenStep.value = newIdx
  const instruction = step.instruction || step.act_desc || ''
  const roadName = step.road?.name || step.dir_desc || ''
  const distance = step.distance || 0

  let speakText = ''
  if (distance > 0 && distance < 500) {
    speakText = `${instruction}，${distance}米后${roadName ? '进入' + roadName : ''}`
  } else if (distance >= 500) {
    speakText = instruction
  }

  console.log('[HorizontalDashboard] 准备播报:', speakText, '步骤:', step)
  if (speakText) {
    speakNavi(speakText)
  }
})

// ===== 偏航检测与自动重规划 =====
function checkOffRoute() {
  if (!naviActive.value || !hasLocation.value || !hasDestination.value) return
  if (routePoints.value.length < 2) return

  // 计算当前位置到路线的最近距离
  const currentLat = navLat.value
  const currentLng = navLng.value
  let minDist = Infinity

  // 每隔5个点采样，提高性能
  for (let i = 0; i < routePoints.value.length; i += 5) {
    const p = routePoints.value[i]
    const d = haversineDistance(currentLat, currentLng, p.latitude, p.longitude)
    if (d < minDist) minDist = d
  }

  // 偏航超过阈值，自动重规划
  if (minDist > offRouteDistance) {
    const now = Date.now()
    // 至少30秒间隔才重规划
    if (now - lastRerouteTime.value > 30000) {
      lastRerouteTime.value = now
      console.log('[HorizontalDashboard] 偏航重规划，距路线:', Math.round(minDist), 'm')
      speakNavi('已偏离路线，正在重新规划')
      fetchRoute()
    }
  }
}

// Haversine 距离计算（米）
function haversineDistance(lat1, lng1, lat2, lng2) {
  const R = 6371000
  const dLat = (lat2 - lat1) * Math.PI / 180
  const dLng = (lng2 - lng1) * Math.PI / 180
  const a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(lat1 * Math.PI / 180) * Math.cos(lat2 * Math.PI / 180) *
    Math.sin(dLng / 2) * Math.sin(dLng / 2)
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
  return R * c
}

// ===== 到达检测 =====
function checkArrival() {
  if (!naviActive.value || !hasDestination.value) return
  const dist = navDistanceKm.value
  // 距离小于 50 米判定到达
  if (dist < 0.05) {
    naviActive.value = false
    speakNavi('已到达目的地')
    uni.showToast({ title: '已到达目的地', icon: 'success', duration: 3000 })
  }
}

// 定时检测偏航和到达
let naviCheckTimer = null
watch(naviActive, (active) => {
  if (naviCheckTimer) {
    clearInterval(naviCheckTimer)
    naviCheckTimer = null
  }
  if (active) {
    naviCheckTimer = setInterval(() => {
      checkOffRoute()
      checkArrival()
    }, 10000) // 每10秒检测一次
  }
})

// ===== 右侧车辆状态 =====
const vehicleModelLabel = computed(() => {
  return currentVehicle.value?.display_name || currentVehicle.value?.vehicle_name || 'Tesla'
})
// 使用和首页一样的状态判断逻辑
const stateOutput = computed(() => vehicleDataStore.stateOutput)
const vehicleStatusText = computed(() => getDisplayStateLabel(stateOutput.value, vehicleData.value))
const batteryPercent = computed(() => Math.round(vehicleData.value?.soc || 0))
const rangeKm = computed(() => Math.round(vehicleData.value?.range_km || 0))
const batteryGradient = computed(() => {
  const p = batteryPercent.value
  if (p > 60) return 'linear-gradient(90deg, #ccc, #fff)'
  if (p >= 20) return 'linear-gradient(90deg, #888, #bbb)'
  return 'linear-gradient(90deg, #e02020, #ff6b6b)'
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
    ctx.strokeStyle = '#ffffff'
    ctx.stroke()

    // 填充
    const lastX = pad + (w - pad * 2)
    ctx.lineTo(lastX, h / 2)
    ctx.lineTo(pad, h / 2)
    ctx.closePath()
    ctx.fillStyle = 'rgba(255,255,255,0.06)'
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
	background-color: #000000;
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
	background-color: #0a0a0a;
	font-size: 20rpx;
	box-sizing: border-box;

	.status-left {
		display: flex;
		align-items: center;
		gap: 15rpx;
		color: #666;
		.time { color: #fff; font-weight: bold; }
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
				&.active { background: #e02020; }
			}
		}
	}

	.status-center {
		display: flex;
		align-items: center;
		gap: 40rpx;

		.ready-tag { color: #333; font-weight: bold; letter-spacing: 2rpx;
			&.active { color: #e02020; }
		}
		.gears {
			display: flex; gap: 15rpx; color: #333; font-weight: bold;
			text.active { color: #fff; }
		}
	}

	.status-right {
		display: flex;
		align-items: center;
		gap: 15rpx;
		color: #666;
		.ap-status {
			color: #e02020; font-weight: bold; font-size: 18rpx;
			padding: 2rpx 8rpx; border-radius: 4rpx;
		}
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
	background: linear-gradient(135deg, #111 0%, #0a0a0a 100%);
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

	// 导航模式：导航指令横幅
	.navi-banner {
		position: absolute; top: 0; left: 0; right: 0; z-index: 3;
		height: 40rpx; background: linear-gradient(180deg, rgba(20, 20, 20, 0.97) 0%, rgba(10, 10, 10, 0.92) 100%);
		display: flex; align-items: center;
		padding: 0 20rpx;
		overflow: hidden;

		.navi-instruction {
			font-size: 11rpx; font-weight: bold; color: #fff;
			white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
			width: 100%;
		}
	}

	// 非导航模式：小号导航提示
	.nav-overlay {
		position: absolute; top: 12rpx; left: 12rpx; z-index: 2;
		background: rgba(10, 10, 10, 0.9);
		border-radius: 6rpx; padding: 8rpx 15rpx;
		display: flex; align-items: center; gap: 12rpx;

		.arrow { font-size: 30rpx; color: #e02020; font-weight: bold; }
		.nav-turn-icon { font-size: 36rpx; color: #e02020; font-weight: bold; }
		.nav-info {
			display: flex; flex-direction: column;
			.dist { font-size: 20rpx; font-weight: bold; }
			.road { font-size: 16rpx; color: #666; max-width: 200rpx; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
		}
	}
	
	.map-bar {
		position: absolute; bottom: 0; left: 0; right: 0; z-index: 2;
		background: rgba(10, 10, 10, 0.95);
		display: flex; flex-direction: row; align-items: stretch;
		color: #666;
		.bar-info {
			flex: 1; display: flex; flex-direction: column; justify-content: center;
			padding: 4rpx 16rpx;
		}
		.bar-row {
			font-size: 16rpx; line-height: 1.4;
		}
		.bar-voice-btn {
			display: flex; align-items: center; justify-content: center;
			width: 30rpx; align-self: stretch;
			&:active { opacity: 0.7; }
			&.muted { opacity: 0.5; }
			.voice-icon { font-size: 16rpx; line-height: 1; color: #fff; }
		}
	}
}

/* ================= 右上左：时速 ================= */
.speed-panel {
	justify-content: flex-start;
	padding: 15rpx;
	position: relative;
	
	.speed-top-row {
		display: flex;
		justify-content: flex-start;
		align-items: center;
		width: 100%;
		font-size: 20rpx;
		.cruise {
			display: flex; align-items: center; justify-content: center;
			color: #555; font-weight: bold;
			width: 32rpx; height: 32rpx; border-radius: 50%;
			font-size: 16rpx;
			&.active {
				color: #3e9bf8;
			}
		}
		.limit {
			color: #ff4d4d; border-radius: 50%;
			width: 32rpx; height: 32rpx; font-size: 16rpx;
			font-weight: bold;
			display: flex;
			align-items: center;
			justify-content: center;
			margin-left: auto;
		}
	}
	
	.speed-value {
		flex: 1;
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		.num {
			font-size: 76rpx; font-weight: bold; line-height: 1; font-family: sans-serif;
		}
		.unit {
			font-size: 18rpx; color: #555;
			position: absolute;
			left: 50%;
			margin-left: 50rpx;
		}
	}
	
	.mini-trip {
		padding-top: 10rpx;
		display: flex;
		justify-content: center;
		gap: 15rpx;
		font-size: 16rpx;
		color: #666;
		.split { color: #222; }
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
		.model { font-size: 18rpx; font-weight: bold; color: #666; letter-spacing: 2rpx; }
		.chk-ok { font-size: 14rpx; color: #fff; }
	}
	
	.battery-box {
		.soc-header {
			display: flex;
			justify-content: space-between;
			align-items: baseline;
			.soc-num { font-size: 32rpx; font-weight: bold; .pct { font-size: 16rpx; } }
			.range { font-size: 16rpx; color: #fff; font-weight: bold; }
		}
		.battery-bar {
			height: 6rpx; background: #1a1a1a; border-radius: 3rpx; margin-top: 6rpx; overflow: hidden;
			.battery-fill { height: 100%; border-radius: 3rpx; transition: width 0.5s ease; }
		}
	}
	
	.status-shortcuts {
		display: flex; justify-content: space-around;
		padding-top: 8rpx;
		.ico { opacity: 0.3;
			&.active { opacity: 1; }
		}
	}
}

/* ================= 下左：功率+胎压（左右各半） ================= */
.power-chart-panel {
	padding: 10rpx;
	
	.panel-header {
		display: flex; justify-content: space-between; align-items: center; font-size: 16rpx; flex-shrink: 0;
		.title { color: #555; }
		.power-value { font-size: 14rpx; font-weight: bold; color: #fff;
			&.negative { color: #fff; }
			&.high:not(.negative) { color: #e02020; }
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
		margin-left: 8rpx;
		padding-left: 8rpx;
		
		.car-top-mini { width: 28rpx; height: 50rpx; }
		.tpms-col {
			display: flex; flex-direction: column; justify-content: space-around; height: 80%;
			font-size: 13rpx; font-weight: bold; color: #fff;
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
			position: absolute; bottom: 0; width: 4rpx; height: 80%; background: rgba(255, 255, 255, 0.15);
			border-radius: 2rpx;
		}
		.line-l { left: 20rpx; transform: rotate(8deg); }
		.line-r { right: 20rpx; transform: rotate(-8deg); }
		
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
