import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

// 地图底图样式（layerRoot.setMapStyle() 参数）
// 控制地图底图瓦片的渲染风格
export const MAP_STYLES = [
  { key: 1, label: '标准地图', desc: '普通彩色地图' },
  { key: 2, label: '夜间地图', desc: '深色地图，适合夜间' },
  { key: 3, label: '卫星地图', desc: '卫星影像地图' },
]

// 导航日夜模式（navigator.setDayNightMode() 参数）
// 控制导航界面的日间/夜间/自动模式
export const NAVI_DAY_NIGHT_MODES = [
  { key: 1, label: '日间模式', desc: '白色背景、浅色道路' },
  { key: 2, label: '夜间模式', desc: '深色背景、深色道路' },
  { key: 3, label: '自动模式', desc: '根据时间和经纬度自动切换日夜' },
]

// 导航面板组件定义（UIComponentConfig.UIComponent 枚举）
export const UI_COMPONENTS = [
  { key: 'INFO_VIEW', label: '导航信息面板', default: true, desc: '导航状态信息面板' },
  { key: 'STATUS_VIEW', label: '定位信号状态', default: true, desc: 'GPS定位信号状态' },
  { key: 'ENLARGE_INFO_VIEW', label: '路口放大图', default: true, desc: '路口放大图' },
  { key: 'GUIDE_LANE_VIEW', label: '车道线', default: true, desc: '车道线指示' },
  { key: 'BOTTOM_PANEL_VIEW', label: '底部面板', default: true, desc: '底部导航信息面板' },
  { key: 'SPEED_VIEW_VIEW', label: '车速表', default: true, desc: '迈速表显示' },
  { key: 'SERVICE_AREA_VIEW', label: '服务区', default: true, desc: '服务区提示' },
  { key: 'ROAD_LIMIT_VIEW', label: '道路限速', default: true, desc: '当前道路限速视图' },
  { key: 'TRAFFIC_BAR_VIEW', label: '路况条', default: true, desc: '剩余路线路况光柱' },
  { key: 'ROUTE_EXPLAIN_VIEW', label: '路线解释', default: true, desc: '路线解释性视图' },
  { key: 'CONTINUE_VIEW', label: '继续导航', default: true, desc: '继续导航视图' },
  { key: 'TOAST_TIPS_VIEW', label: 'Toast提示', default: true, desc: 'Toast提示信息' },
  { key: 'PREVIEW_SWITCH_VIEW', label: '全览按钮', default: true, desc: '全览模式切换按钮' },
  { key: 'ROAD_TYPE_SWITCH_VIEW', label: '道路类型切换', default: true, desc: '道路类型切换按钮' },
  { key: 'MAP_TRAFFIC_SWITCH_VIEW', label: '路况按钮', default: true, desc: '底图路况开关按钮' },
  { key: 'ZOOM_CONTROLLER_VIEW', label: '缩放控件', default: true, desc: 'ZoomBar缩放控件' },
  { key: 'TTS_MUTE_SWITCH_VIEW', label: 'TTS静音', default: true, desc: '语音播报静音按钮' },
  { key: 'REROUTE_SWITCH_VIEW', label: '路线刷新', default: true, desc: '路线刷新按钮' },
  { key: 'OVER_SPEED_ANIMATION_VIEW', label: '超速动画', default: true, desc: '超速动画提示' },
  { key: 'ROUTE_RECOMMEND_VIEW', label: '路线推荐', default: true, desc: '路线推荐视图' },
  { key: 'EXIT_VIEW', label: '出口信息', default: true, desc: '高速出口信息视图' },
  { key: 'SETTING_VIEW', label: '设置按钮', default: true, desc: '导航设置按钮' },
]

const STORAGE_KEY = 'mapSettings'

function getDefaultSettings() {
  const uiComponentConfig = {}
  UI_COMPONENTS.forEach(c => { uiComponentConfig[c.key] = c.default })

  return {
    // 地图模式设置（非导航状态下）
    mapMode: {
      mapStyle: 1,              // 地图底图样式（1=标准, 2=夜间, 3=卫星）
      trafficEnabled: true,     // 实时路况
      showVehicleMarker: true,  // 车辆位置标记
    },
    // 导航模式设置（导航状态下）
    naviMode: {
      mapStyle: 2,              // 导航时地图底图默认夜间
      dayNightMode: 2,          // 导航日夜模式（1=DAY日间, 2=NIGHT夜间, 3=AUTO自动）
      trafficEnabled: true,     // 实时路况
      uiComponentConfig,        // 导航面板组件开关
    },
  }
}

export const useMapSettingsStore = defineStore('mapSettings', () => {
  const stored = uni.getStorageSync(STORAGE_KEY)
  const defaults = getDefaultSettings()

  // 兼容旧版设置结构
  let merged
  if (stored) {
    merged = { ...defaults }
    if (stored.mapMode) {
      merged.mapMode = { ...defaults.mapMode, ...stored.mapMode }
    } else {
      // 旧版结构迁移：mapType → mapStyle
      merged.mapMode.mapStyle = stored.mapType === 3 ? 2 : (stored.mapType || 1)
      merged.mapMode.trafficEnabled = stored.trafficEnabled !== undefined ? stored.trafficEnabled : true
      merged.mapMode.showVehicleMarker = stored.showVehicleMarker !== undefined ? stored.showVehicleMarker : true
    }
    if (stored.naviMode) {
      merged.naviMode = { ...defaults.naviMode, ...stored.naviMode }
    }
    // 迁移旧版 uiComponentConfig
    if (stored.uiComponentConfig && !stored.naviMode) {
      merged.naviMode.uiComponentConfig = { ...defaults.naviMode.uiComponentConfig, ...stored.uiComponentConfig }
    }
  } else {
    merged = defaults
  }

  // 兼容旧版 naviUITheme → dayNightMode
  if (merged.naviMode.dayNightMode === undefined) {
    if (merged.naviMode.naviUITheme !== undefined) {
      // 旧版 naviUITheme: 1=经典(白浅) → 1=DAY, 2=暗色(墨渊) → 2=NIGHT
      merged.naviMode.dayNightMode = merged.naviMode.naviUITheme
      delete merged.naviMode.naviUITheme
    } else {
      merged.naviMode.dayNightMode = 2
    }
  }

  // 确保 uiComponentConfig 包含所有 key
  UI_COMPONENTS.forEach(c => {
    if (merged.naviMode.uiComponentConfig[c.key] === undefined) {
      merged.naviMode.uiComponentConfig[c.key] = c.default
    }
  })

  const settings = ref(merged)

  const mapStyleLabel = computed(() => {
    const found = MAP_STYLES.find(t => t.key === settings.value.mapMode.mapStyle)
    return found ? found.label : '标准'
  })

  const naviMapStyleLabel = computed(() => {
    const found = MAP_STYLES.find(t => t.key === settings.value.naviMode.mapStyle)
    return found ? found.label : '夜间'
  })

  const naviDayNightModeLabel = computed(() => {
    const found = NAVI_DAY_NIGHT_MODES.find(t => t.key === settings.value.naviMode.dayNightMode)
    return found ? found.label : '夜间'
  })

  function save() {
    uni.setStorageSync(STORAGE_KEY, { ...settings.value })
  }

  // 地图模式设置
  function setMapModeStyle(style) {
    settings.value.mapMode.mapStyle = style
    save()
  }
  function setMapModeTraffic(enabled) {
    settings.value.mapMode.trafficEnabled = enabled
    save()
  }
  function setMapModeVehicleMarker(show) {
    settings.value.mapMode.showVehicleMarker = show
    save()
  }

  // 导航模式设置
  function setNaviModeStyle(style) {
    settings.value.naviMode.mapStyle = style
    save()
  }
  function setNaviDayNightMode(mode) {
    settings.value.naviMode.dayNightMode = mode
    save()
  }
  function setNaviModeTraffic(enabled) {
    settings.value.naviMode.trafficEnabled = enabled
    save()
  }
  function setUIComponent(key, visible) {
    settings.value.naviMode.uiComponentConfig[key] = visible
    save()
  }
  function setAllUIComponent(visible) {
    UI_COMPONENTS.forEach(c => {
      settings.value.naviMode.uiComponentConfig[c.key] = visible
    })
    save()
  }

  function resetToDefault() {
    settings.value = getDefaultSettings()
    save()
  }

  return {
    settings,
    mapStyleLabel,
    naviMapStyleLabel,
    naviDayNightModeLabel,
    MAP_STYLES,
    NAVI_DAY_NIGHT_MODES,
    UI_COMPONENTS,
    setMapModeStyle,
    setMapModeTraffic,
    setMapModeVehicleMarker,
    setNaviModeStyle,
    setNaviDayNightMode,
    setNaviModeTraffic,
    setUIComponent,
    setAllUIComponent,
    resetToDefault,
    save,
  }
})
