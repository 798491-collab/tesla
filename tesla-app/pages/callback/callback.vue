<template>
  <view class="callback-container" :class="themeClass">
    <NavBar title="授权回调" />
    <view v-if="loading" class="state-section">
      <view class="spin-wrap">
        <Icon name="Sync" :size="48" themeColor="primary" class="spin-icon" />
      </view>
      <text class="state-title">正在处理授权...</text>
      <text class="state-desc">请稍候，正在验证您的授权信息</text>
    </view>

    <view v-else-if="error" class="state-section">
      <view class="error-icon-wrap">
        <Icon name="Warning" :size="48" themeColor="primary" />
      </view>
      <text class="state-title error-title">授权失败</text>
      <text class="error-message">{{ errorMessage }}</text>
      <button class="btn-primary" @click="goBack">
        <Icon name="ChevronBack" :size="16" color="#fff" />
        <text>返回重试</text>
      </button>
    </view>

    <view v-else-if="success" class="state-section">
      <view class="success-icon-wrap">
        <Icon name="Shield" :size="48" color="#fff" />
      </view>
      <text class="state-title success-title">授权成功</text>
      <text class="state-desc">正在跳转至车辆选择页面...</text>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useThemeStore } from '@/store/theme'
import NavBar from '@/components/NavBar/NavBar.vue'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)

const loading = ref(true)
const error = ref(false)
const success = ref(false)
const errorMessage = ref('')

onMounted(() => {
  handleCallback()
})

const handleCallback = () => {
  // #ifdef H5
  handleH5Callback()
  // #endif
  
  // #ifdef APP-PLUS
  handleAppCallback()
  // #endif
}

// 从 URL 参数中获取数据
const getUrlParams = () => {
  // #ifdef H5
  const urlParams = new URLSearchParams(window.location.search)
  return {
    auth_id: urlParams.get('auth_id'),
    auth_data: urlParams.get('auth_data'), // 兼容旧方式
    error: urlParams.get('error'),
    error_description: urlParams.get('error_description')
  }
  // #endif
  
  // #ifdef APP-PLUS
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  const options = currentPage.options || currentPage.$route?.query || {}
  return {
    auth_id: options.auth_id,
    auth_data: options.auth_data, // 兼容旧方式
    error: options.error,
    error_description: options.error_description
  }
  // #endif
  
  return {}
}

// H5端和APP端统一处理回调
const handleH5Callback = async () => {
  await processCallback()
}

const handleAppCallback = async () => {
  await processCallback()
}

// 统一处理回调逻辑
const processCallback = async () => {
  const params = getUrlParams()
  
  if (params.error) {
    loading.value = false
    error.value = true
    errorMessage.value = decodeURIComponent(params.error_description || '授权过程中发生错误')
    return
  }
  
  // 优先使用新的 auth_id 方式
  if (params.auth_id) {
    await fetchAuthData(params.auth_id)
    return
  }
  
  // 兼容旧方式：直接从 URL 获取 auth_data
  if (params.auth_data) {
    await processAuthData(params.auth_data)
    return
  }
  
  loading.value = false
  error.value = true
  errorMessage.value = '未获取到授权信息'
}

// 通过 auth_id 从后端获取授权数据
const fetchAuthData = async (authId) => {
  try {
    const response = await uni.request({
      url: `${getBaseUrl()}/api/tesla/auth_data?auth_id=${authId}`,
      method: 'GET'
    })
    
    if (response.statusCode === 200 && response.data.code === 200) {
      const authData = response.data.data
      
      // 存储授权数据到本地
      uni.setStorageSync('tesla_auth_data', JSON.stringify(authData))
      
      loading.value = false
      success.value = true
      
      // 延迟跳转到绑定页面
      setTimeout(() => {
        uni.redirectTo({
          url: '/pages/bind/bind'
        })
      }, 1500)
    } else {
      throw new Error(response.data.message || '获取授权数据失败')
    }
  } catch (e) {
    loading.value = false
    error.value = true
    errorMessage.value = e.message || '获取授权数据失败，请重试'
  }
}

// 处理旧的 auth_data 方式（兼容）
const processAuthData = (authData) => {
  try {
    let base64 = authData.replace(/-/g, '+').replace(/_/g, '/')
    while (base64.length % 4) {
      base64 += '='
    }
    const decodedData = JSON.parse(atob(base64))
    
    // 存储授权数据到本地
    uni.setStorageSync('tesla_auth_data', JSON.stringify(decodedData))
    
    loading.value = false
    success.value = true
    
    // 延迟跳转到绑定页面
    setTimeout(() => {
      uni.redirectTo({
        url: '/pages/bind/bind'
      })
    }, 1500)
  } catch (e) {
    loading.value = false
    error.value = true
    errorMessage.value = '解析授权数据失败'
  }
}

// 获取基础 URL
const getBaseUrl = () => {
  // #ifdef H5
  return ''
  // #endif

  // #ifdef APP-PLUS
  return import.meta.env.VITE_API_BASE_URL || 'https://your-domain.com'
  // #endif

  return ''
}

const goBack = () => {
  uni.redirectTo({
    url: '/pages/bind/bind'
  })
}
</script>

<style lang="scss" scoped>
.callback-container {
  height: 100vh;
  overflow: hidden;
  box-sizing: border-box;
  background: linear-gradient(160deg, var(--dark-page-bg) 0%, var(--bg-card) 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60rpx;
  padding-top: calc(var(--status-bar-height, 44px) + 88rpx);
}

.state-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.spin-wrap {
  width: 120rpx;
  height: 120rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 40rpx;

  .spin-icon {
    animation: spin 1.2s linear infinite;
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.state-title {
  font-size: 40rpx;
  font-weight: 700;
  color: var(--dark-page-text);
  margin-bottom: 16rpx;

  &.error-title {
    color: #f87171;
  }

  &.success-title {
    color: #5BE7C4;
  }
}

.state-desc {
  font-size: 28rpx;
  color: var(--dark-page-text-hint);
  line-height: 1.6;
}

.error-icon-wrap {
  width: 140rpx;
  height: 140rpx;
  background: rgba(37, 99, 235, 0.15);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 40rpx;
  border: 3rpx solid rgba(37, 99, 235, 0.3);
}

.error-message {
  font-size: 28rpx;
  color: var(--dark-page-text-hint);
  margin-bottom: 60rpx;
  line-height: 1.6;
  max-width: 500rpx;
}

.success-icon-wrap {
  width: 140rpx;
  height: 140rpx;
  background: linear-gradient(135deg, #5BE7C4, #3cc9a5);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 40rpx;
  box-shadow: 0 8rpx 32rpx rgba(34, 197, 94, 0.3);
}

.btn-primary {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  background: var(--gradient);
  color: #ffffff;
  border-radius: 28rpx;
  height: 96rpx;
  font-size: 30rpx;
  font-weight: 600;
  border: none;
  padding: 0 64rpx;
  box-shadow: 0 8rpx 24rpx rgba(37, 99, 235, 0.35);
}
</style>
