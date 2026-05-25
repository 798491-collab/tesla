<template>
  <view class="month-container" :class="themeClass">
    <NavBar title="月度充电" />
    <scroll-view scroll-y class="main-scroll">
    <view class="month-header-card">
      <view class="header-row">
        <Icon name="Flash" :size="22" themeColor="primary" />
        <text class="header-title">{{ formatMonth(month) }} 充电记录</text>
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
        <text class="ai-loading-text">AI 正在分析中...</text>
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
          <view class="log-info full">
            <text class="info-label">位置</text>
            <text class="info-value small">{{ log.address || log.location || '--' }}</text>
            <text class="info-poi" v-if="log.poi_name">{{ log.poi_name }}</text>
          </view>
          <view class="log-info full">
            <text class="info-label">城市</text>
            <text class="info-value">{{ log.city || '--' }}</text>
            <text class="info-district" v-if="log.district">{{ log.district }}</text>
          </view>
          <view class="log-info price-section" :class="{ 'editing': editingLogId === log.id }">
            <view class="price-header">
              <text class="info-label">{{ log.charge_type === 'DC' ? '充电费用' : '电价' }}</text>
              <view v-if="editingLogId !== log.id" class="price-edit-btn" @click.stop="startEditPrice(log)">
                <text class="edit-btn-text">{{ (log.charge_type === 'DC' ? log.total_cost : log.price_per_kwh) ? '修改' : '添加' }}</text>
              </view>
            </view>
            <view v-if="editingLogId === log.id" class="price-edit-area">
              <view class="price-input-wrap">
                <input
                  class="price-input"
                  v-model="priceInput"
                  type="digit"
                  :placeholder="log.charge_type === 'DC' ? '0.00' : '0.00'"
                  maxlength="8"
                />
                <text class="price-unit">{{ log.charge_type === 'DC' ? '元' : '元/kWh' }}</text>
              </view>
              <view class="price-actions">
                <view class="price-btn save" @click.stop="savePrice(log)">
                  <text class="btn-text">保存</text>
                </view>
                <view class="price-btn cancel" @click.stop="cancelEditPrice">
                  <text class="btn-text">取消</text>
                </view>
              </view>
              <view v-if="log.charge_type !== 'DC' && priceInput && !isNaN(parseFloat(priceInput))" class="price-total">
                <text class="price-total-label">预估费用</text>
                <text class="price-total-value">¥{{ calculateTotalCost({ ...log, price_per_kwh: parseFloat(priceInput) }) }}</text>
              </view>
            </view>
            <view v-else class="price-display">
              <template v-if="log.charge_type === 'DC'">
                <text class="info-value" :class="{ 'highlight': log.total_cost }">
                  {{ log.total_cost ? '¥' + log.total_cost.toFixed(2) : '--' }}
                </text>
              </template>
              <template v-else>
                <text class="info-value" :class="{ 'highlight': log.price_per_kwh }">
                  {{ log.price_per_kwh ? log.price_per_kwh.toFixed(2) : '--' }}
                </text>
                <text class="price-unit-static" v-if="log.price_per_kwh">元/kWh</text>
                <view v-if="log.total_cost" class="price-total-display">
                  <text class="price-total-value">¥{{ log.total_cost.toFixed(2) }}</text>
                </view>
              </template>
            </view>
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
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getChargingLogs, updateChargingPrice } from '@/api/charging.js'
import { getChargingAnalysis, triggerChargingAnalysis } from '@/api/ai.js'
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
    } else {
      aiLoading.value = true
      await triggerChargingAnalysis(vin.value, refId)
      setTimeout(async () => {
        const res2 = await getChargingAnalysis(vin.value, refId)
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

const toggleChargeAI = async (log) => {
  if (expandedLogId.value === log.id) {
    expandedLogId.value = null
    return
  }
  expandedLogId.value = log.id

  if (chargingAiMap.value[log.id] || chargingAiLoadingMap.value[log.id]) return

  const refId = `charging:${log.id}`
  chargingAiLoadingMap.value[log.id] = true

  try {
    const res = await getChargingAnalysis(vin.value, refId)
    if (res?.data) {
      chargingAiMap.value[log.id] = res.data
    } else {
      await triggerChargingAnalysis(vin.value, refId)
      setTimeout(async () => {
        const res2 = await getChargingAnalysis(vin.value, refId)
        if (res2?.data) {
          chargingAiMap.value[log.id] = res2.data
        }
        chargingAiLoadingMap.value[log.id] = false
      }, 15000)
      return
    }
  } catch (e) {}
  chargingAiLoadingMap.value[log.id] = false
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

  .info-poi {
    font-size: 20rpx;
    color: var(--color-primary);
    display: block;
    margin-top: 4rpx;
  }

  .info-district {
    font-size: 20rpx;
    color: var(--dark-page-text-hint);
    display: block;
    margin-top: 4rpx;
  }

  &.price-section {
    grid-column: span 2;
    background: var(--dark-page-glass-bg);
    border-radius: 16rpx;
    padding: 16rpx 20rpx;
    margin-top: 8rpx;

    &.editing {
      background: var(--dark-page-icon-wrap-bg);
      border: 1rpx solid var(--color-primary);
    }

    .price-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8rpx;

        .price-edit-btn {
          padding: 6rpx 12rpx;
          border-radius: 8rpx;
          background: var(--color-primary);
          display: flex;
          align-items: center;
          justify-content: center;

          .edit-btn-text {
            font-size: 22rpx;
            color: #fff;
            font-weight: 500;
          }

          &:active {
            opacity: 0.8;
          }
        }
      }

    .price-display {
      display: flex;
      align-items: center;
      gap: 8rpx;
      flex-wrap: wrap;

      .price-unit-static {
        font-size: 22rpx;
        color: var(--dark-page-text-hint);
      }

      .price-total-display {
        margin-left: auto;
        padding: 4rpx 12rpx;
        background: var(--color-primary);
        border-radius: 12rpx;

        .price-total-value {
          font-size: 24rpx;
          color: #fff;
          font-weight: 600;
        }
      }
    }

    .price-edit-area {
      .price-input-wrap {
        display: flex;
        align-items: center;
        gap: 12rpx;
        margin-bottom: 12rpx;

        .price-input {
          flex: 1;
          height: 64rpx;
          background: var(--dark-page-glass-bg);
          border-radius: 12rpx;
          padding: 0 20rpx;
          font-size: 28rpx;
          color: var(--dark-page-text);
          border: 1rpx solid var(--dark-page-glass-border);

          &:focus {
            border-color: var(--color-primary);
          }
        }

        .price-unit {
          font-size: 24rpx;
          color: var(--dark-page-text-secondary);
          white-space: nowrap;
        }
      }

      .price-actions {
        display: flex;
        gap: 16rpx;
        margin-bottom: 12rpx;

        .price-btn {
          flex: 1;
          height: 56rpx;
          border-radius: 12rpx;
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 8rpx;

          .btn-text {
            font-size: 24rpx;
            font-weight: 500;
          }

          &.save {
            background: var(--color-primary);

            .btn-text {
              color: #fff;
            }

            &:active {
              opacity: 0.8;
            }
          }

          &.cancel {
            background: var(--dark-page-glass-bg);
            border: 1rpx solid var(--dark-page-glass-border);

            .btn-text {
              color: var(--dark-page-text-secondary);
            }

            &:active {
              background: var(--dark-page-icon-wrap-bg);
            }
          }
        }
      }

      .price-total {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding-top: 12rpx;
        border-top: 1rpx solid var(--dark-page-glass-border);

        .price-total-label {
          font-size: 22rpx;
          color: var(--dark-page-text-hint);
        }

        .price-total-value {
          font-size: 28rpx;
          color: var(--color-primary);
          font-weight: 700;
        }
      }
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
