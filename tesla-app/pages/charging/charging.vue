<template>
  <view class="charging-container" :class="themeClass" :style="{ paddingTop: 'calc(' + statusBarHeight + 'px + 88rpx)' }">
    <NavBar title="充电记录" />
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
    <view class="city-stats-card" v-if="cityStats && cityStats.city_count && Object.keys(cityStats.city_count).length > 0">
      <view class="card-header">
        <view class="card-title">
          <view class="title-icon">
            <Icon name="Location" :size="20" color="#fff" />
          </view>
          <text class="title-text">城市充电分布</text>
        </view>
      </view>
      <view class="city-list">
        <view class="city-item" v-for="(count, city) in cityStats.city_count" :key="city">
          <text class="city-name">{{ city }}</text>
          <view class="city-bar">
            <view class="city-bar-fill" :style="{ width: getCityBarWidth(count) + '%' }"></view>
          </view>
          <text class="city-count">{{ count }}次</text>
        </view>
      </view>
    </view>

    <view class="month-section">
      <view class="section-header">
        <view class="section-title">
          <view class="section-icon">
            <Icon name="Flash" :size="18" themeColor="primary" />
          </view>
          <text class="section-title-text">月度充电</text>
        </view>
        <view class="section-link" @click="goChargingMap" v-if="currentVIN">
          <Icon name="Map" :size="16" themeColor="primary" />
          <text class="link-text">充电地图</text>
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
              <text class="stat-value">{{ fmt(item.total_kwh, 1) }}</text>
              <text class="stat-label">充电量(kWh)</text>
            </view>
            <view class="stat-item">
              <text class="stat-value">{{ item.charge_count || 0 }}</text>
              <text class="stat-label">充电次数</text>
            </view>
            <view class="stat-item">
              <text class="stat-value">{{ fmt(item.max_power, 1) }}</text>
              <text class="stat-label">最大功率(kW)</text>
            </view>
            <view class="stat-item">
              <text class="stat-value">{{ fmt(item.avg_kwh_per_charge, 1) }}</text>
              <text class="stat-label">次均充电(kWh)</text>
            </view>
            <view class="stat-item cost-item">
              <text class="stat-value cost-value">
                <text v-if="item.total_cost !== undefined && item.total_cost !== null">¥{{ fmt(item.total_cost, 2) }}</text>
                <text v-else class="no-cost">--</text>
              </text>
              <text class="stat-label cost-label">总花费</text>
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
          <Icon name="FlashOff" :size="64" color="#ccc" />
        </view>
        <text class="empty-text">暂无充电记录</text>
        <text class="empty-sub">开始充电后将在此显示</text>
      </view>
    </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getUserVehicles } from '@/api/vehicle.js'
import { getChargingStats, getMonthlyChargingList } from '@/api/charging.js'
import { getChargingAnalysis } from '@/api/ai.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useSystemInfo } from '@/utils/system.js'
import { useThemeStore } from '@/store/theme'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)

const monthList = ref([])
const cityStats = ref(null)
const monthAiSummaries = ref({})

const loadMonthAiSummaries = async () => {
  for (const item of monthList.value) {
    try {
      const refId = `charging_monthly:${item.month}`
      const res = await getChargingAnalysis(currentVIN.value, refId)
      if (res?.data?.summary) {
        monthAiSummaries.value[item.month] = res.data.summary
      }
    } catch (e) {}
  }
}
const vehicles = ref([])
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

const formatMonth = (month) => {
  if (!month) return ''
  const [y, m] = month.split('-')
  return `${y}年${parseInt(m)}月`
}

const loadVehicles = () => {
  getUserVehicles().then((res) => {
    vehicles.value = res.data || []
    if (vehicles.value.length > 0) {
      loadChargingData()
    }
  })
}

const onVehicleChange = (e) => {
  vehicleIndex.value = e.detail.value
  loadChargingData()
}

const loadChargingData = () => {
  if (!currentVIN.value) return
  loading.value = true

  getMonthlyChargingList(currentVIN.value).then((res) => {
    monthList.value = res.data || []
    loadMonthAiSummaries()
  }).catch(() => {
    monthList.value = []
  })

  const now = new Date()
  const monthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
  const start = `${monthAgo.getFullYear()}-${String(monthAgo.getMonth() + 1).padStart(2, '0')}-${String(monthAgo.getDate()).padStart(2, '0')}`
  const end = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
  getChargingStats(currentVIN.value, start, end).then((res) => {
    cityStats.value = res.data
  }).catch(() => {
    cityStats.value = null
  }).finally(() => {
    loading.value = false
  })
}

const goMonthDetail = (month) => {
  uni.navigateTo({ url: `/pages/charging/month?vin=${currentVIN.value}&month=${month}` })
}

const goChargingMap = () => {
  uni.navigateTo({ url: `/pages/charging/map?vin=${currentVIN.value}` })
}

const getCityBarWidth = (count) => {
  if (!cityStats.value || !cityStats.value.city_count) return 0
  const maxCount = Math.max(...Object.values(cityStats.value.city_count))
  if (maxCount === 0) return 0
  return (count / maxCount) * 100
}
</script>

<style lang="scss" scoped>
.charging-container {
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

.city-stats-card {
  background: var(--bg-card);
  border-radius: 28rpx;
  padding: 32rpx;
  margin-bottom: 24rpx;
  box-shadow: var(--shadow-card);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24rpx;

  .card-title {
    display: flex;
    align-items: center;
    gap: 16rpx;

    .title-icon {
      width: 48rpx;
      height: 48rpx;
      border-radius: 14rpx;
      background: linear-gradient(135deg, var(--color-primary), #1d4ed8);
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .title-text {
      font-size: 32rpx;
      font-weight: 700;
      color: var(--text-primary);
    }
  }
}

.city-list {
  .city-item {
    display: flex;
    align-items: center;
    gap: 16rpx;
    padding: 18rpx 0;
    border-bottom: 1rpx solid var(--border-divider);

    &:last-child {
      border-bottom: none;
    }

    .city-name {
      font-size: 28rpx;
      color: var(--text-primary);
      font-weight: 600;
      width: 120rpx;
      flex-shrink: 0;
    }

    .city-bar {
      flex: 1;
      height: 20rpx;
      background: var(--bg-bar);
      border-radius: 10rpx;
      overflow: hidden;

      .city-bar-fill {
        height: 100%;
        background: var(--bg-bar-fill);
        border-radius: 10rpx;
      }
    }

    .city-count {
      font-size: 24rpx;
      color: var(--text-tertiary);
      width: 80rpx;
      text-align: right;
      flex-shrink: 0;
      font-weight: 500;
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
  grid-template-columns: repeat(5, 1fr);
  gap: 12rpx;
}

.month-ai-row {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-top: 16rpx;
  padding: 14rpx 16rpx;
  background: var(--bg-entry);
  border: 1rpx solid var(--border-ai);
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
  }

  .stat-label {
    font-size: 18rpx;
    color: var(--text-tertiary);
    margin-top: 4rpx;
    display: block;
  }

  &.cost-item {
    background: linear-gradient(135deg, rgba(255, 183, 77, 0.15), rgba(255, 167, 38, 0.1));
    border: 1rpx solid rgba(255, 183, 77, 0.3);

    .cost-value {
      color: #ff9800;
      font-size: 26rpx;

      .no-cost {
        color: var(--text-tertiary);
        font-size: 24rpx;
      }
    }

    .cost-label {
      color: #ff9800;
      font-weight: 600;
    }
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
