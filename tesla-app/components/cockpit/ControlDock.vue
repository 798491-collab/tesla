<template>
  <view class="control-dock">
    <view class="dock-scroll">
      <view class="dock-item" :class="{ active: !locked }" @click="$emit('command', 'lock')">
        <view class="dock-icon-wrap">
          <Icon :name="locked ? 'LockClosed' : 'LockOpen'" :size="22" :color="locked ? '' : '#fff'" :themeColor="locked ? 'primary' : ''" />
        </view>
        <text class="dock-label">{{ locked ? '已锁' : '已解锁' }}</text>
      </view>

      <view class="dock-item" :class="{ active: climateOn }" @click="$emit('command', 'climate')">
        <view class="dock-icon-wrap">
          <Icon name="Snow" :size="22" :color="climateOn ? '#fff' : ''" :themeColor="climateOn ? '' : 'primary'" />
        </view>
        <text class="dock-label">{{ climateOn ? '空调开' : '空调关' }}</text>
      </view>

      <view class="dock-item" :class="{ active: isCharging }" @click="$emit('command', 'charge')">
        <view class="dock-icon-wrap">
          <Icon name="Flash" :size="22" :color="isCharging ? '#fff' : ''" :themeColor="isCharging ? '' : 'primary'" />
        </view>
        <text class="dock-label">{{ isCharging ? '充电中' : '充电' }}</text>
      </view>

      <view class="dock-item" :class="{ active: trunkOpen }" @click="$emit('command', 'trunk')">
        <view class="dock-icon-wrap">
          <Icon name="Exit" :size="22" :color="trunkOpen ? '#fff' : ''" :themeColor="trunkOpen ? '' : 'primary'" />
        </view>
        <text class="dock-label">{{ trunkOpen ? '开' : '后备箱' }}</text>
      </view>

      <view class="dock-item" :class="{ active: sentryOn }" @click="$emit('command', 'sentry')">
        <view class="dock-icon-wrap">
          <Icon name="Shield" :size="22" :color="sentryOn ? '#fff' : ''" :themeColor="sentryOn ? '' : 'primary'" />
        </view>
        <text class="dock-label">{{ sentryOn ? '哨兵开' : '哨兵' }}</text>
      </view>

      <view class="dock-item" :class="{ active: windowsOpen }" @click="$emit('command', 'window')">
        <view class="dock-icon-wrap">
          <Icon name="Window" :size="22" :color="windowsOpen ? '#fff' : ''" :themeColor="windowsOpen ? '' : 'primary'" />
        </view>
        <text class="dock-label">{{ windowsOpen ? '开窗' : '车窗' }}</text>
      </view>
    </view>
  </view>
</template>

<script setup>
defineProps({
  locked: { type: Boolean, default: true },
  climateOn: { type: Boolean, default: false },
  isCharging: { type: Boolean, default: false },
  trunkOpen: { type: Boolean, default: false },
  sentryOn: { type: Boolean, default: false },
  windowsOpen: { type: Boolean, default: false },
  vin: { type: String, default: '' }
})

defineEmits(['command'])
</script>

<style lang="scss" scoped>
.control-dock {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 16rpx 24rpx calc(env(safe-area-inset-bottom) + 16rpx);
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-radius: 28rpx 28rpx 0 0;
  z-index: 100;
}

.dock-scroll {
  display: flex;
  flex-direction: row;
  gap: 16rpx;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;

  &::-webkit-scrollbar {
    display: none;
  }
}

.dock-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8rpx;
  flex-shrink: 0;
  padding: 8rpx 0;
  transition: transform 0.15s ease;

  &:active {
    transform: scale(0.88);
  }
}

.dock-icon-wrap {
  width: 80rpx;
  height: 80rpx;
  border-radius: 24rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.06);
  transition: all 0.3s ease;
}

.dock-item.active .dock-icon-wrap {
  background: linear-gradient(135deg, #5BE7C4, #3cc9a5);
  box-shadow: 0 4rpx 16rpx rgba(91, 231, 196, 0.35);
}

.dock-label {
  font-size: 20rpx;
  color: rgba(255, 255, 255, 0.6);
  white-space: nowrap;
}

:global(.visionpro-theme) .control-dock {
  background: rgba(255, 255, 255, 0.58);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
}

:global(.visionpro-theme) .dock-icon-wrap {
  background: rgba(15, 23, 42, 0.05);
}

:global(.visionpro-theme) .dock-item.active .dock-icon-wrap {
  background: linear-gradient(135deg, #0F172A, #334155);
  box-shadow: 0 4rpx 16rpx rgba(15, 23, 42, 0.25);
}

:global(.visionpro-theme) .dock-label {
  color: rgba(15, 23, 42, 0.5);
}
</style>
