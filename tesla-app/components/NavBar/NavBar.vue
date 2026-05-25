<template>
  <view class="navbar" :style="{ paddingTop: statusBarHeight + 'px' }">
    <view class="navbar-inner">
      <view class="navbar-left" v-if="showBack" @click="goBack">
        <Icon name="ChevronBack" :size="22" :color="iconColor" />
      </view>
      <view class="navbar-title">
        <text class="navbar-title-text">{{ title }}</text>
      </view>
      <view class="navbar-right">
        <slot name="right"></slot>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'
import { useThemeStore } from '@/store/theme'

const props = defineProps({
  title: {
    type: String,
    default: ''
  },
  showBack: {
    type: Boolean,
    default: true
  }
})

const themeStore = useThemeStore()

const iconColor = computed(() => themeStore.colors.iconColor)

const statusBarHeight = uni.getSystemInfoSync().statusBarHeight || 0

const goBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
  } else {
    uni.reLaunch({ url: '/pages/dashboard/dashboard' })
  }
}
</script>

<style lang="scss" scoped>
.navbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 998;
  background: var(--navbar-bg);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-bottom: 1rpx solid var(--navbar-border);
}

.navbar-inner {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 24rpx;
  position: relative;
}

.navbar-left {
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  &:active {
    opacity: 0.6;
  }
}

.navbar-title {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
}

.navbar-title-text {
  font-size: 34rpx;
  font-weight: 600;
  color: var(--navbar-title);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.navbar-right {
  width: 72rpx;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-shrink: 0;
}
</style>
