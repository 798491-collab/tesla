import { createSSRApp } from 'vue'
import App from './App.vue'
import * as Pinia from 'pinia'
import Icon from './components/Icon/Icon.vue'

export function createApp() {
  const app = createSSRApp(App)
  app.use(Pinia.createPinia())
  // 全局注册 Icon 组件
  app.component('Icon', Icon)
  return {
    app,
    Pinia
  }
}
