const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'https://your-domain.com'

let isRedirecting = false
let isRefreshing = false
let refreshSubscribers = []

const onRefreshed = (newToken) => {
  refreshSubscribers.forEach(callback => callback(newToken))
  refreshSubscribers = []
}

const subscribeTokenRefresh = (callback) => {
  refreshSubscribers.push(callback)
}

const doRefreshToken = async () => {
  const refreshToken = uni.getStorageSync('refreshToken')
  if (!refreshToken) {
    throw new Error('No refresh token')
  }

  return new Promise((resolve, reject) => {
    uni.request({
      url: BASE_URL + '/api/refresh-token',
      method: 'POST',
      data: { refresh_token: refreshToken },
      header: { 'Content-Type': 'application/json' },
      success: (res) => {
        if (res.data && res.data.code === 200) {
          const { token, refreshToken: newRefreshToken, expiresAt } = res.data.data
          uni.setStorageSync('token', token)
          uni.setStorageSync('refreshToken', newRefreshToken)
          uni.setStorageSync('tokenExpiresAt', expiresAt)
          resolve(token)
        } else {
          reject(new Error('Refresh failed'))
        }
      },
      fail: reject
    })
  })
}

const request = (options) => {
  return new Promise((resolve, reject) => {
    const token = uni.getStorageSync('token')
    const expiresAt = uni.getStorageSync('tokenExpiresAt')

    const doRequest = (currentToken) => {
      const url = options.url.startsWith('/') ? options.url : '/' + options.url

      uni.request({
        url: BASE_URL + url,
        method: options.method || 'GET',
        data: options.data || {},
        timeout: options.timeout || 30000,
        header: {
          'Content-Type': 'application/json',
          'Authorization': currentToken ? 'Bearer ' + currentToken : ''
        },
        success: (res) => {
          if (res.statusCode >= 200 && res.statusCode < 300) {
            if (res.data && res.data.code === 200) {
              resolve(res.data)
            } else if (res.data && res.data.code === 202) {
              resolve(res.data)
            } else if (res.data && res.data.code === 401) {
              handleTokenExpired(resolve, reject, options)
            } else if (res.data && res.data.code === 403) {
              reject(new Error(res.data.message || '权限不足'))
            } else if (res.data && res.data.code !== undefined) {
              reject(new Error(res.data.message || '请求失败'))
            } else {
              resolve(res.data)
            }
          } else if (res.statusCode === 401) {
            handleTokenExpired(resolve, reject, options)
          } else {
            reject(new Error('网络请求失败'))
          }
        },
        fail: (err) => {
          uni.showToast({
            title: '网络请求失败',
            icon: 'none'
          })
          reject(err)
        }
      })
    }

    const handleTokenExpired = (resolve, reject, options) => {
      if (isRefreshing) {
        subscribeTokenRefresh((newToken) => {
          doRequest(newToken)
        })
        return
      }

      isRefreshing = true
      doRefreshToken()
        .then(newToken => {
          isRefreshing = false
          onRefreshed(newToken)
          doRequest(newToken)
        })
        .catch(() => {
          isRefreshing = false
          handleUnauthorized()
          reject(new Error('登录已过期'))
        })
    }

    if (token && expiresAt && Date.now() / 1000 > expiresAt - 3600) {
      // 提前1小时刷新token（access token有效期7天）
      if (isRefreshing) {
        subscribeTokenRefresh((newToken) => {
          doRequest(newToken)
        })
        return
      }

      isRefreshing = true
      doRefreshToken()
        .then(newToken => {
          isRefreshing = false
          onRefreshed(newToken)
          doRequest(newToken)
        })
        .catch(() => {
          isRefreshing = false
          handleUnauthorized()
          reject(new Error('登录已过期'))
        })
    } else {
      doRequest(token)
    }
  })
}

const handleUnauthorized = () => {
  if (isRedirecting) return
  isRedirecting = true

  uni.removeStorageSync('token')
  uni.removeStorageSync('refreshToken')
  uni.removeStorageSync('tokenExpiresAt')
  uni.removeStorageSync('userInfo')
  uni.removeStorageSync('currentVehicle')

  uni.showToast({
    title: '登录已过期，请重新登录',
    icon: 'none',
    duration: 1500
  })

  setTimeout(() => {
    uni.reLaunch({
      url: '/pages/login/login',
      complete: () => {
        isRedirecting = false
      }
    })
  }, 600)
}

export const get = (url, params = {}) => {
  return request({ url, method: 'GET', data: params })
}

export const post = (url, data = {}) => {
  return request({ url, method: 'POST', data })
}

export const put = (url, data = {}) => {
  return request({ url, method: 'PUT', data })
}

export const del = (url, data = {}) => {
  return request({ url, method: 'DELETE', data })
}

export default request
