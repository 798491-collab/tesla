<template>
  <view class="power-gauge">
    <view
      class="canvas-wrap"
      id="powerContainer"
      :gaugeData="gaugeData"
      :change:gaugeData="powerRender.onDataChange"
    ></view>
  </view>
</template>

<script>
export default {
  props: {
    power: { type: Number, default: 0 },
    maxPower: { type: Number, default: 200 },
    theme: { type: String, default: 'dark' }
  },
  computed: {
    gaugeData() {
      return {
        power: this.power,
        maxPower: this.maxPower,
        theme: this.theme,
        ts: Date.now()
      }
    }
  }
}
</script>

<script module="powerRender" lang="renderjs">
const THEME_COLORS = {
  dark: {
    barBg: 'rgba(255,255,255,0.06)',
    centerLine: 'rgba(255,255,255,0.15)',
    majorTick: 'rgba(255,255,255,0.2)',
    minorTick: 'rgba(255,255,255,0.08)',
    tickLabel: 'rgba(255,255,255,0.25)',
    unitText: 'rgba(255,255,255,0.3)'
  },
  light: {
    barBg: 'rgba(0,0,0,0.06)',
    centerLine: 'rgba(0,0,0,0.12)',
    majorTick: 'rgba(0,0,0,0.15)',
    minorTick: 'rgba(0,0,0,0.06)',
    tickLabel: 'rgba(0,0,0,0.25)',
    unitText: 'rgba(0,0,0,0.3)'
  }
}

export default {
  data() {
    return {
      ctx: null,
      canvas: null,
      currentPower: 0,
      targetPower: 0,
      maxPower: 200,
      animId: null,
      width: 0,
      height: 0,
      dpr: 1,
      lerpFactor: 0.1,
      theme: 'dark'
    }
  },
  mounted() {
    this.$nextTick(() => {
      this.initCanvas()
      this.startAnimation()
    })
  },
  beforeDestroy() {
    this.stopAnimation()
  },
  methods: {
    initCanvas() {
      const container = document.getElementById('powerContainer')
      if (!container) return
      const rect = container.getBoundingClientRect()
      this.dpr = window.devicePixelRatio || 1
      this.width = rect.width
      this.height = rect.height

      const canvas = document.createElement('canvas')
      canvas.width = this.width * this.dpr
      canvas.height = this.height * this.dpr
      canvas.style.width = this.width + 'px'
      canvas.style.height = this.height + 'px'
      canvas.style.display = 'block'
      container.appendChild(canvas)

      this.ctx = canvas.getContext('2d')
      this.ctx.scale(this.dpr, this.dpr)
      this.canvas = canvas
    },

    onDataChange(newVal) {
      if (newVal && typeof newVal.power === 'number') {
        this.targetPower = newVal.power
      }
      if (newVal && typeof newVal.maxPower === 'number' && newVal.maxPower > 0) {
        this.maxPower = newVal.maxPower
      }
      if (newVal && typeof newVal.theme === 'string') {
        this.theme = newVal.theme
      }
    },

    startAnimation() {
      const loop = () => {
        const diff = this.targetPower - this.currentPower
        if (Math.abs(diff) < 0.1) {
          this.currentPower = this.targetPower
        } else {
          this.currentPower += diff * this.lerpFactor
        }
        this.draw()
        this.animId = requestAnimationFrame(loop)
      }
      loop()
    },

    stopAnimation() {
      if (this.animId) {
        cancelAnimationFrame(this.animId)
        this.animId = null
      }
    },

    draw() {
      const ctx = this.ctx
      if (!ctx) return

      const w = this.width
      const h = this.height
      ctx.clearRect(0, 0, w, h)

      const tc = THEME_COLORS[this.theme] || THEME_COLORS.dark

      const isRegen = this.currentPower < 0
      const absPower = Math.abs(this.currentPower)
      const ratio = Math.min(absPower / this.maxPower, 1)

      const barY = 28
      const barH = 14
      const barLeft = 8
      const barRight = w - 8
      const barW = barRight - barLeft
      const centerX = barLeft + barW / 2

      ctx.beginPath()
      ctx.roundRect(barLeft, barY, barW, barH, 7)
      ctx.fillStyle = tc.barBg
      ctx.fill()

      ctx.beginPath()
      ctx.moveTo(centerX, barY + 2)
      ctx.lineTo(centerX, barY + barH - 2)
      ctx.strokeStyle = tc.centerLine
      ctx.lineWidth = 1
      ctx.stroke()

      if (absPower > 0.3) {
        const fillW = ratio * (barW / 2)

        if (isRegen) {
          const grad = ctx.createLinearGradient(centerX, 0, centerX - fillW, 0)
          grad.addColorStop(0, 'rgba(96,165,250,0.9)')
          grad.addColorStop(1, 'rgba(96,165,250,0.3)')
          ctx.beginPath()
          ctx.roundRect(centerX - fillW, barY, fillW, barH, 7)
          ctx.fillStyle = grad
          ctx.fill()

          ctx.save()
          ctx.beginPath()
          ctx.roundRect(centerX - fillW, barY, fillW, barH, 7)
          ctx.strokeStyle = 'rgba(96,165,250,0.4)'
          ctx.lineWidth = 1
          ctx.shadowBlur = 12
          ctx.shadowColor = 'rgba(96,165,250,0.5)'
          ctx.stroke()
          ctx.restore()
        } else {
          const grad = ctx.createLinearGradient(centerX, 0, centerX + fillW, 0)
          if (ratio < 0.4) {
            grad.addColorStop(0, 'rgba(74,222,128,0.9)')
            grad.addColorStop(1, 'rgba(74,222,128,0.5)')
          } else if (ratio < 0.7) {
            grad.addColorStop(0, 'rgba(251,191,36,0.9)')
            grad.addColorStop(1, 'rgba(251,191,36,0.5)')
          } else {
            grad.addColorStop(0, 'rgba(251,191,36,0.9)')
            grad.addColorStop(1, 'rgba(239,68,68,0.8)')
          }
          ctx.beginPath()
          ctx.roundRect(centerX, barY, fillW, barH, 7)
          ctx.fillStyle = grad
          ctx.fill()

          ctx.save()
          ctx.beginPath()
          ctx.roundRect(centerX, barY, fillW, barH, 7)
          const glowColor = ratio < 0.4 ? 'rgba(74,222,128,0.5)' : ratio < 0.7 ? 'rgba(251,191,36,0.5)' : 'rgba(239,68,68,0.5)'
          ctx.strokeStyle = glowColor
          ctx.lineWidth = 1
          ctx.shadowBlur = 12
          ctx.shadowColor = glowColor
          ctx.stroke()
          ctx.restore()
        }
      }

      const tickCount = 10
      for (let i = 0; i <= tickCount; i++) {
        const x = barLeft + (i / tickCount) * barW
        const isMajor = i % 5 === 0
        const tickH = isMajor ? 5 : 3

        ctx.beginPath()
        ctx.moveTo(x, barY + barH + 3)
        ctx.lineTo(x, barY + barH + 3 + tickH)
        ctx.strokeStyle = isMajor ? tc.majorTick : tc.minorTick
        ctx.lineWidth = 1
        ctx.stroke()

        if (isMajor) {
          const val = Math.round((i / tickCount - 0.5) * 2 * this.maxPower)
          ctx.fillStyle = tc.tickLabel
          ctx.font = '9px -apple-system, BlinkMacSystemFont, sans-serif'
          ctx.textAlign = 'center'
          ctx.textBaseline = 'top'
          ctx.fillText(val.toString(), x, barY + barH + 10)
        }
      }

      const powerColor = isRegen ? '#60a5fa' : (ratio > 0.7 ? '#ef4444' : ratio > 0.4 ? '#fbbf24' : '#4ade80')

      ctx.fillStyle = powerColor
      ctx.font = '200 24px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'
      ctx.fillText(Math.round(this.currentPower).toString(), w / 2, 12)

      ctx.fillStyle = tc.unitText
      ctx.font = '10px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.fillText('kW', w / 2 + 28, 12)

      if (isRegen) {
        ctx.fillStyle = 'rgba(96,165,250,0.5)'
        ctx.font = '8px -apple-system, BlinkMacSystemFont, sans-serif'
        ctx.textAlign = 'left'
        ctx.fillText('REGEN', barLeft, 12)
      }
    }
  }
}
</script>

<style scoped>
.power-gauge {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
}

.canvas-wrap {
  width: 100%;
  height: 120rpx;
  max-height: 15vh;
  position: relative;
}

/* 横屏时调整功率表 */
@media screen and (orientation: landscape) {
  .canvas-wrap {
    height: 100rpx;
    max-height: 12vh;
  }
}
</style>
