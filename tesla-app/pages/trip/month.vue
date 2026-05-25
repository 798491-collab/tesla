<template>
  <view class="month-container" :class="themeClass">
    <NavBar title="月度行驶" />
    <scroll-view scroll-y class="main-scroll">
    <view class="month-header-card">
      <view class="header-row">
        <Icon name="Navigate" :size="22" themeColor="primary" />
        <text class="header-title">{{ formatMonth(month) }} 行驶记录</text>
      </view>
    </view>

    <view class="ai-card" v-if="aiResult" @click="aiExpanded = !aiExpanded">
      <view class="ai-card-header">
        <view class="ai-card-title">
          <Icon name="Sparkles" :size="18" themeColor="primary" />
          <text class="ai-title-text">AI 行程月度分析</text>
        </view>
        <view class="ai-header-right">
          <Icon :name="aiExpanded ? 'ChevronUp' : 'ChevronDown'" :size="16" themeColor="hint" />
        </view>
      </view>
      <view class="ai-summary-row" v-if="!aiExpanded">
        <text class="ai-summary-text">{{ aiResult.summary || '点击查看详细分析' }}</text>
      </view>
      <view class="ai-card-body" v-if="aiExpanded">
        <text class="ai-text" v-for="(line, i) in aiLines" :key="i">{{ line }}</text>
      </view>
      <text class="ai-time" v-if="aiExpanded">{{ formatAITime(aiResult.created_at) }}</text>
    </view>

    <view class="ai-card ai-loading" v-if="aiLoading">
      <view class="ai-card-header">
        <view class="ai-card-title">
          <Icon name="Sparkles" :size="18" themeColor="primary" />
          <text class="ai-title-text">AI 行程分析</text>
        </view>
      </view>
      <view class="ai-card-body">
        <view class="ai-spinner"></view>
        <text class="ai-loading-text">AI 正在分析中...</text>
      </view>
    </view>

    <view class="log-list" v-if="logs.length > 0">
      <view class="log-item" v-for="log in logs" :key="log.id" @click="goRoute(log)">
        <view class="log-header">
          <view class="log-date">
            <Icon name="Calendar" :size="14" themeColor="hint" />
            <text class="date-text">{{ formatDate(log.start_time) }}</text>
          </view>
          <view class="log-distance">
            <Icon name="Speedometer" :size="16" themeColor="primary" />
            <text class="distance-text">{{ log.distance?.toFixed(1) }} km</text>
          </view>
        </view>
        <view class="log-body">
          <view class="log-route">
            <view class="route-point">
              <view class="point-dot start"></view>
              <view class="point-detail">
                <text class="point-label">出发</text>
                <text class="point-address">{{ log.start_address || formatCoord(log.start_lat, log.start_lng) }}</text>
                <text class="point-city" v-if="log.start_city">{{ log.start_city }}</text>
              </view>
            </view>
            <view class="route-connector">
              <view class="connector-line"></view>
              <view class="connector-dot"></view>
              <view class="connector-line"></view>
            </view>
            <view class="route-point">
              <view class="point-dot end"></view>
              <view class="point-detail">
                <text class="point-label">到达</text>
                <text class="point-address">{{ log.end_address || formatCoord(log.end_lat, log.end_lng) }}</text>
                <text class="point-city" v-if="log.end_city">{{ log.end_city }}</text>
              </view>
            </view>
          </view>
          <view class="log-stats">
            <view class="stat-item">
              <Icon name="Car" :size="14" themeColor="hint" />
              <text class="stat-text">{{ log.avg_speed?.toFixed(1) }} km/h</text>
            </view>
            <view class="stat-item">
              <Icon name="BatteryCharging" :size="14" themeColor="hint" />
              <text class="stat-text">{{ log.avg_consumption?.toFixed(1) || '--' }} kWh</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <view class="empty-state" v-else-if="!loading">
      <text class="empty-text">该月暂无行驶记录</text>
    </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getTripLogs } from '@/api/trip.js'
import { getTripAnalysis, triggerTripAnalysis } from '@/api/ai.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)
const hintColor = computed(() => themeStore.colors.hint)

const logs = ref([])
const month = ref('')
const vin = ref('')
const aiResult = ref(null)
const aiLoading = ref(false)
const aiExpanded = ref(false)

const aiLines = computed(() => {
  if (!aiResult.value?.result) return []
  return aiResult.value.result.split('\n').filter(l => l.trim()).map(l => l.replace(/^#{1,3}\s*/, '').replace(/\*\*/g, '').replace(/^[-*]\s*/, '• ').trim())
})
const loading = ref(false)

onLoad((options) => {
  vin.value = options?.vin || ''
  month.value = options?.month || ''

  if (vin.value && month.value) {
    loadLogs()
    loadAIAnalysis()
  }
})

const formatMonth = (m) => {
  if (!m) return ''
  const [y, mo] = m.split('-')
  return `${y}年${parseInt(mo)}月`
}

const loadLogs = () => {
  loading.value = true
  const [y, m] = month.value.split('-')
  const start = `${y}-${m}-01`
  const nextMonth = m === '12' ? `${parseInt(y) + 1}-01` : `${y}-${String(parseInt(m) + 1).padStart(2, '0')}`
  const end = `${nextMonth}-01`

  getTripLogs(vin.value, start, end).then((res) => {
    logs.value = res.data || []
  }).catch(() => {
    logs.value = []
  }).finally(() => {
    loading.value = false
  })
}

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

const formatCoord = (lat, lng) => {
  if (!lat && !lng) return '--'
  return `${lat?.toFixed(4)}, ${lng?.toFixed(4)}`
}

const goRoute = (log) => {
  uni.setStorageSync('currentTrip', JSON.stringify(log))
  uni.setStorageSync('currentTripVIN', vin.value)
  uni.navigateTo({ url: `/pages/trip/route?id=${log.id}` })
}

const loadAIAnalysis = async () => {
  if (!vin.value || !month.value) return
  const refId = `trip_monthly:${month.value}`
  try {
    const res = await getTripAnalysis(vin.value, refId)
    if (res?.data) {
      aiResult.value = res.data
    } else {
      aiLoading.value = true
      await triggerTripAnalysis(vin.value, refId)
      setTimeout(async () => {
        const res2 = await getTripAnalysis(vin.value, refId)
        if (res2?.data) aiResult.value = res2.data
        aiLoading.value = false
      }, 15000)
    }
  } catch (e) {
    aiLoading.value = false
  }
}

const formatAITime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}
</script>

<style lang="scss" scoped>
.month-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
  box-sizing: border-box;
  background: linear-gradient(180deg, var(--dark-page-bg) 0%, var(--dark-page-icon-wrap-bg) 100%);
  padding: 0 32rpx 40rpx;
  display: flex;
  flex-direction: column;
  padding-top: calc(var(--status-bar-height, 44px) + 88rpx);
}

.main-scroll {
  flex: 1;
  overflow: hidden;
}

.month-header-card {
  background: var(--dark-page-icon-wrap-bg);
  border-radius: 20rpx;
  padding: 28rpx 32rpx;
  margin-bottom: 24rpx;

  .header-row {
    display: flex;
    align-items: center;
    gap: 12rpx;

    .header-title {
      font-size: 32rpx;
      font-weight: 700;
      color: var(--dark-page-text);
    }
  }
}

.ai-card {
  background: var(--dark-page-glass-bg);
  border: 1rpx solid var(--dark-page-glass-border);
  border-radius: 20rpx;
  padding: 24rpx 28rpx;
  margin-bottom: 24rpx;

  .ai-card-header {
    display: flex;
    align-items: center;
    gap: 8rpx;

    .ai-card-title {
      display: flex;
      align-items: center;
      gap: 8rpx;

      .ai-title-text {
        font-size: 28rpx;
        font-weight: 700;
        color: var(--dark-page-text);
      }
    }

    .ai-header-right {
      display: flex;
      align-items: center;
      gap: 12rpx;
      flex-shrink: 0;
    }

    .ai-time {
      font-size: 22rpx;
      color: var(--dark-page-text-hint);
    }
  }

  .ai-card-body {
    margin-top: 16rpx;

    .ai-text {
      font-size: 26rpx;
      color: var(--dark-page-text-secondary);
      line-height: 1.8;
      display: block;
    }
  }

  .ai-summary-row {
    margin-top: 12rpx;
    padding: 14rpx 18rpx;
    background: var(--dark-page-glass-bg);
    border-radius: 12rpx;

    .ai-summary-text {
      font-size: 26rpx;
      color: var(--dark-page-text-secondary);
      line-height: 1.6;
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  &.ai-loading {
    .ai-card-body {
      display: flex;
      align-items: center;
      gap: 16rpx;
      padding: 12rpx 0;

      .ai-spinner {
        width: 32rpx;
        height: 32rpx;
        border: 3rpx solid var(--dark-page-glass-border);
        border-top-color: var(--color-primary);
        border-radius: 50%;
        animation: ai-spin 0.8s linear infinite;
      }

      .ai-loading-text {
        font-size: 26rpx;
        color: var(--dark-page-text-hint);
      }
    }
  }
}

@keyframes ai-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.log-item {
  background: var(--dark-page-icon-wrap-bg);
  border-radius: 20rpx;
  padding: 28rpx;
  margin-bottom: 16rpx;

  &:active {
    background: var(--dark-page-press-bg);
  }
}

.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20rpx;

  .log-date {
    display: flex;
    align-items: center;
    gap: 8rpx;

    .date-text {
      font-size: 26rpx;
      color: var(--dark-page-text-secondary);
      font-weight: 500;
    }
  }

  .log-distance {
    display: flex;
    align-items: center;
    gap: 8rpx;

    .distance-text {
      font-size: 30rpx;
      font-weight: 800;
      color: var(--color-primary);
    }
  }
}

.log-body {
  padding: 0;
}

.log-route {
  padding: 20rpx;
  background: var(--dark-page-glass-bg);
  border-radius: 20rpx;
  margin-bottom: 16rpx;
}

.route-point {
  display: flex;
  align-items: flex-start;

  .point-dot {
    width: 20rpx;
    height: 20rpx;
    border-radius: 50%;
    margin-right: 16rpx;
    margin-top: 4rpx;
    flex-shrink: 0;

    &.start {
      background: #3cc9a5;
      box-shadow: 0 0 0 6rpx rgba(60, 201, 165, 0.2);
    }

    &.end {
      background: var(--color-primary);
      box-shadow: 0 0 0 6rpx rgba(37, 99, 235, 0.2);
    }
  }

  .point-detail {
    flex: 1;
    min-width: 0;

    .point-label {
      font-size: 20rpx;
      color: var(--dark-page-text-hint);
      display: block;
    }

    .point-address {
      font-size: 26rpx;
      color: var(--dark-page-text);
      font-weight: 600;
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .point-city {
      font-size: 20rpx;
      color: var(--dark-page-text-hint);
      display: block;
      margin-top: 2rpx;
    }
  }
}

.route-connector {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 6rpx 0;
  margin-left: 8rpx;
  width: 20rpx;

  .connector-line {
    width: 2rpx;
    flex: 1;
    background: var(--dark-page-glass-border);
  }

  .connector-dot {
    width: 8rpx;
    height: 8rpx;
    border-radius: 50%;
    background: var(--dark-page-glass-border);
    margin: 4rpx 0;
  }
}

.log-stats {
  display: flex;
  gap: 32rpx;
  padding-top: 16rpx;
  border-top: 1rpx solid var(--dark-page-glass-border);

  .stat-item {
    display: flex;
    align-items: center;
    gap: 8rpx;

    .stat-text {
      font-size: 24rpx;
      color: var(--dark-page-text-secondary);
      font-weight: 500;
    }
  }
}

.empty-state {
  text-align: center;
  padding: 80rpx 40rpx;
  background: var(--dark-page-icon-wrap-bg);
  border-radius: 20rpx;

  .empty-text {
    font-size: 30rpx;
    color: var(--dark-page-text-hint);
    display: block;
    font-weight: 500;
  }
}
</style>
