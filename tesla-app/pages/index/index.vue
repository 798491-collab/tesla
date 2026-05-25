<template>
  <view class="login-container" :class="themeClass" :style="{ paddingTop: statusBarHeight + 'px' }">
    <view class="logo-section">
      <view class="logo-icon-wrap">
        <Icon name="CarSport" :size="40" themeColor="primary" />
      </view>
      <text class="logo-text">Tesla</text>
      <text class="sub-title">管理平台</text>
    </view>

    <view class="form-card">
      <view class="form-item">
        <view class="input-wrap">
          <view class="input-icon">
            <Icon name="Person" :size="18" themeColor="primary" />
          </view>
          <input
            class="input"
            v-model="form.username"
            placeholder="请输入用户名"
            maxlength="50"
          />
        </view>
      </view>
      <view class="form-item last">
        <view class="input-wrap">
          <view class="input-icon">
            <Icon name="LockClosed" :size="18" themeColor="primary" />
          </view>
          <input
            class="input"
            v-model="form.password"
            placeholder="请输入密码"
            password
            maxlength="50"
          />
        </view>
      </view>

      <button class="btn-primary" @click="handleLogin" :disabled="loading">
        <text>{{ loading ? '登录中...' : '登录' }}</text>
      </button>

      <view class="action-links">
        <text class="link" @click="goRegister">注册账号</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useUserStore } from '@/store/user'
import { useSystemInfo } from '@/utils/system.js'
import { useThemeStore } from '@/store/theme'

const userStore = useUserStore()
const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)
const form = ref({
  username: '',
  password: ''
})
const loading = ref(false)

const { statusBarHeight } = useSystemInfo()

onMounted(() => {
  if (userStore.isLoggedIn) {
    userStore.fetchUserInfo().then(() => {
      uni.reLaunch({ url: '/pages/dashboard/dashboard' })
    }).catch(() => {
      userStore.logout()
    })
  }
})

const handleLogin = async () => {
  if (!form.value.username || !form.value.password) {
    uni.showToast({ title: '请输入用户名和密码', icon: 'none' })
    return
  }
  loading.value = true
  try {
    await userStore.loginAction(form.value)
    uni.showToast({ title: '登录成功', icon: 'success' })
    setTimeout(() => {
      uni.reLaunch({ url: '/pages/dashboard/dashboard' })
    }, 500)
  } catch (err) {
    uni.showToast({ title: err.message || '登录失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

const goRegister = () => {
  uni.navigateTo({ url: '/pages/register/register' })
}
</script>

<style lang="scss" scoped>
.login-container {
  height: 100vh;
  overflow: hidden;
  box-sizing: border-box;
  background: linear-gradient(160deg, var(--dark-page-bg) 0%, var(--bg-card) 50%, var(--bg-card) 100%);
  padding: 0 40rpx;
}

.logo-section {
  text-align: center;
  padding: 120rpx 0 80rpx;

  .logo-icon-wrap {
    width: 120rpx;
    height: 120rpx;
    background: var(--bg-icon-wrap);
    border-radius: 32rpx;
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 32rpx;
    border: 2rpx solid var(--dark-page-glass-border);
  }

  .logo-text {
    font-size: 72rpx;
    font-weight: 700;
    color: var(--dark-page-text);
    letter-spacing: 4rpx;
    display: block;
  }

  .sub-title {
    display: block;
    font-size: 30rpx;
    color: var(--dark-page-text-hint);
    margin-top: 12rpx;
    letter-spacing: 8rpx;
  }
}

.form-card {
  background: var(--dark-page-glass-bg);
  border: 1rpx solid var(--dark-page-glass-border);
  border-radius: 32rpx;
  padding: 48rpx 40rpx;
}

.form-item {
  margin-bottom: 28rpx;

  &.last {
    margin-bottom: 0;
  }

  .input-wrap {
    display: flex;
    align-items: center;
    background: var(--dark-page-glass-bg);
    border: 1rpx solid var(--dark-page-glass-border);
    border-radius: 24rpx;
    height: 100rpx;
    padding: 0 28rpx;
  }

  .input-icon {
    width: 56rpx;
    height: 56rpx;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 16rpx;
    flex-shrink: 0;
  }

  .input {
    flex: 1;
    font-size: 30rpx;
    color: var(--dark-page-text);
    height: 100rpx;
  }
}

.btn-primary {
  margin-top: 48rpx;
  background: var(--gradient);
  color: var(--dark-page-text);
  border-radius: 28rpx;
  height: 100rpx;
  font-size: 34rpx;
  font-weight: 600;
  text-align: center;
  border: none;
  box-shadow: 0 8rpx 32rpx rgba(255, 95, 109, 0.35);

  &[disabled] {
    opacity: 0.5;
    box-shadow: none;
  }
}

.action-links {
  display: flex;
  justify-content: center;
  margin-top: 40rpx;

  .link {
    font-size: 28rpx;
    color: var(--dark-page-text-hint);
    padding: 12rpx 24rpx;
  }
}

@media screen and (orientation: landscape) {
  .login-container {
    height: auto;
    min-height: 100vh;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: center;
    gap: 60rpx;
    padding: 40rpx;
  }

  .logo-section {
    padding: 0;
    flex-shrink: 0;

    .logo-icon-wrap {
      width: 100rpx;
      height: 100rpx;
      border-radius: 28rpx;
      margin-bottom: 24rpx;
    }

    .logo-text {
      font-size: 56rpx;
    }

    .sub-title {
      font-size: 26rpx;
      margin-top: 8rpx;
    }
  }

  .form-card {
    width: 50%;
    max-width: 480rpx;
    padding: 40rpx 36rpx;
  }

  .form-item {
    margin-bottom: 20rpx;

    .input-wrap {
      height: 88rpx;
      border-radius: 20rpx;
    }

    .input {
      height: 88rpx;
      font-size: 28rpx;
    }
  }

  .btn-primary {
    margin-top: 32rpx;
    height: 88rpx;
    font-size: 30rpx;
  }

  .action-links {
    margin-top: 24rpx;
  }
}

@media screen and (orientation: landscape) and (max-height: 500px) {
  .login-container {
    gap: 40rpx;
    padding: 24rpx;
  }

  .logo-section {
    .logo-icon-wrap {
      width: 80rpx;
      height: 80rpx;
      border-radius: 20rpx;
      margin-bottom: 16rpx;
    }

    .logo-text {
      font-size: 44rpx;
    }

    .sub-title {
      font-size: 22rpx;
    }
  }

  .form-card {
    padding: 28rpx 24rpx;
  }

  .form-item {
    margin-bottom: 16rpx;

    .input-wrap {
      height: 72rpx;
      border-radius: 16rpx;
    }

    .input {
      height: 72rpx;
      font-size: 26rpx;
    }
  }

  .btn-primary {
    margin-top: 24rpx;
    height: 72rpx;
    font-size: 28rpx;
  }
}
</style>
