<template>
  <view class="page" :class="themeClass">
    <NavBar title="车辆AI分析" />

    <scroll-view scroll-y class="main-scroll" @scrolltolower="loadMore">
      <view class="report-list" v-if="list.length > 0">
        <view class="report-card" v-for="item in list" :key="item.id" @click="toggleExpand(item.id)">
          <view class="report-header">
            <view class="report-type-badge daily">
              <Icon name="Sparkles" :size="12" color="#fff" />
              <text class="badge-text">日报</text>
            </view>
            <text class="report-ref">{{ formatRefId(item.ref_id) }}</text>
            <view class="report-expand">
              <Icon :name="expandedId === item.id ? 'ChevronUp' : 'ChevronDown'" :size="16" themeColor="hint" />
            </view>
          </view>
          <text class="report-time">{{ formatTime(item.created_at) }}</text>
          <view class="report-summary-row" @click.stop="goToDetail">
            <text class="report-summary-text">{{ item.summary || '点击查看分析报告' }}</text>
            <view class="report-go-btn">
              <text class="report-go-text">车辆详情</text>
              <Icon name="ChevronForward" :size="14" themeColor="hint" />
            </view>
          </view>
          <view class="report-body" v-if="expandedId === item.id">
            <text class="report-line" v-for="(line, i) in getLines(item)" :key="i">{{ line }}</text>
          </view>
        </view>
      </view>

      <view class="loading-state" v-if="loading">
        <view class="ai-spinner"></view>
        <text class="loading-text">加载中...</text>
      </view>

      <view class="empty-state" v-if="!loading && list.length === 0">
        <Icon name="Sparkles" :size="48" themeColor="hint" />
        <text class="empty-text">暂无车辆AI分析报告</text>
        <text class="empty-hint">每日早上8点自动生成分析报告</text>
      </view>

      <view class="no-more" v-if="!loading && list.length > 0 && noMore">
        <text class="no-more-text">没有更多了</text>
      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getAnalysisList } from '@/api/ai.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import { useThemeStore } from '@/store/theme'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const hintColor = computed(() => themeStore.colors.hint)

const vin = ref('')
const list = ref([])
const loading = ref(false)
const page = ref(1)
const noMore = ref(false)
const expandedId = ref(null)

onLoad((options) => {
  vin.value = options?.vin || ''
  if (vin.value) loadList()
})

const loadList = async () => {
  if (loading.value || noMore.value) return
  loading.value = true
  try {
    const res = await getAnalysisList(vin.value, 'vehicle', page.value, 20)
    const newList = res?.data?.list || []
    if (page.value === 1) {
      list.value = newList
    } else {
      list.value = [...list.value, ...newList]
    }
    if (newList.length < 20) noMore.value = true
  } catch (e) {}
  loading.value = false
}

const loadMore = () => {
  if (!noMore.value) { page.value++; loadList() }
}

const toggleExpand = (id) => {
  expandedId.value = expandedId.value === id ? null : id
}

const getLines = (item) => {
  if (!item?.result) return []
  return item.result.split('\n').filter(l => l.trim()).map(l => l.replace(/^#{1,3}\s*/, '').replace(/\*\*/g, '').replace(/^[-*]\s*/, '• ').trim())
}

const goToDetail = () => {
  uni.navigateTo({ url: `/pages/vehicle/detail` })
}

const formatRefId = (refId) => {
  if (refId.startsWith('vehicle_daily:')) return refId.replace('vehicle_daily:', '')
  return refId
}

const formatTime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}
</script>

<style lang="scss" scoped>
.page {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: linear-gradient(180deg, var(--dark-page-bg) 0%, var(--dark-page-icon-wrap-bg) 100%);
  display: flex;
  flex-direction: column;
  padding-top: calc(var(--status-bar-height, 44px) + 88rpx);
}

.main-scroll { flex: 1; overflow: hidden; padding: 0 32rpx 40rpx; }

.report-card {
  background: var(--dark-page-icon-wrap-bg); border-radius: 20rpx; padding: 24rpx 28rpx; margin-bottom: 16rpx;

  .report-header {
    display: flex; align-items: center; gap: 12rpx;
    .report-type-badge {
      display: flex; align-items: center; gap: 4rpx;
      padding: 4rpx 14rpx; border-radius: 12rpx;
      &.daily { background: linear-gradient(135deg, #f59e0b, #fbbf24); }
      .badge-text { font-size: 20rpx; color: #fff; font-weight: 600; }
    }
    .report-ref { font-size: 26rpx; color: var(--dark-page-text); font-weight: 600; }
    .report-expand { padding: 8rpx; flex-shrink: 0; }
  }

  .report-time { font-size: 22rpx; color: var(--dark-page-text-hint); display: block; margin-top: 8rpx; }

  .report-body {
    margin-top: 16rpx; padding-top: 16rpx; border-top: 1rpx solid var(--dark-page-glass-border);
    .report-line { font-size: 26rpx; color: var(--dark-page-text-secondary); line-height: 1.8; display: block; }
  }

  .report-summary-row {
    margin-top: 12rpx;
    padding: 16rpx 20rpx;
    background: var(--dark-page-glass-bg);
    border-radius: 16rpx;
    display: flex;
    align-items: center;
    gap: 12rpx;

    .report-summary-text {
      font-size: 26rpx;
      color: var(--dark-page-text-secondary);
      line-height: 1.6;
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .report-go-btn {
      display: flex;
      align-items: center;
      gap: 4rpx;
      flex-shrink: 0;
      padding: 6rpx 14rpx;
      background: var(--dark-page-glass-bg);
      border-radius: 14rpx;

      .report-go-text {
        font-size: 20rpx;
        color: var(--dark-page-text-hint);
        font-weight: 500;
      }
    }
  }
}

.loading-state {
  display: flex; flex-direction: column; align-items: center; padding: 60rpx 0;
  .ai-spinner {
    width: 36rpx; height: 36rpx;
    border: 3rpx solid var(--dark-page-glass-border); border-top-color: var(--color-primary);
    border-radius: 50%; animation: spin 0.8s linear infinite;
  }
  .loading-text { font-size: 26rpx; color: var(--dark-page-text-hint); margin-top: 16rpx; }
}

@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

.empty-state {
  display: flex; flex-direction: column; align-items: center; padding: 160rpx 60rpx;
  .empty-text { font-size: 30rpx; color: var(--dark-page-text); font-weight: 600; margin-top: 24rpx; }
  .empty-hint { font-size: 24rpx; color: var(--dark-page-text-hint); margin-top: 12rpx; }
}

.no-more { text-align: center; padding: 32rpx 0; .no-more-text { font-size: 24rpx; color: var(--dark-page-text-hint); } }
</style>
