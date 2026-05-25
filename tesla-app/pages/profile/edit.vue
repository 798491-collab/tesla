<template>
  <view class="container" :class="themeClass">
    <NavBar title="编辑资料" />

    <view class="page-title-wrap">
      <text class="page-title">编辑资料</text>
      <text class="page-subtitle">修改您的个人信息</text>
    </view>

    <view class="form-card">
      <view class="form-item">
        <view class="input-wrap">
          <view class="input-icon-wrap">
            <Icon name="Person" :size="20" themeColor="primary" />
          </view>
          <input
            class="input"
            v-model="form.nickname"
            placeholder="请输入昵称"
            maxlength="20"
          />
        </view>
      </view>
      <view class="form-item last">
        <view class="input-wrap">
          <view class="input-icon-wrap">
            <Icon name="Desktop" :size="20" themeColor="primary" />
          </view>
          <input
            class="input"
            v-model="form.phone"
            placeholder="请输入手机号"
            maxlength="11"
            type="number"
          />
        </view>
      </view>
    </view>

    <button class="btn-submit" @click="submit" :disabled="loading">
      <text>{{ loading ? '保存中...' : '保存修改' }}</text>
    </button>
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getUserInfo, updateUserInfo } from '@/api/user.js'
import { useThemeStore } from '@/store/theme'
import NavBar from '@/components/NavBar/NavBar.vue'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)

const form = ref({
  nickname: '',
  phone: ''
})
const loading = ref(false)

onMounted(() => {
  loadUserInfo()
})

const loadUserInfo = () => {
  getUserInfo().then((res) => {
    const data = res.data || {}
    form.value.nickname = data.nickname || ''
    form.value.phone = data.phone || ''
  })
}

const submit = async () => {
  if (!form.value.nickname.trim()) {
    uni.showToast({ title: '请输入昵称', icon: 'none' })
    return
  }

  loading.value = true
  try {
    await updateUserInfo({
      nickname: form.value.nickname.trim(),
      phone: form.value.phone.trim()
    })
    uni.showToast({ title: '保存成功', icon: 'success' })
    setTimeout(() => uni.navigateBack(), 1500)
  } catch (err) {
    uni.showToast({ title: err.message || '保存失败', icon: 'none' })
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
