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
      userStore.fetchUserInfo().catch(() => {
        userStore.logout()
        uni.reLaunch({ url: '/pages/login/login' })
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
    if (!userStore.isLoggedIn && userStore.canRefresh) {
      userStore.refreshTokenAction().catch(() => {
        userStore.logout()
      })
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
