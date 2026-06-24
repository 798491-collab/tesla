<template>
  <view class="page" :class="themeClass" :style="{ paddingTop: 'calc(' + statusBarHeight + 'px + 88rpx)' }">
    <NavBar :title="currentVehicle?.vehicle_name || currentVehicle?.display_name || '车辆详情'" />

    <scroll-view scroll-y class="scroll-content" v-if="currentVehicle">
      <view class="content">
        <view class="glass-card">
          <view class="card-header">
            <Icon name="CarSport" :size="20" themeColor="primary" />
            <text class="card-title">车辆信息</text>
          </view>
          <view class="info-rows">
            <view class="info-row">
              <view class="info-left">
                <Icon name="Key" :size="16" themeColor="inactive" />
                <text class="info-label">VIN</text>
              </view>
              <text class="info-value mono">{{ maskVIN(currentVehicle.vin) }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Person" :size="16" themeColor="inactive" />
                <text class="info-label">显示名称</text>
              </view>
              <text class="info-value">{{ currentVehicle.display_name || currentVehicle.vehicle_name || '--' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="CarSport" :size="16" themeColor="inactive" />
                <text class="info-label">车型</text>
              </view>
              <text class="info-value">{{ vehicleData.car_type || '--' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Settings" :size="16" themeColor="inactive" />
                <text class="info-label">软件版本</text>
              </view>
              <text class="info-value">{{ vehicleData.version || '--' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="ColorPalette" :size="16" themeColor="inactive" />
                <text class="info-label">外观颜色</text>
              </view>
              <text class="info-value">{{ vehicleData.exterior_color || '--' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Speedometer" :size="16" themeColor="inactive" />
                <text class="info-label">轮毂</text>
              </view>
              <text class="info-value">{{ vehicleData.wheel_type || '--' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Flash" :size="16" themeColor="inactive" />
                <text class="info-label">状态</text>
              </view>
              <text class="info-value" :style="{ color: stateColor }">{{ stateText }}</text>
            </view>
          </view>
        </view>

        <view class="glass-card ai-analysis-card">
          <view class="card-header">
            <Icon name="Sparkles" :size="20" themeColor="primary" />
            <text class="card-title">AI 车辆分析</text>
            <view class="ai-header-action" @click="goToVehicleAI">
              <text class="ai-header-action-text">更多</text>
              <Icon name="ChevronForward" :size="14" themeColor="inactive" />
            </view>
          </view>
          <view v-if="aiResult" class="ai-content">
            <view class="ai-summary-box" @click="aiExpanded = !aiExpanded">
              <text class="ai-summary-text">{{ aiResult.summary || '点击查看详细分析' }}</text>
              <Icon :name="aiExpanded ? 'ChevronUp' : 'ChevronDown'" :size="16" themeColor="inactive" />
            </view>
            <view class="ai-detail" v-if="aiExpanded">
              <text class="ai-detail-line" v-for="(line, i) in getAiLines()" :key="i">{{ line }}</text>
            </view>
          </view>
          <view v-else-if="aiLoading" class="ai-loading-box">
            <view class="ai-spinner"></view>
            <text class="ai-loading-text">AI 正在分析中...</text>
          </view>
          <view v-else class="ai-empty-box">
            <text class="ai-empty-text">暂无AI分析报告</text>
            <view class="ai-trigger-btn" @click="triggerAnalysis">
              <Icon name="Sparkles" :size="14" color="#fff" />
              <text class="ai-trigger-text">生成分析</text>
            </view>
          </view>
        </view>

        <view class="glass-card" v-if="vehicleData">
          <view class="card-header">
            <Icon name="BatteryFull" :size="20" themeColor="primary" />
            <text class="card-title">电池</text>
          </view>
          <view class="battery-section">
            <view class="battery-top">
              <text class="battery-percent">{{ vehicleData.soc ?? 0 }}%</text>
              <text class="battery-range">{{ vehicleData.range_km != null ? Math.round(vehicleData.range_km) + ' km' : '-- km' }}</text>
            </view>
            <view class="battery-bar-track">
              <view class="battery-bar-fill" :style="{ width: (vehicleData.soc || 0) + '%' }"></view>
            </view>
            <view class="charging-row" v-if="vehicleData.charging">
              <Icon name="BatteryCharging" :size="16" themeColor="charging" />
              <text class="charging-text">充电中 {{ vehicleData.charge_power > 0 ? (Math.round(vehicleData.charge_power * 10) / 10) + ' kW · ' : '' }}{{ getChargeType(vehicleData).label }}充电</text>
            </view>
            <view class="info-rows" style="margin-top: 20rpx;">
              <view class="info-row">
                <view class="info-left">
                  <Icon name="BatteryFull" :size="16" themeColor="inactive" />
                  <text class="info-label">可用SOC</text>
                </view>
                <text class="info-value">{{ vehicleData.usable_soc != null ? vehicleData.usable_soc + '%' : '--' }}</text>
              </view>
              <view class="info-row">
                <view class="info-left">
                  <Icon name="Thermometer" :size="16" themeColor="inactive" />
                  <text class="info-label">电池温度</text>
                </view>
                <text class="info-value">{{ vehicleData.battery_temp != null ? vehicleData.battery_temp + '°C' : '--' }}</text>
              </view>
              <view class="info-row">
                <view class="info-left">
                  <Icon name="Navigate" :size="16" themeColor="inactive" />
                  <text class="info-label">额定续航</text>
                </view>
                <text class="info-value">{{ vehicleData.rated_range_km != null ? Math.round(vehicleData.rated_range_km) + ' km' : '--' }}</text>
              </view>
              <view class="info-row">
                <view class="info-left">
                  <Icon name="Flash" :size="16" themeColor="inactive" />
                  <text class="info-label">剩余能量</text>
                </view>
                <text class="info-value">{{ vehicleData.energy_remaining != null ? Number(vehicleData.energy_remaining).toFixed(1) + ' kWh' : '--' }}</text>
              </view>
              <view class="info-row">
                <view class="info-left">
                  <Icon name="BatteryCharging" :size="16" themeColor="inactive" />
                  <text class="info-label">电池包电压</text>
                </view>
                <text class="info-value">{{ vehicleData.pack_voltage != null ? Number(vehicleData.pack_voltage).toFixed(1) + ' V' : '--' }}</text>
              </view>
              <view class="info-row">
                <view class="info-left">
                  <Icon name="BatteryCharging" :size="16" themeColor="inactive" />
                  <text class="info-label">电池包电流</text>
                </view>
                <text class="info-value">{{ vehicleData.pack_current != null ? Number(vehicleData.pack_current).toFixed(1) + ' A' : '--' }}</text>
              </view>
              <view class="info-row" v-if="vehicleData.charging_state">
                <view class="info-left">
                  <Icon name="Flash" :size="16" themeColor="inactive" />
                  <text class="info-label">充电状态</text>
                </view>
                <text class="info-value" :style="{ color: vehicleData.charging ? chargingColor : vehicleData.charging_state === 'Complete' ? chargingCompleteColor : infoValueColor }">{{ formatChargingState(vehicleData.charging_state) }}</text>
              </view>
              <view class="info-row" v-if="vehicleData.charge_speed">
                <view class="info-left">
                  <Icon name="Speedometer" :size="16" themeColor="inactive" />
                  <text class="info-label">充电速度</text>
                </view>
                <text class="info-value">{{ vehicleData.charge_speed }} km/h</text>
              </view>
              <view class="info-row" v-if="vehicleData.charge_power > 0">
                <view class="info-left">
                  <Icon name="Flash" :size="16" themeColor="inactive" />
                  <text class="info-label">充电器功率</text>
                </view>
                <text class="info-value">{{ Math.round(vehicleData.charge_power * 10) / 10 }} kW</text>
              </view>
              <view class="info-row" v-if="vehicleData.charging">
                <view class="info-left">
                  <Icon name="Flash" :size="16" themeColor="inactive" />
                  <text class="info-label">充电类型</text>
                </view>
                <text class="info-value">{{ getChargeType(vehicleData).label }}充电</text>
              </view>
              <view class="info-row" v-if="vehicleData.voltage >= 50">
                <view class="info-left">
                  <Icon name="BatteryCharging" :size="16" themeColor="inactive" />
                  <text class="info-label">充电电压</text>
                </view>
                <text class="info-value">{{ vehicleData.voltage }} V</text>
              </view>
              <view class="info-row" v-if="vehicleData.ampere > 0">
                <view class="info-left">
                  <Icon name="BatteryCharging" :size="16" themeColor="inactive" />
                  <text class="info-label">充电电流</text>
                </view>
                <text class="info-value">{{ vehicleData.ampere }} A</text>
              </view>
              <view class="info-row" v-if="vehicleData.charge_limit_soc">
                <view class="info-left">
                  <Icon name="Settings" :size="16" themeColor="inactive" />
                  <text class="info-label">充电限制</text>
                </view>
                <text class="info-value">{{ vehicleData.charge_limit_soc }}%</text>
              </view>
              <view class="info-row" v-if="vehicleData.minutes_to_full > 0">
                <view class="info-left">
                  <Icon name="Time" :size="16" themeColor="inactive" />
                  <text class="info-label">充满剩余</text>
                </view>
                <text class="info-value">{{ formatMinutes(vehicleData.minutes_to_full) }}</text>
              </view>
              <view class="info-row" v-if="vehicleData.time_to_full_charge > 0">
                <view class="info-left">
                  <Icon name="Time" :size="16" themeColor="inactive" />
                  <text class="info-label">充满时间</text>
                </view>
                <text class="info-value">{{ formatHours(vehicleData.time_to_full_charge) }}</text>
              </view>
              <view class="info-row" v-if="vehicleData.added_energy">
                <view class="info-left">
                  <Icon name="Add" :size="16" themeColor="inactive" />
                  <text class="info-label">本次充电</text>
                </view>
                <text class="info-value">{{ vehicleData.added_energy }} kWh</text>
              </view>
            </view>
          </view>
        </view>

        <view class="glass-card" v-if="vehicleData">
          <view class="card-header">
            <Icon name="Thermometer" :size="20" themeColor="primary" />
            <text class="card-title">空调温度</text>
          </view>
          <view class="climate-grid">
            <view class="climate-item">
              <view class="climate-icon-wrap">
                <Icon name="Home" :size="20" themeColor="info" />
              </view>
              <text class="climate-label">车内</text>
              <text class="climate-value">{{ vehicleData.inside_temp != null ? vehicleData.inside_temp.toFixed(1) + '°' : '--' }}</text>
            </view>
            <view class="climate-item">
              <view class="climate-icon-wrap">
                <Icon name="Sunny" :size="20" themeColor="warning" />
              </view>
              <text class="climate-label">车外</text>
              <text class="climate-value">{{ vehicleData.outside_temp != null ? vehicleData.outside_temp.toFixed(1) + '°' : '--' }}</text>
            </view>
          </view>
          <view class="info-rows" style="margin-top: 20rpx;">
            <view class="info-row">
              <view class="info-left">
                <Icon name="Snow" :size="16" themeColor="inactive" />
                <text class="info-label">空调</text>
              </view>
              <text class="info-value" :style="{ color: vehicleData.is_ac_on ? acOnColor : inactiveIconColor }">{{ vehicleData.is_ac_on ? '开启' : '关闭' }}</text>
            </view>
            <view class="info-row" v-if="vehicleData.is_climate_on">
              <view class="info-left">
                <Icon name="Thermometer" :size="16" themeColor="inactive" />
                <text class="info-label">空调状态</text>
              </view>
              <text class="info-value" :style="{ color: acOnColor }">运行中</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Person" :size="16" themeColor="inactive" />
                <text class="info-label">驾驶位温度</text>
              </view>
              <text class="info-value">{{ vehicleData.driver_temp_setting != null ? vehicleData.driver_temp_setting + '°C' : '--' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Person" :size="16" themeColor="inactive" />
                <text class="info-label">副驾温度</text>
              </view>
              <text class="info-value">{{ vehicleData.passenger_temp_setting != null ? vehicleData.passenger_temp_setting + '°C' : '--' }}</text>
            </view>
            <view class="info-row" v-if="vehicleData.hvac_fan_speed != null">
              <view class="info-left">
                <Icon name="Settings" :size="16" themeColor="inactive" />
                <text class="info-label">风扇档位</text>
              </view>
              <text class="info-value">{{ vehicleData.hvac_fan_speed }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Settings" :size="16" themeColor="inactive" />
                <text class="info-label">方向盘加热</text>
              </view>
              <text class="info-value" :style="{ color: vehicleData.steering_wheel_heater ? acOnColor : inactiveIconColor }">{{ vehicleData.steering_wheel_heater ? '开启' : '关闭' }}</text>
            </view>
            <view class="info-row" v-if="vehicleData.seat_heater">
              <view class="info-left">
                <Icon name="Person" :size="16" themeColor="inactive" />
                <text class="info-label">座椅加热</text>
              </view>
              <text class="info-value">{{ formatSeatHeater(vehicleData.seat_heater) }}</text>
            </view>
          </view>
        </view>

        <view class="glass-card" v-if="vehicleData">
          <view class="card-header">
            <Icon name="Speedometer" :size="20" themeColor="primary" />
            <text class="card-title">车辆状态</text>
          </view>
          <view class="info-rows">
            <view class="info-row">
              <view class="info-left">
                <Icon name="Speedometer" :size="16" themeColor="inactive" />
                <text class="info-label">速度</text>
              </view>
              <text class="info-value">{{ vehicleData.speed != null ? vehicleData.speed + ' km/h' : '--' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="CarSport" :size="16" themeColor="inactive" />
                <text class="info-label">挡位</text>
              </view>
              <text class="info-value">{{ vehicleData.gear ? formatGear(vehicleData.gear) : '--' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Navigate" :size="16" themeColor="inactive" />
                <text class="info-label">里程</text>
              </view>
              <text class="info-value">{{ vehicleData.odometer_km != null ? Number(vehicleData.odometer_km).toFixed(1) + ' km' : '--' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Navigate" :size="16" themeColor="inactive" />
                <text class="info-label">方向</text>
              </view>
              <text class="info-value">{{ vehicleData.heading != null ? vehicleData.heading + '°' : '--' }}</text>
            </view>
          </view>
        </view>

        <view class="glass-card" v-if="vehicleData">
          <view class="card-header">
            <Icon name="Shield" :size="20" themeColor="primary" />
            <text class="card-title">安全</text>
          </view>
          <view class="info-rows">
            <view class="info-row">
              <view class="info-left">
                <Icon name="LockClosed" :size="16" themeColor="inactive" />
                <text class="info-label">车锁</text>
              </view>
              <text class="info-value" :style="{ color: vehicleData.locked ? lockedColor : unlockedColor }">{{ vehicleData.locked ? '已锁定' : '未锁定' }}</text>
            </view>
            <view class="info-row">
              <view class="info-left">
                <Icon name="Eye" :size="16" themeColor="inactive" />
                <text class="info-label">哨兵模式</text>
              </view>
              <text class="info-value" :style="{ color: vehicleData.sentry_mode ? sentryOnColor : inactiveIconColor }">{{ vehicleData.sentry_mode ? '开启' : '关闭' }}</text>
            </view>
          </view>
          <view class="door-section">
            <text class="section-subtitle">车门</text>
            <view class="door-grid">
              <view class="door-item" :class="{ open: vehicleData.door_fl }">
                <Icon name="CarSport" :size="18" :themeColor="vehicleData.door_fl ? 'doorOpen' : 'inactiveLight'" />
                <text class="door-label">左前</text>
                <text class="door-status">{{ vehicleData.door_fl ? '开' : '关' }}</text>
              </view>
              <view class="door-item" :class="{ open: vehicleData.door_fr }">
                <Icon name="CarSport" :size="18" :themeColor="vehicleData.door_fr ? 'doorOpen' : 'inactiveLight'" />
                <text class="door-label">右前</text>
                <text class="door-status">{{ vehicleData.door_fr ? '开' : '关' }}</text>
              </view>
              <view class="door-item" :class="{ open: vehicleData.door_rl }">
                <Icon name="CarSport" :size="18" :themeColor="vehicleData.door_rl ? 'doorOpen' : 'inactiveLight'" />
                <text class="door-label">左后</text>
                <text class="door-status">{{ vehicleData.door_rl ? '开' : '关' }}</text>
              </view>
              <view class="door-item" :class="{ open: vehicleData.door_rr }">
                <Icon name="CarSport" :size="18" :themeColor="vehicleData.door_rr ? 'doorOpen' : 'inactiveLight'" />
                <text class="door-label">右后</text>
                <text class="door-status">{{ vehicleData.door_rr ? '开' : '关' }}</text>
              </view>
            </view>
          </view>
          <view class="trunk-section">
            <text class="section-subtitle">储物</text>
            <view class="trunk-grid">
              <view class="trunk-item" :class="{ open: vehicleData.frunk_open }">
                <Icon name="Exit" :size="18" :themeColor="vehicleData.frunk_open ? 'doorOpen' : 'inactiveLight'" />
                <text class="trunk-label">前备箱</text>
                <text class="trunk-status">{{ vehicleData.frunk_open ? '开' : '关' }}</text>
              </view>
              <view class="trunk-item" :class="{ open: vehicleData.trunk_open }">
                <Icon name="Exit" :size="18" :themeColor="vehicleData.trunk_open ? 'doorOpen' : 'inactiveLight'" />
                <text class="trunk-label">后备箱</text>
                <text class="trunk-status">{{ vehicleData.trunk_open ? '开' : '关' }}</text>
              </view>
            </view>
          </view>
          <view class="window-section">
            <text class="section-subtitle">车窗</text>
            <view class="door-grid">
              <view class="door-item" :class="{ open: vehicleData.fd_window }">
                <Icon name="Window" :size="18" :themeColor="vehicleData.fd_window ? 'doorOpen' : 'inactiveLight'" />
                <text class="door-label">左前</text>
                <text class="door-status">{{ vehicleData.fd_window ? '开' : '关' }}</text>
              </view>
              <view class="door-item" :class="{ open: vehicleData.fp_window }">
                <Icon name="Window" :size="18" :themeColor="vehicleData.fp_window ? 'doorOpen' : 'inactiveLight'" />
                <text class="door-label">右前</text>
                <text class="door-status">{{ vehicleData.fp_window ? '开' : '关' }}</text>
              </view>
              <view class="door-item" :class="{ open: vehicleData.rd_window }">
                <Icon name="Window" :size="18" :themeColor="vehicleData.rd_window ? 'doorOpen' : 'inactiveLight'" />
                <text class="door-label">左后</text>
                <text class="door-status">{{ vehicleData.rd_window ? '开' : '关' }}</text>
              </view>
              <view class="door-item" :class="{ open: vehicleData.rp_window }">
                <Icon name="Window" :size="18" :themeColor="vehicleData.rp_window ? 'doorOpen' : 'inactiveLight'" />
                <text class="door-label">右后</text>
                <text class="door-status">{{ vehicleData.rp_window ? '开' : '关' }}</text>
              </view>
            </view>
          </view>
        </view>

        <view class="glass-card" v-if="vehicleData && (vehicleData.tpms_fl || vehicleData.tpms_fr || vehicleData.tpms_rl || vehicleData.tpms_rr)">
          <view class="card-header">
            <Icon name="Car" :size="20" themeColor="primary" />
            <text class="card-title">胎压</text>
          </view>
          <view class="tire-grid">
            <view class="tire-item">
              <text class="tire-label">左前</text>
              <text class="tire-value">{{ vehicleData.tpms_fl ? vehicleData.tpms_fl.toFixed(1) : '--' }}</text>
              <text class="tire-unit">bar</text>
            </view>
            <view class="tire-item">
              <text class="tire-label">右前</text>
              <text class="tire-value">{{ vehicleData.tpms_fr ? vehicleData.tpms_fr.toFixed(1) : '--' }}</text>
              <text class="tire-unit">bar</text>
            </view>
            <view class="tire-item">
              <text class="tire-label">左后</text>
              <text class="tire-value">{{ vehicleData.tpms_rl ? vehicleData.tpms_rl.toFixed(1) : '--' }}</text>
              <text class="tire-unit">bar</text>
            </view>
            <view class="tire-item">
              <text class="tire-label">右后</text>
              <text class="tire-value">{{ vehicleData.tpms_rr ? vehicleData.tpms_rr.toFixed(1) : '--' }}</text>
              <text class="tire-unit">bar</text>
            </view>
          </view>
        </view>

        <view class="glass-card">
          <view class="card-header">
            <Icon name="Settings" :size="20" themeColor="primary" />
            <text class="card-title">快捷操作</text>
          </view>
          <view class="action-grid">
            <view class="action-item" @click="goToLocation">
              <view class="action-icon-wrap">
                <Icon name="Location" :size="24" themeColor="primary" />
              </view>
              <text class="action-label">定位</text>
            </view>
            <view class="action-item" @click="goToCharging">
              <view class="action-icon-wrap">
                <Icon name="Flash" :size="24" themeColor="primary" />
              </view>
              <text class="action-label">充电记录</text>
            </view>
            <view class="action-item" @click="goToTrip">
              <view class="action-icon-wrap">
                <Icon name="Navigate" :size="24" themeColor="primary" />
              </view>
              <text class="action-label">行驶记录</text>
            </view>
          </view>
        </view>


      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { useVehicleStore } from '@/store/vehicle'
import { useThemeStore } from '@/store/theme'
import { useVehicleData, initVehicleData, destroyVehicleData, suspendVehicleData } from '@/utils/vehicle-data'
import { getDisplayStateLabel, getDisplayStateColor, getChargeType } from '@/utils/vehicle-state'
import { getLatestAnalysis, triggerVehicleAnalysis } from '@/api/ai.js'
import Icon from '@/components/Icon/Icon.vue'
import NavBar from '@/components/NavBar/NavBar.vue'

const statusBarHeight = uni.getSystemInfoSync().statusBarHeight

const vehicleStore = useVehicleStore()
const themeStore = useThemeStore()
const vehicleDataStore = useVehicleData()
const currentVehicle = computed(() => vehicleStore.currentVehicle)
const vehicleData = computed(() => vehicleDataStore.data)
const stateOutput = computed(() => vehicleDataStore.stateOutput)
const themeClass = computed(() => themeStore.themeClass)
const primaryColor = computed(() => themeStore.colors.primary)
const inactiveIconColor = computed(() => themeStore.colors.inactiveIcon)
const inactiveIconColorLight = computed(() => themeStore.colors.inactiveIconLight)
const backIconColor = computed(() => themeStore.colors.headerIcon)
const infoValueColor = computed(() => themeStore.colors.infoValue)
const chargingColor = computed(() => themeStore.colors.charging)
const chargingCompleteColor = computed(() => themeStore.colors.chargingComplete)
const acOnColor = computed(() => themeStore.colors.acOn)
const lockedColor = computed(() => themeStore.colors.locked)
const unlockedColor = computed(() => themeStore.colors.unlocked)
const sentryOnColor = computed(() => themeStore.colors.sentryOn)
const doorOpenColor = computed(() => themeStore.colors.doorOpen)
const warningColor = computed(() => themeStore.colors.warning)
const infoBlueColor = computed(() => themeStore.colors.info)

const aiResult = ref(null)
const aiLoading = ref(false)
const aiExpanded = ref(false)

const stateText = computed(() => getDisplayStateLabel(stateOutput.value, vehicleData.value))

const stateColor = computed(() => getDisplayStateColor(stateOutput.value, vehicleData.value))

const maskVIN = (vin) => {
  if (!vin || vin.length <= 6) return vin || '--'
  return vin.slice(0, -6) + '******'
}

const formatChargingState = (s) => {
  if (s === 'Charging') return '充电中'
  if (s === 'Complete') return '已完成'
  if (s === 'Not Charging') return '未充电'
  if (s === 'NoPower') return '无电源'
  if (s === 'Starting') return '准备中'
  if (s === 'Stopped') return '已停止'
  if (s === 'Disconnected') return '未连接'
  return s || '未知'
}

const formatMinutes = (val) => {
  if (val == null || val <= 0) return '--'
  const h = Math.floor(val / 60)
  const m = val % 60
  if (h > 0) return `${h}h ${m}min`
  return `${m}min`
}

const formatHours = (val) => {
  if (val == null || val <= 0) return '--'
  const h = Math.floor(val)
  const m = Math.round((val - h) * 60)
  if (h > 0) return `${h}h ${m}min`
  return `${m}min`
}

const formatSeatHeater = (sh) => {
  if (!sh) return '--'
  const parts = []
  if (sh.left > 0) parts.push('左前' + sh.left + '档')
  if (sh.right > 0) parts.push('右前' + sh.right + '档')
  if (sh.rear_left > 0) parts.push('左后' + sh.rear_left + '档')
  if (sh.rear_right > 0) parts.push('右后' + sh.rear_right + '档')
  if (sh.rear_center > 0) parts.push('后中' + sh.rear_center + '档')
  return parts.length > 0 ? parts.join(' · ') : '全部关闭'
}

const formatGear = (g) => {
  if (!g) return 'P'
  const map = { D: '前进', R: '倒车', N: '空挡', P: '停车' }
  return map[g] || g
}

const goToLocation = () => {
  const vin = currentVehicle.value?.vin || ''
  uni.navigateTo({ url: '/pages/vehicle/location?vin=' + vin })
}

const goToCharging = () => {
  uni.navigateTo({ url: '/pages/charging/charging' })
}

const goToTrip = () => {
  uni.navigateTo({ url: '/pages/trip/trip' })
}

const loadAIAnalysis = async () => {
  const vin = currentVehicle.value?.vin
  if (!vin) return
  try {
    const res = await getLatestAnalysis(vin, 'vehicle')
    if (res?.data) {
      aiResult.value = res.data
    }
  } catch (e) {}
}

const triggerAnalysis = async () => {
  const vin = currentVehicle.value?.vin
  if (!vin || aiLoading.value) return
  aiLoading.value = true
  try {
    await triggerVehicleAnalysis(vin)
    setTimeout(async () => {
      await loadAIAnalysis()
      aiLoading.value = false
    }, 15000)
  } catch (e) {
    aiLoading.value = false
  }
}

const goToVehicleAI = () => {
  const vin = currentVehicle.value?.vin || ''
  uni.navigateTo({ url: `/pages/ai/vehicle?vin=${vin}` })
}

const getAiLines = () => {
  if (!aiResult.value?.result) return []
  return aiResult.value.result.split('\n').filter(l => l.trim()).map(l => l.replace(/^#{1,3}\s*/, '').replace(/\*\*/g, '').replace(/^[-*]\s*/, '• ').trim())
}

onMounted(() => {
  if (!currentVehicle.value) {
    uni.navigateBack()
  }
  if (currentVehicle.value?.vin) {
    initVehicleData(currentVehicle.value.vin)
  }
  loadAIAnalysis()
})

onShow(() => {
  if (currentVehicle.value?.vin) {
    initVehicleData(currentVehicle.value.vin)
  }
})

onHide(() => {
  suspendVehicleData()
})

onUnmounted(() => {
  destroyVehicleData()
})
</script>

<style lang="scss" scoped>
.page {
  height: 100vh;
  overflow: hidden;
  box-sizing: border-box;
  background: linear-gradient(170deg, var(--dark-page-bg) 0%, var(--bg-card) 40%, var(--bg-card-secondary) 100%);
  display: flex;
  flex-direction: column;
}

.scroll-content {
  flex: 1;
  height: 0;
}

.content {
  padding: 0 24rpx calc(60rpx + env(safe-area-inset-bottom));
}

.glass-card {
  background: var(--dark-page-glass-bg);
  border: 1rpx solid var(--dark-page-glass-border);
  border-radius: 28rpx;
  padding: 28rpx;
  margin-bottom: 20rpx;
}

.card-header {
  display: flex;
  gap: 12rpx;
  margin-bottom: 24rpx;

  :deep(.icon-wrapper) {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40rpx;
    height: 40rpx;
    line-height: 40rpx;
    overflow: visible;

    image {
      display: block;
    }
  }

  .card-title {
    font-size: 28rpx;
    font-weight: 600;
    color: var(--dark-page-text);
    line-height: 40rpx;
  }
}

.info-rows {
  .info-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16rpx 0;
    border-bottom: 1rpx solid var(--dark-page-divider);

    &:last-child {
      border-bottom: none;
      padding-bottom: 0;
    }

    &:first-child {
      padding-top: 0;
    }

    .info-left {
      display: flex;
      align-items: center;
      gap: 12rpx;

      :deep(.icon-wrapper) {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 28rpx;
        height: 28rpx;
        line-height: 28rpx;
        overflow: visible;

        image {
          display: block;
        }
      }

      .info-label {
        font-size: 26rpx;
        color: var(--dark-page-text-hint);
        line-height: 28rpx;
      }
    }

    .info-value {
      font-size: 26rpx;
      font-weight: 500;
      color: var(--dark-page-text);
      line-height: 28rpx;

      &.mono {
        font-family: 'SF Mono', 'Menlo', monospace;
        font-size: 22rpx;
        color: var(--dark-page-text-secondary);
        letter-spacing: 0.5rpx;
      }
    }
  }
}

.battery-section {
  .battery-top {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: 20rpx;

    .battery-percent {
      font-size: 56rpx;
      font-weight: 700;
      color: var(--dark-page-text);
      letter-spacing: -1rpx;
    }

    .battery-range {
      font-size: 28rpx;
      color: var(--dark-page-text-hint);
    }
  }

  .battery-bar-track {
    height: 16rpx;
    background: var(--dark-page-bar-bg);
    border-radius: 8rpx;
    overflow: hidden;

    .battery-bar-fill {
      height: 100%;
      border-radius: 8rpx;
      background: var(--bg-bar-fill);
    }
  }

  .charging-row {
    display: flex;
    align-items: center;
    gap: 8rpx;
    margin-top: 20rpx;
    padding: 14rpx 20rpx;
    background: rgba(251, 191, 36, 0.1);
    border-radius: 16rpx;

    .charging-text {
      font-size: 24rpx;
      color: var(--color-warning);
      font-weight: 500;
    }
  }
}

.climate-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20rpx;

  .climate-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 24rpx 16rpx;
    background: var(--dark-page-glass-bg);
    border-radius: 20rpx;

    .climate-icon-wrap {
      width: 56rpx;
      height: 56rpx;
      border-radius: 50%;
      background: var(--dark-page-glass-bg);
      display: flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 12rpx;
    }

    .climate-label {
      font-size: 22rpx;
      color: var(--dark-page-text-hint);
      margin-bottom: 6rpx;
    }

    .climate-value {
      font-size: 36rpx;
      font-weight: 700;
      color: var(--dark-page-text);
    }
  }
}

.section-subtitle {
  font-size: 24rpx;
  color: var(--dark-page-text-hint);
  font-weight: 500;
  display: block;
  margin-bottom: 16rpx;
  margin-top: 24rpx;
}

.door-section {
  .door-grid {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr 1fr;
    gap: 12rpx;
  }

  .door-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 20rpx 8rpx;
    background: var(--dark-page-glass-bg);
    border-radius: 16rpx;
    border: 1rpx solid var(--dark-page-glass-border);

    &.open {
      background: rgba(251, 191, 36, 0.08);
      border-color: rgba(251, 191, 36, 0.2);
    }

    .door-label {
      font-size: 20rpx;
      color: var(--dark-page-text-hint);
      margin-top: 8rpx;
    }

    .door-status {
      font-size: 22rpx;
      font-weight: 600;
      color: var(--dark-page-text-secondary);
      margin-top: 4rpx;
    }
  }
}

.trunk-section {
  .trunk-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12rpx;
  }

  .trunk-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 20rpx 16rpx;
    background: var(--dark-page-glass-bg);
    border-radius: 16rpx;
    border: 1rpx solid var(--dark-page-glass-border);

    &.open {
      background: rgba(251, 191, 36, 0.08);
      border-color: rgba(251, 191, 36, 0.2);
    }

    .trunk-label {
      font-size: 22rpx;
      color: var(--dark-page-text-hint);
      margin-top: 8rpx;
    }

    .trunk-status {
      font-size: 22rpx;
      font-weight: 600;
      color: var(--dark-page-text-secondary);
      margin-top: 4rpx;
    }
  }
}

.window-section {
  .door-grid {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr 1fr;
    gap: 12rpx;
  }

  .door-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 20rpx 8rpx;
    background: var(--dark-page-glass-bg);
    border-radius: 16rpx;
    border: 1rpx solid var(--dark-page-glass-border);

    &.open {
      background: rgba(251, 191, 36, 0.08);
      border-color: rgba(251, 191, 36, 0.2);
    }

    .door-label {
      font-size: 20rpx;
      color: var(--dark-page-text-hint);
      margin-top: 8rpx;
    }

    .door-status {
      font-size: 22rpx;
      font-weight: 600;
      color: var(--dark-page-text-secondary);
      margin-top: 4rpx;
    }
  }
}

.tire-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16rpx;

  .tire-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 24rpx 16rpx;
    background: var(--dark-page-glass-bg);
    border-radius: 20rpx;

    .tire-label {
      font-size: 22rpx;
      color: var(--dark-page-text-hint);
      margin-bottom: 8rpx;
    }

    .tire-value {
      font-size: 40rpx;
      font-weight: 700;
      color: var(--dark-page-text);
      line-height: 1;
    }

    .tire-unit {
      font-size: 20rpx;
      color: var(--dark-page-text-hint);
      margin-top: 6rpx;
    }
  }
}

.action-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16rpx;

  .action-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 24rpx 8rpx;
    background: var(--dark-page-glass-bg);
    border-radius: 20rpx;

    &:active {
      background: var(--dark-page-press-bg);
    }

    .action-icon-wrap {
      width: 72rpx;
      height: 72rpx;
      border-radius: 50%;
      background: var(--bg-icon-wrap);
      display: flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 10rpx;
    }

    .action-label {
      font-size: 22rpx;
      color: var(--dark-page-text-secondary);
    }
  }
}

.ai-analysis-card {
  .card-header {
    display: flex;
    gap: 12rpx;
    margin-bottom: 24rpx;

    .card-title {
      flex: 1;
      line-height: 40rpx;
    }

    .ai-header-action {
      display: flex;
      align-items: center;
      gap: 4rpx;
      padding: 8rpx 16rpx;
      background: var(--dark-page-glass-bg);
      border-radius: 16rpx;

      .ai-header-action-text {
        font-size: 22rpx;
        color: var(--dark-page-text-hint);
      }
    }
  }

  .ai-content {
    .ai-summary-box {
      display: flex;
      align-items: center;
      gap: 12rpx;
      padding: 16rpx 20rpx;
      background: rgba(255, 95, 109, 0.08);
      border: 1rpx solid rgba(255, 95, 109, 0.15);
      border-radius: 16rpx;

      .ai-summary-text {
        flex: 1;
        font-size: 26rpx;
        color: var(--dark-page-text-secondary);
        line-height: 1.6;
      }
    }

    .ai-detail {
      margin-top: 16rpx;
      padding-top: 16rpx;
      border-top: 1rpx solid var(--dark-page-divider);

      .ai-detail-line {
        font-size: 24rpx;
        color: var(--dark-page-text-secondary);
        line-height: 1.8;
        display: block;
      }
    }
  }

  .ai-loading-box {
    display: flex;
    align-items: center;
    gap: 12rpx;
    padding: 16rpx 0;

    .ai-spinner {
      width: 28rpx;
      height: 28rpx;
      border: 2rpx solid rgba(255, 95, 109, 0.2);
      border-top-color: var(--color-spinner);
      border-radius: 50%;
      animation: ai-spin 0.8s linear infinite;
    }

    .ai-loading-text {
      font-size: 24rpx;
      color: var(--dark-page-text-hint);
    }
  }

  .ai-empty-box {
    display: flex;
    align-items: center;
    justify-content: space-between;

    .ai-empty-text {
      font-size: 24rpx;
      color: var(--dark-page-text-hint);
    }

    .ai-trigger-btn {
      display: flex;
      align-items: center;
      gap: 6rpx;
      padding: 10rpx 24rpx;
      background: var(--gradient);
      border-radius: 20rpx;

      .ai-trigger-text {
        font-size: 22rpx;
        color: #ffffff;
        font-weight: 500;
      }
    }
  }
}

@keyframes ai-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}


</style>