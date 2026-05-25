<template>
	<view class="page" :class="themeClass" :style="{ paddingTop: statusBarHeight + 'px' }">
		<view class="header">
			<text class="title">我的车辆</text>
			<view class="add-btn" @click="goToBind">
				<Icon name="Add" :size="18" color="#fff" />
			</view>
		</view>

		<scroll-view scroll-y class="main-scroll">
			<view class="loading-state" v-if="loading">
				<view class="spin-icon">
					<Icon name="Sync" :size="40" themeColor="primary" />
				</view>
				<text class="loading-text">加载中...</text>
			</view>

			<view class="vehicle-list" v-else-if="vehicles.length > 0">
				<view v-for="vehicle in vehicles" :key="vehicle.id" class="vehicle-item">
					<!-- 车辆卡片（背景图卡片） -->
					<view class="vehicle-card" @click="selectVehicle(vehicle)">
						<!-- 背景图 -->
						<image 
							class="vehicle-card-bg" 
							src="/static/vehicle-card-bg.jpg" 
							mode="aspectFill"
						/>
						<view class="vehicle-card-overlay"></view>
						
						<!-- 车辆信息区域 -->
						<view class="vehicle-info-section">
							<view class="vehicle-info-left">
								<view class="vehicle-name-row">
									<text class="vehicle-name">{{ vehicle.vehicle_name || vehicle.display_name || 'Tesla' }}</text>
									<Icon name="Create" :size="16" color="rgba(255,255,255,0.6)" />
								</view>
								<text class="vehicle-vin">{{ maskVIN(vehicle.vin) }}</text>
							</view>
							<view class="rebind-btn" @click.stop="goToBind">
								<text class="rebind-text">重新绑定</text>
							</view>
						</view>
					</view>

					<!-- 功能卡片列表 -->
					<view class="feature-list">
						<!-- 虚拟钥匙 -->
						<view class="feature-card" @click.stop="onKeyCardClick(vehicle)">
							<view class="feature-icon-wrap key-icon">
								<Icon :name="vehicleKeys[vehicle.vin]?.paired ? 'LockClosed' : 'LockOpen'" :size="22" color="#fff" />
							</view>
							<view class="feature-content">
								<text class="feature-title">虚拟钥匙</text>
								<text class="feature-subtitle" v-if="keyChecking[vehicle.vin]">检查中...</text>
								<text class="feature-subtitle" v-else-if="vehicleKeys[vehicle.vin]?.paired">已配对</text>
								<text class="feature-subtitle" v-else>未配对，点击配置</text>
							</view>
							<view class="feature-right">
								<view class="connection-status" v-if="!vehicleKeys[vehicle.vin]?.paired">
									<view class="connection-dot"></view>
									<text class="connection-text">未配对</text>
								</view>
								<Icon name="ChevronForward" :size="18" themeColor="hint" />
							</view>
						</view>

						<!-- 行车AI分析 -->
						<view class="feature-card" @click.stop="goToTripAI(vehicle)">
							<view class="feature-icon-wrap trip-icon">
								<Icon name="Navigate" :size="22" color="#fff" />
							</view>
							<view class="feature-content">
								<text class="feature-title">行车 AI 分析</text>
								<text class="feature-subtitle">{{ formatSummary(vehicleTripSummary[vehicle.vin], '智能分析驾驶表现') }}</text>
							</view>
							<view class="feature-right">
								<text class="feature-action-text">查看报告</text>
							<Icon name="ChevronForward" :size="18" themeColor="hint" />
						</view>
					</view>

					<!-- 充电AI分析 -->
						<view class="feature-card" @click.stop="goToChargingAI(vehicle)">
							<view class="feature-icon-wrap charging-icon">
								<Icon name="Flash" :size="22" color="#fff" />
							</view>
							<view class="feature-content">
								<text class="feature-title">充电 AI 分析</text>
								<text class="feature-subtitle">{{ formatSummary(vehicleChargingSummary[vehicle.vin], '优化充电建议与习惯') }}</text>
							</view>
							<view class="feature-right">
								<text class="feature-action-text">查看报告</text>
							<Icon name="ChevronForward" :size="18" themeColor="hint" />
						</view>
					</view>

					<!-- 车辆AI报告 -->
						<view class="feature-card" @click.stop="goToVehicleAI(vehicle)">
							<view class="feature-icon-wrap vehicle-icon">
								<Icon name="Sparkles" :size="22" color="#fff" />
							</view>
							<view class="feature-content">
								<text class="feature-title">车辆 AI 报告</text>
								<text class="feature-subtitle">{{ formatSummary(vehicleReportSummary[vehicle.vin], '全面诊断车辆健康状态') }}</text>
							</view>
							<view class="feature-right">
								<text class="feature-action-text">查看报告</text>
							<Icon name="ChevronForward" :size="18" themeColor="hint" />
						</view>
					</view>
				</view>
				</view>
			</view>

			<view class="empty-state" v-else>
				<view class="empty-icon-wrap">
					<Icon name="Car" :size="72" color="#d5d8dc" />
				</view>
				<text class="empty-text">暂无车辆</text>
				<text class="empty-hint">点击右上角添加您的第一辆Tesla</text>
				<view class="bind-btn" @click="goToBind">
					<Icon name="Add" :size="18" color="#fff" />
					<text class="bind-text">绑定车辆</text>
				</view>
			</view>
			<view class="tabbar-spacer"></view>
		</scroll-view>

		<view class="pairing-modal-mask" v-if="showPairingModal" @click="showPairingModal = false">
			<view class="pairing-modal" @click.stop>
				<view class="pairing-modal-header">
					<Icon name="Key" :size="24" themeColor="primary" />
					<text class="pairing-modal-title">虚拟钥匙配对</text>
				</view>
				<view class="pairing-modal-body">
					<text class="pairing-modal-desc">远程控制需要先完成虚拟钥匙配对，请按以下步骤操作：</text>
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
							<text class="step-text">返回后点击"检查配对状态"</text>
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
						<Icon name="Open" :size="16" color="#fff" />
						<text>打开配对链接</text>
					</button>
				</view>
				<view class="pairing-check-row" v-if="!pairingPaired">
					<text class="pairing-check-link" @click="checkPairingStatus">已配对？点击检查状态</text>
				</view>
			</view>
		</view>

		<TabBar :currentIndex="1" />
	</view>
</template>

<script setup>
	import {
		ref,
		computed,
		onMounted,
		reactive
	} from 'vue'
	import {
		onShow
	} from '@dcloudio/uni-app'
	import {
		useVehicleStore
	} from '@/store/vehicle'
	import {
		useThemeStore
	} from '@/store/theme'
	import TabBar from '@/components/TabBar/TabBar.vue'
	import {
		getLatestAnalysis
	} from '@/api/ai.js'
	import {
		getPairingURL,
		getFleetStatus
	} from '@/api/vehicle.js'
	import Icon from '@/components/Icon/Icon.vue'

	const statusBarHeight = uni.getSystemInfoSync().statusBarHeight

	const themeStore = useThemeStore()
	const themeClass = computed(() => themeStore.themeClass)
	const primaryColor = computed(() => themeStore.colors.primary)
	const successColor = computed(() => themeStore.colors.success)
	const tertiaryColor = computed(() => themeStore.colors.secondaryTextColor)

	const vehicleStore = useVehicleStore()
	const vehicles = computed(() => vehicleStore.vehicles)
	const currentVehicle = computed(() => vehicleStore.currentVehicle)
	const loading = computed(() => vehicleStore.loading)

	const vehicleTripSummary = ref({})
	const vehicleChargingSummary = ref({})
	const vehicleReportSummary = ref({})

	const vehicleKeys = reactive({})
	const keyChecking = reactive({})

	const showPairingModal = ref(false)
	const pairingURL = ref('')
	const pairingChecking = ref(false)
	const pairingPaired = ref(false)
	const pairingVIN = ref('')

	const loadLatestSummaries = async (vin) => {
		try {
			const [tripRes, chargingRes, vehicleRes] = await Promise.all([
				getLatestAnalysis(vin, 'trip'),
				getLatestAnalysis(vin, 'charging'),
				getLatestAnalysis(vin, 'vehicle')
			])
			if (tripRes?.data?.summary) vehicleTripSummary.value[vin] = tripRes.data.summary
			if (chargingRes?.data?.summary) vehicleChargingSummary.value[vin] = chargingRes.data.summary
			if (vehicleRes?.data?.summary) vehicleReportSummary.value[vin] = vehicleRes.data.summary
		} catch (e) {}
	}

	const checkVehicleKeyStatus = async (vin) => {
		keyChecking[vin] = true
		try {
			const res = await getFleetStatus(vin)
			vehicleKeys[vin] = {
				paired: res.data?.key_paired || false
			}
		} catch (e) {
			vehicleKeys[vin] = { paired: false }
		} finally {
			keyChecking[vin] = false
		}
	}

	onMounted(() => {
		vehicleStore.fetchVehicles()
	})

	onShow(async () => {
		const res = await vehicleStore.fetchVehicles()
		const list = vehicleStore.vehicles
		list.forEach(v => {
			loadLatestSummaries(v.vin)
			if (vehicleKeys[v.vin] === undefined) {
				checkVehicleKeyStatus(v.vin)
			}
		})
	})

	const selectVehicle = (vehicle) => {
		vehicleStore.setCurrentVehicle(vehicle)
		uni.reLaunch({
			url: '/pages/dashboard/dashboard'
		})
	}

	const goToBind = () => {
		uni.navigateTo({
			url: '/pages/bind/bind'
		})
	}

	const isCurrent = (id) => {
		return currentVehicle.value?.id === id
	}

	const maskVIN = (vin) => {
		if (!vin || vin.length <= 6) return vin || ''
		return vin.slice(0, -6) + '******'
	}

	const formatSummary = (summary, defaultText) => {
		if (!summary) return defaultText
		// 取前两行（按换行符分割）
		const lines = summary.split('\n').filter(line => line.trim())
		if (lines.length === 0) return defaultText
		if (lines.length === 1) return lines[0]
		return lines[0] + ' · ' + lines[1]
	}

	const onKeyCardClick = async (vehicle) => {
		if (vehicleKeys[vehicle.vin]?.paired) {
			uni.showToast({ title: '虚拟钥匙已配对', icon: 'success' })
			return
		}
		await showPairingGuide(vehicle.vin)
	}

	const showPairingGuide = async (vin) => {
		pairingVIN.value = vin
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
		// #ifdef APP-PLUS
		plus.runtime.openURL(pairingURL.value, (err) => {
			uni.setClipboardData({
				data: pairingURL.value,
				success: () => {
					uni.showToast({ title: '配对链接已复制，请在浏览器中打开', icon: 'none' })
				}
			})
		})
		// #endif
		// #ifdef H5
		window.open(pairingURL.value, '_blank')
		// #endif
	}

	const checkPairingStatus = async () => {
		if (!pairingVIN.value) return
		pairingChecking.value = true
		pairingPaired.value = false

		try {
			const res = await getFleetStatus(pairingVIN.value)
			const keyPaired = res.data?.key_paired
			if (keyPaired) {
				pairingPaired.value = true
				vehicleKeys[pairingVIN.value] = { paired: true }
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

	const goToTripAI = (vehicle) => {
		uni.navigateTo({
			url: `/pages/ai/trip?vin=${vehicle.vin}`
		})
	}
	const goToChargingAI = (vehicle) => {
		uni.navigateTo({
			url: `/pages/ai/charging?vin=${vehicle.vin}`
		})
	}
	const goToVehicleAI = (vehicle) => {
		uni.navigateTo({
			url: `/pages/ai/vehicle?vin=${vehicle.vin}`
		})
	}
</script>

<style lang="scss" scoped>
	.page {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		overflow: hidden;
		box-sizing: border-box;
		background: var(--bg-page);
		display: flex;
		flex-direction: column;
	}

	.main-scroll {
		flex: 1;
		height: 0;
	}

	.tabbar-spacer {
		height: 130rpx;
	}

	.header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 24rpx 32rpx 20rpx;

		.title {
			font-size: 40rpx;
			font-weight: 700;
			color: var(--text-primary);
			letter-spacing: 1rpx;
		}

		.add-btn {
			width: 64rpx;
			height: 64rpx;
			border-radius: 50%;
			background: var(--color-primary);
			display: flex;
			align-items: center;
			justify-content: center;
			box-shadow: 0 4rpx 16rpx rgba(37, 99, 235, 0.3);
		}
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 200rpx 0;

		.spin-icon {
			animation: spin 1s linear infinite;
		}

		.loading-text {
			font-size: 28rpx;
			color: var(--text-hint);
			margin-top: 24rpx;
		}
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}

		to {
			transform: rotate(360deg);
		}
	}

	.vehicle-list {
		padding: 0 24rpx;
	}

	.vehicle-item {
		margin-bottom: 20rpx;
	}

	.vehicle-card {
		position: relative;
		border-radius: 28rpx;
		margin-bottom: 16rpx;
		box-shadow: var(--shadow-card);
		overflow: hidden;
		height: 280rpx;
	}

	.vehicle-card-bg {
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

	.vehicle-card-overlay {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: linear-gradient(
			135deg,
			rgba(6, 11, 20, 0.95) 0%,
			rgba(6, 11, 20, 0.7) 50%,
			rgba(6, 11, 20, 0.4) 100%
		);
		z-index: 1;
	}

	/* ========== 车辆信息区域 ========== */
	.vehicle-info-section {
		position: relative;
		z-index: 2;
		display: flex;
		align-items: center;
		padding: 28rpx;
		height: 100%;
		box-sizing: border-box;
	}

	.vehicle-info-left {
		flex: 1;
		min-width: 0;
	}

	.vehicle-name-row {
		display: flex;
		align-items: center;
		gap: 12rpx;
		margin-bottom: 8rpx;
	}

	.vehicle-name {
		font-size: 32rpx;
		font-weight: 600;
		color: #ffffff;
	}

	.vehicle-vin {
		font-size: 22rpx;
		color: rgba(255, 255, 255, 0.6);
		display: block;
		margin-bottom: 12rpx;
	}

	.rebind-btn {
		margin-left: 16rpx;
		flex-shrink: 0;
		background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
		border-radius: 20rpx;
		padding: 8rpx 16rpx;

		.rebind-text {
			font-size: 20rpx;
			color: #ffffff;
			font-weight: 500;
		}
	}

	/* ========== 功能卡片列表 ========== */
	.feature-list {
		padding-top: 20rpx;
		display: flex;
		flex-direction: column;
		gap: 16rpx;
	}

	.feature-card {
		display: flex;
		align-items: center;
		padding: 24rpx;
		background: var(--bg-card-secondary);
		border-radius: 20rpx;

		&:active {
			opacity: 0.85;
		}
	}

	.feature-icon-wrap {
		width: 52rpx;
		height: 52rpx;
		border-radius: 14rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		margin-right: 20rpx;

		&.key-icon {
			background: linear-gradient(135deg, #7C879B, #64748B);
		}

		&.trip-icon {
			background: linear-gradient(135deg, #7B6CFF, #5f4ee0);
		}

		&.charging-icon {
			background: linear-gradient(135deg, #8CFB89, #6ce06a);
		}

		&.vehicle-icon {
			background: linear-gradient(135deg, #FF7FA5, #e0608a);
		}
	}

	.feature-content {
		flex: 1;
		min-width: 0;
	}

	.feature-title {
		font-size: 28rpx;
		font-weight: 600;
		color: var(--text-primary);
		display: block;
		margin-bottom: 4rpx;
	}

	.feature-subtitle {
		font-size: 22rpx;
		color: var(--text-tertiary);
		display: block;
	}

	.feature-right {
		display: flex;
		align-items: center;
		gap: 12rpx;
		flex-shrink: 0;
	}

	.connection-status {
		display: flex;
		align-items: center;
		gap: 8rpx;

		.connection-dot {
			width: 8rpx;
			height: 8rpx;
			border-radius: 50%;
			background: #FF6B6B;
		}

		.connection-text {
			font-size: 22rpx;
			color: #FF6B6B;
		}
	}

	.feature-action-text {
		font-size: 24rpx;
		color: var(--text-tertiary);
	}

	/* ========== 空状态 ========== */
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 260rpx 60rpx 80rpx;

		.empty-icon-wrap {
			width: 160rpx;
			height: 160rpx;
			border-radius: 50%;
			background: var(--bg-empty-icon);
			display: flex;
			align-items: center;
			justify-content: center;
			margin-bottom: 32rpx;
		}

		.empty-text {
			font-size: 34rpx;
			font-weight: 600;
			color: var(--text-empty);
			margin-bottom: 12rpx;
		}

		.empty-hint {
			font-size: 26rpx;
			color: var(--text-empty-hint);
			margin-bottom: 80rpx;
		}

		.bind-btn {
			display: flex;
			align-items: center;
			justify-content: center;
			gap: 10rpx;
			background: var(--color-primary);
			border-radius: 44rpx;
			height: 88rpx;
			padding: 0 56rpx;
			box-shadow: 0 6rpx 24rpx rgba(37, 99, 235, 0.25);

			.bind-text {
				font-size: 30rpx;
				color: #ffffff;
				font-weight: 600;
			}
		}
	}

	/* ========== 配对弹窗 ========== */
	.pairing-modal-mask {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 9999;
	}

	.pairing-modal {
		width: 620rpx;
		background: var(--bg-card);
		border-radius: 32rpx;
		overflow: hidden;

		.pairing-modal-header {
			display: flex;
			align-items: center;
			justify-content: center;
			gap: 16rpx;
			padding: 40rpx 32rpx 20rpx;

			.pairing-modal-title {
				font-size: 34rpx;
				font-weight: 700;
				color: var(--text-primary);
			}
		}

		.pairing-modal-body {
			padding: 20rpx 32rpx;

			.pairing-modal-desc {
				font-size: 26rpx;
				color: var(--text-secondary);
				display: block;
				margin-bottom: 28rpx;
				line-height: 1.6;
			}

			.pairing-steps {
				display: flex;
				flex-direction: column;
				gap: 20rpx;
				margin-bottom: 28rpx;

				.pairing-step {
					display: flex;
					align-items: center;
					gap: 16rpx;

					.step-num {
						width: 44rpx;
						height: 44rpx;
						border-radius: 50%;
						background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
					color: #fff;
					font-size: 24rpx;
					font-weight: 700;
						display: flex;
						align-items: center;
						justify-content: center;
						flex-shrink: 0;
					}

					.step-text {
						font-size: 26rpx;
						color: var(--text-primary);
					}
				}
			}

			.pairing-status {
				display: flex;
				align-items: center;
				justify-content: center;
				gap: 12rpx;
				padding: 20rpx;

				.pairing-spinner {
					width: 36rpx;
					height: 36rpx;
					border: 3rpx solid var(--border-divider);
					border-top-color: var(--color-primary);
					border-radius: 50%;
					animation: spin 0.8s linear infinite;
				}

				.pairing-status-text {
					font-size: 26rpx;
					color: var(--text-secondary);
				}

				&.paired {
					.pairing-status-text {
						color: #52c41a;
						font-weight: 600;
					}
				}
			}
		}

		.pairing-modal-footer {
			display: flex;
			gap: 20rpx;
			padding: 20rpx 32rpx;

			.pairing-btn-secondary {
				flex: 1;
				height: 80rpx;
				border-radius: 40rpx;
				background: var(--bg-card-secondary);
				color: var(--text-secondary);
				font-size: 28rpx;
				font-weight: 600;
				display: flex;
				align-items: center;
				justify-content: center;
				border: none;
			}

			.pairing-btn-primary {
				flex: 2;
				height: 80rpx;
				border-radius: 40rpx;
				background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
			color: #fff;
			font-size: 28rpx;
			font-weight: 600;
			display: flex;
			align-items: center;
			justify-content: center;
			gap: 8rpx;
			border: none;

			&[disabled] {
					opacity: 0.5;
				}
			}
		}

		.pairing-check-row {
			padding: 16rpx 32rpx 32rpx;
			display: flex;
			justify-content: center;

			.pairing-check-link {
				font-size: 26rpx;
				color: var(--color-primary);
				font-weight: 500;
			}
		}
	}
</style>