<template>
  <view class="debug-page" :class="themeClass">
    <NavBar title="BLE调试" />

    <scroll-view scroll-y class="main-scroll">
    <view class="section glass">
      <view class="section-header">
        <Icon name="Bluetooth" :size="20" themeColor="primary" />
        <text class="section-title">BLE 状态机</text>
      </view>
      <view class="state-row">
        <view
          v-for="s in stateList"
          :key="s.key"
          :class="['state-chip', { active: bleState.connectionState === s.key }]"
        >
          <text class="chip-text">{{ s.label }}</text>
        </view>
      </view>
      <view class="info-grid">
        <view class="info-item">
          <view class="info-top">
            <Icon name="Flash" :size="14" color="rgba(255,255,255,0.4)" />
            <text class="info-label">连接状态</text>
          </view>
          <text :class="['info-value', bleState.connectionState]">{{ bleState.connectionState }}</text>
        </view>
        <view class="info-item">
          <view class="info-top">
            <Icon name="Scan" :size="14" color="rgba(255,255,255,0.4)" />
            <text class="info-label">设备ID</text>
          </view>
          <text class="info-value mono">{{ bleState.deviceId || '无' }}</text>
        </view>
        <view class="info-item">
          <view class="info-top">
            <Icon name="Sync" :size="14" color="rgba(255,255,255,0.4)" />
            <text class="info-label">数据包计数</text>
          </view>
          <text class="info-value mono">{{ bleState.packetCount }}</text>
        </view>
        <view class="info-item">
          <view class="info-top">
            <Icon name="Warning" :size="14" color="rgba(255,255,255,0.4)" />
            <text class="info-label">重连次数</text>
          </view>
          <text class="info-value mono">{{ bleState.reconnectAttempts }}</text>
        </view>
      </view>
      <view v-if="bleState.lastError" class="error-bar">
        <Icon name="Warning" :size="16" themeColor="primary" />
        <text class="error-text">{{ bleState.lastError }}</text>
      </view>
    </view>

    <view class="section glass">
      <view class="section-header">
        <Icon name="Speedometer" :size="20" themeColor="primary" />
        <text class="section-title">模拟器控制</text>
      </view>
      <view class="sim-toggle-row">
        <view :class="['sim-btn', { active: isSimRunning }]" @click="toggleSimulator">
          <Icon :name="isSimRunning ? 'Power' : 'Flash'" :size="20" color="#fff" />
          <text class="sim-btn-text">{{ isSimRunning ? '停止模拟' : '启动模拟' }}</text>
        </view>
      </view>
      <view v-if="isSimRunning" class="sim-controls">
        <view class="control-group">
          <view class="control-label-row">
            <Icon name="Car" :size="16" color="rgba(255,255,255,0.5)" />
            <text class="control-label">挡位选择</text>
          </view>
          <view class="gear-row">
            <view
              v-for="g in ['P', 'R', 'N', 'D']"
              :key="g"
              :class="['gear-btn', { active: currentGear === g }]"
              @click="setGear(g)"
            >
              <text class="gear-text">{{ g }}</text>
            </view>
          </view>
        </view>
        <view class="control-group">
          <view class="control-label-row">
            <Icon name="SpeedometerOutline" :size="16" color="rgba(255,255,255,0.5)" />
            <text class="control-label">目标速度: {{ targetSpeed }} km/h</text>
          </view>
          <view class="slider-wrapper">
            <slider
              :min="0"
              :max="200"
              :value="targetSpeed"
              :step="5"
              activeColor="#007AFF"
              backgroundColor="rgba(255,255,255,0.08)"
              block-color="#007AFF"
              block-size="20"
              @change="onSpeedChange"
            />
          </view>
          <view class="slider-ticks">
            <text class="tick">0</text>
            <text class="tick">50</text>
            <text class="tick">100</text>
            <text class="tick">150</text>
            <text class="tick">200</text>
          </view>
        </view>
        <view class="control-group">
          <view class="control-label-row">
            <Icon name="BatteryCharging" :size="16" color="rgba(255,255,255,0.5)" />
            <text class="control-label">充电模拟</text>
          </view>
          <view class="charge-toggle-row">
            <view :class="['charge-btn', { active: isCharging }]" @click="toggleCharging">
              <Icon :name="isCharging ? 'BatteryCharging' : 'BatteryFull'" :size="18" :color="isCharging ? chargingColor : '#fff'" />
              <text class="charge-btn-text">{{ isCharging ? '停止充电' : '开始充电' }}</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <view class="section glass">
      <view class="section-header">
        <Icon name="Desktop" :size="20" themeColor="primary" />
        <text class="section-title">实时数据</text>
        <text class="section-badge">200ms</text>
      </view>
      <view class="data-grid">
        <view class="data-item" v-for="item in dataItems" :key="item.key">
          <view class="data-top">
            <Icon :name="dataIconMap[item.key]" :size="14" color="rgba(255,255,255,0.35)" />
            <text class="data-key">{{ item.label }}</text>
          </view>
          <text class="data-val">{{ item.value }}</text>
        </view>
      </view>
    </view>

    <view class="section glass">
      <view class="section-header">
        <Icon name="InformationCircle" :size="20" themeColor="primary" />
        <text class="section-title">真机测试步骤</text>
      </view>
      <view class="steps">
        <view class="step-item">
          <view class="step-num">1</view>
          <text class="step-text">HBuilderX → 运行 → 运行到手机/模拟器 → 选择设备</text>
        </view>
        <view class="step-item">
          <view class="step-num">2</view>
          <text class="step-text">打开车辆仪表盘页面，BLE 自动扫描连接</text>
        </view>
        <view class="step-item">
          <view class="step-num">3</view>
          <text class="step-text">或在调试页面启动模拟器测试数据流</text>
        </view>
        <view class="step-item">
          <view class="step-num">4</view>
          <text class="step-text">Tesla BLE 名称格式：Tesla + VIN后6位（如 Tesla 130307）</text>
        </view>
        <view class="step-item">
          <view class="step-num">5</view>
          <text class="step-text">BLE 是命令通道，实时数据仍通过云端 API 获取</text>
        </view>
      </view>
    </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  getBLEState,
  isSimulatorMode,
  setSimulatorShiftState,
  setSimulatorCharging,
  setSimulatorSpeed,
  BLEState
} from '@/utils/ble.js'
import { useVehicleData, startSimulatorMode, stopSimulatorMode } from '@/utils/vehicle-data.js'
import { useThemeStore } from '@/store/theme'
import NavBar from '@/components/NavBar/NavBar.vue'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)
const chargingColor = computed(() => themeStore.colors.charging)

const vehicleStore = useVehicleData()
const bleState = ref(getBLEState())
const isSimRunning = ref(false)
const currentGear = ref('P')
const targetSpeed = ref(60)
const isCharging = ref(false)

const dataIconMap = {
  speed: 'Speedometer',
  shift: 'Car',
  battery: 'BatteryFull',
  range: 'Navigate',
  charging: 'BatteryCharging',
  power: 'Flash',
  inside: 'Thermometer',
  outside: 'Sunny',
  ac: 'Snow',
  locked: 'LockClosed',
  source: 'Cloud',
  connected: 'Bluetooth'
}

const stateList = [
  { key: BLEState.IDLE, label: '空闲' },
  { key: BLEState.INITIALIZING, label: '初始化' },
  { key: BLEState.SCANNING, label: '扫描' },
  { key: BLEState.CONNECTING, label: '连接' },
  { key: BLEState.CONNECTED, label: '已连接' },
  { key: BLEState.RECONNECTING, label: '重连' },
  { key: BLEState.FAILED, label: '失败' }
]

const dataItems = computed(() => {
  const d = vehicleStore.data
  return [
    { key: 'speed', label: '速度', value: `${d.speed ?? '-'} km/h` },
    { key: 'shift', label: '挡位', value: d.gear ?? '-' },
    { key: 'battery', label: '电量', value: `${d.soc ?? '-'}%` },
    { key: 'range', label: '续航', value: `${d.range_km ?? '-'} km` },
    { key: 'charging', label: '充电', value: d.charging ? '充电中' : (d.charge_power > 0 ? '已连接' : '未充电') },
    { key: 'power', label: '功率', value: `${d.charge_power ?? '-'} kW` },
    { key: 'inside', label: '内温', value: `${d.inside_temp ?? '-'}°C` },
    { key: 'outside', label: '外温', value: `${d.outside_temp ?? '-'}°C` },
    { key: 'ac', label: '空调', value: d.is_ac_on ? '开' : '关' },
    { key: 'locked', label: '锁车', value: d.locked ? '已锁' : '未锁' },
    { key: 'source', label: '数据源', value: vehicleStore.source },
    { key: 'connected', label: 'BLE连接', value: vehicleStore.bleConnected ? '是' : '否' }
  ]
})

let stateTimer = null

onMounted(() => {
  stateTimer = setInterval(() => {
    bleState.value = getBLEState()
    isSimRunning.value = isSimulatorMode()
  }, 200)
})

onUnmounted(() => {
  if (stateTimer) clearInterval(stateTimer)
})

function toggleSimulator() {
  if (isSimRunning.value) {
    stopSimulatorMode()
    isSimRunning.value = false
    currentGear.value = 'P'
    isCharging.value = false
  } else {
    startSimulatorMode()
    isSimRunning.value = true
  }
}

function setGear(g) {
  currentGear.value = g
  setSimulatorShiftState(g)
}

function onSpeedChange(e) {
  targetSpeed.value = e.detail.value
  setSimulatorSpeed(e.detail.value)
}

function toggleCharging() {
  isCharging.value = !isCharging.value
  setSimulatorCharging(isCharging.value)
  if (isCharging.value) {
    currentGear.value = 'P'
  }
}
</script>

<style lang="scss" scoped>
.debug-page {
  padding: 24rpx;
  background: var(--dark-page-bg);
  height: 100vh;
  overflow: hidden;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  padding-top: calc(var(--status-bar-height, 44px) + 88rpx);
}

.main-scroll {
  flex: 1;
  overflow: hidden;
}

.section {
  margin-bottom: 28rpx;
}

.section.glass {
  background: var(--dark-page-glass-bg);
  border: 1px solid var(--dark-page-glass-border);
  border-radius: 24rpx;
  padding: 32rpx;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 12rpx;
  margin-bottom: 28rpx;
}

.section-title {
  color: var(--dark-page-text);
  font-size: 28rpx;
  font-weight: 600;
  flex: 1;
}

.section-badge {
  background: rgba(37, 99, 235, 0.15);
  color: var(--color-primary);
  font-size: 20rpx;
  padding: 4rpx 14rpx;
  border-radius: 20rpx;
  font-weight: 500;
}

.state-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
  margin-bottom: 24rpx;
}

.state-chip {
  padding: 10rpx 20rpx;
  border-radius: 12rpx;
  background: var(--dark-page-glass-bg);
  border: 1px solid var(--dark-page-glass-border);
}

.state-chip.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  box-shadow: 0 4rpx 20rpx rgba(37, 99, 235, 0.35);
}

.chip-text {
  color: var(--dark-page-text-hint);
  font-size: 22rpx;
  font-weight: 500;
}

.state-chip.active .chip-text {
  color: #fff;
}

.info-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 16rpx;
}

.info-item {
  width: calc(50% - 8rpx);
  background: var(--dark-page-glass-bg);
  border: 1px solid var(--dark-page-glass-border);
  border-radius: 16rpx;
  padding: 16rpx 20rpx;
}

.info-top {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-bottom: 8rpx;
}

.info-label {
  color: var(--dark-page-text-hint);
  font-size: 20rpx;
}

.info-value {
  color: var(--dark-page-text);
  font-size: 26rpx;
  font-weight: 600;
}

.info-value.mono {
  font-family: 'Menlo', 'Courier New', monospace;
  font-size: 22rpx;
}

.info-value.connected {
  color: #5BE7C4;
}

.info-value.reconnecting {
  color: #fbbf24;
}

.info-value.failed {
  color: #2563EB;
}

.error-bar {
  display: flex;
  align-items: center;
  gap: 12rpx;
  margin-top: 20rpx;
  padding: 16rpx 20rpx;
  background: rgba(37, 99, 235, 0.1);
  border: 1px solid rgba(37, 99, 235, 0.25);
  border-radius: 12rpx;
}

.error-text {
  color: var(--color-primary);
  font-size: 22rpx;
  font-weight: 500;
  flex: 1;
}

.sim-toggle-row {
  margin-bottom: 8rpx;
}

.sim-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  padding: 24rpx;
  border-radius: 16rpx;
  background: var(--dark-page-glass-bg);
  border: 1px solid var(--dark-page-glass-border);
}

.sim-btn.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  box-shadow: 0 4rpx 24rpx rgba(37, 99, 235, 0.35);
}

.sim-btn-text {
  color: var(--dark-page-text);
  font-size: 28rpx;
  font-weight: 600;
}

.sim-controls {
  margin-top: 24rpx;
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.control-group {
  background: var(--dark-page-glass-bg);
  border: 1px solid var(--dark-page-glass-border);
  border-radius: 16rpx;
  padding: 20rpx 24rpx;
}

.control-label-row {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-bottom: 16rpx;
}

.control-label {
  color: var(--dark-page-text-hint);
  font-size: 22rpx;
  font-weight: 500;
}

.gear-row {
  display: flex;
  gap: 16rpx;
}

.gear-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20rpx 0;
  border-radius: 14rpx;
  background: var(--dark-page-glass-bg);
  border: 1px solid var(--dark-page-glass-border);
}

.gear-btn.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  box-shadow: 0 4rpx 16rpx rgba(37, 99, 235, 0.3);
}

.gear-text {
  color: var(--dark-page-text-hint);
  font-size: 32rpx;
  font-weight: 700;
  letter-spacing: 2rpx;
}

.gear-btn.active .gear-text {
  color: #fff;
}

.slider-wrapper {
  margin: 0 -8rpx;
}

.slider-ticks {
  display: flex;
  justify-content: space-between;
  padding: 0 4rpx;
  margin-top: -8rpx;
}

.tick {
  color: var(--dark-page-text-hint);
  font-size: 18rpx;
}

.charge-toggle-row {
  display: flex;
}

.charge-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  padding: 20rpx;
  border-radius: 14rpx;
  background: var(--dark-page-glass-bg);
  border: 1px solid var(--dark-page-glass-border);
}

.charge-btn.active {
  background: rgba(251, 191, 36, 0.12);
  border-color: rgba(251, 191, 36, 0.35);
  box-shadow: 0 4rpx 16rpx rgba(251, 191, 36, 0.15);
}

.charge-btn-text {
  color: var(--dark-page-text);
  font-size: 26rpx;
  font-weight: 500;
}

.data-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 14rpx;
}

.data-item {
  width: calc(50% - 7rpx);
  background: var(--dark-page-glass-bg);
  border: 1px solid var(--dark-page-glass-border);
  border-radius: 14rpx;
  padding: 16rpx 20rpx;
}

.data-top {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-bottom: 8rpx;
}

.data-key {
  color: var(--dark-page-text-hint);
  font-size: 20rpx;
}

.data-val {
  color: var(--dark-page-text);
  font-size: 26rpx;
  font-weight: 600;
}

.steps {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.step-item {
  display: flex;
  align-items: flex-start;
  gap: 16rpx;
  padding: 16rpx 20rpx;
  background: var(--dark-page-glass-bg);
  border: 1px solid var(--dark-page-glass-border);
  border-radius: 12rpx;
}

.step-num {
  min-width: 40rpx;
  height: 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(37, 99, 235, 0.15);
  color: var(--color-primary);
  font-size: 22rpx;
  font-weight: 700;
  border-radius: 10rpx;
}

.step-text {
  color: var(--dark-page-text-hint);
  font-size: 22rpx;
  line-height: 1.6;
  flex: 1;
}
</style>
