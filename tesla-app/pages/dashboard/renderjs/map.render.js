/**
 * Map RenderJS - 腾讯地图渲染层
 * 
 * 在 renderjs 中运行，直接操作 DOM 和地图
 * 避免 uni-app 逻辑层和视图层通信开销
 */

export default {
  data() {
    return {
      map: null,
      marker: null,
      polyline: null,
      // 当前地图状态
      currentCenter: null,
      currentZoom: 16,
      currentRotation: 0,
      // 轨迹点
      pathPoints: []
    }
  },
  
  mounted() {
    this.initMap()
  },
  
  beforeDestroy() {
    this.destroyMap()
  },
  
  methods: {
    /**
     * 初始化腾讯地图
     */
    initMap() {
      // 检查 TMap 是否加载
      if (typeof TMap === 'undefined') {
        console.error('TMap SDK not loaded')
        return
      }
      
      const container = document.getElementById('tencent-map-container')
      if (!container) return
      
      // 初始化地图
      this.map = new TMap.Map(container, {
        center: new TMap.LatLng(39.9042, 116.4074), // 默认北京
        zoom: 16,
        mapStyleId: 'style1', // 深色主题
        showControl: false, // 隐藏默认控件
        draggable: true,
        scrollable: true,
        doubleClickZoom: true,
        mapTypeControl: false
      })
      
      // 初始化车辆标记
      this.initMarker()
      
      // 初始化轨迹线
      this.initPolyline()
    },
    
    /**
     * 初始化车辆标记
     */
    initMarker() {
      if (!this.map) return
      
      this.marker = new TMap.MultiMarker({
        map: this.map,
        styles: {
          'car': new TMap.MarkerStyle({
            width: 40,
            height: 40,
            anchor: { x: 20, y: 20 },
            src: '/static/car-marker.png'
          })
        },
        geometries: [{
          id: 'vehicle',
          styleId: 'car',
          position: new TMap.LatLng(39.9042, 116.4074),
          properties: {}
        }]
      })
    },
    
    /**
     * 初始化轨迹线
     */
    initPolyline() {
      if (!this.map) return
      
      this.polyline = new TMap.MultiPolyline({
        map: this.map,
        styles: {
          'route': new TMap.PolylineStyle({
            color: 'rgba(232, 33, 39, 0.8)',
            width: 6,
            lineCap: 'round',
            lineJoin: 'round'
          })
        },
        geometries: []
      })
    },
    
    /**
     * 更新车辆位置
     */
    updateVehiclePosition(lat, lng, heading) {
      if (!this.map || !this.marker) return
      
      const position = new TMap.LatLng(lat, lng)
      
      // 更新标记位置
      this.marker.updateGeometries([{
        id: 'vehicle',
        position: position,
        properties: {}
      }])
      
      // 设置标记旋转（朝向）
      // 注意：TMap 标记本身不支持旋转，需要通过 CSS 或自定义图片实现
      // 这里使用 setRotation 是假设的 API，实际需要根据腾讯地图文档调整
      
      // 更新地图中心（跟车模式）
      // 车辆位于屏幕下 1/3 处
      this.updateMapCenter(lat, lng, heading)
    },
    
    /**
     * 更新地图中心（跟车模式）
     */
    updateMapCenter(lat, lng, heading) {
      if (!this.map) return
      
      // 计算偏移后的中心点
      // 让车辆位于屏幕下 1/3 处
      const offset = this.calculateOffset(lat, lng, heading, 0.3)
      
      this.map.setCenter(new TMap.LatLng(offset.lat, offset.lng))
      
      // 设置地图旋转（车头朝上）
      this.map.setRotation(-heading)
    },
    
    /**
     * 计算偏移后的坐标
     */
    calculateOffset(lat, lng, heading, ratio) {
      // 将角度转为弧度
      const rad = (heading * Math.PI) / 180
      
      // 计算偏移（简化计算，实际应考虑地图投影）
      const offsetLat = Math.cos(rad) * ratio * 0.01
      const offsetLng = Math.sin(rad) * ratio * 0.01
      
      return {
        lat: lat - offsetLat,
        lng: lng - offsetLng
      }
    },
    
    /**
     * 更新地图缩放级别
     */
    updateZoom(zoom) {
      if (!this.map) return
      
      // 平滑缩放
      const currentZoom = this.map.getZoom()
      if (Math.abs(currentZoom - zoom) > 0.5) {
        this.map.setZoom(zoom)
      }
    },
    
    /**
     * 添加轨迹点
     */
    addPathPoint(lat, lng) {
      this.pathPoints.push(new TMap.LatLng(lat, lng))
      
      // 限制轨迹点数量
      if (this.pathPoints.length > 1000) {
        this.pathPoints.shift()
      }
      
      // 更新轨迹线
      if (this.polyline) {
        this.polyline.updateGeometries([{
          id: 'route',
          styleId: 'route',
          paths: this.pathPoints
        }])
      }
    },
    
    /**
     * 清除轨迹
     */
    clearPath() {
      this.pathPoints = []
      if (this.polyline) {
        this.polyline.setGeometries([])
      }
    },
    
    /**
     * 销毁地图
     */
    destroyMap() {
      if (this.map) {
        this.map.destroy()
        this.map = null
      }
    },
    
    /**
     * 接收来自逻辑层的数据
     */
    receiveData(data) {
      const { lat, lng, heading, zoom, path } = data
      
      if (lat !== undefined && lng !== undefined) {
        this.updateVehiclePosition(lat, lng, heading || 0)
      }
      
      if (zoom !== undefined) {
        this.updateZoom(zoom)
      }
      
      if (path && path.length > 0) {
        path.forEach(point => {
          this.addPathPoint(point.lat, point.lng)
        })
      }
    }
  }
}
