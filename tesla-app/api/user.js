import { get, post } from '@/utils/request.js'

export const login = (data) => {
  return post('/api/login', data)
}

export const register = (data) => {
  return post('/api/register', data)
}

export const logout = () => {
  return post('/api/logout')
}

export const getUserInfo = () => {
  return get('/api/user/info')
}

export const changePassword = (data) => {
  return post('/api/user/change_password', data)
}

export const updateUserInfo = (data) => {
  return post('/api/user/update', data)
}
