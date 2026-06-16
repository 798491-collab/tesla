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
    }
  },
  onShow: function() {
    console.log('App Show')
    uni.$emit('appShow')
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
