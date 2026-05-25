const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'https://your-domain.com'

let isRedirecting = false

const request = (options) => {
  return new Promise((resolve, reject) => {
    const token = uni.getStorageSync('token')
    const expiresAt = uni.getStorageSync('tokenExpiresAt')

    if (token && expiresAt && Date.now() / 1000 > expiresAt) {
      handleUnauthorized()
      reject(new Error('登录已过期'))
      return
    }

    const url = options.url.startsWith('/') ? options.url : '/' + options.url

    uni.request({
      url: BASE_URL + url,
      method: options.method || 'GET',
      data: options.data || {},
      timeout: options.timeout || 30000,
      header: {
        'Content-Type': 'application/json',
        'Authorization': token ? 'Bearer ' + token : ''
      },
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          if (res.data && res.data.code === 200) {
            resolve(res.data)
          } else if (res.data && res.data.code === 401) {
            handleUnauthorized()
            setTimeout(() => {
              reject(new Error('登录已过期'))
            }, 1000)
          } else if (res.data && res.data.code !== undefined) {
            reject(new Error(res.data.message || '请求失败'))
          } else {
            resolve(res.data)
          }
        } else if (res.statusCode === 401) {
          handleUnauthorized()
          setTimeout(() => {
            reject(new Error('登录已过期'))
          }, 1000)
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
  })
}

const handleUnauthorized = () => {
  if (isRedirecting) return
  isRedirecting = true

  uni.removeStorageSync('token')
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
