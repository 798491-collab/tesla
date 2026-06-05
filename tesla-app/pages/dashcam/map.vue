<template>
  <view class="map-container" :class="themeClass">
    <NavBar title="轨迹地图" />

    <view class="map-wrap" :class="{ 'map-dark-filter': mapStyle === 'dark' && useMapFilter }">
      <map
        id="mapId"
        class="map"
        :latitude="centerLat"
        :longitude="centerLng"
        :markers="mapMarkers"
        :polyline="polyline"
        :scale="mapScale"
        :enable-3D="true"
        :show-compass="true"
        :enable-overlooking="true"
        :enable-zoom="true"
        :enable-scroll="true"
        :enable-rotate="true"
        :show-location="false"
        :subkey="tencentMapKey"
        @updated="onMapUpdated"
        @markertap="onMarkerTap"
        @regionchange="onRegionChange"
      ></map>

      <view class="map-controls">
        <view class="control-group">
          <view
            class="ctrl-btn"
            :class="{ active: mapStyle === 'standard' }"
            @click="mapStyle = 'standard'"
          >
            <Icon name="Map" :size="16" :color="mapStyle === 'standard' ? '#fff' : '#666'" />
          </view>
          <view
            class="ctrl-btn"
            :class="{ active: mapStyle === 'dark' }"
            @click="mapStyle = 'dark'"
          >
            <Icon name="Moon" :size="16" :color="mapStyle === 'dark' ? '#fff' : '#666'" />
          </view>
        </view>
        <view class="control-group">
          <view class="ctrl-btn" @click="fitBounds">
            <Icon name="Expand" :size="16" color="#666" />
          </view>
        </view>
      </view>

      <view class="date-picker">
        <view class="date-btn" @click="prevDay">
          <Icon name="ChevronBack" :size="16" color="#666" />
        </view>
        <view class="date-display" @click="showDatePicker = true">
          <Icon name="Calendar" :size="14" :color="dateIconColor" />
          <text class="date-text">{{ displayDate }}</text>
        </view>
        <view class="date-btn" @click="nextDay">
          <Icon name="ChevronForward" :size="16" color="#666" />
        </view>
      </view>
    </view>

    <view class="bottom-panel" v-if="selectedEvent">
      <view class="event-card-mini">
        <view class="event-thumb-mini" v-if="selectedEvent.thumbnail">
          <image :src="selectedEvent.thumbnail" class="thumb-img-mini" mode="aspectFill" />
        </view>
        <view class="event-info-mini">
          <view class="event-type-row">
            <view class="type-dot" :style="{ backgroundColor: getTypeColor(selectedEvent.event_type) }"></view>
            <text class="type-label-mini">{{ getTypeLabel(selectedEvent.event_type) }}</text>
          </view>
          <text class="event-time-mini">{{ formatEventTime(selectedEvent.event_time) }}</text>
        </view>
        <view class="play-btn-mini" @click="goPlayer(selectedEvent.id)">
          <Icon name="Play" :size="16" color="#fff" />
        </view>
      </view>
    </view>

    <view class="track-stats" v-if="trackStats.pointCount > 0">
      <view class="stat-chip">
        <Icon name="Navigate" :size="12" :color="statIconColor" />
        <text class="stat-chip-text">{{ formatDistance(trackStats.totalDistance) }}</text>
      </view>
      <view class="stat-chip">
        <Icon name="Speed" :size="12" :color="statIconColor" />
        <text class="stat-chip-text">{{ trackStats.avgSpeed.toFixed(0) }} km/h</text>
      </view>
      <view class="stat-chip">
        <Icon name="Videocam" :size="12" :color="statIconColor" />
        <text class="stat-chip-text">{{ eventCount }} 个事件</text>
      </view>
    </view>

    <view class="empty-map" v-if="!loading && trackStats.pointCount === 0 && !selectedEvent">
      <Icon name="MapOutline" :size="48" :color="emptyIconColor" />
      <text class="empty-map-text">当日无轨迹数据</text>
      <text class="empty-map-sub">选择有行驶记录的日期查看轨迹</text>
    </view>

    <view class="loading-overlay" v-if="loading">
      <view class="loading-spinner"></view>
      <text class="loading-text">加载轨迹...</text>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted, getCurrentInstance, watch } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import NavBar from '@/components/NavBar/NavBar.vue'
import Icon from '@/components/Icon/Icon.vue'
import { useThemeStore } from '@/store/theme'
import { initDB, getEvents, getTracks } from '@/utils/dashcam-db.js'
import { getCachedTracks, getTrackStats } from '@/utils/dashcam-gps.js'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const dateIconColor = computed(() => themeStore.colors.primary)
const statIconColor = computed(() => themeStore.colors.primary)
const emptyIconColor = computed(() => themeStore.colors.inactiveIcon)

const useMapFilter = ref(true)
const mapStyle = ref('standard')
const tencentMapKey = import.meta.env.VITE_TENCENT_MAP_KEY || ''
const darkStyleId = import.meta.env.VITE_TENCENT_MAP_STYLE_DARK || '2'
let isStyleSet = false
const mapScale = ref(12)
const centerLat = ref(39.9042)
const centerLng = ref(116.4074)
const selectedEvent = ref(null)
const loading = ref(false)
const showDatePicker = ref(false)
const currentDate = ref(new Date())
const trackPoints = ref([])
const events = ref([])
const trackStats = ref({ totalDistance: 0, avgSpeed: 0, maxSpeed: 0, duration: 0, pointCount: 0 })
const eventCount = ref(0)
const vin = ref('')

const currentLayerStyle = computed(() => {
  const isDark = mapStyle.value === 'dark' || themeStore.resolvedTheme === 'dark' || themeStore.resolvedTheme === 'visionpro'
  if (isDark) {
    const styleId = darkStyleId || '2'
    // #ifdef APP-PLUS
    return parseInt(styleId) || 2
    // #endif
    // #ifdef H5
    return 'style' + styleId
    // #endif
    // #ifndef APP-PLUS || H5
    return styleId
    // #endif
  }
  return 1
})

function applyMapDarkStyle() {
  if (isStyleSet) return
  try {
    const mapCtx = uni.createMapContext('mapId', getCurrentInstance())
    if (mapCtx?.setMapStyle) {
      mapCtx.setMapStyle({
        styleId: darkStyleId,
        success: () => {
          console.log('[Map] 地图墨渊主题设置成功')
          isStyleSet = true
          useMapFilter.value = false
        },
        fail: (err) => {
          console.warn('[Map] 地图主题设置失败，尝试整数参数:', err)
          try {
            mapCtx.setMapStyle(parseInt(darkStyleId) || 2)
            isStyleSet = true
            useMapFilter.value = false
          } catch (e) {}
        }
      })
    }
  } catch (e) {}
}

function onMapUpdated() {
  applyMapDarkStyle()
}

const displayDate = computed(() => {
  const d = currentDate.value
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const weekDays = ['日', '一', '二', '三', '四', '五', '六']
  return `${y}-${m}-${day} 周${weekDays[d.getDay()]}`
})

const polyline = computed(() => {
  if (trackPoints.value.length < 2) return []
  const points = trackPoints.value.map(p => ({
    latitude: p.latitude,
    longitude: p.longitude
  }))
  return [{
    points,
    color: '#5B8CFF',
    width: 6,
    dottedLine: false,
    arrowLine: true,
    borderColor: '#3B6FE8',
    borderWidth: 2
  }]
})

const mapMarkers = computed(() => {
  const markers = []
  events.value.forEach((evt, idx) => {
    if (!evt.latitude || !evt.longitude) return
    markers.push({
      id: 1000 + idx,
      latitude: evt.latitude,
      longitude: evt.longitude,
      title: getTypeLabel(evt.event_type),
      iconPath: '/static/dashcam-marker.png',
      width: 28,
      height: 28,
      callout: {
        content: getTypeLabel(evt.event_type) + ' ' + formatEventTime(evt.event_time).split(' ')[1],
        color: '#ffffff',
        fontSize: 12,
        borderRadius: 8,
        bgColor: getTypeColor(evt.event_type),
        padding: 6,
        display: 'BYCLICK',
        anchorX: 0,
        anchorY: 0
      }
    })
  })
  if (trackPoints.value.length > 0) {
    const first = trackPoints.value[0]
    markers.push({
      id: 1,
      latitude: first.latitude,
      longitude: first.longitude,
      title: '起点',
      iconPath: '/static/marker-start.png',
      width: 24,
      height: 24
    })
    const last = trackPoints.value[trackPoints.value.length - 1]
    markers.push({
      id: 2,
      latitude: last.latitude,
      longitude: last.longitude,
      title: '终点',
      iconPath: '/static/marker-end.png',
      width: 24,
      height: 24
    })
  }
  return markers
})

const TYPE_COLORS = {
  recent: '#60a5fa',
  saved: '#5BE7C4',
  sentry: '#f97316'
}

const TYPE_LABELS = {
  recent: '最近',
  saved: '已保存',
  sentry: '哨兵'
}

const getTypeColor = (type) => TYPE_COLORS[type] || '#60a5fa'
const getTypeLabel = (type) => TYPE_LABELS[type] || type

const formatEventTime = (ts) => {
  if (!ts) return ''
  const d = new Date(ts)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const formatDistance = (meters) => {
  if (meters >= 1000) return (meters / 1000).toFixed(1) + ' km'
  return Math.round(meters) + ' m'
}

const getDayRange = (date) => {
  const d = new Date(date)
  d.setHours(0, 0, 0, 0)
  const start = d.getTime()
  d.setDate(d.getDate() + 1)
  const end = d.getTime()
  return { start, end }
}

const prevDay = () => {
  const d = new Date(currentDate.value)
  d.setDate(d.getDate() - 1)
  currentDate.value = d
  loadDayData()
}

const nextDay = () => {
  const d = new Date(currentDate.value)
  d.setDate(d.getDate() + 1)
  if (d > new Date()) return
  currentDate.value = d
  loadDayData()
}

const loadDayData = async () => {
  loading.value = true
  selectedEvent.value = null
  const { start, end } = getDayRange(currentDate.value)

  try {
    const tracks = await getTracks({ vin: vin.value, startTime: start, endTime: end })
    trackPoints.value = tracks || []

    if (trackPoints.value.length > 0) {
      centerLat.value = trackPoints.value[0].latitude
      centerLng.value = trackPoints.value[0].longitude
      mapScale.value = 13
    }

    try {
      const stats = await getTrackStats(vin.value, start, end)
      trackStats.value = stats
    } catch (e) {
      trackStats.value = { totalDistance: 0, avgSpeed: 0, maxSpeed: 0, duration: 0, pointCount: 0 }
    }
  } catch (e) {
    trackPoints.value = []
  }

  try {
    const evts = await getEvents({ startTime: start, endTime: end, limit: 100 })
    events.value = (evts || []).filter(e => e.latitude && e.longitude)
    eventCount.value = (evts || []).length
  } catch (e) {
    events.value = []
  }

  loading.value = false
}

const onMarkerTap = (e) => {
  const markerId = e.markerId || e.detail?.markerId
  if (markerId >= 1000) {
    const idx = markerId - 1000
    if (events.value[idx]) {
      selectedEvent.value = events.value[idx]
    }
  }
}

const onRegionChange = () => {}

const fitBounds = () => {
  if (trackPoints.value.length < 2) return
  let minLat = 90, maxLat = -90, minLng = 180, maxLng = -180
  trackPoints.value.forEach(p => {
    if (p.latitude < minLat) minLat = p.latitude
    if (p.latitude > maxLat) maxLat = p.latitude
    if (p.longitude < minLng) minLng = p.longitude
    if (p.longitude > maxLng) maxLng = p.longitude
  })
  centerLat.value = (minLat + maxLat) / 2
  centerLng.value = (minLng + maxLng) / 2
  const latDiff = maxLat - minLat
  const lngDiff = maxLng - minLng
  const maxDiff = Math.max(latDiff, lngDiff)
  if (maxDiff > 0.5) mapScale.value = 9
  else if (maxDiff > 0.2) mapScale.value = 10
  else if (maxDiff > 0.05) mapScale.value = 12
  else mapScale.value = 14
}

const goPlayer = (id) => {
  uni.navigateTo({ url: '/pages/dashcam/player?id=' + id })
}

onLoad((options) => {
  if (options?.lat && options?.lng) {
    centerLat.value = parseFloat(options.lat)
    centerLng.value = parseFloat(options.lng)
    mapScale.value = 15
  }
  if (options?.vin) {
    vin.value = options.vin
  }
  if (options?.date) {
    currentDate.value = new Date(options.date)
  }
})

onMounted(async () => {
  try {
    await initDB()
  } catch (e) {}
  loadDayData()
  setTimeout(() => applyMapDarkStyle(), 500)
})

watch(mapStyle, (val) => {
  if (val === 'dark') {
    useMapFilter.value = true
    isStyleSet = false
    applyMapDarkStyle()
  }
})

watch(() => themeStore.resolvedTheme, () => {
  if (mapStyle.value === 'dark') {
    useMapFilter.value = true
    isStyleSet = false
    applyMapDarkStyle()
  }
})
</script>

<style lang="scss" scoped>
.map-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--bg-page);
  display: flex;
  flex-direction: column;
}

.map-wrap {
  flex: 1;
  position: relative;
}

.map {
  width: 100%;
  height: 100%;
}

.map-dark-filter .map {
  filter: invert(90%) hue-rotate(180deg) brightness(0.95) contrast(0.9);
  transition: filter 0.3s ease;
}

.map-controls {
  position: absolute;
  top: calc(var(--status-bar-height, 44px) + 100rpx);
  right: 24rpx;
  display: flex;
  flex-direction: column;
  gap: 12rpx;
  z-index: 10;
}

.control-group {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.ctrl-btn {
  width: 64rpx;
  height: 64rpx;
  border-radius: 16rpx;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(12px);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.12);

  &.active {
    background: #2563EB;
    box-shadow: 0 4rpx 16rpx rgba(37, 99, 235, 0.35);
  }
}

.dark-theme .ctrl-btn {
  background: rgba(30, 30, 40, 0.92);

  &.active {
    background: #5B8CFF;
  }
}

.date-picker {
  position: absolute;
  top: calc(var(--status-bar-height, 44px) + 100rpx);
  left: 24rpx;
  display: flex;
  align-items: center;
  gap: 8rpx;
  z-index: 10;
}

.date-btn {
  width: 56rpx;
  height: 56rpx;
  border-radius: 14rpx;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(12px);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.12);
}

.dark-theme .date-btn {
  background: rgba(30, 30, 40, 0.92);
}

.date-display {
  display: flex;
  align-items: center;
  gap: 8rpx;
  padding: 12rpx 20rpx;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(12px);
  border-radius: 14rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.12);
}

.dark-theme .date-display {
  background: rgba(30, 30, 40, 0.92);
}

.date-text {
  font-size: 24rpx;
  color: var(--text-primary);
  font-weight: 500;
}

.bottom-panel {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 20rpx 24rpx;
  padding-bottom: calc(20rpx + env(safe-area-inset-bottom));
  z-index: 20;
}

.event-card-mini {
  display: flex;
  align-items: center;
  gap: 16rpx;
  background: var(--bg-card);
  border-radius: 20rpx;
  padding: 20rpx 24rpx;
  box-shadow: 0 8rpx 32rpx rgba(0, 0, 0, 0.2);
}

.event-thumb-mini {
  width: 80rpx;
  height: 60rpx;
  border-radius: 10rpx;
  overflow: hidden;
  flex-shrink: 0;
}

.thumb-img-mini {
  width: 100%;
  height: 100%;
}

.event-info-mini {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.event-type-row {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.type-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;
}

.type-label-mini {
  font-size: 22rpx;
  font-weight: 600;
  color: var(--text-primary);
}

.event-time-mini {
  font-size: 20rpx;
  color: var(--text-tertiary);
}

.play-btn-mini {
  width: 56rpx;
  height: 56rpx;
  border-radius: 28rpx;
  background: var(--gradient);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.track-stats {
  position: absolute;
  bottom: calc(140rpx + env(safe-area-inset-bottom));
  left: 24rpx;
  right: 24rpx;
  display: flex;
  gap: 12rpx;
  z-index: 15;
}

.stat-chip {
  display: flex;
  align-items: center;
  gap: 6rpx;
  padding: 8rpx 16rpx;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(12px);
  border-radius: 20rpx;
  box-shadow: 0 4rpx 12rpx rgba(0, 0, 0, 0.1);
}

.dark-theme .stat-chip {
  background: rgba(30, 30, 40, 0.92);
}

.stat-chip-text {
  font-size: 20rpx;
  color: var(--text-primary);
  font-weight: 500;
}

.empty-map {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16rpx;
  z-index: 5;
}

.empty-map-text {
  font-size: 30rpx;
  color: var(--text-tertiary);
  font-weight: 500;
}

.empty-map-sub {
  font-size: 24rpx;
  color: var(--text-placeholder);
}

.loading-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16rpx;
  z-index: 50;
}

.loading-spinner {
  width: 48rpx;
  height: 48rpx;
  border: 4rpx solid var(--bg-spinner-track);
  border-top-color: var(--color-spinner);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.loading-text {
  font-size: 26rpx;
  color: var(--text-tertiary);
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
