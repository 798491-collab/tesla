<template>
  <view class="player-container" :class="themeClass">
    <NavBar title="视频回放" />

    <scroll-view scroll-y class="player-scroll" :style="{ paddingTop: statusBarHeight + 44 + 'px' }">
      <view class="video-grid-wrap">
        <view class="video-cell" @click="onCellTap('front')">
          <video
            v-if="videoMap.front"
            id="frontPlayer"
            class="video-player"
            :src="videoMap.front"
            :autoplay="false"
            :show-fullscreen-btn="true"
            :show-play-btn="false"
            :show-center-play-btn="false"
            :enable-progress-gesture="false"
            @timeupdate="onTimeUpdate"
            @play="onPlay"
            @pause="onPause"
            @ended="onEnded"
            @loadedmetadata="onLoadedMetadata"
          />
          <view v-else class="video-placeholder">
            <view class="placeholder-icon-wrap">
              <Icon name="Eye" :size="28" themeColor="inactive" />
            </view>
            <text class="placeholder-label">前视摄像头</text>
          </view>
          <view class="camera-badge front-badge">
            <Icon name="CarSport" :size="12" color="#fff" />
            <text class="badge-text">前视</text>
          </view>
        </view>

        <view class="video-cell" @click="onCellTap('back')">
          <video
            v-if="videoMap.back"
            id="backPlayer"
            class="video-player"
            :src="videoMap.back"
            :autoplay="false"
            :show-fullscreen-btn="true"
            :show-play-btn="false"
            :show-center-play-btn="false"
            :enable-progress-gesture="false"
            muted
          />
          <view v-else class="video-placeholder">
            <view class="placeholder-icon-wrap">
              <Icon name="EyeOff" :size="28" themeColor="inactive" />
            </view>
            <text class="placeholder-label">后视摄像头</text>
          </view>
          <view class="camera-badge back-badge">
            <Icon name="CarSport" :size="12" color="#fff" />
            <text class="badge-text">后视</text>
          </view>
        </view>

        <view class="video-cell" @click="onCellTap('left_repeater')">
          <video
            v-if="videoMap.left_repeater"
            id="leftPlayer"
            class="video-player"
            :src="videoMap.left_repeater"
            :autoplay="false"
            :show-fullscreen-btn="true"
            :show-play-btn="false"
            :show-center-play-btn="false"
            :enable-progress-gesture="false"
            muted
          />
          <view v-else class="video-placeholder">
            <view class="placeholder-icon-wrap">
              <Icon name="EyeOff" :size="28" themeColor="inactive" />
            </view>
            <text class="placeholder-label">左侧摄像头</text>
          </view>
          <view class="camera-badge left-badge">
            <Icon name="CarSport" :size="12" color="#fff" />
            <text class="badge-text">左视</text>
          </view>
        </view>

        <view class="video-cell" @click="onCellTap('right_repeater')">
          <video
            v-if="videoMap.right_repeater"
            id="rightPlayer"
            class="video-player"
            :src="videoMap.right_repeater"
            :autoplay="false"
            :show-fullscreen-btn="true"
            :show-play-btn="false"
            :show-center-play-btn="false"
            :enable-progress-gesture="false"
            muted
          />
          <view v-else class="video-placeholder">
            <view class="placeholder-icon-wrap">
              <Icon name="EyeOff" :size="28" themeColor="inactive" />
            </view>
            <text class="placeholder-label">右侧摄像头</text>
          </view>
          <view class="camera-badge right-badge">
            <Icon name="CarSport" :size="12" color="#fff" />
            <text class="badge-text">右视</text>
          </view>
        </view>

        <cover-view class="center-play-btn" @click.stop="togglePlay">
          <cover-view class="center-play-icon">{{ isPlaying ? '❚❚' : '▶' }}</cover-view>
        </cover-view>
      </view>

      <view class="event-info-card" v-if="eventData">
        <view class="info-header">
          <view class="info-header-left">
            <view class="info-type-dot" :style="{ backgroundColor: getTypeColor(eventData.event_type) }"></view>
            <text class="info-type-label">{{ getTypeLabel(eventData.event_type) }}</text>
          </view>
          <text class="info-time">{{ formatEventTime(eventData.event_time) }}</text>
        </view>

        <view class="info-details">
          <view class="info-detail-item" v-if="eventData.latitude && eventData.longitude">
            <view class="detail-icon-wrap location-wrap">
              <Icon name="Location" :size="14" color="#fff" />
            </view>
            <text class="detail-value">{{ eventData.latitude.toFixed(4) }}, {{ eventData.longitude.toFixed(4) }}</text>
          </view>
          <view class="info-detail-item">
            <view class="detail-icon-wrap time-wrap">
              <Icon name="Time" :size="14" color="#fff" />
            </view>
            <text class="detail-value">{{ formatTime(currentTime) }} / {{ formatTime(duration) }}</text>
          </view>
          <view class="info-detail-item" v-if="videoCount > 0">
            <view class="detail-icon-wrap video-wrap">
              <Icon name="Eye" :size="14" color="#fff" />
            </view>
            <text class="detail-value">{{ videoCount }} 个摄像头</text>
          </view>
        </view>

        <view class="info-actions">
          <view class="action-item map-action" v-if="eventData.latitude && eventData.longitude" @click="openMap">
            <Icon name="Map" :size="18" color="#fff" />
            <text class="action-label">查看地图</text>
          </view>
          <view class="action-item delete-action" @click="handleDelete">
            <Icon name="Trash" :size="18" color="#EF4444" />
            <text class="action-label danger-label">删除事件</text>
          </view>
        </view>
      </view>

      <view class="bottom-spacer"></view>
    </scroll-view>

    <view class="control-bar">
      <view class="control-main">
        <view class="progress-wrap">
          <slider
            class="progress-slider"
            :value="currentTime"
            :min="0"
            :max="duration || 1"
            :block-size="12"
            activeColor="var(--color-success)"
            backgroundColor="var(--dark-page-glass-bg, rgba(255,255,255,0.15))"
            blockColor="var(--color-success)"
            @change="onSeek"
            @changing="onSeeking"
          />
        </view>
        <text class="time-text">{{ formatTime(currentTime) }} / {{ formatTime(duration) }}</text>
      </view>
      <view class="speed-bar">
        <view
          v-for="s in speedOptions"
          :key="s"
          class="speed-item"
          :class="{ active: playbackRate === s }"
          @click="changeSpeed(s)"
        >
          <text class="speed-text">{{ s }}x</text>
        </view>
      </view>
    </view>

    <view class="delete-confirm-mask" v-if="showDeleteConfirm" @click="cancelDelete">
      <view class="delete-confirm-modal" @click.stop>
        <view class="delete-confirm-icon-wrap">
          <Icon name="Warning" :size="36" color="#EF4444" />
        </view>
        <text class="delete-confirm-title">确认删除</text>
        <text class="delete-confirm-desc">将删除此事件的所有视频文件，此操作不可恢复</text>
        <view class="delete-confirm-btns">
          <view class="delete-confirm-btn cancel" @click="cancelDelete">
            <text class="delete-confirm-btn-label">取消</text>
          </view>
          <view class="delete-confirm-btn danger" @click="confirmDelete">
            <text class="delete-confirm-btn-label danger-label">删除</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import NavBar from '@/components/NavBar/NavBar.vue'
import Icon from '@/components/Icon/Icon.vue'
import { useThemeStore } from '@/store/theme'
import { getEventById, deleteEvent } from '@/utils/dashcam-db.js'
import { waitForPlus, scanLocalVideos, deleteImportedEvent } from '@/utils/dashcam-scanner.js'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)

const statusBarHeight = uni.getSystemInfoSync().statusBarHeight || 0

const eventId = ref(null)
const eventData = ref(null)
const videoMap = ref({})
const isPlaying = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const playbackRate = ref(1)
const speedOptions = [0.5, 1, 1.5, 2]
const isSeeking = ref(false)
const showDeleteConfirm = ref(false)

const videoCount = computed(() => Object.keys(videoMap.value).length)

const TYPE_COLORS = {
  recent: '#60a5fa',
  saved: '#5BE7C4',
  sentry: '#f97316'
}

const getTypeColor = (type) => TYPE_COLORS[type] || '#60a5fa'

const getTypeLabel = (type) => {
  const map = { recent: '最近片段', sentry: '哨兵事件', saved: '已保存' }
  return map[type] || type || ''
}

let frontCtx = null
let backCtx = null
let leftCtx = null
let rightCtx = null

onLoad((options) => {
  eventId.value = options?.id || null
})

let _lastLoadTs = 0

const loadEventData = async () => {
  if (!eventId.value) return

  const now = Date.now()
  if (now - _lastLoadTs < 1500) return
  _lastLoadTs = now

  try {
    await waitForPlus()

    let event = null
    const isLocal = String(eventId.value).startsWith('local_')

    if (isLocal) {
      const localId = String(eventId.value).replace(/^local_/, '')
      const localMatch = localId.match(/^(recent|saved|sentry)_(\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2})$/)
      let eventType = 'recent'
      let timeKey = localId
      if (localMatch) {
        eventType = localMatch[1]
        timeKey = localMatch[2]
      }
      const localVids = await scanLocalVideos()
      const matched = localVids.filter(v => {
        return v.eventType === eventType && v.name && v.name.includes(timeKey)
      })
      if (matched.length > 0) {
        event = {
          id: eventId.value,
          event_type: eventType,
          _isLocal: true,
          videos: matched.map(v => ({
            camera: v.camera,
            file_path: v.path,
            file_size: v.size
          }))
        }
      }
    } else {
      const safeId = parseInt(eventId.value, 10)
      if (!isNaN(safeId)) {
        event = await getEventById(safeId)
      }
    }

    if (!event) return
    eventData.value = event

    const map = {}
    if (event.videos && event.videos.length) {
      event.videos.forEach(v => {
        let videoPath = v.file_path
        if (videoPath && !videoPath.startsWith('file://') && !videoPath.startsWith('content://')) {
          videoPath = 'file://' + videoPath
        }
        try {
          if (videoPath && videoPath.startsWith('file://')) {
            const rawPath = videoPath.replace(/^file:\/\//, '')
            const converted = plus.io.convertLocalFileSystemURL(rawPath)
            if (converted) videoPath = converted
          }
        } catch (e) {}
        map[v.camera] = videoPath
      })
    }
    videoMap.value = map
  } catch (e) {
    console.error('[Dashcam:player] load event failed:', e)
  }

  frontCtx = uni.createVideoContext('frontPlayer')
  backCtx = uni.createVideoContext('backPlayer')
  leftCtx = uni.createVideoContext('leftPlayer')
  rightCtx = uni.createVideoContext('rightPlayer')
}

onMounted(() => {
  loadEventData()
})

onShow(() => {
  loadEventData()
})

const onCellTap = (camera) => {
  if (!videoMap.value[camera]) return
  const ctxMap = { front: frontCtx, back: backCtx, left_repeater: leftCtx, right_repeater: rightCtx }
  const ctx = ctxMap[camera]
  if (ctx) {
    ctx.requestFullScreen({ direction: 0 })
  }
}

const onLoadedMetadata = (e) => {
  duration.value = e.detail.duration || 0
}

const onTimeUpdate = (e) => {
  if (isSeeking.value) return
  currentTime.value = e.detail.currentTime
  const frontTime = e.detail.currentTime

  const syncPlayer = (ctx) => {
    if (!ctx) return
    ctx.getVideoPosition({
      success: (res) => {
        if (Math.abs(res.position - frontTime) > 0.5) {
          ctx.seek(frontTime)
        }
      },
      fail: () => {}
    })
  }

  syncPlayer(backCtx)
  syncPlayer(leftCtx)
  syncPlayer(rightCtx)
}

const onPlay = () => {
  isPlaying.value = true
  backCtx?.play()
  leftCtx?.play()
  rightCtx?.play()
}

const onPause = () => {
  isPlaying.value = false
  backCtx?.pause()
  leftCtx?.pause()
  rightCtx?.pause()
}

const onEnded = () => {
  isPlaying.value = false
  backCtx?.pause()
  leftCtx?.pause()
  rightCtx?.pause()
  currentTime.value = 0
}

const togglePlay = () => {
  if (isPlaying.value) {
    frontCtx?.pause()
  } else {
    frontCtx?.play()
  }
}

const onSeeking = () => {
  isSeeking.value = true
}

const onSeek = (e) => {
  const time = e.detail.value
  currentTime.value = time
  frontCtx?.seek(time)
  backCtx?.seek(time)
  leftCtx?.seek(time)
  rightCtx?.seek(time)
  setTimeout(() => {
    isSeeking.value = false
  }, 300)
}

const changeSpeed = (speed) => {
  playbackRate.value = speed
  frontCtx?.playbackRate(speed)
  backCtx?.playbackRate(speed)
  leftCtx?.playbackRate(speed)
  rightCtx?.playbackRate(speed)
}

const handleDelete = () => {
  showDeleteConfirm.value = true
}

const confirmDelete = async () => {
  showDeleteConfirm.value = false
  if (!eventData.value) return

  const isLocal = String(eventData.value.id).startsWith('local_')
  try {
    if (isLocal) {
      if (eventData.value.videos && eventData.value.videos.length) {
        for (const v of eventData.value.videos) {
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
      const paths = await deleteEvent(eventData.value.id)
      for (const p of paths) {
        try {
          const rawPath = p.replace(/^file:\/\//, '')
          const File = plus.android.importClass('java.io.File')
          const f = new File(rawPath)
          plus.android.invoke(f, 'delete')
        } catch (e) {}
      }
      if (eventData.value.thumbnail) {
        try {
          const rawPath = eventData.value.thumbnail.replace(/^file:\/\//, '')
          const File = plus.android.importClass('java.io.File')
          const f = new File(rawPath)
          plus.android.invoke(f, 'delete')
        } catch (e) {}
      }
    }
    uni.showToast({ title: '已删除', icon: 'success' })
    setTimeout(() => {
      uni.navigateBack()
    }, 800)
  } catch (e) {
    console.error('[Dashcam:player] delete failed:', e)
    uni.showToast({ title: '删除失败', icon: 'none' })
  }
}

const cancelDelete = () => {
  showDeleteConfirm.value = false
}

const openMap = () => {
  if (!eventData.value) return
  uni.navigateTo({
    url: `/pages/dashcam/map?lat=${eventData.value.latitude}&lng=${eventData.value.longitude}`
  })
}

const formatTime = (seconds) => {
  if (!seconds || isNaN(seconds)) return '00:00'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

const formatEventTime = (ts) => {
  if (!ts) return ''
  const d = new Date(ts)
  const y = d.getFullYear()
  const mo = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${y}-${mo}-${day} ${h}:${mi}`
}
</script>

<style lang="scss" scoped>
.player-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--bg-page);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.player-scroll {
  flex: 1;
  overflow: hidden;
}

.video-grid-wrap {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 6rpx;
  padding: 16rpx 16rpx 8rpx;
  aspect-ratio: 2 / 2;
}

.video-cell {
  position: relative;
  background: #000;
  border-radius: 16rpx;
  overflow: hidden;

  &:active {
    opacity: 0.92;
  }
}

.video-player {
  width: 100%;
  height: 100%;
}

.video-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: var(--bg-card-secondary);
  gap: 12rpx;
}

.placeholder-icon-wrap {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  background: var(--bg-icon-wrap);
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder-label {
  font-size: 22rpx;
  color: var(--text-tertiary);
  font-weight: 500;
}

.camera-badge {
  position: absolute;
  top: 12rpx;
  left: 12rpx;
  display: flex;
  align-items: center;
  gap: 6rpx;
  padding: 4rpx 14rpx;
  border-radius: 20rpx;
  backdrop-filter: blur(8px);
}

.front-badge {
  background: rgba(59, 130, 246, 0.7);
}

.back-badge {
  background: rgba(168, 85, 247, 0.7);
}

.left-badge {
  background: rgba(34, 197, 94, 0.7);
}

.right-badge {
  background: rgba(249, 115, 22, 0.7);
}

.badge-text {
  font-size: 20rpx;
  color: #fff;
  font-weight: 600;
}

.event-info-card {
  margin: 16rpx 24rpx;
  padding: 28rpx;
  background: var(--bg-card);
  border-radius: 24rpx;
  border: 1rpx solid var(--border-card);
  box-shadow: var(--shadow-card);
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.info-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.info-header-left {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.info-type-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 50%;
  flex-shrink: 0;
}

.info-type-label {
  font-size: 28rpx;
  font-weight: 700;
  color: var(--text-primary);
}

.info-time {
  font-size: 24rpx;
  color: var(--text-tertiary);
  font-weight: 500;
}

.info-details {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.info-detail-item {
  display: flex;
  align-items: center;
  gap: 14rpx;
}

.detail-icon-wrap {
  width: 40rpx;
  height: 40rpx;
  border-radius: 12rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.location-wrap {
  background: linear-gradient(135deg, #3B82F6, #2563EB);
}

.time-wrap {
  background: linear-gradient(135deg, #8B5CF6, #7C3AED);
}

.video-wrap {
  background: linear-gradient(135deg, #22C55E, #16A34A);
}

.detail-value {
  font-size: 26rpx;
  color: var(--text-secondary);
  font-weight: 500;
}

.info-actions {
  display: flex;
  gap: 16rpx;
  margin-top: 4rpx;
}

.action-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  height: 76rpx;
  border-radius: 38rpx;
}

.map-action {
  background: var(--gradient);
}

.action-label {
  font-size: 26rpx;
  color: #fff;
  font-weight: 600;
}

.danger-label {
  color: #EF4444;
}

.delete-action {
  background: rgba(239, 68, 68, 0.1);
  border: 1rpx solid rgba(239, 68, 68, 0.25);
}

.bottom-spacer {
  height: 200rpx;
}

.control-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--dark-page-card, #1F2937);
  border-top: 1rpx solid var(--dark-page-glass-border, rgba(255, 255, 255, 0.1));
  padding: 20rpx 28rpx;
  padding-bottom: calc(20rpx + env(safe-area-inset-bottom));
  z-index: 100;
}

.control-main {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.center-play-btn {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 100rpx;
  height: 100rpx;
  border-radius: 50rpx;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.85), rgba(99, 102, 241, 0.85));
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
  box-shadow: 0 8rpx 24rpx rgba(59, 130, 246, 0.4);

  &:active {
    transform: translate(-50%, -50%) scale(0.92);
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.95), rgba(99, 102, 241, 0.95));
  }
}

.center-play-icon {
  font-size: 36rpx;
  color: #fff;
  text-align: center;
  line-height: 100rpx;
}

.progress-wrap {
  flex: 1;
  min-width: 0;
}

.progress-slider {
  margin: 0;
  padding: 0;
}

.time-text {
  font-size: 24rpx;
  color: var(--dark-page-text, rgba(255, 255, 255, 0.85));
  white-space: nowrap;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
  font-weight: 500;
}

.speed-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16rpx;
  margin-top: 16rpx;
}

.speed-item {
  padding: 8rpx 24rpx;
  border-radius: 24rpx;
  background: var(--dark-page-glass-bg, rgba(255, 255, 255, 0.06));
  border: 1rpx solid var(--dark-page-glass-border, rgba(255, 255, 255, 0.1));

  &.active {
    background: var(--gradient);
    border-color: transparent;
  }

  &:active {
    transform: scale(0.95);
  }
}

.speed-text {
  font-size: 24rpx;
  color: var(--dark-page-text-secondary, rgba(255, 255, 255, 0.5));
  font-weight: 500;

  .speed-item.active & {
    color: #fff;
    font-weight: 700;
  }
}

.delete-confirm-mask {
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

.delete-confirm-icon-wrap {
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

.delete-confirm-btn-label {
  font-size: 28rpx;
  color: var(--dark-page-text-secondary, rgba(255, 255, 255, 0.6));
  font-weight: 500;

  &.danger-label {
    color: #fff;
    font-weight: 600;
  }
}
</style>
