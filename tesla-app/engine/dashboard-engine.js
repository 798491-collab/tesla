const _raf = typeof requestAnimationFrame !== 'undefined'
  ? requestAnimationFrame
  : (cb) => setTimeout(cb, 16)

const _caf = typeof cancelAnimationFrame !== 'undefined'
  ? cancelAnimationFrame
  : clearTimeout

class DashboardEngine {
  constructor(options = {}) {
    this.rawState = {
      speed: 0,
      soc: 0,
      heading: 0,
      gear: 'P',
      power: 0,
      lat: 0,
      lng: 0,
      timestamp: 0
    }

    this.renderState = {
      speed: 0,
      soc: 0,
      heading: 0,
      power: 0,
      lat: 0,
      lng: 0
    }

    this.lerpFactor = options.lerpFactor || 0.08
    this.isRunning = false
    this._rafId = null
    this._lastTime = 0

    this.onUpdate = options.onUpdate || (() => {})
    this.onRender = options.onRender || (() => {})
  }

  updateRawState(newState) {
    this.rawState = {
      ...this.rawState,
      ...newState,
      timestamp: Date.now()
    }
  }

  lerp(start, end, t) {
    return start + (end - start) * t
  }

  lerpAngle(start, end, t) {
    let diff = end - start
    if (diff > 180) diff -= 360
    if (diff < -180) diff += 360
    return start + diff * t
  }

  computeRenderState() {
    this.renderState.speed = this.lerp(
      this.renderState.speed,
      this.rawState.speed,
      this.lerpFactor
    )

    this.renderState.soc = this.lerp(
      this.renderState.soc,
      this.rawState.soc,
      this.lerpFactor * 0.5
    )

    this.renderState.power = this.lerp(
      this.renderState.power,
      this.rawState.power,
      this.lerpFactor
    )

    this.renderState.heading = this.lerpAngle(
      this.renderState.heading,
      this.rawState.heading,
      this.lerpFactor
    )

    this.renderState.lat = this.lerp(
      this.renderState.lat,
      this.rawState.lat,
      this.lerpFactor
    )
    this.renderState.lng = this.lerp(
      this.renderState.lng,
      this.rawState.lng,
      this.lerpFactor
    )

    this.renderState.gear = this.rawState.gear
  }

  tick(timestamp) {
    if (!this.isRunning) return

    if (!this._lastTime) this._lastTime = timestamp
    const delta = timestamp - this._lastTime
    this._lastTime = timestamp

    if (delta > 0 && delta < 100) {
      this.computeRenderState()
      this.onUpdate(this.renderState)
      this.onRender(this.renderState)
    }

    this._rafId = _raf((t) => this.tick(t || Date.now()))
  }

  start() {
    if (this.isRunning) return
    this.isRunning = true
    this._lastTime = 0
    this._rafId = _raf((t) => this.tick(t || Date.now()))
  }

  stop() {
    this.isRunning = false
    if (this._rafId) {
      _caf(this._rafId)
      this._rafId = null
    }
    this._lastTime = 0
  }

  getZoomBySpeed(speed) {
    if (speed <= 20) return 18
    if (speed <= 60) return 16
    if (speed <= 100) return 15
    return 14
  }

  getState() {
    return {
      raw: this.rawState,
      render: this.renderState
    }
  }
}

export default DashboardEngine
