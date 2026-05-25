<template>
	<view class="bind-container" :class="themeClass">
		<NavBar title="绑定车辆" class="bind-navbar" />
		<view class="bind-header" :style="{ paddingTop: `${navBarHeight + 40}px` }">
			<view class="header-icon">
				<Icon name="CarSport" :size="28" color="#fff" />
			</view>
			<text class="title">绑定Tesla车辆</text>
			<text class="subtitle">通过Tesla官方OAuth授权绑定您的车辆</text>
		</view>

		<view class="steps-card">
			<view class="step" :class="{ active: currentStep >= 1 }">
				<view class="step-circle">1</view>
				<text class="step-label">获取授权</text>
			</view>
			<view class="step-line" :class="{ active: currentStep >= 2 }"></view>
			<view class="step" :class="{ active: currentStep >= 2 }">
				<view class="step-circle">2</view>
				<text class="step-label">登录授权</text>
			</view>
			<view class="step-line" :class="{ active: currentStep >= 3 }"></view>
			<view class="step" :class="{ active: currentStep >= 3 }">
				<view class="step-circle">3</view>
				<text class="step-label">选择车辆</text>
			</view>
		</view>

		<view class="bind-content">
			<scroll-view scroll-y class="main-scroll">
				<view v-if="currentStep === 1" class="step-content">
					<view class="info-card">
						<view class="info-card-header">
							<view class="info-icon-wrap">
								<Icon name="Key" :size="20" themeColor="info" />
							</view>
							<text class="info-title">绑定说明</text>
						</view>
						<view class="info-steps">
							<view class="info-step-item">
								<view class="step-dot">1</view>
								<text class="step-desc">点击按钮将跳转到Tesla官方授权页面</text>
							</view>
							<view class="info-step-item">
								<view class="step-dot">2</view>
								<text class="step-desc">在Tesla页面输入账号密码完成登录</text>
							</view>
							<view class="info-step-item">
								<view class="step-dot">3</view>
								<text class="step-desc">授权完成后将自动返回应用</text>
							</view>
							<view class="info-step-item">
								<view class="step-dot">4</view>
								<text class="step-desc">请确保所有权限已勾选，否则无法正常运行</text>
							</view>
						</view>
					</view>

					<button class="btn-primary" @click="startTeslaAuth">
						<Icon name="Key" :size="18" color="#fff" />
						<text>开始授权</text>
					</button>
				</view>

				<view v-if="currentStep === 2 && isMP" class="step-content">
					<view class="info-card">
						<view class="info-card-header">
							<view class="info-icon-wrap">
								<Icon name="InformationCircle" :size="20" themeColor="info" />
							</view>
							<text class="info-title">请在浏览器中完成授权</text>
						</view>
						<text class="info-desc">授权链接已复制到剪贴板，请在浏览器中打开完成登录授权</text>
					</view>

					<view class="input-card">
						<view class="input-wrap">
							<view class="input-icon">
								<Icon name="LockClosed" :size="18" themeColor="info" />
							</view>
							<input class="input" v-model="authCode" placeholder="请输入授权码(code)" />
						</view>
					</view>

					<button class="btn-primary" @click="submitAuthCode">
						<Icon name="ChevronForward" :size="18" color="#fff" />
						<text>提交授权码</text>
					</button>
					<button class="btn-secondary" @click="currentStep = 1">
						<Icon name="ChevronBack" :size="16" themeColor="hint" />
						<text>上一步</text>
					</button>
				</view>

				<view v-if="currentStep === 3" class="step-content">
					<view class="section-header">
						<Icon name="Car" :size="20" themeColor="primary" />
						<text class="section-title">选择要绑定的车辆</text>
					</view>

					<view class="vehicle-list">
						<view class="vehicle-item" v-for="vehicle in vehicles" :key="vehicle.vin"
							:class="{ selected: selectedVIN === vehicle.vin }" @click="selectVehicle(vehicle)">
							<view class="vehicle-icon-wrap">
								<Icon name="CarSport" :size="24" themeColor="hint" />
							</view>
							<view class="vehicle-info">
								<text class="vehicle-name">{{ vehicle.display_name }}</text>
								<text class="vehicle-vin">{{ vehicle.vin }}</text>
							</view>
							<view class="vehicle-state" :class="vehicle.state">
								<text>{{ vehicle.state }}</text>
							</view>
							<view v-if="selectedVIN === vehicle.vin" class="vehicle-check">
								<Icon name="Shield" :size="18" themeColor="primary" />
							</view>
						</view>
					</view>

					<button class="btn-primary" :disabled="!selectedVIN" @click="bindSelectedVehicle">
						<Icon name="LockClosed" :size="18" color="#fff" />
						<text>绑定选中车辆</text>
					</button>
					<button class="btn-secondary" @click="currentStep = 1">
						<Icon name="Sync" :size="16" themeColor="hint" />
						<text>重新授权</text>
					</button>
				</view>
			</scroll-view>
		</view>
	</view>
</template>

<script setup>
	import {
		ref,
		computed,
		onMounted,
		onUnmounted
	} from 'vue'
	import {
		onShow
	} from '@dcloudio/uni-app'
	import {
		useThemeStore
	} from '@/store/theme'
	import NavBar from '@/components/NavBar/NavBar.vue'

	const themeStore = useThemeStore()
	const themeClass = computed(() => themeStore.themeClass)
	const primaryColor = computed(() => themeStore.colors.primary)
	const infoColor = computed(() => themeStore.colors.infoColor)
	const secondaryTextColor = computed(() => themeStore.colors.secondaryTextColor)
	const currentStep = ref(1)
	const authCode = ref('')
	const vehicles = ref([])
	const selectedVIN = ref('')
	const tokenData = ref(null)
	const isMP = ref(false)
	const clientId = ref('')

	const statusBarHeight = uni.getSystemInfoSync().statusBarHeight || 0
	const navBarHeight = statusBarHeight + 44 // 44px = 88rpx NavBar inner height

	const checkPlatform = () => {
		// #ifdef MP-WEIXIN || MP-ALIPAY || MP-BAIDU || MP-TOUTIAO
		isMP.value = true
		// #endif
	}

	onMounted(() => {
		checkPlatform()
		checkAuthData()
		// #ifdef APP-PLUS
		handleUrlScheme()
		plus.globalEvent.addEventListener('newintent', handleUrlScheme)
		// #endif
	})

	onUnmounted(() => {
		// #ifdef APP-PLUS
		plus.globalEvent.removeEventListener('newintent', handleUrlScheme)
		// #endif
	})

	onShow(() => {
		if (currentStep.value === 1 || currentStep.value === 2) {
			checkAuthData()
		}
	})

	const handleUrlScheme = () => {
		// #ifdef APP-PLUS
		const args = plus.runtime.arguments
		if (args && args.startsWith('teslaapp://callback')) {
			const url = new URL(args)
			const authId = url.searchParams.get('auth_id')
			const authDataParam = url.searchParams.get('auth_data')
			const errorParam = url.searchParams.get('error')
			const errorDesc = url.searchParams.get('error_description')

			if (errorParam) {
				uni.showToast({
					title: decodeURIComponent(errorDesc || errorParam),
					icon: 'none'
				})
				return
			}

			if (authId) {
				fetchAuthDataById(authId)
			} else if (authDataParam) {
				processAuthDataFromUrl(authDataParam)
			}
		}
		// #endif
	}

	const fetchAuthDataById = async (authId) => {
		uni.showLoading({ title: '处理中...' })
		try {
			const response = await uni.request({
				url: (import.meta.env.VITE_API_BASE_URL || 'https://your-domain.com') + '/api/tesla/auth_data?auth_id=' + authId,
				method: 'GET'
			})
			if (response.statusCode === 200 && response.data.code === 200) {
				setAuthData(response.data.data)
			} else {
				throw new Error(response.data.message || '获取授权数据失败')
			}
		} catch (e) {
			uni.showToast({ title: e.message || '获取授权数据失败', icon: 'none' })
		} finally {
			uni.hideLoading()
		}
	}

	const processAuthDataFromUrl = (authDataParam) => {
		try {
			let base64 = authDataParam.replace(/-/g, '+').replace(/_/g, '/')
			while (base64.length % 4) { base64 += '=' }
			const decodedData = JSON.parse(atob(base64))
			setAuthData(decodedData)
		} catch (e) {
			uni.showToast({ title: '解析授权数据失败', icon: 'none' })
		}
	}

	const checkAuthData = () => {
		const authDataStr = uni.getStorageSync('tesla_auth_data')
		if (authDataStr) {
			try {
				const authData = JSON.parse(authDataStr)
				uni.removeStorageSync('tesla_auth_data')
				setAuthData(authData)
			} catch (e) {
				console.error('解析授权数据失败', e)
			}
		}
	}

	import request from '@/utils/request.js'

	const startTeslaAuth = () => {
		let authUrl = '/api/tesla/auth'
		// #ifdef APP-PLUS
		authUrl += '?platform=app'
		// #endif

		request({
			url: authUrl,
			method: 'GET'
		}).then((res) => {
			const authUrl = res.data.auth_url
			const match = authUrl.match(/client_id=([^&]+)/)
			if (match) {
				clientId.value = match[1]
			}

			// #ifdef H5
			window.location.href = authUrl
			// #endif

			// #ifdef APP-PLUS
			plus.runtime.openURL(authUrl)
			// #endif

			// #ifdef MP-WEIXIN || MP-ALIPAY || MP-BAIDU || MP-TOUTIAO
			uni.setClipboardData({
				data: authUrl,
				success: () => {
					uni.showToast({
						title: '授权链接已复制',
						icon: 'success'
					})
					currentStep.value = 2
				}
			})
			// #endif
		}).catch((err) => {
			console.error('获取授权链接失败', err)
		})
	}

	const submitAuthCode = () => {
		if (!authCode.value) {
			uni.showToast({
				title: '请输入授权码',
				icon: 'none'
			})
			return
		}

		uni.showLoading({
			title: '处理中...'
		})

		request({
			url: '/api/tesla/callback?code=' + encodeURIComponent(authCode.value),
			method: 'GET'
		}).then((res) => {
			tokenData.value = {
				access_token: res.data.access_token,
				refresh_token: res.data.refresh_token,
				expires_in: res.data.expires_in,
				scope: res.data.scope || ''
			}
			vehicles.value = res.data.vehicles || []
			currentStep.value = 3
		}).catch((err) => {
			uni.showToast({
				title: err.message || '授权处理失败',
				icon: 'none'
			})
		}).finally(() => {
			uni.hideLoading()
		})
	}

	const setAuthData = (data) => {
		tokenData.value = {
			access_token: data.access_token,
			refresh_token: data.refresh_token,
			expires_in: data.expires_in,
			scope: data.scope || ''
		}
		vehicles.value = data.vehicles || []
		currentStep.value = 3
	}

	const selectVehicle = (vehicle) => {
		selectedVIN.value = vehicle.vin
	}

	const bindSelectedVehicle = () => {
		if (!selectedVIN.value || !tokenData.value) return

		uni.showLoading({
			title: '绑定中...'
		})

		request({
			url: '/api/tesla/bind',
			method: 'POST',
			data: {
				access_token: tokenData.value.access_token,
				refresh_token: tokenData.value.refresh_token,
				expires_in: tokenData.value.expires_in,
				vin: selectedVIN.value
			}
		}).then((res) => {
			uni.showToast({
				title: '绑定成功',
				icon: 'success'
			})
			setTimeout(() => {
				uni.reLaunch({
					url: '/pages/dashboard/dashboard'
				})
			}, 1500)
		}).catch((err) => {
			console.error('绑定失败', err)
			uni.showToast({
				title: '绑定失败',
				icon: 'none'
			})
		}).finally(() => {
			uni.hideLoading()
		})
	}

	defineExpose({
		setAuthData
	})
</script>

<style lang="scss" scoped>
	.bind-container {
		height: 100vh;
		overflow: hidden;
		box-sizing: border-box;
		background: var(--bg-page);
		display: flex;
		flex-direction: column;
	}

	

	.bind-header {
		background: var(--gradient);
		padding: 48rpx 40rpx 64rpx;
		border-radius: 0 0 40rpx 40rpx;

		.header-icon {
			width: 80rpx;
			height: 80rpx;
			background: rgba(255, 255, 255, 0.2);
			border-radius: 24rpx;
			display: flex;
			align-items: center;
			justify-content: center;
			margin-bottom: 24rpx;
			border: 2rpx solid rgba(255, 255, 255, 0.15);
		}

		.title {
			font-size: 44rpx;
			font-weight: 700;
			color: #ffffff;
			display: block;
		}

		.subtitle {
			font-size: 26rpx;
			color: rgba(255, 255, 255, 0.75);
			margin-top: 12rpx;
			display: block;
		}
	}

	.steps-card {
		background: var(--bg-card);
		margin: -32rpx 32rpx 0;
		border-radius: 28rpx;
		box-shadow: var(--shadow-card);
		padding: 40rpx 48rpx;
		display: flex;
		align-items: center;
		justify-content: space-between;

		.step {
			display: flex;
			flex-direction: column;
			align-items: center;

			&.active {
				.step-circle {
					background: var(--gradient);
					color: #ffffff;
					box-shadow: 0 4rpx 16rpx rgba(37, 99, 235, 0.3);
				}

				.step-label {
					color: var(--color-primary);
					font-weight: 600;
				}
			}
		}

		.step-circle {
			width: 56rpx;
			height: 56rpx;
			background: var(--bg-card-secondary);
			border-radius: 50%;
			display: flex;
			align-items: center;
			justify-content: center;
			font-size: 24rpx;
			color: var(--text-tertiary);
			font-weight: 600;
		}

		.step-label {
			font-size: 22rpx;
			color: var(--text-tertiary);
			margin-top: 12rpx;
		}

		.step-line {
			flex: 1;
			height: 4rpx;
			background: var(--bg-card-secondary);
			margin: 0 16rpx;
			margin-bottom: 36rpx;
			border-radius: 2rpx;

			&.active {
				background: var(--gradient);
			}
		}
	}

	.bind-content {
		padding: 32rpx;
		flex: 1;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.main-scroll {
		flex: 1;
		overflow: hidden;
	}

	.step-content {
		.info-card {
			background: var(--bg-card);
			border-radius: 28rpx;
			box-shadow: var(--shadow-card);
			padding: 32rpx;
			margin-bottom: 24rpx;

			.info-card-header {
				display: flex;
				align-items: center;
				margin-bottom: 24rpx;
			}

			.info-icon-wrap {
				width: 56rpx;
				height: 56rpx;
				background: var(--bg-icon-wrap);
				border-radius: 16rpx;
				display: flex;
				align-items: center;
				justify-content: center;
				margin-right: 16rpx;
			}

			.info-title {
				font-size: 32rpx;
				font-weight: 600;
				color: var(--text-primary);
			}

			.info-desc {
				font-size: 28rpx;
				color: var(--text-secondary);
				line-height: 1.6;
			}

			.info-steps {
				padding-left: 8rpx;
			}

			.info-step-item {
				display: flex;
				align-items: flex-start;
				margin-bottom: 20rpx;

				&:last-child {
					margin-bottom: 0;
				}
			}

			.step-dot {
				width: 40rpx;
				height: 40rpx;
				background: var(--bg-icon-wrap);
				border-radius: 50%;
				display: flex;
				align-items: center;
				justify-content: center;
				font-size: 22rpx;
				color: var(--color-primary);
				font-weight: 600;
				margin-right: 16rpx;
				flex-shrink: 0;
			}

			.step-desc {
				font-size: 28rpx;
				color: var(--text-secondary);
				line-height: 40rpx;
			}
		}

		.input-card {
			background: var(--bg-card);
			border-radius: 28rpx;
			box-shadow: var(--shadow-card);
			padding: 16rpx 32rpx;
			margin-bottom: 24rpx;
		}

		.input-wrap {
			display: flex;
			align-items: center;
			background: var(--bg-input);
			border-radius: 20rpx;
			height: 96rpx;
			padding: 0 24rpx;
		}

		.input-icon {
			width: 56rpx;
			height: 56rpx;
			display: flex;
			align-items: center;
			justify-content: center;
			margin-right: 16rpx;
			flex-shrink: 0;
		}

		.input {
			flex: 1;
			font-size: 30rpx;
			color: var(--text-primary);
			height: 96rpx;
		}

		.section-header {
			display: flex;
			align-items: center;
			margin-bottom: 24rpx;
		}

		.section-title {
			font-size: 32rpx;
			font-weight: 600;
			color: var(--text-primary);
			margin-left: 12rpx;
		}
	}

	.vehicle-list {
		margin-bottom: 32rpx;
	}

	.vehicle-item {
		background: var(--bg-card);
		border-radius: 28rpx;
		box-shadow: var(--shadow-card);
		padding: 28rpx 32rpx;
		margin-bottom: 16rpx;
		border: 3rpx solid transparent;
		display: flex;
		align-items: center;

		&.selected {
			border-color: var(--color-primary);
			background: var(--bg-icon-wrap);

			.vehicle-icon-wrap {
				background: var(--bg-icon-wrap);
			}
		}

		.vehicle-icon-wrap {
			width: 80rpx;
			height: 80rpx;
			background: var(--bg-card-secondary);
			border-radius: 20rpx;
			display: flex;
			align-items: center;
			justify-content: center;
			margin-right: 24rpx;
			flex-shrink: 0;
		}

		.vehicle-info {
			flex: 1;
		}

		.vehicle-name {
			font-size: 30rpx;
			font-weight: 600;
			color: var(--text-primary);
			display: block;
		}

		.vehicle-vin {
			font-size: 24rpx;
			color: var(--text-tertiary);
			margin-top: 6rpx;
			display: block;
		}

		.vehicle-state {
			font-size: 22rpx;
			padding: 6rpx 16rpx;
			border-radius: 12rpx;
			background: var(--bg-card-secondary);
			color: var(--text-tertiary);
			margin-right: 16rpx;

			&.online {
				background: rgba(82, 196, 26, 0.12);
				color: #52c41a;
			}
		}

		.vehicle-check {
			width: 48rpx;
			height: 48rpx;
			display: flex;
			align-items: center;
			justify-content: center;
		}
	}

	.btn-primary {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 12rpx;
		background: var(--gradient);
		color: #ffffff;
		border-radius: 28rpx;
		height: 96rpx;
		font-size: 32rpx;
		font-weight: 600;
		border: none;
		box-shadow: 0 8rpx 24rpx rgba(37, 99, 235, 0.25);

		&[disabled] {
			opacity: 0.5;
			box-shadow: none;
		}
	}

	.btn-secondary {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8rpx;
		background: var(--bg-card);
		color: var(--text-secondary);
		border: 2rpx solid var(--border-card);
		border-radius: 28rpx;
		height: 96rpx;
		font-size: 30rpx;
		margin-top: 20rpx;
	}
</style>