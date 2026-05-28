<template>
  <view class="profile-container" :class="themeClass">
    <NavBar title="我的" :showBack="false" />

    <scroll-view class="profile-scroll" scroll-y :show-scrollbar="false">
      <view class="profile-body">
        <!-- 用户信息卡片 -->
        <view class="user-card">
          <view class="avatar">
            <text class="avatar-text">{{ userInfo.nickname?.[0] || 'U' }}</text>
          </view>
          <view class="user-info">
            <text class="nickname">{{ userInfo.nickname || '用户' }}</text>
            <text class="username">@{{ userInfo.username || '' }}</text>
          </view>
          <view class="edit-btn" @click="editProfile">
            <Icon name="Settings" :size="18" themeColor="primary" />
          </view>
        </view>

        <!-- 功能菜单 -->
        <view class="menu-section">
          <view class="section-title">
            <text class="section-title-text">账号设置</text>
          </view>
          <view class="menu-card">
            <view class="menu-item" @click="editProfile">
              <view class="menu-icon-wrap">
                <Icon name="Person" :size="22" themeColor="primary" />
              </view>
              <text class="menu-label">编辑资料</text>
              <text class="menu-value">修改个人信息</text>
              <Icon name="ChevronForward" :size="16" themeColor="chevron" />
            </view>
            <view class="menu-divider"></view>
            <view class="menu-item" @click="changePassword">
              <view class="menu-icon-wrap">
                <Icon name="LockClosed" :size="22" themeColor="primary" />
              </view>
              <text class="menu-label">修改密码</text>
              <text class="menu-value">更新登录密码</text>
              <Icon name="ChevronForward" :size="16" themeColor="chevron" />
            </view>
          </view>
        </view>

        <view class="menu-section">
          <view class="section-title">
            <text class="section-title-text">车辆工具</text>
          </view>
          <view class="menu-card">
            <view class="menu-item" @click="goDashcam">
              <view class="menu-icon-wrap dashcam-icon">
                <Icon name="Videocam" :size="22" color="#5BE7C4" />
              </view>
              <text class="menu-label">行车记录仪</text>
              <text class="menu-value">本地视频+GPS融合</text>
              <Icon name="ChevronForward" :size="16" themeColor="chevron" />
            </view>
          </view>
        </view>

        <view class="menu-section">
          <view class="section-title">
            <text class="section-title-text">偏好设置</text>
          </view>
          <view class="menu-card">
            <view class="menu-item" @click="showThemePicker">
              <view class="menu-icon-wrap">
                <Icon name="Settings" :size="22" themeColor="primary" />
              </view>
              <text class="menu-label">主题模式</text>
              <text class="menu-value">{{ themeModeLabel }}</text>
              <Icon name="ChevronForward" :size="16" themeColor="chevron" />
            </view>
          </view>
        </view>

        <view class="menu-section">
          <view class="section-title">
            <text class="section-title-text">其他</text>
          </view>
          <view class="menu-card">
            <view class="menu-item" @click="goAbout">
              <view class="menu-icon-wrap">
                <Icon name="InformationCircle" :size="22" themeColor="primary" />
              </view>
              <text class="menu-label">关于</text>
              <text class="menu-value">v1.2.0</text>
              <Icon name="ChevronForward" :size="16" themeColor="chevron" />
            </view>
            <view class="menu-divider"></view>
            <view class="menu-item" @click="bleDebug">
              <view class="menu-icon-wrap">
                <Icon name="Bluetooth" :size="22" themeColor="primary" />
              </view>
              <text class="menu-label">BLE 调试</text>
              <text class="menu-value">蓝牙连接调试</text>
              <Icon name="ChevronForward" :size="16" themeColor="chevron" />
            </view>
          </view>
        </view>

        <!-- 退出登录 -->
        <view class="logout-wrap">
          <button class="btn-logout" @click="logout">退出登录</button>
        </view>
      </view>

      <view class="tabbar-spacer"></view>
    </scroll-view>

    <TabBar :currentIndex="3" />
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getUserInfo } from '@/api/user.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'
import TabBar from '@/components/TabBar/TabBar.vue'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)

const userInfo = ref({})

onMounted(() => {
  loadUserInfo()
})

const loadUserInfo = () => {
  getUserInfo().then((res) => {
    userInfo.value = res.data || {}
  }).catch(() => {
    const localUser = uni.getStorageSync('userInfo')
    if (localUser) {
      try {
        userInfo.value = typeof localUser === 'string' ? JSON.parse(localUser) : localUser
      } catch (e) {
        userInfo.value = {}
      }
    }
  })
}

const themeModeLabel = computed(() => {
  const mode = themeStore.themeMode
  if (mode === 'light') return '浅色'
  if (mode === 'dark') return '深色'
  if (mode === 'visionpro') return 'Vision Pro'
  return '跟随系统'
})

const showThemePicker = () => {
  uni.showActionSheet({
    itemList: ['跟随系统', '浅色模式', '深色模式', 'Vision Pro'],
    success: (res) => {
      const modes = ['system', 'light', 'dark', 'visionpro']
      themeStore.setThemeMode(modes[res.tapIndex])
    }
  })
}

const goAbout = () => {
  uni.navigateTo({ url: '/pages/profile/about' })
}

const editProfile = () => {
  uni.navigateTo({ url: '/pages/profile/edit' })
}

const changePassword = () => {
  uni.navigateTo({ url: '/pages/profile/change-password' })
}

const bleDebug = () => {
  uni.navigateTo({ url: '/pages/debug/ble' })
}

const goDashcam = () => {
  uni.navigateTo({ url: '/pages/dashcam/index' })
}

const logout = () => {
  uni.showModal({
    title: '确认退出',
    content: '确定要退出登录吗？',
    success: (res) => {
      if (res.confirm) {
        uni.removeStorageSync('token')
        uni.removeStorageSync('userInfo')
        uni.reLaunch({ url: '/pages/index/index' })
      }
    }
  })
}
</script>

<style lang="scss" scoped>
.profile-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--bg-page-solid, var(--bg-page));
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.profile-scroll {
  flex: 1;
  height: 0;
  margin-top: calc(var(--status-bar-height) + 88rpx);
}

.profile-body {
  padding: 24rpx 32rpx 0;
}

.tabbar-spacer {
  height: 140rpx;
}

/* 用户信息卡片 */
.user-card {
  display: flex;
  align-items: center;
  padding: 36rpx 28rpx;
  background: var(--bg-card);
  border-radius: 24rpx;
  border: 1rpx solid var(--border-card);
  box-shadow: var(--shadow-card);
  margin-bottom: 32rpx;
}

.avatar {
  width: 100rpx;
  height: 100rpx;
  border-radius: 50%;
  background: var(--gradient);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.avatar-text {
  font-size: 42rpx;
  color: #ffffff;
  font-weight: 700;
}

.user-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  margin-left: 24rpx;
  min-width: 0;
}

.nickname {
  font-size: 34rpx;
  font-weight: 700;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.username {
  font-size: 24rpx;
  color: var(--text-tertiary);
}

.edit-btn {
  width: 64rpx;
  height: 64rpx;
  border-radius: 50%;
  background: var(--bg-icon-wrap);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  &:active {
    opacity: 0.7;
  }
}

/* 菜单分组 */
.menu-section {
  margin-bottom: 28rpx;
}

.section-title {
  padding: 0 8rpx 16rpx;
}

.section-title-text {
  font-size: 24rpx;
  font-weight: 500;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 2rpx;
}

.menu-card {
  background: var(--bg-card);
  border-radius: 24rpx;
  border: 1rpx solid var(--border-card);
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 28rpx 24rpx;
  transition: background 0.15s ease;

  &:active {
    background: var(--bg-card-hover);
  }
}

.menu-icon-wrap {
  width: 56rpx;
  height: 56rpx;
  border-radius: 14rpx;
  background: var(--bg-icon-wrap);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  &.dashcam-icon {
    background: rgba(91, 231, 196, 0.12);
  }
}

.menu-label {
  flex: 1;
  font-size: 28rpx;
  font-weight: 500;
  color: var(--text-primary);
  margin-left: 20rpx;
}

.menu-value {
  font-size: 24rpx;
  color: var(--text-tertiary);
  margin-right: 8rpx;
}

.menu-divider {
  height: 1rpx;
  background: var(--border-divider);
  margin: 0 24rpx 0 100rpx;
}

/* 退出登录 */
.logout-wrap {
  padding: 16rpx 0 0;
}

.btn-logout {
  width: 100%;
  height: 88rpx;
  background: var(--bg-card);
  border: 1rpx solid var(--border-card);
  border-radius: 24rpx;
  font-size: 28rpx;
  font-weight: 500;
  color: #FF6B6B;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--shadow-card);

  &:active {
    opacity: 0.7;
  }
}
</style>
