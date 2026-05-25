import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const themeMode = ref(uni.getStorageSync('themeMode') || 'system')

  const isDark = ref(false)

  const resolvedTheme = computed(() => {
    if (themeMode.value === 'visionpro') return 'visionpro'
    if (themeMode.value === 'dark') return 'dark'
    if (themeMode.value === 'light') return 'light'
    return isDark.value ? 'dark' : 'light'
  })

  const isDarkMode = computed(() => resolvedTheme.value === 'dark')

  const themeClass = computed(() => resolvedTheme.value + '-theme')

  const colors = computed(() => {
    const t = resolvedTheme.value
    if (t === 'visionpro') {
      return {
        primary: '#0F172A',
        primaryLight: 'rgba(15, 23, 42, 0.06)',
        headerIcon: '#0F172A',
        chevron: '#64748B',
        inactiveIcon: 'rgba(15, 23, 42, 0.25)',
        inactiveIconLight: 'rgba(15, 23, 42, 0.12)',
        infoValue: '#0F172A',
        hint: '#64748B',
        quickActionIcon: '#334155',
        infoColor: '#0F172A',
        secondaryTextColor: '#64748B',
        iconColor: '#0F172A',
        success: '#5EEAD4',
        warning: '#FFB86B',
        danger: '#FF6B6B',
        info: '#0F172A',
        ai: '#7C6CFF',
        batteryHigh: '#5EEAD4',
        batteryMid: '#FFB86B',
        batteryLow: '#FF6B6B',
        charging: '#FFB86B',
        chargingComplete: '#5EEAD4',
        locked: '#5EEAD4',
        unlocked: '#FF6B6B',
        acOn: '#0F172A',
        sentryOn: '#FFB86B',
        doorOpen: '#FFB86B',
        online: '#5EEAD4',
        asleep: '#94A3B8',
        offline: '#CBD5E1',
        mapLine: '#0F172A',
        mapLineBorder: '#0a1628',
        mapStart: '#5EEAD4',
        mapEnd: '#0F172A',
      }
    }
    if (t === 'dark') {
      return {
        primary: '#e0e0e0',
        primaryLight: 'rgba(224, 224, 224, 0.08)',
        headerIcon: '#DCE3F1',
        chevron: '#64748B',
        inactiveIcon: 'rgba(255, 255, 255, 0.3)',
        inactiveIconLight: 'rgba(255, 255, 255, 0.15)',
        infoValue: '#DCE3F1',
        hint: '#64748B',
        quickActionIcon: '#DCE3F1',
        infoColor: '#b0b0b0',
        secondaryTextColor: '#94A3B8',
        iconColor: '#DCE3F1',
        success: '#5BE7C4',
        warning: '#fbbf24',
        danger: '#FF6B6B',
        info: '#b0b0b0',
        ai: '#7B6CFF',
        batteryHigh: '#5BE7C4',
        batteryMid: '#fbbf24',
        batteryLow: '#FF6B6B',
        charging: '#fbbf24',
        chargingComplete: '#5BE7C4',
        locked: '#5BE7C4',
        unlocked: '#FF6B6B',
        acOn: '#b0b0b0',
        sentryOn: '#fbbf24',
        doorOpen: '#fbbf24',
        online: '#5BE7C4',
        asleep: '#7C879B',
        offline: '#64748B',
        mapLine: '#e0e0e0',
        mapLineBorder: '#c0c0c0',
        mapStart: '#5BE7C4',
        mapEnd: '#e0e0e0',
      }
    }
    return {
      primary: '#1a1a1a',
      primaryLight: 'rgba(26, 26, 26, 0.08)',
      headerIcon: '#1F2937',
      chevron: '#8e8ea0',
      inactiveIcon: 'rgba(0, 0, 0, 0.25)',
      inactiveIconLight: 'rgba(0, 0, 0, 0.12)',
      infoValue: '#1F2937',
      hint: '#8e8ea0',
      quickActionIcon: '#555770',
      infoColor: '#4a4a4a',
      secondaryTextColor: '#6b7280',
      iconColor: '#1F2937',
      success: '#22C55E',
      warning: '#fbbf24',
      danger: '#FF6B6B',
      info: '#6b7280',
      ai: '#635BFF',
      batteryHigh: '#22C55E',
      batteryMid: '#fbbf24',
      batteryLow: '#FF6B6B',
      charging: '#fbbf24',
      chargingComplete: '#22C55E',
      locked: '#22C55E',
      unlocked: '#FF6B6B',
      acOn: '#4a4a4a',
      sentryOn: '#fbbf24',
      doorOpen: '#fbbf24',
      online: '#52c41a',
      asleep: '#fa8c16',
      offline: '#999999',
      mapLine: '#1a1a1a',
      mapLineBorder: '#333333',
      mapStart: '#52c41a',
      mapEnd: '#1a1a1a',
    }
  })

  const setThemeMode = (mode) => {
    themeMode.value = mode
    uni.setStorageSync('themeMode', mode)
    applyTheme()
  }

  const detectSystemTheme = () => {
    try {
      const res = uni.getSystemInfoSync()
      isDark.value = res.osTheme === 'dark'
    } catch (e) {
      isDark.value = false
    }
  }

  const applyTheme = () => {
    const theme = resolvedTheme.value
    const pages = getCurrentPages()
    if (pages.length > 0) {
      const page = pages[pages.length - 1]
      if (page?.$vm?.$el?.classList) {
        page.$vm.$el.classList.remove('light-theme', 'dark-theme', 'visionpro-theme')
        page.$vm.$el.classList.add(theme + '-theme')
      }
    }
    const navConfig = {
      light: { frontColor: '#000000', backgroundColor: '#ffffff' },
      dark: { frontColor: '#ffffff', backgroundColor: '#0f0f1a' },
      visionpro: { frontColor: '#000000', backgroundColor: '#EEF4FF' }
    }
    const cfg = navConfig[theme] || navConfig.light
    uni.setNavigationBarColor({
      frontColor: cfg.frontColor,
      backgroundColor: cfg.backgroundColor,
      animation: { duration: 200, timingFunc: 'easeIn' }
    })
  }

  const initTheme = () => {
    detectSystemTheme()
    applyTheme()
    uni.onThemeChange((result) => {
      isDark.value = result.theme === 'dark'
      if (themeMode.value === 'system') {
        applyTheme()
      }
    })
  }

  const toggleTheme = () => {
    const modes = ['light', 'dark', 'visionpro', 'system']
    const idx = modes.indexOf(themeMode.value)
    setThemeMode(modes[(idx + 1) % modes.length])
  }

  return {
    themeMode,
    isDark,
    resolvedTheme,
    isDarkMode,
    themeClass,
    colors,
    setThemeMode,
    detectSystemTheme,
    applyTheme,
    initTheme,
    toggleTheme
  }
})
