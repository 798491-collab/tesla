<template>
  <view class="container" :class="themeClass">
    <NavBar title="修改密码" />

    <view class="page-title-wrap">
      <text class="page-title">修改密码</text>
      <text class="page-subtitle">更新您的账户密码</text>
    </view>

    <view class="form-card">
      <view class="form-item">
        <view class="input-wrap">
          <view class="input-icon-wrap">
            <Icon name="LockClosed" :size="20" themeColor="primary" />
          </view>
          <input
            class="input"
            type="password"
            v-model="form.oldPassword"
            placeholder="请输入当前密码"
          />
        </view>
      </view>
      <view class="form-item">
        <view class="input-wrap">
          <view class="input-icon-wrap">
            <Icon name="LockOpen" :size="20" themeColor="primary" />
          </view>
          <input
            class="input"
            type="password"
            v-model="form.newPassword"
            placeholder="请输入新密码（6-20位）"
          />
        </view>
      </view>
      <view class="form-item last">
        <view class="input-wrap">
          <view class="input-icon-wrap">
            <Icon name="Shield" :size="20" themeColor="primary" />
          </view>
          <input
            class="input"
            type="password"
            v-model="form.confirmPassword"
            placeholder="请再次输入新密码"
          />
        </view>
      </view>
    </view>

    <button class="btn-submit" @click="submit" :disabled="loading">
      <text>{{ loading ? '提交中...' : '确认修改' }}</text>
    </button>
  </view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { changePassword as apiChangePassword } from '@/api/user.js'
import { useThemeStore } from '@/store/theme'
import NavBar from '@/components/NavBar/NavBar.vue'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)

const form = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})
const loading = ref(false)

const submit = async () => {
  if (!form.value.oldPassword) {
    uni.showToast({ title: '请输入当前密码', icon: 'none' })
    return
  }
  if (!form.value.newPassword || form.value.newPassword.length < 6) {
    uni.showToast({ title: '新密码至少6位', icon: 'none' })
    return
  }
  if (form.value.newPassword !== form.value.confirmPassword) {
    uni.showToast({ title: '两次输入的密码不一致', icon: 'none' })
    return
  }

  loading.value = true
  try {
    await apiChangePassword({
      old_password: form.value.oldPassword,
      new_password: form.value.newPassword
    })
    uni.showToast({ title: '修改成功', icon: 'success' })
    setTimeout(() => uni.navigateBack(), 1500)
  } catch (err) {
    uni.showToast({ title: err.message || '修改失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}
</script>

<style lang="scss" scoped>
.container {
  min-height: 100vh;
  box-sizing: border-box;
  background: linear-gradient(180deg, var(--dark-page-bg) 0%, var(--bg-card) 100%);
  padding: 0 32rpx;
  padding-top: calc(var(--status-bar-height, 44px) + 88rpx);
}

.page-title-wrap {
  padding: 24rpx 8rpx 40rpx;

  .page-title {
    font-size: 40rpx;
    font-weight: 700;
    color: var(--dark-page-text);
    display: block;
  }

  .page-subtitle {
    font-size: 26rpx;
    color: var(--dark-page-text-hint);
    margin-top: 8rpx;
    display: block;
  }
}

.form-card {
  background: var(--dark-page-icon-wrap-bg);
  border-radius: 24rpx;
  padding: 8rpx 24rpx;
  overflow: hidden;
}

.form-item {
  padding: 20rpx 0;
  border-bottom: 1rpx solid var(--dark-page-glass-border);

  &.last {
    border-bottom: none;
  }

  .input-wrap {
    display: flex;
    align-items: center;
    background: var(--dark-page-glass-bg);
    border-radius: 20rpx;
    height: 96rpx;
    padding: 0 24rpx;
  }

  .input-icon-wrap {
    width: 48rpx;
    height: 48rpx;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 16rpx;
    flex-shrink: 0;
  }

  .input {
    flex: 1;
    font-size: 28rpx;
    color: var(--dark-page-text);
    height: 96rpx;
  }
}

.btn-submit {
  margin-top: 48rpx;
  background: var(--gradient);
  color: #ffffff;
  border-radius: 20rpx;
  height: 96rpx;
  line-height: 96rpx;
  font-size: 30rpx;
  font-weight: 600;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;

  &:disabled {
    opacity: 0.5;
  }
}
</style>
