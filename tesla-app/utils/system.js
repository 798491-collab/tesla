import { ref, onMounted } from 'vue'

// 获取系统信息（状态栏高度等）
export function useSystemInfo() {
  const statusBarHeight = ref(0)
  const screenHeight = ref(0)
  const windowHeight = ref(0)
  const safeAreaBottom = ref(0)

  onMounted(() => {
    const sysInfo = uni.getSystemInfoSync()
    statusBarHeight.value = sysInfo.statusBarHeight || 0
    screenHeight.value = sysInfo.screenHeight || 0
    windowHeight.value = sysInfo.windowHeight || 0
    safeAreaBottom.value = sysInfo.safeAreaInsets?.bottom || 0
  })

  return {
    statusBarHeight,
    screenHeight,
    windowHeight,
    safeAreaBottom
  }
}

// 获取导航栏样式
export function getNavBarStyle(statusBarHeight) {
  return {
    paddingTop: `${statusBarHeight}px`,
    height: `${statusBarHeight + 44}px`
  }
}
