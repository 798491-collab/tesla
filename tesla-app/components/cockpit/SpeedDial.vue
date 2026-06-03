<template>
  <view class="speed-dial">
    <view
      class="canvas-wrap"
      id="speedDialContainer"
      :gaugeData="gaugeData"
      :change:gaugeData="dialRender.onDataChange"
    ></view>
  </view>
</template>

<script>
export default {
  props: {
    speed: { type: Number, default: 0 },
    maxSpeed: { type: Number, default: 240 },
    gear: { type: String, default: 'P' },
    soc: { type: Number, default: 0 },
    rangeKm: { type: Number, default: 0 },
    isCharging: { type: Boolean, default: false },
    power: { type: Number, default: 0 },
    theme: { type: String, default: 'dark' }
  },
  computed: {
    gaugeData() {
      return {
        speed: this.speed,
        maxSpeed: this.maxSpeed,
        gear: this.gear,
        soc: this.soc,
        rangeKm: this.rangeKm,
        isCharging: this.isCharging,
        power: this.power,
        theme: this.theme,
        ts: Date.now()
      }
    }
  }
}
</script>

<script module="dialRender" lang="renderjs">
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
    speedText: '#ffffff',
    unitText: 'rgba(255,255,255,0.3)',
    gearText: '#ffffff',
    gearInactive: 'rgba(255,255,255,0.3)',
    infoText: 'rgba(255,255,255,0.5)',
    dotFill: '#ffffff',
    chargingColor: '#4ade80'
  },
  light: {
    arcBg: 'rgba(0,0,0,0.06)',
    majorTick: 'rgba(0,0,0,0.2)',
    minorTick: 'rgba(0,0,0,0.06)',
    tickLabel: 'rgba(0,0,0,0.35)',
    speedText: '#1a1a2e',
    unitText: 'rgba(0,0,0,0.3)',
    gearText: '#1a1a2e',
    gearInactive: 'rgba(0,0,0,0.3)',
    infoText: 'rgba(0,0,0,0.5)',
    dotFill: '#1a1a2e',
    chargingColor: '#22c55e'
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
      gear: 'P',
      soc: 0,
      rangeKm: 0,
      isCharging: false,
      power: 0,
      theme: 'dark',
      animId: null,
      width: 0,
      height: 0,
      dpr: 1,
      lerpFactor: 0.1,
      breathPhase: 0,
      chargePhase: 0,
      prevSpeed: 0,
      accelGlow: 0,
      brakeGlow: 0
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
      const container = document.getElementById('speedDialContainer')
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
      if (!newVal) return
      if (typeof newVal.speed === 'number') {
        this.targetSpeed = newVal.speed
      }
      if (typeof newVal.maxSpeed === 'number' && newVal.maxSpeed > 0) {
        this.maxSpeed = newVal.maxSpeed
      }
      if (typeof newVal.gear === 'string') {
        this.gear = newVal.gear
      }
      if (typeof newVal.soc === 'number') {
        this.soc = newVal.soc
      }
      if (typeof newVal.rangeKm === 'number') {
        this.rangeKm = newVal.rangeKm
      }
      if (typeof newVal.isCharging === 'boolean') {
        this.isCharging = newVal.isCharging
      }
      if (typeof newVal.power === 'number') {
        this.power = newVal.power
      }
      if (typeof newVal.theme === 'string') {
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

        this.breathPhase += 0.02
        if (this.isCharging) {
          this.chargePhase += 0.03
        }

        const speedDelta = this.currentSpeed - this.prevSpeed
        if (speedDelta > 0.3) {
          this.accelGlow = Math.min(1, this.accelGlow + 0.08)
        } else {
          this.accelGlow = Math.max(0, this.accelGlow - 0.04)
        }
        if (speedDelta < -0.3 || this.power < -5) {
          this.brakeGlow = Math.min(1, this.brakeGlow + 0.08)
        } else {
          this.brakeGlow = Math.max(0, this.brakeGlow - 0.04)
        }

        this.prevSpeed = this.currentSpeed
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

      const isIdle = this.currentSpeed < 0.5 && !this.isCharging

      if (isIdle) {
        const breathAlpha = 0.06 + Math.sin(this.breathPhase) * 0.03
        ctx.beginPath()
        ctx.arc(cx, cy, r, START, END)
        ctx.strokeStyle = this.theme === 'dark'
          ? `rgba(255,255,255,${breathAlpha})`
          : `rgba(0,0,0,${breathAlpha})`
        ctx.lineWidth = 8
        ctx.lineCap = 'round'
        ctx.stroke()

        const breathGlowAlpha = 0.02 + Math.sin(this.breathPhase) * 0.015
        ctx.save()
        ctx.beginPath()
        ctx.arc(cx, cy, r, START, END)
        ctx.strokeStyle = this.theme === 'dark'
          ? `rgba(255,255,255,${breathGlowAlpha})`
          : `rgba(0,0,0,${breathGlowAlpha})`
        ctx.lineWidth = 20
        ctx.lineCap = 'round'
        ctx.stroke()
        ctx.restore()
      } else {
        ctx.beginPath()
        ctx.arc(cx, cy, r, START, END)
        ctx.strokeStyle = tc.arcBg
        ctx.lineWidth = 8
        ctx.lineCap = 'round'
        ctx.stroke()
      }

      const ratio = Math.min(Math.max(this.currentSpeed / this.maxSpeed, 0), 1)
      const activeEnd = START + ratio * SWEEP

      if (this.isCharging) {
        const chargeRatio = Math.min(Math.max(this.soc / 100, 0), 1)
        const chargeEnd = START + chargeRatio * SWEEP
        const pulse = Math.sin(this.chargePhase) * 0.015
        const displayEnd = START + Math.min(chargeRatio + pulse, 1) * SWEEP

        const segments = Math.max(2, Math.floor(chargeRatio * 50))
        const segAngle = (displayEnd - START) / segments

        for (let i = 0; i < segments; i++) {
          const t = i / Math.max(1, segments - 1)
          const a1 = START + i * segAngle
          const a2 = a1 + segAngle + 0.008
          const alpha = 0.3 + t * 0.7

          ctx.beginPath()
          ctx.arc(cx, cy, r, a1, a2)
          ctx.strokeStyle = `rgba(74,222,128,${alpha})`
          ctx.lineWidth = 8
          ctx.lineCap = (i === 0 || i === segments - 1) ? 'round' : 'butt'
          ctx.stroke()
        }

        ctx.save()
        ctx.beginPath()
        ctx.arc(cx, cy, r, START, displayEnd)
        ctx.strokeStyle = tc.chargingColor
        ctx.lineWidth = 14
        ctx.lineCap = 'round'
        ctx.globalAlpha = 0.25
        ctx.shadowBlur = 24
        ctx.shadowColor = tc.chargingColor
        ctx.stroke()
        ctx.restore()

        const dotX = cx + r * Math.cos(displayEnd)
        const dotY = cy + r * Math.sin(displayEnd)
        ctx.save()
        ctx.beginPath()
        ctx.arc(dotX, dotY, 5, 0, Math.PI * 2)
        ctx.fillStyle = tc.dotFill
        ctx.shadowBlur = 14
        ctx.shadowColor = tc.chargingColor
        ctx.fill()
        ctx.restore()

        const flowOffset = (this.chargePhase * 2) % (Math.PI * 2)
        for (let i = 0; i < 6; i++) {
          const dotAngle = START + ((flowOffset / (Math.PI * 2)) + i / 6) * chargeRatio * SWEEP
          if (dotAngle > displayEnd) continue
          const dx = cx + r * Math.cos(dotAngle)
          const dy = cy + r * Math.sin(dotAngle)
          const dotAlpha = 0.15 + Math.sin(this.chargePhase + i) * 0.1
          ctx.beginPath()
          ctx.arc(dx, dy, 2, 0, Math.PI * 2)
          ctx.fillStyle = `rgba(74,222,128,${dotAlpha})`
          ctx.fill()
        }
      } else if (this.currentSpeed > 0.5) {
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

        if (this.accelGlow > 0.01) {
          ctx.save()
          ctx.beginPath()
          ctx.arc(cx, cy, r + 4, START, activeEnd)
          ctx.strokeStyle = `rgba(96,165,250,${this.accelGlow * 0.3})`
          ctx.lineWidth = 6
          ctx.lineCap = 'round'
          ctx.shadowBlur = 20
          ctx.shadowColor = `rgba(96,165,250,${this.accelGlow * 0.5})`
          ctx.stroke()
          ctx.restore()
        }

        if (this.brakeGlow > 0.01) {
          ctx.save()
          ctx.beginPath()
          ctx.arc(cx, cy, r + 4, START, activeEnd)
          ctx.strokeStyle = `rgba(239,68,68,${this.brakeGlow * 0.3})`
          ctx.lineWidth = 6
          ctx.lineCap = 'round'
          ctx.shadowBlur = 20
          ctx.shadowColor = `rgba(239,68,68,${this.brakeGlow * 0.5})`
          ctx.stroke()
          ctx.restore()
        }

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
        const isMajor = i % 2 === 0
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

      ctx.fillStyle = tc.speedText
      ctx.font = '200 72px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'
      ctx.fillText(Math.round(this.currentSpeed).toString(), cx, cy - 16)

      ctx.fillStyle = tc.unitText
      ctx.font = '12px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.fillText('km/h', cx, cy + 24)

      const gears = ['P', 'R', 'N', 'D']
      const gearSpacing = 28
      const gearStartX = cx - ((gears.length - 1) * gearSpacing) / 2
      const gearY = cy + 50

      for (let i = 0; i < gears.length; i++) {
        const gx = gearStartX + i * gearSpacing
        const isActive = gears[i] === this.gear
        ctx.fillStyle = isActive ? tc.gearText : tc.gearInactive
        ctx.font = (isActive ? '600 ' : '400 ') + '18px -apple-system, BlinkMacSystemFont, sans-serif'
        ctx.textAlign = 'center'
        ctx.textBaseline = 'middle'
        ctx.fillText(gears[i], gx, gearY)
      }

      const infoY = cy + 74
      const socText = Math.round(this.soc) + '%'
      const rangeText = this.rangeKm > 0 ? Math.round(this.rangeKm) + ' km' : ''

      ctx.fillStyle = tc.infoText
      ctx.font = '400 13px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'

      if (rangeText) {
        ctx.fillText(socText + '  ·  ' + rangeText, cx, infoY)
      } else {
        ctx.fillText(socText, cx, infoY)
      }

      if (this.isCharging) {
        const pulseAlpha = 0.4 + Math.sin(this.chargePhase * 2) * 0.3
        ctx.fillStyle = this.theme === 'dark'
          ? `rgba(74,222,128,${pulseAlpha})`
          : `rgba(34,197,94,${pulseAlpha})`
        ctx.font = '600 9px -apple-system, BlinkMacSystemFont, sans-serif'
        ctx.textAlign = 'center'
        ctx.textBaseline = 'middle'
        ctx.fillText('CHARGING', cx, cy + 94)
      }
    }
  }
}
</script>

<style scoped>
.speed-dial {
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

@media screen and (orientation: landscape) {
  .canvas-wrap {
    width: 380rpx;
    height: 380rpx;
    max-width: 40vh;
    max-height: 40vh;
  }
}
</style>
