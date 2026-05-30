<template>
  <view class="timeline-container" :class="themeClass" :style="{ paddingTop: 'calc(' + statusBarHeight + 'px + 88rpx)' }">
    <NavBar title="时间线" />

    <scroll-view scroll-y class="main-scroll">
      <view class="filter-bar">
        <view
          class="filter-item"
          :class="{ active: filter === 'all' }"
          @click="filter = 'all'; loadTimeline()"
        >
          <text class="filter-text">全部</text>
        </view>
        <view
          class="filter-item"
          :class="{ active: filter === 'recent' }"
          @click="filter = 'recent'; loadTimeline()"
        >
          <text class="filter-text">最近</text>
        </view>
        <view
          class="filter-item"
          :class="{ active: filter === 'sentry' }"
          @click="filter = 'sentry'; loadTimeline()"
        >
          <text class="filter-text">哨兵</text>
        </view>
        <view
          class="filter-item"
          :class="{ active: filter === 'saved' }"
          @click="filter = 'saved'; loadTimeline()"
        >
          <text class="filter-text">已保存</text>
        </view>
      </view>

      <view class="timeline-body" v-if="groupedEvents.length > 0">
        <view class="date-group" v-for="group in groupedEvents" :key="group.date">
          <view class="date-header">
            <view class="date-dot-wrap">
              <view class="date-dot"></view>
              <view class="date-line"></view>
            </view>
            <view class="date-info">
              <text class="date-label">{{ group.dateLabel }}</text>
              <text class="date-count">{{ group.events.length }} 个事件</text>
            </view>
          </view>

          <view class="event-item" v-for="event in group.events" :key="event.id" @click="goPlayer(event.id)">
            <view class="timeline-dot-wrap">
              <view class="timeline-dot" :style="{ backgroundColor: getTypeColor(event.event_type) }"></view>
              <view class="timeline-line"></view>
            </view>
            <view class="event-card-tl">
              <view class="event-top-tl">
                <view class="event-time-tl">
                  <Icon name="Time" :size="14" :color="timeIconColor" />
                  <text class="time-text">{{ formatTimeOnly(event.event_time) }}</text>
                </view>
                <view class="event-type-badge" :style="{ backgroundColor: getTypeColor(event.event_type) + '20', borderColor: getTypeColor(event.event_type) }">
                  <text class="badge-text-tl" :style="{ color: getTypeColor(event.event_type) }">{{ getTypeLabel(event.event_type) }}</text>
                </view>
              </view>

              <view class="event-body-tl">
                <view class="event-thumb-tl" v-if="event.thumbnail">
                  <image :src="event.thumbnail" class="thumb-img-tl" mode="aspectFill" />
                </view>
                <view class="event-details-tl">
                  <view class="detail-item-tl" v-if="event.duration">
                    <Icon name="Timer" :size="13" :color="detailIconColor" />
                    <text class="detail-text-tl">{{ formatDuration(event.duration) }}</text>
                  </view>
                  <view class="detail-item-tl" v-if="event.latitude && event.longitude">
                    <Icon name="Location" :size="13" :color="detailIconColor" />
                    <text class="detail-text-tl">{{ event.latitude.toFixed(4) }}, {{ event.longitude.toFixed(4) }}</text>
                  </view>
                </view>
              </view>

              <view class="event-actions-tl">
                <view class="action-chip" @click.stop="goMap(event)">
                  <Icon name="Map" :size="13" :color="mapChipColor" />
                  <text class="action-chip-text">地图</text>
                </view>
                <view class="action-chip play-chip" @click.stop="goPlayer(event.id)">
                  <Icon name="Play" :size="13" color="#fff" />
                  <text class="action-chip-text white">播放</text>
                </view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <view class="empty-state" v-else-if="!loading">
        <view class="empty-icon">
          <Icon name="TimeOutline" :size="64" :color="emptyIconColor" />
        </view>
        <text class="empty-text">暂无时间线记录</text>
        <text class="empty-sub">导入行车记录仪视频后查看时间线</text>
      </view>

      <view class="loading-state" v-if="loading">
        <view class="loading-spinner"></view>
        <text class="loading-text">加载中...</text>
      </view>

      <view class="bottom-spacer"></view>
    </scroll-view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import NavBar from '@/components/NavBar/NavBar.vue'
import Icon from '@/components/Icon/Icon.vue'
import { useThemeStore } from '@/store/theme'
import { initDB, getEvents } from '@/utils/dashcam-db.js'
import { batchFuseEvents } from '@/utils/dashcam-gps.js'
import { waitForPlus } from '@/utils/dashcam-scanner.js'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const statusBarHeight = uni.getSystemInfoSync().statusBarHeight || 0
const timeIconColor = computed(() => themeStore.colors.hint)
const detailIconColor = computed(() => themeStore.colors.hint)
const mapChipColor = computed(() => themeStore.colors.primary)
const emptyIconColor = computed(() => themeStore.colors.inactiveIcon)

const filter = ref('all')
const events = ref([])
const loading = ref(false)

const TYPE_COLORS = {
  recent: '#60a5fa',
  saved: '#5BE7C4',
  sentry: '#f97316'
}

const TYPE_LABELS = {
  recent: '最近',
  saved: '已保存',
  sentry: '哨兵'
}

const getTypeColor = (type) => TYPE_COLORS[type] || '#60a5fa'
const getTypeLabel = (type) => TYPE_LABELS[type] || type

const groupedEvents = computed(() => {
  if (!events.value.length) return []
  const groups = {}
  events.value.forEach(evt => {
    const d = new Date(evt.event_time)
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    if (!groups[key]) {
      groups[key] = {
        date: key,
        dateLabel: formatDateLabel(d),
        events: []
      }
    }
    groups[key].events.push(evt)
  })
  return Object.values(groups).sort((a, b) => b.date.localeCompare(a.date))
})

const formatDateLabel = (d) => {
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  const isToday = d.toDateString() === today.toDateString()
  const isYesterday = d.toDateString() === yesterday.toDateString()
  const pad = (n) => String(n).padStart(2, '0')
  const base = `${d.getMonth() + 1}月${d.getDate()}日`
  if (isToday) return `今天 ${base}`
  if (isYesterday) return `昨天 ${base}`
  const weekDays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return `${weekDays[d.getDay()]} ${base}`
}

const formatTimeOnly = (ts) => {
  if (!ts) return ''
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

const formatDuration = (seconds) => {
  if (!seconds) return ''
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return m > 0 ? `${m}分${s}秒` : `${s}秒`
}

const loadTimeline = async () => {
  console.log('[Dashcam:timeline] loadTimeline start, clearing old data')
  events.value = []
  loading.value = true
  try {
    const options = { limit: 200 }
    if (filter.value !== 'all') {
      options.eventType = filter.value
    }
    let list = await getEvents(options)
    list = list || []
    console.log('[Dashcam:timeline] getEvents returned', list.length, 'events')
    try {
      list = await batchFuseEvents(list)
    } catch (e) {}
    events.value = list
    console.log('[Dashcam:timeline] final list:', list.length, 'events')
  } catch (e) {
    console.error('[Dashcam:timeline] loadTimeline error:', e)
    events.value = []
  } finally {
    loading.value = false
  }
}

const goPlayer = (id) => {
  uni.navigateTo({ url: '/pages/dashcam/player?id=' + id })
}

const goMap = (event) => {
  if (event.latitude && event.longitude) {
    uni.navigateTo({
      url: `/pages/dashcam/map?lat=${event.latitude}&lng=${event.longitude}`
    })
  } else {
    uni.navigateTo({ url: '/pages/dashcam/map' })
  }
}

let _tlInitDone = false
let _tlInitPromise = null
let _tlLastLoadTs = 0

const tlEnsureInit = async () => {
  if (_tlInitDone) return
  if (!_tlInitPromise) {
    _tlInitPromise = (async () => {
      console.log('[Dashcam:timeline] waiting for plus ready...')
      await waitForPlus()
      console.log('[Dashcam:timeline] plus ready, initializing DB')
      await initDB()
      _tlInitDone = true
      console.log('[Dashcam:timeline] DB init complete')
    })()
  }
  return _tlInitPromise
}

const tlSafeReload = () => {
  const now = Date.now()
  if (now - _tlLastLoadTs < 1500) {
    console.log('[Dashcam:timeline] skip reload, too frequent')
    return
  }
  _tlLastLoadTs = now
  loadTimeline()
}

onMounted(async () => {
  console.log('[Dashcam:timeline] onMounted')
  try {
    await tlEnsureInit()
  } catch (e) {
    console.error('[Dashcam:timeline] init error:', e)
  }
  tlSafeReload()
})

onShow(() => {
  console.log('[Dashcam:timeline] onShow, reloading data')
  tlEnsureInit().then(() => {
    tlSafeReload()
  })
})
</script>

<style lang="scss" scoped>
.timeline-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
  background: var(--bg-page);
  padding: 0 24rpx 40rpx;
  display: flex;
  flex-direction: column;
}

.main-scroll {
  flex: 1;
  overflow: hidden;
}

.filter-bar {
  display: flex;
  gap: 12rpx;
  margin-bottom: 28rpx;
  background: var(--bg-card);
  border-radius: 20rpx;
  padding: 8rpx;
  box-shadow: var(--shadow-card);
}

.filter-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 60rpx;
  border-radius: 16rpx;
  transition: all 0.25s ease;

  &.active {
    background: var(--bg-filter-active);

    .filter-text {
      color: var(--text-filter-active);
      font-weight: 700;
    }
  }
}

.filter-text {
  font-size: 24rpx;
  color: var(--text-filter);
  font-weight: 500;
}

.date-group {
  margin-bottom: 8rpx;
}

.date-header {
  display: flex;
  align-items: center;
  gap: 16rpx;
  padding: 16rpx 0;
}

.date-dot-wrap {
  position: relative;
  width: 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.date-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 50%;
  background: var(--color-primary);
  z-index: 2;
}

.date-line {
  position: absolute;
  top: 16rpx;
  bottom: -16rpx;
  left: 50%;
  width: 2rpx;
  background: var(--border-divider);
  transform: translateX(-50%);
}

.date-info {
  display: flex;
  align-items: baseline;
  gap: 12rpx;
}

.date-label {
  font-size: 28rpx;
  font-weight: 700;
  color: var(--text-primary);
}

.date-count {
  font-size: 22rpx;
  color: var(--text-tertiary);
}

.event-item {
  display: flex;
  gap: 16rpx;
  padding: 0 0 0 0;
}

.timeline-dot-wrap {
  position: relative;
  width: 40rpx;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 24rpx;
  flex-shrink: 0;
}

.timeline-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;
  z-index: 2;
}

.timeline-line {
  position: absolute;
  top: 36rpx;
  bottom: -12rpx;
  left: 50%;
  width: 2rpx;
  background: var(--border-divider);
  transform: translateX(-50%);
}

.event-card-tl {
  flex: 1;
  background: var(--bg-card);
  border-radius: 20rpx;
  padding: 20rpx 24rpx;
  margin-bottom: 16rpx;
  box-shadow: var(--shadow-card);
}

.event-top-tl {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16rpx;
}

.event-time-tl {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.time-text {
  font-size: 26rpx;
  font-weight: 600;
  color: var(--text-primary);
}

.event-type-badge {
  padding: 4rpx 14rpx;
  border-radius: 8rpx;
  border: 1rpx solid;
}

.badge-text-tl {
  font-size: 20rpx;
  font-weight: 600;
}

.event-body-tl {
  display: flex;
  gap: 16rpx;
  margin-bottom: 16rpx;
}

.event-thumb-tl {
  width: 120rpx;
  height: 80rpx;
  border-radius: 12rpx;
  overflow: hidden;
  flex-shrink: 0;
}

.thumb-img-tl {
  width: 100%;
  height: 100%;
}

.event-details-tl {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
  justify-content: center;
}

.detail-item-tl {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.detail-text-tl {
  font-size: 22rpx;
  color: var(--text-tertiary);
}

.event-actions-tl {
  display: flex;
  gap: 12rpx;
}

.action-chip {
  display: flex;
  align-items: center;
  gap: 6rpx;
  padding: 8rpx 20rpx;
  border-radius: 20rpx;
  background: var(--bg-card-secondary);
  border: 1rpx solid var(--border-card);

  &.play-chip {
    background: var(--gradient);
    border: none;
  }
}

.action-chip-text {
  font-size: 22rpx;
  color: var(--text-secondary);
  font-weight: 500;

  &.white {
    color: #fff;
  }
}

.empty-state {
  text-align: center;
  padding: 120rpx 40rpx;
  background: var(--bg-card);
  border-radius: 28rpx;
}

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

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80rpx 40rpx;
  gap: 20rpx;
}

.loading-spinner {
  width: 48rpx;
  height: 48rpx;
  border: 4rpx solid var(--bg-spinner-track);
  border-top-color: var(--color-spinner);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.loading-text {
  font-size: 26rpx;
  color: var(--text-tertiary);
}

.bottom-spacer {
  height: 40rpx;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
