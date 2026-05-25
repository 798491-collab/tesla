const ONLINE_STATE_MAP = {
  online:     { label: '在线',   color: '#5BE7C4', icon: 'Wifi',    markerColor: '#52c41a', vpColor: '#5EEAD4' },
  asleep:     { label: '休眠中', color: '#7C879B', icon: 'Moon',    markerColor: '#fa8c16', vpColor: '#94A3B8' },
  offline:    { label: '离线',   color: '#64748B', icon: 'WifiOff', markerColor: '#999999', vpColor: '#CBD5E1' },
  in_service: { label: '维修中', color: '#fbbf24', icon: 'Wrench',  markerColor: '#fa8c16', vpColor: '#FFB86B' },
  waking:     { label: '唤醒中', color: '#60a5fa', icon: 'Zap',     markerColor: '#60a5fa', vpColor: '#0F172A' },
  driving:    { label: '行驶中', color: '#5BE7C4', icon: 'CarSport', markerColor: '#52c41a', vpColor: '#5EEAD4' },
  charging:   { label: '充电中', color: '#5BE7C4', icon: 'Battery', markerColor: '#52c41a', vpColor: '#FFB86B' },
  climate_on: { label: '空调运行', color: '#60a5fa', icon: 'Thermometer', markerColor: '#60a5fa', vpColor: '#0F172A' },
}

const DRIVE_STATE_MAP = {
  parked:    { label: '驻车', icon: 'Parking' },
  driving:   { label: '行驶中', icon: 'CarSport' },
  reversing: { label: '倒车中', icon: 'CarSport' },
}

const CHARGE_STATE_MAP = {
  disconnected: { label: '未连接', color: '#64748B',  vpColor: '#94A3B8' },
  charging:     { label: '充电中', color: '#5BE7C4',  vpColor: '#FFB86B' },
  complete:     { label: '已充满', color: '#52c41a',  vpColor: '#5EEAD4' },
  supercharging:{ label: '超充中', color: '#a78bfa',  vpColor: '#7C6CFF' },
}

const COMMAND_STATE_MAP = {
  idle:     { label: '空闲', color: '#7C879B', vpColor: '#94A3B8' },
  sending:  { label: '发送中', color: '#60a5fa', vpColor: '#0F172A' },
  success:  { label: '成功', color: '#5BE7C4', vpColor: '#5EEAD4' },
  failed:   { label: '失败', color: '#FF6B6B', vpColor: '#FF6B6B' },
  timeout:  { label: '超时', color: '#fbbf24', vpColor: '#FFB86B' },
  rejected: { label: '被拒绝', color: '#FF6B6B', vpColor: '#FF6B6B' },
}

const REFRESH_INTERVALS = {
  online:  15000,
  asleep:  120000,
  offline: 300000,
}

export function getOnlineStateLabel(stateOutput) {
  const s = stateOutput?.state?.online_state
  return ONLINE_STATE_MAP[s]?.label || '未知'
}

export function getOnlineStateColor(stateOutput) {
  const s = stateOutput?.state?.online_state
  return ONLINE_STATE_MAP[s]?.color || '#7C879B'
}

export function getOnlineStateColorForTheme(stateOutput, theme) {
  const s = stateOutput?.state?.online_state
  const entry = ONLINE_STATE_MAP[s]
  if (!entry) return '#7C879B'
  return theme === 'visionpro' ? (entry.vpColor || entry.color) : entry.color
}

export function getOnlineStateIcon(stateOutput) {
  const s = stateOutput?.state?.online_state
  return ONLINE_STATE_MAP[s]?.icon || 'HelpCircle'
}

export function getOnlineStateMarkerColor(stateOutput) {
  const s = stateOutput?.state?.online_state
  return ONLINE_STATE_MAP[s]?.markerColor || '#999999'
}

export function getDriveStateLabel(stateOutput) {
  const s = stateOutput?.drive?.drive_state
  return DRIVE_STATE_MAP[s]?.label || '驻车'
}

export function getDriveStateIcon(stateOutput) {
  const s = stateOutput?.drive?.drive_state
  return DRIVE_STATE_MAP[s]?.icon || 'CarSport'
}

export function getChargeStateLabel(stateOutput) {
  const s = stateOutput?.charge?.charge_state
  return CHARGE_STATE_MAP[s]?.label || '未连接'
}

export function getChargeStateColor(stateOutput) {
  const s = stateOutput?.charge?.charge_state
  return CHARGE_STATE_MAP[s]?.color || '#64748B'
}

export function getChargeStateColorForTheme(stateOutput, theme) {
  const s = stateOutput?.charge?.charge_state
  const entry = CHARGE_STATE_MAP[s]
  if (!entry) return '#64748B'
  return theme === 'visionpro' ? (entry.vpColor || entry.color) : entry.color
}

export function getCommandStateLabel(stateOutput) {
  const s = stateOutput?.command?.command_state
  return COMMAND_STATE_MAP[s]?.label || '空闲'
}

export function getCommandStateColor(stateOutput) {
  const s = stateOutput?.command?.command_state
  return COMMAND_STATE_MAP[s]?.color || '#7C879B'
}

export function getCommandStateColorForTheme(stateOutput, theme) {
  const s = stateOutput?.command?.command_state
  const entry = COMMAND_STATE_MAP[s]
  if (!entry) return '#7C879B'
  return theme === 'visionpro' ? (entry.vpColor || entry.color) : entry.color
}

export function getConfidence(stateOutput) {
  return stateOutput?.state?.confidence ?? 0.5
}

export function getConfidenceLevel(stateOutput) {
  const c = getConfidence(stateOutput)
  if (c >= 0.8) return 'stable'
  if (c >= 0.5) return 'unstable'
  return 'critical'
}

export function getRefreshInterval(stateOutput) {
  const s = stateOutput?.state?.online_state
  return REFRESH_INTERVALS[s] || 15000
}

export function isVehicleOnline(stateOutput) {
  const s = stateOutput?.state?.online_state
  return s && s !== 'offline' && s !== 'unknown'
}

export function canControlVehicle(stateOutput) {
  const s = stateOutput?.state?.online_state
  return s === 'online' || s === 'driving' || s === 'charging' || s === 'climate_on' || s === 'waking'
}

export function isVehicleDriving(stateOutput) {
  return stateOutput?.drive?.drive_state === 'driving'
}

export function isVehicleCharging(stateOutput) {
  const s = stateOutput?.charge?.charge_state
  return s === 'charging' || s === 'supercharging'
}

export function isVehicleAsleep(stateOutput) {
  return stateOutput?.state?.online_state === 'asleep'
}

export function getStateChangedAt(stateOutput) {
  return stateOutput?.state?.changed_at || null
}

export function getLastSuccessAt(stateOutput) {
  return stateOutput?.meta?.last_success_at || null
}

export function isStateLocked(stateOutput) {
  const lockUntil = stateOutput?.meta?.state_lock_until || 0
  return lockUntil > Math.floor(Date.now() / 1000)
}

export function getTransitionCount(stateOutput) {
  return stateOutput?.meta?.state_transition_count || 0
}

export function getLastStateSource(stateOutput) {
  return stateOutput?.meta?.last_state_source || 'unknown'
}

export function getStateLabel(state) {
  return ONLINE_STATE_MAP[state]?.label || '未知'
}

export function getStateColor(state) {
  return ONLINE_STATE_MAP[state]?.color || '#7C879B'
}

export function getStateIcon(state) {
  return ONLINE_STATE_MAP[state]?.icon || 'HelpCircle'
}

export function getStateMarkerColor(state) {
  return ONLINE_STATE_MAP[state]?.markerColor || '#999999'
}

export const VEHICLE_ONLINE_STATES = Object.freeze(Object.keys(ONLINE_STATE_MAP))
