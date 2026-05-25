# Tesla 3D 模型组件

基于 Three.js 的 Tesla 车辆 3D 模型交互组件，支持车门/后备箱开关、灯光控制、充电特效、换挡动画等功能。

## 架构

组件采用 uni-app **renderjs 双 script 模式**：

- **逻辑层 `<script>`**：管理 props、事件回调，通过 `change:prop` 向 renderjs 传递数据
- **视图层 `<script module="tesla" lang="renderjs">`**：运行 Three.js 渲染，通过 `ownerVm.callMethod` 回传事件

Three.js 通过 IIFE bundle（`static/three-bundle.js`）加载，renderjs 在 `mounted` 中动态注入 `<script>` 标签。

## 基本用法

```vue
<template>
  <TeslaScene
    :state="sceneState"
    :darkModeProp="isDark"
    :licensePlate="plateObj"
    @onDoorClick="handleDoorClick"
    @onTrunkClick="handleTrunkClick"
    @onSceneReady="handleReady"
  />
</template>

<script setup>
import { ref } from 'vue'
import TeslaScene from '@/components/model/TeslaScene.vue'

const sceneState = ref({
  doors: { frontLeft: 0, frontRight: 0, rearLeft: 0, rearRight: 0 },
  trunks: { rear: 0 },
  lights: {
    drl: false, headlightLow: false, headlightHigh: false,
    turnLeft: false, turnRight: false,
    tailLight: false, brakeLight: false,
    frontFog: false, rearFog: false, hazard: false,
  },
  charging: false,
  gear: 'P',
  speed: 0,
  mirrorFolded: false,
  colors: {
    carpaint: '#1a1a2e', interior: '#2a2a2a', tire: '#1a1a1a',
    caliper: '#00a651', leather: '#1a1a1a', carpet: '#111111',
    chrome: '#c0c0c0', glass: '#3a5a5a', rim: '#c0c0c0',
  },
})

const isDark = ref(true)
const plateObj = ref({ front: '沪ACF9908', rear: '沪ACF9908' })

// 点击车门回调
function handleDoorClick(doorKey) {
  const s = sceneState.value
  s.doors[doorKey] = s.doors[doorKey] > 0.5 ? 0 : 1
}

// 点击后备箱回调
function handleTrunkClick() {
  const s = sceneState.value
  s.trunks.rear = s.trunks.rear > 0.5 ? 0 : 1
}

function handleReady() {
  console.log('3D场景加载完成')
}
</script>
```

## Props

| Prop | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `darkMode` | Boolean | `true` | 暗色模式（旧接口，建议用 darkModeProp） |
| `licensePlateFront` | String | `''` | 前车牌文字（旧接口，建议用 licensePlate） |
| `licensePlateRear` | String | `''` | 后车牌文字（旧接口，建议用 licensePlate） |
| `state` | Object | `null` | 车辆状态对象（通过 change:stateData 传递给 renderjs） |
| `darkModeProp` | Boolean | `true` | 暗色模式（通过 change:darkModeVal 传递给 renderjs） |
| `licensePlate` | Object | `null` | 车牌对象 `{ front: '', rear: '' }`（通过 change:licensePlateVal 传递给 renderjs） |

## Events

| 事件名 | 参数 | 说明 |
|--------|------|------|
| `onDoorClick` | `doorKey: string` | 用户点击3D模型车门触发，doorKey 为 `frontLeft`/`frontRight`/`rearLeft`/`rearRight` |
| `onTrunkClick` | 无 | 用户点击3D模型后备箱触发 |
| `onSceneReady` | 无 | 3D场景加载完成 |

## State 对象结构

```js
{
  // 车门开合度：0=关，1=开
  doors: {
    frontLeft: 0,   // 左前门
    frontRight: 0,  // 右前门
    rearLeft: 0,    // 左后门
    rearRight: 0,   // 右后门
  },
  // 后备箱开合度：0=关，1=开
  trunks: {
    rear: 0,
  },
  // 灯光开关
  lights: {
    drl: false,          // 日行灯
    headlightLow: false, // 近光灯
    headlightHigh: false,// 远光灯
    turnLeft: false,     // 左转向灯
    turnRight: false,    // 右转向灯
    tailLight: false,    // 尾灯
    brakeLight: false,   // 刹车灯
    frontFog: false,     // 前雾灯
    rearFog: false,      // 后雾灯
    hazard: false,       // 双闪
  },
  // 充电状态：true 激活充电特效（绿色呼吸灯+充电粒子+充电口发光）
  charging: false,
  // 挡位：P/R/N/D，影响灯光自动控制和车轮旋转方向
  gear: 'P',
  // 速度 km/h，影响车轮旋转速度
  speed: 0,
  // 后视镜折叠
  mirrorFolded: false,
  // 颜色配置
  colors: {
    carpaint: '#1a1a2e', // 车漆
    interior: '#2a2a2a', // 内饰
    tire: '#1a1a1a',     // 轮胎
    caliper: '#00a651',  // 卡钳
    leather: '#1a1a1a',  // 皮革
    carpet: '#111111',   // 地毯
    chrome: '#c0c0c0',   // 镀铬
    glass: '#3a5a5a',    // 玻璃（透明度自动计算）
    rim: '#c0c0c0',      // 轮毂
  },
}
```

## 灯光自动控制逻辑

当 `charging=true` 时，灯光由充电特效接管（绿色呼吸灯）。

非充电状态下，灯光根据挡位自动控制：

| 挡位 | 自动灯光 |
|------|----------|
| P | 全部关闭 |
| D/R | 近光灯 + 尾灯 + 刹车灯 |
| N | 日行灯 |

手动设置的灯光优先级高于自动控制。

## 充电特效

设置 `state.charging = true` 后自动激活：

- 前灯绿色呼吸灯效果
- 尾灯绿色呼吸灯效果
- 充电口绿色发光
- 充电粒子从充电桩飞向充电口
- 中控屏幕绿色能量环

## 交互功能

- **旋转**：手指拖拽旋转车辆视角
- **缩放**：双指缩放
- **点击车门**：触发 `onDoorClick` 事件
- **点击后备箱**：触发 `onTrunkClick` 事件

## 平台支持

| 平台 | 支持 | 说明 |
|------|------|------|
| H5 | ✅ | 通过 renderjs 运行 |
| APP-PLUS | ✅ | 通过 renderjs 运行 |
| 小程序 | ❌ | 不支持 WebGL |

## 文件依赖

- `static/three-bundle.js` — Three.js IIFE bundle（含 OrbitControls、GLTFLoader、postprocessing）
- `static/model/scene.gltf` — Tesla 车辆 3D 模型

## 重新构建 three-bundle.js

如需更新 Three.js 版本，运行：

```bash
# 1. 安装依赖
npm install three postprocessing

# 2. 创建入口文件 static/three-entry.js
# 3. 使用 Vite 构建 IIFE bundle
npx vite build --config build-three.mjs
```
