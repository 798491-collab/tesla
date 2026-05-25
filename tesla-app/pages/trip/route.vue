<template>
  <view class="route-container" :class="themeClass">
    <NavBar title="行驶轨迹" />
    <scroll-view scroll-y class="main-scroll">
    <map
      id="routeMap"
      class="map"
      :latitude="centerLat"
      :longitude="centerLng"
      :markers="markers"
      :polyline="polyline"
      :scale="scale"
      show-location
    ></map>

    <view class="info-card">
      <view class="trip-info" v-if="trip">
        <view class="trip-header">
          <view class="trip-date-wrap">
            <Icon name="Calendar" :size="16" themeColor="hint" />
            <text class="trip-date">{{ formatDate(trip.start_time) }}</text>
          </view>
          <view class="trip-distance-wrap">
            <Icon name="Speedometer" :size="18" themeColor="primary" />
            <text class="trip-distance">{{ trip.distance?.toFixed(1) }} km</text>
          </view>
        </view>

        <view class="trip-route">
          <view class="route-point start-point">
            <view class="point-dot-wrap start-dot">
              <Icon name="Navigate" :size="14" color="#fff" />
            </view>
            <view class="point-info">
              <text class="point-label">出发</text>
              <text class="point-text">{{ trip.start_address || formatCoord(trip.start_lat, trip.start_lng) }}</text>
              <text class="point-time" v-if="trip.start_time">{{ formatTime(trip.start_time) }}</text>
            </view>
          </view>
          <view class="route-line">
            <view class="line-dash" v-for="i in 4" :key="i"></view>
          </view>
          <view class="route-point end-point">
            <view class="point-dot-wrap end-dot">
              <Icon name="Location" :size="14" color="#fff" />
            </view>
            <view class="point-info">
              <text class="point-label">到达</text>
              <text class="point-text">{{ trip.end_address || formatCoord(trip.end_lat, trip.end_lng) }}</text>
              <text class="point-time" v-if="trip.end_time">{{ formatTime(trip.end_time) }}</text>
            </view>
          </view>
        </view>

        <view class="trip-stats">
          <view class="stat-item">
            <view class="stat-icon-wrap">
              <Icon name="Car" :size="14" themeColor="primary" />
            </view>
            <text class="stat-value">{{ trip.avg_speed?.toFixed(1) || '--' }}</text>
            <text class="stat-label">km/h 均速</text>
          </view>
          <view class="stat-item">
            <view class="stat-icon-wrap">
              <Icon name="BatteryCharging" :size="14" themeColor="primary" />
            </view>
            <text class="stat-value">{{ trip.energy_used?.toFixed(1) || '--' }}</text>
            <text class="stat-label">kWh 能耗</text>
          </view>
          <view class="stat-item">
            <view class="stat-icon-wrap">
              <Icon name="BatteryFull" :size="14" themeColor="primary" />
            </view>
            <text class="stat-value">{{ trip.start_battery_level || '--' }}→{{ trip.end_battery_level || '--' }}</text>
            <text class="stat-label">电量变化</text>
          </view>
          <view class="stat-item">
            <view class="stat-icon-wrap">
              <Icon name="Location" :size="14" themeColor="primary" />
            </view>
            <text class="stat-value">{{ points.length }}</text>
            <text class="stat-label">轨迹点</text>
          </view>
        </view>
      </view>
    </view>

    <view class="ai-section" v-if="aiResult || aiLoading">
      <view class="ai-card" v-if="aiResult" @click="aiExpanded = !aiExpanded">
        <view class="ai-card-header">
          <view class="ai-card-title">
            <Icon name="Sparkles" :size="18" themeColor="primary" />
            <text class="ai-title-text">AI 行程分析</text>
          </view>
          <Icon :name="aiExpanded ? 'ChevronUp' : 'ChevronDown'" :size="16" themeColor="hint" />
        </view>
        <view class="ai-summary-row" v-if="!aiExpanded">
          <text class="ai-summary-text">{{ aiResult.summary || '点击查看详细分析' }}</text>
        </view>
        <view class="ai-card-body" v-if="aiExpanded">
          <text class="ai-text" v-for="(line, i) in aiLines" :key="i">{{ line }}</text>
        </view>
        <text class="ai-time" v-if="aiExpanded">{{ formatAITime(aiResult.created_at) }}</text>
      </view>

      <view class="ai-card ai-loading" v-if="aiLoading">
        <view class="ai-card-header">
          <view class="ai-card-title">
            <Icon name="Sparkles" :size="18" themeColor="primary" />
            <text class="ai-title-text">AI 行程分析</text>
          </view>
        </view>
        <view class="ai-card-body">
          <view class="ai-spinner"></view>
          <text class="ai-loading-text">AI 正在分析中...</text>
        </view>
      </view>
    </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getTripPoints } from '@/api/trip.js'
import { getTripAnalysis, triggerTripAnalysis } from '@/api/ai.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const mapStartColor = computed(() => themeStore.colors.mapStart)
const mapEndColor = computed(() => themeStore.colors.mapEnd)
const mapLineColor = computed(() => themeStore.colors.mapLine)
const mapLineBorderColor = computed(() => themeStore.colors.mapLineBorder)
const primaryColor = computed(() => themeStore.colors.primary)
const tertiaryColor = computed(() => themeStore.colors.hint)

const tripId = ref('')
const trip = ref(null)
const points = ref([])
const aiResult = ref(null)
const aiLoading = ref(false)
const aiExpanded = ref(false)

const centerLat = ref(39.9042)
const centerLng = ref(116.4074)
const scale = ref(12)

const aiLines = computed(() => {
  if (!aiResult.value?.result) return []
  return aiResult.value.result.split('\n').filter(l => l.trim()).map(l => l.replace(/^#{1,3}\s*/, '').replace(/\*\*/g, '').replace(/^[-*]\s*/, '• ').trim())
})

const markers = computed(() => {
  const mks = []
  if (trip.value?.start_lat && trip.value?.start_lng) {
    mks.push({
      id: 1,
      latitude: trip.value.start_lat,
      longitude: trip.value.start_lng,
      title: '起点',
      iconPath: '/static/car-marker.png',
      width: 30,
      height: 30,
      anchor: { x: 0.5, y: 0.5 },
      callout: {
        content: '起点',
        color: '#ffffff',
        fontSize: 13,
        fontWeight: 'bold',
        borderRadius: 6,
        bgColor: mapStartColor.value,
        padding: 8,
        display: 'ALWAYS',
        anchorX: 0,
        anchorY: 0
      }
    })
  }
  if (trip.value?.end_lat && trip.value?.end_lng) {
    mks.push({
      id: 2,
      latitude: trip.value.end_lat,
      longitude: trip.value.end_lng,
      title: '终点',
      iconPath: '/static/car-marker.png',
      width: 30,
      height: 30,
      anchor: { x: 0.5, y: 0.5 },
      callout: {
        content: '终点',
        color: '#ffffff',
        fontSize: 13,
        fontWeight: 'bold',
        borderRadius: 6,
        bgColor: mapEndColor.value,
        padding: 8,
        display: 'ALWAYS',
        anchorX: 0,
        anchorY: 0
      }
    })
  }
  return mks
})

const polyline = computed(() => {
  if (points.value.length < 2) return []
  return [{
    points: points.value.map(p => ({
      latitude: p.latitude,
      longitude: p.longitude
    })),
    color: mapLineColor.value,
    width: 6,
    arrowLine: true,
    borderColor: mapLineBorderColor.value,
    borderWidth: 1
  }]
})

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  tripId.value = currentPage.$page?.options?.id || currentPage.options?.id || ''
  if (tripId.value) {
    loadData()
    loadAIAnalysis()
  }
})

const calcScaleFromSpan = (latSpan, lngSpan) => {
  const paddedSpan = Math.max(latSpan, lngSpan) * 1.5
  const s = Math.log2(360 / paddedSpan)
  return Math.max(3, Math.min(18, Math.round(s)))
}

const loadData = () => {
  const vin = uni.getStorageSync('currentTripVIN') || ''

  const tripData = uni.getStorageSync('currentTrip')
  if (tripData) {
    try {
      trip.value = typeof tripData === 'string' ? JSON.parse(tripData) : tripData
      if (trip.value.start_lat && trip.value.start_lng) {
        centerLat.value = trip.value.start_lat
        centerLng.value = trip.value.start_lng
      }
    } catch (e) {}
  }

  if (vin && tripId.value) {
    getTripPoints(vin, tripId.value).then((res) => {
      const pts = res.data || []
      points.value = pts
      if (pts.length >= 2) {
        let minLat = Infinity, maxLat = -Infinity
        let minLng = Infinity, maxLng = -Infinity
        for (const p of pts) {
          if (p.latitude < minLat) minLat = p.latitude
          if (p.latitude > maxLat) maxLat = p.latitude
          if (p.longitude < minLng) minLng = p.longitude
          if (p.longitude > maxLng) maxLng = p.longitude
        }
        centerLat.value = (minLat + maxLat) / 2
        centerLng.value = (minLng + maxLng) / 2
        const latSpan = maxLat - minLat
        const lngSpan = maxLng - minLng
        scale.value = calcScaleFromSpan(latSpan, lngSpan)
      }
    })
  }
}

const loadAIAnalysis = async () => {
  const vin = uni.getStorageSync('currentTripVIN') || ''
  if (!vin || !tripId.value) return
  const refId = `trip:${tripId.value}`
  try {
    const res = await getTripAnalysis(vin, refId)
    if (res?.data) {
      aiResult.value = res.data
    } else {
      aiLoading.value = true
      await triggerTripAnalysis(vin, refId)
      setTimeout(async () => {
        const res2 = await getTripAnalysis(vin, refId)
        if (res2?.data) aiResult.value = res2.data
        aiLoading.value = false
      }, 15000)
    }
  } catch (e) {
    aiLoading.value = false
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

const formatTime = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

const formatCoord = (lat, lng) => {
  if (!lat && !lng) return '--'
  return `${lat?.toFixed(4)}, ${lng?.toFixed(4)}`
}

const formatAITime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}
</script>

<style lang="scss" scoped>
.route-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  padding-top: calc(var(--status-bar-height, 44px) + 88rpx);
  box-sizing: border-box;
  background: var(--bg-page);
  display: flex;
  flex-direction: column;
}

.main-scroll {
  flex: 1;
  overflow: hidden;
}

.map {
  width: 100%;
  height: calc(55vh - 88rpx);
}

.info-card {
  background: var(--bg-card);
  padding: 28rpx 32rpx;
  box-shadow: 0 -4rpx 20rpx rgba(0, 0, 0, 0.06);
  border-radius: 28rpx 28rpx 0 0;
}

.trip-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24rpx;

  .trip-date-wrap {
    display: flex;
    align-items: center;
    gap: 8rpx;

    .trip-date {
      font-size: 26rpx;
      color: var(--text-secondary);
      font-weight: 500;
    }
  }

  .trip-distance-wrap {
    display: flex;
    align-items: center;
    gap: 8rpx;

    .trip-distance {
      font-size: 34rpx;
      font-weight: 800;
      color: var(--color-primary);
    }
  }
}

.trip-route {
  padding: 24rpx;
  background: var(--bg-card-secondary);
  border-radius: 20rpx;
  margin-bottom: 24rpx;
}

.route-point {
  display: flex;
  align-items: flex-start;

  .point-dot-wrap {
    width: 44rpx;
    height: 44rpx;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 20rpx;
    flex-shrink: 0;
  }

  .start-dot {
    background: linear-gradient(135deg, #52c41a, #73d13d);
    box-shadow: 0 4rpx 12rpx rgba(82, 196, 26, 0.3);
  }

  .end-dot {
    background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
    box-shadow: 0 4rpx 12rpx rgba(37, 99, 235, 0.3);
  }

  .point-info {
    flex: 1;
    min-width: 0;

    .point-label {
      font-size: 20rpx;
      color: var(--text-placeholder);
      display: block;
    }

    .point-text {
      font-size: 28rpx;
      color: var(--text-primary);
      font-weight: 600;
      display: block;
      margin-top: 4rpx;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .point-time {
      font-size: 22rpx;
      color: var(--text-tertiary);
      display: block;
      margin-top: 4rpx;
    }
  }
}

.route-line {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8rpx 0;
  margin-left: 20rpx;
  width: 44rpx;

  .line-dash {
    width: 3rpx;
    height: 8rpx;
    background: var(--bg-card-secondary);
    border-radius: 2rpx;
    margin: 4rpx 0;
  }
}

.trip-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12rpx;
  padding-top: 24rpx;
  border-top: 1rpx solid var(--border-divider);
}

.stat-item {
  text-align: center;

  .stat-icon-wrap {
    width: 40rpx;
    height: 40rpx;
    border-radius: 50%;
    background: var(--bg-icon-wrap);
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 8rpx;
  }

  .stat-value {
    font-size: 26rpx;
    font-weight: 700;
    color: var(--text-primary);
    display: block;
  }

  .stat-label {
    font-size: 18rpx;
    color: var(--text-placeholder);
    margin-top: 4rpx;
    display: block;
  }
}

.ai-section {
  padding: 0 24rpx 40rpx;
}

.ai-card {
  background: var(--bg-ai-card);
  border: 1rpx solid var(--border-ai);
  border-radius: 24rpx;
  padding: 24rpx 28rpx;
  margin-bottom: 24rpx;

  .ai-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .ai-card-title {
      display: flex;
      align-items: center;
      gap: 8rpx;

      .ai-title-text {
        font-size: 28rpx;
        font-weight: 700;
        color: var(--text-primary);
      }
    }

    .ai-summary-row {
      margin-top: 12rpx;
      padding: 14rpx 18rpx;
      background: var(--bg-card-secondary);
      border-radius: 12rpx;

      .ai-summary-text {
        font-size: 26rpx;
        color: var(--text-secondary);
        line-height: 1.6;
        display: block;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }
  }

  .ai-card-body {
    margin-top: 16rpx;

    .ai-text {
      font-size: 26rpx;
      color: var(--text-ai);
      line-height: 1.8;
      display: block;
    }
  }

  .ai-time {
    font-size: 22rpx;
    color: var(--text-hint);
    display: block;
    margin-top: 12rpx;
  }

  &.ai-loading {
    .ai-card-body {
      display: flex;
      align-items: center;
      gap: 16rpx;
      padding: 12rpx 0;

      .ai-spinner {
        width: 32rpx;
        height: 32rpx;
        border: 3rpx solid var(--bg-spinner-track);
        border-top-color: var(--color-spinner);
        border-radius: 50%;
        animation: ai-spin 0.8s linear infinite;
      }

      .ai-loading-text {
        font-size: 26rpx;
        color: var(--text-hint);
      }
    }
  }
}

@keyframes ai-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
