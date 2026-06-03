<template>
  <view class="soc-panel">
    <view class="soc-header">
      <Icon name="Battery" :size="20" themeColor="primary" />
      <text class="soc-title">能源</text>
    </view>

    <view class="soc-main">
      <text class="soc-value" :style="{ color: socColor }">{{ Math.round(soc) }}%</text>
      <view class="soc-bar-track">
        <view class="soc-bar-fill" :style="{ width: soc + '%', backgroundColor: socColor }" />
      </view>
    </view>

    <view class="soc-info-list">
      <view class="soc-info-item">
        <text class="soc-info-label">续航</text>
        <text class="soc-info-value">{{ Math.round(rangeKm) }} km</text>
      </view>
      <view class="soc-info-item" v-if="batteryTemp !== null">
        <text class="soc-info-label">电池温度</text>
        <text class="soc-info-value">{{ batteryTemp.toFixed(1) }}°C</text>
      </view>
      <view class="soc-info-item">
        <text class="soc-info-label">功率</text>
        <text class="soc-info-value" :class="{ positive: power > 0, negative: power < 0 }">{{ powerDisplay }}</text>
      </view>
      <view class="soc-info-item" v-if="insideTemp !== null">
        <text class="soc-info-label">车内</text>
        <text class="soc-info-value">{{ insideTemp.toFixed(1) }}°C</text>
      </view>
      <view class="soc-info-item" v-if="outsideTemp !== null">
        <text class="soc-info-label">车外</text>
        <text class="soc-info-value">{{ outsideTemp.toFixed(1) }}°C</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  soc: { type: Number, default: 0 },
  rangeKm: { type: Number, default: 0 },
  batteryTemp: { type: Number, default: null },
  power: { type: Number, default: 0 },
  isCharging: { type: Boolean, default: false },
  insideTemp: { type: Number, default: null },
  outsideTemp: { type: Number, default: null }
})

const socColor = computed(() => {
  if (props.soc > 60) return '#4caf50'
  if (props.soc >= 20) return '#ff9800'
  return '#f44336'
})

const powerDisplay = computed(() => {
  const abs = Math.abs(props.power).toFixed(1)
  if (props.power > 0) return `+${abs} kW`
  if (props.power < 0) return `-${abs} kW`
  return `${abs} kW`
})
</script>

<style lang="scss" scoped>
.soc-panel {
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-radius: 24rpx;
  padding: 24rpx;
}

.soc-header {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8rpx;
  margin-bottom: 20rpx;
}

.soc-title {
  font-size: 24rpx;
  color: rgba(255, 255, 255, 0.7);
  font-weight: 500;
}

.soc-main {
  margin-bottom: 20rpx;
}

.soc-value {
  font-size: 56rpx;
  font-weight: 700;
  line-height: 1.2;
  margin-bottom: 12rpx;
}

.soc-bar-track {
  width: 100%;
  height: 6rpx;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3rpx;
  overflow: hidden;
}

.soc-bar-fill {
  height: 100%;
  border-radius: 3rpx;
  transition: width 0.6s ease, background-color 0.3s ease;
}

.soc-info-list {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.soc-info-item {
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
}

.soc-info-label {
  font-size: 22rpx;
  color: rgba(255, 255, 255, 0.5);
}

.soc-info-value {
  font-size: 28rpx;
  color: rgba(255, 255, 255, 0.9);
  font-weight: 500;
}

.soc-info-value.positive {
  color: #f44336;
}

.soc-info-value.negative {
  color: #4caf50;
}

:global(.visionpro-theme) .soc-panel {
  background: rgba(255, 255, 255, 0.58);
  backdrop-filter: blur(30px);
  -webkit-backdrop-filter: blur(30px);
  border: 1rpx solid rgba(255, 255, 255, 0.65);
}

:global(.visionpro-theme) .soc-title {
  color: rgba(15, 23, 42, 0.6);
}

:global(.visionpro-theme) .soc-value {
  color: #0F172A;
}

:global(.visionpro-theme) .soc-bar-track {
  background: rgba(15, 23, 42, 0.08);
}

:global(.visionpro-theme) .soc-info-label {
  color: rgba(15, 23, 42, 0.45);
}

:global(.visionpro-theme) .soc-info-value {
  color: rgba(15, 23, 42, 0.85);
}
</style>
