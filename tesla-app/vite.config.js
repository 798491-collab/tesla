import uni from '@dcloudio/vite-plugin-uni'

export default {
  plugins: [
    uni()
  ],
  base: './',
  build: {
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true
      }
    }
  },
  server: {
    port: 3000,
    host: '0.0.0.0'
  }
}
