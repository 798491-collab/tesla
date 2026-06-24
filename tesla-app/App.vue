<script>
import { useThemeStore } from '@/store/theme'
import { useUserStore } from '@/store/user'

export default {
  onLaunch: function() {
    console.log('App Launch')
    const themeStore = useThemeStore()
    themeStore.initTheme()

    const userStore = useUserStore()
    if (userStore.isLoggedIn) {
      // token有效，但如果即将过期（1小时内），提前刷新
      if (userStore.isTokenExpiringSoon) {
        userStore.refreshTokenAction().catch(() => {})
      }
      userStore.fetchUserInfo().catch(() => {
        // fetchUserInfo失败可能是token已失效，尝试刷新
        userStore.refreshTokenAction().then(() => {
          return userStore.fetchUserInfo()
        }).catch(() => {
          userStore.logout()
          uni.reLaunch({ url: '/pages/login/login' })
        })
      })
    } else if (userStore.canRefresh) {
      userStore.refreshTokenAction().then(() => {
        userStore.fetchUserInfo().catch(() => {})
      }).catch(() => {
        userStore.logout()
      })
    }
  },
  onShow: function() {
    console.log('App Show')
    uni.$emit('appShow')

    const userStore = useUserStore()
    // App从后台恢复时，检查token状态
    if (!userStore.isLoggedIn && userStore.canRefresh) {
      // token已过期，尝试用refreshToken刷新
      userStore.refreshTokenAction().catch(() => {
        userStore.logout()
      })
    } else if (userStore.isLoggedIn && userStore.isTokenExpiringSoon) {
      // token即将过期，提前刷新，避免后续请求失败
      userStore.refreshTokenAction().catch(() => {})
    }
  },
  onHide: function() {
    console.log('App Hide')
  }
}
</script>

<style lang="scss">
/* #ifndef APP-NVUE */
@import "@/styles/common.scss";
/* #endif */
</style>
