<template>
  <view class="icon-wrapper" :style="wrapperStyle">

    <!-- SVG image（本地 + vicons 都走 base64） -->
    <image
      v-if="iconSrc"
      :src="iconSrc"
      :style="imageStyle"
      mode="aspectFit"
    />

    <!-- fallback -->
    <view v-else class="empty-icon"></view>

  </view>
</template>

<script setup>
import { computed } from 'vue'
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
 * SVG base64（App/H5 都稳定）
 */
const iconSrc = computed(() => {
  // 优先使用本地图标
  if (localIcon.value) {
    let content = localIcon.value.content
    content = content.replace(/currentColor/g, props.color)

    const svg = `<svg xmlns="http://www.w3.org/2000/svg"
      viewBox="${localIcon.value.viewBox}"
      fill="${props.color}">
      ${content}
    </svg>`

    return 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg)
  }

  // 回退：从 vicons 组件渲染 SVG
  // uni-app 不支持 <component :is="">，所以这里无法动态渲染
  // 所有图标应通过 iconPaths.js 提供
  return ''
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

.empty-icon {
  width: 100%;
  height: 100%;
}
</style>
