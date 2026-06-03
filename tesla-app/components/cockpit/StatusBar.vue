<template>
  <view class="cockpit-status-bar">
    <view class="status-left">
      <view class="status-dot" :style="{ backgroundColor: stateColor }" />
      <text class="status-label">{{ stateText }}</text>
    </view>
    <view class="status-center">
      <text class="status-time">{{ currentTime }}</text>
    </view>
    <view class="status-right">
      <view class="signal-bars">
        <view
          v-for="i in 4"
          :key="i"
          class="signal-bar"
          :class="{ active: i <= gpsLevel }"
          :style="{ height: (12 + i * 6) + 'rpx' }"
        />
      </view>
      <text class="latency" v-if="latencyDisplay">{{ latencyDisplay }}ms</text>
      <text class="vin">{{ vinShort }}</text>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useThemeStore } from '@/store/theme'
import { getDisplayStateLabel, getDisplayStateColor, isVehicleOnline, isVehicleAsleep } from '@/utils/vehicle-state'

const props = defineProps({
  vehicleData: {
    type: Object,
    default: () => ({})
  },
  stateOutput: {
    type: Object,
    default: () => ({})
  },
  latency: {
    type: Number,
    default: null
  },
  vin: {
    type: String,
    default: ''
  }
})

const themeStore = useThemeStore()

const currentTime = ref('')
let timer = null

const updateTime = () => {
  const now = new Date()
  const h = String(now.getHours()).padStart(2, '0')
  const m = String(now.getMinutes()).padStart(2, '0')
  currentTime.value = `${h}:${m}`
}

onMounted(() => {
  updateTime()
  timer = setInterval(updateTime, 10000)
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
})

const stateText = computed(() => {
  if (isVehicleAsleep(props.stateOutput)) return '休眠中'
  if (!isVehicleOnline(props.stateOutput)) return '离线'
  return getDisplayStateLabel(props.stateOutput, props.vehicleData)
})

const stateColor = computed(() => {
  if (isVehicleAsleep(props.stateOutput)) return '#9ca3af'
  if (!isVehicleOnline(props.stateOutput)) return '#6b7280'
  return getDisplayStateColor(props.stateOutput, props.vehicleData)
})

const gpsLevel = computed(() => {
  const state = props.vehicleData?.gps_state
  if (state === 4) return 4
  if (state === 3) return 3
  if (state === 2) return 2
  if (state === 1) return 1
  return 0
})

const latencyDisplay = computed(() => {
  return props.latency ?? null
})

const vinShort = computed(() => {
  if (!props.vin || typeof props.vin !== 'string') return ''
  return props.vin.slice(-8)
})
</script>

<style lang="scss" scoped>
.cockpit-status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 80rpx;
  padding: 0 24rpx;
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
}

.status-left {
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.status-dot {
  width: 14rpx;
  height: 14rpx;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-label {
  font-size: 24rpx;
  color: rgba(255, 255, 255, 0.7);
}

.status-center {
  display: flex;
  align-items: center;
}

.status-time {
  font-size: 28rpx;
  color: #fff;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.status-right {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.signal-bars {
  display: flex;
  align-items: flex-end;
  gap: 4rpx;
  height: 36rpx;
}

.signal-bar {
  width: 6rpx;
  border-radius: 2rpx;
  background: rgba(255, 255, 255, 0.2);
  transition: background 0.3s;

  &.active {
    background: rgba(255, 255, 255, 0.7);
  }
}

.latency {
  font-size: 22rpx;
  color: rgba(255, 255, 255, 0.5);
  font-variant-numeric: tabular-nums;
}

.vin {
  font-size: 22rpx;
  color: rgba(255, 255, 255, 0.5);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.5rpx;
}

:global(.visionpro-theme) {
  .cockpit-status-bar {
    background: rgba(255, 255, 255, 0.35);
  }

  .status-label {
    color: rgba(15, 23, 42, 0.7);
  }

  .status-time {
    color: #0F172A;
  }

  .status-dot {
    box-shadow: 0 0 6rpx currentColor;
  }

  .signal-bar {
    background: rgba(15, 23, 42, 0.15);

    &.active {
      background: rgba(15, 23, 42, 0.6);
    }
  }

  .latency,
  .vin {
    color: rgba(15, 23, 42, 0.5);
  }
}
</style>
