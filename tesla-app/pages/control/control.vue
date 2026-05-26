<template>
  <view class="control-container" :class="themeClass">
    <view class="control-header">
      <view class="status-bar"></view>
      <view class="header-bar">
        <text class="header-title">车辆控制</text>
      </view>
    </view>

    <scroll-view class="control-scroll" scroll-y :show-scrollbar="false">
    <view class="control-body" v-if="selectedVIN">
      <view class="vehicle-card" v-if="vehicles.length > 1">
        <picker @change="onVehicleChange" :value="vehicleIndex" :range="vehicleNames">
          <view class="vehicle-picker">
            <view class="vehicle-picker-left">
              <Icon name="Car" :size="20" themeColor="primary" />
              <text class="vehicle-picker-name">{{ vehicles[vehicleIndex]?.vehicle_name }}</text>
            </view>
            <Icon name="ChevronDown" :size="16" themeColor="header" />
          </view>
        </picker>
      </view>

      <view class="vehicle-status-row">
        <text class="status-label">车辆状态：</text>
        <view class="status-dot" :style="{ backgroundColor: stateColor }"></view>
        <text class="status-value" :style="{ color: stateColor }">{{ stateText }}</text>
      </view>

      <view class="control-grid">
        <view class="control-item" @click="sendCommand('door_lock')">
          <view class="control-icon" :class="{ active: locked }">
            <Icon name="LockClosed" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">上锁</text>
            <text class="control-sub">{{ locked ? '已上锁' : '未上锁' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('door_unlock')">
          <view class="control-icon">
            <Icon name="LockOpen" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">解锁</text>
            <text class="control-sub">解锁车门</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('auto_conditioning_start')">
          <view class="control-icon" :class="{ active: climateOn }">
            <Icon name="Snow" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">开空调</text>
            <text class="control-sub">{{ climateOn ? '运行中' : '已关闭' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('auto_conditioning_stop')">
          <view class="control-icon">
            <Icon name="Sunny" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">关空调</text>
            <text class="control-sub">关闭空调</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('actuate_trunk')">
          <view class="control-icon" :class="{ active: trunkOpen }">
            <Icon name="Exit" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">后备箱</text>
            <text class="control-sub">{{ trunkOpen ? '已打开' : '已关闭' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('actuate_frunk')">
          <view class="control-icon" :class="{ active: frunkOpen }">
            <Icon name="Exit" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">前备箱</text>
            <text class="control-sub">{{ frunkOpen ? '已打开' : '已关闭' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('set_sentry_mode')">
          <view class="control-icon" :class="{ active: sentryOn }">
            <Icon name="Shield" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">哨兵模式</text>
            <text class="control-sub">{{ sentryOn ? '已开启' : '已关闭' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('honk_horn')">
          <view class="control-icon">
            <Icon name="VolumeHigh" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">鸣笛</text>
            <text class="control-sub">鸣响喇叭</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('flash_lights')">
          <view class="control-icon">
            <Icon name="Bulb" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">闪灯</text>
            <text class="control-sub">闪烁灯光</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('charge_start')">
          <view class="control-icon" :class="{ active: charging }">
            <Icon name="BatteryCharging" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">开始充电</text>
            <text class="control-sub">{{ charging ? '充电中' : '未充电' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('charge_stop')">
          <view class="control-icon">
            <Icon name="Battery" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">停止充电</text>
            <text class="control-sub">停止充电</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('charge_port_door_open')">
          <view class="control-icon">
            <Icon name="Plug" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">打开充电口</text>
            <text class="control-sub">{{ chargePortOpen ? '已打开' : '已关闭' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('charge_port_door_close')">
          <view class="control-icon">
            <Icon name="Plug" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">关闭充电口</text>
            <text class="control-sub">关闭充电口</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('set_temps')">
          <view class="control-icon">
            <Icon name="Thermometer" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">设置温度</text>
            <text class="control-sub">{{ insideTemp ? insideTemp + '°C' : '未设置' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('remote_seat_heater')">
          <view class="control-icon" :class="{ active: seatHeaterOn }">
            <Icon name="Seat" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">座椅加热</text>
            <text class="control-sub">{{ seatHeaterOn ? '已开启' : '已关闭' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('remote_steering_wheel_heater')">
          <view class="control-icon" :class="{ active: steeringHeaterOn }">
            <Icon name="Steering" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">方向盘加热</text>
            <text class="control-sub">{{ steeringHeaterOn ? '已开启' : '已关闭' }}</text>
          </view>
        </view>
        <view class="control-item" @click="sendCommand('wake')">
          <view class="control-icon">
            <Icon name="Alarm" :size="24" themeColor="primary" />
          </view>
          <view class="control-content">
            <text class="control-title">唤醒车辆</text>
            <text class="control-sub">从休眠中唤醒</text>
          </view>
        </view>
      </view>
    </view>

    <view class="empty-state" v-else>
      <view class="empty-icon-wrap">
        <Icon name="CarOutline" :size="80" themeColor="inactiveLight" />
      </view>
      <text class="empty-title">暂无绑定车辆</text>
      <text class="empty-subtitle">请在车辆页面添加您的 Tesla</text>
    </view>
    <view class="tabbar-spacer"></view>
    </scroll-view>

    <view class="pairing-modal-mask" v-if="showPairingModal" @click="showPairingModal = false">
      <view class="pairing-modal" @click.stop>
        <view class="pairing-modal-header">
          <Icon name="Key" :size="28" themeColor="primary" />
          <text class="pairing-modal-title">虚拟钥匙未配对</text>
        </view>
        <view class="pairing-modal-body">
          <text class="pairing-modal-desc">远程控制需要先完成虚拟钥匙配对。请按以下步骤操作：</text>
          <view class="pairing-steps">
            <view class="pairing-step">
              <view class="step-num">1</view>
              <text class="step-text">点击下方按钮打开配对链接</text>
            </view>
            <view class="pairing-step">
              <view class="step-num">2</view>
              <text class="step-text">在 Tesla App 中确认添加钥匙</text>
            </view>
            <view class="pairing-step">
              <view class="step-num">3</view>
              <text class="step-text">等待车辆确认（需在线）</text>
            </view>
          </view>
          <view class="pairing-status" v-if="pairingChecking">
            <view class="pairing-spinner"></view>
            <text class="pairing-status-text">正在检查配对状态...</text>
          </view>
          <view class="pairing-status paired" v-else-if="pairingPaired">
            <Icon name="CheckmarkCircle" :size="20" themeColor="success" />
            <text class="pairing-status-text">配对成功！</text>
          </view>
        </view>
        <view class="pairing-modal-footer">
          <button class="pairing-btn-secondary" @click="showPairingModal = false">取消</button>
          <button class="pairing-btn-primary" @click="openPairingURL" :disabled="!pairingURL">
            <Icon name="Key" :size="16" color="#fff" />
            <text>打开配对链接</text>
          </button>
        </view>
        <view class="pairing-check-row" v-if="!pairingPaired">
          <text class="pairing-check-link" @click="checkPairingStatus">已配对？点击检查状态</text>
        </view>
      </view>
    </view>

    <view class="temp-modal-mask" v-if="showTempModal" @click="showTempModal = false">
      <view class="temp-modal" @click.stop>
        <view class="temp-modal-header">
          <text class="temp-modal-title">设置空调温度</text>
        </view>
        <view class="temp-modal-body">
          <view class="temp-row">
            <text class="temp-label">主驾温度</text>
            <view class="temp-control">
              <view class="temp-btn" @click="tempDriver = Math.max(15, tempDriver - 0.5)">-</view>
              <text class="temp-value">{{ tempDriver }}°C</text>
              <view class="temp-btn" @click="tempDriver = Math.min(28, tempDriver + 0.5)">+</view>
            </view>
          </view>
          <view class="temp-row">
            <text class="temp-label">副驾温度</text>
            <view class="temp-control">
              <view class="temp-btn" @click="tempPassenger = Math.max(15, tempPassenger - 0.5)">-</view>
              <text class="temp-value">{{ tempPassenger }}°C</text>
              <view class="temp-btn" @click="tempPassenger = Math.min(28, tempPassenger + 0.5)">+</view>
            </view>
          </view>
        </view>
        <view class="temp-modal-footer">
          <button class="pairing-btn-secondary" @click="showTempModal = false">取消</button>
          <button class="pairing-btn-primary" @click="confirmSetTemps">确认</button>
        </view>
      </view>
    </view>

    <view class="seat-modal-mask" v-if="showSeatModal" @click="showSeatModal = false">
      <view class="seat-modal" @click.stop>
        <view class="seat-modal-header">
          <text class="seat-modal-title">座椅加热</text>
        </view>
        <view class="seat-modal-body">
          <view class="seat-select-row">
            <text class="seat-select-label">选择座椅</text>
            <picker :range="seatNames" @change="onSeatChange" :value="seatIndex">
              <view class="seat-picker">{{ seatNames[seatIndex] }}</view>
            </picker>
          </view>
          <view class="seat-level-row">
            <view v-for="lv in 4" :key="lv - 1" class="seat-level-item" :class="{ active: seatLevel === lv - 1 }" @click="seatLevel = lv - 1">
              <text class="seat-level-text">{{ lv - 1 === 0 ? '关' : 'Lv' + (lv - 1) }}</text>
            </view>
          </view>
        </view>
        <view class="seat-modal-footer">
          <button class="pairing-btn-secondary" @click="showSeatModal = false">取消</button>
          <button class="pairing-btn-primary" @click="confirmSeatHeater">确认</button>
        </view>
      </view>
    </view>

    <TabBar :currentIndex="2" />
  </view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { getUserVehicles, getVehicleState, wakeVehicle, getPairingURL, getFleetStatus } from '@/api/vehicle.js'
import {
  doorLock, doorUnlock,
  autoConditioningStart, autoConditioningStop,
  honkHorn, flashLights,
  actuateTrunk, actuateFrunk,
  setSentryMode,
  chargeStart, chargeStop,
  chargePortDoorOpen, chargePortDoorClose,
  setTemps,
  remoteSeatHeater,
  remoteSteeringWheelHeater
} from '@/api/control.js'
import Icon from '@/components/Icon/Icon.vue'
import { useThemeStore } from '@/store/theme'
import { useVehicleStore } from '@/store/vehicle'
import { useVehicleData, initVehicleData, destroyVehicleData } from '@/utils/vehicle-data'
import { getDisplayStateLabel, getDisplayStateColor, canControlVehicle, isVehicleOnline, isVehicleCharging } from '@/utils/vehicle-state'
import TabBar from '@/components/TabBar/TabBar.vue'

const themeStore = useThemeStore()
const themeClass = computed(() => themeStore.themeClass)
const successColor = computed(() => themeStore.colors.success)
const primaryColor = computed(() => themeStore.colors.primary)
const headerIconColor = computed(() => themeStore.colors.headerIcon)
const inactiveIconColorLight = computed(() => themeStore.colors.inactiveIconLight)

const vehicleStore = useVehicleStore()
const vehicleDataStore = useVehicleData()
const vehicles = ref([])
const vehicleIndex = ref(0)
let statusTimer = null

const showPairingModal = ref(false)
const pairingURL = ref('')
const pairingChecking = ref(false)
const pairingPaired = ref(false)

const showTempModal = ref(false)
const tempDriver = ref(22)
const tempPassenger = ref(22)

const showSeatModal = ref(false)
const seatIndex = ref(0)
const seatLevel = ref(0)
const seatNames = ['主驾', '副驾', '后排左', '后排中', '后排右', '第三排']

const vehicleNames = computed(() => vehicles.value.map(v => v.vehicle_name))
const selectedVIN = computed(() => vehicles.value[vehicleIndex.value]?.vin)

const vehicleData = computed(() => vehicleDataStore.data)
const stateOutput = computed(() => vehicleDataStore.stateOutput)
const vehicleOnline = computed(() => isVehicleOnline(stateOutput.value))

const locked = computed(() => vehicleData.value?.locked !== false)
const climateOn = computed(() => vehicleData.value?.is_ac_on)
const trunkOpen = computed(() => vehicleData.value?.trunk_open)
const frunkOpen = computed(() => vehicleData.value?.frunk_open)
const sentryOn = computed(() => vehicleData.value?.sentry_mode)
const charging = computed(() => vehicleData.value?.charging || isVehicleCharging(stateOutput.value))
const chargePortOpen = computed(() => vehicleData.value?.charge_port_door_open)
const insideTemp = computed(() => vehicleData.value?.inside_temp)
const seatHeaterOn = computed(() => {
  const s = vehicleData.value?.seat_heater
  if (!s) return false
  return Object.values(s).some(v => v > 0)
})
const steeringHeaterOn = computed(() => vehicleData.value?.steering_wheel_heater)

const stateText = computed(() => getDisplayStateLabel(stateOutput.value, vehicleData.value))
const stateColor = computed(() => getDisplayStateColor(stateOutput.value, vehicleData.value))

onMounted(() => { loadVehicles() })

onShow(() => {
  if (vehicles.value.length > 0 && selectedVIN.value) {
    initVehicleData(selectedVIN.value)
  }
})

onHide(() => {
  destroyVehicleData()
})

onUnmounted(() => {
  destroyVehicleData()
})

watch(() => selectedVIN.value, (newVIN) => {
  if (newVIN) {
    initVehicleData(newVIN)
  }
})

const loadVehicles = () => {
  getUserVehicles().then((res) => {
    vehicles.value = res.data || []
    if (vehicles.value.length > 0 && selectedVIN.value) {
      initVehicleData(selectedVIN.value)
    }
  })
}

const onVehicleChange = (e) => {
  vehicleIndex.value = e.detail.value
}

const onSeatChange = (e) => {
  seatIndex.value = e.detail.value
}

const commandMap = {
  door_lock: doorLock,
  door_unlock: doorUnlock,
  auto_conditioning_start: autoConditioningStart,
  auto_conditioning_stop: autoConditioningStop,
  honk_horn: honkHorn,
  flash_lights: flashLights,
  actuate_trunk: actuateTrunk,
  actuate_frunk: actuateFrunk,
  set_sentry_mode: () => setSentryMode(selectedVIN.value, !sentryOn.value),
  charge_start: chargeStart,
  charge_stop: chargeStop,
  charge_port_door_open: chargePortDoorOpen,
  charge_port_door_close: chargePortDoorClose,
  remote_steering_wheel_heater: remoteSteeringWheelHeater,
}

const commandNames = {
  door_lock: '上锁',
  door_unlock: '解锁',
  auto_conditioning_start: '开空调',
  auto_conditioning_stop: '关空调',
  honk_horn: '鸣笛',
  flash_lights: '闪灯',
  actuate_trunk: '后备箱',
  actuate_frunk: '前备箱',
  set_sentry_mode: '哨兵模式',
  charge_start: '开始充电',
  charge_stop: '停止充电',
  charge_port_door_open: '打开充电口',
  charge_port_door_close: '关闭充电口',
  set_temps: '设置温度',
  remote_seat_heater: '座椅加热',
  remote_steering_wheel_heater: '方向盘加热',
  wake: '唤醒',
}

const sendCommand = (command) => {
  if (!selectedVIN.value) {
    uni.showToast({ title: '请先选择车辆', icon: 'none' })
    return
  }

  if (command === 'wake') {
    uni.showLoading({ title: '唤醒中...' })
    wakeVehicle(selectedVIN.value).then(() => {
      uni.showLoading({ title: '等待车辆上线...' })
      const checkOnline = (retries) => {
        if (retries <= 0) {
          uni.hideLoading()
          uni.showToast({ title: '唤醒命令已发送，请稍后刷新查看', icon: 'none' })
          return
        }
        setTimeout(() => {
          if (vehicleOnline.value) {
            uni.hideLoading()
            uni.showToast({ title: '车辆已上线', icon: 'success' })
          } else {
            checkOnline(retries - 1)
          }
        }, 3000)
      }
      checkOnline(10)
    }).catch((err) => {
      uni.hideLoading()
      uni.showToast({ title: err.message || '唤醒失败', icon: 'none' })
    })
    return
  }

  if (command === 'set_temps') {
    const t = vehicleStateInfo.value?.inside_temp
    if (t) {
      tempDriver.value = t
      tempPassenger.value = t
    }
    showTempModal.value = true
    return
  }

  if (command === 'remote_seat_heater') {
    seatLevel.value = 0
    showSeatModal.value = true
    return
  }

  const fn = commandMap[command]
  if (!fn) {
    uni.showToast({ title: '功能开发中', icon: 'none' })
    return
  }

  uni.showLoading({ title: '执行中...' })
  const promise = typeof fn === 'function' && fn.length === 0
    ? fn()
    : fn(selectedVIN.value)

  Promise.resolve(promise).then((res) => {
    uni.hideLoading()
    const status = res?.status || res?.data?.result
    if (status === 'waking') {
      uni.showToast({ title: '车辆唤醒中，请稍候...', icon: 'none', duration: 3000 })
    } else if (status === 'pending') {
      uni.showToast({ title: '命令已发送，等待确认...', icon: 'none', duration: 3000 })
    } else {
      uni.showToast({ title: `${commandNames[command] || '操作'}成功`, icon: 'success' })
    }
  }).catch((err) => {
    uni.hideLoading()
    const errMsg = (err.message || '').toLowerCase()
    if (errMsg.includes('public key not paired') || errMsg.includes('virtual key not paired')) {
      showPairingGuide(selectedVIN.value)
      return
    }
    if (errMsg.includes('waking') || errMsg.includes('vehicle waking')) {
      uni.showToast({ title: '车辆唤醒中，命令将自动重试', icon: 'none', duration: 3000 })
      return
    }
    if (errMsg.includes('pending') || errMsg.includes('timeout')) {
      uni.showToast({ title: '命令已发送，等待车辆确认', icon: 'none', duration: 3000 })
      return
    }
    uni.showToast({ title: err.message || `${commandNames[command] || '操作'}失败`, icon: 'none' })
  })
}

const confirmSetTemps = () => {
  showTempModal.value = false
  uni.showLoading({ title: '设置温度中...' })
  setTemps(selectedVIN.value, tempDriver.value, tempPassenger.value).then((res) => {
    uni.hideLoading()
    const status = res?.status
    if (status === 'waking') {
      uni.showToast({ title: '车辆唤醒中...', icon: 'none', duration: 3000 })
    } else if (status === 'pending') {
      uni.showToast({ title: '命令已发送，等待确认...', icon: 'none', duration: 3000 })
    } else {
      uni.showToast({ title: '温度设置成功', icon: 'success' })
    }
  }).catch((err) => {
    uni.hideLoading()
    const errMsg = (err.message || '').toLowerCase()
    if (errMsg.includes('public key not paired') || errMsg.includes('virtual key not paired')) {
      showPairingGuide(selectedVIN.value)
      return
    }
    if (errMsg.includes('waking')) {
      uni.showToast({ title: '车辆唤醒中，命令将自动重试', icon: 'none', duration: 3000 })
      return
    }
    uni.showToast({ title: err.message || '设置失败', icon: 'none' })
  })
}

const confirmSeatHeater = () => {
  showSeatModal.value = false
  uni.showLoading({ title: '设置座椅加热...' })
  remoteSeatHeater(selectedVIN.value, seatIndex.value, seatLevel.value).then((res) => {
    uni.hideLoading()
    const status = res?.status
    if (status === 'waking') {
      uni.showToast({ title: '车辆唤醒中...', icon: 'none', duration: 3000 })
    } else if (status === 'pending') {
      uni.showToast({ title: '命令已发送，等待确认...', icon: 'none', duration: 3000 })
    } else {
      uni.showToast({ title: '座椅加热设置成功', icon: 'success' })
    }
  }).catch((err) => {
    uni.hideLoading()
    const errMsg = (err.message || '').toLowerCase()
    if (errMsg.includes('public key not paired') || errMsg.includes('virtual key not paired')) {
      showPairingGuide(selectedVIN.value)
      return
    }
    if (errMsg.includes('waking')) {
      uni.showToast({ title: '车辆唤醒中，命令将自动重试', icon: 'none', duration: 3000 })
      return
    }
    uni.showToast({ title: err.message || '设置失败', icon: 'none' })
  })
}

const showPairingGuide = async (vin) => {
  pairingPaired.value = false
  pairingChecking.value = false
  showPairingModal.value = true

  try {
    const res = await getPairingURL(vin)
    pairingURL.value = res.data?.pairing_url || ''
  } catch (e) {
    pairingURL.value = ''
    uni.showToast({ title: '获取配对链接失败', icon: 'none' })
  }
}

const openPairingURL = () => {
  if (!pairingURL.value) return
  plus.runtime.openURL(pairingURL.value, (err) => {
    uni.setClipboardData({
      data: pairingURL.value,
      success: () => {
        uni.showToast({ title: '配对链接已复制，请在浏览器中打开', icon: 'none' })
      }
    })
  })
}

const checkPairingStatus = async () => {
  if (!selectedVIN.value) return
  pairingChecking.value = true
  pairingPaired.value = false

  try {
    const res = await getFleetStatus(selectedVIN.value)
    const keyPaired = res.data?.key_paired
    if (keyPaired) {
      pairingPaired.value = true
      setTimeout(() => {
        showPairingModal.value = false
        uni.showToast({ title: '虚拟钥匙配对成功！', icon: 'success' })
      }, 1500)
    } else {
      uni.showToast({ title: '尚未配对，请在 Tesla App 中确认', icon: 'none' })
    }
  } catch (e) {
    uni.showToast({ title: '检查失败，请稍后重试', icon: 'none' })
  } finally {
    pairingChecking.value = false
  }
}
</script>

<style lang="scss" scoped>
.control-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(180deg, var(--dark-page-bg) 0%, var(--bg-card) 100%);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-sizing: border-box;
}

.control-scroll {
  flex: 1;
  height: 0;
}

.status-bar {
  flex-shrink: 0;
  height: var(--status-bar-height);
}

.header-bar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32rpx;
  height: 60rpx;
}

.header-title {
  font-size: 40rpx;
  font-weight: 700;
  color: var(--dark-page-text);
}

.vehicle-status-row {
  display: flex;
  align-items: center;
  gap: 10rpx;
  margin-bottom: 32rpx;
}

.status-label {
  font-size: 24rpx;
  color: var(--dark-page-text-hint);
}

.status-dot {
  width: 14rpx;
  height: 14rpx;
  border-radius: 50%;
}

.status-value {
  font-size: 24rpx;
  font-weight: 500;
}

.control-body {
  flex: 1;
  padding: 0 32rpx;
}

.tabbar-spacer {
  height: 130rpx;
}

.vehicle-card {
  background: var(--dark-page-glass-bg);
  border: 1rpx solid var(--dark-page-glass-border);
  border-radius: 20rpx;
  margin-bottom: 24rpx;
}

.vehicle-picker {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20rpx 24rpx;
}

.vehicle-picker-left {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.vehicle-picker-name {
  font-size: 28rpx;
  font-weight: 600;
  color: var(--dark-page-text);
}

.control-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr;
  gap: 12rpx;
}

.control-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8rpx;
  padding: 20rpx 8rpx;
  background: var(--dark-page-icon-wrap-bg);
  border-radius: 16rpx;
  transition: background 0.2s ease;

  &:active {
    background: var(--dark-page-press-bg);
  }
}

.control-icon {
  width: 56rpx;
  height: 56rpx;
  border-radius: 50%;
  background: var(--dark-page-glass-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  &.active {
    background: linear-gradient(135deg, #5BE7C4, #3cc9a5);
  }
}

.control-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2rpx;
  min-width: 0;
}

.control-title {
  font-size: 22rpx;
  font-weight: 600;
  color: var(--dark-page-text);
  text-align: center;
}

.control-sub {
  font-size: 18rpx;
  color: var(--dark-page-text-hint);
  text-align: center;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 120rpx 40rpx;
  flex: 1;
}

.empty-icon-wrap {
  margin-bottom: 32rpx;
}

.empty-title {
  font-size: 34rpx;
  font-weight: 600;
  color: var(--dark-page-text-secondary);
  margin-bottom: 12rpx;
}

.empty-subtitle {
  font-size: 26rpx;
  color: var(--dark-page-text-hint);
}

.pairing-modal-mask,
.temp-modal-mask,
.seat-modal-mask {
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

.pairing-modal,
.temp-modal,
.seat-modal {
  width: 100%;
  max-width: 600rpx;
  background: var(--dark-page-card, #1F2937);
  border-radius: 32rpx;
  overflow: hidden;
  border: 1rpx solid var(--dark-page-card-border, rgba(255, 255, 255, 0.1));
}

.pairing-modal-header,
.temp-modal-header,
.seat-modal-header {
  display: flex;
  align-items: center;
  gap: 16rpx;
  padding: 36rpx 32rpx 20rpx;
}

.pairing-modal-title,
.temp-modal-title,
.seat-modal-title {
  font-size: 34rpx;
  font-weight: 700;
  color: var(--dark-page-text);
}

.pairing-modal-body {
  padding: 0 32rpx 24rpx;
}

.pairing-modal-desc {
  font-size: 26rpx;
  color: var(--dark-page-text-secondary);
  line-height: 1.6;
  display: block;
  margin-bottom: 24rpx;
}

.pairing-steps {
  margin-bottom: 24rpx;
}

.pairing-step {
  display: flex;
  align-items: center;
  gap: 16rpx;
  margin-bottom: 16rpx;

  .step-num {
    width: 40rpx;
    height: 40rpx;
    border-radius: 50%;
    background: var(--color-primary-light);
    color: var(--color-primary);
    font-size: 22rpx;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .step-text {
    font-size: 26rpx;
    color: var(--dark-page-text-secondary);
  }
}

.pairing-status {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 16rpx 20rpx;
  background: var(--dark-page-glass-bg);
  border-radius: 16rpx;

  &.paired {
    background: rgba(91, 231, 196, 0.1);
  }

  .pairing-spinner {
    width: 28rpx;
    height: 28rpx;
    border: 2rpx solid var(--dark-page-glass-border);
    border-top-color: var(--color-primary);
    border-radius: 50%;
    animation: pairing-spin 0.8s linear infinite;
  }

  .pairing-status-text {
    font-size: 24rpx;
    color: var(--dark-page-text-secondary);
  }
}

.pairing-modal-footer,
.temp-modal-footer,
.seat-modal-footer {
  display: flex;
  gap: 16rpx;
  padding: 0 32rpx 24rpx;
}

.pairing-btn-secondary {
  flex: 1;
  height: 80rpx;
  border-radius: 40rpx;
  background: var(--dark-page-glass-bg);
  color: var(--dark-page-text-secondary);
  font-size: 28rpx;
  border: none;
  font-weight: 500;
}

.pairing-btn-primary {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8rpx;
  height: 80rpx;
  border-radius: 40rpx;
  background: var(--gradient);
  color: #fff;
  font-size: 28rpx;
  border: none;
  font-weight: 500;

  &[disabled] {
    opacity: 0.5;
  }
}

.pairing-check-row {
  padding: 0 32rpx 28rpx;
  text-align: center;
}

.pairing-check-link {
  font-size: 24rpx;
  color: var(--color-primary);
  text-decoration: underline;
}

.temp-modal-body {
  padding: 0 32rpx 24rpx;
}

.temp-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20rpx 0;
  border-bottom: 1rpx solid var(--dark-page-glass-border);
}

.temp-label {
  font-size: 28rpx;
  color: var(--dark-page-text);
}

.temp-control {
  display: flex;
  align-items: center;
  gap: 24rpx;
}

.temp-btn {
  width: 56rpx;
  height: 56rpx;
  border-radius: 50%;
  background: var(--dark-page-glass-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32rpx;
  color: var(--dark-page-text);
  font-weight: 600;
}

.temp-value {
  font-size: 32rpx;
  font-weight: 700;
  color: var(--color-primary);
  min-width: 100rpx;
  text-align: center;
}

.seat-modal-body {
  padding: 0 32rpx 24rpx;
}

.seat-select-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.seat-select-label {
  font-size: 28rpx;
  color: var(--dark-page-text);
}

.seat-picker {
  padding: 12rpx 24rpx;
  background: var(--dark-page-glass-bg);
  border-radius: 12rpx;
  font-size: 26rpx;
  color: var(--dark-page-text);
}

.seat-level-row {
  display: flex;
  gap: 16rpx;
}

.seat-level-item {
  flex: 1;
  height: 72rpx;
  border-radius: 16rpx;
  background: var(--dark-page-glass-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;

  &.active {
    background: linear-gradient(135deg, #5BE7C4, #3cc9a5);
  }
}

.seat-level-text {
  font-size: 24rpx;
  color: var(--dark-page-text);
}

.seat-level-item.active .seat-level-text {
  color: #fff;
  font-weight: 600;
}

@keyframes pairing-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
