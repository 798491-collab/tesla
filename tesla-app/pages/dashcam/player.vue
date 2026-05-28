<template>
  <view class="player-container" :class="themeClass">
    <NavBar title="视频回放" />

    <view class="video-grid" :style="{ paddingTop: statusBarHeight + 44 + 'px' }">
      <view class="video-cell">
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
          <Icon name="Videocam" :size="32" :color="placeholderColor" />
          <text class="placeholder-label">Front</text>
        </view>
        <text class="camera-label">Front</text>
      </view>

      <view class="video-cell">
        <video
          v-if="videoMap.back"
          id="backPlayer"
          class="video-player"
          :src="videoMap.back"
          :autoplay="false"
          :show-fullscreen-btn="false"
          :show-play-btn="false"
          :show-center-play-btn="false"
          :enable-progress-gesture="false"
          muted
        />
        <view v-else class="video-placeholder">
          <Icon name="Videocam" :size="32" :color="placeholderColor" />
          <text class="placeholder-label">Back</text>
        </view>
        <text class="camera-label">Back</text>
      </view>

      <view class="video-cell">
        <video
          v-if="videoMap.left_repeater"
          id="leftPlayer"
          class="video-player"
          :src="videoMap.left_repeater"
          :autoplay="false"
          :show-fullscreen-btn="false"
          :show-play-btn="false"
          :show-center-play-btn="false"
          :enable-progress-gesture="false"
          muted
        />
        <view v-else class="video-placeholder">
          <Icon name="Videocam" :size="32" :color="placeholderColor" />
          <text class="placeholder-label">Left</text>
        </view>
        <text class="camera-label">Left</text>
      </view>

      <view class="video-cell">
        <video
          v-if="videoMap.right_repeater"
          id="rightPlayer"
          class="video-player"
          :src="videoMap.right_repeater"
          :autoplay="false"
          :show-fullscreen-btn="false"
          :show-play-btn="false"
          :show-center-play-btn="false"
          :enable-progress-gesture="false"
          muted
        />
        <view v-else class="video-placeholder">
          <Icon name="Videocam" :size="32" :color="placeholderColor" />
          <text class="placeholder-label">Right</text>
        </view>
        <text class="camera-label">Right</text>
      </view>
    </view>

    <view class="event-info-section" v-if="eventData">
      <view class="info-row">
        <Icon name="Time" :size="16" :color="infoIconColor" />
        <text class="info-text">{{ formatEventTime(eventData.event_time) }}</text>
      </view>
      <view class="info-row">
        <Icon name="Shield" :size="16" :color="infoIconColor" />
        <text class="info-text">{{ getTypeLabel(eventData.event_type) }}</text>
      </view>
      <view class="info-row" v-if="eventData.latitude && eventData.longitude">
        <Icon name="Location" :size="16" :color="infoIconColor" />
        <text class="info-text">{{ eventData.latitude.toFixed(4) }}, {{ eventData.longitude.toFixed(4) }}</text>
      </view>
      <view class="map-btn" v-if="eventData.latitude && eventData.longitude" @click="openMap">
        <Icon name="Map" :size="16" color="#fff" />
        <text class="map-btn-text">查看地图位置</text>
      </view>
    </view>

    <view class="control-bar">
      <view class="control-main">
        <view class="play-btn" @click="togglePlay">
          <Icon :name="isPlaying ? 'Pause' : 'Play'" :size="22" color="#fff" />
        </view>
        <view class="progress-wrap">
          <slider
            class="progress-slider"
            :value="currentTime"
            :min="0"
            :max="duration || 1"
            :block-size="12"
            activeColor="#e0e0e0"
            backgroundColor="rgba(255,255,255,0.2)"
            blockColor="#e0e0e0"
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
  </view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import NavBar from '@/components/NavBar/NavBar.vue'
import Icon from '@/components/Icon/Icon.vue'
import { useThemeStore } from '@/store/theme'
import { getEventById } from '@/utils/dashcam-db.js'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const placeholderColor = computed(() => themeStore.colors.inactiveIcon)
const infoIconColor = computed(() => themeStore.colors.hint)

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

let frontCtx = null
let backCtx = null
let leftCtx = null
let rightCtx = null

onLoad((options) => {
  eventId.value = options?.id || null
})

onMounted(async () => {
  if (!eventId.value) return
  try {
    const event = await getEventById(eventId.value)
    if (!event) return
    eventData.value = event
    const map = {}
    if (event.videos && event.videos.length) {
      event.videos.forEach(v => {
        map[v.camera] = v.file_path
      })
    }
    videoMap.value = map
  } catch (e) {}

  frontCtx = uni.createVideoContext('frontPlayer')
  backCtx = uni.createVideoContext('backPlayer')
  leftCtx = uni.createVideoContext('leftPlayer')
  rightCtx = uni.createVideoContext('rightPlayer')
})

const onLoadedMetadata = (e) => {
  duration.value = e.detail.duration || 0
}

const onTimeUpdate = (e) => {
  if (isSeeking.value) return
  currentTime.value = e.detail.currentTime
  const frontTime = e.detail.currentTime

  const syncPlayer = (ctx, label) => {
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

  syncPlayer(backCtx, 'back')
  syncPlayer(leftCtx, 'left')
  syncPlayer(rightCtx, 'right')
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

const getTypeLabel = (type) => {
  const map = { recent: '最近片段', sentry: '哨兵事件', saved: '已保存' }
  return map[type] || type || ''
}
</script>

<style lang="scss" scoped>
.player-container {
  min-height: 100vh;
  background: var(--bg-page);
  display: flex;
  flex-direction: column;
}

.video-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 4rpx;
  flex: 1;
  min-height: 0;
  padding-bottom: 16rpx;
}

.video-cell {
  position: relative;
  background: #000;
  aspect-ratio: 16 / 9;
  overflow: hidden;
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
  gap: 8rpx;
}

.placeholder-label {
  font-size: 22rpx;
  color: var(--text-hint);
}

.camera-label {
  position: absolute;
  top: 8rpx;
  left: 8rpx;
  font-size: 20rpx;
  color: rgba(255, 255, 255, 0.8);
  background: rgba(0, 0, 0, 0.5);
  padding: 2rpx 12rpx;
  border-radius: 6rpx;
}

.event-info-section {
  padding: 20rpx 24rpx;
  display: flex;
  flex-direction: column;
  gap: 12rpx;
  background: var(--bg-card);
  border-top: 1rpx solid var(--border-card);
}

.info-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.info-text {
  font-size: 26rpx;
  color: var(--text-secondary);
}

.map-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8rpx;
  height: 64rpx;
  border-radius: 32rpx;
  background: var(--gradient);
  margin-top: 8rpx;
}

.map-btn-text {
  font-size: 26rpx;
  color: #fff;
  font-weight: 600;
}

.control-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: var(--dark-page-card, #1F2937);
  border-top: 1rpx solid var(--dark-page-glass-border, rgba(255, 255, 255, 0.1));
  padding: 16rpx 24rpx;
  padding-bottom: calc(16rpx + env(safe-area-inset-bottom));
  z-index: 100;
}

.control-main {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.play-btn {
  width: 64rpx;
  height: 64rpx;
  border-radius: 32rpx;
  background: var(--gradient);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
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
  font-size: 22rpx;
  color: var(--dark-page-text-secondary, #555770);
  white-space: nowrap;
  flex-shrink: 0;
}

.speed-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16rpx;
  margin-top: 12rpx;
}

.speed-item {
  padding: 6rpx 20rpx;
  border-radius: 20rpx;
  background: var(--dark-page-glass-bg, rgba(255, 255, 255, 0.06));
  border: 1rpx solid var(--dark-page-glass-border, rgba(255, 255, 255, 0.1));

  &.active {
    background: var(--gradient);
    border-color: transparent;
  }
}

.speed-text {
  font-size: 22rpx;
  color: var(--dark-page-text-secondary, #555770);

  .speed-item.active & {
    color: #fff;
    font-weight: 600;
  }
}
</style>
