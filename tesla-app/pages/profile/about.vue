<template>
  <view class="about-container" :class="themeClass">
    <NavBar title="关于" />

    <scroll-view class="about-scroll" scroll-y :show-scrollbar="false">
      <view class="about-body">
        <!-- 应用图标和名称 -->
        <view class="app-header">
          <view class="app-logo">
            <text class="logo-text">T</text>
          </view>
          <text class="app-name">Tesla 管理平台</text>
          <text class="app-version">v1.2.0</text>
        </view>

        <!-- 应用描述 -->
        <view class="desc-card">
          <text class="desc-text">
            Tesla 管理平台是一款基于 Tesla Fleet API 与 VCP 协议的个人车辆管理工具。提供车辆状态监控、远程控制、充电管理、行驶记录分析等功能，让您随时随地掌控爱车动态。
          </text>
        </view>

        <!-- 链接列表 -->
        <view class="links-card">
          <view class="link-item" @click="openLink('用户协议')">
            <Icon name="DocumentText" :size="22" themeColor="primary" />
            <text class="link-label">用户协议</text>
            <Icon name="ChevronForward" :size="16" themeColor="chevron" />
          </view>
          <view class="link-divider"></view>
          <view class="link-item" @click="openLink('隐私政策')">
            <Icon name="ShieldCheckmark" :size="22" themeColor="primary" />
            <text class="link-label">隐私政策</text>
            <Icon name="ChevronForward" :size="16" themeColor="chevron" />
          </view>
          <view class="link-divider"></view>
          <view class="link-item" @click="openLink('开源许可')">
            <Icon name="CodeSlash" :size="22" themeColor="primary" />
            <text class="link-label">开源许可</text>
            <Icon name="ChevronForward" :size="16" themeColor="chevron" />
          </view>
        </view>

        <!-- 版权信息 -->
        <view class="footer">
          <text class="footer-text">© 2024-2026 Tesla 管理平台</text>
          <text class="footer-sub">Based on Tesla Fleet API + VCP</text>
        </view>
      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { computed } from 'vue'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)

const openLink = (name) => {
  uni.showModal({
    title: name,
    content: `${name}页面暂未上线，敬请期待`,
    showCancel: false
  })
}
</script>

<style lang="scss" scoped>
.about-container {
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

.about-scroll {
  flex: 1;
  height: 0;
  margin-top: calc(var(--status-bar-height) + 88rpx);
}

.about-body {
  padding: 32rpx 32rpx 0;
  display: flex;
  flex-direction: column;
  align-items: center;
}

/* 应用头部 */
.app-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40rpx 0 48rpx;
}

.app-logo {
  width: 140rpx;
  height: 140rpx;
  border-radius: 32rpx;
  background: var(--gradient);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 24rpx;
  box-shadow: var(--shadow-card);
}

.logo-text {
  font-size: 72rpx;
  color: #ffffff;
  font-weight: 800;
}

.app-name {
  font-size: 36rpx;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 8rpx;
}

.app-version {
  font-size: 24rpx;
  color: var(--text-tertiary);
  padding: 4rpx 20rpx;
  background: var(--bg-icon-wrap);
  border-radius: 20rpx;
}

/* 描述卡片 */
.desc-card {
  width: 100%;
  background: var(--bg-card);
  border-radius: 24rpx;
  border: 1rpx solid var(--border-card);
  box-shadow: var(--shadow-card);
  padding: 28rpx;
  margin-bottom: 28rpx;
}

.desc-text {
  font-size: 26rpx;
  line-height: 44rpx;
  color: var(--text-secondary);
}

/* 链接列表 */
.links-card {
  width: 100%;
  background: var(--bg-card);
  border-radius: 24rpx;
  border: 1rpx solid var(--border-card);
  box-shadow: var(--shadow-card);
  overflow: hidden;
  margin-bottom: 40rpx;
}

.link-item {
  display: flex;
  align-items: center;
  padding: 28rpx 24rpx;
  transition: background 0.15s ease;

  &:active {
    background: var(--bg-card-hover);
  }
}

.link-label {
  flex: 1;
  font-size: 28rpx;
  font-weight: 500;
  color: var(--text-primary);
  margin-left: 20rpx;
}

.link-divider {
  height: 1rpx;
  background: var(--border-divider);
  margin: 0 24rpx 0 68rpx;
}

/* 版权信息 */
.footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8rpx;
  padding: 20rpx 0 60rpx;
}

.footer-text {
  font-size: 22rpx;
  color: var(--text-tertiary);
}

.footer-sub {
  font-size: 20rpx;
  color: var(--text-placeholder);
}
</style>
