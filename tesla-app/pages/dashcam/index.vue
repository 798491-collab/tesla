<template>
  <view class="dashcam-container" :class="themeClass" :style="{ paddingTop: 'calc(' + statusBarHeight + 'px + 88rpx)' }">
    <NavBar title="行车记录仪" />

    <scroll-view scroll-y class="main-scroll">
      <view class="stats-card">
        <view class="stat-item">
          <text class="stat-value">{{ stats.eventCount }}</text>
          <text class="stat-label">已导入事件</text>
        </view>
        <view class="stat-divider"></view>
        <view class="stat-item">
          <text class="stat-value">{{ stats.videoCount }}</text>
          <text class="stat-label">视频数</text>
        </view>
        <view class="stat-divider"></view>
        <view class="stat-item">
          <text class="stat-value">{{ formatSize(stats.totalSize) }}</text>
          <text class="stat-label">占用空间</text>
        </view>
      </view>

      <view class="tab-bar">
        <view
          class="tab-item"
          :class="{ active: activeTab === 'recent' }"
          @click="switchTab('recent')"
        >
          <text class="tab-text">最近事件</text>
        </view>
        <view
          class="tab-item"
          :class="{ active: activeTab === 'sentry' }"
          @click="switchTab('sentry')"
        >
          <text class="tab-text">哨兵事件</text>
        </view>
        <view
          class="tab-item"
          :class="{ active: activeTab === 'saved' }"
          @click="switchTab('saved')"
        >
          <text class="tab-text">已保存事件</text>
        </view>
      </view>

      <view class="event-list" v-if="events.length > 0">
        <view
          class="event-card"
          v-for="event in events"
          :key="event.id"
          @click="goPlayer(event.id)"
        >
          <view class="event-thumb">
            <image
              v-if="event.thumbnail"
              :src="event.thumbnail"
              class="thumb-img"
              mode="aspectFill"
            />
            <view v-else class="thumb-placeholder">
              <Icon name="Videocam" :size="28" :color="thumbIconColor" />
            </view>
          </view>
          <view class="event-info">
            <view class="event-top">
              <text class="event-time">{{ formatEventTime(event.event_time) }}</text>
              <view class="event-type-tag" :style="{ backgroundColor: getTypeColor(event.event_type) + '20', borderColor: getTypeColor(event.event_type) }">
                <text class="event-type-text" :style="{ color: getTypeColor(event.event_type) }">{{ getTypeLabel(event.event_type) }}</text>
              </view>
            </view>
            <view class="event-detail">
              <view class="detail-row" v-if="event.duration">
                <Icon name="Time" :size="14" :color="detailIconColor" />
                <text class="detail-text">{{ formatDuration(event.duration) }}</text>
              </view>
              <view class="detail-row" v-if="event.latitude && event.longitude">
                <Icon name="Location" :size="14" :color="detailIconColor" />
                <text class="detail-text">{{ formatLocation(event.latitude, event.longitude) }}</text>
              </view>
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
        <text class="empty-text">暂无事件记录</text>
        <text class="empty-sub">扫描U盘导入行车记录仪视频</text>
      </view>

      <view class="loading-state" v-if="loading">
        <view class="loading-spinner"></view>
        <text class="loading-text">{{ loadingText }}</text>
      </view>

      <view class="bottom-spacer"></view>
    </scroll-view>

    <view class="action-bar">
      <view class="action-btn scan-btn" @click="handleScanUSB">
        <Icon name="Usb" :size="20" color="#fff" />
        <text class="action-btn-text">扫描U盘</text>
      </view>
      <view class="action-btn import-btn" @click="handleImport" :class="{ disabled: scannedEvents.length === 0 }">
        <Icon name="Download" :size="20" color="#fff" />
        <text class="action-btn-text">导入视频</text>
        <view class="import-badge" v-if="scannedEvents.length > 0">
          <text class="badge-text">{{ scannedEvents.length }}</text>
        </view>
      </view>
    </view>

    <view class="scan-result-mask" v-if="showScanResult" @click="showScanResult = false">
      <view class="scan-result-modal" @click.stop>
        <view class="modal-header">
          <text class="modal-title">扫描结果</text>
          <view class="modal-close" @click="showScanResult = false">
            <Icon name="Close" :size="20" :color="modalCloseColor" />
          </view>
        </view>
        <scroll-view scroll-y class="modal-scroll">
          <view class="scan-summary">
            <text class="scan-summary-text">共发现 {{ scannedEvents.length }} 个事件</text>
          </view>
          <view class="scan-event-item" v-for="(item, idx) in scannedEvents" :key="idx">
            <view class="scan-event-type" :style="{ backgroundColor: getTypeColor(item.eventType) + '20' }">
              <text class="scan-event-type-text" :style="{ color: getTypeColor(item.eventType) }">{{ getTypeLabel(item.eventType) }}</text>
            </view>
            <view class="scan-event-info">
              <text class="scan-event-name">{{ item.fileName }}</text>
              <text class="scan-event-camera">{{ item.camera }}</text>
            </view>
          </view>
        </scroll-view>
        <view class="modal-footer">
          <view class="modal-btn cancel-btn" @click="showScanResult = false">
            <text class="modal-btn-text">取消</text>
          </view>
          <view class="modal-btn confirm-btn" @click="startImport">
            <text class="modal-btn-text confirm-text">开始导入</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import NavBar from '@/components/NavBar/NavBar.vue'
import Icon from '@/components/Icon/Icon.vue'
import { useThemeStore } from '@/store/theme'
import { initDB, getEvents, getStorageStats, cleanOldRecentClips, insertEvent, insertVideo, updateEvent } from '@/utils/dashcam-db.js'
import { selectTeslaCamDir, scanTeslaCam, importEvent, generateThumbnail } from '@/utils/dashcam-scanner.js'
import { batchFuseEvents } from '@/utils/dashcam-gps.js'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)

const statusBarHeight = uni.getSystemInfoSync().statusBarHeight || 0

const thumbIconColor = computed(() => themeStore.colors.inactiveIcon)
const detailIconColor = computed(() => themeStore.colors.hint)
const arrowColor = computed(() => themeStore.colors.chevron)
const emptyIconColor = computed(() => themeStore.colors.inactiveIcon)
const modalCloseColor = computed(() => themeStore.colors.hint)

const activeTab = ref('recent')
const events = ref([])
const stats = ref({ eventCount: 0, videoCount: 0, trackCount: 0, totalSize: 0 })
const loading = ref(false)
const loadingText = ref('加载中...')
const scannedEvents = ref([])
const showScanResult = ref(false)

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

const formatSize = (bytes) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

const switchTab = (tab) => {
  activeTab.value = tab
  loadEvents()
}

const loadStats = () => {
  getStorageStats().then((res) => {
    stats.value = res
  }).catch(() => {})
}

const loadEvents = () => {
  loading.value = true
  loadingText.value = '加载中...'
  getEvents({ eventType: activeTab.value, limit: 50 }).then((res) => {
    events.value = res || []
    return batchFuseEvents(events.value)
  }).then((fused) => {
    events.value = fused
  }).catch(() => {
    events.value = []
  }).finally(() => {
    loading.value = false
  })
}

const handleScanUSB = async () => {
  loading.value = true
  loadingText.value = '选择TeslaCam目录...'
  try {
    const treeUri = await selectTeslaCamDir()
    if (!treeUri) {
      loading.value = false
      return
    }
    loadingText.value = '扫描中...'
    const result = await scanTeslaCam(treeUri)
    scannedEvents.value = result.events || []
    if (scannedEvents.value.length > 0) {
      showScanResult.value = true
    } else {
      uni.showToast({ title: '未发现行车记录仪视频', icon: 'none' })
    }
  } catch (e) {
    uni.showToast({ title: e.message || '扫描失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

const handleImport = () => {
  if (scannedEvents.value.length === 0) {
    uni.showToast({ title: '请先扫描U盘', icon: 'none' })
    return
  }
  showScanResult.value = true
}

const startImport = async () => {
  showScanResult.value = false
  loading.value = true
  const total = scannedEvents.value.length
  let imported = 0

  for (const item of scannedEvents.value) {
    loadingText.value = `导入中 ${imported + 1}/${total}...`
    try {
      const localDir = '_doc'
      const files = await importEvent(item, localDir)
      const eventId = await insertEvent({
        vin: '',
        event_type: item.eventType,
        event_time: item.eventTime,
        duration: 0,
        latitude: null,
        longitude: null,
        thumbnail: '',
        imported: 1
      })
      for (const fp of files) {
        await insertVideo({
          event_id: eventId,
          camera: item.camera,
          file_path: fp,
          duration: 0,
          file_size: item.size || 0
        })
      }
      if (files.length > 0) {
        try {
          const thumbPath = await generateThumbnail(files[0], localDir)
          if (thumbPath && eventId) {
            await updateEvent(eventId, { thumbnail: thumbPath })
          }
        } catch (e) {}
      }
      imported++
    } catch (e) {
      console.error('import event failed', e)
    }
  }

  loadingText.value = 'GPS融合中...'
  try {
    await batchFuseEvents(events.value)
  } catch (e) {}

  try {
    await cleanOldRecentClips(7)
  } catch (e) {}

  scannedEvents.value = []
  loading.value = false
  uni.showToast({ title: `成功导入 ${imported} 个事件`, icon: 'success' })
  loadStats()
  loadEvents()
}

const goPlayer = (id) => {
  uni.navigateTo({ url: '/pages/dashcam/player?id=' + id })
}

onMounted(async () => {
  try {
    await initDB()
  } catch (e) {}
  loadStats()
  loadEvents()
})
</script>

<style lang="scss" scoped>
.dashcam-container {
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

.stats-card {
  display: flex;
  align-items: center;
  justify-content: space-around;
  background: var(--bg-card);
  border-radius: 28rpx;
  padding: 32rpx 24rpx;
  margin-bottom: 24rpx;
  box-shadow: var(--shadow-card);
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8rpx;
  flex: 1;
}

.stat-value {
  font-size: 36rpx;
  font-weight: 800;
  color: var(--color-primary);
}

.stat-label {
  font-size: 22rpx;
  color: var(--text-tertiary);
}

.stat-divider {
  width: 1rpx;
  height: 60rpx;
  background: var(--border-divider);
}

.tab-bar {
  display: flex;
  gap: 12rpx;
  margin-bottom: 24rpx;
  background: var(--bg-card);
  border-radius: 20rpx;
  padding: 8rpx;
  box-shadow: var(--shadow-card);
}

.tab-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 64rpx;
  border-radius: 16rpx;
  transition: all 0.25s ease;

  &.active {
    background: var(--bg-filter-active);

    .tab-text {
      color: var(--text-filter-active);
      font-weight: 700;
    }
  }
}

.tab-text {
  font-size: 26rpx;
  color: var(--text-filter);
  font-weight: 500;
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

.event-thumb {
  width: 140rpx;
  height: 100rpx;
  border-radius: 16rpx;
  overflow: hidden;
  flex-shrink: 0;
  background: var(--bg-card-secondary);
}

.thumb-img {
  width: 100%;
  height: 100%;
}

.thumb-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-card-secondary);
}

.event-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.event-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12rpx;
}

.event-time {
  font-size: 26rpx;
  font-weight: 600;
  color: var(--text-primary);
}

.event-type-tag {
  padding: 4rpx 16rpx;
  border-radius: 8rpx;
  border: 1rpx solid;
  flex-shrink: 0;
}

.event-type-text {
  font-size: 20rpx;
  font-weight: 600;
}

.event-detail {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.detail-row {
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
  height: 160rpx;
}

.action-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  gap: 16rpx;
  padding: 20rpx 32rpx;
  padding-bottom: calc(20rpx + env(safe-area-inset-bottom));
  background: var(--bg-card);
  border-top: 1rpx solid var(--border-card);
  z-index: 100;
}

.action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  height: 88rpx;
  border-radius: 44rpx;
  font-weight: 600;
  position: relative;

  &.scan-btn {
    background: var(--gradient);
  }

  &.import-btn {
    background: linear-gradient(135deg, #5BE7C4, #3cc9a5);

    &.disabled {
      opacity: 0.4;
    }
  }
}

.action-btn-text {
  font-size: 28rpx;
  color: #fff;
  font-weight: 600;
}

.import-badge {
  position: absolute;
  top: -8rpx;
  right: -8rpx;
  min-width: 36rpx;
  height: 36rpx;
  border-radius: 18rpx;
  background: #f97316;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 8rpx;
}

.badge-text {
  font-size: 20rpx;
  color: #fff;
  font-weight: 700;
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

.scan-result-modal {
  width: 100%;
  max-width: 640rpx;
  max-height: 80vh;
  background: var(--dark-page-card, #1F2937);
  border-radius: 32rpx;
  overflow: hidden;
  border: 1rpx solid var(--dark-page-card-border, rgba(255, 255, 255, 0.1));
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 32rpx 32rpx 20rpx;
}

.modal-title {
  font-size: 34rpx;
  font-weight: 700;
  color: var(--dark-page-text);
}

.modal-close {
  width: 56rpx;
  height: 56rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--dark-page-glass-bg);
}

.modal-scroll {
  flex: 1;
  overflow: hidden;
  padding: 0 32rpx;
  max-height: 50vh;
}

.scan-summary {
  padding: 16rpx 0;
  border-bottom: 1rpx solid var(--dark-page-glass-border);
  margin-bottom: 16rpx;
}

.scan-summary-text {
  font-size: 26rpx;
  color: var(--dark-page-text-secondary);
}

.scan-event-item {
  display: flex;
  align-items: center;
  gap: 16rpx;
  padding: 16rpx 0;
  border-bottom: 1rpx solid var(--dark-page-glass-border);

  &:last-child {
    border-bottom: none;
  }
}

.scan-event-type {
  padding: 4rpx 14rpx;
  border-radius: 8rpx;
  flex-shrink: 0;
}

.scan-event-type-text {
  font-size: 20rpx;
  font-weight: 600;
}

.scan-event-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.scan-event-name {
  font-size: 24rpx;
  color: var(--dark-page-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scan-event-camera {
  font-size: 20rpx;
  color: var(--dark-page-text-hint);
}

.modal-footer {
  display: flex;
  gap: 16rpx;
  padding: 20rpx 32rpx 28rpx;
}

.modal-btn {
  flex: 1;
  height: 80rpx;
  border-radius: 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;

  &.cancel-btn {
    background: var(--dark-page-glass-bg);
  }

  &.confirm-btn {
    background: linear-gradient(135deg, #5BE7C4, #3cc9a5);
  }
}

.modal-btn-text {
  font-size: 28rpx;
  color: var(--dark-page-text-secondary);
  font-weight: 500;
}

.confirm-text {
  color: #fff;
  font-weight: 600;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
