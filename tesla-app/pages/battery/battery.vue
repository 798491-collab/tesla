<template>
  <view class="battery-container" :class="themeClass">
    <NavBar title="电池数据" />

    <scroll-view class="battery-scroll" scroll-y :show-scrollbar="false">
      <view class="battery-body">

        <!-- SOC 概览 -->
        <view class="soc-overview card">
          <view class="soc-main-row">
            <view class="soc-circle">
              <text class="soc-percent" :style="{ color: socColor }">{{ socDisplay }}</text>
              <text class="soc-unit">%</text>
            </view>
            <view class="soc-summary">
              <view class="summary-item">
                <Icon name="BatteryFull" :size="18" themeColor="primary" />
                <text class="summary-label">可用SOC</text>
                <text class="summary-value">{{ formatValue(data.usable_soc, '%') }}</text>
              </view>
              <view class="summary-item">
                <Icon name="Navigate" :size="18" themeColor="primary" />
                <text class="summary-label">续航里程</text>
                <text class="summary-value">{{ formatValue(data.range_km, ' km', 0) }}</text>
              </view>
              <view class="summary-item">
                <Icon name="Thermometer" :size="18" themeColor="primary" />
                <text class="summary-label">电池温度</text>
                <text class="summary-value">{{ formatValue(data.battery_temp, '°C', 1) }}</text>
              </view>
              <view class="summary-item">
                <Icon name="Flash" :size="18" themeColor="primary" />
                <text class="summary-label">剩余能量</text>
                <text class="summary-value">{{ formatValue(data.energy_remaining, ' kWh', 1) }}</text>
              </view>
              <view class="summary-item">
                <Icon name="Navigate" :size="18" themeColor="primary" />
                <text class="summary-label">额定续航</text>
                <text class="summary-value">{{ formatValue(data.rated_range_km, ' km', 0) }}</text>
              </view>
              <view class="summary-item">
                <Icon name="Speedometer" :size="18" themeColor="primary" />
                <text class="summary-label">总里程</text>
                <text class="summary-value">{{ formatValue(data.odometer_km, ' km', 1) }}</text>
              </view>
            </view>
          </view>
          <view class="soc-bar-track">
            <view class="soc-bar-fill" :style="{ width: socDisplay + '%', backgroundColor: socColor }" />
          </view>
        </view>

        <!-- 电池包数据 -->
        <view class="section">
          <view class="section-title">
            <Icon name="BatteryFull" :size="16" themeColor="primary" />
            <text class="section-title-text">电池包</text>
          </view>
          <view class="card data-grid">
            <view class="data-item">
              <text class="data-label">电池包电压</text>
              <text class="data-value">{{ formatValue(data.pack_voltage, ' V', 1) }}</text>
            </view>
            <view class="data-item">
              <text class="data-label">电池包电流</text>
              <text class="data-value">{{ formatValue(data.pack_current, ' A', 1) }}</text>
            </view>
            <view class="data-item">
              <text class="data-label">电池电量</text>
              <text class="data-value">{{ formatValue(data.battery_level, '', 1) }}</text>
            </view>
            <view class="data-item">
              <text class="data-label">剩余能量</text>
              <text class="data-value">{{ formatValue(data.energy_remaining, ' kWh', 2) }}</text>
            </view>
          </view>
        </view>

        <!-- 充电数据 -->
        <view class="section">
          <view class="section-title">
            <Icon name="BatteryCharging" :size="16" themeColor="primary" />
            <text class="section-title-text">充电</text>
          </view>
          <view class="card">
            <view class="data-row">
              <text class="data-label">充电状态</text>
              <text class="data-value" :class="chargingStateClass">{{ chargingStateText }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电功率</text>
              <text class="data-value charging-val">{{ formatValue(data.charge_power, ' kW', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">DC充电功率</text>
              <text class="data-value">{{ formatValue(data.dc_charging_power, ' kW', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">AC充电功率</text>
              <text class="data-value">{{ formatValue(data.ac_charging_power, ' kW', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电电流</text>
              <text class="data-value">{{ formatValue(data.charge_amps, ' A', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电器电压</text>
              <text class="data-value">{{ formatValue(data.charger_voltage, ' V', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电限制SOC</text>
              <text class="data-value">{{ formatValue(data.charge_limit_soc, '%') }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充满剩余</text>
              <text class="data-value">{{ formatMinutes(data.minutes_to_full) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充满时间</text>
              <text class="data-value">{{ formatHours(data.time_to_full_charge) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电速度</text>
              <text class="data-value">{{ formatValue(data.charge_speed, ' km/h', 0) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">已充入能量</text>
              <text class="data-value">{{ formatValue(data.added_energy, ' kWh', 2) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">快充</text>
              <text class="data-value">{{ data.fast_charger_present ? '是' : '否' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">快充类型</text>
              <text class="data-value">{{ data.fast_charger_type || '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">超充状态</text>
              <text class="data-value">{{ data.supercharging != null ? (data.supercharging ? '超充中' : '否') : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电口盖</text>
              <text class="data-value">{{ data.charge_port_door_open ? '已打开' : '已关闭' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电口状态</text>
              <text class="data-value">{{ data.charge_port_open != null ? (data.charge_port_open ? '已打开' : '已关闭') : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电口锁扣</text>
              <text class="data-value">{{ formatChargePortLatch(data.charge_port_latch) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电线缆类型</text>
              <text class="data-value">{{ data.charging_cable_type || '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电器相数</text>
              <text class="data-value">{{ data.charger_phases != null ? data.charger_phases : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">请求充电电流</text>
              <text class="data-value">{{ formatValue(data.charge_current_request, ' A', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">最大充电电流</text>
              <text class="data-value">{{ formatValue(data.charge_current_request_max, ' A', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">DC充入能量</text>
              <text class="data-value">{{ formatValue(data.dc_charging_energy_in, ' kWh', 2) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">AC充入能量</text>
              <text class="data-value">{{ formatValue(data.ac_charging_energy_in, ' kWh', 2) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电使能请求</text>
              <text class="data-value">{{ data.charge_enable_request != null ? (data.charge_enable_request ? '是' : '否') : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">充电口寒冷模式</text>
              <text class="data-value">{{ data.charge_port_cold_weather_mode != null ? (data.charge_port_cold_weather_mode ? '是' : '否') : '--' }}</text>
            </view>
          </view>
        </view>

        <!-- 电池健康 -->
        <view class="section">
          <view class="section-title">
            <Icon name="Shield" :size="16" themeColor="primary" />
            <text class="section-title-text">电池健康</text>
          </view>
          <view class="card">
            <view class="data-row">
              <text class="data-label">模组最高温度</text>
              <text class="data-value temp-val">{{ formatValue(data.module_temp_max, '°C', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">模组最低温度</text>
              <text class="data-value temp-val">{{ formatValue(data.module_temp_min, '°C', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">最高温度模组</text>
              <text class="data-value">{{ data.num_module_temp_max != null ? '#' + data.num_module_temp_max : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">最低温度模组</text>
              <text class="data-value">{{ data.num_module_temp_min != null ? '#' + data.num_module_temp_min : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">电芯最高电压</text>
              <text class="data-value">{{ formatValue(data.brick_voltage_max, ' V', 3) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">电芯最低电压</text>
              <text class="data-value">{{ formatValue(data.brick_voltage_min, ' V', 3) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">最高电压电芯</text>
              <text class="data-value">{{ data.num_brick_voltage_max != null ? '#' + data.num_brick_voltage_max : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">最低电压电芯</text>
              <text class="data-value">{{ data.num_brick_voltage_min != null ? '#' + data.num_brick_voltage_min : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">电池加热器</text>
              <text class="data-value">{{ data.battery_heater_on != null ? (data.battery_heater_on ? '开启' : '关闭') : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">BMS状态</text>
              <text class="data-value">{{ data.bms_state || '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">BMS充满完成</text>
              <text class="data-value">{{ data.bms_full_charge_complete != null ? (data.bms_full_charge_complete ? '是' : '否') : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">DC-DC使能</text>
              <text class="data-value">{{ data.dcdc_enable != null ? (data.dcdc_enable ? '是' : '否') : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">生命周期能耗</text>
              <text class="data-value">{{ formatValue(data.lifetime_energy_used, ' kWh', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">预热使能</text>
              <text class="data-value">{{ data.preconditioning_enabled != null ? (data.preconditioning_enabled ? '是' : '否') : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">功率不足加热</text>
              <text class="data-value">{{ data.not_enough_power_to_heat != null ? (data.not_enough_power_to_heat ? '是' : '否') : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">绝缘电阻</text>
              <text class="data-value">{{ data.isolation_resistance != null ? data.isolation_resistance + ' Ω' : '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">高压互锁</text>
              <text class="data-value">{{ data.hvil != null ? data.hvil : '--' }}</text>
            </view>
          </view>
        </view>

        <!-- 外放电 -->
        <view class="section" v-if="hasPowershare">
          <view class="section-title">
            <Icon name="Flash" :size="16" themeColor="primary" />
            <text class="section-title-text">外放电</text>
          </view>
          <view class="card">
            <view class="data-row">
              <text class="data-label">外放电状态</text>
              <text class="data-value">{{ data.powershare_status || '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">外放电类型</text>
              <text class="data-value">{{ data.powershare_type || '--' }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">瞬时功率</text>
              <text class="data-value">{{ formatValue(data.powershare_instantaneous_power_kw, ' kW', 2) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">剩余时间</text>
              <text class="data-value">{{ formatValue(data.powershare_hours_left, ' h', 1) }}</text>
            </view>
            <view class="data-divider"></view>
            <view class="data-row">
              <text class="data-label">停止原因</text>
              <text class="data-value">{{ data.powershare_stop_reason || '--' }}</text>
            </view>
          </view>
        </view>

        <!-- 数据来源 -->
        <view class="source-info">
          <text class="source-text">数据来源: {{ sourceLabel }}</text>
        </view>

      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { computed } from 'vue'
import { useVehicleData } from '@/utils/vehicle-data.js'
import { useThemeStore } from '@/store/theme'
import NavBar from '@/components/NavBar/NavBar.vue'

const themeStore = useThemeStore()
const vehicleData = useVehicleData()
const themeClass = computed(() => themeStore.themeClass)

const data = computed(() => vehicleData.data || {})

const socDisplay = computed(() => {
  const soc = data.value.soc
  if (soc == null) return '--'
  return Math.round(soc)
})

const socColor = computed(() => {
  const soc = data.value.soc
  if (soc == null) return '#64748B'
  if (soc > 60) return '#22C55E'
  if (soc >= 20) return '#f59e0b'
  return '#f44336'
})

const chargingStateText = computed(() => {
  const state = data.value.charging_state || data.value.charge_state
  if (!state) return '--'
  const map = {
    'Charging': '充电中',
    'Complete': '充电完成',
    'Disconnected': '未连接',
    'Stopped': '已停止',
    'NoPower': '无电源',
    'Starting': '启动中',
  }
  return map[state] || state
})

const chargingStateClass = computed(() => {
  const state = data.value.charging_state || data.value.charge_state
  if (state === 'Charging') return 'state-charging'
  if (state === 'Complete') return 'state-complete'
  return ''
})

const hasPowershare = computed(() => {
  const d = data.value
  return d.powershare_status || d.powershare_type || d.powershare_instantaneous_power_kw != null
})

const sourceLabel = computed(() => {
  const source = vehicleData.source
  const map = { ws: 'WebSocket', cloud: '云端', ble: '蓝牙' }
  return map[source] || source
})

function formatValue(val, unit, decimals) {
  if (val == null) return '--'
  if (decimals !== undefined) {
    return Number(val).toFixed(decimals) + unit
  }
  return val + unit
}

function formatMinutes(val) {
  if (val == null) return '--'
  if (val <= 0) return '--'
  const h = Math.floor(val / 60)
  const m = val % 60
  if (h > 0) return `${h}h ${m}min`
  return `${m}min`
}

function formatHours(val) {
  if (val == null || val <= 0) return '--'
  const h = Math.floor(val)
  const m = Math.round((val - h) * 60)
  if (h > 0) return `${h}h ${m}min`
  return `${m}min`
}

function formatChargePortLatch(val) {
  if (val == null) return '--'
  const map = { 'Engaged': '已锁止', 'Disengaged': '已解锁', 'Blocked': '受阻' }
  return map[val] || val
}
</script>

<style lang="scss" scoped>
.battery-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--bg-page-solid, var(--bg-page));
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.battery-scroll {
  flex: 1;
  height: 0;
  margin-top: calc(var(--status-bar-height) + 88rpx);
}

.battery-body {
  padding: 24rpx 32rpx 40rpx;
}

/* SOC 概览 */
.soc-overview {
  padding: 32rpx;
  margin-bottom: 28rpx;
}

.soc-main-row {
  display: flex;
  align-items: center;
  gap: 32rpx;
  margin-bottom: 24rpx;
}

.soc-circle {
  display: flex;
  align-items: baseline;
  flex-shrink: 0;
}

.soc-percent {
  font-size: 72rpx;
  font-weight: 700;
  line-height: 1;
}

.soc-unit {
  font-size: 28rpx;
  font-weight: 500;
  color: var(--text-tertiary);
  margin-left: 4rpx;
}

.soc-summary {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.summary-label {
  font-size: 24rpx;
  color: var(--text-tertiary);
  flex: 1;
}

.summary-value {
  font-size: 26rpx;
  color: var(--text-primary);
  font-weight: 500;
}

.soc-bar-track {
  width: 100%;
  height: 8rpx;
  background: var(--bg-bar, rgba(255, 255, 255, 0.06));
  border-radius: 4rpx;
  overflow: hidden;
}

.soc-bar-fill {
  height: 100%;
  border-radius: 4rpx;
  transition: width 0.6s ease, background-color 0.3s ease;
}

/* 分组 */
.section {
  margin-bottom: 28rpx;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8rpx;
  padding: 0 8rpx 16rpx;
}

.section-title-text {
  font-size: 24rpx;
  font-weight: 500;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 2rpx;
}

/* 卡片 */
.card {
  background: var(--bg-card);
  border-radius: 24rpx;
  border: 1rpx solid var(--border-card);
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

/* 数据网格 (2列) */
.data-grid {
  display: flex;
  flex-wrap: wrap;
  padding: 8rpx 0;
}

.data-grid .data-item {
  width: 50%;
  padding: 20rpx 24rpx;
  box-sizing: border-box;
}

.data-grid .data-label {
  font-size: 22rpx;
  color: var(--text-tertiary);
  margin-bottom: 8rpx;
}

.data-grid .data-value {
  font-size: 30rpx;
  font-weight: 600;
  color: var(--text-primary);
}

/* 数据行 */
.data-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24rpx;
}

.data-label {
  font-size: 26rpx;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.data-value {
  font-size: 28rpx;
  font-weight: 500;
  color: var(--text-primary);
  text-align: right;
}

.data-divider {
  height: 1rpx;
  background: var(--border-divider);
  margin: 0 24rpx;
}

/* 充电状态颜色 */
.state-charging {
  color: #22C55E;
}

.state-complete {
  color: #3B82F6;
}

.charging-val {
  color: var(--color-success, #22C55E);
}

.temp-val {
  color: var(--color-warning, #f59e0b);
}

/* 数据来源 */
.source-info {
  text-align: center;
  padding: 24rpx 0;
}

.source-text {
  font-size: 22rpx;
  color: var(--text-tertiary);
}
</style>
