<template>
  <view class="charging-map-container" :class="themeClass">
    <NavBar title="充电地图" />
    <map
      class="map"
      :latitude="centerLat"
      :longitude="centerLng"
      :markers="markers"
      :scale="11"
    ></map>

    <view class="list-card">
      <view class="list-header">
        <view class="header-left">
          <view class="header-icon">
            <Icon name="Map" :size="18" color="#fff" />
          </view>
          <text class="list-title">充电位置</text>
        </view>
        <view class="header-count">
          <text class="count-text">{{ logs.length }} 条记录</text>
        </view>
      </view>
      <scroll-view scroll-y class="log-scroll">
        <view class="log-item" v-for="log in logs" :key="log.id" @click="focusLocation(log)">
          <view class="log-type-badge" :class="log.charge_type?.toLowerCase()">
            <Icon :name="log.charge_type === 'DC' ? 'Flash' : 'BatteryCharging'" :size="16" color="#fff" />
          </view>
          <view class="log-center">
            <text class="log-address">{{ log.address || log.location || '未知位置' }}</text>
            <view class="log-meta">
              <Icon name="Time" :size="12" color="#bfbfbf" />
              <text class="log-time">{{ formatDate(log.start_time) }}</text>
            </view>
          </view>
          <view class="log-right">
            <text class="log-kwh">{{ log.charge_kwh?.toFixed(1) }}</text>
            <text class="log-unit">kWh</text>
          </view>
        </view>
        <view class="empty-tip" v-if="logs.length === 0">
          <Icon name="FlashOff" :size="48" color="#d9d9d9" />
          <text class="empty-text">暂无充电记录</text>
        </view>
      </scroll-view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getChargingLogs } from '@/api/charging.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)

const vin = ref('')
const logs = ref([])
const focusLat = ref(39.9042)
const focusLng = ref(116.4074)

const centerLat = computed(() => focusLat.value)
const centerLng = computed(() => focusLng.value)

const markers = computed(() => {
  return logs.value.filter(log => log.latitude && log.longitude).map((log, index) => ({
    id: log.id || index + 1,
    latitude: log.latitude,
    longitude: log.longitude,
    title: log.address || log.location || '充电位置',
    iconPath: '/static/charging-marker.png',
    width: 30,
    height: 30,
    callout: {
      content: log.charge_type === 'DC' ? '⚡快充' : '🔌慢充',
      color: '#ffffff',
      fontSize: 12,
      borderRadius: 6,
      bgColor: log.charge_type === 'DC' ? themeStore.colors.primary : '#52c41a',
      padding: 6,
      display: 'ALWAYS'
    }
  }))
})

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  vin.value = currentPage.$page?.options?.vin || currentPage.options?.vin || ''
  if (vin.value) {
    loadData()
  }
})

const loadData = () => {
  getChargingLogs(vin.value).then((res) => {
    logs.value = res.data || []
    if (logs.value.length > 0 && logs.value[0].latitude) {
      focusLat.value = logs.value[0].latitude
      focusLng.value = logs.value[0].longitude
    }
  })
}

const focusLocation = (log) => {
  if (log.latitude && log.longitude) {
    focusLat.value = log.latitude
    focusLng.value = log.longitude
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}
</script>

<style lang="scss" scoped>
.charging-map-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-page);
  padding-top: calc(var(--status-bar-height, 44px) + 88rpx);
  box-sizing: border-box;
}

.map {
  height: calc(55vh - 88rpx);
  width: 100%;
}

.list-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: 28rpx 28rpx 0 0;
  margin-top: -24rpx;
  position: relative;
  box-shadow: var(--shadow-card);
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 28rpx 32rpx 16rpx;

  .header-left {
    display: flex;
    align-items: center;
    gap: 14rpx;

    .header-icon {
      width: 44rpx;
      height: 44rpx;
      border-radius: 12rpx;
      background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .list-title {
      font-size: 32rpx;
      font-weight: 700;
      color: var(--text-primary);
    }
  }

  .header-count {
    .count-text {
      font-size: 24rpx;
      color: var(--text-tertiary);
      padding: 6rpx 16rpx;
      background: var(--bg-card-secondary);
      border-radius: 12rpx;
    }
  }
}

.log-scroll {
  flex: 1;
  padding: 0 32rpx 32rpx;
}

.log-item {
  display: flex;
  align-items: center;
  padding: 20rpx 0;
  border-bottom: 1rpx solid var(--border-divider);

  &:last-child {
    border-bottom: none;
  }

  &:active {
    opacity: 0.7;
  }
}

.log-type-badge {
  width: 56rpx;
  height: 56rpx;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 20rpx;
  flex-shrink: 0;

  &.dc {
    background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  }

  &.ac {
    background: linear-gradient(135deg, #52c41a, #73d13d);
  }
}

.log-center {
  flex: 1;
  min-width: 0;

  .log-address {
    font-size: 28rpx;
    color: var(--text-primary);
    font-weight: 500;
    display: block;
    margin-bottom: 8rpx;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .log-meta {
    display: flex;
    align-items: center;
    gap: 6rpx;

    .log-time {
      font-size: 22rpx;
      color: var(--text-placeholder);
    }
  }
}

.log-right {
  margin-left: 16rpx;
  text-align: right;
  flex-shrink: 0;

  .log-kwh {
    font-size: 32rpx;
    font-weight: 800;
    color: var(--color-primary);
    display: block;
  }

  .log-unit {
    font-size: 20rpx;
    color: var(--text-placeholder);
    display: block;
  }
}

.empty-tip {
  text-align: center;
  padding: 80rpx 40rpx;

  .empty-text {
    font-size: 26rpx;
    color: var(--text-placeholder);
    margin-top: 16rpx;
    display: block;
  }
}
</style>
