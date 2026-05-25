<template>
  <view class="instrument-page" :class="themeClass">
    <view class="cluster-screen" v-if="currentVehicle">
      <!-- 实时地图背景 -->
      <view class="map-bg-container">
        <map
          id="dashboardMap"
          class="map-bg"
          :latitude="vehicleLat"
          :longitude="vehicleLng"
          :scale="16"
          :enable-3D="true"
          :show-compass="false"
          :enable-zoom="false"
          :enable-scroll="false"
          :enable-rotate="false"
          :markers="mapMarkers"
          :enable-satellite="false"
          :enable-traffic="false"
          :layer-style="mapLayerStyle"
        ></map>
        <view class="map-overlay"></view>
      </view>

      <!-- 顶部状态栏 -->
      <view class="top-status-bar">
        <text class="top-date">{{ currentDate }}</text>
        <text class="top-time">{{ currentTime }}</text>
        <text class="top-temp">{{ formatTemp(outsideTemp) }}</text>
      </view>

      <!-- 主内容区：左列(圆环+PRND) + 右列(圆环+信息) -->
      <view class="main-cluster">
        <!-- 左列 -->
        <view class="cluster-col left-col">
          <view class="gauge-ring left-ring">
            <view class="ring-outer">
              <view class="ring-inner">
                <template v-if="isCharging">
                  <text class="ring-unit">kW</text>
                  <text class="ring-value">{{ chargePowerDisplay }}</text>
                  <text class="ring-sub" v-if="chargeVoltage > 0">{{ chargeVoltage }}V</text>
                  <text class="ring-sub" v-if="chargeAmpere > 0">{{ chargeAmpere }}A</text>
                  <text class="ring-label">Charging</text>
                </template>
                <template v-else>
                  <text class="ring-unit">km/h</text>
                  <text class="ring-value">{{ Math.round(speed) }}</text>
                  <text class="ring-label">Speed</text>
                </template>
              </view>
            </view>
            <svg class="ring-svg" viewBox="0 0 200 200">
              <defs>
                <linearGradient id="leftGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" :stop-color="leftGradientStart"/>
                  <stop offset="100%" :stop-color="leftGradientEnd"/>
                </linearGradient>
              </defs>
              <circle class="ring-track" cx="100" cy="100" r="88" fill="none" stroke-width="6"/>
              <circle class="ring-progress" cx="100" cy="100" r="88" fill="none" stroke-width="6"
                stroke="url(#leftGradient)"
                :stroke-dasharray="leftRingCircumference"
                :stroke-dashoffset="leftRingOffset"
                stroke-linecap="round"/>
            </svg>
          </view>
          <!-- PRND 挡位 -->
          <view class="gear-selector">
            <text class="gear-item" :class="{ active: shiftState === 'P' }">P</text>
            <text class="gear-item" :class="{ active: shiftState === 'R' }">R</text>
            <text class="gear-item" :class="{ active: shiftState === 'N' }">N</text>
            <text class="gear-item" :class="{ active: shiftState === 'D' }">D</text>
          </view>
        </view>

        <!-- 右列 -->
        <view class="cluster-col right-col">
          <view class="gauge-ring right-ring">
            <view class="ring-outer">
              <view class="ring-inner">
                <template v-if="isCharging">
                  <text class="ring-pct">{{ Math.round(batteryPercent) }}%</text>
                  <text class="ring-value">{{ Math.round(rangeKm) }}</text>
                  <text class="ring-label">Range</text>
                </template>
                <template v-else>
                  <text class="ring-unit">kW</text>
                  <text class="ring-value">{{ drivePowerDisplay }}</text>
                  <text class="ring-label">Power</text>
                </template>
              </view>
            </view>
            <svg class="ring-svg" viewBox="0 0 200 200">
              <defs>
                <linearGradient id="rightGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" :stop-color="rightGradientStart"/>
                  <stop offset="100%" :stop-color="rightGradientEnd"/>
                </linearGradient>
              </defs>
              <circle class="ring-track" cx="100" cy="100" r="88" fill="none" stroke-width="6"/>
              <circle class="ring-progress" cx="100" cy="100" r="88" fill="none" stroke-width="6"
                stroke="url(#rightGradient)"
                :stroke-dasharray="rightRingCircumference"
                :stroke-dashoffset="rightRingOffset"
                stroke-linecap="round"/>
            </svg>
          </view>
          <!-- 右列信息：时间/温度/里程 -->
          <view class="info-panel">
            <text class="info-time">{{ currentTime }}</text>
            <text class="info-temp">{{ formatTemp(insideTemp) }}</text>
            <text class="info-odo">{{ odometerKm.toFixed(1) }} km</text>
          </view>
        </view>
      </view>
    </view>

    <view class="empty-screen" v-else>
      <Icon name="Car" :size="64" themeColor="inactiveLight" />
      <text class="empty-text">暂无车辆连接</text>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import Icon from '@/components/Icon/Icon.vue'
import { useVehicleStore } from '@/store/vehicle'
import { useVehicleData, initVehicleData, destroyVehicleData } from '@/utils/vehicle-data'
import { useThemeStore } from '@/store/theme'

const vehicleStore = useVehicleStore()
const vehicleDataStore = useVehicleData()
const themeStore = useThemeStore()

const themeClass = computed(() => themeStore.themeClass)
const inactiveIconColorLight = computed(() => themeStore.colors.inactiveIconLight)
const leftGradientStart = computed(() => themeStore.resolvedTheme === 'visionpro' ? '#0F172A' : '#3b82f6')
const leftGradientEnd = computed(() => themeStore.resolvedTheme === 'visionpro' ? '#334155' : '#06b6d4')
const rightGradientStart = computed(() => themeStore.resolvedTheme === 'visionpro' ? '#0F172A' : '#8b5cf6')
const rightGradientEnd = computed(() => themeStore.resolvedTheme === 'visionpro' ? '#334155' : '#ec4899')
const currentVehicle = computed(() => vehicleStore.currentVehicle)
const vehicleData = computed(() => vehicleDataStore.data)

const speed = computed(() => vehicleData.value?.speed || 0)
const batteryPercent = computed(() => vehicleData.value?.soc || 0)
const rangeKm = computed(() => vehicleData.value?.range_km || 0)
const shiftState = computed(() => vehicleData.value?.gear || 'P')
const outsideTemp = computed(() => vehicleData.value?.outside_temp)
const insideTemp = computed(() => vehicleData.value?.inside_temp)
const odometerKm = computed(() => vehicleData.value?.odometer_km || 0)

const isCharging = computed(() => vehicleData.value?.charging === true)

const chargePower = computed(() => vehicleData.value?.charge_power || 0)
const chargeVoltage = computed(() => vehicleData.value?.voltage || 0)
const chargeAmpere = computed(() => vehicleData.value?.ampere || 0)
const chargePowerDisplay = computed(() => Math.round(chargePower.value))

const drivePower = computed(() => vehicleData.value?.power || 0)
const drivePowerDisplay = computed(() => Math.round(Math.abs(drivePower.value)))

const vehicleLat = computed(() => vehicleData.value?.latitude || 39.9042)
const vehicleLng = computed(() => vehicleData.value?.longitude || 116.4074)
const vehicleHeading = computed(() => vehicleData.value?.heading || 0)

const mapLayerStyle = computed(() => {
  const isDark = themeStore.resolvedTheme === 'dark' || themeStore.resolvedTheme === 'visionpro'
  if (isDark) {
    const styleId = import.meta.env.VITE_TENCENT_MAP_STYLE_DARK || '2'
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

const mapMarkers = computed(() => {
  return [{
    id: 1,
    latitude: vehicleLat.value,
    longitude: vehicleLng.value,
    iconPath: '/static/car-marker.png',
    width: 48,
    height: 48,
    rotate: vehicleHeading.value,
    anchor: { x: 0.5, y: 0.5 }
  }]
})

const RING_RADIUS = 88
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS

const leftRingCircumference = RING_CIRCUMFERENCE
const leftRingOffset = computed(() => {
  let pct = 0
  if (isCharging.value) {
    pct = Math.min(chargePower.value / 250, 1)
  } else {
    pct = Math.min(speed.value / 240, 1)
  }
  return RING_CIRCUMFERENCE * (1 - pct)
})

const rightRingCircumference = RING_CIRCUMFERENCE
const rightRingOffset = computed(() => {
  let pct = 0
  if (isCharging.value) {
    pct = Math.min(batteryPercent.value / 100, 1)
  } else {
    pct = Math.min(Math.abs(drivePower.value) / 400, 1)
  }
  return RING_CIRCUMFERENCE * (1 - pct)
})

const currentTime = ref('')
const currentDate = ref('')
let timeTimer = null

const updateTime = () => {
  const now = new Date()
  const h = now.getHours().toString().padStart(2, '0')
  const m = now.getMinutes().toString().padStart(2, '0')
  currentTime.value = `${h}:${m}`
  const d = now.getDate().toString().padStart(2, '0')
  const mo = (now.getMonth() + 1).toString().padStart(2, '0')
  currentDate.value = `${d} / ${mo}`
}

const formatTemp = (t) => {
  if (t === null || t === undefined) return '--°C'
  return `${t.toFixed(1)}°C`
}

onMounted(() => {
  try { uni.setKeepScreenOn({ keepScreenOn: true }) } catch (e) {}

  // #ifdef APP-PLUS
  // 强制横屏
  try {
    plus.screen.lockOrientation('landscape')
  } catch (e) {}
  // 全屏模式
  try {
    plus.navigator.setFullscreen(true)
  } catch (e) {}
  // #endif

  updateTime()
  timeTimer = setInterval(updateTime, 1000)
  if (currentVehicle.value) {
    initVehicleData(currentVehicle.value.vin)
  }
  setTimeout(() => {
    try {
      const mapContext = uni.createMapContext('dashboardMap')
      if (mapContext && mapContext.setMapStyle) {
        const isDark = themeStore.resolvedTheme === 'dark' || themeStore.resolvedTheme === 'visionpro'
        if (isDark) {
          const styleId = import.meta.env.VITE_TENCENT_MAP_STYLE_DARK || '2'
          // #ifdef APP-PLUS
          mapContext.setMapStyle(parseInt(styleId) || 2)
          // #endif
          // #ifdef H5
          mapContext.setMapStyle('style' + styleId)
          // #endif
        }
      }
    } catch (e) {}
  }, 500)
})

onUnmounted(() => {
  try { uni.setKeepScreenOn({ keepScreenOn: false }) } catch (e) {}

  // #ifdef APP-PLUS
  // 恢复屏幕方向
  try {
    plus.screen.unlockOrientation()
  } catch (e) {}
  // 退出全屏
  try {
    plus.navigator.setFullscreen(false)
  } catch (e) {}
  // #endif

  if (timeTimer) clearInterval(timeTimer)
  destroyVehicleData()
})
</script>

<style lang="scss" scoped>
.instrument-page {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  position: relative;
  padding-top: calc(var(--status-bar-height, 44px) + 88rpx);
  box-sizing: border-box;
}

.cluster-screen {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  position: relative;
  z-index: 1;
  padding: calc(4rpx + env(safe-area-inset-top)) 40rpx calc(10rpx + env(safe-area-inset-bottom));
  box-sizing: border-box;
  gap: 4rpx;
}

.map-bg-container {
  position: absolute;
  inset: 0;
  z-index: 0;
}

.map-bg {
  width: 100%;
  height: 100%;
}

.map-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    180deg,
    rgba(0, 0, 0, 0.5) 0%,
    rgba(0, 0, 0, 0.3) 30%,
    rgba(0, 0, 0, 0.3) 70%,
    rgba(0, 0, 0, 0.5) 100%
  );
  pointer-events: none;
}

.top-status-bar {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20rpx;
  flex-shrink: 0;

  .top-date,
  .top-temp {
    font-size: 24rpx;
    color: rgba(255, 255, 255, 0.6);
    font-weight: 400;
    min-width: 80rpx;
  }

  .top-time {
    font-size: 36rpx;
    font-weight: 700;
    color: #ffffff;
    letter-spacing: 2rpx;
  }
}

.main-cluster {
  width: 100%;
  display: flex;
  align-items: flex-start;
  justify-content: space-around;
  padding: 0 40rpx;
  box-sizing: border-box;
}

.cluster-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16rpx;
  flex: 1;
}

.cluster-col .gauge-ring {
  align-self: center;
}

.cluster-col .gear-selector,
.cluster-col .info-panel {
  width: 100%;
}

.gauge-ring {
  position: relative;
  width: 180rpx;
  height: 180rpx;
  flex-shrink: 0;

  .ring-outer {
    position: absolute;
    inset: 0;
    border-radius: 50%;
    background: linear-gradient(145deg, rgba(255,255,255,0.12), rgba(255,255,255,0.04));
    box-shadow:
      inset 0 2px 4px rgba(255,255,255,0.08),
      0 8px 32px rgba(0,0,0,0.4);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .ring-inner {
    width: 144rpx;
    height: 144rpx;
    border-radius: 50%;
    background: linear-gradient(145deg, rgba(255,255,255,0.08), rgba(255,255,255,0.02));
    box-shadow:
      inset 0 2px 8px rgba(0,0,0,0.3),
      0 1px 2px rgba(255,255,255,0.08);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2rpx;
  }

  .ring-pct {
    font-size: 12rpx;
    color: rgba(255, 255, 255, 0.5);
    font-weight: 400;
  }

  .ring-unit {
    font-size: 12rpx;
    color: rgba(255, 255, 255, 0.5);
    font-weight: 400;
    margin-bottom: -2rpx;
  }

  .ring-value {
    font-size: 40rpx;
    font-weight: 200;
    color: #ffffff;
    line-height: 1;
    font-family: 'SF Pro Display', -apple-system, sans-serif;
  }

  .ring-sub {
    font-size: 10rpx;
    color: rgba(255, 255, 255, 0.4);
    font-weight: 400;
    line-height: 1.2;
  }

  .ring-label {
    font-size: 14rpx;
    color: rgba(255, 255, 255, 0.5);
    font-weight: 400;
    margin-top: 2rpx;
  }
}

.ring-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);

  .ring-track {
    stroke: rgba(255,255,255,0.1);
  }

  .ring-progress {
    transition: stroke-dashoffset 0.6s ease;
  }
}

.gear-selector {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: center;
  gap: 16rpx;
  height: 56rpx;
  box-sizing: border-box;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-radius: 28rpx;
  padding: 0 24rpx;
}

.gear-item {
  font-size: 26rpx;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.5);
  transition: all 0.3s ease;
  min-width: 36rpx;
  text-align: center;
  line-height: 1;

  &.active {
    color: #ffffff;
    font-weight: 700;
    font-size: 30rpx;
    text-shadow: 0 0 12px rgba(255, 255, 255, 0.5);
  }
}

.info-panel {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: center;
  gap: 16rpx;
  height: 56rpx;
  box-sizing: border-box;
  white-space: nowrap;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-radius: 28rpx;
  padding: 0 24rpx;

  .info-time {
    font-size: 26rpx;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.5);
    letter-spacing: 1rpx;
    white-space: nowrap;
    line-height: 1;
  }

  .info-temp {
    font-size: 26rpx;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.5);
    white-space: nowrap;
    line-height: 1;
  }

  .info-odo {
    font-size: 26rpx;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.5);
    white-space: nowrap;
    line-height: 1;
  }
}

.empty-screen {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 24rpx;
  position: relative;
  z-index: 1;

  .empty-text {
    font-size: 28rpx;
    color: rgba(255, 255, 255, 0.5);
    letter-spacing: 2rpx;
  }
}

/* ========== 横屏适配 ========== */
@media screen and (orientation: landscape) {
  .cluster-screen {
    padding: calc(6rpx + env(safe-area-inset-top)) 40rpx calc(6rpx + env(safe-area-inset-bottom));
    gap: 4rpx;
  }

  .main-cluster {
    flex-direction: row;
    justify-content: space-around;
    padding: 0 40rpx;
  }

  .cluster-col {
    gap: 53rpx;
  }

  .gauge-ring {
    width: 180rpx;
    height: 180rpx;

    .ring-inner {
      width: 144rpx;
      height: 144rpx;
    }

    .ring-value {
      font-size: 40rpx;
    }

    .ring-label {
      font-size: 14rpx;
    }

    .ring-unit,
    .ring-pct {
      font-size: 12rpx;
    }

    .ring-sub {
      font-size: 10rpx;
    }
  }

  .gear-selector {
    height: 48rpx;
    gap: 16rpx;
    border-radius: 24rpx;
    padding: 0 20rpx;
  }

  .gear-item {
    font-size: 22rpx;
    min-width: 32rpx;

    &.active {
      font-size: 26rpx;
    }
  }

  .info-panel {
    height: 48rpx;
    gap: 16rpx;
    border-radius: 24rpx;
    padding: 0 20rpx;

    .info-time,
    .info-temp,
    .info-odo {
      font-size: 22rpx;
    }
  }
}

/* ========== 竖屏适配 ========== */
@media screen and (orientation: portrait) {
  .cluster-screen {
    padding: calc(4rpx + env(safe-area-inset-top)) 32rpx calc(4rpx + env(safe-area-inset-bottom));
    gap: 4rpx;
  }

  .top-status-bar {
    padding: 0 12rpx;

    .top-date,
    .top-temp {
      font-size: 22rpx;
    }

    .top-time {
      font-size: 32rpx;
    }
  }

  .main-cluster {
    flex-direction: row;
    justify-content: space-around;
    align-items: flex-start;
    padding: 0 20rpx;
  }

  .cluster-col {
    gap: 14rpx;
  }

  .gauge-ring {
    width: 200rpx;
    height: 200rpx;

    .ring-inner {
      width: 160rpx;
      height: 160rpx;
      gap: 2rpx;
    }

    .ring-value {
      font-size: 44rpx;
    }

    .ring-label {
      font-size: 16rpx;
    }

    .ring-unit,
    .ring-pct {
      font-size: 14rpx;
    }

    .ring-sub {
      font-size: 12rpx;
    }
  }

  .gear-selector {
    height: 48rpx;
    gap: 12rpx;
    border-radius: 24rpx;
    padding: 0 16rpx;
  }

  .gear-item {
    font-size: 22rpx;
    min-width: 30rpx;

    &.active {
      font-size: 28rpx;
    }
  }

  .info-panel {
    height: 48rpx;
    gap: 12rpx;
    border-radius: 24rpx;
    padding: 0 16rpx;

    .info-time,
    .info-temp,
    .info-odo {
      font-size: 22rpx;
    }
  }
}

/* ========== Vision Pro 主题适配 ========== */
.visionpro-theme {
  .map-overlay {
    background: linear-gradient(
      180deg,
      rgba(245, 249, 255, 0.7) 0%,
      rgba(238, 244, 255, 0.5) 30%,
      rgba(238, 244, 255, 0.5) 70%,
      rgba(245, 249, 255, 0.7) 100%
    );
  }

  .top-date,
  .top-temp {
    color: rgba(15, 23, 42, 0.5);
  }

  .top-time {
    color: #0F172A;
  }

  .ring-outer {
    background: linear-gradient(145deg, rgba(255,255,255,0.58), rgba(255,255,255,0.3));
    box-shadow:
      inset 0 2px 4px rgba(255,255,255,0.3),
      0 8px 32px rgba(15, 23, 42, 0.08);
  }

  .ring-inner {
    background: linear-gradient(145deg, rgba(255,255,255,0.4), rgba(255,255,255,0.2));
  }

  .ring-pct,
  .ring-unit,
  .ring-label,
  .ring-sub {
    color: rgba(15, 23, 42, 0.5);
  }

  .ring-value {
    color: #0F172A;
  }

  .ring-track {
    stroke: rgba(15, 23, 42, 0.08);
  }

  .gear-selector {
    background: rgba(255, 255, 255, 0.35);
  }

  .gear-item {
    color: rgba(15, 23, 42, 0.4);

    &.active {
      color: #0F172A;
      text-shadow: none;
    }
  }

  .info-panel {
    background: rgba(255, 255, 255, 0.35);

    .info-time,
    .info-temp,
    .info-odo {
      color: rgba(15, 23, 42, 0.4);
    }
  }

  .empty-text {
    color: rgba(15, 23, 42, 0.4);
  }
}
</style>
