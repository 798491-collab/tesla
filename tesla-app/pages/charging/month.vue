<template>
  <view class="month-container" :class="themeClass">
    <NavBar title="月度充电" />
    <scroll-view scroll-y class="main-scroll">
    <view class="month-header-card">
      <view class="header-row">
        <view class="header-left">
          <Icon name="Flash" :size="22" themeColor="primary" />
          <text class="header-title">{{ formatMonth(month) }} 充电地图</text>
        </view>
        <view class="map-entry-btn" @click="goChargingMap" v-if="hasMapData">
          <Icon name="Map" :size="16" themeColor="primary" />
          <text class="map-entry-text">充电地图</text>
        </view>
      </view>
    </view>

    <view class="ai-card" v-if="aiResult" @click="aiExpanded = !aiExpanded">
      <view class="ai-card-header">
        <view class="ai-card-title">
          <Icon name="Sparkles" :size="18" themeColor="primary" />
          <text class="ai-title-text">AI 充电月度分析</text>
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
          <text class="ai-title-text">AI 充电月度分析</text>
        </view>
      </view>
      <view class="ai-card-body">
        <view class="ai-spinner"></view>
        <text class="ai-loading-text">分析生成中...</text>
      </view>
    </view>

    <view class="log-list" v-if="logs.length > 0">
      <view class="log-item" v-for="log in logs" :key="log.id">
        <view class="log-header">
          <view class="log-date">
            <Icon name="Calendar" :size="14" themeColor="hint" />
            <text class="date-text">{{ formatDate(log.start_time) }}</text>
          </view>
          <view class="log-type" :class="log.charge_type?.toLowerCase()">
            <Icon :name="log.charge_type === 'DC' ? 'Flash' : 'Power'" :size="12" color="#ffffff" />
            <text class="type-text">{{ log.charge_type === 'DC' ? '快充' : '慢充' }}</text>
          </view>
        </view>
        <view class="log-body">
          <view class="log-info">
            <text class="info-label">SOC</text>
            <text class="info-value">{{ log.soc_start }}% → {{ log.soc_end }}%</text>
          </view>
          <view class="log-info">
            <text class="info-label">电量</text>
            <text class="info-value highlight">{{ log.charge_kwh?.toFixed(2) }} kWh</text>
          </view>
          <view class="log-info">
            <text class="info-label">时长</text>
            <text class="info-value">{{ formatDuration(log.charge_duration_minutes) }}</text>
          </view>
          <view class="log-info">
            <text class="info-label">{{ log.charge_type === 'DC' ? '费用' : '电价' }}</text>
            <view class="info-value-row" v-if="editingLogId !== log.id" @click.stop="startEditPrice(log)">
              <template v-if="log.charge_type === 'DC'">
                <text class="info-value" :class="{ 'highlight': log.total_cost }">
                  {{ log.total_cost ? '¥' + log.total_cost.toFixed(2) : '添加' }}
                </text>
              </template>
              <template v-else>
                <text class="info-value" :class="{ 'highlight': log.price_per_kwh }">
                  {{ log.price_per_kwh ? log.price_per_kwh.toFixed(2) + '元/kWh' : '添加' }}
                </text>
                <text class="price-total-tag" v-if="log.total_cost">¥{{ log.total_cost.toFixed(2) }}</text>
              </template>
            </view>
            <view v-else class="price-inline-edit">
              <input class="price-input-sm" v-model="priceInput" type="digit" :placeholder="log.charge_type === 'DC' ? '0.00' : '0.00'" maxlength="8" />
              <text class="price-unit-sm">{{ log.charge_type === 'DC' ? '元' : '元/kWh' }}</text>
              <view class="price-btn-sm save" @click.stop="savePrice(log)">
                <text class="btn-text-sm">✓</text>
              </view>
              <view class="price-btn-sm cancel" @click.stop="cancelEditPrice">
                <text class="btn-text-sm">✕</text>
              </view>
            </view>
          </view>
          <view class="log-info full location-row">
            <Icon name="Location" :size="14" themeColor="hint" />
            <text class="info-value small">{{ [log.city, log.district, log.address || log.location].filter(Boolean).join(' · ') || '--' }}</text>
          </view>
        </view>
        <view class="log-ai-section">
          <view class="log-ai-toggle" @click.stop="toggleChargeAI(log)">
            <Icon name="Sparkles" :size="14" themeColor="primary" />
            <text class="log-ai-toggle-text">AI 充电分析</text>
            <Icon :name="expandedLogId === log.id ? 'ChevronUp' : 'ChevronDown'" :size="14" themeColor="hint" />
          </view>
          <view class="log-ai-content" v-if="expandedLogId === log.id">
            <view v-if="chargingAiLoadingMap[log.id]" class="log-ai-loading">
              <view class="ai-spinner-sm"></view>
              <text class="ai-loading-text-sm">分析中...</text>
            </view>
            <view v-else-if="chargingAiMap[log.id]" class="log-ai-result">
              <view class="log-ai-summary" @click.stop="toggleChargeAIDetail(log.id)">
                <text class="log-ai-summary-text">{{ chargingAiMap[log.id].summary || '点击查看详细分析' }}</text>
                <Icon :name="expandedChargeDetail === log.id ? 'ChevronUp' : 'ChevronDown'" :size="14" themeColor="hint" />
              </view>
              <view class="log-ai-detail" v-if="expandedChargeDetail === log.id">
                <text class="log-ai-text" v-for="(line, i) in getChargeAiLines(log.id)" :key="i">{{ line }}</text>
              </view>
            </view>
            <view v-else class="log-ai-empty">
              <text class="log-ai-empty-text">暂无分析结果</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <view class="empty-state" v-else-if="!loading">
      <text class="empty-text">该月暂无充电记录</text>
    </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getChargingLogs, updateChargingPrice } from '@/api/charging.js'
import { getChargingAnalysis } from '@/api/ai.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'
import { useVehicleData } from '@/utils/vehicle-data.js'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)
const hintColor = computed(() => themeStore.colors.hint)
const vehicleStore = useVehicleData()

const logs = ref([])
const month = ref('')
const vin = ref('')
const aiResult = ref(null)
const aiLoading = ref(false)
const aiExpanded = ref(false)
const expandedLogId = ref(null)
const expandedChargeDetail = ref(null)
const chargingAiMap = ref({})
const chargingAiLoadingMap = ref({})
const editingLogId = ref(null)
const priceInput = ref('')

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

  getChargingLogs(vin.value, start, end).then((res) => {
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

const formatDuration = (minutes) => {
  if (!minutes || minutes <= 0) return '--'
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (h > 0) return `${h}h${m > 0 ? m + 'min' : ''}`
  return `${m}min`
}

const goAIAnalysis = () => {
  uni.navigateTo({ url: `/pages/ai/analysis?vin=${vin.value}&type=charging&mode=monthly&month=${month.value}` })
}

const loadAIAnalysis = async () => {
  if (!vin.value || !month.value) return
  const refId = `charging_monthly:${month.value}`
  try {
    const res = await getChargingAnalysis(vin.value, refId)
    if (res?.data) {
      aiResult.value = res.data
      aiLoading.value = false
    } else {
      aiLoading.value = true
    }
  } catch (e) {
    aiLoading.value = false
  }
}

watch(() => vehicleStore.analysisNotification, (notification) => {
  if (!notification) return
  const monthlyRefId = `charging_monthly:${month.value}`
  if (notification.type === 'analysis_complete' && notification.refId === monthlyRefId) {
    loadAIAnalysis()
  }
  // 单次充电分析完成时刷新
  if (notification.type === 'analysis_complete' && notification.refId?.startsWith('charging:')) {
    const chargeId = notification.refId.replace('charging:', '')
    if (chargingAiLoadingMap.value[chargeId]) {
      loadChargeAI(chargeId)
    }
  }
})

const formatAITime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

const toggleChargeAI = async (log) => {
  if (expandedLogId.value === log.id) {
    expandedLogId.value = null
    return
  }
  expandedLogId.value = log.id

  if (chargingAiMap.value[log.id] || chargingAiLoadingMap.value[log.id]) return

  await loadChargeAI(log.id)
}

const loadChargeAI = async (chargeId) => {
  const refId = `charging:${chargeId}`
  chargingAiLoadingMap.value[chargeId] = true

  try {
    const res = await getChargingAnalysis(vin.value, refId)
    if (res?.data) {
      chargingAiMap.value[chargeId] = res.data
      chargingAiLoadingMap.value[chargeId] = false
    } else {
      // 后端会自动分析，等待 WS 通知
    }
  } catch (e) {
    chargingAiLoadingMap.value[chargeId] = false
  }
}

const toggleChargeAIDetail = (logId) => {
  expandedChargeDetail.value = expandedChargeDetail.value === logId ? null : logId
}

const getChargeAiLines = (logId) => {
  const result = chargingAiMap.value[logId]?.result
  if (!result) return []
  return result.split('\n').filter(l => l.trim()).map(l => l.replace(/^#{1,3}\s*/, '').replace(/\*\*/g, '').replace(/^[-*]\s*/, '• ').trim())
}

const startEditPrice = (log) => {
  editingLogId.value = log.id
  const isDC = log.charge_type === 'DC'
  if (isDC) {
    priceInput.value = log.total_cost ? String(log.total_cost) : ''
  } else {
    priceInput.value = log.price_per_kwh ? String(log.price_per_kwh) : ''
  }
}

const savePrice = async (log) => {
  const price = parseFloat(priceInput.value)
  if (isNaN(price) || price < 0) {
    uni.showToast({ title: '请输入有效的金额', icon: 'none' })
    return
  }

  try {
    const isDC = log.charge_type === 'DC'
    const payload = isDC
      ? { total_cost: price }
      : { price_per_kwh: price }

    const res = await updateChargingPrice(log.id, payload)

    if (isDC) {
      log.total_cost = price
    } else {
      log.price_per_kwh = price
      if (res?.data?.total_cost) {
        log.total_cost = res.data.total_cost
      }
    }

    uni.showToast({ title: '保存成功', icon: 'success' })
    editingLogId.value = null
    priceInput.value = ''
  } catch (e) {
    uni.showToast({ title: '保存失败', icon: 'none' })
  }
}

const cancelEditPrice = () => {
  editingLogId.value = null
  priceInput.value = ''
}

const calculateTotalCost = (log) => {
  if (!log.price_per_kwh || !log.charge_kwh) return 0
  return (log.price_per_kwh * log.charge_kwh).toFixed(2)
}

const hasMapData = computed(() => {
  return logs.value.some(log => log.latitude && log.longitude)
})

const goChargingMap = () => {
  uni.navigateTo({ url: `/pages/charging/map?vin=${vin.value}` })
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
    justify-content: space-between;

    .header-left {
      display: flex;
      align-items: center;
      gap: 12rpx;

      .header-title {
        font-size: 32rpx;
        font-weight: 700;
        color: var(--dark-page-text);
      }
    }

    .map-entry-btn {
      display: flex;
      align-items: center;
      gap: 6rpx;
      padding: 10rpx 20rpx;
      background: var(--dark-page-glass-bg);
      border: 1rpx solid var(--dark-page-glass-border);
      border-radius: 20rpx;

      .map-entry-text {
        font-size: 24rpx;
        color: var(--color-primary);
        font-weight: 500;
      }
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

  .log-type {
    display: flex;
    align-items: center;
    gap: 6rpx;
    padding: 8rpx 18rpx;
    border-radius: 16rpx;
    font-size: 22rpx;
    font-weight: 600;

    &.dc {
      background: var(--gradient);

      .type-text {
        color: #ffffff;
      }
    }

    &.ac {
      background: linear-gradient(135deg, #3cc9a5, #52c41a);

      .type-text {
        color: #ffffff;
      }
    }
  }
}

.log-body {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16rpx;
  padding-top: 20rpx;
  border-top: 1rpx solid var(--dark-page-glass-border);
}

.log-ai-section {
  margin-top: 16rpx;
  border-top: 1rpx solid var(--dark-page-glass-border);
  padding-top: 16rpx;

  .log-ai-toggle {
    display: flex;
    align-items: center;
    gap: 8rpx;
    padding: 8rpx 0;

    .log-ai-toggle-text {
      font-size: 24rpx;
      color: var(--dark-page-text-secondary);
      font-weight: 500;
      flex: 1;
    }
  }

  .log-ai-content {
    background: var(--dark-page-glass-bg);
    border-radius: 16rpx;
    padding: 20rpx;
    margin-top: 8rpx;

    .log-ai-loading {
      display: flex;
      align-items: center;
      gap: 12rpx;

      .ai-spinner-sm {
        width: 24rpx;
        height: 24rpx;
        border: 2rpx solid var(--dark-page-glass-border);
        border-top-color: var(--color-primary);
        border-radius: 50%;
        animation: ai-spin 0.8s linear infinite;
      }

      .ai-loading-text-sm {
        font-size: 24rpx;
        color: var(--dark-page-text-hint);
      }
    }

    .log-ai-result {
      .log-ai-summary {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12rpx;

        .log-ai-summary-text {
          font-size: 24rpx;
          color: var(--dark-page-text-secondary);
          flex: 1;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
      }

      .log-ai-detail {
        margin-top: 12rpx;
        padding-top: 12rpx;
        border-top: 1rpx solid var(--dark-page-glass-border);
      }

      .log-ai-text {
        font-size: 24rpx;
        color: var(--dark-page-text-secondary);
        line-height: 1.7;
        display: block;
      }
    }

    .log-ai-empty {
      .log-ai-empty-text {
        font-size: 24rpx;
        color: var(--dark-page-text-hint);
      }
    }
  }
}

.log-info {
  &.full {
    grid-column: span 2;
  }

  .info-label {
    font-size: 22rpx;
    color: var(--dark-page-text-hint);
    display: block;
    margin-bottom: 6rpx;
  }

  .info-value {
    font-size: 28rpx;
    color: var(--dark-page-text);
    font-weight: 600;

    &.highlight {
      color: var(--color-primary);
    }

    &.small {
      font-size: 24rpx;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .info-value-row {
    display: flex;
    align-items: center;
    gap: 8rpx;
    flex-wrap: nowrap;
  }

  .price-total-tag {
    font-size: 20rpx;
    color: #fff;
    background: var(--color-primary);
    padding: 2rpx 10rpx;
    border-radius: 8rpx;
    font-weight: 600;
    white-space: nowrap;
  }

  .price-inline-edit {
    display: flex;
    align-items: center;
    gap: 8rpx;

    .price-input-sm {
      width: 100rpx;
      height: 48rpx;
      background: var(--dark-page-glass-bg);
      border-radius: 8rpx;
      padding: 0 12rpx;
      font-size: 24rpx;
      color: var(--dark-page-text);
      border: 1rpx solid var(--color-primary);
    }

    .price-unit-sm {
      font-size: 20rpx;
      color: var(--dark-page-text-hint);
      white-space: nowrap;
    }

    .price-btn-sm {
      width: 56rpx;
      height: 56rpx;
      border-radius: 10rpx;
      display: flex;
      align-items: center;
      justify-content: center;

      &.save {
        background: var(--color-primary);
      }

      &.cancel {
        background: var(--dark-page-glass-bg);
        border: 1rpx solid var(--dark-page-glass-border);
      }

      .btn-text-sm {
        font-size: 28rpx;
        color: #fff;
        font-weight: 700;
      }

      &.cancel .btn-text-sm {
        color: var(--dark-page-text-secondary);
      }
    }
  }

  &.location-row {
    display: flex;
    align-items: center;
    gap: 8rpx;
    padding-top: 12rpx;
    margin-top: 4rpx;
    border-top: 1rpx solid var(--dark-page-glass-border);

    .info-value.small {
      flex: 1;
      min-width: 0;
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
