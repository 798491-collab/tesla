<template>
	<view class="custom-tabbar">
		<view class="tabbar-content">
			<view v-for="(item, index) in tabList" :key="index" class="tab-item"
				:class="{ active: currentIndex === index }" @click="switchTab(index, item.pagePath)">
				<Icon :name="item.icon" :size="24" :color="currentIndex === index ? activeColor : inactiveColor" />
				<text class="tab-text" :class="{ 'tab-text-active': currentIndex === index }">
					{{ item.text }}
				</text>
			</view>
		</view>
	</view>
</template>

<script setup>
	import {
		computed
	} from 'vue'
	import {
		useThemeStore
	} from '@/store/theme'

	const props = defineProps({
		currentIndex: {
			type: Number,
			default: 0
		}
	})

	const themeStore = useThemeStore()

	const activeColor = computed(() => {
		const t = themeStore.resolvedTheme
		if (t === 'visionpro') return '#0F172A'
		return t === 'dark' ? '#e0e0e0' : '#1a1a1a'
	})
	const inactiveColor = computed(() => {
		const t = themeStore.resolvedTheme
		if (t === 'visionpro') return '#94A3B8'
		return '#7C879B'
	})

	const tabList = [{
			text: '首页',
			icon: 'Home',
			pagePath: '/pages/dashboard/dashboard'
		},
		{
			text: '车辆',
			icon: 'CarSport',
			pagePath: '/pages/vehicle/vehicle'
		},
		{
			text: '控制',
			icon: 'Settings',
			pagePath: '/pages/control/control'
		},
		{
			text: '我的',
			icon: 'Person',
			pagePath: '/pages/profile/profile'
		}
	]

	const switchTab = (index, pagePath) => {
		if (index === props.currentIndex) return
		uni.reLaunch({
			url: pagePath
		})
	}
</script>

<style lang="scss" scoped>
	.custom-tabbar {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 9999;
		background: var(--tabbar-bg);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border-top: 1rpx solid var(--tabbar-border);
		box-shadow: var(--tabbar-shadow);
		padding-bottom: constant(safe-area-inset-bottom);
		padding-bottom: env(safe-area-inset-bottom);
	}

	.tabbar-content {
		display: flex;
		align-items: center;
		justify-content: space-around;
		height: 110rpx;
	}

	.tab-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		flex: 1;
		gap: 4rpx;
		padding: 8rpx 0;
		transition: all 0.2s ease;

		&:active {
			opacity: 0.7;
		}
	}

	.tab-text {
		font-size: 20rpx;
		color: var(--tabbar-text);
		transition: color 0.2s ease;

		&.tab-text-active {
			color: var(--tabbar-text-active);
			font-weight: 600;
		}
	}
</style>
