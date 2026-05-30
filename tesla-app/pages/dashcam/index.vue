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

      <view class="day-list" v-if="dayGroups.length > 0">
        <view
          class="day-card"
          v-for="day in dayGroups"
          :key="day.date"
          @click="goDay(day.date)"
          @longpress="onLongPressDay(day)"
        >
          <view class="day-left">
            <view class="day-type-dots">
              <view
                v-for="t in day.types"
                :key="t"
                class="type-dot"
                :style="{ backgroundColor: getTypeColor(t) }"
              ></view>
            </view>
          </view>
          <view class="day-center">
            <text class="day-date-main">{{ formatDayDate(day.date) }}</text>
            <text class="day-date-week">{{ getWeekDay(day.date) }}</text>
          </view>
          <view class="day-right">
            <text class="day-count">{{ day.count }}事件</text>
            <text class="day-videos">{{ day.videoCount }}视频</text>
          </view>
          <view class="day-arrow">
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
            <text class="scan-summary-text">共发现 {{ scannedDayGroups.length }} 天 {{ scannedEvents.length }} 个事件，已选 {{ selectedDayCount }} 个事件</text>
            <view class="select-all-bar" @click="toggleSelectAll">
              <view class="custom-check" :class="{ checked: isAllSelected }">
                <view v-if="isAllSelected" class="check-mark"></view>
              </view>
              <text class="select-all-text">全选</text>
            </view>
          </view>
          <view class="scan-day-group" v-for="day in scannedDayGroups" :key="day.date">
            <view class="scan-day-header" @click="toggleScanDay(day.date)">
              <view class="scan-day-expand-icon" :class="{ expanded: isScanDayExpanded(day.date) }">
                <Icon name="ChevronForward" :size="14" color="var(--dark-page-text-secondary)" />
              </view>
              <view class="custom-check" :class="{ checked: isDaySelected(day.date) }" @click.stop="toggleSelectDay(day.date)">
                <view v-if="isDaySelected(day.date)" class="check-mark"></view>
              </view>
              <text class="scan-day-date">{{ formatDayDate(day.date) }}</text>
              <text class="scan-day-count">{{ day.count }}个事件</text>
              <view class="scan-day-dots">
                <view
                  v-for="t in day.types"
                  :key="t"
                  class="type-dot-sm"
                  :style="{ backgroundColor: getTypeColor(t) }"
                ></view>
              </view>
            </view>
            <view v-if="isScanDayExpanded(day.date)" class="scan-day-events">
              <view class="scan-event-item" v-for="item in day.events" :key="item.eventId" @click="toggleSelectEvent(item.eventId)">
                <view class="custom-check" :class="{ checked: selectedEventIds.includes(item.eventId) }">
                  <view v-if="selectedEventIds.includes(item.eventId)" class="check-mark"></view>
                </view>
                <view class="scan-event-main">
                  <view class="scan-event-header">
                    <view class="scan-event-type" :style="{ backgroundColor: getTypeColor(item.eventType) + '20' }">
                      <text class="scan-event-type-text" :style="{ color: getTypeColor(item.eventType) }">{{ getTypeLabel(item.eventType) }}</text>
                    </view>
                    <text class="scan-event-time">{{ formatEventTimeOnly(item.eventTime) }}</text>
                  </view>
                  <view class="scan-event-cameras">
                    <view class="camera-tag" v-for="(video, camera) in item.videos" :key="camera" :class="{ 'has-video': video }">
                      <text class="camera-tag-text">{{ getCameraLabel(camera) }}</text>
                    </view>
                  </view>
                </view>
              </view>
            </view>
          </view>
        </scroll-view>
        <view class="modal-footer">
          <view class="modal-btn cancel-btn" @click="showScanResult = false">
            <text class="modal-btn-text">取消</text>
          </view>
          <view class="modal-btn confirm-btn" :class="{ disabled: selectedEventIds.length === 0 }" @click="startImport">
            <text class="modal-btn-text confirm-text">导入选中 ({{ selectedEventIds.length }})</text>
          </view>
        </view>
      </view>
    </view>

    <view class="scan-result-mask" v-if="showTimeFilter" @click="showTimeFilter = false">
      <view class="time-filter-modal" @click.stop>
        <view class="modal-header">
          <text class="modal-title">选择扫描范围</text>
          <view class="modal-close" @click="showTimeFilter = false">
            <Icon name="Close" :size="20" :color="modalCloseColor" />
          </view>
        </view>
        <view class="time-filter-list">
          <view
            class="time-filter-item"
            v-for="(option, idx) in TIME_FILTER_OPTIONS"
            :key="idx"
            @click="selectTimeFilter(option)"
          >
            <view class="time-filter-left">
              <text class="time-filter-label">{{ option.label }}</text>
              <text class="time-filter-desc">{{ option.desc }}</text>
            </view>
            <view class="time-filter-right">
              <view class="speed-badge" :class="'speed-' + option.speed">
                <text class="speed-text">{{ option.speed }}</text>
              </view>
              <Icon name="ChevronRight" :size="16" :color="themeStore.colors.chevron" />
            </view>
          </view>
        </view>
      </view>
    </view>

    <view class="scan-result-mask" v-if="showDeleteConfirm" @click="cancelDelete">
      <view class="delete-confirm-modal" @click.stop>
        <view class="delete-confirm-icon">
          <Icon name="Trash" :size="32" color="#EF4444" />
        </view>
        <text class="delete-confirm-title">确认删除</text>
        <text class="delete-confirm-desc" v-if="deleteTarget">将删除{{ deleteTarget.date }}的所有视频文件（{{ deleteTarget.count }}个事件），此操作不可恢复</text>
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
import { onShow } from '@dcloudio/uni-app'
import NavBar from '@/components/NavBar/NavBar.vue'
import Icon from '@/components/Icon/Icon.vue'
import { useThemeStore } from '@/store/theme'
import { initDB, getEventsWithVideoCount, getStorageStats, cleanOldRecentClips, insertEvent, insertVideo, updateEvent, checkEventExists, cleanPendingImports, selectSql, deleteEvent } from '@/utils/dashcam-db.js'
import { selectTeslaCamDir, scanTeslaCam, importEvent, generateThumbnail, waitForPlus, scanLocalVideos, deleteImportedEvent } from '@/utils/dashcam-scanner.js'
import { batchFuseEvents } from '@/utils/dashcam-gps.js'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)

const statusBarHeight = uni.getSystemInfoSync().statusBarHeight || 0

const arrowColor = computed(() => themeStore.colors.chevron)
const emptyIconColor = computed(() => themeStore.colors.inactiveIcon)
const modalCloseColor = computed(() => themeStore.colors.hint)

const activeTab = ref('recent')
const allEvents = ref([])
const stats = ref({ eventCount: 0, videoCount: 0, trackCount: 0, totalSize: 0 })
const loading = ref(false)
const loadingText = ref('加载中...')
const scannedEvents = ref([])
const showScanResult = ref(false)
const selectedEventIds = ref([])
const showDeleteConfirm = ref(false)
const deleteTarget = ref(null)
const expandedScanDays = ref([])

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

const CAMERA_LABELS = {
  front: '前视',
  back: '后视',
  left_repeater: '左视',
  right_repeater: '右视',
  left: '左视',
  right: '右视'
}

const getCameraLabel = (camera) => CAMERA_LABELS[camera] || camera

const dayGroups = computed(() => {
  const map = {}
  for (const e of allEvents.value) {
    const d = new Date(e.event_time)
    const pad = (n) => String(n).padStart(2, '0')
    const dateKey = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
    if (!map[dateKey]) {
      map[dateKey] = { date: dateKey, count: 0, videoCount: 0, types: new Set() }
    }
    map[dateKey].count++
    map[dateKey].types.add(e.event_type || 'recent')
    const vc = e.video_count || (e.videos ? e.videos.length : 0) || 0
    map[dateKey].videoCount += vc
  }
  return Object.values(map)
    .map(g => ({ ...g, types: [...g.types] }))
    .sort((a, b) => b.date.localeCompare(a.date))
})

const scannedDayGroups = computed(() => {
  const map = {}
  for (const item of scannedEvents.value) {
    const d = new Date(item.eventTime)
    const pad = (n) => String(n).padStart(2, '0')
    const dateKey = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
    if (!map[dateKey]) {
      map[dateKey] = { date: dateKey, count: 0, types: new Set(), events: [] }
    }
    map[dateKey].count++
    map[dateKey].types.add(item.eventType || 'recent')
    map[dateKey].events.push(item)
  }
  return Object.values(map)
    .map(g => ({ ...g, types: [...g.types] }))
    .sort((a, b) => b.date.localeCompare(a.date))
})

const selectedDayCount = computed(() => {
  return selectedEventIds.value.length
})

const isAllSelected = computed(() => {
  return scannedEvents.value.length > 0 && selectedEventIds.value.length === scannedEvents.value.length
})

const isDaySelected = (date) => {
  const dayGroup = scannedDayGroups.value.find(g => g.date === date)
  if (!dayGroup) return false
  return dayGroup.events.every(e => selectedEventIds.value.includes(e.eventId))
}

const isScanDayExpanded = (date) => expandedScanDays.value.includes(date)

const toggleScanDay = (date) => {
  const pos = expandedScanDays.value.indexOf(date)
  if (pos > -1) {
    expandedScanDays.value.splice(pos, 1)
  } else {
    expandedScanDays.value.push(date)
  }
}

const formatDayDate = (dateStr) => {
  if (!dateStr) return ''
  const parts = dateStr.split('-')
  if (parts.length === 3) return `${parts[0]}年${parseInt(parts[1])}月${parseInt(parts[2])}日`
  return dateStr
}

const getWeekDay = (dateStr) => {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return days[d.getDay()]
}

const formatEventTimeOnly = (ts) => {
  if (!ts) return ''
  const d = new Date(ts)
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
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
  }).catch((err) => {
    console.error('[Dashcam:index] loadStats error:', err)
  })
}

const loadEvents = async () => {
  allEvents.value = []
  loading.value = true
  loadingText.value = '加载中...'

  try {
    const res = await getEventsWithVideoCount({ eventType: activeTab.value, limit: 200 })
    if (res && res.length > 0) {
      allEvents.value = res
      try {
        const fused = await batchFuseEvents(allEvents.value)
        allEvents.value = fused
      } catch (e) {}
    } else {
      try {
        const localVids = await scanLocalVideos()
        if (localVids.length > 0) {
          const grouped = groupLocalVideosToEvents(localVids)
          allEvents.value = grouped
        }
      } catch (e) {}
    }
  } catch (err) {
    console.error('[Dashcam:index] loadEvents error:', err)
    allEvents.value = []
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
    const eventType = v.eventType || 'recent'
    const mapKey = eventType + '_' + timeKey
    if (!eventMap[mapKey]) {
      const ts = new Date(
        Number(match[1]),
        Number(match[2]) - 1,
        Number(match[3]),
        Number(match[4]),
        Number(match[5]),
        Number(match[6])
      ).getTime()
      eventMap[mapKey] = {
        id: 'local_' + mapKey,
        event_type: eventType,
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

const TIME_FILTER_OPTIONS = [
  { label: '近24小时', value: 1, speed: '很快', desc: '仅扫描今天' },
  { label: '最近7天', value: 7, speed: '快', desc: '约1-2分钟' },
  { label: '最近15天', value: 15, speed: '中等', desc: '约3-5分钟' },
  { label: '全部（高级）', value: 0, speed: '较慢', desc: '完整扫描U盘' }
]

const showTimeFilter = ref(false)

const handleScanUSB = async () => {
  showTimeFilter.value = true
}

const selectTimeFilter = async (option) => {
  showTimeFilter.value = false
  loading.value = true
  loadingText.value = '选择TeslaCam目录...'

  try {
    const treeUri = await selectTeslaCamDir()
    if (!treeUri) {
      loading.value = false
      return
    }

    loadingText.value = option.value === 0 ? '完整扫描中...' : `扫描最近${option.label}...`

    let minEventTime = 0
    if (option.value > 0) {
      const now = Date.now()
      minEventTime = now - (option.value * 24 * 60 * 60 * 1000)
    }

    const result = await scanTeslaCam(treeUri, minEventTime)
    scannedEvents.value = result.events || []

    if (scannedEvents.value.length > 0) {
      selectedEventIds.value = scannedEvents.value.map(e => e.eventId)
      expandedScanDays.value = []
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
  selectedEventIds.value = scannedEvents.value.map(e => e.eventId)
  showScanResult.value = true
}

const toggleSelectEvent = (eventId) => {
  const pos = selectedEventIds.value.indexOf(eventId)
  if (pos > -1) {
    selectedEventIds.value.splice(pos, 1)
  } else {
    selectedEventIds.value.push(eventId)
  }
}

const toggleSelectDay = (date) => {
  const dayGroup = scannedDayGroups.value.find(g => g.date === date)
  if (!dayGroup) return
  const dayEventIds = dayGroup.events.map(e => e.eventId)
  const allSelected = dayEventIds.every(id => selectedEventIds.value.includes(id))
  if (allSelected) {
    selectedEventIds.value = selectedEventIds.value.filter(id => !dayEventIds.includes(id))
  } else {
    for (const id of dayEventIds) {
      if (!selectedEventIds.value.includes(id)) {
        selectedEventIds.value.push(id)
      }
    }
  }
}

const toggleSelectAll = () => {
  if (isAllSelected.value) {
    selectedEventIds.value = []
  } else {
    selectedEventIds.value = scannedEvents.value.map(e => e.eventId)
  }
}

const startImport = async () => {
  if (selectedEventIds.value.length === 0) {
    uni.showToast({ title: '请选择要导入的事件', icon: 'none' })
    return
  }
  showScanResult.value = false
  loading.value = true
  const itemsToImport = scannedEvents.value.filter(e => selectedEventIds.value.includes(e.eventId))
  const total = itemsToImport.length
  let imported = 0
  let failed = 0

  for (let i = 0; i < itemsToImport.length; i++) {
    const item = itemsToImport[i]
    const typeLabel = getTypeLabel(item.eventType)
    loadingText.value = `导入中 ${i + 1}/${total} [${typeLabel}]...`

    await new Promise(r => setTimeout(r, 50))

    try {
      const exists = await checkEventExists(item.eventTime, item.eventType)
      if (exists) {
        imported++
        continue
      }

      const eventId = await insertEvent({
        vin: '',
        event_type: item.eventType,
        event_time: item.eventTime,
        duration: 0,
        latitude: null,
        longitude: null,
        thumbnail: '',
        imported: 0
      })

      const videoResults = await importEvent(item, '_doc')

      if (videoResults.length === 0) {
        failed++
        continue
      }

      for (const result of videoResults) {
        await insertVideo({
          event_id: eventId,
          camera: result.camera,
          file_path: result.path,
          duration: 0,
          file_size: result.size || 0
        })
      }

      await updateEvent(eventId, { imported: 1 })

      try {
        const thumbVideo = videoResults.find(v => v.camera === 'front') ||
                           videoResults.find(v => v.camera === 'back') ||
                           videoResults[0]
        const thumbPath = await generateThumbnail(thumbVideo.path, '_doc')
        if (thumbPath) {
          await updateEvent(eventId, { thumbnail: thumbPath })
        }
      } catch (e) {}

      imported++
    } catch (e) {
      console.error('import event failed', e)
      failed++
    }
  }

  loadingText.value = 'GPS融合中...'
  try {
    const latestEvents = await getEventsWithVideoCount({ limit: 100 })
    await batchFuseEvents(latestEvents)
  } catch (e) {}

  try {
    await cleanOldRecentClips(7)
  } catch (e) {}

  scannedEvents.value = []
  selectedEventIds.value = []
  loading.value = false

  const msg = failed > 0
    ? `导入完成 ${imported} 个，失败 ${failed} 个`
    : `成功导入 ${imported} 个时间点`
  uni.showToast({ title: msg, icon: imported > 0 ? 'success' : 'none' })
  loadStats()
  loadEvents()
}

const onLongPressDay = (day) => {
  deleteTarget.value = day
  showDeleteConfirm.value = true
}

const confirmDelete = async () => {
  const target = deleteTarget.value
  if (!target) return
  showDeleteConfirm.value = false

  const dayEvents = allEvents.value.filter(e => {
    const d = new Date(e.event_time)
    const pad = (n) => String(n).padStart(2, '0')
    const dateKey = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
    return dateKey === target.date
  })

  try {
    for (const event of dayEvents) {
      const isLocal = String(event.id).startsWith('local_')
      if (isLocal) {
        if (event.videos && event.videos.length) {
          for (const v of event.videos) {
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
        const paths = await deleteEvent(event.id)
        for (const p of paths) {
          try {
            const rawPath = p.replace(/^file:\/\//, '')
            const File = plus.android.importClass('java.io.File')
            const f = new File(rawPath)
            plus.android.invoke(f, 'delete')
          } catch (e) {}
        }
      }
    }
    allEvents.value = allEvents.value.filter(e => {
      const d = new Date(e.event_time)
      const pad = (n) => String(n).padStart(2, '0')
      const dateKey = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
      return dateKey !== target.date
    })
    uni.showToast({ title: '已删除', icon: 'success' })
    loadStats()
  } catch (e) {
    console.error('[Dashcam:index] delete failed:', e)
    uni.showToast({ title: '删除失败', icon: 'none' })
  }
  deleteTarget.value = null
}

const cancelDelete = () => {
  showDeleteConfirm.value = false
  deleteTarget.value = null
}

const goDay = (date) => {
  uni.navigateTo({
    url: '/pages/dashcam/day?date=' + date + '&type=' + activeTab.value
  })
}

let _initDone = false
let _initPromise = null
let _lastLoadTs = 0

const ensureInit = async () => {
  if (_initDone) return
  if (!_initPromise) {
    _initPromise = (async () => {
      await waitForPlus()
      await initDB()
      await cleanPendingImports()
      _initDone = true
    })()
  }
  return _initPromise
}

const safeReload = () => {
  const now = Date.now()
  const gap = now - _lastLoadTs
  if (gap < 1500) return
  _lastLoadTs = now
  loadStats()
  loadEvents()
}

onMounted(async () => {
  try {
    await ensureInit()
  } catch (e) {
    console.error('[Dashcam:index] init error:', e)
  }
  safeReload()
})

onShow(() => {
  ensureInit().then(() => {
    safeReload()
  }).catch(() => {
    _lastLoadTs = 0
    loadStats()
    loadEvents()
  })
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

.day-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.day-card {
  display: flex;
  align-items: center;
  background: var(--bg-card);
  border-radius: 24rpx;
  padding: 24rpx;
  box-shadow: var(--shadow-card);

  &:active {
    opacity: 0.9;
  }
}

.day-left {
  flex-shrink: 0;
  margin-right: 16rpx;
}

.day-type-dots {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.type-dot {
  width: 10rpx;
  height: 10rpx;
  border-radius: 5rpx;
}

.type-dot-sm {
  width: 8rpx;
  height: 8rpx;
  border-radius: 4rpx;
}

.day-center {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 12rpx;
}

.day-date-main {
  font-size: 28rpx;
  font-weight: 700;
  color: var(--text-primary);
}

.day-date-week {
  font-size: 22rpx;
  color: var(--text-tertiary);
  font-weight: 500;
}

.day-right {
  display: flex;
  gap: 12rpx;
  flex-shrink: 0;
  margin-right: 12rpx;
}

.day-count {
  font-size: 22rpx;
  color: var(--text-tertiary);
  font-weight: 500;
}

.day-videos {
  font-size: 22rpx;
  color: var(--text-placeholder);
}

.day-arrow {
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
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.scan-summary-text {
  font-size: 24rpx;
  color: var(--dark-page-text-secondary);
  flex: 1;
}

.select-all-bar {
  display: flex;
  align-items: center;
  gap: 8rpx;
  flex-shrink: 0;
}

.select-all-text {
  font-size: 24rpx;
  color: var(--dark-page-text-secondary);
}

.scan-day-group {
  margin-bottom: 16rpx;
}

.scan-day-header {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 12rpx 0;
  border-bottom: 1rpx solid var(--dark-page-glass-border);
}

.scan-day-expand-icon {
  transition: transform 0.2s;
  display: flex;
  align-items: center;

  &.expanded {
    transform: rotate(90deg);
  }
}

.scan-day-events {
  padding-left: 8rpx;
}

.scan-day-date {
  font-size: 26rpx;
  font-weight: 600;
  color: var(--dark-page-text);
}

.scan-day-count {
  font-size: 22rpx;
  color: var(--dark-page-text-secondary);
}

.scan-day-dots {
  display: flex;
  gap: 6rpx;
  margin-left: auto;
}

.scan-event-item {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 12rpx 0 12rpx 24rpx;

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

.scan-event-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.scan-event-header {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.scan-event-time {
  font-size: 24rpx;
  color: var(--dark-page-text);
  font-weight: 500;
}

.scan-event-cameras {
  display: flex;
  gap: 8rpx;
  flex-wrap: wrap;
}

.camera-tag {
  padding: 4rpx 12rpx;
  border-radius: 8rpx;
  background: var(--dark-page-glass-bg);
  opacity: 0.5;

  &.has-video {
    opacity: 1;
    background: rgba(91, 231, 196, 0.2);
  }
}

.camera-tag-text {
  font-size: 20rpx;
  color: var(--dark-page-text-secondary);
}

.custom-check {
  width: 40rpx;
  height: 40rpx;
  border-radius: 8rpx;
  border: 2rpx solid var(--dark-page-text-hint);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  &.checked {
    background: #5BE7C4;
    border-color: #5BE7C4;
  }
}

.check-mark {
  width: 20rpx;
  height: 10rpx;
  border-left: 4rpx solid #fff;
  border-bottom: 4rpx solid #fff;
  transform: rotate(-45deg) translate(2rpx, -2rpx);
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

.time-filter-modal {
  width: 100%;
  max-width: 640rpx;
  background: var(--dark-page-card, #1F2937);
  border-radius: 32rpx;
  overflow: hidden;
  border: 1rpx solid var(--dark-page-card-border, rgba(255, 255, 255, 0.1));
  display: flex;
  flex-direction: column;
}

.time-filter-list {
  padding: 16rpx 0;
}

.time-filter-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 28rpx 32rpx;
  border-bottom: 1rpx solid var(--dark-page-glass-border);

  &:last-child {
    border-bottom: none;
  }

  &:active {
    background: var(--dark-page-glass-bg);
  }
}

.time-filter-left {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.time-filter-label {
  font-size: 30rpx;
  font-weight: 600;
  color: var(--dark-page-text);
}

.time-filter-desc {
  font-size: 24rpx;
  color: var(--dark-page-text-secondary);
}

.time-filter-right {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.speed-badge {
  padding: 6rpx 14rpx;
  border-radius: 20rpx;
  font-size: 22rpx;
  font-weight: 600;

  &.speed-很快 {
    background: rgba(91, 231, 196, 0.2);
    color: #5BE7C4;
  }

  &.speed-快 {
    background: rgba(59, 130, 246, 0.2);
    color: #3B82F6;
  }

  &.speed-中等 {
    background: rgba(245, 158, 11, 0.2);
    color: #F59E0B;
  }

  &.speed-较慢 {
    background: rgba(239, 68, 68, 0.2);
    color: #EF4444;
  }
}

.speed-text {
  font-size: 22rpx;
  font-weight: 600;
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
