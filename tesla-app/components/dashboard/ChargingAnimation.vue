<template>
  <view class="charging-anim">
    <view
      class="canvas-wrap"
      id="chargingContainer"
      :animData="animData"
      :change:animData="chargingRender.onDataChange"
    ></view>
  </view>
</template>

<script>
export default {
  props: {
    soc: { type: Number, default: 0 },
    rangeKm: { type: Number, default: 0 },
    isCharging: { type: Boolean, default: false },
    chargeRate: { type: Number, default: 0 },
    theme: { type: String, default: 'dark' }
  },
  computed: {
    animData() {
      return {
        soc: this.soc,
        rangeKm: this.rangeKm,
        isCharging: this.isCharging,
        chargeRate: this.chargeRate,
        theme: this.theme,
        ts: Date.now()
      }
    }
  }
}
</script>

<script module="chargingRender" lang="renderjs">
const THEME_COLORS = {
  dark: {
    capFill: 'rgba(255,255,255,0.08)',
    borderStroke: 'rgba(255,255,255,0.15)',
    rangeText: 'rgba(255,255,255,0.35)',
    bubbleFill: 'rgba(255,255,255,'
  },
  light: {
    capFill: 'rgba(0,0,0,0.06)',
    borderStroke: 'rgba(0,0,0,0.12)',
    rangeText: 'rgba(0,0,0,0.35)',
    bubbleFill: 'rgba(255,255,255,'
  }
}

export default {
  data() {
    return {
      ctx: null,
      canvas: null,
      currentSoc: 0,
      targetSoc: 0,
      isCharging: false,
      chargeRate: 0,
      rangeKm: 0,
      animId: null,
      width: 0,
      height: 0,
      dpr: 1,
      time: 0,
      bubbles: [],
      particles: [],
      lerpFactor: 0.05,
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
      const container = document.getElementById('chargingContainer')
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
      if (typeof newVal.soc === 'number') this.targetSoc = newVal.soc
      if (typeof newVal.isCharging === 'boolean') this.isCharging = newVal.isCharging
      if (typeof newVal.chargeRate === 'number') this.chargeRate = newVal.chargeRate
      if (typeof newVal.rangeKm === 'number') this.rangeKm = newVal.rangeKm
      if (typeof newVal.theme === 'string') this.theme = newVal.theme
    },

    startAnimation() {
      const loop = () => {
        this.time += 0.016

        const diff = this.targetSoc - this.currentSoc
        if (Math.abs(diff) < 0.05) {
          this.currentSoc = this.targetSoc
        } else {
          this.currentSoc += diff * this.lerpFactor
        }

        if (this.isCharging) {
          this.spawnBubble()
          this.spawnParticle()
        }

        this.updateBubbles()
        this.updateParticles()
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

    spawnBubble() {
      if (Math.random() > 0.15) return
      const w = this.width
      const h = this.height
      const batW = w * 0.44
      const batH = h * 0.52
      const batX = (w - batW) / 2
      const batY = h * 0.06
      const fillH = (this.currentSoc / 100) * batH
      const waterY = batY + batH - fillH

      if (fillH < 10) return

      this.bubbles.push({
        x: batX + 10 + Math.random() * (batW - 20),
        y: batY + batH - 5,
        r: 1.5 + Math.random() * 3,
        speed: 0.4 + Math.random() * 0.8,
        topY: waterY,
        alpha: 0.3 + Math.random() * 0.4
      })
    },

    spawnParticle() {
      if (Math.random() > 0.08) return
      const w = this.width
      const h = this.height
      const cx = w / 2
      const batH = h * 0.52
      const batY = h * 0.06
      const batBottom = batY + batH

      this.particles.push({
        x: cx + (Math.random() - 0.5) * 60,
        y: batBottom + 10 + Math.random() * 20,
        vx: (Math.random() - 0.5) * 0.5,
        vy: -1.2 - Math.random() * 1.5,
        life: 1,
        decay: 0.012 + Math.random() * 0.01,
        size: 1 + Math.random() * 2
      })
    },

    updateBubbles() {
      for (let i = this.bubbles.length - 1; i >= 0; i--) {
        const b = this.bubbles[i]
        b.y -= b.speed
        b.x += Math.sin(this.time * 3 + i) * 0.3
        if (b.y < b.topY) {
          this.bubbles.splice(i, 1)
        }
      }
      if (this.bubbles.length > 30) {
        this.bubbles.splice(0, this.bubbles.length - 30)
      }
    },

    updateParticles() {
      for (let i = this.particles.length - 1; i >= 0; i--) {
        const p = this.particles[i]
        p.x += p.vx
        p.y += p.vy
        p.life -= p.decay
        if (p.life <= 0) {
          this.particles.splice(i, 1)
        }
      }
      if (this.particles.length > 40) {
        this.particles.splice(0, this.particles.length - 40)
      }
    },

    draw() {
      const ctx = this.ctx
      if (!ctx) return

      const w = this.width
      const h = this.height
      ctx.clearRect(0, 0, w, h)

      const tc = THEME_COLORS[this.theme] || THEME_COLORS.dark

      const batW = w * 0.44
      const batH = h * 0.52
      const batX = (w - batW) / 2
      const batY = h * 0.06
      const capW = batW * 0.38
      const capH = 12
      const capX = (w - capW) / 2
      const r = 14

      ctx.beginPath()
      ctx.roundRect(capX, batY - capH, capW, capH + 2, [r, r, 0, 0])
      ctx.fillStyle = tc.capFill
      ctx.fill()

      ctx.beginPath()
      ctx.roundRect(batX, batY, batW, batH, r)
      ctx.strokeStyle = tc.borderStroke
      ctx.lineWidth = 2
      ctx.stroke()

      const fillH = (this.currentSoc / 100) * batH
      if (fillH > 1) {
        const waterY = batY + batH - fillH

        ctx.save()
        ctx.beginPath()
        ctx.roundRect(batX + 2, batY + 2, batW - 4, batH - 4, r - 2)
        ctx.clip()

        const socColor = this.currentSoc > 60 ? '#4ade80' : this.currentSoc > 30 ? '#fbbf24' : '#ef4444'
        const socColorAlpha = this.currentSoc > 60 ? 'rgba(74,222,128,' : this.currentSoc > 30 ? 'rgba(251,191,36,' : 'rgba(239,68,68,'

        const grad = ctx.createLinearGradient(0, waterY, 0, batY + batH)
        grad.addColorStop(0, socColorAlpha + '0.6)')
        grad.addColorStop(1, socColorAlpha + '0.9)')
        ctx.fillStyle = grad

        ctx.beginPath()
        ctx.moveTo(batX, batY + batH)
        ctx.lineTo(batX + batW, batY + batH)
        ctx.lineTo(batX + batW, waterY)

        const waveAmp = this.isCharging ? 4 : 1.5
        const waveFreq = this.isCharging ? 0.04 : 0.03
        for (let x = batX + batW; x >= batX; x -= 2) {
          const wave1 = Math.sin(x * waveFreq + this.time * 2.5) * waveAmp
          const wave2 = Math.sin(x * waveFreq * 1.6 + this.time * 1.8) * waveAmp * 0.5
          ctx.lineTo(x, waterY + wave1 + wave2)
        }
        ctx.closePath()
        ctx.fill()

        if (this.isCharging) {
          const glowGrad = ctx.createLinearGradient(0, waterY - 8, 0, waterY + 8)
          glowGrad.addColorStop(0, 'rgba(255,255,255,0)')
          glowGrad.addColorStop(0.5, 'rgba(255,255,255,0.15)')
          glowGrad.addColorStop(1, 'rgba(255,255,255,0)')
          ctx.fillStyle = glowGrad
          ctx.fillRect(batX, waterY - 8, batW, 16)
        }

        for (const b of this.bubbles) {
          ctx.beginPath()
          ctx.arc(b.x, b.y, b.r, 0, Math.PI * 2)
          ctx.fillStyle = `${tc.bubbleFill}${b.alpha})`
          ctx.fill()
        }

        ctx.restore()
      }

      if (this.isCharging) {
        const cx = w / 2
        const batBottom = batY + batH

        for (const p of this.particles) {
          ctx.beginPath()
          ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2)
          ctx.fillStyle = `rgba(251,191,36,${p.life * 0.6})`
          ctx.fill()
        }

        const boltCx = cx
        const boltCy = batY + batH * 0.35
        const boltSize = 20 + Math.sin(this.time * 4) * 3
        const boltAlpha = 0.7 + Math.sin(this.time * 6) * 0.3

        ctx.save()
        ctx.translate(boltCx, boltCy)
        ctx.beginPath()
        ctx.moveTo(-3, -boltSize * 0.5)
        ctx.lineTo(4, -boltSize * 0.1)
        ctx.lineTo(0, -boltSize * 0.1)
        ctx.lineTo(3, boltSize * 0.5)
        ctx.lineTo(-4, boltSize * 0.1)
        ctx.lineTo(0, boltSize * 0.1)
        ctx.closePath()
        ctx.fillStyle = `rgba(251,191,36,${boltAlpha})`
        ctx.shadowBlur = 20
        ctx.shadowColor = 'rgba(251,191,36,0.6)'
        ctx.fill()
        ctx.restore()
      }

      const socColor = this.currentSoc > 60 ? '#4ade80' : this.currentSoc > 30 ? '#fbbf24' : '#ef4444'
      ctx.fillStyle = socColor
      ctx.font = '700 44px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'
      ctx.fillText(Math.round(this.currentSoc) + '%', w / 2, batY + batH + 48)

      ctx.fillStyle = tc.rangeText
      ctx.font = '13px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.fillText(Math.round(this.rangeKm) + ' km 续航', w / 2, batY + batH + 78)

      if (this.isCharging && this.chargeRate > 0) {
        const pulseAlpha = 0.5 + Math.sin(this.time * 3) * 0.3
        ctx.fillStyle = `rgba(251,191,36,${pulseAlpha})`
        ctx.font = '14px -apple-system, BlinkMacSystemFont, sans-serif'
        ctx.fillText('⚡ ' + this.chargeRate + ' kW 充电中', w / 2, batY + batH + 104)
      }
    }
  }
}
</script>

<style scoped>
.charging-anim {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
}

.canvas-wrap {
  width: 100%;
  height: 720rpx;
  position: relative;
}
</style>
