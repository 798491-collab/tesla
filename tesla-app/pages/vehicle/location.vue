<template>
  <view class="location-container" :class="themeClass">
    <NavBar title="车辆位置" />
    <view class="map-wrap" :class="{ 'map-dark-filter': mapStyle === 'dark' }">
      <map
        class="map"
        :latitude="centerLat"
        :longitude="centerLng"
        :markers="markers"
        :scale="15"
        :enable-3D="true"
        :show-compass="true"
        :enable-overlooking="true"
        :enable-zoom="true"
        :enable-scroll="true"
        :enable-rotate="true"
        :enable-satellite="mapStyle === 'satellite'"
        :enable-traffic="mapStyle === 'traffic'"
        :show-location="true"
        :layer-style="currentLayerStyle"
      ></map>
      <view class="map-style-switcher">
        <view
          class="style-btn"
          :class="{ active: mapStyle === 'standard' }"
          @click="mapStyle = 'standard'"
        >
          <Icon name="Map" :size="16" :color="mapStyle === 'standard' ? '#fff' : '#666'" />
          <text>标准</text>
        </view>
        <view
          class="style-btn"
          :class="{ active: mapStyle === 'satellite' }"
          @click="mapStyle = 'satellite'"
        >
          <Icon name="Globe" :size="16" :color="mapStyle === 'satellite' ? '#fff' : '#666'" />
          <text>卫星</text>
        </view>
        <view
          class="style-btn"
          :class="{ active: mapStyle === 'dark' }"
          @click="mapStyle = 'dark'"
        >
          <Icon name="Moon" :size="16" :color="mapStyle === 'dark' ? '#fff' : '#666'" />
          <text>墨渊</text>
        </view>
        <view
          class="style-btn"
          :class="{ active: mapStyle === 'traffic' }"
          @click="mapStyle = 'traffic'"
        >
          <Icon name="Car" :size="16" :color="mapStyle === 'traffic' ? '#fff' : '#666'" />
          <text>路况</text>
        </view>
      </view>
    </view>

    <view class="info-card">
      <view class="info-header">
        <view class="status-badge" :class="getStatusClass(state)">
          <Icon :name="getStatusIcon(state)" :size="14" color="#fff" />
          <text class="status-badge-text">{{ getStatusText(state) }}</text>
        </view>
        <view class="update-time-wrap">
          <Icon name="Time" :size="14" color="#999" />
          <text class="update-time">{{ formatTime(state.updated_at) }}</text>
        </view>
      </view>

      <view class="info-rows">
        <view class="info-row" v-if="state.address">
          <view class="info-label-wrap">
            <view class="info-icon-box">
              <Icon name="Location" :size="16" themeColor="primary" />
            </view>
            <text class="info-label">当前位置</text>
          </view>
          <text class="info-value">{{ state.address }}</text>
        </view>
        <view class="info-row" v-if="hasLocation">
          <view class="info-label-wrap">
            <view class="info-icon-box">
              <Icon name="Navigate" :size="16" themeColor="info" />
            </view>
            <text class="info-label">经纬度</text>
          </view>
          <text class="info-value small">{{ state.latitude?.toFixed(6) }}, {{ state.longitude?.toFixed(6) }}</text>
        </view>
        <view class="info-row" v-if="state.speed !== undefined && state.speed !== null">
          <view class="info-label-wrap">
            <view class="info-icon-box">
              <Icon name="Car" :size="16" :color="aiColor" />
            </view>
            <text class="info-label">速度</text>
          </view>
          <text class="info-value">{{ state.speed }} km/h</text>
        </view>
        <view class="info-row">
          <view class="info-label-wrap">
            <view class="info-icon-box">
              <Icon name="BatteryFull" :size="16" themeColor="success" />
            </view>
            <text class="info-label">电量</text>
          </view>
          <view class="info-value-wrap">
            <text class="info-value">{{ state.soc || 0 }}%</text>
            <view class="battery-bar">
              <view class="battery-fill" :style="{ width: (state.soc || 0) + '%' }"></view>
            </view>
          </view>
        </view>
      </view>
    </view>

    <view class="warn-card" v-if="!locationAuthorized">
      <view class="warn-header">
        <view class="warn-icon-wrap">
          <Icon name="Warning" :size="20" themeColor="warning" />
        </view>
        <text class="warn-title">位置权限未授权</text>
      </view>
      <text class="warn-text">需要授权"车辆位置"权限才能获取位置数据。请按以下步骤操作：</text>
      <view class="warn-steps">
        <view class="warn-step">
          <view class="step-dot">1</view>
          <text class="step-text">点击下方按钮撤销旧授权</text>
        </view>
        <view class="warn-step">
          <view class="step-dot">2</view>
          <text class="step-text">重新绑定车辆，授权时确保勾选"车辆位置"</text>
        </view>
      </view>
      <button class="btn-action" @click="goReauth">重新授权</button>
    </view>

    <view class="offline-card" v-if="locationAuthorized && !hasLocation && !state.online">
      <Icon name="InformationCircle" :size="18" themeColor="info" />
      <text class="offline-text">车辆当前离线，位置数据暂时不可用。车辆上线后将自动获取。</text>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, onActivated } from 'vue'
import { getVehicleState } from '@/api/vehicle.js'
import { useVehicleStore } from '@/store/vehicle'
import { useVehicleData, initVehicleData, destroyVehicleData } from '@/utils/vehicle-data'
import { get } from '@/utils/request.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'
import { getDisplayStateLabel, getDisplayStateColor, getOnlineStateIcon, getOnlineStateMarkerColor, isVehicleOnline, getRefreshInterval } from '@/utils/vehicle-state'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)
const successColor = computed(() => themeStore.colors.success)
const warningColor = computed(() => themeStore.colors.warning)
const infoBlueColor = computed(() => themeStore.colors.info)
const aiColor = computed(() => themeStore.colors.ai)

const vehicleDataStore = useVehicleData()
const vehicleWSData = computed(() => vehicleDataStore.data)
const vehicleWSStateOutput = computed(() => vehicleDataStore.stateOutput)

const vin = ref('')
const state = ref({})
const stateOutput = ref(null)
const locationAuthorized = ref(false)
const mapStyle = ref('standard')
const darkStyleId = import.meta.env.VITE_TENCENT_MAP_STYLE_DARK || ''
let refreshTimer = null

watch(() => themeStore.resolvedTheme, (theme) => {
  if ((theme === 'dark' || theme === 'visionpro') && mapStyle.value === 'standard') {
    mapStyle.value = 'dark'
  }
}, { immediate: true })

watch(() => vehicleWSData.value, (wsData) => {
  if (!wsData || !Object.keys(wsData).length) return
  const latitude = wsData.latitude
  const longitude = wsData.longitude
  const currentState = state.value
  if (latitude && longitude && latitude !== 0 && longitude !== 0) {
    locationAuthorized.value = true
    state.value = {
      ...currentState,
      latitude,
      longitude,
      heading: wsData.heading ?? currentState.heading,
      speed: wsData.speed ?? currentState.speed,
      gear: wsData.gear ?? currentState.gear,
      driving: wsData.driving ?? currentState.driving,
      charging: wsData.charging ?? currentState.charging,
      soc: wsData.soc ?? currentState.soc,
      range_km: wsData.range_km ?? currentState.range_km,
      online: isVehicleOnline(vehicleWSStateOutput.value),
      state: vehicleWSStateOutput.value?.state?.online_state || currentState.state,
      updated_at: new Date().toISOString()
    }
  }
  if (vehicleWSStateOutput.value) {
    stateOutput.value = vehicleWSStateOutput.value
  }
}, { deep: true })

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

const hasLocation = computed(() => state.value.latitude && state.value.latitude !== 0)

const centerLat = computed(() => state.value.latitude || 39.9042)
const centerLng = computed(() => state.value.longitude || 116.4074)

const markers = computed(() => {
  if (!hasLocation.value) return []

  const vehicleState = state.value.state
  const bgColor = getOnlineStateMarkerColor(stateOutput.value)
  const statusText = getStatusText(state.value)

  return [{
    id: 1,
    latitude: state.value.latitude,
    longitude: state.value.longitude,
    title: '车辆位置',
    iconPath: '/static/car-marker.png',
    width: 30,
    height: 30,
    rotate: state.value.heading || 0,
    callout: {
      content: '🚗 ' + statusText,
      color: '#ffffff',
      fontSize: 14,
      fontWeight: 'bold',
      borderRadius: 10,
      bgColor: bgColor,
      padding: 10,
      display: 'ALWAYS',
      anchorX: 0,
      anchorY: 0
    }
  }]
})

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  vin.value = currentPage.$page?.options?.vin || currentPage.options?.vin || ''
  if (!vin.value) {
    const vehicleStore = useVehicleStore()
    vin.value = vehicleStore.currentVehicle?.vin || ''
  }
  if (vin.value) {
    initVehicleData(vin.value)
    loadState()
    loadAuthStatus()
    startAutoRefresh()
  }
})

onActivated(() => {
  if (vin.value) {
    initVehicleData(vin.value)
    loadState()
    loadAuthStatus()
    startAutoRefresh()
  }
})

onUnmounted(() => {
  destroyVehicleData()
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})

const loadState = async () => {
  try {
    const res = await getVehicleState(vin.value)
    const data = res.data || {}

    const latitude = data.latitude
    const longitude = data.longitude

    const hasValidLocation = latitude && longitude && latitude !== 0 && longitude !== 0
    if (hasValidLocation) {
      locationAuthorized.value = true
    }

    state.value = {
      latitude: latitude,
      longitude: longitude,
      heading: data.heading,
      speed: data.speed,
      gear: data.gear,
      driving: data.driving,
      charging: data.charging,
      soc: data.soc,
      range_km: data.range_km,
      online: isVehicleOnline(data.state_output),
      state: data.state_output?.state?.online_state || data.state,
      updated_at: new Date().toISOString()
    }
    if (data.state_output) {
      stateOutput.value = data.state_output
    }
  } catch (err) {
    console.error('获取车辆数据失败:', err)
  }
}

const loadAuthStatus = () => {
  get(`/api/tesla/vehicle/${vin.value}/detail`).then((res) => {
    locationAuthorized.value = res.data?.location_authorized || false
  }).catch(() => {})
}

const startAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
  const getInterval = () => {
    return getRefreshInterval(stateOutput.value)
  }
  const tick = () => {
    if (vin.value) {
      loadState()
    }
    if (refreshTimer) clearInterval(refreshTimer)
    refreshTimer = setInterval(tick, getInterval())
  }
  tick()
}

const goReauth = () => {
  uni.navigateTo({ url: '/pages/bind/bind' })
}

const formatTime = (t) => {
  if (!t) return '--'
  const d = new Date(t)
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

const getStatusText = (s) => {
  if (!s) return '离线'
  return getDisplayStateLabel(stateOutput.value, vehicleWSData.value)
}

const getStatusClass = (s) => {
  if (!s) return 'offline'
  if (s.state) return s.state
  return 'offline'
}

const getStatusIcon = (s) => {
  if (!s) return 'Ellipse'
  return getOnlineStateIcon(stateOutput.value)
}
</script>

<style lang="scss" scoped>
.location-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-page);
  padding-top: calc(var(--status-bar-height, 44px) + 88rpx);
  box-sizing: border-box;
}

.map-wrap {
  flex: 1;
  width: 100%;
  position: relative;
}

.map {
  width: 100%;
  height: 100%;
}

.map-style-switcher {
  position: absolute;
  top: 24rpx;
  right: 24rpx;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
  z-index: 10;

  .style-btn {
    display: flex;
    align-items: center;
    gap: 6rpx;
    padding: 12rpx 20rpx;
    background: rgba(255, 255, 255, 0.92);
    backdrop-filter: blur(12px);
    border-radius: 16rpx;
    box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.12);
    font-size: 22rpx;
    color: #666;
    font-weight: 500;
    transition: all 0.2s;

    &.active {
      background: #2563EB;
      color: #fff;
      box-shadow: 0 4rpx 16rpx rgba(37, 99, 235, 0.35);
    }
  }
}

.dark-theme .map-style-switcher .style-btn {
  background: rgba(30, 30, 40, 0.92);
  color: #aaa;

  &.active {
    background: #5B8CFF;
    color: #fff;
  }
}

.visionpro-theme .map-style-switcher .style-btn {
  background: rgba(255, 255, 255, 0.72);
  color: #334155;
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.65);

  &.active {
    background: #0F172A;
    color: #fff;
    box-shadow: 0 4rpx 16rpx rgba(15, 23, 42, 0.2);
  }
}

.map-dark-filter .map {
  filter: invert(90%) hue-rotate(180deg) brightness(0.95) contrast(0.9);
  transition: filter 0.3s ease;
}

.info-card {
  background: var(--bg-card);
  padding: 28rpx 32rpx;
  padding-bottom: calc(28rpx + env(safe-area-inset-bottom));
  border-radius: 32rpx 32rpx 0 0;
  box-shadow: var(--shadow-card);

  .info-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24rpx;
    padding-bottom: 20rpx;
    border-bottom: 1rpx solid var(--border-divider);

    .status-badge {
      display: flex;
      align-items: center;
      gap: 8rpx;
      padding: 8rpx 20rpx;
      border-radius: 24rpx;
      font-size: 22rpx;
      font-weight: 600;

      &.online {
        background: rgba(82, 196, 26, 0.12);
        color: #52c41a;
      }

      &.offline {
        background: rgba(255, 77, 79, 0.12);
        color: #ff4d4f;
      }

      &.asleep, &.suspended {
        background: rgba(250, 140, 22, 0.12);
        color: #fa8c16;
      }

      &.driving {
        background: rgba(24, 144, 255, 0.12);
        color: var(--color-info);
      }

      &.charging {
        background: rgba(114, 46, 209, 0.12);
        color: #722ed1;
      }

      &.updating {
        background: rgba(167, 139, 250, 0.12);
        color: #a78bfa;
      }

      &.climate_on {
        background: rgba(96, 165, 250, 0.12);
        color: var(--color-info);
      }

      &.sentry_on {
        background: rgba(251, 191, 36, 0.12);
        color: #fbbf24;
      }

      &.waking {
        background: rgba(96, 165, 250, 0.12);
        color: var(--color-info);
      }

      .status-badge-text {
        font-size: 22rpx;
      }
    }

    .update-time-wrap {
      display: flex;
      align-items: center;
      gap: 6rpx;

      .update-time {
        font-size: 22rpx;
        color: var(--text-tertiary);
      }
    }
  }

  .info-rows {
    .info-row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 16rpx 0;

      .info-label-wrap {
        display: flex;
        align-items: center;
        gap: 12rpx;

        .info-icon-box {
          width: 48rpx;
          height: 48rpx;
          border-radius: 12rpx;
          background: rgba(0, 0, 0, 0.04);
          display: flex;
          align-items: center;
          justify-content: center;
        }

        .info-label {
          font-size: 28rpx;
          color: var(--text-secondary);
        }
      }

      .info-value {
        font-size: 28rpx;
        color: var(--text-primary);
        font-weight: 600;

        &.small {
          font-size: 24rpx;
          color: var(--text-secondary);
          font-weight: 500;
        }
      }

      .info-value-wrap {
        display: flex;
        align-items: center;
        gap: 16rpx;

        .info-value {
          font-size: 28rpx;
          color: var(--text-primary);
          font-weight: 600;
        }

        .battery-bar {
          width: 120rpx;
          height: 12rpx;
          border-radius: 6rpx;
          background: var(--bg-card-secondary);
          overflow: hidden;

          .battery-fill {
            height: 100%;
            border-radius: 6rpx;
            background: linear-gradient(90deg, #52c41a, #73d13d);
          }
        }
      }
    }
  }
}

.warn-card {
  background: var(--bg-card);
  padding: 28rpx 32rpx;
  padding-bottom: calc(28rpx + env(safe-area-inset-bottom));
  border-top: 2rpx solid var(--border-divider);

  .warn-header {
    display: flex;
    align-items: center;
    gap: 12rpx;
    margin-bottom: 16rpx;

    .warn-icon-wrap {
      width: 48rpx;
      height: 48rpx;
      border-radius: 50%;
      background: rgba(250, 140, 22, 0.12);
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .warn-title {
      font-size: 30rpx;
      font-weight: 600;
      color: var(--text-primary);
    }
  }

  .warn-text {
    font-size: 26rpx;
    color: var(--text-secondary);
    line-height: 1.6;
    display: block;
    margin-bottom: 20rpx;
  }

  .warn-steps {
    margin-bottom: 24rpx;

    .warn-step {
      display: flex;
      align-items: center;
      gap: 16rpx;
      padding: 10rpx 0;

      .step-dot {
        width: 36rpx;
        height: 36rpx;
        border-radius: 50%;
        background: rgba(250, 140, 22, 0.15);
        color: #fa8c16;
        font-size: 22rpx;
        font-weight: 600;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
      }

      .step-text {
        font-size: 26rpx;
        color: var(--text-secondary);
        line-height: 1.5;
      }
    }
  }
}

.btn-action {
  background: var(--gradient);
  color: #ffffff;
  border-radius: 48rpx;
  height: 80rpx;
  line-height: 80rpx;
  font-size: 28rpx;
  border: none;
  font-weight: 500;
  box-shadow: 0 6rpx 20rpx rgba(37, 99, 235, 0.3);
}

.offline-card {
  background: var(--bg-card);
  padding: 24rpx 32rpx;
  padding-bottom: calc(24rpx + env(safe-area-inset-bottom));
  border-top: 2rpx solid var(--border-divider);
  display: flex;
  align-items: flex-start;
  gap: 12rpx;

  .offline-text {
    font-size: 26rpx;
    color: var(--text-secondary);
    line-height: 1.6;
    flex: 1;
  }
}
</style>
