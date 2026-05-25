<template>
  <view class="battery-gauge">
    <view
      class="canvas-wrap"
      id="batteryContainer"
      :gaugeData="gaugeData"
      :change:gaugeData="batteryRender.onDataChange"
    ></view>
  </view>
</template>

<script>
export default {
  props: {
    soc: { type: Number, default: 0 },
    rangeKm: { type: Number, default: 0 },
    isCharging: { type: Boolean, default: false }
  },
  computed: {
    gaugeData() {
      return {
        soc: this.soc,
        rangeKm: this.rangeKm,
        isCharging: this.isCharging,
        ts: Date.now()
      }
    }
  }
}
</script>

<script module="batteryRender" lang="renderjs">
function getSocColor(soc) {
  if (soc > 60) return { r: 74, g: 222, b: 128 }
  if (soc > 30) return { r: 251, g: 191, b: 36 }
  return { r: 239, g: 68, b: 68 }
}

function colorToStr(c, alpha) {
  if (alpha !== undefined) return `rgba(${c.r},${c.g},${c.b},${alpha})`
  return `rgb(${c.r},${c.g},${c.b})`
}

export default {
  data() {
    return {
      ctx: null,
      canvas: null,
      currentSoc: 0,
      targetSoc: 0,
      rangeKm: 0,
      isCharging: false,
      animId: null,
      chargePhase: 0,
      width: 0,
      height: 0,
      dpr: 1,
      lerpFactor: 0.06
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
      const container = document.getElementById('batteryContainer')
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
      if (newVal && typeof newVal.soc === 'number') {
        this.targetSoc = newVal.soc
      }
      if (newVal && typeof newVal.rangeKm === 'number') {
        this.rangeKm = newVal.rangeKm
      }
      if (newVal && typeof newVal.isCharging === 'boolean') {
        this.isCharging = newVal.isCharging
      }
    },

    startAnimation() {
      const loop = () => {
        const diff = this.targetSoc - this.currentSoc
        if (Math.abs(diff) < 0.05) {
          this.currentSoc = this.targetSoc
        } else {
          this.currentSoc += diff * this.lerpFactor
        }
        if (this.isCharging) {
          this.chargePhase += 0.03
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
      const cx = w / 2
      const cy = h / 2
      const r = Math.max(1, Math.min(cx, cy) - 14)

      ctx.clearRect(0, 0, w, h)

      const START = Math.PI * 0.75
      const END = Math.PI * 2.25
      const SWEEP = END - START

      ctx.beginPath()
      ctx.arc(cx, cy, r, START, END)
      ctx.strokeStyle = 'rgba(255,255,255,0.06)'
      ctx.lineWidth = 6
      ctx.lineCap = 'round'
      ctx.stroke()

      const ratio = Math.min(Math.max(this.currentSoc / 100, 0), 1)
      const activeEnd = START + ratio * SWEEP
      const socColor = getSocColor(this.currentSoc)
      const mainColor = colorToStr(socColor)

      if (this.currentSoc > 0.5) {
        let displayEnd = activeEnd
        if (this.isCharging) {
          const pulse = Math.sin(this.chargePhase) * 0.02
          displayEnd = START + Math.min(ratio + pulse, 1) * SWEEP
        }

        const segments = Math.max(2, Math.floor(ratio * 40))
        const segAngle = (displayEnd - START) / segments

        for (let i = 0; i < segments; i++) {
          const t = i / Math.max(1, segments - 1)
          const a1 = START + i * segAngle
          const a2 = a1 + segAngle + 0.008
          const alpha = 0.3 + t * 0.7

          ctx.beginPath()
          ctx.arc(cx, cy, r, a1, a2)
          ctx.strokeStyle = colorToStr(socColor, alpha)
          ctx.lineWidth = 6
          ctx.lineCap = (i === 0 || i === segments - 1) ? 'round' : 'butt'
          ctx.stroke()
        }

        ctx.save()
        ctx.beginPath()
        ctx.arc(cx, cy, r, START, displayEnd)
        ctx.strokeStyle = mainColor
        ctx.lineWidth = 12
        ctx.lineCap = 'round'
        ctx.globalAlpha = 0.2
        ctx.shadowBlur = 20
        ctx.shadowColor = mainColor
        ctx.stroke()
        ctx.restore()

        const dotX = cx + r * Math.cos(displayEnd)
        const dotY = cy + r * Math.sin(displayEnd)
        ctx.save()
        ctx.beginPath()
        ctx.arc(dotX, dotY, 4, 0, Math.PI * 2)
        ctx.fillStyle = '#ffffff'
        ctx.shadowBlur = 10
        ctx.shadowColor = mainColor
        ctx.fill()
        ctx.restore()
      }

      const totalTicks = 10
      for (let i = 0; i <= totalTicks; i++) {
        const angle = START + (i / totalTicks) * SWEEP
        const isMajor = i % 5 === 0
        const innerR = r - (isMajor ? 14 : 9)
        const outerR = r - 4

        ctx.beginPath()
        ctx.moveTo(cx + innerR * Math.cos(angle), cy + innerR * Math.sin(angle))
        ctx.lineTo(cx + outerR * Math.cos(angle), cy + outerR * Math.sin(angle))
        ctx.strokeStyle = isMajor ? 'rgba(255,255,255,0.2)' : 'rgba(255,255,255,0.06)'
        ctx.lineWidth = isMajor ? 1.5 : 1
        ctx.lineCap = 'round'
        ctx.stroke()

        if (isMajor) {
          const textR = r - 24
          const val = i * 10
          ctx.fillStyle = 'rgba(255,255,255,0.3)'
          ctx.font = '10px -apple-system, BlinkMacSystemFont, sans-serif'
          ctx.textAlign = 'center'
          ctx.textBaseline = 'middle'
          ctx.fillText(val + '%', cx + textR * Math.cos(angle), cy + textR * Math.sin(angle))
        }
      }

      ctx.fillStyle = mainColor
      ctx.font = '200 40px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'
      ctx.fillText(Math.round(this.currentSoc).toString(), cx, cy - 10)

      ctx.fillStyle = 'rgba(255,255,255,0.3)'
      ctx.font = '12px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.fillText('%', cx, cy + 16)

      if (this.rangeKm > 0) {
        ctx.fillStyle = 'rgba(255,255,255,0.5)'
        ctx.font = '600 14px -apple-system, BlinkMacSystemFont, sans-serif'
        ctx.fillText(Math.round(this.rangeKm) + ' km', cx, cy + 34)
      }

      if (this.isCharging) {
        const pulseAlpha = 0.4 + Math.sin(this.chargePhase * 2) * 0.3
        ctx.fillStyle = colorToStr({ r: 251, g: 191, b: 36 }, pulseAlpha)
        ctx.font = '600 9px -apple-system, BlinkMacSystemFont, sans-serif'
        ctx.fillText('CHARGING', cx, cy + 50)
      }
    }
  }
}
</script>

<style scoped>
.battery-gauge {
  display: flex;
  justify-content: center;
  align-items: center;
}

.canvas-wrap {
  width: 300rpx;
  height: 300rpx;
  position: relative;
}
</style>
