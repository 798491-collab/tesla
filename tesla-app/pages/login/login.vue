<template>
  <view class="login-page" :class="themeClass">
    <view class="bg-layer">
      <view class="bg-circle bg-circle-1"></view>
      <view class="bg-circle bg-circle-2"></view>
    </view>

    <view class="content">
      <view class="logo-area">
        <view class="logo-icon-wrap">
          <Icon name="CarSport" :size="56" themeColor="primary" />
        </view>
        <text class="logo-title">Tesla 管理平台</text>
        <text class="logo-subtitle">智能车辆管理 · 尽在掌控</text>
      </view>

      <view class="glass-card">
        <view class="input-item">
          <view class="input-icon">
            <Icon name="Person" :size="20" themeColor="hint" />
          </view>
          <input
            class="input-field"
            v-model="username"
            placeholder="请输入用户名"
            placeholder-class="input-placeholder"
          />
        </view>

        <view class="input-item">
          <view class="input-icon">
            <Icon name="LockClosed" :size="20" themeColor="hint" />
          </view>
          <input
            class="input-field"
            v-model="password"
            :password="!showPassword"
            placeholder="请输入密码"
            placeholder-class="input-placeholder"
          />
          <view class="input-toggle" @tap="togglePassword">
            <Icon :name="showPassword ? 'EyeOff' : 'Eye'" :size="20" themeColor="hint" />
          </view>
        </view>

        <view class="agreement-row" @tap="agreed = !agreed">
          <view class="checkbox-wrap" :class="{ checked: agreed }">
            <Icon v-if="agreed" name="Checkmark" :size="14" color="#fff" />
          </view>
          <text class="agreement-text">
            我已阅读并同意<text class="agreement-link" @tap.stop="goToDoc('user-agreement')">《用户协议》</text>和<text class="agreement-link" @tap.stop="goToDoc('privacy-policy')">《隐私政策》</text>
          </text>
        </view>

        <view class="btn-login" :class="{ 'btn-loading': loading }" @tap="login">
          <text class="btn-text">{{ loading ? '登录中...' : '登 录' }}</text>
        </view>
      </view>

      <view class="bottom-link" @tap="goToRegister">
        <text class="link-text">还没有账号？</text>
        <text class="link-action">立即注册</text>
      </view>
    </view>

    <!-- 用户协议弹窗 -->
    <view v-if="showAgreeModal" class="modal-mask" @tap="showAgreeModal = false">
      <view class="modal-content" @tap.stop>
        <text class="modal-title">用户协议及隐私政策</text>
        <view class="modal-body">
          <text class="modal-text">请仔细阅读并同意以下条款：<text class="agreement-link" @tap="goToDoc('user-agreement')">《用户协议》</text>和<text class="agreement-link" @tap="goToDoc('privacy-policy')">《隐私政策》</text>，点击「同意并登录」即表示您已阅读并同意全部条款。</text>
        </view>
        <view class="modal-actions">
          <view class="modal-btn modal-btn-cancel" @tap="showAgreeModal = false">
            <text class="modal-btn-text-cancel">不同意</text>
          </view>
          <view class="modal-btn modal-btn-confirm" @tap="agreeAndLogin">
            <text class="modal-btn-text-confirm">同意并登录</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useUserStore } from '@/store/user'
import { useVehicleStore } from '@/store/vehicle'
import { useThemeStore } from '@/store/theme'

const userStore = useUserStore()
const vehicleStore = useVehicleStore()
const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)
const hintColor = computed(() => themeStore.colors.hint)
const username = ref(userStore.savedUsername || '')
const password = ref('')
const showPassword = ref(false)
const loading = ref(false)
const agreed = ref(false)
const showAgreeModal = ref(false)

const togglePassword = () => {
  showPassword.value = !showPassword.value
}

const goToDoc = (type) => {
  uni.navigateTo({ url: `/pages/doc/doc?type=${type}` })
}

const doLogin = async () => {
  loading.value = true
  try {
    await userStore.loginAction({ username: username.value, password: password.value })
    await vehicleStore.fetchVehicles()
    uni.reLaunch({ url: '/pages/dashboard/dashboard' })
  } catch (e) {
    uni.showToast({ title: e.message || '登录失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

const login = async () => {
  if (!username.value || !password.value) {
    uni.showToast({ title: '请输入用户名和密码', icon: 'none' })
    return
  }
  if (!agreed.value) {
    showAgreeModal.value = true
    return
  }
  await doLogin()
}

const agreeAndLogin = async () => {
  agreed.value = true
  showAgreeModal.value = false
  await doLogin()
}

const goToRegister = () => {
  uni.navigateTo({ url: '/pages/register/register' })
}
</script>

<style lang="scss" scoped>
.login-page {
  position: relative;
  height: 100vh;
  overflow: hidden;
  box-sizing: border-box;
  background: linear-gradient(180deg, var(--dark-page-bg) 0%, var(--bg-card) 100%);
}

.bg-layer {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
}

.bg-circle {
  position: absolute;
  border-radius: 50%;
  opacity: 0.06;
  background: var(--color-primary);
}

.bg-circle-1 {
  width: 600rpx;
  height: 600rpx;
  top: -200rpx;
  right: -150rpx;
}

.bg-circle-2 {
  width: 400rpx;
  height: 400rpx;
  bottom: -100rpx;
  left: -100rpx;
}

.content {
  position: relative;
  z-index: 1;
  padding: 0 48rpx;
  padding-top: 180rpx;
}

.logo-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 80rpx;
}

.logo-icon-wrap {
  width: 120rpx;
  height: 120rpx;
  border-radius: 32rpx;
  background: var(--bg-icon-wrap);
  border: 1rpx solid var(--dark-page-glass-border);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 32rpx;
}

.logo-title {
  font-size: 44rpx;
  font-weight: 700;
  color: var(--dark-page-text);
  letter-spacing: 4rpx;
  margin-bottom: 12rpx;
}

.logo-subtitle {
  font-size: 26rpx;
  color: var(--dark-page-text-hint);
  letter-spacing: 2rpx;
}

.glass-card {
  background: var(--dark-page-glass-bg);
  border: 1rpx solid var(--dark-page-glass-border);
  border-radius: 32rpx;
  padding: 48rpx 40rpx;
}

.input-item {
  display: flex;
  align-items: center;
  height: 100rpx;
  background: var(--dark-page-glass-bg);
  border: 1rpx solid var(--dark-page-glass-border);
  border-radius: 20rpx;
  padding: 0 28rpx;
  margin-bottom: 28rpx;

  &:focus-within {
    border-color: var(--color-primary);
    background: var(--dark-page-glass-bg);
  }
}

.input-icon {
  width: 44rpx;
  height: 44rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 20rpx;
  flex-shrink: 0;
}

.input-field {
  flex: 1;
  height: 100rpx;
  font-size: 30rpx;
  color: var(--dark-page-text);
  background: transparent;
}

.input-placeholder {
  color: var(--dark-page-text-hint);
  font-size: 30rpx;
}

.input-toggle {
  width: 48rpx;
  height: 48rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.btn-login {
  margin-top: 20rpx;
  height: 100rpx;
  border-radius: 20rpx;
  background: var(--gradient);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8rpx 32rpx rgba(255, 95, 109, 0.3);

  &:active {
    opacity: 0.9;
  }
}

.btn-loading {
  opacity: 0.7;
  pointer-events: none;
}

.btn-text {
  font-size: 32rpx;
  font-weight: 600;
  color: var(--dark-page-text);
  letter-spacing: 4rpx;
}

.bottom-link {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 48rpx;
  padding: 20rpx 0;
}

.link-text {
  font-size: 28rpx;
  color: var(--dark-page-text-hint);
}

.link-action {
  font-size: 28rpx;
  color: var(--color-primary);
  margin-left: 8rpx;
  font-weight: 500;
}

.agreement-row {
  display: flex;
  align-items: flex-start;
  margin-top: 24rpx;
  margin-bottom: 8rpx;
}

.checkbox-wrap {
  width: 36rpx;
  height: 36rpx;
  border-radius: 8rpx;
  border: 2rpx solid var(--dark-page-glass-border);
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12rpx;
  margin-top: 4rpx;
  flex-shrink: 0;
  transition: all 0.2s;

  &.checked {
    background: var(--color-primary);
    border-color: var(--color-primary);
  }
}

.agreement-text {
  font-size: 24rpx;
  color: var(--dark-page-text-hint);
  line-height: 1.6;
}

.agreement-link {
  color: var(--color-primary);
}

.modal-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-content {
  width: 600rpx;
  background: var(--modal-bg);
  border: 1rpx solid rgba(255, 255, 255, 0.1);
  border-radius: 28rpx;
  padding: 48rpx 40rpx 36rpx;
}

.modal-title {
  font-size: 34rpx;
  font-weight: 700;
  color: var(--dark-page-text);
  text-align: center;
  margin-bottom: 32rpx;
}

.modal-body {
  margin-bottom: 40rpx;
}

.modal-text {
  font-size: 28rpx;
  color: var(--dark-page-text-hint);
  line-height: 1.7;
}

.modal-actions {
  display: flex;
  gap: 24rpx;
}

.modal-btn {
  flex: 1;
  height: 84rpx;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;

  &:active {
    opacity: 0.9;
  }
}

.modal-btn-cancel {
  background: var(--dark-page-glass-bg);
  border: 1rpx solid var(--dark-page-glass-border);
}

.modal-btn-confirm {
  background: var(--gradient);
  box-shadow: 0 6rpx 24rpx rgba(255, 95, 109, 0.3);
}

.modal-btn-text-cancel {
  font-size: 28rpx;
  color: var(--dark-page-text-hint);
  font-weight: 500;
}

.modal-btn-text-confirm {
  font-size: 28rpx;
  color: #fff;
  font-weight: 600;
}
</style>
