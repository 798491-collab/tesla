<template>
  <view class="map-panel">
    <map
      v-if="hasLocation"
      id="cockpitMap"
      class="map-view"
      :latitude="latitude"
      :longitude="longitude"
      :scale="16"
      :enable-3D="true"
      :show-compass="false"
      :enable-zoom="false"
      :enable-scroll="false"
      :enable-rotate="false"
      :markers="markers"
      :enable-satellite="false"
      :enable-traffic="false"
      :subkey="tencentMapKey"
      @updated="onMapUpdated"
    ></map>
    <view v-else class="map-fallback">
      <Icon name="Location" :size="32" themeColor="inactiveLight" />
      <text class="map-fallback-text">暂无位置信息</text>
    </view>
    <view class="map-overlay-top">
      <view class="map-badge">
        <Icon name="Navigate" :size="14" color="#fff" />
        <text class="map-badge-text">导航</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, onMounted, getCurrentInstance } from 'vue'
import { useThemeStore } from '@/store/theme'

const props = defineProps({
  latitude: {
    type: Number,
    default: 39.9042
  },
  longitude: {
    type: Number,
    default: 116.4074
  },
  heading: {
    type: Number,
    default: 0
  },
  hasLocation: {
    type: Boolean,
    default: false
  }
})

const themeStore = useThemeStore()

const tencentMapKey = import.meta.env.VITE_TENCENT_MAP_KEY || ''
const darkStyleId = import.meta.env.VITE_TENCENT_MAP_STYLE_DARK || '2'
let isStyleSet = false

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

function applyMapDarkStyle() {
  if (isStyleSet) return
  try {
    const mapCtx = uni.createMapContext('cockpitMap', getCurrentInstance())
    if (mapCtx?.setMapStyle) {
      mapCtx.setMapStyle({
        styleId: darkStyleId,
        success: () => {
          console.log('[Cockpit] 地图墨渊主题设置成功')
          isStyleSet = true
        },
        fail: (err) => {
          console.warn('[Cockpit] 地图主题设置失败，尝试整数参数:', err)
          try {
            mapCtx.setMapStyle(parseInt(darkStyleId) || 2)
            isStyleSet = true
          } catch (e) {}
        }
      })
    }
  } catch (e) {}
}

function onMapUpdated() {
  applyMapDarkStyle()
}

onMounted(() => {
  setTimeout(() => applyMapDarkStyle(), 500)
})

const markers = computed(() => {
  return [
    {
      id: 1,
      latitude: props.latitude,
      longitude: props.longitude,
      iconPath: '/static/car-marker.png',
      rotate: props.heading,
      width: 28,
      height: 28,
      anchor: { x: 0.5, y: 0.5 }
    }
  ]
})
</script>

<style lang="scss" scoped>
.map-panel {
  position: relative;
  width: 100%;
  height: 100%;
  border-radius: 24rpx;
  overflow: hidden;
}

.map-view {
  width: 100%;
  height: 100%;
}

.map-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16rpx;
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1rpx solid rgba(255, 255, 255, 0.12);
  border-radius: 24rpx;
}

.map-fallback-text {
  font-size: 24rpx;
  color: rgba(255, 255, 255, 0.5);
}

.map-overlay-top {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  padding: 20rpx 24rpx;
  background: linear-gradient(to bottom, rgba(0, 0, 0, 0.4), transparent);
  display: flex;
  align-items: center;
  pointer-events: none;
}

.map-badge {
  display: flex;
  align-items: center;
  gap: 6rpx;
  padding: 8rpx 20rpx;
  background: rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1rpx solid rgba(255, 255, 255, 0.15);
  border-radius: 100rpx;
  pointer-events: auto;
}

.map-badge-text {
  font-size: 22rpx;
  color: #fff;
}

:global(.visionpro-theme) .map-fallback {
  background: rgba(255, 255, 255, 0.58);
  border: 1rpx solid rgba(255, 255, 255, 0.65);
}

:global(.visionpro-theme) .map-fallback-text {
  color: rgba(15, 23, 42, 0.5);
}

:global(.visionpro-theme) .map-overlay-top {
  background: linear-gradient(to bottom, rgba(238, 244, 255, 0.6), transparent);
}

:global(.visionpro-theme) .map-badge {
  background: rgba(15, 23, 42, 0.08);
  border: 1rpx solid rgba(15, 23, 42, 0.1);
}

:global(.visionpro-theme) .map-badge-text {
  color: #0F172A;
}
</style>
