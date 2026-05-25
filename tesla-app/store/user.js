import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login, getUserInfo } from '@/api/user.js'

export const useUserStore = defineStore('user', () => {
  const token = ref(uni.getStorageSync('token') || '')
  const expiresAt = ref(uni.getStorageSync('tokenExpiresAt') || 0)
  const userInfo = ref(uni.getStorageSync('userInfo') || null)
  const currentVehicle = ref(uni.getStorageSync('currentVehicle') || null)
  const savedUsername = ref(uni.getStorageSync('savedUsername') || '')

  const isLoggedIn = computed(() => {
    if (!token.value) return false
    if (expiresAt.value && Date.now() / 1000 > expiresAt.value) {
      clearAuth()
      return false
    }
    return true
  })

  const isTokenExpiringSoon = computed(() => {
    if (!token.value || !expiresAt.value) return false
    const remaining = expiresAt.value - Date.now() / 1000
    return remaining > 0 && remaining < 300
  })

  const setToken = (newToken, newExpiresAt) => {
    token.value = newToken
    uni.setStorageSync('token', newToken)
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
    expiresAt.value = 0
    userInfo.value = null
    currentVehicle.value = null
    uni.removeStorageSync('token')
    uni.removeStorageSync('tokenExpiresAt')
    uni.removeStorageSync('userInfo')
    uni.removeStorageSync('currentVehicle')
  }

  const loginAction = async (data) => {
    const res = await login(data)
    setToken(res.data.token, res.data.expiresAt)
    setUserInfo(res.data.user)
    // 保存用户名以便下次自动填充
    if (data.username) {
      savedUsername.value = data.username
      uni.setStorageSync('savedUsername', data.username)
    }
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

  const checkTokenExpiry = () => {
    if (!token.value) return false
    if (expiresAt.value && Date.now() / 1000 > expiresAt.value) {
      clearAuth()
      return false
    }
    return true
  }

  return {
    token,
    expiresAt,
    userInfo,
    currentVehicle,
    savedUsername,
    isLoggedIn,
    isTokenExpiringSoon,
    setToken,
    setUserInfo,
    setCurrentVehicle,
    clearAuth,
    loginAction,
    fetchUserInfo,
    logout,
    checkTokenExpiry
  }
})
