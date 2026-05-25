<template>
    <view class="dashboard" :class="themeClass">
        <scroll-view class="dashboard-scroll" scroll-y :show-scrollbar="false">
            <view class="dashboard-header">
                <!-- 3D模型 - 覆盖整个header区域 -->
                <!-- #ifdef APP-PLUS || H5 -->
                <TeslaScene ref="teslaSceneRef" :darkMode="isDarkTheme" :state="sceneState"
                    :darkModeProp="isDarkTheme" :licensePlate="licensePlateObj" class="dashboard-hero-3d"
                      :style="`top: ${statusBarHeight}px; `"
                    
                    
                    @onDoorClick="onDoorClick" @onTrunkClick="onTrunkClick" @onSceneReady="onModelLoaded" />
                <!-- 加载占位 -->
                <view v-if="!modelLoaded" class="model-loading-overlay">
                <!-- <view  class="model-loading-overlay"> -->
                    <view class="model-loading-spinner"></view>
                    <text class="model-loading-text">模型加载中...</text>
                </view>
                <!-- #endif -->
                <!-- #ifndef APP-PLUS || H5 -->
                <image class="dashboard-hero-bg" src="/static/dashboard-bg.jpg" mode="aspectFill" />
                <!-- #endif -->

                <view class="hero-left">
                    <!-- 状态栏占位 -->
                    <view class="status-bar"></view>
                    <!-- 顶部栏 -->
                    <view class="header">
                        <view class="tesla-logo">
                            <text class="logo-text">T E S L A</text>
                        </view>
                    </view>
                    <!-- 车辆信息 -->
                    <view class="vehicle-header" v-if="currentVehicle">
                        <view class="vehicle-model-row">
                            <text class="vehicle-model">{{ vehicleModelLabel }}</text>
                            <Icon name="ChevronDown" :size="16" themeColor="header" />
                        </view>
                        <view class="vehicle-status-row">
                            <view class="status-dot" :style="{ backgroundColor: stateColor }"></view>
                            <text class="vehicle-status-text">车辆状态 · {{ stateText }}</text>
                        </view>
                       
                       
                       <!-- 电量续航条 -->
                       <view class="battery-bar-section">
                         <view class="battery-bar-info">
                           <text class="battery-bar-percent"
                                 :style="{ color: batteryColor }">{{ batteryPercent }}%</text>
                           <text class="battery-bar-range">{{ rangeKm }} km</text>
                         </view>
                       
                      
                         <!-- 2. 轨道：中间层 -->
                         <view class="battery-bar-track">
                           <!-- 3. 电量填充：最上层 -->
                           <view class="battery-bar-fill"
                                 :style="{ width: batteryPercent + '%', backgroundColor: batteryColor }"></view>
                         </view>
                         <!-- 1. 流光条：在 track 下方，始终占位 -->
                         <view class="battery-flow-line" v-if="isChargingOrDebug"></view>
                                                
                       
                         <view class="battery-bar-detail" v-if="isChargingOrDebug">
                           <text class="battery-bar-charging-text">剩余 {{ chargeTimeRemaining }} · {{ chargeType }} · {{ chargeAmps }}A</text>
                         </view>
                       </view>
                       
                       
                       
                    </view>
                    <!-- 总里程 -->
                    <view class="range-section" v-if="currentVehicle">
                        <text class="range-value">{{ totalKm }}<text class="range-unit">km</text></text>
                        <text class="range-label">总里程</text>
                    </view>
                </view>

                <view class="debug-btn" @click="showDebugModal = true">
                    <text class="debug-btn-text">?</text>
                </view>
            </view>
            <!-- 快捷操作 -->
            <view class="quick-actions" v-if="currentVehicle">
                <view class="quick-action-item" @click="toggleLock">
                    <view class="quick-action-icon" :class="{ active: locked }">
                        <Icon :name="locked ? 'LockClosed' : 'LockOpen'" :size="24"
                            :themeColor="locked ? '' : 'quickAction'" :color="locked ? '#ffffff' : ''" />
                    </view>
                    <text class="quick-action-label">{{ locked ? '已上锁' : '已解锁' }}</text>
                </view>
                <view class="quick-action-item" @click="toggleClimate">
                    <view class="quick-action-icon" :class="{ active: climateOn }">
                        <Icon name="Snow" :size="24" :themeColor="climateOn ? '' : 'quickAction'"
                            :color="climateOn ? '#ffffff' : ''" />
                    </view>
                    <text class="quick-action-label">{{ climateOn ? '空调开' : '空调关' }}</text>
                </view>
                <view class="quick-action-item" @click="goToCharging">
                    <view class="quick-action-icon" :class="{ active: isCharging }">
                        <Icon name="Flash" :size="24" :themeColor="isCharging ? '' : 'quickAction'"
                            :color="isCharging ? '#ffffff' : ''" />
                    </view>
                    <text class="quick-action-label">{{ isCharging ? '正在充电' : '未充电' }}</text>
                </view>
                <view class="quick-action-item" @click="toggleTrunk">
                    <view class="quick-action-icon" :class="{ active: trunkOpen }">
                        <Icon name="Exit" :size="24" :themeColor="trunkOpen ? '' : 'quickAction'"
                            :color="trunkOpen ? '#ffffff' : ''" />
                    </view>
                    <text class="quick-action-label">{{ trunkOpen ? '后备箱开' : '后备箱关' }}</text>
                </view>
            </view>

            <!-- 六宫格菜单 -->
            <view class="menu-grid" v-if="currentVehicle">
                <view class="menu-grid-row">
                    <view class="menu-grid-item" @click="goToDetail">
                        <view class="menu-grid-icon">
                            <Icon name="CarSport" :size="26" themeColor="primary" />
                        </view>
                        <view class="menu-grid-content">
                            <text class="menu-grid-title">查看车辆</text>
                            <text class="menu-grid-subtitle">车辆详情与实时数据</text>
                        </view>
                    </view>
                    <view class="menu-grid-item">
                        <view class="menu-grid-icon climate-icon">
                            <view class="climate-temps">
                                <text class="climate-temp-inner">{{ formatTemp(insideTemp) }}</text>
                                <text class="climate-temp-outer">{{ formatTemp(outsideTemp) }}</text>
                            </view>
                        </view>
                        <view class="menu-grid-content">
                            <text class="menu-grid-title">空调温度</text>
                            <text class="menu-grid-subtitle">车内{{ formatTemp(insideTemp) }} ·
                                车外{{ formatTemp(outsideTemp) }}</text>
                        </view>
                    </view>
                </view>
                <view class="menu-grid-row">
                    <view class="menu-grid-item" @click="goToTrip">
                        <view class="menu-grid-icon">
                            <Icon name="Navigate" :size="26" themeColor="primary" />
                        </view>
                        <view class="menu-grid-content">
                            <text class="menu-grid-title">驾驶数据</text>
                            <text class="menu-grid-subtitle">最近行程与能耗</text>
                        </view>
                    </view>
                    <view class="menu-grid-item" @click="goToCharging">
                        <view class="menu-grid-icon">
                            <Icon name="Flash" :size="26" themeColor="primary" />
                        </view>
                        <view class="menu-grid-content">
                            <text class="menu-grid-title">充电信息</text>
                            <text class="menu-grid-subtitle">充电记录与充电网络</text>
                        </view>
                    </view>
                </view>
                <view class="menu-grid-row">
                    <view class="menu-grid-item" @click="goToInstrument">
                        <view class="menu-grid-icon">
                            <Icon name="Speedometer" :size="26" themeColor="primary" />
                        </view>
                        <view class="menu-grid-content">
                            <text class="menu-grid-title">仪表显示</text>
                            <text class="menu-grid-subtitle">个性化设置与显示</text>
                        </view>
                    </view>
                    <view class="menu-grid-item location-grid-item" @click="goToLocation">
                        <map v-if="hasLocation" class="location-card-map" :latitude="locationLat"
                            :longitude="locationLng" :markers="locationMarkers" :scale="15" :enable-scroll="false"
                            :enable-zoom="false" :layer-style="dashboardMapLayerStyle"></map>
                        <view class="location-card-overlay">
                            <view class="location-card-badge">
                                <Icon name="Location" :size="12" color="#fff" />
                                <text class="location-card-label">当前位置</text>
                            </view>
                        </view>
                        <view v-if="!hasLocation" class="location-card-fallback">
                            <Icon name="Location" :size="26" themeColor="primary" />
                        </view>
                    </view>
                </view>
            </view>

            <!-- 空状态 -->
            <view class="empty-state" v-if="!currentVehicle">
                <view class="empty-icon-wrap">
                    <Icon name="CarOutline" :size="80" themeColor="inactiveLight" />
                </view>
                <text class="empty-title">暂无绑定车辆</text>
                <text class="empty-subtitle">请在车辆页面添加您的 Tesla</text>
            </view>

            <view class="tabbar-spacer"></view>
        </scroll-view>

        <!-- 调试弹窗 -->
        <view class="debug-modal-mask" v-if="showDebugModal" @click="showDebugModal = false">
            <view class="debug-modal" @click.stop>
                <view class="debug-modal-header">
                    <text class="debug-modal-title">调试控制</text>
                    <view class="debug-modal-close" @click="showDebugModal = false">
                        <text class="debug-modal-close-text">✕</text>
                    </view>
                </view>
                <view class="debug-modal-section">
                    <text class="debug-modal-label">灯光</text>
                    <view class="debug-modal-btn" :class="{ active: sceneState.lights.headlightLow }"
                        @click="debugToggleLights">
                        {{ sceneState.lights.headlightLow ? '关灯' : '开灯' }}
                    </view>
                </view>
                <view class="debug-modal-section">
                    <text class="debug-modal-label">档位</text>
                    <view class="debug-modal-gears">
                        <view class="debug-modal-gear" :class="{ active: sceneState.gear === 'P' }"
                            @click="debugSetGear('P')">P</view>
                        <view class="debug-modal-gear" :class="{ active: sceneState.gear === 'R' }"
                            @click="debugSetGear('R')">R</view>
                        <view class="debug-modal-gear" :class="{ active: sceneState.gear === 'N' }"
                            @click="debugSetGear('N')">N</view>
                        <view class="debug-modal-gear" :class="{ active: sceneState.gear === 'D' }"
                            @click="debugSetGear('D')">D</view>
                    </view>
                </view>
                <view class="debug-modal-section">
                    <text class="debug-modal-label">充电特效</text>
                    <view class="debug-modal-btn" :class="{ active: sceneState.charging }" @click="debugToggleCharging">
                        {{ sceneState.charging ? '关闭充电' : '开启充电' }}
                    </view>
                </view>
                <view class="debug-modal-section">
                    <text class="debug-modal-label">后视镜</text>
                    <view class="debug-modal-btn" :class="{ active: sceneState.mirrorFolded }" @click="debugToggleMirror">
                        {{ sceneState.mirrorFolded ? '展开' : '折叠' }}
                    </view>
                </view>
                <view class="debug-modal-section">
                    <text class="debug-modal-label">哨兵模式</text>
                    <view class="debug-modal-btn" :class="{ active: sceneState.sentryMode }" @click="debugToggleSentry">
                        {{ sceneState.sentryMode ? '关闭' : '开启' }}
                    </view>
                </view>
                <view class="debug-modal-section">
                    <text class="debug-modal-label">远光灯</text>
                    <view class="debug-modal-btn" :class="{ active: sceneState.lights.headlightHigh }" @click="debugToggleHighBeam">
                        {{ sceneState.lights.headlightHigh ? '关闭' : '开启' }}
                    </view>
                </view>
            </view>
        </view>

        <TabBar :currentIndex="0" />
    </view>
</template>

<script setup>
    import {
        ref,
        computed,
        onMounted,
        onUnmounted,
        watch
    } from 'vue'
    import {
        onShow,
        onHide
    } from '@dcloudio/uni-app'
    import {
        useVehicleStore
    } from '@/store/vehicle'
    import {
        useUserStore
    } from '@/store/user'
    import {
        useVehicleData,
        initVehicleData,
        destroyVehicleData
    } from '@/utils/vehicle-data'
    import {
        getOnlineStateLabel,
        getOnlineStateColor,
        isVehicleOnline
    } from '@/utils/vehicle-state'
    import {
        useThemeStore
    } from '@/store/theme'
    import TabBar from '@/components/TabBar/TabBar.vue'
    import TeslaScene from '@/components/model/TeslaScene.vue'
    import {
        doorLock,
        doorUnlock,
        autoConditioningStart,
        autoConditioningStop,
        actuateTrunk
    } from '@/api/control.js'
    import {
        getVehicleState,
        wakeVehicle
    } from '@/api/vehicle.js'
    

    const statusBarHeight = ref(44) // 默认值
    
    onMounted(() => {
      // #ifdef APP-PLUS || MP-WEIXIN
      const windowInfo = uni.getWindowInfo()
      statusBarHeight.value = windowInfo.statusBarHeight
      // #endif
    })

    const vehicleStore = useVehicleStore()
    const themeStore = useThemeStore()
    const themeClass = computed(() => themeStore.themeClass)
    const isDarkTheme = computed(() => themeStore.resolvedTheme === 'dark' || themeStore.resolvedTheme === 'visionpro')
    const primaryColor = computed(() => themeStore.colors.primary)
    const inactiveIconColor = computed(() => themeStore.colors.inactiveIcon)
    const inactiveIconColorLight = computed(() => themeStore.colors.inactiveIconLight)
    const headerIconColor = computed(() => themeStore.colors.headerIcon)
    const quickActionIconColor = computed(() => themeStore.colors.quickActionIcon)
    const chargingColor = computed(() => themeStore.colors.charging)
    const chargingCompleteColor = computed(() => themeStore.colors.chargingComplete)

    const dashboardMapLayerStyle = computed(() => {
        const isDark = themeStore.resolvedTheme === 'dark' || themeStore.resolvedTheme === 'visionpro'
        if (isDark) {
            const styleId = import.meta.env.VITE_TENCENT_MAP_STYLE_DARK || '2'
            // #ifdef APP-PLUS
            return parseInt(styleId) || 2
            // #endif
            // #ifdef H5
            return 'style' + styleId
            // #endif
            // #ifndef APP-PLUS || H5
            return styleId
            // #endif
        }
        return 1
    })

    const userStore = useUserStore()
    const vehicleDataStore = useVehicleData()
    const currentVehicle = computed(() => vehicleStore.currentVehicle)
    const vehicleData = computed(() => vehicleDataStore.data)
    const stateOutput = computed(() => vehicleDataStore.stateOutput)
    const batteryPercent = computed(() => vehicleData.value?.soc || 0)
    const rangeKm = computed(() => vehicleData.value?.range_km || 0)
    const totalKm = computed(() => {
        const odometer = vehicleData.value?.odometer_km
        if (odometer === null || odometer === undefined) return '--'
        return Math.round(odometer).toLocaleString()
    })
    const insideTemp = computed(() => {
        const t = vehicleData.value?.inside_temp
        return (t !== null && t !== undefined && t !== 0) ? t : null
    })
    const outsideTemp = computed(() => {
        const t = vehicleData.value?.outside_temp
        return (t !== null && t !== undefined && t !== 0) ? t : null
    })
    const hasLocation = computed(() => {
        const lat = vehicleData.value?.latitude
        const lng = vehicleData.value?.longitude
        return lat && lng && lat !== 0 && lng !== 0
    })
    const locationLat = computed(() => {
        const lat = vehicleData.value?.latitude
        return (lat && lat !== 0) ? lat : 39.9042
    })
    const locationLng = computed(() => {
        const lng = vehicleData.value?.longitude
        return (lng && lng !== 0) ? lng : 116.4074
    })
    const locationMarkers = computed(() => {
        if (!hasLocation.value) return []
        return [{
            id: 1,
            latitude: vehicleData.value.latitude,
            longitude: vehicleData.value.longitude,
            title: '车辆位置',
            iconPath: '/static/car-marker.png',
            width: 30,
            height: 30,
        }]
    })
    const isCharging = computed(() => vehicleData.value?.charging === true)
    const chargeRate = computed(() => vehicleData.value?.charge_power || 0)
    const chargeEnergyAdded = computed(() => vehicleData.value?.added_energy || 0)
    const chargeLimit = computed(() => vehicleData.value?.charge_limit_soc || 90)
    const chargeAmps = computed(() => vehicleData.value?.charge_amps || 32)
    const chargeType = computed(() => {
        const type = vehicleData.value?.charge_type
        if (type === 'supercharger') return '超级充电桩'
        if (type === 'destination') return '目的地充电桩'
        return '交流充电桩'
    })
    const chargeTimeRemaining = computed(() => {
        const mins = vehicleData.value?.charge_time_remaining || 0
        if (mins <= 0) return '计算中...'
        const h = Math.floor(mins / 60)
        const m = Math.round(mins % 60)
        if (h > 0) return `${h}小时 ${m}分钟`
        return `${m}分钟`
    })
    const recentTripDistance = computed(() => {
        return (vehicleData.value?.recent_trip_distance || 32).toFixed(0)
    })
    const recentTripEfficiency = computed(() => {
        return (vehicleData.value?.recent_trip_efficiency || 22).toFixed(0)
    })
    const trunkOpen = computed(() => vehicleData.value?.trunk_open || vehicleData.value?.frunk_open)
    const locked = computed(() => vehicleData.value?.locked !== false)
    const climateOn = computed(() => vehicleData.value?.is_ac_on)

    const stateText = computed(() => getOnlineStateLabel(stateOutput.value))

    const stateColor = computed(() => getOnlineStateColor(stateOutput.value))

    const batteryColor = computed(() => {
        const p = batteryPercent.value
        if (p > 60) return '#4ade80'
        if (p >= 20) return '#facc15'
        return '#ef4444'
    })

    const vehicleModelLabel = computed(() => {
        const name = (currentVehicle.value?.display_name || '').toLowerCase()
        if (name.includes('model y') || name.includes('modely')) return 'Model Y'
        if (name.includes('model 3') || name.includes('model3')) return 'Model 3'
        if (name.includes('model s') || name.includes('models')) return 'Model S'
        if (name.includes('model x') || name.includes('modelx')) return 'Model X'
        return currentVehicle.value?.display_name || 'Tesla'
    })

    const formatTemp = (t) => {
        if (t === null || t === undefined) return '--'
        return `${t.toFixed(0)}°`
    }

    const goToProfile = () => {
        uni.reLaunch({
            url: '/pages/profile/profile'
        })
    }

    const goToControl = () => {
        uni.reLaunch({
            url: '/pages/control/control'
        })
    }

    const goToDetail = () => {
        uni.navigateTo({
            url: '/pages/vehicle/detail'
        })
    }

    const goToInstrument = () => {
        uni.navigateTo({
            url: '/pages/dashboard/instrument'
        })
    }

    const goToCharging = () => {
        uni.navigateTo({
            url: '/pages/charging/charging'
        })
    }

    const goToTrip = () => {
        uni.navigateTo({
            url: '/pages/trip/trip'
        })
    }

    const goToLocation = () => {
        const vin = currentVehicle.value?.vin || ''
        uni.navigateTo({
            url: '/pages/vehicle/location?vin=' + vin
        })
    }

    const currentVIN = computed(() => currentVehicle.value?.vin)

    const executeCommand = async (commandFn, commandName, needWake = true) => {
        if (!currentVIN.value) {
            uni.showToast({
                title: '请先选择车辆',
                icon: 'none'
            })
            return false
        }

        const vehicleOnline = isVehicleOnline(stateOutput.value)

        if (needWake && !vehicleOnline) {
            uni.showLoading({
                title: '唤醒车辆中...'
            })
            try {
                await wakeVehicle(currentVIN.value)
                uni.showLoading({
                    title: '车辆唤醒中，等待上线...'
                })
                await new Promise(resolve => setTimeout(resolve, 5000))
                for (let i = 0; i < 6; i++) {
                    try {
                        const stateRes = await getVehicleState(currentVIN.value)
                        if (stateRes.data?.online) {
                            break
                        }
                    } catch (e) {}
                    await new Promise(resolve => setTimeout(resolve, 3000))
                }
            } catch (err) {
                uni.hideLoading()
                uni.showToast({
                    title: '唤醒失败: ' + (err.message || '未知错误'),
                    icon: 'none'
                })
                return false
            }
        }

        uni.showLoading({
            title: '执行中...'
        })
        try {
            await commandFn(currentVIN.value)
            uni.hideLoading()
            uni.showToast({
                title: `${commandName}成功`,
                icon: 'success'
            })
            return true
        } catch (err) {
            uni.hideLoading()
            const errMsg = (err.message || '').toLowerCase()
            if (errMsg.includes('public key not paired') || errMsg.includes('virtual key not paired')) {
                uni.showToast({
                    title: '请先完成虚拟钥匙配对',
                    icon: 'none'
                })
                return false
            }
            uni.showToast({
                title: err.message || `${commandName}失败`,
                icon: 'none'
            })
            return false
        }
    }

    const toggleLock = async () => {
        if (locked.value) {
            await executeCommand(doorUnlock, '解锁')
        } else {
            await executeCommand(doorLock, '上锁')
        }
    }

    const toggleClimate = async () => {
        if (climateOn.value) {
            await executeCommand(autoConditioningStop, '关闭空调')
        } else {
            await executeCommand(autoConditioningStart, '开启空调')
        }
    }

    const toggleTrunk = async () => {
        await executeCommand(actuateTrunk, '后备箱操作')
    }

    // ===================== 3D模型车辆数据同步 =====================
    const teslaSceneRef = ref(null)
    const modelLoaded = ref(false)

    function onModelLoaded() {
        modelLoaded.value = true
        console.log('[Dashboard] 3D scene ready')
    }

    // 传递给 renderjs 的 state 对象
    const sceneState = ref({
        doors: {
            frontLeft: 0,
            frontRight: 0,
            rearLeft: 0,
            rearRight: 0
        },
        trunks: {
            rear: 0,
            front: 0
        },
        lights: {
            drl: false,
            headlightLow: false,
            headlightHigh: false,
            turnLeft: false,
            turnRight: false,
            tailLight: false,
            brakeLight: false,
            frontFog: false,
            rearFog: false,
            hazard: false,
        },
        charging: false,
        gear: 'P',
        speed: 0,
        mirrorFolded: false,
        locked: true,
        sentryMode: false,
        climateOn: false,
        soc: 0,
        chargeLimitSoc: 90,
        colors: {
            carpaint: '#1a1a2e',
            interior: '#2a2a2a',
            tire: '#1a1a1a',
            caliper: '#00a651',
            leather: '#1a1a1a',
            carpet: '#111111',
            chrome: '#c0c0c0',
            glass: '#3a5a5a',
            rim: '#c0c0c0',
        },
    })

    const licensePlateObj = computed(() => ({
        front: currentVehicle.value?.license_plate_front || '',
        rear: currentVehicle.value?.license_plate_rear || ''
    }))

    // 同步车辆数据到3D模型
    function syncVehicleDataToScene() {
        const d = vehicleData.value
        if (!d || !Object.keys(d).length) return

        const s = sceneState.value

        const gear = d.gear || d.shift_state || 'P'
        if (['P', 'R', 'N', 'D'].includes(gear)) {
            s.gear = gear
        }

        if (d.speed !== undefined && d.speed !== null) {
            s.speed = Number(d.speed)
        }

        const isVehicleCharging = d.charging === true || d.charging_state === 'Charging'
            || d.charge_state === 'charging' || d.charge_state === 'supercharging'
        s.charging = isVehicleCharging

        s.doors.frontLeft = d.door_fl ? 1 : 0
        s.doors.frontRight = d.door_fr ? 1 : 0
        s.doors.rearLeft = d.door_rl ? 1 : 0
        s.doors.rearRight = d.door_rr ? 1 : 0

        s.trunks.rear = d.trunk_open ? 1 : 0
        s.trunks.front = d.frunk_open ? 1 : 0

        s.locked = d.locked !== false
        s.mirrorFolded = d.mirror_folded === true
        s.sentryMode = d.sentry_mode === true
        s.climateOn = d.is_ac_on === true

        if (d.soc !== undefined && d.soc !== null) {
            s.soc = Number(d.soc)
        }
        if (d.charge_limit_soc !== undefined && d.charge_limit_soc !== null) {
            s.chargeLimitSoc = Number(d.charge_limit_soc)
        }

        if (isVehicleCharging) {
            // charging mode lights handled by TeslaScene internally
        } else {
            if (gear === 'D' || gear === 'R') {
                s.lights.headlightLow = true
                s.lights.tailLight = true
                s.lights.brakeLight = gear === 'R'
                s.lights.drl = false
                s.lights.headlightHigh = false
                s.lights.frontFog = false
                s.lights.rearFog = false
                s.lights.turnLeft = false
                s.lights.turnRight = false
                s.lights.hazard = false
            } else if (gear === 'N') {
                s.lights.drl = true
                s.lights.headlightLow = false
                s.lights.headlightHigh = false
                s.lights.tailLight = false
                s.lights.brakeLight = false
                s.lights.frontFog = false
                s.lights.rearFog = false
                s.lights.turnLeft = false
                s.lights.turnRight = false
                s.lights.hazard = false
            } else {
                s.lights.drl = false
                s.lights.headlightLow = false
                s.lights.headlightHigh = false
                s.lights.tailLight = false
                s.lights.brakeLight = false
                s.lights.frontFog = false
                s.lights.rearFog = false
                s.lights.turnLeft = false
                s.lights.turnRight = false
                s.lights.hazard = false
            }

            if (s.sentryMode && gear === 'P') {
                s.lights.hazard = true
            }
        }
    }

    // renderjs 回调
    function onDoorClick(doorKey) {
        const s = sceneState.value
        s.doors[doorKey] = s.doors[doorKey] > 0.5 ? 0 : 1
    }

    function onTrunkClick() {
        const s = sceneState.value
        s.trunks.rear = s.trunks.rear > 0.5 ? 0 : 1
    }

    function onSceneReady() {
        console.log('[Dashboard] 3D scene ready (legacy)')
    }

    // 调试控制
    const showDebugModal = ref(false)
    const isChargingOrDebug = computed(() => isCharging.value || sceneState.value.charging)

    const debugToggleLights = () => {
        const s = sceneState.value
        const isOn = s.lights.headlightLow
        s.lights.headlightLow = !isOn
        s.lights.tailLight = !isOn
        s.lights.drl = isOn
    }

    const debugSetGear = (gear) => {
        sceneState.value.gear = gear
    }

    const debugToggleCharging = () => {
        sceneState.value.charging = !sceneState.value.charging
    }

    const debugToggleMirror = () => {
        sceneState.value.mirrorFolded = !sceneState.value.mirrorFolded
    }

    const debugToggleSentry = () => {
        sceneState.value.sentryMode = !sceneState.value.sentryMode
    }

    const debugToggleHighBeam = () => {
        sceneState.value.lights.headlightHigh = !sceneState.value.lights.headlightHigh
    }

    // 监听车辆数据变化，同步到3D模型
    watch(() => vehicleData.value, () => {
        syncVehicleDataToScene()
    }, {
        deep: true
    })

    onMounted(async () => {
        if (!vehicleStore.hasVehicles) {
            await vehicleStore.fetchVehicles()
        }
    })

    onShow(async () => {
        if (!userStore.checkTokenExpiry()) {
            uni.reLaunch({
                url: '/pages/login/login'
            })
            return
        }
        if (!vehicleStore.hasVehicles) {
            await vehicleStore.fetchVehicles()
        }
        if (currentVehicle.value?.vin) {
            initVehicleData(currentVehicle.value.vin)
        } else if (vehicleStore.hasVehicles && vehicleStore.currentVehicle?.vin) {
            initVehicleData(vehicleStore.currentVehicle.vin)
        }
    })

    onHide(() => {
        destroyVehicleData()
    })

    onUnmounted(() => {
        destroyVehicleData()
    })

    watch(() => currentVehicle.value, (newVal) => {
        destroyVehicleData()
        if (newVal) {
            initVehicleData(newVal.vin)
        }
    })
</script>

<style lang="scss" scoped>
    .dashboard {
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

    .dashboard-scroll {
        flex: 1;
        height: 0;
    }

    .tabbar-spacer {
        height: 130rpx;
    }

    .hero-left {
        position: absolute;
        top: 0;
        left: 0;
        z-index: 2;
        width: 45%;
        max-width: 360rpx;
        display: flex;
        flex-direction: column;
        background: transparent;
        pointer-events: auto;
    }

    /* ========== 顶部栏 ========== */
    .header {
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0 32rpx;
        height: 60rpx;
    }

    .model-loading-spinner {
        width: 48rpx;
        height: 48rpx;
        border: 4rpx solid rgba(255, 255, 255, 0.15);
        border-top-color: #5BE7C4;
        border-radius: 50%;
        animation: modelSpin 0.8s linear infinite;
    }

    .model-loading-text {
        margin-top: 16rpx;
        font-size: 24rpx;
        color: var(--dark-page-text-hint);
    }

    @keyframes modelSpin {
        to {
            transform: rotate(360deg);
        }
    }

    .dashboard-hero-bg {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        width: 100%;
        height: 100%;
        object-fit: cover;
        z-index: 0;
    }



    .hero-left>.status-bar,
    .hero-left>.header,
    .hero-left>.vehicle-header,
    .hero-left>.range-section {
        position: relative;
        z-index: 2;
    }

    .status-bar {
        flex-shrink: 0;
        height: var(--status-bar-height);
    }
    
    .dashboard-header {
        position: relative;
        display: flex;
        min-height: 650rpx;
        overflow: hidden;
    }

    .dashboard-hero-3d {
        position: absolute;
        top: 100px;
        left: calc(20% - 20px);
        right: 0;
        bottom: 0;
        width: 66.67%;
        height: 66.67%;
        z-index: 0;
        pointer-events: auto;
    }

    .model-loading-overlay {
        position: absolute;
        top: 0;
        left: 30%;
        right: 0;
        bottom: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        z-index: 1;
        pointer-events: none;
    }

    .tesla-logo {
        .logo-text {
            font-size: 28rpx;
            font-weight: 700;
            color: var(--dark-page-text);
            letter-spacing: 4rpx;
        }
    }

    .header-right {
        width: 60rpx;
        height: 60rpx;
        display: flex;
        align-items: center;
        justify-content: center;
        position: relative;

        &:active {
            opacity: 0.6;
        }
    }

    .notification-badge {
        position: absolute;
        top: 10rpx;
        right: 10rpx;
        width: 16rpx;
        height: 16rpx;
        border-radius: 50%;
        background: #FF6B6B;
        border: 2rpx solid var(--dark-page-bg);
    }

    /* ========== 车辆信息 ========== */
    .vehicle-header {
        flex-shrink: 0;
        padding: 16rpx 32rpx 12rpx;
    }

    .vehicle-model-row {
        display: flex;
        align-items: center;
        gap: 8rpx;
    }

    .vehicle-model {
        font-size: 40rpx;
        font-weight: 700;
        color: var(--dark-page-text);
    }

    .vehicle-status-row {
        display: flex;
        align-items: center;
        gap: 10rpx;
        margin-top: 8rpx;
    }

    .status-dot {
        width: 14rpx;
        height: 14rpx;
        border-radius: 50%;
    }

    .vehicle-status-text {
        font-size: 24rpx;
        color: var(--dark-page-text-hint);
    }

    /* ========== 续航与电量 ========== */
    .range-section {
        flex-shrink: 0;
        padding: 8rpx 32rpx 16rpx;
    }

    .range-value {
        font-size: 56rpx;
        font-weight: 700;
        color: var(--dark-page-text);

        .range-unit {
            font-size: 28rpx;
            font-weight: 500;
            color: var(--dark-page-text-secondary);
            margin-left: 4rpx;
        }
    }

    .range-label {
        display: block;
        font-size: 24rpx;
        color: var(--dark-page-text-secondary);
        margin-top: 8rpx;
    }

    /* ========== 快捷操作 ========== */
    .quick-actions {
        flex-shrink: 0;
        display: flex;
        justify-content: flex-start;
        align-items: flex-start;
        gap: 0;
        background: var(--dark-page-glass-bg);
        border: 1rpx solid var(--dark-page-glass-border);
        border-radius: 28rpx;
        margin: 16rpx 32rpx 32rpx;
        padding: 28rpx 0;
    }

    .visionpro-theme .quick-actions {
        background: rgba(255, 255, 255, 0.58);
        backdrop-filter: blur(40px);
        -webkit-backdrop-filter: blur(40px);
        border: 1px solid rgba(255, 255, 255, 0.65);
        box-shadow: 0 4px 20px rgba(15, 23, 42, 0.06);
    }

    .quick-action-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 10rpx;
        flex: 1;
        min-width: 0;

        &:active {
            opacity: 0.7;
        }
    }

    .quick-action-icon {
        width: 80rpx;
        height: 80rpx;
        border-radius: 24rpx;
        background: var(--dark-page-icon-wrap-bg);
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s ease;

        &.active {
            background: linear-gradient(135deg, #5BE7C4, #3cc9a5);
            box-shadow: 0 4rpx 16rpx rgba(91, 231, 196, 0.3);
        }
    }

    .visionpro-theme .quick-action-icon {
        background: rgba(255, 255, 255, 0.5);
        backdrop-filter: blur(10px);
        -webkit-backdrop-filter: blur(10px);
        border: 1px solid rgba(255, 255, 255, 0.4);

        &.active {
            background: linear-gradient(135deg, #0F172A, #334155);
            box-shadow: 0 4rpx 16rpx rgba(15, 23, 42, 0.2);
        }
    }

    .quick-action-label {
        font-size: 22rpx;
        color: var(--dark-page-text-secondary);
    }

    .quick-action-sub {
        font-size: 18rpx;
        color: var(--dark-page-text-hint);
    }

/* ========== 电量续航条 ========== */
.battery-bar-section {
  margin-top: 20rpx;
}

.battery-bar-info {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 10rpx;
}

.battery-bar-percent {
  font-size: 28rpx;
  font-weight: 700;
}

.battery-bar-range {
  font-size: 22rpx;
  color: var(--dark-page-text-hint);
}

/* 下方流光条（全屏贯穿） */
.battery-flow-line {
  position: relative;
  width: 100%;
  height: 4rpx;
  margin-top:-4rpx;
  border-radius: 2rpx;
  overflow: hidden;
  background: transparent;
}

.battery-flow-line::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg,
    transparent 0%,
    #00ff88 50%,
    transparent 100%
  );
  animation: fullSweep 2s linear infinite;
}

/* 全屏横扫动画 */
@keyframes fullSweep {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(100%);
  }
}

/* 电池轨道 */
.battery-bar-track {
  height: 4rpx;
  border-radius: 2rpx;
  background: var(--dark-page-glass-border);
  overflow: hidden;
  position: relative;
}

/* 电量填充 */
.battery-bar-fill {
  height: 100%;
  border-radius: 2rpx;
  transition: width 0.8s ease;
  position: relative;
  z-index: 2;
}

.battery-bar-detail {
  margin-top: 8rpx;
}

.battery-bar-charging-text {
  font-size:18rpx;
  color: var(--dark-page-text-hint);
}
    /* ========== 六宫格菜单 ========== */
    .menu-grid {
        flex-shrink: 0;
        display: flex;
        flex-direction: column;
        gap: 20rpx;
        margin: 0 32rpx 16rpx;
    }

    .menu-grid-row {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 20rpx;
    }

    .menu-grid-item {
        display: flex;
        align-items: center;
        gap: 16rpx;
        padding: 28rpx 20rpx;
        background: var(--dark-page-icon-wrap-bg);
        border-radius: 20rpx;
        transition: background 0.2s ease;
        min-height: 94rpx;
        justify-content: flex-start;

        &:active {
            background: var(--dark-page-press-bg);
        }
    }

    .visionpro-theme .menu-grid-item {
        background: rgba(255, 255, 255, 0.5);
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
        border: 1px solid rgba(255, 255, 255, 0.4);
        border-radius: 22px;
        box-shadow: 0 2px 12px rgba(15, 23, 42, 0.04);
    }

    .menu-grid-icon {
        width: 64rpx;
        height: 64rpx;
        border-radius: 50%;
        background: var(--dark-page-glass-bg);
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;

        &.climate-icon {
            background: linear-gradient(135deg, rgba(96, 165, 250, 0.2), rgba(251, 191, 36, 0.2));
        }
    }

    .visionpro-theme .menu-grid-icon {
        background: rgba(15, 23, 42, 0.05);

        &.climate-icon {
            background: linear-gradient(135deg, rgba(15, 23, 42, 0.08), rgba(255, 184, 107, 0.12));
        }
    }

    .climate-temps {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 2rpx;
    }

    .climate-temp-inner {
        font-size: 18rpx;
        font-weight: 600;
        color: var(--color-info);
        line-height: 1;
    }

    .climate-temp-outer {
        font-size: 16rpx;
        color: #fbbf24;
        line-height: 1;
    }

    .menu-grid-content {
        display: flex;
        flex-direction: column;
        gap: 4rpx;
        min-width: 0;
    }

    .menu-grid-title {
        font-size: 26rpx;
        font-weight: 600;
        color: var(--dark-page-text);
    }

    .menu-grid-subtitle {
        font-size: 20rpx;
        color: var(--dark-page-text-hint);
    }

    .menu-grid-item-full {
        grid-column: 1 / -1;
    }

    .location-grid-item {
        position: relative;
        overflow: hidden;
        padding: 0 !important;
    }

    .location-card-map {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        z-index: 0;
    }

    .location-card-overlay {
        position: relative;
        z-index: 1;
        display: flex;
        align-items: flex-end;
        justify-content: flex-start;
        padding: 12rpx 16rpx;
        width: 100%;
        height: 100%;
        box-sizing: border-box;
    }

    .location-card-badge {
        display: flex;
        align-items: center;
        gap: 6rpx;
        padding: 6rpx 14rpx;
        border-radius: 16rpx;
        background: rgba(0, 0, 0, 0.45);
        backdrop-filter: blur(8px);
    }

    .location-card-label {
        font-size: 20rpx;
        font-weight: 600;
        color: #ffffff;
    }

    .location-card-fallback {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--dark-page-glass-bg);
        z-index: 0;
    }

    /* ========== 空状态 ========== */
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

    /* ========== 调试按钮 ========== */
    .debug-btn {
        position: absolute;
        top: 100rpx;
        right: 24rpx;
        width: 56rpx;
        height: 56rpx;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.25);
        backdrop-filter: blur(10px);
        -webkit-backdrop-filter: blur(10px);
        border: 1rpx solid rgba(255, 255, 255, 0.35);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 3;

        &:active {
            opacity: 0.7;
        }
    }

    .debug-btn-text {
        font-size: 28rpx;
        font-weight: 700;
        color: rgba(255, 255, 255, 0.9);
    }

    /* ========== 调试弹窗 ========== */
    .debug-modal-mask {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        z-index: 999;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .debug-modal {
        width: 560rpx;
        background: var(--modal-bg);
        border: 1rpx solid rgba(255, 255, 255, 0.1);
        border-radius: 28rpx;
        padding: 32rpx;
    }

    .debug-modal-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 32rpx;
    }

    .debug-modal-title {
        font-size: 32rpx;
        font-weight: 700;
        color: var(--dark-page-text);
    }

    .debug-modal-close {
        width: 48rpx;
        height: 48rpx;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.08);
        display: flex;
        align-items: center;
        justify-content: center;

        &:active {
            opacity: 0.7;
        }
    }

    .debug-modal-close-text {
        font-size: 24rpx;
        color: var(--dark-page-text-hint);
    }

    .debug-modal-section {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 20rpx 0;
        border-bottom: 1rpx solid rgba(255, 255, 255, 0.06);

        &:last-child {
            border-bottom: none;
        }
    }

    .debug-modal-label {
        font-size: 28rpx;
        font-weight: 500;
        color: var(--dark-page-text);
    }

    .debug-modal-btn {
        padding: 12rpx 32rpx;
        border-radius: 16rpx;
        background: rgba(255, 255, 255, 0.08);
        border: 1rpx solid rgba(255, 255, 255, 0.1);
        font-size: 24rpx;
        color: var(--dark-page-text-secondary);

        &.active {
            background: linear-gradient(135deg, #5BE7C4, #3cc9a5);
            color: #ffffff;
            border-color: transparent;
        }

        &:active {
            opacity: 0.7;
        }
    }

    .debug-modal-gears {
        display: flex;
        gap: 12rpx;
    }

    .debug-modal-gear {
        width: 64rpx;
        height: 64rpx;
        border-radius: 16rpx;
        background: rgba(255, 255, 255, 0.08);
        border: 1rpx solid rgba(255, 255, 255, 0.1);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 28rpx;
        font-weight: 700;
        color: var(--dark-page-text-secondary);

        &.active {
            background: linear-gradient(135deg, #5BE7C4, #3cc9a5);
            color: #ffffff;
            border-color: transparent;
        }

        &:active {
            opacity: 0.7;
        }
    }


    /* ========== 横屏适配 ========== */
    @media screen and (orientation: landscape) {
        .dashboard {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            overflow-y: auto;
        }

        .dashboard-header {
            min-height: auto;
        }



        .status-bar {
            display: none;
        }

        .header {
            height: 50rpx;
            padding: 0 24rpx;
        }

        .tesla-logo .logo-text {
            font-size: 24rpx;
        }

        .vehicle-header {
            padding: 8rpx 24rpx;
        }

        .vehicle-model {
            font-size: 32rpx;
        }

        .range-section {
            padding: 4rpx 24rpx 8rpx;
        }

        .range-value {
            font-size: 40rpx;
        }

        .quick-actions {
            margin: 8rpx 24rpx;
            padding: 16rpx 12rpx;
        }

        .quick-action-icon {
            width: 60rpx;
            height: 60rpx;
            border-radius: 18rpx;
        }

        .charging-card {
            margin: 0 24rpx 8rpx;
            padding: 20rpx;
        }

        .menu-grid {
            margin: 0 24rpx 8rpx;
            padding: 16rpx;
            gap: 12rpx;
        }

        .menu-grid-row {
            gap: 12rpx;
        }

        .menu-grid-item {
            padding: 14rpx 12rpx;
            gap: 12rpx;
        }

        .menu-grid-icon {
            width: 48rpx;
            height: 48rpx;
        }

        .menu-grid-title {
            font-size: 22rpx;
        }

        .menu-grid-subtitle {
            font-size: 16rpx;
        }

        .climate-temp-inner {
            font-size: 14rpx;
        }

        .climate-temp-outer {
            font-size: 12rpx;
        }

        .location-card-overlay {
            padding: 8rpx 12rpx;
        }

        .location-card-badge {
            padding: 4rpx 10rpx;
        }

        .location-card-label {
            font-size: 18rpx;
        }
    }

    @media screen and (orientation: landscape) and (max-height: 500px) {
        .quick-actions {
            padding: 12rpx 8rpx;
        }

        .quick-action-icon {
            width: 48rpx;
            height: 48rpx;
            border-radius: 14rpx;
        }

        .quick-action-label {
            font-size: 18rpx;
        }
    }
</style>