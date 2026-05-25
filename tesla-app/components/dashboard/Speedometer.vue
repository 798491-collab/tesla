<template>
  <view class="speedometer">
    <view
      class="canvas-wrap"
      id="speedContainer"
      :gaugeData="gaugeData"
      :change:gaugeData="speedRender.onDataChange"
    ></view>
  </view>
</template>

<script>
export default {
  props: {
    speed: { type: Number, default: 0 },
    maxSpeed: { type: Number, default: 240 },
    theme: { type: String, default: 'dark' }
  },
  computed: {
    gaugeData() {
      return {
        speed: this.speed,
        maxSpeed: this.maxSpeed,
        theme: this.theme,
        ts: Date.now()
      }
    }
  }
}
</script>

<script module="speedRender" lang="renderjs">
const COLOR_STOPS = [
  { pos: 0, r: 96, g: 165, b: 250 },
  { pos: 0.35, r: 74, g: 222, b: 128 },
  { pos: 0.65, r: 251, g: 191, b: 36 },
  { pos: 1.0, r: 239, g: 68, b: 68 }
]

const THEME_COLORS = {
  dark: {
    arcBg: 'rgba(255,255,255,0.06)',
    majorTick: 'rgba(255,255,255,0.3)',
    minorTick: 'rgba(255,255,255,0.08)',
    tickLabel: 'rgba(255,255,255,0.4)',
    innerCircle: 'rgba(255,255,255,0.03)',
    speedText: '#ffffff',
    unitText: 'rgba(255,255,255,0.3)',
    dotFill: '#ffffff'
  },
  light: {
    arcBg: 'rgba(0,0,0,0.06)',
    majorTick: 'rgba(0,0,0,0.2)',
    minorTick: 'rgba(0,0,0,0.06)',
    tickLabel: 'rgba(0,0,0,0.35)',
    innerCircle: 'rgba(0,0,0,0.03)',
    speedText: '#1a1a2e',
    unitText: 'rgba(0,0,0,0.3)',
    dotFill: '#1a1a2e'
  }
}

function interpolateColor(t) {
  t = Math.max(0, Math.min(1, t))
  let lower = COLOR_STOPS[0]
  let upper = COLOR_STOPS[COLOR_STOPS.length - 1]
  for (let i = 0; i < COLOR_STOPS.length - 1; i++) {
    if (t >= COLOR_STOPS[i].pos && t <= COLOR_STOPS[i + 1].pos) {
      lower = COLOR_STOPS[i]
      upper = COLOR_STOPS[i + 1]
      break
    }
  }
  const range = upper.pos - lower.pos
  const lt = range === 0 ? 0 : (t - lower.pos) / range
  const r = Math.round(lower.r + (upper.r - lower.r) * lt)
  const g = Math.round(lower.g + (upper.g - lower.g) * lt)
  const b = Math.round(lower.b + (upper.b - lower.b) * lt)
  return `rgb(${r},${g},${b})`
}

export default {
  data() {
    return {
      ctx: null,
      canvas: null,
      currentSpeed: 0,
      targetSpeed: 0,
      maxSpeed: 240,
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
      const container = document.getElementById('speedContainer')
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
      if (newVal && typeof newVal.speed === 'number') {
        this.targetSpeed = newVal.speed
      }
      if (newVal && typeof newVal.maxSpeed === 'number' && newVal.maxSpeed > 0) {
        this.maxSpeed = newVal.maxSpeed
      }
      if (newVal && typeof newVal.theme === 'string') {
        this.theme = newVal.theme
      }
    },

    startAnimation() {
      const loop = () => {
        const diff = this.targetSpeed - this.currentSpeed
        if (Math.abs(diff) < 0.1) {
          this.currentSpeed = this.targetSpeed
        } else {
          this.currentSpeed += diff * this.lerpFactor
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
      const r = Math.max(1, Math.min(cx, cy) - 18)
      const tc = THEME_COLORS[this.theme] || THEME_COLORS.dark

      ctx.clearRect(0, 0, w, h)

      const START = Math.PI * 0.75
      const END = Math.PI * 2.25
      const SWEEP = END - START

      ctx.beginPath()
      ctx.arc(cx, cy, r, START, END)
      ctx.strokeStyle = tc.arcBg
      ctx.lineWidth = 8
      ctx.lineCap = 'round'
      ctx.stroke()

      const ratio = Math.min(Math.max(this.currentSpeed / this.maxSpeed, 0), 1)
      const activeEnd = START + ratio * SWEEP

      if (this.currentSpeed > 0.5) {
        const segments = Math.max(2, Math.floor(ratio * 50))
        const segAngle = (activeEnd - START) / segments

        for (let i = 0; i < segments; i++) {
          const t = segments === 1 ? ratio : (i / (segments - 1)) * ratio
          const a1 = START + i * segAngle
          const a2 = a1 + segAngle + 0.008

          ctx.beginPath()
          ctx.arc(cx, cy, r, a1, a2)
          ctx.strokeStyle = interpolateColor(t)
          ctx.lineWidth = 8
          ctx.lineCap = (i === 0 || i === segments - 1) ? 'round' : 'butt'
          ctx.stroke()
        }

        ctx.save()
        ctx.beginPath()
        ctx.arc(cx, cy, r, START, activeEnd)
        ctx.strokeStyle = interpolateColor(ratio)
        ctx.lineWidth = 14
        ctx.lineCap = 'round'
        ctx.globalAlpha = 0.25
        ctx.shadowBlur = 24
        ctx.shadowColor = interpolateColor(ratio)
        ctx.stroke()
        ctx.restore()

        const dotX = cx + r * Math.cos(activeEnd)
        const dotY = cy + r * Math.sin(activeEnd)
        ctx.save()
        ctx.beginPath()
        ctx.arc(dotX, dotY, 5, 0, Math.PI * 2)
        ctx.fillStyle = tc.dotFill
        ctx.shadowBlur = 14
        ctx.shadowColor = interpolateColor(ratio)
        ctx.fill()
        ctx.restore()
      }

      const totalTicks = this.maxSpeed / 10
      for (let i = 0; i <= totalTicks; i++) {
        const angle = START + (i / totalTicks) * SWEEP
        const isMajor = i % 4 === 0
        const innerR = r - (isMajor ? 18 : 11)
        const outerR = r - 6

        ctx.beginPath()
        ctx.moveTo(cx + innerR * Math.cos(angle), cy + innerR * Math.sin(angle))
        ctx.lineTo(cx + outerR * Math.cos(angle), cy + outerR * Math.sin(angle))
        ctx.strokeStyle = isMajor ? tc.majorTick : tc.minorTick
        ctx.lineWidth = isMajor ? 2 : 1
        ctx.lineCap = 'round'
        ctx.stroke()

        if (isMajor) {
          const textR = r - 30
          const val = i * 10
          ctx.fillStyle = tc.tickLabel
          ctx.font = '11px -apple-system, BlinkMacSystemFont, sans-serif'
          ctx.textAlign = 'center'
          ctx.textBaseline = 'middle'
          ctx.fillText(val.toString(), cx + textR * Math.cos(angle), cy + textR * Math.sin(angle))
        }
      }

      ctx.beginPath()
      ctx.arc(cx, cy, r - 42, 0, Math.PI * 2)
      ctx.strokeStyle = tc.innerCircle
      ctx.lineWidth = 1
      ctx.stroke()

      ctx.fillStyle = tc.speedText
      ctx.font = '200 52px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'
      ctx.fillText(Math.round(this.currentSpeed).toString(), cx, cy - 8)

      ctx.fillStyle = tc.unitText
      ctx.font = '12px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.fillText('km/h', cx, cy + 26)
    }
  }
}
</script>

<style scoped>
.speedometer {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 0;
  width: 100%;
}

.canvas-wrap {
  width: 580rpx;
  height: 580rpx;
  max-width: 45vh;
  max-height: 45vh;
  position: relative;
}

/* 横屏时缩小速度表 */
@media screen and (orientation: landscape) {
  .canvas-wrap {
    width: 380rpx;
    height: 380rpx;
    max-width: 40vh;
    max-height: 40vh;
  }
}
</style>
