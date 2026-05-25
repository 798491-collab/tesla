<template>
  <view class="icon-wrapper" :style="wrapperStyle">

    <!-- 本地 SVG -->
    <image
      v-if="iconSrc"
      :src="iconSrc"
      :style="imageStyle"
      mode="aspectFit"
    />

    <!-- vicons -->
    <view
      v-else-if="iconComponent"
      class="vicon-wrap"
    >
      <component
        :is="iconComponent"
        :size="sizeNum"
        :color="color"
        class="vicon"
      />
    </view>

    <!-- fallback -->
    <view v-else class="empty-icon"></view>

  </view>
</template>

<script setup>
import { computed } from 'vue'
import * as VIcons from '@vicons/ionicons5'
import iconData from '@/utils/iconPaths.js'

const props = defineProps({
  name: {
    type: String,
    required: true
  },
  size: {
    type: [Number, String],
    default: 24
  },
  color: {
    type: String,
    default: '#333'
  }
})

/**
 * size 统一成 number（给 vicons 用）
 */
const sizeNum = computed(() => {
  return typeof props.size === 'number'
    ? props.size
    : parseInt(props.size) || 24
})

const sizeStr = computed(() => {
  return typeof props.size === 'number'
    ? `${props.size}px`
    : props.size
})

/**
 * 本地图标
 */
const localIcon = computed(() => {
  return iconData[props.name] || null
})

/**
 * vicons
 */
const iconComponent = computed(() => {
  return VIcons[props.name] || null
})

/**
 * SVG base64（App/H5 都稳定）
 */
const iconSrc = computed(() => {
  if (!localIcon.value) return ''

  let content = localIcon.value.content

  content = content.replace(/currentColor/g, props.color)

  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg"
      viewBox="${localIcon.value.viewBox}"
      fill="${props.color}">
      ${content}
    </svg>
  `

  return 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg)
})

/**
 * 样式
 */
const wrapperStyle = computed(() => ({
  width: sizeStr.value,
  height: sizeStr.value,
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  overflow: 'hidden',
  lineHeight: 1
}))

const imageStyle = computed(() => ({
  width: '100%',
  height: '100%'
}))
</script>

<style scoped>
.icon-wrapper {
  line-height: 1;
}

.vicon-wrap {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* vicons */
.vicon {
  width: 100%;
  height: 100%;
}

/* fallback */
.empty-icon {
  width: 100%;
  height: 100%;
}
</style>