<template>
  <view class="day-container" :class="themeClass" :style="{ paddingTop: 'calc(' + statusBarHeight + 'px + 88rpx)' }">
    <NavBar :title="navTitle" />

    <scroll-view scroll-y class="main-scroll">
      <view class="day-summary-card">
        <view class="summary-item">
          <text class="summary-value">{{ events.length }}</text>
          <text class="summary-label">事件数</text>
        </view>
        <view class="summary-divider"></view>
        <view class="summary-item">
          <text class="summary-value">{{ totalVideos }}</text>
          <text class="summary-label">视频数</text>
        </view>
        <view class="summary-divider"></view>
        <view class="summary-item" v-if="typeBreakdown.sentry > 0">
          <text class="summary-value sentry-color">{{ typeBreakdown.sentry }}</text>
          <text class="summary-label">哨兵</text>
        </view>
        <view class="summary-divider" v-if="typeBreakdown.sentry > 0 && typeBreakdown.saved > 0"></view>
        <view class="summary-item" v-if="typeBreakdown.saved > 0">
          <text class="summary-value saved-color">{{ typeBreakdown.saved }}</text>
          <text class="summary-label">已保存</text>
        </view>
      </view>

      <view class="event-list" v-if="events.length > 0">
        <view
          class="event-card"
          v-for="event in events"
          :key="event.id"
          @click="goPlayer(event.id)"
          @longpress="onLongPressEvent(event)"
        >
          <view class="event-left">
            <view class="event-time-block">
              <text class="event-time-text">{{ formatEventTime(event.event_time) }}</text>
            </view>
            <view class="event-type-dot" :style="{ backgroundColor: getTypeColor(event.event_type) }"></view>
          </view>
          <view class="event-info">
            <view class="event-top-row">
              <view class="event-camera-count">
                <Icon name="Videocam" :size="14" :color="detailIconColor" />
                <text class="camera-count-text">{{ getVideoCount(event) }}路</text>
              </view>
            </view>
            <view class="event-detail-row" v-if="event.latitude && event.longitude">
              <Icon name="Location" :size="13" :color="detailIconColor" />
              <text class="detail-text">{{ formatLocation(event.latitude, event.longitude) }}</text>
            </view>
            <view class="event-detail-row" v-if="event.duration">
              <Icon name="Time" :size="13" :color="detailIconColor" />
              <text class="detail-text">{{ formatDuration(event.duration) }}</text>
            </view>
          </view>
          <view class="event-arrow">
            <Icon name="ChevronForward" :size="18" :color="arrowColor" />
          </view>
        </view>
      </view>

      <view class="empty-state" v-else-if="!loading">
        <view class="empty-icon">
          <Icon name="VideocamOutline" :size="64" :color="emptyIconColor" />
        </view>
        <text class="empty-text">当天无事件记录</text>
      </view>

      <view class="loading-state" v-if="loading">
        <view class="loading-spinner"></view>
        <text class="loading-text">加载中...</text>
      </view>

      <view class="bottom-spacer"></view>
    </scroll-view>

    <view class="scan-result-mask" v-if="showDeleteConfirm" @click="cancelDelete">
      <view class="delete-confirm-modal" @click.stop>
        <view class="delete-confirm-icon">
          <Icon name="Trash" :size="32" color="#EF4444" />
        </view>
        <text class="delete-confirm-title">确认删除</text>
        <text class="delete-confirm-desc">将删除此事件的所有视频文件，此操作不可恢复</text>
        <view class="delete-confirm-btns">
          <view class="delete-confirm-btn cancel" @click="cancelDelete">
            <text class="delete-confirm-btn-text">取消</text>
          </view>
          <view class="delete-confirm-btn danger" @click="confirmDelete">
            <text class="delete-confirm-btn-text danger-text">删除</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import NavBar from '@/components/NavBar/NavBar.vue'
import Icon from '@/components/Icon/Icon.vue'
import { useThemeStore } from '@/store/theme'
import { initDB, getEventsWithVideoCount, selectSql, deleteEvent } from '@/utils/dashcam-db.js'
import { waitForPlus, scanLocalVideos, deleteImportedEvent } from '@/utils/dashcam-scanner.js'
import { batchFuseEvents } from '@/utils/dashcam-gps.js'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const statusBarHeight = uni.getSystemInfoSync().statusBarHeight || 0
const detailIconColor = computed(() => themeStore.colors.hint)
const arrowColor = computed(() => themeStore.colors.chevron)
const emptyIconColor = computed(() => themeStore.colors.inactiveIcon)

const dateStr = ref('')
const eventType = ref('')
const events = ref([])
const loading = ref(false)
const showDeleteConfirm = ref(false)
const deleteTarget = ref(null)

const navTitle = computed(() => {
  if (!dateStr.value) return '日期详情'
  const parts = dateStr.value.split('-')
  if (parts.length === 3) return `${parts[0]}年${parseInt(parts[1])}月${parseInt(parts[2])}日`
  return dateStr.value
})

const totalVideos = computed(() => {
  let count = 0
  for (const e of events.value) {
    count += getVideoCount(e)
  }
  return count
})

const typeBreakdown = computed(() => {
  const map = { recent: 0, saved: 0, sentry: 0 }
  for (const e of events.value) {
    const t = e.event_type || 'recent'
    if (map[t] !== undefined) map[t]++
  }
  return map
})

const TYPE_COLORS = {
  recent: '#60a5fa',
  saved: '#5BE7C4',
  sentry: '#f97316'
}

const TYPE_LABELS = {
  recent: '最近片段',
  saved: '已保存',
  sentry: '哨兵事件'
}

const getTypeColor = (type) => TYPE_COLORS[type] || '#60a5fa'
const getTypeLabel = (type) => TYPE_LABELS[type] || type

const getVideoCount = (event) => {
  return event.video_count || (event.videos ? event.videos.length : 0) || 0
}

const formatEventTime = (ts) => {
  if (!ts) return ''
  const d = new Date(ts)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const formatDuration = (seconds) => {
  if (!seconds) return ''
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return m > 0 ? `${m}分${s}秒` : `${s}秒`
}

const formatLocation = (lat, lng) => {
  if (!lat || !lng) return ''
  return `${lat.toFixed(4)}, ${lng.toFixed(4)}`
}

const loadDayEvents = async () => {
  if (!dateStr.value) return
  loading.value = true
  events.value = []

  try {
    const dayStart = new Date(dateStr.value + 'T00:00:00').getTime()
    const dayEnd = new Date(dateStr.value + 'T23:59:59.999').getTime()

    const opts = { startTime: dayStart, endTime: dayEnd, limit: 200 }
    if (eventType.value) opts.eventType = eventType.value

    const res = await getEventsWithVideoCount(opts)
    if (res && res.length > 0) {
      events.value = res
      try {
        const fused = await batchFuseEvents(events.value)
        events.value = fused
      } catch (e) {}
    } else {
      const localVids = await scanLocalVideos()
      const matched = localVids.filter(v => {
        if (eventType.value && v.eventType !== eventType.value) return false
        const match = v.name.match(/^(\d{4})-(\d{2})-(\d{2})/)
        if (!match) return false
        const vDate = match[1] + '-' + match[2] + '-' + match[3]
        return vDate === dateStr.value
      })
      if (matched.length > 0) {
        events.value = groupLocalVideosToEvents(matched)
      }
    }
  } catch (e) {
    console.error('[Dashcam:day] loadDayEvents error:', e)
  } finally {
    loading.value = false
  }
}

const groupLocalVideosToEvents = (videos) => {
  const eventMap = {}
  for (const v of videos) {
    const match = v.name.match(/^(\d{4})-(\d{2})-(\d{2})_(\d{2})-(\d{2})-(\d{2})/)
    if (!match) continue
    const timeKey = match[1] + '-' + match[2] + '-' + match[3] + '_' + match[4] + '-' + match[5] + '-' + match[6]
    const et = v.eventType || 'recent'
    const mapKey = et + '_' + timeKey
    if (!eventMap[mapKey]) {
      const ts = new Date(
        Number(match[1]), Number(match[2]) - 1, Number(match[3]),
        Number(match[4]), Number(match[5]), Number(match[6])
      ).getTime()
      eventMap[mapKey] = {
        id: 'local_' + mapKey,
        event_type: et,
        event_time: ts,
        videos: [],
        thumbnail: v.path,
        imported: 1,
        _isLocal: true
      }
    }
    eventMap[mapKey].videos.push({
      camera: v.camera,
      file_path: v.path,
      file_size: v.size
    })
  }
  return Object.values(eventMap).sort((a, b) => b.event_time - a.event_time)
}

const goPlayer = (id) => {
  uni.navigateTo({ url: '/pages/dashcam/player?id=' + encodeURIComponent(id) })
}

const onLongPressEvent = (event) => {
  deleteTarget.value = event
  showDeleteConfirm.value = true
}

const confirmDelete = async () => {
  const target = deleteTarget.value
  if (!target) return
  showDeleteConfirm.value = false

  const isLocal = String(target.id).startsWith('local_')
  try {
    if (isLocal) {
      if (target.videos && target.videos.length) {
        for (const v of target.videos) {
          if (v.file_path) {
            try {
              const rawPath = v.file_path.replace(/^file:\/\//, '')
              const File = plus.android.importClass('java.io.File')
              const f = new File(rawPath)
              plus.android.invoke(f, 'delete')
            } catch (e) {}
          }
        }
      }
    } else {
      const paths = await deleteEvent(target.id)
      for (const p of paths) {
        try {
          const rawPath = p.replace(/^file:\/\//, '')
          const File = plus.android.importClass('java.io.File')
          const f = new File(rawPath)
          plus.android.invoke(f, 'delete')
        } catch (e) {}
      }
    }
    events.value = events.value.filter(e => e.id !== target.id)
    uni.showToast({ title: '已删除', icon: 'success' })
  } catch (e) {
    console.error('[Dashcam:day] delete failed:', e)
    uni.showToast({ title: '删除失败', icon: 'none' })
  }
  deleteTarget.value = null
}

const cancelDelete = () => {
  showDeleteConfirm.value = false
  deleteTarget.value = null
}

let _initDone = false
let _initPromise = null

const ensureInit = async () => {
  if (_initDone) return
  if (!_initPromise) {
    _initPromise = (async () => {
      await waitForPlus()
      await initDB()
      _initDone = true
    })()
  }
  return _initPromise
}

onLoad((options) => {
  dateStr.value = options?.date || ''
  eventType.value = options?.type || ''
})

onMounted(async () => {
  await ensureInit()
  loadDayEvents()
})

onShow(() => {
  if (_initDone) loadDayEvents()
})
</script>

<style lang="scss" scoped>
.day-container {
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

.day-summary-card {
  display: flex;
  align-items: center;
  justify-content: space-around;
  background: var(--bg-card);
  border-radius: 28rpx;
  padding: 28rpx 24rpx;
  margin-bottom: 24rpx;
  box-shadow: var(--shadow-card);
}

.summary-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6rpx;
  flex: 1;
}

.summary-value {
  font-size: 36rpx;
  font-weight: 800;
  color: var(--color-primary);
}

.sentry-color {
  color: #f97316;
}

.saved-color {
  color: #5BE7C4;
}

.summary-label {
  font-size: 22rpx;
  color: var(--text-tertiary);
}

.summary-divider {
  width: 1rpx;
  height: 50rpx;
  background: var(--border-divider);
}

.event-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.event-card {
  display: flex;
  align-items: center;
  gap: 20rpx;
  background: var(--bg-card);
  border-radius: 24rpx;
  padding: 24rpx;
  box-shadow: var(--shadow-card);

  &:active {
    opacity: 0.9;
  }
}

.event-left {
  display: flex;
  align-items: center;
  gap: 16rpx;
  flex-shrink: 0;
}

.event-time-block {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-width: 200rpx;
}

.event-time-text {
  font-size: 26rpx;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1.2;
}

.event-type-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 6rpx;
  flex-shrink: 0;
}

.event-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.event-top-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.event-type-label {
  font-size: 26rpx;
  font-weight: 600;
}

.event-camera-count {
  display: flex;
  align-items: center;
  gap: 6rpx;
}

.camera-count-text {
  font-size: 22rpx;
  color: var(--text-tertiary);
}

.event-detail-row {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.detail-text {
  font-size: 22rpx;
  color: var(--text-tertiary);
}

.event-arrow {
  flex-shrink: 0;
  display: flex;
  align-items: center;
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

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
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

.scan-result-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  padding: 40rpx;
}

.delete-confirm-modal {
  width: 560rpx;
  background: var(--dark-page-card, #1F2937);
  border-radius: 28rpx;
  padding: 48rpx 40rpx 36rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20rpx;
  border: 1rpx solid var(--dark-page-card-border, rgba(255, 255, 255, 0.1));
}

.delete-confirm-icon {
  width: 96rpx;
  height: 96rpx;
  border-radius: 50%;
  background: rgba(239, 68, 68, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 8rpx;
}

.delete-confirm-title {
  font-size: 34rpx;
  font-weight: 700;
  color: var(--dark-page-text, #fff);
}

.delete-confirm-desc {
  font-size: 26rpx;
  color: var(--dark-page-text-secondary, rgba(255, 255, 255, 0.6));
  text-align: center;
  line-height: 1.5;
}

.delete-confirm-btns {
  display: flex;
  gap: 20rpx;
  width: 100%;
  margin-top: 16rpx;
}

.delete-confirm-btn {
  flex: 1;
  height: 80rpx;
  border-radius: 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;

  &.cancel {
    background: var(--dark-page-glass-bg, rgba(255, 255, 255, 0.06));
  }

  &.danger {
    background: #EF4444;
  }
}

.delete-confirm-btn-text {
  font-size: 28rpx;
  color: var(--dark-page-text-secondary, rgba(255, 255, 255, 0.6));
  font-weight: 500;

  &.danger-text {
    color: #fff;
    font-weight: 600;
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
