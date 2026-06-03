<template>
  <view class="cockpit-page" :class="themeClass">
    <view class="cockpit-screen" v-if="currentVehicle">
      <view class="map-bg-container">
        <map
          v-if="hasLocation"
          id="cockpitMap"
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
        <view class="map-dark-overlay"></view>
      </view>

      <view class="cockpit-content">
        <StatusBar
          :vehicleData="vehicleData"
          :stateOutput="stateOutput"
          :latency="commandLatency"
          :vin="currentVIN"
        />

        <view class="cockpit-main">
          <view class="cockpit-left">
            <SocPanel
              :soc="batteryPercent"
              :rangeKm="rangeKm"
              :batteryTemp="batteryTemp"
              :power="drivePower"
              :isCharging="isCharging"
              :insideTemp="insideTemp"
              :outsideTemp="outsideTemp"
            />
          </view>

          <view class="cockpit-center">
            <SpeedDial
              :speed="speed"
              :maxSpeed="240"
              :gear="shiftState"
              :soc="batteryPercent"
              :rangeKm="rangeKm"
              :isCharging="isCharging"
              :power="drivePower"
              :theme="dialTheme"
            />
          </view>

          <view class="cockpit-right">
            <MapPanel
              :latitude="vehicleLat"
              :longitude="vehicleLng"
              :heading="vehicleHeading"
              :hasLocation="hasLocation"
            />
          </view>
        </view>

        <ControlDock
          :locked="locked"
          :climateOn="climateOn"
          :isCharging="isCharging"
          :trunkOpen="trunkOpen"
          :sentryOn="sentryOn"
          :windowsOpen="windowsOpen"
          :vin="currentVIN"
          @command="handleCommand"
        />
      </view>
    </view>

    <view class="empty-screen" v-else>
      <Icon name="CarSport" :size="64" themeColor="inactiveLight" />
      <text class="empty-text">暂无车辆连接</text>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { useVehicleStore } from '@/store/vehicle'
import { useUserStore } from '@/store/user'
import { useVehicleData, initVehicleData, destroyVehicleData } from '@/utils/vehicle-data'
import { useThemeStore } from '@/store/theme'
import { isVehicleOnline } from '@/utils/vehicle-state'
import { getVehicleState, wakeVehicle } from '@/api/vehicle.js'
import {
  doorLock, doorUnlock,
  autoConditioningStart, autoConditioningStop,
  actuateTrunk, setSentryMode,
  windowControl, chargeStart, chargeStop
} from '@/api/control.js'
import StatusBar from '@/components/cockpit/StatusBar.vue'
import SpeedDial from '@/components/cockpit/SpeedDial.vue'
import SocPanel from '@/components/cockpit/SocPanel.vue'
import MapPanel from '@/components/cockpit/MapPanel.vue'
import ControlDock from '@/components/cockpit/ControlDock.vue'

const vehicleStore = useVehicleStore()
const userStore = useUserStore()
const vehicleDataStore = useVehicleData()
const themeStore = useThemeStore()

const themeClass = computed(() => themeStore.themeClass)
const dialTheme = computed(() => {
  const t = themeStore.resolvedTheme
  return t === 'visionpro' ? 'light' : t
})

const currentVehicle = computed(() => vehicleStore.currentVehicle)
const vehicleData = computed(() => vehicleDataStore.data)
const stateOutput = computed(() => vehicleDataStore.stateOutput)

const speed = computed(() => vehicleData.value?.speed || 0)
const batteryPercent = computed(() => vehicleData.value?.soc || 0)
const rangeKm = computed(() => vehicleData.value?.range_km || 0)
const shiftState = computed(() => vehicleData.value?.gear || 'P')
const drivePower = computed(() => vehicleData.value?.power || 0)
const isCharging = computed(() => vehicleData.value?.charging === true)
const locked = computed(() => vehicleData.value?.locked !== false)
const climateOn = computed(() => vehicleData.value?.is_ac_on)
const trunkOpen = computed(() => vehicleData.value?.trunk_open || vehicleData.value?.frunk_open)
const sentryOn = computed(() => vehicleData.value?.sentry_mode === true)
const windowsOpen = computed(() => vehicleData.value?.windows_open === true)
const insideTemp = computed(() => {
  const t = vehicleData.value?.inside_temp
  return (t !== null && t !== undefined && t !== 0) ? t : null
})
const outsideTemp = computed(() => {
  const t = vehicleData.value?.outside_temp
  return (t !== null && t !== undefined && t !== 0) ? t : null
})
const batteryTemp = computed(() => {
  const t = vehicleData.value?.battery_temp
  return (t !== null && t !== undefined) ? t : null
})

const hasLocation = computed(() => {
  const lat = vehicleData.value?.latitude
  const lng = vehicleData.value?.longitude
  return lat && lng && lat !== 0 && lng !== 0
})
const vehicleLat = computed(() => {
  const lat = vehicleData.value?.latitude
  return (lat && lat !== 0) ? lat : 39.9042
})
const vehicleLng = computed(() => {
  const lng = vehicleData.value?.longitude
  return (lng && lng !== 0) ? lng : 116.4074
})
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
  if (!hasLocation.value) return []
  return [{
    id: 1,
    latitude: vehicleData.value.latitude,
    longitude: vehicleData.value.longitude,
    iconPath: '/static/car-marker.png',
    width: 30,
    height: 30,
    rotate: vehicleHeading.value,
    anchor: { x: 0.5, y: 0.5 }
  }]
})

const currentVIN = computed(() => currentVehicle.value?.vin)
const commandLatency = computed(() => vehicleDataStore.commandLatencyMs)

const executeCommand = async (commandFn, commandName, needWake = true) => {
  if (!currentVIN.value) {
    uni.showToast({ title: '请先选择车辆', icon: 'none' })
    return false
  }

  const vehicleOnline = isVehicleOnline(stateOutput.value)

  if (needWake && !vehicleOnline) {
    uni.showLoading({ title: '唤醒车辆中...' })
    try {
      await wakeVehicle(currentVIN.value)
      uni.showLoading({ title: '车辆唤醒中，等待上线...' })
      await new Promise(resolve => setTimeout(resolve, 5000))
      for (let i = 0; i < 6; i++) {
        try {
          const stateRes = await getVehicleState(currentVIN.value)
          if (stateRes.data?.online) break
        } catch (e) {}
        await new Promise(resolve => setTimeout(resolve, 3000))
      }
    } catch (err) {
      uni.hideLoading()
      uni.showToast({ title: '唤醒失败: ' + (err.message || '未知错误'), icon: 'none' })
      return false
    }
  }

  uni.showLoading({ title: '执行中...' })
  try {
    await commandFn(currentVIN.value)
    uni.hideLoading()
    uni.showToast({ title: `${commandName}成功`, icon: 'success' })
    return true
  } catch (err) {
    uni.hideLoading()
    const errMsg = (err.message || '').toLowerCase()
    if (errMsg.includes('public key not paired') || errMsg.includes('virtual key not paired')) {
      uni.showToast({ title: '请先完成虚拟钥匙配对', icon: 'none' })
      return false
    }
    uni.showToast({ title: err.message || `${commandName}失败`, icon: 'none' })
    return false
  }
}

const handleCommand = async (cmd) => {
  switch (cmd) {
    case 'lock':
      if (locked.value) {
        await executeCommand(doorUnlock, '解锁')
      } else {
        await executeCommand(doorLock, '上锁')
      }
      break
    case 'climate':
      if (climateOn.value) {
        await executeCommand(autoConditioningStop, '关闭空调')
      } else {
        await executeCommand(autoConditioningStart, '开启空调')
      }
      break
    case 'charge':
      if (isCharging.value) {
        await executeCommand(chargeStop, '停止充电')
      } else {
        await executeCommand(chargeStart, '开始充电')
      }
      break
    case 'trunk':
      await executeCommand(actuateTrunk, '后备箱操作')
      break
    case 'sentry':
      await executeCommand((vin) => setSentryMode(vin, !sentryOn.value), sentryOn.value ? '关闭哨兵' : '开启哨兵')
      break
    case 'window':
      await executeCommand((vin) => windowControl(vin, windowsOpen.value ? 'vent' : 'close'), windowsOpen.value ? '关窗' : '开窗')
      break
  }
}

onMounted(async () => {
  try { uni.setKeepScreenOn({ keepScreenOn: true }) } catch (e) {}
  if (!vehicleStore.hasVehicles) {
    await vehicleStore.fetchVehicles()
  }
  if (currentVehicle.value?.vin) {
    initVehicleData(currentVehicle.value.vin)
  }
})

onShow(async () => {
  if (!userStore.checkTokenExpiry()) {
    uni.reLaunch({ url: '/pages/login/login' })
    return
  }
  if (!vehicleStore.hasVehicles) {
    await vehicleStore.fetchVehicles()
  }
  if (currentVehicle.value?.vin) {
    initVehicleData(currentVehicle.value.vin)
  }
})

onHide(() => {
  destroyVehicleData()
})

onUnmounted(() => {
  try { uni.setKeepScreenOn({ keepScreenOn: false }) } catch (e) {}
  destroyVehicleData()
})

watch(() => currentVehicle.value, (newVal) => {
  destroyVehicleData()
  if (newVal) {
    initVehicleData(newVal.vin)
  }
})
</script>

<style lang="scss" scoped>
.cockpit-page {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: #060B14;
  overflow: hidden;
}

.cockpit-screen {
  width: 100%;
  height: 100%;
  position: relative;
  z-index: 1;
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

.map-dark-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    180deg,
    rgba(6, 11, 20, 0.85) 0%,
    rgba(6, 11, 20, 0.7) 30%,
    rgba(6, 11, 20, 0.7) 70%,
    rgba(6, 11, 20, 0.85) 100%
  );
  pointer-events: none;
}

.cockpit-content {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  padding-top: env(safe-area-inset-top);
  box-sizing: border-box;
}

.cockpit-main {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16rpx 24rpx;
  gap: 24rpx;
  min-height: 0;
}

.cockpit-left {
  flex-shrink: 0;
  width: 260rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cockpit-center {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
}

.cockpit-right {
  flex-shrink: 0;
  width: 300rpx;
  height: 360rpx;
  display: flex;
  align-items: center;
  justify-content: center;
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

.visionpro-theme {
  .map-dark-overlay {
    background: linear-gradient(
      180deg,
      rgba(238, 244, 255, 0.88) 0%,
      rgba(238, 244, 255, 0.75) 30%,
      rgba(238, 244, 255, 0.75) 70%,
      rgba(238, 244, 255, 0.88) 100%
    );
  }

  .empty-text {
    color: rgba(15, 23, 42, 0.4);
  }
}

@media screen and (orientation: landscape) {
  .cockpit-main {
    padding: 8rpx 32rpx;
    gap: 32rpx;
  }

  .cockpit-left {
    width: 240rpx;
  }

  .cockpit-right {
    width: 280rpx;
    height: 320rpx;
  }
}

@media screen and (orientation: portrait) {
  .cockpit-main {
    flex-direction: column;
    padding: 8rpx 24rpx;
    gap: 16rpx;
  }

  .cockpit-left {
    width: 100%;
    order: 2;
  }

  .cockpit-center {
    order: 1;
  }

  .cockpit-right {
    width: 100%;
    height: 240rpx;
    order: 3;
  }
}
</style>
