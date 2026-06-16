<template>
  <view class="settings-container" :class="themeClass">
    <NavBar title="地图设置" />

    <scroll-view class="settings-scroll" scroll-y :show-scrollbar="false">
      <view class="settings-body">

        <!-- 地图模式设置（非导航状态下） -->
        <view class="menu-section">
          <view class="section-title">
            <text class="section-title-text">地图模式</text>
          </view>
          <view class="section-hint">
            <text class="section-hint-text">非导航状态下的地图显示设置</text>
          </view>
          <view class="menu-card">
            <!-- 地图底图样式 -->
            <view class="sub-section-label">
              <text class="sub-section-text">地图底图样式</text>
            </view>
            <view
              v-for="item in MAP_STYLES"
              :key="item.key"
              class="menu-item"
              @click="onMapModeStyleChange(item.key)"
            >
              <view class="radio-dot" :class="{ active: settings.mapMode.mapStyle === item.key }">
                <view v-if="settings.mapMode.mapStyle === item.key" class="radio-dot-inner"></view>
              </view>
              <view class="radio-info">
                <text class="menu-label">{{ item.label }}</text>
                <text class="radio-desc">{{ item.desc }}</text>
              </view>
              <text v-if="settings.mapMode.mapStyle === item.key" class="check-mark">✓</text>
            </view>
            <view class="menu-divider"></view>
            <view class="menu-item">
              <text class="menu-label">实时路况</text>
              <switch
                :checked="settings.mapMode.trafficEnabled"
                @change="onMapModeTrafficChange"
                color="#3875F6"
              />
            </view>
            <view class="menu-divider"></view>
            <view class="menu-item">
              <text class="menu-label">车辆位置标记</text>
              <switch
                :checked="settings.mapMode.showVehicleMarker"
                @change="onMapModeVehicleMarkerChange"
                color="#3875F6"
              />
            </view>
          </view>
        </view>

        <!-- 导航模式设置（导航状态下） -->
        <view class="menu-section">
          <view class="section-title">
            <text class="section-title-text">导航模式</text>
          </view>
          <view class="section-hint">
            <text class="section-hint-text">导航状态下的地图和UI显示设置</text>
          </view>
          <view class="menu-card">
            <!-- 导航地图底图样式 -->
            <view class="sub-section-label">
              <text class="sub-section-text">地图底图样式</text>
            </view>
            <view
              v-for="item in MAP_STYLES"
              :key="'navi-' + item.key"
              class="menu-item"
              @click="onNaviModeStyleChange(item.key)"
            >
              <view class="radio-dot" :class="{ active: settings.naviMode.mapStyle === item.key }">
                <view v-if="settings.naviMode.mapStyle === item.key" class="radio-dot-inner"></view>
              </view>
              <view class="radio-info">
                <text class="menu-label">{{ item.label }}</text>
                <text class="radio-desc">{{ item.desc }}</text>
              </view>
              <text v-if="settings.naviMode.mapStyle === item.key" class="check-mark">✓</text>
            </view>
            <view class="menu-divider"></view>
            <!-- 导航日夜模式（独立于地图底图） -->
            <view class="sub-section-label">
              <text class="sub-section-text">导航日夜模式</text>
            </view>
            <view
              v-for="item in NAVI_DAY_NIGHT_MODES"
              :key="'dn-' + item.key"
              class="menu-item"
              @click="onNaviDayNightModeChange(item.key)"
            >
              <view class="radio-dot" :class="{ active: settings.naviMode.dayNightMode === item.key }">
                <view v-if="settings.naviMode.dayNightMode === item.key" class="radio-dot-inner"></view>
              </view>
              <view class="radio-info">
                <text class="menu-label">{{ item.label }}</text>
                <text class="radio-desc">{{ item.desc }}</text>
              </view>
              <text v-if="settings.naviMode.dayNightMode === item.key" class="check-mark">✓</text>
            </view>
            <view class="menu-divider"></view>
            <view class="menu-item">
              <text class="menu-label">实时路况</text>
              <switch
                :checked="settings.naviMode.trafficEnabled"
                @change="onNaviModeTrafficChange"
                color="#3875F6"
              />
            </view>
          </view>
        </view>

        <!-- 导航面板组件开关（仅导航模式下生效） -->
        <view class="menu-section">
          <view class="section-title">
            <view class="section-title-row">
              <text class="section-title-text">导航面板组件</text>
              <text class="section-title-action" @click="toggleAllComponents">
                {{ allComponentsOn ? '全部关闭' : '全部开启' }}
              </text>
            </view>
          </view>
          <view class="section-hint">
            <text class="section-hint-text">仅在导航模式下生效</text>
          </view>
          <view class="menu-card">
            <view
              v-for="item in UI_COMPONENTS"
              :key="item.key"
              class="menu-item"
            >
              <view class="switch-info">
                <text class="menu-label">{{ item.label }}</text>
                <text class="switch-desc">{{ item.desc }}</text>
              </view>
              <switch
                :checked="settings.naviMode.uiComponentConfig[item.key]"
                @change="onUIComponentChange(item.key, $event)"
                color="#3875F6"
              />
            </view>
          </view>
        </view>

        <!-- 重置 -->
        <view class="menu-section">
          <view class="menu-card">
            <view class="menu-item center" @click="resetSettings">
              <text class="reset-text">恢复默认设置</text>
            </view>
          </view>
        </view>

      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { computed } from 'vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'
import { useMapSettingsStore, MAP_STYLES, NAVI_DAY_NIGHT_MODES, UI_COMPONENTS } from '@/store/map-settings'

const themeStore = useThemeStore()
const mapSettingsStore = useMapSettingsStore()
const themeClass = computed(() => themeStore.themeClass)
const settings = computed(() => mapSettingsStore.settings)

const allComponentsOn = computed(() => {
  return UI_COMPONENTS.every(c => settings.value.naviMode.uiComponentConfig[c.key])
})

// 地图模式设置
function onMapModeStyleChange(key) {
  mapSettingsStore.setMapModeStyle(key)
}
function onMapModeTrafficChange(e) {
  mapSettingsStore.setMapModeTraffic(e.detail.value)
}
function onMapModeVehicleMarkerChange(e) {
  mapSettingsStore.setMapModeVehicleMarker(e.detail.value)
}

// 导航模式设置
function onNaviModeStyleChange(key) {
  mapSettingsStore.setNaviModeStyle(key)
}
function onNaviDayNightModeChange(key) {
  mapSettingsStore.setNaviDayNightMode(key)
}
function onNaviModeTrafficChange(e) {
  mapSettingsStore.setNaviModeTraffic(e.detail.value)
}

function onUIComponentChange(key, e) {
  mapSettingsStore.setUIComponent(key, e.detail.value)
}

function toggleAllComponents() {
  mapSettingsStore.setAllUIComponent(!allComponentsOn.value)
}

function resetSettings() {
  uni.showModal({
    title: '确认重置',
    content: '确定要恢复地图默认设置吗？',
    success: (res) => {
      if (res.confirm) {
        mapSettingsStore.resetToDefault()
        uni.showToast({ title: '已恢复默认', icon: 'success' })
      }
    }
  })
}
</script>

<style lang="scss" scoped>
.settings-container {
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

.settings-scroll {
  flex: 1;
  height: 0;
  margin-top: calc(var(--status-bar-height) + 88rpx);
}

.settings-body {
  padding: 24rpx 32rpx 60rpx;
}

.menu-section {
  margin-bottom: 28rpx;
}

.section-title {
  padding: 0 8rpx 12rpx;
}

.section-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-title-text {
  font-size: 24rpx;
  font-weight: 500;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 2rpx;
}

.section-hint {
  padding: 0 8rpx 12rpx;
}

.section-hint-text {
  font-size: 22rpx;
  color: var(--text-quaternary);
}

.sub-section-label {
  padding: 16rpx 24rpx 8rpx;
}

.sub-section-text {
  font-size: 22rpx;
  color: var(--text-tertiary);
  font-weight: 500;
}

.section-title-action {
  font-size: 24rpx;
  color: #3875F6;
  font-weight: 500;
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

  &.center {
    justify-content: center;
  }
}

.menu-divider {
  height: 1rpx;
  background: var(--border-divider);
  margin: 0 24rpx 0 24rpx;
}

.menu-label {
  flex: 1;
  font-size: 28rpx;
  font-weight: 500;
  color: var(--text-primary);
  margin-right: 16rpx;
}

.switch-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  margin-right: 16rpx;
}

.switch-desc {
  font-size: 22rpx;
  color: var(--text-tertiary);
}

.radio-dot {
  width: 40rpx;
  height: 40rpx;
  border-radius: 50%;
  border: 3rpx solid var(--border-divider);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16rpx;
  transition: all 0.2s ease;

  &.active {
    border-color: #3875F6;
  }
}

.radio-dot-inner {
  width: 24rpx;
  height: 24rpx;
  border-radius: 50%;
  background: #3875F6;
}

.radio-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  margin-right: 16rpx;
}

.radio-desc {
  font-size: 22rpx;
  color: var(--text-tertiary);
}

.check-mark {
  color: #3875F6;
  font-size: 28rpx;
  font-weight: bold;
}

.reset-text {
  font-size: 28rpx;
  color: #FF6B6B;
  font-weight: 500;
}
</style>
