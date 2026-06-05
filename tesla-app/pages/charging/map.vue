<template>
  <view class="charging-map-container" :class="themeClass">
    <NavBar title="充电地图" />
    <map
      id="chargingMap"
      class="map"
      :latitude="centerLat"
      :longitude="centerLng"
      :markers="markers"
      :scale="mapScale"
      :enable-3D="true"
      :show-compass="true"
      :enable-zoom="true"
      :enable-scroll="true"
      :enable-rotate="true"
      :subkey="tencentMapKey"
      @markertap="onMarkerTap"
      @updated="onMapUpdated"
    ></map>

    <view class="list-card">
      <view class="list-header">
        <view class="header-left">
          <view class="header-icon">
            <Icon name="Flash" :size="18" color="#fff" />
          </view>
          <text class="list-title">充电位置</text>
        </view>
        <view class="header-count">
          <text class="count-text">{{ groupedLocations.length }} 个位置 · {{ logs.length }} 次</text>
        </view>
      </view>
      <scroll-view scroll-y class="log-scroll">
        <view class="log-item" v-for="group in groupedLocations" :key="group.key" @click="focusLocation(group)">
          <view class="log-type-badge" :class="{ 'dc': group.isDC, 'ac': !group.isDC }">
            <Icon :name="group.isDC ? 'Flash' : 'BatteryCharging'" :size="16" color="#fff" />
            <text class="badge-count" v-if="group.count > 1">{{ group.count }}</text>
          </view>
          <view class="log-center">
            <view class="log-top-row">
              <text class="log-address">{{ group.address }}</text>
              <text class="log-kwh">{{ group.totalKwh.toFixed(1) }}<text class="log-unit"> kWh</text></text>
            </view>
            <view class="log-meta">
              <Icon name="Time" :size="12" color="#bfbfbf" />
              <text class="log-time">{{ formatDate(group.latestTime) }}</text>
              <text class="log-count-tag" v-if="group.count > 1">共{{ group.count }}次</text>
            </view>
          </view>
        </view>
        <view class="empty-tip" v-if="groupedLocations.length === 0">
          <Icon name="FlashOff" :size="48" color="#d9d9d9" />
          <text class="empty-text">暂无充电记录</text>
        </view>
      </scroll-view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted, getCurrentInstance } from 'vue'
import { getChargingLogs } from '@/api/charging.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const tencentMapKey = import.meta.env.VITE_TENCENT_MAP_KEY || ''
const darkStyleId = import.meta.env.VITE_TENCENT_MAP_STYLE_DARK || '2'
let isStyleSet = false

const vin = ref('')
const logs = ref([])
const focusLat = ref(39.9042)
const focusLng = ref(116.4074)
const mapScale = ref(11)

const centerLat = computed(() => focusLat.value)
const centerLng = computed(() => focusLng.value)

const groupedLocations = computed(() => {
  const locationMap = {}
  logs.value.forEach(log => {
    if (!log.latitude || !log.longitude) return
    const key = `${log.latitude.toFixed(4)}_${log.longitude.toFixed(4)}`
    if (!locationMap[key]) {
      locationMap[key] = {
        key,
        latitude: log.latitude,
        longitude: log.longitude,
        address: log.address || log.location || '未知位置',
        count: 0,
        totalKwh: 0,
        isDC: false,
        latestTime: log.start_time,
        logIds: []
      }
    }
    locationMap[key].count++
    locationMap[key].totalKwh += log.charge_kwh || 0
    locationMap[key].logIds.push(log.id)
    if (log.charge_type === 'DC') locationMap[key].isDC = true
    if (log.start_time && (!locationMap[key].latestTime || new Date(log.start_time) > new Date(locationMap[key].latestTime))) {
      locationMap[key].latestTime = log.start_time
    }
  })
  return Object.values(locationMap).sort((a, b) => new Date(b.latestTime) - new Date(a.latestTime))
})

const markers = computed(() => {
  return groupedLocations.value.map((loc, idx) => {
    const icon = loc.isDC ? '⚡' : '🔌'
    const label = loc.count > 1 ? `${icon}${loc.count}` : icon
    return {
      id: idx + 1,
      latitude: loc.latitude,
      longitude: loc.longitude,
      title: loc.address,
      iconPath: '/static/marker-transparent.png',
      width: 1,
      height: 1,
      callout: {
        content: label,
        color: '#ffffff',
        fontSize: 14,
        borderRadius: 20,
        bgColor: loc.isDC ? '#2563EB' : '#389e0d',
        padding: 8,
        display: 'ALWAYS',
        textAlign: 'center'
      }
    }
  })
})

const onMarkerTap = (e) => {
  const markerId = (e.markerId || e.detail?.markerId) - 1
  if (markerId >= 0 && markerId < groupedLocations.value.length) {
    const loc = groupedLocations.value[markerId]
    focusLocation(loc)
  }
}

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  vin.value = currentPage.$page?.options?.vin || currentPage.options?.vin || ''
  if (vin.value) {
    loadData()
  }
  setTimeout(() => applyMapDarkStyle(), 500)
})

const onMapUpdated = () => {
  applyMapDarkStyle()
}

const applyMapDarkStyle = () => {
  if (isStyleSet) return
  if (themeStore.themeClass?.includes('dark')) {
    const mapCtx = uni.createMapContext('chargingMap', getCurrentInstance())
    mapCtx.setMapStyle({ styleId: darkStyleId })
    isStyleSet = true
  }
}

const loadData = () => {
  getChargingLogs(vin.value).then((res) => {
    logs.value = res.data || []
    const withCoords = logs.value.filter(l => l.latitude && l.longitude)
    if (withCoords.length > 0) {
      focusLat.value = withCoords[0].latitude
      focusLng.value = withCoords[0].longitude
    }
  })
}

const focusLocation = (loc) => {
  if (loc.latitude && loc.longitude) {
    focusLat.value = loc.latitude
    focusLng.value = loc.longitude
    mapScale.value = 14
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
  overflow: hidden;
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 28rpx 32rpx 16rpx;
  flex-shrink: 0;

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
      font-size: 22rpx;
      color: var(--text-tertiary);
      padding: 6rpx 14rpx;
      background: var(--bg-card-secondary);
      border-radius: 12rpx;
    }
  }
}

.log-scroll {
  flex: 1;
  padding: 0 32rpx 32rpx;
  overflow: hidden;
  box-sizing: border-box;
}

.log-item {
  display: flex;
  align-items: center;
  padding: 20rpx 0;
  border-bottom: 1rpx solid var(--border-divider);
  width: 100%;
  box-sizing: border-box;

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
  margin-right: 16rpx;
  flex-shrink: 0;
  position: relative;

  &.dc {
    background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  }

  &.ac {
    background: linear-gradient(135deg, #52c41a, #73d13d);
  }

  .badge-count {
    position: absolute;
    top: -8rpx;
    right: -8rpx;
    min-width: 28rpx;
    height: 28rpx;
    line-height: 28rpx;
    text-align: center;
    font-size: 18rpx;
    font-weight: 700;
    color: #fff;
    background: #ff4d4f;
    border-radius: 14rpx;
    padding: 0 6rpx;
  }
}

.log-center {
  flex: 1;
  min-width: 0;
  overflow: hidden;

  .log-top-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12rpx;
    margin-bottom: 8rpx;
  }

  .log-address {
    font-size: 28rpx;
    color: var(--text-primary);
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
  }

  .log-kwh {
    font-size: 26rpx;
    font-weight: 800;
    color: var(--color-primary);
    white-space: nowrap;
    flex-shrink: 0;

    .log-unit {
      font-size: 18rpx;
      font-weight: 400;
      color: var(--text-placeholder);
    }
  }

  .log-meta {
    display: flex;
    align-items: center;
    gap: 6rpx;
    flex-wrap: nowrap;

    .log-time {
      font-size: 22rpx;
      color: var(--text-placeholder);
      flex-shrink: 0;
    }

    .log-count-tag {
      font-size: 20rpx;
      color: var(--color-primary);
      background: rgba(37, 99, 235, 0.1);
      padding: 2rpx 10rpx;
      border-radius: 8rpx;
      flex-shrink: 0;
    }
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
