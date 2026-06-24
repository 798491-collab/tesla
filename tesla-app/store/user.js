import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login, getUserInfo, refreshToken as refreshTokenApi } from '@/api/user.js'

export const useUserStore = defineStore('user', () => {
  const token = ref(uni.getStorageSync('token') || '')
  const refreshToken = ref(uni.getStorageSync('refreshToken') || '')
  const expiresAt = ref(uni.getStorageSync('tokenExpiresAt') || 0)
  const userInfo = ref(uni.getStorageSync('userInfo') || null)
  const currentVehicle = ref(uni.getStorageSync('currentVehicle') || null)
  const savedUsername = ref(uni.getStorageSync('savedUsername') || '')

  const isLoggedIn = computed(() => {
    if (!token.value) return false
    if (expiresAt.value && Date.now() / 1000 > expiresAt.value) {
      return false
    }
    return true
  })

  const isTokenExpiringSoon = computed(() => {
    if (!token.value || !expiresAt.value) return false
    const remaining = expiresAt.value - Date.now() / 1000
    return remaining > 0 && remaining < 300
  })

  const setToken = (newToken, newRefreshToken, newExpiresAt) => {
    token.value = newToken
    uni.setStorageSync('token', newToken)
    if (newRefreshToken) {
      refreshToken.value = newRefreshToken
      uni.setStorageSync('refreshToken', newRefreshToken)
    }
    if (newExpiresAt) {
      expiresAt.value = newExpiresAt
      uni.setStorageSync('tokenExpiresAt', newExpiresAt)
    }
  }

  const setUserInfo = (info) => {
    userInfo.value = info
    uni.setStorageSync('userInfo', info)
  }

  const setCurrentVehicle = (vehicle) => {
    currentVehicle.value = vehicle
    uni.setStorageSync('currentVehicle', vehicle)
  }

  const clearAuth = () => {
    token.value = ''
    refreshToken.value = ''
    expiresAt.value = 0
    userInfo.value = null
    currentVehicle.value = null
    uni.removeStorageSync('token')
    uni.removeStorageSync('refreshToken')
    uni.removeStorageSync('tokenExpiresAt')
    uni.removeStorageSync('userInfo')
    uni.removeStorageSync('currentVehicle')
  }

  const loginAction = async (data) => {
    const res = await login(data)
    setToken(res.data.token, res.data.refreshToken, res.data.expiresAt)
    setUserInfo(res.data.user)
    if (data.username) {
      savedUsername.value = data.username
      uni.setStorageSync('savedUsername', data.username)
    }
    return res
  }

  const refreshTokenAction = async () => {
    if (!refreshToken.value) {
      clearAuth()
      throw new Error('No refresh token available')
    }
    const res = await refreshTokenApi({ refresh_token: refreshToken.value })
    setToken(res.data.token, res.data.refreshToken, res.data.expiresAt)
    return res
  }

  const fetchUserInfo = async () => {
    const res = await getUserInfo()
    setUserInfo(res.data)
    return res.data
  }

  const logout = () => {
    clearAuth()
  }

  const canRefresh = computed(() => {
    return !token.value || (expiresAt.value && Date.now() / 1000 > expiresAt.value)
      ? !!refreshToken.value
      : false
  })

  const checkTokenExpiry = () => {
    if (!token.value) return false
    if (expiresAt.value && Date.now() / 1000 > expiresAt.value) {
      return false
    }
    return true
  }

  return {
    token,
    refreshToken,
    expiresAt,
    userInfo,
    currentVehicle,
    savedUsername,
    isLoggedIn,
    isTokenExpiringSoon,
    canRefresh,
    setToken,
    setUserInfo,
    setCurrentVehicle,
    clearAuth,
    loginAction,
    refreshTokenAction,
    fetchUserInfo,
    logout,
    checkTokenExpiry
  }
})
