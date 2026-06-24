<template>
  <view class="trip-container" :class="themeClass" :style="{ paddingTop: 'calc(' + statusBarHeight + 'px + 88rpx)' }">
    <NavBar title="行驶记录" />
    <view class="vehicle-selector" v-if="vehicles.length > 1">
      <picker @change="onVehicleChange" :value="vehicleIndex" :range="vehicleNames">
        <view class="picker-value">
          <Icon name="CarSport" :size="20" themeColor="primary" />
          <text class="picker-text">{{ vehicles[vehicleIndex]?.vehicle_name }}</text>
          <Icon name="ChevronDown" :size="16" color="#999" />
        </view>
      </picker>
    </view>

    <scroll-view scroll-y class="main-scroll">
    <view class="month-section">
      <view class="section-header">
        <view class="section-title">
          <view class="section-icon">
            <Icon name="Navigate" :size="18" themeColor="primary" />
          </view>
          <text class="section-title-text">月度行驶</text>
        </view>
      </view>

      <view class="month-list" v-if="monthList.length > 0">
        <view class="month-card" v-for="item in monthList" :key="item.month" @click="goMonthDetail(item.month)">
          <view class="month-header">
            <view class="month-label">
              <Icon name="Calendar" :size="16" themeColor="primary" />
              <text class="month-text">{{ formatMonth(item.month) }}</text>
            </view>
            <Icon name="ChevronRight" :size="18" color="#ccc" />
          </view>
          <view class="month-stats">
            <view class="stat-item">
              <text class="stat-value">{{ fmt(item.total_distance, 1) }}</text>
              <text class="stat-label">行驶里程(km)</text>
            </view>
            <view class="stat-item">
              <text class="stat-value">{{ item.trip_count || 0 }}</text>
              <text class="stat-label">行程次数</text>
            </view>
            <view class="stat-item">
              <text class="stat-value">{{ formatDuration(item.total_duration) }}</text>
              <text class="stat-label">行驶时长</text>
            </view>
            <view class="stat-item">
              <text class="stat-value">{{ fmt(item.total_energy, 1) }}</text>
              <text class="stat-label">总能耗(kWh)</text>
            </view>
            <view class="stat-item" v-if="item.total_cost != null">
              <text class="stat-value cost-value">¥{{ fmt(item.total_cost, 2) }}</text>
              <text class="stat-label">电费(元)</text>
            </view>
          </view>
          <view class="month-ai-row" v-if="monthAiSummaries[item.month]" @click.stop="goMonthDetail(item.month)">
            <Icon name="Sparkles" :size="14" themeColor="primary" />
            <text class="month-ai-text">{{ monthAiSummaries[item.month] }}</text>
            <Icon name="ChevronForward" :size="14" color="#ccc" />
          </view>
        </view>
      </view>

      <view class="empty-state" v-else-if="!loading">
        <view class="empty-icon">
          <Icon name="LocationOutline" :size="64" color="#ccc" />
        </view>
        <text class="empty-text">暂无行驶记录</text>
        <text class="empty-sub">开始驾驶后将在此显示</text>
      </view>
    </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getUserVehicles } from '@/api/vehicle.js'
import { getMonthlyTripList } from '@/api/trip.js'
import { getTripAnalysis } from '@/api/ai.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useSystemInfo } from '@/utils/system.js'
import { useThemeStore } from '@/store/theme'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)

const monthList = ref([])
const monthAiSummaries = ref({})
const vehicles = ref([])

const loadMonthAiSummaries = async () => {
  for (const item of monthList.value) {
    try {
      const refId = `trip_monthly:${item.month}`
      const res = await getTripAnalysis(currentVIN.value, refId)
      if (res?.data?.summary) {
        monthAiSummaries.value[item.month] = res.data.summary
      }
    } catch (e) {}
  }
}
const vehicleIndex = ref(0)
const loading = ref(false)

const { statusBarHeight } = useSystemInfo()

const vehicleNames = computed(() => vehicles.value.map(v => v.vehicle_name))
const currentVIN = computed(() => vehicles.value[vehicleIndex.value]?.vin)

onMounted(() => {
  loadVehicles()
})

const fmt = (val, digits) => {
  if (val === undefined || val === null || isNaN(val)) return '0'
  return Number(val).toFixed(digits)
}

const formatDuration = (seconds) => {
  if (!seconds || seconds <= 0) return '0min'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (h > 0) return `${h}h${m > 0 ? m + 'min' : ''}`
  return `${m}min`
}

const formatMonth = (month) => {
  if (!month) return ''
  const [y, m] = month.split('-')
  return `${y}年${parseInt(m)}月`
}

const loadVehicles = () => {
  getUserVehicles().then((res) => {
    vehicles.value = res.data || []
    if (vehicles.value.length > 0) {
      loadTripData()
    }
  })
}

const onVehicleChange = (e) => {
  vehicleIndex.value = e.detail.value
  loadTripData()
}

const loadTripData = () => {
  if (!currentVIN.value) return
  loading.value = true

  getMonthlyTripList(currentVIN.value).then((res) => {
    monthList.value = res.data || []
    loadMonthAiSummaries()
  }).catch(() => {
    monthList.value = []
  }).finally(() => {
    loading.value = false
  })
}

const goMonthDetail = (month) => {
  uni.navigateTo({ url: `/pages/trip/month?vin=${currentVIN.value}&month=${month}` })
}
</script>

<style lang="scss" scoped>
.trip-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
  box-sizing: border-box;
  background: var(--bg-page);
  padding: 0 24rpx 40rpx;
  display: flex;
  flex-direction: column;
}

.main-scroll {
  flex: 1;
  overflow: hidden;
}

.vehicle-selector {
  background: var(--bg-card);
  border-radius: 28rpx;
  padding: 24rpx 32rpx;
  margin-bottom: 24rpx;
  box-shadow: var(--shadow-card);

  .picker-value {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12rpx;

    .picker-text {
      font-size: 30rpx;
      color: var(--text-primary);
      font-weight: 600;
    }
  }
}

.month-section {
  .section-header {
    margin-bottom: 20rpx;
    display: flex;
    justify-content: space-between;
    align-items: center;

    .section-title {
      display: flex;
      align-items: center;
      gap: 12rpx;

      .section-icon {
        width: 40rpx;
        height: 40rpx;
        border-radius: 10rpx;
        background: var(--bg-icon-wrap);
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .section-title-text {
        font-size: 32rpx;
        font-weight: 700;
        color: var(--text-primary);
      }
    }

    .section-link {
      display: flex;
      align-items: center;
      gap: 6rpx;
      padding: 10rpx 20rpx;
      background: var(--bg-icon-wrap);
      border-radius: 20rpx;

      .link-text {
        font-size: 24rpx;
        color: var(--color-primary);
        font-weight: 500;
      }
    }
  }
}

.month-card {
  background: var(--bg-card);
  border-radius: 28rpx;
  padding: 28rpx;
  margin-bottom: 16rpx;
  box-shadow: var(--shadow-card);

  &:active {
    opacity: 0.9;
  }
}

.month-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20rpx;

  .month-label {
    display: flex;
    align-items: center;
    gap: 10rpx;

    .month-text {
      font-size: 30rpx;
      color: var(--text-primary);
      font-weight: 700;
    }
  }
}

.month-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12rpx;
}

.month-ai-row {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-top: 16rpx;
  padding: 14rpx 16rpx;
  background: rgba(37, 99, 235, 0.05);
  border: 1rpx solid rgba(37, 99, 235, 0.1);
  border-radius: 14rpx;

  .month-ai-text {
    flex: 1;
    font-size: 22rpx;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    line-height: 1.4;
  }
}

.stat-item {
  text-align: center;
  padding: 16rpx 8rpx;
  background: var(--bg-card-secondary);
  border-radius: 16rpx;

  .stat-value {
    font-size: 28rpx;
    font-weight: 800;
    color: var(--color-primary);
    display: block;

    &.cost-value {
      color: #f59e0b;
    }
  }

  .stat-label {
    font-size: 18rpx;
    color: var(--text-tertiary);
    margin-top: 4rpx;
    display: block;
  }
}

.empty-state {
  text-align: center;
  padding: 80rpx 40rpx;
  background: var(--bg-card);
  border-radius: 28rpx;

  .empty-icon {
    width: 120rpx;
    height: 120rpx;
    border-radius: 50%;
    background: var(--bg-empty-icon);
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 24rpx;
  }

  .empty-text {
    font-size: 30rpx;
    color: var(--text-tertiary);
    display: block;
    font-weight: 500;
  }

  .empty-sub {
    font-size: 24rpx;
    color: var(--text-placeholder);
    margin-top: 8rpx;
    display: block;
  }
}
</style>
